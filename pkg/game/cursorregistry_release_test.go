package game

import (
	"image"
	"testing"
)

func TestReleaseCursorRegistryResolvesAllTwentyEightSlotsFromTheRealInstall(t *testing.T) {
	f := releaseFront(t)
	if f.CursorRegistry.Err() != nil || f.CursorRegistry.Value() == nil {
		t.Fatalf("the production cursor registry did not resolve: %v", f.CursorRegistry.Err())
	}
	reg := f.CursorRegistry

	want := []struct {
		name    string
		frames  int
		w, h    int
		hotspot image.Point
		period  int64
	}{
		{"default", 1, 32, 32, image.Pt(5, 5), 2000000000},
		{"move", 5, 32, 32, image.Pt(15, 15), 100},
		{"swarm", 5, 44, 44, image.Pt(21, 21), 100},
		{"attack", 10, 32, 32, image.Pt(3, 3), 100},
		{"defend", 8, 32, 32, image.Pt(15, 13), 100},
		{"select", 1, 32, 32, image.Pt(3, 4), 100},
		{"patrol", 8, 32, 32, image.Pt(8, 25), 100},
		{"cast", 14, 32, 32, image.Pt(15, 15), 100},
		{"pickup", 15, 32, 32, image.Pt(12, 13), 66},
		{"arrow0", 1, 32, 32, image.Pt(15, 5), 2000000000},
		{"arrow4", 1, 32, 32, image.Pt(16, 25), 2000000000},
		{"arrow6", 1, 32, 32, image.Pt(6, 16), 2000000000},
		{"arrow2", 1, 32, 32, image.Pt(25, 15), 2000000000},
		{"arrow7", 1, 32, 32, image.Pt(8, 9), 2000000000},
		{"arrow5", 1, 32, 32, image.Pt(8, 23), 2000000000},
		{"arrow1", 1, 32, 32, image.Pt(22, 8), 2000000000},
		{"arrow3", 1, 32, 32, image.Pt(23, 22), 2000000000},
		{"sdefault", 1, 16, 16, image.Pt(2, 2), 2000000000},
		{"smove", 1, 16, 16, image.Pt(0, 0), 2000000000},
		{"sattack", 1, 16, 16, image.Pt(0, 0), 2000000000},
		{"sdefend", 1, 16, 16, image.Pt(0, 0), 2000000000},
		{"spatrol", 1, 16, 16, image.Pt(0, 0), 2000000000},
		{"scast", 1, 16, 16, image.Pt(0, 0), 2000000000},
		{"cantput", 1, 64, 64, image.Pt(38, 36), 2000000000},
		{"town", 1, 32, 32, image.Pt(16, 16), 2000000000},
		{"dice", 15, 32, 32, image.Pt(16, 16), 100},
		{"wait", 10, 32, 32, image.Pt(16, 16), 100},
		{"backpack", 1, 32, 32, image.Pt(16, 16), 100},
	}

	if len(reg.Value().Slots) != len(want) {
		t.Fatalf("the registry resolved %d slots, want %d (SPR16A-CURSOR-067)", len(reg.Value().Slots), len(want))
	}
	for i, w := range want {
		got := reg.Value().Slots[i]
		if got.Name != w.name {
			t.Errorf("slot %d is %q, want %q; slot order is construction order and the eight arrows are not in arrow-number order (SPR16A-CURSOR-067)", i, got.Name, w.name)
			continue
		}
		if len(got.Frames) != w.frames || got.FrameCount != w.frames {
			t.Errorf("slot %d (%s): %d frames resolved, FrameCount %d, want %d from the install's own sheet", i, w.name, len(got.Frames), got.FrameCount, w.frames)
		}
		if got.Hotspot != w.hotspot {
			t.Errorf("slot %d (%s): hotspot %v, want %v", i, w.name, got.Hotspot, w.hotspot)
		}
		if got.PeriodMillis != w.period {
			t.Errorf("slot %d (%s): period %d, want %d", i, w.name, got.PeriodMillis, w.period)
		}
		if len(got.Frames) == 0 {
			t.Errorf("slot %d (%s) resolved no picture at all", i, w.name)
			continue
		}
		for n, pic := range got.Frames {
			if pic == nil {
				t.Errorf("slot %d (%s) frame %d is nil", i, w.name, n)
				continue
			}
			if b := pic.Bounds(); b.Dx() != w.w || b.Dy() != w.h {
				t.Errorf("slot %d (%s) frame %d is %dx%d, want %dx%d", i, w.name, n, b.Dx(), b.Dy(), w.w, w.h)
			}
		}
		// THE HOTSPOT MUST FALL INSIDE THE FRAME, which is what makes it
		// usable as a draw offset at all: the picture's top-left goes at the
		// cursor point less this, so a hotspot outside the art would place
		// the pointer where nothing of it is drawn.
		if w.hotspot.X >= w.w || w.hotspot.Y >= w.h {
			t.Errorf("slot %d (%s): hotspot %v lies outside the %dx%d frame", i, w.name, w.hotspot, w.w, w.h)
		}
	}

	// NO TWO SLOTS RESOLVE TO THE SAME ART. The failure this catches is a
	// loader that silently fell back to one default sheet for entries it could
	// not find: the 28 registrations name 28 distinct paths, so identical
	// pixels under two names mean the archive lookup did not answer with the
	// named entry. It compares PIXELS, over every pair. Keying a map on
	// Frames[0] instead cannot fail — LoadCursorRegistry allocates a fresh
	// image.NewRGBA per frame per call, so two slots always hold two distinct
	// pointers — and that is what this check did until adversarial pass 2
	// measured it by repointing two registrations at one path and finding the
	// test still at exit 0.
	//
	// The other half of the same property, two names sharing one path, is a
	// fact about the registration table and needs no install: it is
	// TestCursorRegistrationsNameOneSheetEach in cursorregistry_test.go.
	//
	// MEASURED BEFORE IT WAS ASSERTED: on gameversions/en and gameversions/ru
	// the 378 pairs of the 28 slots yield zero identical first frames, so no
	// shipped pair forces this to be scoped. Twelve of the slots are
	// single-frame 32x32 and six more are single-frame 16x16, which is the
	// population where a shared sheet would be least visible.
	for i := range reg.Value().Slots {
		a := &reg.Value().Slots[i]
		if len(a.Frames) == 0 {
			continue
		}
		for j := i + 1; j < len(reg.Value().Slots); j++ {
			b := &reg.Value().Slots[j]
			if len(b.Frames) == 0 {
				continue
			}
			if pixelsEqual(a.Frames[0], b.Frames[0]) {
				t.Errorf("slots %s and %s resolved to byte-identical art, so the loader did not read each named entry", a.Name, b.Name)
			}
		}
	}

	// AND A MULTI-FRAME SHEET'S FRAMES DIFFER. Nine of the 28 animate; a
	// decoder returning the same frame N times would animate to no visible
	// effect and pass every count above.
	for i := range reg.Value().Slots {
		s := &reg.Value().Slots[i]
		if len(s.Frames) < 2 {
			continue
		}
		if pixelsEqual(s.Frames[0], s.Frames[1]) {
			t.Errorf("slot %s: frames 0 and 1 are byte-identical, so the animation shows nothing", s.Name)
		}
	}
}

// pixelsEqual compares two decoded frames byte for byte.
func pixelsEqual(a, b *image.RGBA) bool {
	if a == nil || b == nil || a.Bounds() != b.Bounds() || len(a.Pix) != len(b.Pix) {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}
