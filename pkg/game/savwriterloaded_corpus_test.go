//go:build sessioncorpusaudit

// Loaded-document census. A SAVE is built from the World; only the spans
// savDocumentFallbacks names may come from the loaded file. Each save is
// loaded and its Snapshot taken; then the loaded document the Snapshot holds
// is changed twice: once only inside the unknown-meaning spans, once
// everywhere but the join keys, a scalar's known part beside its unknown
// span included. Both SAVEs must be byte-identical: a written leaf that
// differs is a known kind taken from the loaded file. The census names each.
package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"againrom/pkg/formats/sav"
)

// loadedCensusSaves are corpus-relative names; the bytes stay in the corpus.
var loadedCensusSaves = []struct {
	name string
	town bool
}{
	{"2026-10-07/saveorcsdontgo.sav", false},
	{"2026-10-07/beforekargallas.sav", true},
	{"2026-10-06/game0007-original-m70-boltcoming.sav", false},
	{"2026-10-06/game0008-original-m70-dying.sav", false},
	// Loaded original world spell effects.
	{"2026-08-15/game0018.sav", false},
}

// loadedCensusAnchor names the leaves the full change keeps: the format
// version and the keys that join a loaded record to its World object. They are
// read to join, never written from the loaded file.
func loadedCensusAnchor(pattern string) bool {
	switch {
	case pattern == "Version", pattern == "Marker", strings.HasSuffix(pattern, ".v.Identity"), strings.HasSuffix(pattern, ".v.This"), strings.HasSuffix(pattern, ".v.Reference"),
		pattern == "Player.v.Slot", pattern == "World.Cells[].Cell":
		return true
	}
	return false
}

// loadedCensusDeclaredDebt reports a known Token field of an original world
// spell effect record. The World holds no Token for those records, so a SAVE
// copies them from the loaded record (DIV-2502). The census names each one.
func loadedCensusDeclaredDebt(pattern string) bool {
	class, field, ok := strings.Cut(pattern, ".")
	switch class {
	case "SpellEffect", "PointEffect", "AreaEffect", "SpellTransport", "Effect", "Effect_DirectDamage":
	default:
		return false
	}
	switch field {
	case "r.Block12", "v.RuntimeID", "v.T08", "v.T0C", "v.T0E", "v.T18", "v.T1C":
		return ok
	}
	return false
}

const (
	poisonNone = iota
	poisonUnknown
	poisonAll
)

// setLeafBytes writes a leaf from its little-endian bytes.
func setLeafBytes(l censusLeaf, b []byte) {
	v := l.value
	if !v.CanSet() && v.CanAddr() {
		v = reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
	}
	wide := make([]byte, 8)
	copy(wide, b)
	switch v.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(binary.LittleEndian.Uint64(wide))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		size := uint(v.Type().Size()) * 8
		bits := binary.LittleEndian.Uint64(wide)
		v.SetInt(int64(bits<<(64-size)) >> (64 - size))
	case reflect.Slice, reflect.Array:
		reflect.Copy(v, reflect.ValueOf(b))
	}
}

// poisonDocument changes the loaded document: every byte of every
// non-topology leaf, or only the bytes inside unknown-meaning spans. A
// non-nil keep names the only patterns the full change may touch.
func poisonDocument(doc *sav.DocumentData, mode int, keep map[string]bool) {
	spans := unknownRecordSpans()
	for _, l := range censusLeaves(doc) {
		if l.topology || mode == poisonNone || loadedCensusAnchor(l.pattern) || keep != nil && !keep[l.pattern] {
			continue
		}
		v := l.value
		switch v.Kind() {
		case reflect.Bool:
			if mode == poisonAll {
				reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().SetBool(!v.Bool())
			}
			continue
		case reflect.String:
			if mode == poisonAll && v.Len() > 0 {
				reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().SetString("Z" + v.String()[1:])
			}
			continue
		}
		b := l.bytes()
		spanned := false
		for i := range b {
			if unknownSpanAllows(spans, censusKey(l), i) {
				b[i] ^= 0x5a
				spanned = true
			}
		}
		switch {
		case mode != poisonAll:
		case spanned && v.Kind() != reflect.Slice && v.Kind() != reflect.Array:
			// A scalar that holds an unknown span changes there in both modes;
			// the full change also flips the low bit of its first known byte
			// below the span. A byte above it lies outside a narrower field's
			// wire (a 16-bit field whose whole wire is unknown).
			last := -1
			for i := range b {
				if unknownSpanAllows(spans, censusKey(l), i) {
					last = i
				}
			}
			for i := 0; i < last; i++ {
				if !unknownSpanAllows(spans, censusKey(l), i) {
					b[i] ^= 1
					break
				}
			}
		case v.Kind() == reflect.Slice || v.Kind() == reflect.Array:
			for i := range b {
				if !unknownSpanAllows(spans, censusKey(l), i) {
					b[i] ^= 0x5a
				}
			}
		default:
			// The low bit keeps a value inside its field's range.
			b[0] ^= 1
		}
		setLeafBytes(l, b)
	}
}

