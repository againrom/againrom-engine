package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// fillTiles paints one wide grass stroke through the window's own stroke
// route and returns the saved bytes and the tile plane.
func fillTiles(t *testing.T, fill ui.BrushFill) ([]byte, []uint16) {
	t.Helper()
	m, err := mapedit.NewMap(alm.NewMapSpec{Width: 40, Height: 40, Tile: 0x14, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	op := ui.BrushStroke{Plane: ui.BrushPlaneTerrain, Value: int(mapedit.ClassGrass), Radius: 8, Shape: ui.BrushSquare, Fill: fill,
		Cells: []image.Point{{X: 20, Y: 20}}}
	if _, err := paintStroke(m, op, nil); err != nil {
		t.Fatal(err)
	}
	f, err := alm.Open(m.Bytes())
	if err != nil {
		t.Fatalf("the saved map does not load back: %v", err)
	}
	return m.Bytes(), f.Tiles
}

// A random fill draws a fresh seed for each stroke: two identical strokes
// over one area give different interiors, while the edge words stay those of
// the pattern. A fixed variant gives the same interior every time, and the
// pattern is the loss control that does not change between strokes.
func TestRandomFillIsFreshPerStrokeAndKeepsEdges(t *testing.T) {
	vars := mapedit.Variants(mapedit.ClassGrass)
	inside := func(w uint16) bool { return slices.Contains(vars, w) }

	_, pat := fillTiles(t, ui.BrushFill{})
	_, pat2 := fillTiles(t, ui.BrushFill{})
	if !slices.Equal(pat, pat2) {
		t.Fatal("the pattern changed between two identical strokes")
	}
	a, ta := fillTiles(t, ui.BrushFill{Mode: ui.BrushFillRandom})
	b, tb := fillTiles(t, ui.BrushFill{Mode: ui.BrushFillRandom})
	if slices.Equal(a, b) || slices.Equal(ta, tb) {
		t.Fatal("two random strokes over one area gave one interior")
	}
	for name, got := range map[string][]uint16{"first": ta, "second": tb} {
		for i := range pat {
			switch {
			case inside(pat[i]) != inside(got[i]):
				t.Fatalf("%s stroke: cell %d changes between interior and edge", name, i)
			case !inside(pat[i]) && pat[i] != got[i]:
				t.Fatalf("%s stroke: edge cell %d is %#x, the pattern wrote %#x", name, i, got[i], pat[i])
			}
		}
	}
	_, f1 := fillTiles(t, ui.BrushFill{Mode: ui.BrushFillVariant, Variant: 3})
	_, f2 := fillTiles(t, ui.BrushFill{Mode: ui.BrushFillVariant, Variant: 3})
	if !slices.Equal(f1, f2) {
		t.Fatal("a fixed variant gave two interiors")
	}
	for i, w := range f1 {
		if inside(w) && w != vars[3] {
			t.Fatalf("cell %d is %#x, not variant 3 %#x", i, w, vars[3])
		}
	}
}

// With the seed seam fixed, one seed gives one interior.
func TestRandomFillSeedSeam(t *testing.T) {
	old := strokeSeed
	defer func() { strokeSeed = old }()
	strokeSeed = func() uint64 { return 5 }
	a, _ := fillTiles(t, ui.BrushFill{Mode: ui.BrushFillRandom})
	b, _ := fillTiles(t, ui.BrushFill{Mode: ui.BrushFillRandom})
	if !slices.Equal(a, b) {
		t.Fatal("one seed gave two interiors")
	}
	strokeSeed = func() uint64 { return 6 }
	c, _ := fillTiles(t, ui.BrushFill{Mode: ui.BrushFillRandom})
	if slices.Equal(a, c) {
		t.Fatal("two seeds gave one interior")
	}
}
