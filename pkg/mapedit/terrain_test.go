package mapedit_test

import (
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// The new-map list is read from the one Palette: nine entries in the original
// dialog's order (TERR-246), each with the word the dialog stores, and each
// word and name once in the table.
func TestNewMapTerrainsAreTheDialogsNineFromThePalette(t *testing.T) {
	// The dialog's list, written out from TERR-246 (word per entry, in the
	// dialog's order of the claim): Land, Grass, Sand, Savanna, Cracks,
	// Flowers, Rocky, High Rocks, Water. The window's names are the palette's.
	wantWords := []uint16{0x14, 0x11, 0x91, 0xd1, 0x51, 0x191, 0x111, 0x1d1, 0x211}
	wantNames := []string{"Land", "Grass", "Sand", "Savanna", "Cracked", "Flowers", "Stones", "Mountain", "Water"}
	list := mapedit.NewMapTerrains()
	var words []uint16
	var names []string
	for _, p := range list {
		words, names = append(words, p.Word), append(names, p.Name)
		inPalette := false
		for _, q := range mapedit.Palette {
			inPalette = inPalette || q == p
		}
		if !inPalette {
			t.Errorf("%s %#x is not a palette entry", p.Name, p.Word)
		}
		s := alm.DefaultNewMap()
		s.Tile = p.Word
		if _, err := alm.NewMap(s); err != nil {
			t.Errorf("%s %#x: %v", p.Name, p.Word, err)
		}
	}
	if !slices.Equal(words, wantWords) || !slices.Equal(names, wantNames) {
		t.Errorf("new-map list %v %v, want %v %v (TERR-246)", words, names, wantWords, wantNames)
	}
}

func TestPaletteHoldsEachWordAndNameOnce(t *testing.T) {
	words, names, orders := map[uint16]bool{}, map[string]bool{}, map[int]bool{}
	for _, p := range mapedit.Palette {
		if words[p.Word] || names[p.Name] {
			t.Errorf("%s %#x appears twice", p.Name, p.Word)
		}
		words[p.Word], names[p.Name] = true, true
		if p.NewMapOrder != 0 {
			if orders[p.NewMapOrder] {
				t.Errorf("new-map place %d is held twice", p.NewMapOrder)
			}
			orders[p.NewMapOrder] = true
		}
	}
	if len(orders) != 9 {
		t.Errorf("%d new-map places, want the dialog's 9", len(orders))
	}
}

// The height tool clamps at the format's top, which is not the dialog
// slider's top (TERR-245, TERR-246).
func TestHeightToolClampsAtTheFormatTop(t *testing.T) {
	ed, _ := mapedit.New(brushMap(0x14, 120))
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 100, Radius: 0}, []mapedit.Cell{{X: 5, Y: 5}}, nil); err != nil {
		t.Fatal(err)
	}
	_, alts, _ := planes(t, ed.Bytes())
	if got := alts[5*brushSide+5]; int(got) != alm.MaxHeight || alm.MaxHeight <= alm.NewMapMaxAltitude {
		t.Fatalf("clamped height %d, want %d above the dialog's %d", got, alm.MaxHeight, alm.NewMapMaxAltitude)
	}
}

// The brush writes the +0x18 word through alm.TileGroupMask of the plane it
// leaves.
func TestBrushMaskIsTheTileGroupMask(t *testing.T) {
	for _, class := range []uint8{mapedit.ClassGrass, mapedit.ClassWater, mapedit.ClassMountain, mapedit.ClassRoad} {
		ed, _ := mapedit.New(brushMap(0x14, 40))
		if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(class), Radius: 2}, []mapedit.Cell{{X: 10, Y: 10}}, testClass); err != nil {
			t.Fatal(err)
		}
		tiles, _, mask := planes(t, ed.Bytes())
		if want := alm.TileGroupMask(tiles); mask != want {
			t.Errorf("class %d: +0x18 = %#x, want %#x", class, mask, want)
		}
	}
}
