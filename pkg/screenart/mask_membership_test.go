package screenart

import (
	"fmt"
	"image"
	"reflect"
	"slices"
	"testing"
)

func mapMaskOracle(l *Library, screen string, spec Entry, key string) (*image.Paletted, error) {
	m := l.cached(screen, key, spec.rule(), func() *memo {
		mask, err := l.src.Mask(key)
		if err == nil && spec.Size != nil {
			err = checkSize(mask, *spec.Size, key)
		}
		if err != nil {
			return &memo{err: err}
		}
		return &memo{mask: mask}
	})
	if m.err != nil {
		return nil, m.err
	}
	seen := map[int]bool{}
	for _, index := range m.mask.Pix {
		seen[int(index)] = true
	}
	for _, b := range spec.RequiredBytes {
		if !seen[b] {
			return nil, fmt.Errorf("%s: missing required mask index %d", key, b)
		}
	}
	return m.mask, nil
}

type maskObservation struct {
	mask      *image.Paletted
	errorText string
	panicText string
	didPanic  bool
}

func observeMask(l *Library, read func(*Library, string, Entry, string) (*image.Paletted, error), spec Entry) (out maskObservation) {
	defer func() {
		if p := recover(); p != nil {
			out.didPanic, out.panicText = true, fmt.Sprint(p)
		}
	}()
	var err error
	out.mask, err = read(l, "mask-screen", spec, spec.Key)
	if err != nil {
		out.errorText = err.Error()
	}
	return out
}

func equalMaskRead(t *testing.T, got, want *Library, spec Entry) {
	t.Helper()
	a, b := observeMask(got, (*Library).mask, spec), observeMask(want, mapMaskOracle, spec)
	if !reflect.DeepEqual(a, b) || got.Reads() != want.Reads() || !reflect.DeepEqual(got.memo, want.memo) {
		t.Fatalf("spec %+v: observation got %+v want %+v; reads %d/%d; memo differs %t", spec, a, b,
			got.Reads(), want.Reads(), !reflect.DeepEqual(got.memo, want.memo))
	}
}

func maskPair(pix []byte) (got, want *Library, a, b *fakeSource) {
	a, b = newFakeSource(), newFakeSource()
	for _, src := range []*fakeSource{a, b} {
		src.masks["mask"] = &image.Paletted{Pix: slices.Clone(pix), Stride: len(pix), Rect: image.Rect(0, 0, len(pix), 1)}
	}
	return NewLibrary(a), NewLibrary(b), a, b
}

func TestMaskMembershipMatchesMapForEveryByte(t *testing.T) {
	for pixel := 0; pixel < 256; pixel++ {
		got, want, a, b := maskPair([]byte{byte(pixel)})
		spec := Entry{Format: "mask", Key: "mask"}
		equalMaskRead(t, got, want, spec)
		for required := -1; required <= 256; required++ {
			spec.RequiredBytes = []int{required}
			equalMaskRead(t, got, want, spec)
		}
		missing := (pixel + 1) % 256
		for _, required := range [][]int{{pixel, pixel}, {pixel, missing, -1}, {256, missing}, {missing, 256}, {-1, 256}} {
			spec.RequiredBytes = required
			equalMaskRead(t, got, want, spec)
		}
		if a.reads["mask"] != 1 || b.reads["mask"] != 1 {
			t.Fatalf("pixel %d source reads %v/%v, want one", pixel, a.reads, b.reads)
		}
	}
}

func TestMaskMembershipMatchesMapAfterPixelWrite(t *testing.T) {
	got, want, a, b := maskPair([]byte{1, 2})
	spec := Entry{Format: "mask", Key: "mask", RequiredBytes: []int{1, 2}}
	equalMaskRead(t, got, want, spec)
	for _, pix := range [][]byte{{3, 2}, {1, 3}, nil, {}, {1, 1, 2}} {
		a.masks["mask"].Pix, b.masks["mask"].Pix = pix, slices.Clone(pix)
		for _, required := range [][]int{nil, {1, 2}, {2, 1}, {-256}, {512}} {
			spec.RequiredBytes = required
			equalMaskRead(t, got, want, spec)
		}
	}
	pix, required := make([]byte, 1024), make([]int, 256)
	for i := range pix {
		pix[i] = byte(i)
	}
	for i := range required {
		required[i] = i
	}
	a.masks["mask"].Pix, b.masks["mask"].Pix = pix, slices.Clone(pix)
	spec.RequiredBytes = required
	equalMaskRead(t, got, want, spec)
	if a.reads["mask"] != 1 || b.reads["mask"] != 1 {
		t.Fatalf("pixel writes reread the source: %v/%v", a.reads, b.reads)
	}
}

