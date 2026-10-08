package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unsafe"

	"againrom/pkg/sim"
)

// A liveRegion is every scalar of one path pattern in the live World, with the
// index of slices and arrays elided. Mutating the region changes all of them.
type liveRegion struct {
	pattern string
	values  []reflect.Value
}

// worldRegions enumerates the scalar state of a World, including unexported
// fields, grouped by pattern. Maps, interfaces and functions are reported as
// skipped: nothing here can address their elements.
func worldRegions(w *sim.World) (regions map[string]*liveRegion, skipped []string) {
	regions = map[string]*liveRegion{}
	seen := map[uintptr]bool{}
	skip := map[string]bool{}
	add := func(path string, v reflect.Value) {
		r := regions[path]
		if r == nil {
			r = &liveRegion{pattern: path}
			regions[path] = r
		}
		r.values = append(r.values, v)
	}
	var walk func(v reflect.Value, path string)
	scalar := func(k reflect.Kind) bool {
		switch k {
		case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return true
		}
		return false
	}
	walk = func(v reflect.Value, path string) {
		switch k := v.Kind(); {
		case scalar(k):
			add(path, v)
		case k == reflect.Struct:
			t := v.Type()
			for i := 0; i < t.NumField(); i++ {
				f := v.Field(i)
				if !f.CanSet() {
					f = reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
				}
				walk(f, path+"."+t.Field(i).Name)
			}
		case k == reflect.Array:
			if scalar(v.Type().Elem().Kind()) && v.Len() <= 64 {
				for i := 0; i < v.Len(); i++ {
					walk(v.Index(i), fmt.Sprintf("%s[%d]", path, i))
				}
				return
			}
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i), path+"[]")
			}
		case k == reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i), path+"[]")
			}
		case k == reflect.Pointer:
			if v.IsNil() {
				return
			}
			p := v.Pointer()
			if seen[p] && path != "" {
				// A shared target is walked once, under its first path.
				return
			}
			seen[p] = true
			walk(v.Elem(), path+"*")
		case k == reflect.Map, k == reflect.Interface, k == reflect.Func, k == reflect.Chan, k == reflect.String, k == reflect.Float32, k == reflect.Float64:
			skip[path+" ("+k.String()+")"] = true
		}
	}
	walk(reflect.ValueOf(w).Elem(), "World")
	for s := range skip {
		skipped = append(skipped, s)
	}
	sort.Strings(skipped)
	return regions, skipped
}

type regionSave struct {
	v   reflect.Value
	old uint64
}

func mutateRegion(r *liveRegion, delta int64) (restore func()) {
	saved := make([]regionSave, len(r.values))
	for i, v := range r.values {
		switch v.Kind() {
		case reflect.Bool:
			o := uint64(0)
			if v.Bool() {
				o = 1
			}
			saved[i] = regionSave{v, o}
			v.SetBool(!v.Bool())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			saved[i] = regionSave{v, uint64(v.Int())}
			v.SetInt(v.Int() + delta)
		default:
			saved[i] = regionSave{v, v.Uint()}
			v.SetUint(v.Uint() + uint64(delta))
		}
	}
	return func() {
		for _, s := range saved {
			switch s.v.Kind() {
			case reflect.Bool:
				s.v.SetBool(s.old != 0)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				s.v.SetInt(int64(s.old))
			default:
				s.v.SetUint(s.old)
			}
		}
	}
}

type liveResult struct {
	single    int
	pattern   string
	instances int
	refusal   string
	responses map[string][]int
}

func censusDiffAll(base, got map[string][]byte) map[string][]int {
	out := map[string][]int{}
	note := func(k string, offsets ...int) {
		pat := k[:strings.Index(k, "@")]
		out[pat] = append(out[pat], offsets...)
	}
	for k, v := range got {
		b, ok := base[k]
		if ok && string(b) == string(v) {
			continue
		}
		if !ok || len(b) != len(v) {
			note(k, 0)
			continue
		}
		for i := range v {
			if b[i] != v[i] {
				note(k, i)
			}
		}
	}
	for k := range base {
		if _, ok := got[k]; !ok {
			note(k, 0)
		}
	}
	for p, o := range out {
		sort.Ints(o)
		u := o[:0]
		for i, x := range o {
			if i == 0 || x != o[i-1] {
				u = append(u, x)
			}
		}
		out[p] = u
	}
	return out
}

