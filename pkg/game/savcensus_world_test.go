package game

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
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
