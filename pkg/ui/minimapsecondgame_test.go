package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// The second game's first-mission marks through the one mark list: the
// hero's mark takes the mission screen's local colour, a one-cell structure
// (the signpost) and a two-cell one (a mound) their class extents in their
// owner's shade, a hostile creature one cell in its owner's shade while its
// cell is visible and none under fog (R2-ENGINE-593, MISSION-085). A game
// whose mission screen names no local colour keeps the owner's shade.
func TestSecondGameMinimapMarksThroughTheOneList(t *testing.T) {
	blue := color.RGBA{R: 82, G: 121, B: 231, A: 255}
	v := originalViewer(t, 32, 32)
	v.SetMissionScreen(MissionScreen{MinimapLocal: blue})
	v.structureSet = &terrain.StructureSet{}
	v.structureSet.Classes[74] = &terrain.StructureClass{ID: 74, TileWidth: 1, TileHeight: 1}
	v.structureSet.Classes[26] = &terrain.StructureClass{ID: 26, TileWidth: 2, TileHeight: 1}
	v.SetLocalOwner(1)
	plane := make([]byte, 32*32)
	for y := 8; y < 24; y++ {
		for x := 8; x < 16; x++ {
			plane[y*32+x] = FogVisible
		}
	}
	v.SetFog(plane, 32, 32)
	shades := originalShades()
	v.SetStructures([]MapStructure{
		{ID: 1, Cell: image.Pt(10, 9), Kind: 74, Owner: 8, Size: image.Pt(1, 1)},
		{ID: 2, Cell: image.Pt(12, 16), Kind: 26, Owner: 3, Size: image.Pt(2, 1)},
	})
	v.SetEntities([]MapEntity{
		{ID: 29, Cell: image.Pt(9, 10), Owner: 1, Life: LifeAlive},
		{ID: 11, Cell: image.Pt(13, 12), Owner: 3, Life: LifeAlive},
		{ID: 12, Cell: image.Pt(20, 12), Owner: 3, Life: LifeAlive}, // under fog
	})
	px := func(cell image.Point, dx, dy int) image.Point {
		return cell.Sub(image.Pt(8, 8)).Mul(8).Add(image.Pt(dx, dy))
	}
	lookup := originalLookup()
	for _, tc := range []struct {
		name string
		p    image.Point
		want color.RGBA
	}{
		{"hero", px(image.Pt(9, 10), 7, 7), blue},
		{"signpost", px(image.Pt(10, 9), 0, 7), shades[8]},
		{"past the signpost", px(image.Pt(11, 9), 0, 0), lookup.RGBAAt(11, 9)},
		{"creature", px(image.Pt(13, 12), 7, 0), shades[3]},
		{"mound left cell", px(image.Pt(12, 16), 0, 0), shades[3]},
		{"mound right cell", px(image.Pt(13, 16), 7, 7), shades[3]},
		{"past the mound", px(image.Pt(14, 16), 0, 0), lookup.RGBAAt(14, 16%14)},
		{"below the mound", px(image.Pt(12, 17), 0, 0), lookup.RGBAAt(12, 17%14)},
	} {
		if got := originalContent(t, v, tc.p); got != tc.want {
			t.Errorf("%s at %v: %v, want %v", tc.name, tc.p, got, tc.want)
		}
	}
	if got := originalContent(t, v, px(image.Pt(20, 12), 3, 3)); got == shades[3] {
		t.Errorf("a creature under fog drew its mark %v", got)
	}
	a := &App{flow: &flow{viewer: v, screen: ScreenMap}}
	if local, other, err := a.HeadlessMinimapMarkAt(9, 10); err != nil || !local || other {
		t.Errorf("headless mark at the hero: local=%v other=%v %v", local, other, err)
	}
	if local, other, err := a.HeadlessMinimapMarkAt(13, 12); err != nil || local || !other {
		t.Errorf("headless mark at the creature: local=%v other=%v %v", local, other, err)
	}

	v.SetMissionScreen(MissionScreen{})
	if got := originalContent(t, v, px(image.Pt(9, 10), 3, 3)); got != shades[1] {
		t.Errorf("without a local colour the hero's mark is %v, want owner shade %v", got, shades[1])
	}
}
