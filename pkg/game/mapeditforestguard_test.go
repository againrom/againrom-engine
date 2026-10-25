package game

import (
	"image"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// guardMap is a 20x20 grass map with two units, one at cell (6,5) and one at
// (12,9), and a drop cell at (9,9) set through the editor.
func guardMap(t *testing.T) *mapedit.Editor {
	t.Helper()
	tiles := make([]byte, 2*20*20)
	for i := 0; i < 20*20; i++ {
		tiles[2*i] = forestLand
	}
	raw := synth.ALM(synth.ALMOptions{Width: 20, Height: 20, Type1Payload: tiles, Type7Payload: make([]byte, 12),
		Units: []synth.ALMUnit{{X: 6*256 + 128, Y: 5*256 + 128}, {X: 12*256 + 128, Y: 9*256 + 128}}})
	m, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetDropCell(9, 9); err != nil {
		t.Fatal(err)
	}
	return m
}

func guardStroke(thin bool) ui.BrushStroke {
	return ui.BrushStroke{Plane: ui.BrushPlaneObjects, Radius: 16, Shape: ui.BrushSquare,
		Cells: []image.Point{{X: 10, Y: 10}}, Forest: ui.BrushForest{Kinds: []int{3}, Share: 100, Thin: thin}}
}

// A solid planting stroke over the whole map leaves the drop cell and both
// unit cells empty and plants every other cell.
func TestForestPlantingSparesTheDropCellAndUnitCells(t *testing.T) {
	forestSeed(t, 5)
	m := guardMap(t)
	if _, err := paintStroke(m, guardStroke(false), nil); err != nil {
		t.Fatal(err)
	}
	doc, err := m.Map()
	if err != nil {
		t.Fatal(err)
	}
	spared := map[image.Point]bool{{X: 9, Y: 9}: true, {X: 6, Y: 5}: true, {X: 12, Y: 9}: true}
	planted := 0
	for i, o := range doc.Overlay {
		c := image.Pt(i%20, i/20)
		switch {
		case spared[c] && o != 0:
			t.Errorf("cell %v took %d, but the drop cell and unit cells stay empty", c, o)
		case !spared[c] && o != 3:
			t.Errorf("cell %v holds %d, want 3", c, o)
		case o != 0:
			planted++
		}
	}
	if planted != 400-3 {
		t.Fatalf("planted %d cells, want %d", planted, 400-3)
	}
}

// Thinning is unchanged: it clears a ticked kind standing on a unit cell.
func TestForestThinningStillClearsUnitCells(t *testing.T) {
	forestSeed(t, 5)
	m := guardMap(t)
	b := mapedit.Brush{Plane: mapedit.PlaneObject, Value: 3, Radius: 16, Shape: mapedit.BrushSquare}
	if _, err := m.Paint(b, []mapedit.Cell{{X: 10, Y: 10}}, mapload.TileClass); err != nil {
		t.Fatal(err)
	}
	if doc, _ := m.Map(); doc.Overlay[5*20+6] != 3 {
		t.Fatalf("the setup left %d on the unit cell", doc.Overlay[5*20+6])
	}
	if _, err := paintStroke(m, guardStroke(true), nil); err != nil {
		t.Fatal(err)
	}
	doc, _ := m.Map()
	for i, o := range doc.Overlay {
		if o != 0 {
			t.Fatalf("cell %d still holds %d after a solid thin", i, o)
		}
	}
}

// The Forest grid offers the classes that have art and not the one whose
// frame shows nothing, keeping the others in order.
func TestForestGridOffersOnlyClassesWithArt(t *testing.T) {
	raw := forestMap(t)
	d, x := newForestDriver(t, raw)
	x.statics = forestStatics(forestOakish, 9, forestOther)
	d.button(2, 1, "the Forest button")
	for i, want := range []bool{true, true, true, false} {
		if _, ok := d.e.PlaceGridCellRect(i); ok != want {
			t.Fatalf("grid cell %d present = %v, want %v", i, ok, want)
		}
	}
}
