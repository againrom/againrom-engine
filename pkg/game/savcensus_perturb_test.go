package game

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unsafe"

	"againrom/pkg/formats/sav"
)

// A region of a written SAV has one of these origins, observed by changing the
// loaded Document under an unchanged live world.
const (
	// The written value follows the loaded Document.
	sourceDocument = "document"
	// A changed loaded value makes the writer refuse: the Document is checked
	// against live state and cannot deviate from it.
	sourceChecked = "checked"
	// The loaded value does not reach the written bytes; the bytes come from
	// live state, a rule or a constructor value.
	sourceCurrent = "current"
)

type censusFieldResult struct {
	pattern       string
	instances     int
	width         int
	source        string
	docOffsets    []int // byte offsets (or 0 for scalars) that follow the Document
	refused       []int // byte offsets whose change was refused
	collateral    []string
	acceptedMasks []string
	refusedMasks  []string
}

// A censusRun is one prepared experiment: the perturbable inputs, the export
// route that consumes them, and a hook that undoes the preparation.
type censusRun struct {
	leaves []censusLeaf
	export func() ([]byte, error)
	done   func()
}

type censusEnv struct {
	t       *testing.T
	prepare func() *censusRun
	// foreign marks inputs outside the written Document's namespace: a leaf is
	// then reported with every written region it moves.
	foreign bool
}

func (e *censusEnv) baseline() (map[string][]byte, []byte) {
	e.t.Helper()
	r := e.prepare()
	defer r.done()
	raw, err := r.export()
	if err != nil {
		e.t.Fatalf("baseline export: %v", err)
	}
	return flattenSAV(e.t, raw), raw
}

func flattenSAV(t *testing.T, raw []byte) map[string][]byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatalf("decode written SAV: %v", err)
	}
	out := map[string][]byte{}
	for _, l := range censusLeaves(&doc) {
		out[censusKey(l)] = l.bytes()
	}
	return out
}

// perturbation changes bytes [lo,hi) of every byte leaf of a pattern, or XORs
// a scalar with mask.
type perturbation struct {
	pattern string
	lo, hi  int
	mask    uint64
}

func applyPerturbation(leaves []censusLeaf, p perturbation) int {
	n := 0
	for _, l := range leaves {
		if l.pattern != p.pattern || l.topology {
			continue
		}
		v := l.value
		if !v.CanSet() && v.CanAddr() {
			v = reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
		}
		switch v.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			size := uint(v.Type().Size()) * 8
			m := p.mask
			if size < 64 {
				m &= 1<<size - 1
			}
			v.SetUint(v.Uint() ^ m)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			size := uint(v.Type().Size()) * 8
			bits := uint64(v.Int()) ^ p.mask
			v.SetInt(int64(bits<<(64-size)) >> (64 - size))
		case reflect.Bool:
			v.SetBool(!v.Bool())
		case reflect.Float64:
			v.SetFloat(math.Float64frombits(math.Float64bits(v.Float()) ^ p.mask))
		case reflect.String:
			v.SetString(v.String() + "~")
		case reflect.Slice, reflect.Array:
			b := l.bytes()
			hi := p.hi
			if hi > len(b) || hi <= 0 {
				hi = len(b)
			}
			if p.lo >= hi {
				continue
			}
			for i := p.lo; i < hi; i++ {
				b[i] ^= 0xFF
			}
			reflect.Copy(v, reflect.ValueOf(b))
		}
		n++
	}
	return n
}

// run exports once with the loaded inputs perturbed.
func (e *censusEnv) run(p perturbation) (map[string][]byte, error) {
	r := e.prepare()
	defer r.done()
	if applyPerturbation(r.leaves, p) == 0 {
		return nil, fmt.Errorf("no leaf")
	}
	raw, err := safeExport(r.export)
	if err != nil {
		return nil, err
	}
	return flattenSAV(e.t, raw), nil
}

func censusDiff(base, got map[string][]byte, pattern string) (self map[int]bool, collateral map[string]bool) {
	self, collateral = map[int]bool{}, map[string]bool{}
	seen := map[string]bool{}
	for k, v := range got {
		seen[k] = true
		b, ok := base[k]
		if ok && bytes.Equal(b, v) {
			continue
		}
		pat := k[:strings.Index(k, "@")]
		if pat == pattern {
			if !ok || len(b) != len(v) {
				self[0] = true
				continue
			}
			for i := range v {
				if b[i] != v[i] {
					self[i] = true
				}
			}
		} else {
			collateral[pat] = true
		}
	}
	for k := range base {
		if !seen[k] {
			pat := k[:strings.Index(k, "@")]
			if pat == pattern {
				self[0] = true
			} else {
				collateral[pat] = true
			}
		}
	}
	return
}

