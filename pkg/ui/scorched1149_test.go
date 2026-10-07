package ui

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"againrom/pkg/render/terrain"
)

func TestScorched1149GroundUploadKeysAndLiteralPixels(t *testing.T) {
	src := image.NewPaletted(image.Rect(0, 0, 32, 32), color.Palette{color.RGBA{10, 20, 30, 255}})
	dirt := image.NewPaletted(image.Rect(5, 7, 37, 39), color.Palette{color.RGBA{255, 0, 0, 255}, color.RGBA{40, 50, 60, 255}})
	dirt.SetColorIndex(6, 7, 1)
	pixels := scorchedCellPixels(src, dirt, [3]uint8{1, 2, 3})
	if pixels.RGBAAt(0, 0) != (color.RGBA{11, 22, 33, 255}) || pixels.RGBAAt(1, 0) != (color.RGBA{41, 52, 63, 255}) {
		t.Fatal("palette-index transparency, replacement or tint changed")
	}
	set := &terrain.Tileset{Dirt: &terrain.Strip{SubCells: []*image.Paletted{dirt, dirt, dirt, dirt}}}
	for i := range set.Slots {
		set.Slots[i] = &terrain.Strip{SubCells: []*image.Paletted{src}}
	}
	v, err := NewViewer("ground", grid(32, 32), set)
	if err != nil {
		t.Fatal(err)
	}
	_, noOverlay, clean := v.cellTexture(0, 16, 16)
	if noOverlay != nil || clean.dirt != 0 {
		t.Fatal("clean land acquired a scorch")
	}
	for x := 16; x < 20; x++ {
		_, overlay, burned := v.cellTexture(0x2000, x, 16)
		if overlay != dirt || burned.dirt != uint8(x-15) || burned == clean {
			t.Fatal("burned texture reused clean or wrong sub-cell key")
		}
	}
	_, noOverlay, water := v.cellTexture(0x2200, 16, 16)
	if noOverlay != nil || water.dirt != 0 {
		t.Fatal("water acquired a dirt overlay")
	}
}

func TestScorched1149ViewerUsesDeadGeometryAndPreservesMap(t *testing.T) {
	dead := &terrain.StaticClass{Width: 64, Height: 32, CenterX: 10, CenterY: 25, Frame: staticsFrame(4, 8, 9), FireObject: -2}
	live := &terrain.StaticClass{Width: 64, Height: 64, CenterX: 32, CenterY: 60, Frame: staticsFrame(20, 40, 7), Dead: dead, FireObject: -1}
	set := &terrain.StaticSet{}
	set.Classes[1] = live
	g := terrain.Grid{Width: 32, Height: 32, Tiles: make([]uint16, 1024), Altitudes: make([]byte, 1024), Overlay: make([]byte, 1024)}
	g.Overlay[16*32+16], g.Overlay[16*32+17] = 1, 1
	g.Tiles[17*32+17] = 512
	base := slices.Clone(g.Tiles)
	v, err := NewViewerWithStatics("scorch", g, &terrain.Tileset{}, set, true, false, true, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	v.SetScorchedCells([]uint16{0x1010, 0x1111, 0xffff})
	if !slices.Equal(g.Tiles, base) || v.tileWord(16, 16) != 0x2000 || v.tileWord(17, 17) != 0x2200 || v.tileWord(17, 16) != 0 {
		t.Fatal("map alias, water, or neighbour corrupted")
	}
	for _, places := range [][]terrain.StaticPlacement{v.staticsFlat, v.staticsDisplaced} {
		if len(places) != 2 || places[0].Class != dead || places[0].Frame != dead.Frame || places[0].Anchor != image.Pt(-20, 13) || places[0].TopLeft != image.Pt(548, 515) || places[1].Class != live {
			t.Fatalf("dead sheet/geometry differs: %+v", places)
		}
	}
	if len(v.ambientStatics) != 1 || v.ambientStatics[0].fireObject != -2 {
		t.Fatal("burned scenery retained living ambience")
	}
	v.SetScorchedCells(nil)
	if !slices.Equal(v.grid.Tiles, base) || v.staticsFlat[0].Class != live {
		t.Fatal("new empty history did not restore authored scene")
	}
	// An authored bit13 selects the same dead form at construction.
	g.Tiles[16*32+16] |= 0x2000
	places, _, _ := terrain.StaticPlacements(g, set, nil, 0, true)
	if places[0].Class != dead || places[0].Frame != dead.Frame {
		t.Fatal("authored burned tree stays live")
	}
}
