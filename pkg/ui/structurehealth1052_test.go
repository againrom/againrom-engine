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
	v.SetStructures([]MapStructure{{ID: 7, Health: 0xffff, MaxHealth: 300}})
	if entries, ruined := v.StructureRuinFrames(7); entries != 1 || ruined != 1 {
		t.Fatalf("signed-negative health draws %d/%d ruin entries, want 1/1", ruined, entries)
	}
}

// TestStructureWithoutMaximumHealthDrawsIntact: the ruin gate reads 1000 for
// a structure whose maximum health is zero (TERR-STRUCT-210), so its zero
// health draws no ruin.
func TestStructureWithoutMaximumHealthDrawsIntact(t *testing.T) {
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
	v.SetStructures([]MapStructure{{ID: 7}})
	if entries, ruined := v.StructureRuinFrames(7); entries != 1 || ruined != 0 {
		t.Fatalf("zero health of zero maximum draws %d/%d ruin entries, want 0/1", ruined, entries)
	}
}