func mapMaskLoadOracle(l *Library, screen string, entries []Entry) (*Set, error) {
	set := &Set{Frames: map[string][]image.Image{}, Masks: map[string]*image.Paletted{}, numbers: map[string]map[int]string{}}
	for _, spec := range entries {
		if spec.Bound {
			continue
		}
		for _, e := range expand(spec) {
			if e.number != nil {
				if set.numbers[spec.Name] == nil {
					set.numbers[spec.Name] = map[int]string{}
				}
				set.numbers[spec.Name][*e.number] = e.name
			}
			mask, err := mapMaskOracle(l, screen, spec, e.key)
			if err != nil {
				if spec.Required {
					return nil, err
				}
				set.Problems = append(set.Problems, err.Error())
				continue
			}
			set.Masks[e.name] = mask
		}
	}
	return set, nil
}

func TestMaskMembershipMatchesMapLoadPaths(t *testing.T) {
	for _, required := range []bool{false, true} {
		for _, spec := range []Entry{
			{Name: "valid", Format: "mask", Key: "mask", RequiredBytes: []int{1, 2}},
			{Name: "missing-index", Format: "mask", Key: "mask", RequiredBytes: []int{3, -1}},
			{Name: "missing-source", Format: "mask", Key: "absent"},
			{Name: "wrong-size", Format: "mask", Key: "mask", Size: size(2, 2)},
			{Name: "indexed", Format: "mask", Key: "mask%d", Index: []Index{{Count: 2, Base: 1}}, RequiredBytes: []int{1}},
			{Name: "bound", Format: "mask", Key: "absent", Bound: true},
		} {
			got, want, a, b := maskPair([]byte{1, 2})
			for _, src := range []*fakeSource{a, b} {
				src.masks["mask1"] = src.masks["mask"]
			}
			spec.Required = required
			entries := []Entry{spec, {Name: "after", Format: "mask", Key: "mask", RequiredBytes: []int{2}, Required: true}}
			actual, actualErr := got.Load("mask-screen", entries)
			expected, expectedErr := mapMaskLoadOracle(want, "mask-screen", entries)
			if !reflect.DeepEqual(actual, expected) || fmt.Sprint(actualErr) != fmt.Sprint(expectedErr) ||
				!reflect.DeepEqual(a.reads, b.reads) || got.Reads() != want.Reads() || !reflect.DeepEqual(got.memo, want.memo) {
				t.Fatalf("spec %+v: set got %+v want %+v; error %v/%v; source reads %v/%v; library reads %d/%d",
					spec, actual, expected, actualErr, expectedErr, a.reads, b.reads, got.Reads(), want.Reads())
			}
		}
	}
}

func TestMaskMembershipMatchesMapCachedRulesAndNilPaths(t *testing.T) {
	got, want, a, b := maskPair([]byte{1, 2})
	for _, spec := range []Entry{
		{Format: "mask", Key: "mask", Size: size(2, 1)},
		{Format: "mask", Key: "mask", Size: size(1, 2)},
		{Format: "mask", Key: "mask"},
		{Format: "mask", Key: "mask", Size: size(2, 1), RequiredBytes: []int{2}},
	} {
		equalMaskRead(t, got, want, spec)
	}
	if a.reads["mask"] != 1 || b.reads["mask"] != 1 {
		t.Fatalf("conflicting rules reread the source: %v/%v", a.reads, b.reads)
	}
	for _, required := range [][]int{nil, {1}} {
		spec := Entry{Format: "mask", Key: "mask", RequiredBytes: required}
		equalMaskRead(t, NewLibrary(nil), NewLibrary(nil), spec)
		got, want, a, b := maskPair(nil)
		a.masks["mask"], b.masks["mask"] = nil, nil
		equalMaskRead(t, got, want, spec)
		spec.Size = size(1, 1)
		got, want, a, b = maskPair(nil)
		a.masks["mask"], b.masks["mask"] = nil, nil
		equalMaskRead(t, got, want, spec)
	}
}
