package game

// Tests for the front-end's own static-object wiring: the bundle it loaded
// once at startup reaching every map the picker opens, with the art on.
//
// It is an internal test for the reason frontend_test.go is one — the seam it
// needs is the picker's loader, which is unexported, and that seam is exactly
// where a front-end can be left drawing bare ground while every unit test in the
// tree passes. Nothing here goes through NewFrontEnd: assembling the FrontEnd
// directly is what lets a test about map loading need no menu asset set, and
// NewFrontEnd's own default is asserted where an install fixture already exists,
// in cmd/againrom.
//
// The bundle is HAND-BUILT rather than loaded, so this file opens no archive and
// decodes no sheet: what is under test is the hand-off, not the loader, which
// statics_test.go covers over synthetic archive bytes.

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/terrain"
)

// oneClassSet is the smallest bundle that can place: placement byte 1 naming a
// class with a drawable 1x1 frame. Plain data, no archive, no decode.
func oneClassSet() *terrain.StaticSet {
	set := new(terrain.StaticSet)
	set.Classes[1] = &terrain.StaticClass{
		Width: 2, Height: 2, CenterX: 1, CenterY: 1,
		Frame: &terrain.StaticFrame{
			Width: 1, Height: 1,
			Pixels: []terrain.StaticPixel{{Index: 1, Opaque: true}},
		},
	}
	return set
}

func TestFrontEndStatics(t *testing.T) {
	dir := t.TempDir()
	// Two cells place and two are *no object*, so a viewer that received the
	// bundle reports two placements and one that did not reports none.
	data := synth.ALM(synth.ALMOptions{
		Width: 2, Height: 2, Name: "Placed",
		Overlay: []uint8{1, 0, 0, 1},
	})
	if err := os.WriteFile(filepath.Join(dir, "a.alm"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	front := func(set *terrain.StaticSet, m Markers) *FrontEnd {
		return &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Root: dir, Loose: looseOver(t, dir)}, Tiles: &terrain.Tileset{}, Statics: set, Maps: []MapEntry{{Source: "a.alm", Name: "Placed"}}}, Presentation: Presentation{Markers: m}}
	}

	t.Run("the startup bundle reaches the map the picker opens, with the art on", func(t *testing.T) {
		// The game draws this layer as CONTENT: no flag, no opt-in, every map.
		v, _, _, _, _, _, _, _, _, _, err := front(oneClassSet(), Markers{}).loadMap(0)
		if err != nil {
			t.Fatalf("loadMap: %v", err)
		}
		if places, _ := v.Statics(); places != 2 {
			t.Errorf("%d placements, want 2 — the front-end's bundle did not reach the viewer", places)
		}
		if art, _ := v.StaticOverlay(); !art {
			t.Errorf("the object art is off; the game draws this layer unconditionally")
		}
	})

	t.Run("a front-end with no bundle draws the pre-story map screen", func(t *testing.T) {
		// A hand-assembled FrontEnd, and the shape a failed load could not
		// produce: NewFrontEnd either fills the field or returns an error.
		v, _, _, _, _, _, _, _, _, _, err := front(nil, Markers{}).loadMap(0)
		if err != nil {
			t.Fatalf("loadMap: %v", err)
		}
		if places, counts := v.Statics(); places != 0 || counts != (terrain.StaticCounts{}) {
			t.Errorf("placements = %d, counts = %+v with no bundle; want none and a silent census",
				places, counts)
		}
	})

	t.Run("the marker field's third glyph reaches the map screen", func(t *testing.T) {
		// The game asks one question and gets three glyphs, so this field is what
		// -markers writes and what the map screen reads.
		for _, on := range []bool{true, false} {
			v, _, _, _, _, _, _, _, _, _, err := front(oneClassSet(), Markers{Statics: on}).loadMap(0)
			if err != nil {
				t.Fatalf("loadMap: %v", err)
			}
			if _, cross := v.StaticOverlay(); cross != on {
				t.Errorf("Markers{Statics: %v} reached the viewer as %v", on, cross)
			}
		}
	})
}
