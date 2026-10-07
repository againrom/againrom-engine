package ui

import (
	"testing"

	"againrom/pkg/render/terrain"
)

func TestSignedNonpositiveStructureHealthSelectsTheRuinBlock(t *testing.T) {
	set := new(terrain.StructureSet)
	set.Classes[1] = &terrain.StructureClass{
		TileWidth: 1, TileHeight: 1, FullHeight: 1,
		Frames: []*terrain.StaticFrame{structureFrame(), structureFrame()},
	}
	g := terrain.Grid{
		Width: 1, Height: 1, Tiles: []uint16{0}, Altitudes: []uint8{0},
		Structures: []terrain.StructureRecord{{ID: 7, Key: 1}},
	}
	v := newStructureViewer(t, g, set, true)
	v.SetStructures([]MapStructure{{ID: 7, Health: 0xffff}})
	if entries, ruined := v.StructureRuinFrames(7); entries != 1 || ruined != 1 {
		t.Fatalf("signed-negative health draws %d/%d ruin entries, want 1/1", ruined, entries)
	}
}