// region classifies one pattern by perturbing it.
func (e *censusEnv) region(base map[string][]byte, pattern string, leaves []censusLeaf) censusFieldResult {
	r := censusFieldResult{pattern: pattern, instances: len(leaves)}
	width := 0
	for _, l := range leaves {
		if w := len(l.bytes()); w > width {
			width = w
		}
	}
	r.width = width
	if e.foreign {
		if os.Getenv("AGAINROM_CENSUS_HIGH") != "" {
			all := map[string]map[int]bool{}
			for _, mask := range []uint64{1, 0x100, 0x10000, 0x1000000, 0x100000000, 0x10000000000, 0x1000000000000, 0x100000000000000} {
				kind := leaves[0].value.Kind()
				if kind != reflect.Slice && kind != reflect.Array && kind != reflect.String && kind != reflect.Bool {
					bits := leaves[0].value.Type().Size() * 8
					if bits < 64 && mask >= uint64(1)<<bits {
						continue
					}
				} else if mask != 1 {
					continue
				}
				got, err := e.run(perturbation{pattern: pattern, mask: mask})
				if err != nil {
					r.refusedMasks = append(r.refusedMasks, fmt.Sprintf("%x", mask))
					continue
				}
				r.acceptedMasks = append(r.acceptedMasks, fmt.Sprintf("%x", mask))
				for name, offsets := range censusDiffAll(base, got) {
					if all[name] == nil {
						all[name] = map[int]bool{}
					}
					for _, offset := range offsets {
						all[name][offset] = true
					}
				}
			}
			r.source = sourceCurrent
			if len(all) != 0 {
				r.source = sourceDocument
			} else if len(r.acceptedMasks) == 0 {
				r.source = sourceChecked
			}
			for name, offsets := range all {
				var sorted []int
				for offset := range offsets {
					sorted = append(sorted, offset)
				}
				sort.Ints(sorted)
				r.collateral = append(r.collateral, name+":"+offsetRanges(sorted))
			}
			sort.Strings(r.collateral)
			return r
		}
		var got map[string][]byte
		var err error
		for _, mask := range []uint64{0xFFFFFFFF, 0x55555555, 1} {
			got, err = e.run(perturbation{pattern: pattern, mask: mask})
			if err == nil {
				break
			}
		}
		if err != nil {
			r.source, r.refused = sourceChecked, []int{0}
			return r
		}
		effects := censusDiffAll(base, got)
		r.source = sourceCurrent
		if len(effects) != 0 {
			r.source = sourceDocument
		}
		for k, o := range effects {
			if len(o) <= 16 {
				r.collateral = append(r.collateral, k+":"+offsetRanges(o))
			} else {
				r.collateral = append(r.collateral, fmt.Sprintf("%s:%dB", k, len(o)))
			}
		}
		sort.Strings(r.collateral)
		return r
	}
	k := leaves[0].value.Kind()
	isBytes := k == reflect.Slice || k == reflect.Array
	if !isBytes {
		if os.Getenv("AGAINROM_CENSUS_HIGH") != "" && k != reflect.String {
			docs, collateral := map[int]bool{}, map[string]bool{}
			for _, mask := range []uint64{1, 0x100, 0x10000, 0x1000000, 0x80, 0x4000, 0x800000, 0x80000000} {
				got, err := e.run(perturbation{pattern: pattern, mask: mask})
				if err != nil {
					r.refusedMasks = append(r.refusedMasks, fmt.Sprintf("%x", mask))
					continue
				}
				r.acceptedMasks = append(r.acceptedMasks, fmt.Sprintf("%x", mask))
				self, col := censusDiff(base, got, pattern)
				for off := range self {
					docs[off] = true
				}
				for name := range col {
					collateral[name] = true
				}
			}
			for off := range docs {
				r.docOffsets = append(r.docOffsets, off)
			}
			sort.Ints(r.docOffsets)
			r.collateral = sortedKeys(collateral)
			r.source = sourceCurrent
			if len(docs) != 0 {
				r.source = sourceDocument
			} else if len(r.acceptedMasks) == 0 {
				r.source = sourceChecked
			}
			return r
		}
		var got map[string][]byte
		var err error
		for _, mask := range []uint64{0xFFFFFFFF, 0x55555555, 1} {
			got, err = e.run(perturbation{pattern: pattern, mask: mask})
			if err == nil {
				break
			}
		}
		if err != nil {
			r.source, r.refused = sourceChecked, []int{0}
			return r
		}
		self, col := censusDiff(base, got, pattern)
		for o := range self {
			r.docOffsets = append(r.docOffsets, o)
		}
		sort.Ints(r.docOffsets)
		r.collateral = sortedKeys(col)
		r.source = sourceCurrent
		if len(self) != 0 {
			r.source = sourceDocument
		}
		return r
	}
	if width == 0 {
		r.source = sourceCurrent
		return r
	}
	got, err := e.run(perturbation{pattern: pattern})
	if err == nil {
		self, col := censusDiff(base, got, pattern)
		for o := range self {
			r.docOffsets = append(r.docOffsets, o)
		}
		r.collateral = sortedKeys(col)
		r.source = sourceCurrent
		if len(self) != 0 {
			r.source = sourceDocument
		}
		sort.Ints(r.docOffsets)
		return r
	}
	if width > 512 {
		r.source, r.refused = sourceChecked, []int{-1}
		return r
	}
	// Some offsets are checked. Find each one, and perturb the rest alone.
	docs, col := map[int]bool{}, map[string]bool{}
	for off := 0; off < width; off++ {
		got, err := e.run(perturbation{pattern: pattern, lo: off, hi: off + 1})
		if err != nil {
			r.refused = append(r.refused, off)
			continue
		}
		self, c := censusDiff(base, got, pattern)
		for o := range self {
			docs[o] = true
		}
		for k := range c {
			col[k] = true
		}
	}
	for o := range docs {
		r.docOffsets = append(r.docOffsets, o)
	}
	sort.Ints(r.docOffsets)
	r.collateral = sortedKeys(col)
	switch {
	case len(r.docOffsets) != 0:
		r.source = sourceDocument
	case len(r.refused) != 0:
		r.source = sourceChecked
	default:
		r.source = sourceCurrent
	}
	return r
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func offsetRanges(offsets []int) string {
	if len(offsets) == 0 {
		return ""
	}
	var parts []string
	start, prev := offsets[0], offsets[0]
	flush := func() {
		if start == prev {
			parts = append(parts, fmt.Sprint(start))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", start, prev))
		}
	}
	for _, o := range offsets[1:] {
		if o == prev+1 {
			prev = o
			continue
		}
		flush()
		start, prev = o, o
	}
	flush()
	return strings.Join(parts, ",")
}

// table runs the whole census and writes a TSV to AGAINROM_CENSUS_OUT.
func (e *censusEnv) table(name string) []censusFieldResult {
	base, _ := e.baseline()
	probe := e.prepare()
	names, by := censusPatterns(probe.leaves)
	probe.done()
	var results []censusFieldResult
	for _, pattern := range names {
		if !censusMatch(pattern, os.Getenv("AGAINROM_CENSUS_MATCH")) {
			continue
		}
		leaves := by[pattern]
		if leaves[0].topology {
			results = append(results, censusFieldResult{pattern: pattern, instances: len(leaves), source: "topology"})
			continue
		}
		results = append(results, e.region(base, pattern, leaves))
	}
	if dir := os.Getenv("AGAINROM_CENSUS_OUT"); dir != "" {
		var b strings.Builder
		b.WriteString("pattern\tinstances\twidth\tsource\tdocument_offsets\trefused_offsets\tcollateral\taccepted_masks\trefused_masks\tunknown_meaning_spans\n")
		for _, r := range results {
			fmt.Fprintf(&b, "%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", r.pattern, r.instances, r.width, r.source, offsetRanges(r.docOffsets), offsetRanges(r.refused), strings.Join(r.collateral, ","), strings.Join(r.acceptedMasks, ","), strings.Join(r.refusedMasks, ","), censusUnknownSpans(r.pattern))
		}
		if err := os.WriteFile(filepath.Join(dir, name+".tsv"), []byte(b.String()), 0o644); err != nil {
			e.t.Fatal(err)
		}
	}
	return results
}

func censusUnknownSpans(pattern string) string {
	var out []string
	for _, entry := range savDocumentFallbacks {
		if entry.Pattern == pattern {
			out = append(out, fmt.Sprintf("%s[%s]:%s (%s)", entry.Span, entry.Indices, entry.Reason, entry.Page))
		}
	}
	return strings.Join(out, "; ")
}

func censusMatch(pattern, match string) bool {
	if match == "" {
		return true
	}
	for _, term := range strings.Split(match, "|") {
		if strings.Contains(pattern, term) {
			return true
		}
	}
	return false
}