// unknownSpanAllows reports whether byte at of a leaf lies in an
// unknown-meaning span of its pattern.
func unknownSpanAllows(spans map[string][]unknownSpan, key string, at int) bool {
	pattern, instance, _ := strings.Cut(key, "@")
	for _, span := range spans[pattern] {
		if span.indices[0] >= 0 {
			open := strings.LastIndex(instance, "[")
			if open < 0 {
				continue
			}
			index, err := strconv.Atoi(strings.TrimSuffix(instance[open+1:], "]"))
			if err != nil || index < span.indices[0] || index > span.indices[1] {
				continue
			}
		}
		if slices.Contains(span.offsets, at) {
			return true
		}
	}
	return false
}

// loadedCensusDiff lists the patterns whose written leaves differ.
func loadedCensusDiff(a, b map[string][]byte) []string {
	seen := map[string]bool{}
	for k, v := range a {
		if w, ok := b[k]; !ok || !bytes.Equal(v, w) {
			seen[k[:strings.Index(k, "@")]] = true
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			seen[k[:strings.Index(k, "@")]] = true
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// loadedCensusMission returns the three SAVEs and, when the full change is
// refused, the known kinds a per-pattern change names instead.
func loadedCensusMission(t *testing.T, name string) ([3]map[string][]byte, []string) {
	f := censusMissionFront(t, name)
	export := func(mode int, keep map[string]bool) ([]byte, error) {
		snap, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		if snap.SavedDocument == nil || snap.SavedDocument.Document == nil {
			t.Fatal("the Snapshot holds no loaded document")
		}
		state, err := cloneSavedDocument(snap.SavedDocument)
		if err != nil {
			t.Fatal(err)
		}
		poisonDocument(state.Document, mode, keep)
		snap.SavedDocument = state
		return safeExport(func() ([]byte, error) { return f.ExportCurrentSave(snap, label) })
	}
	var out [3]map[string][]byte
	for mode := range out {
		raw, err := export(mode, nil)
		if err != nil && mode == poisonAll {
			t.Logf("the full change is refused (%v); each pattern is changed alone", err)
			return out, loadedCensusPerPattern(t, f, export)
		}
		if err != nil {
			t.Fatalf("SAVE refused with the loaded document changed (mode %d): %v", mode, err)
		}
		out[mode] = flattenSAV(t, raw)
	}
	return out, nil
}

// loadedCensusPerPattern changes one pattern at a time, inside its unknown
// spans and then everywhere. A pattern whose full change alters the SAVE or
// is refused is a known kind taken from the loaded file.
func loadedCensusPerPattern(t *testing.T, f *FrontEnd, export func(int, map[string]bool) ([]byte, error)) []string {
	t.Helper()
	snap, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	var known []string
	tried := map[string]bool{}
	for _, l := range censusLeaves(snap.SavedDocument.Document) {
		if tried[l.pattern] || l.topology || loadedCensusAnchor(l.pattern) {
			continue
		}
		tried[l.pattern] = true
		keep := map[string]bool{l.pattern: true}
		base, err := export(poisonUnknown, keep)
		if err != nil {
			t.Fatalf("SAVE refused with %s changed in its unknown spans: %v", l.pattern, err)
		}
		full, err := export(poisonAll, keep)
		switch {
		case err != nil:
			t.Logf("SAVE refuses a changed %s: %v", l.pattern, err)
			known = append(known, l.pattern+" (SAVE refused)")
		case len(loadedCensusDiff(flattenSAV(t, base), flattenSAV(t, full))) != 0:
			known = append(known, l.pattern)
		}
	}
	sort.Strings(known)
	return known
}

func loadedCensusTown(t *testing.T, name string) [3]map[string][]byte {
	raw := censusCorpusFile(t, name)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	var out [3]map[string][]byte
	var keep map[string]bool
	for mode := range out {
		if _, city, err := f.RestoreOriginal(raw); err != nil || !city {
			t.Fatalf("restore town: city=%v err=%v", city, err)
		}
		snap, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		if snap.OriginalCity == nil {
			t.Fatal("the Snapshot holds no loaded town")
		}
		provenance, err := sav.CityFromData(snap.OriginalCity.Document)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := provenance.DocumentData()
		if err != nil {
			t.Fatal(err)
		}
		if keep == nil {
			keep = loadedCensusTownPatterns(t, doc)
		}
		poisonDocument(&doc, mode, keep)
		changed, err := loadedCensusCity(doc)
		if err != nil {
			t.Fatalf("read the changed town (mode %d): %v", mode, err)
		}
		city := *snap.OriginalCity
		city.Document = changed.Data()
		snap.OriginalCity = &city
		written, err := safeExport(func() ([]byte, error) { return f.ExportCurrentSave(snap, label) })
		if err != nil {
			t.Fatalf("town SAVE refused with the loaded town changed (mode %d): %v", mode, err)
		}
		out[mode] = flattenSAV(t, written)
	}
	return out
}

func loadedCensusCity(doc sav.DocumentData) (*sav.CityProvenance, error) {
	wire, err := sav.EncodeDocumentData(doc)
	if err != nil {
		return nil, err
	}
	file, err := sav.Open(wire)
	if err != nil {
		return nil, err
	}
	p, err := file.CityProvenance()
	if err != nil {
		return nil, err
	}
	return sav.CityFromData(p.Data())
}

// loadedCensusTownPatterns keeps each pattern whose change the town format
// still reads; a pattern the format refuses is a format constraint, not a
// value the writer could carry.
func loadedCensusTownPatterns(t *testing.T, doc sav.DocumentData) map[string]bool {
	t.Helper()
	keep, tried := map[string]bool{}, map[string]bool{}
	for _, l := range censusLeaves(&doc) {
		if tried[l.pattern] {
			continue
		}
		tried[l.pattern] = true
		probe, err := sav.CloneDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		poisonDocument(&probe, poisonAll, map[string]bool{l.pattern: true})
		if _, err := loadedCensusCity(probe); err == nil {
			keep[l.pattern] = true
		} else {
			t.Logf("town format refuses a changed %s: %v", l.pattern, err)
		}
	}
	return keep
}

func TestSAVWriterCensusLoadedDocument(t *testing.T) {
	if os.Getenv("AGAINROM_SAVE_CORPUS") == "" {
		t.Skip("no AGAINROM_SAVE_CORPUS: the loaded-document census needs owner saves")
	}
	var report []string
	for _, input := range loadedCensusSaves {
		t.Run(filepath.Base(input.name), func(t *testing.T) {
			var out [3]map[string][]byte
			var probed []string
			if input.town {
				out = loadedCensusTown(t, input.name)
			} else {
				out, probed = loadedCensusMission(t, input.name)
			}
			unknown := loadedCensusDiff(out[poisonNone], out[poisonUnknown])
			known := loadedCensusDiff(out[poisonUnknown], out[poisonAll])
			if out[poisonAll] == nil {
				known = probed
			}
			var declared, undeclared []string
			for _, k := range known {
				if loadedCensusDeclaredDebt(k) {
					declared = append(declared, k)
				} else {
					undeclared = append(undeclared, k)
				}
			}
			line := fmt.Sprintf("%s: %d known kinds from the loaded file %v, %d of them the declared world-effect Token debt (DIV-2502); %d written kinds follow its unknown-meaning spans %v",
				input.name, len(known), known, len(declared), len(unknown), unknown)
			report = append(report, line)
			t.Log(line)
			if len(undeclared) != 0 {
				t.Errorf("known kinds taken from the loaded file: %v", undeclared)
			}
		})
	}
	if dir := os.Getenv("AGAINROM_CENSUS_OUT"); dir != "" {
		if err := os.WriteFile(filepath.Join(dir, "loaded-document-census.txt"), []byte(strings.Join(report, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