// liveCensus mutates each live region and records which written fields move.
func liveCensus(t *testing.T, w *sim.World, export func() ([]byte, error), match func(pattern string) bool) []liveResult {
	t.Helper()
	baseRaw, err := export()
	if err != nil {
		t.Fatalf("baseline export: %v", err)
	}
	base := flattenSAV(t, baseRaw)
	regions, _ := worldRegions(w)
	names := make([]string, 0, len(regions))
	for n := range regions {
		if match == nil || match(n) {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	var results []liveResult
	for _, n := range names {
		r := regions[n]
		res := liveResult{pattern: n, instances: len(r.values)}
		try := func(sub *liveRegion, delta int64) bool {
			restore := mutateRegion(sub, delta)
			raw, err := safeExport(export)
			restore()
			if err != nil {
				msg := err.Error()
				if len(msg) > 90 {
					msg = msg[:90]
				}
				res.refusal = msg
				return false
			}
			res.refusal = ""
			if res.responses == nil {
				res.responses = map[string][]int{}
			}
			for key, offsets := range censusDiffAll(base, flattenSAV(t, raw)) {
				res.responses[key] = append(res.responses[key], offsets...)
			}
			return true
		}
		ok := false
		for _, delta := range []int64{1, -1, 17} {
			if ok = try(r, delta); ok {
				break
			}
		}
		if ok && os.Getenv("AGAINROM_CENSUS_WIDE") != "" {
			for _, delta := range []int64{0x102, 0x1020304, 0x102030405060708} {
				_ = try(r, delta)
			}
			res.refusal = ""
		}
		// Some invariants bind every instance together; one instance at a time
		// still shows where its bytes land.
		for _, i := range spreadIndexes(len(r.values)) {
			if ok {
				break
			}
			one := &liveRegion{pattern: n, values: r.values[i : i+1]}
			for _, delta := range []int64{1, -1} {
				if ok = try(one, delta); ok {
					res.single = i
					break
				}
			}
		}
		results = append(results, res)
	}
	again, err := export()
	if err != nil || string(again) != string(baseRaw) {
		t.Errorf("the live World did not return to its baseline after the census (err %v)", err)
	}
	return results
}

func writeLiveCensus(t *testing.T, name string, results []liveResult) {
	dir := os.Getenv("AGAINROM_CENSUS_OUT")
	if dir == "" {
		return
	}
	var b strings.Builder
	b.WriteString("live_pattern\tinstances\toutcome\tresponses\n")
	for _, r := range results {
		outcome := "responds"
		switch {
		case r.refusal != "":
			outcome = "refused: " + r.refusal
		case len(r.responses) == 0:
			outcome = "no-response"
		}
		var parts []string
		keys := make([]string, 0, len(r.responses))
		for k := range r.responses {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			seen := map[int]bool{}
			for _, offset := range r.responses[k] {
				seen[offset] = true
			}
			o := make([]int, 0, len(seen))
			for offset := range seen {
				o = append(o, offset)
			}
			sort.Ints(o)
			if len(o) <= 24 {
				parts = append(parts, k+":"+offsetRanges(o))
			} else {
				parts = append(parts, k+":"+fmt.Sprint(len(o))+"B")
			}
		}
		fmt.Fprintf(&b, "%s\t%d\t%s\t%s\n", r.pattern, r.instances, outcome, strings.Join(parts, " "))
	}
	if err := os.WriteFile(filepath.Join(dir, name+".tsv"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// safeExport turns a panic in the writer into a refusal, so a mutation that
// leaves the World inconsistent is reported rather than aborting the census.
func safeExport(export func() ([]byte, error)) (raw []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			raw, err = nil, fmt.Errorf("panic: %v", r)
		}
	}()
	return export()
}

// spreadIndexes picks up to eight instances across a region.
func spreadIndexes(n int) []int {
	if n <= 1 {
		return nil
	}
	var out []int
	for _, f := range []float64{0, 0.08, 0.17, 0.3, 0.45, 0.6, 0.8, 0.999} {
		i := int(f * float64(n))
		if len(out) == 0 || out[len(out)-1] != i {
			out = append(out, i)
		}
	}
	return out
}
