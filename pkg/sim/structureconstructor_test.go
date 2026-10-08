package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestStructureConstructorKeepsFreshGhostTerrainAndAliases(t *testing.T) {
	grid := make([]byte, 16)
	grid[1] = blockAir
	w, err := NewWorld(7, Bounds{Width: 4, Height: 4}, ModeCanonical, grid, nil)
	if err != nil {
		t.Fatal(err)
	}
	structures := []Structure{
		{ID: 7, Kind: 1, Col: 0, Row: 0, Width: 2, Height: 1, Attach: 3, Blocking: 3},
		{ID: 9, Kind: 1, Col: 0, Row: 0, Width: 3, Height: 1, Attach: 7, Blocking: 7},
	}
	w.DeclareStructures(structures)
	sources := make([]SavedStructure, len(structures))
	for i, st := range structures {
		sources[i] = SavedStructure{ID: st.ID, Class: GeneratedBuilding, SourceKey: uint32(21 + i), RuntimeID: uint32(31 + i),
			Position: [12]byte{byte(st.Col), byte(st.Row)}, Kind: 1, Blocking: st.Blocking}
		sources[i].Base52[14], sources[i].Base52[15] = st.Width, st.Height
		binary.LittleEndian.PutUint32(sources[i].Base52[18:], st.Blocking)
	}
	cells := []SavedStructureCell{{Cell: 0, ID: 7, HasStructure: true}, {Cell: 1, ID: 7, HasStructure: true}}
	beforeGrid, beforeSlots := append([]byte(nil), w.grid...), w.structureSlots
	if w.terrainOpen(DomainGhost, 1, 0) {
		t.Fatal("fixture fresh Ghost must read air bit")
	}
	if err := w.ConstructSavedStructures(sources, cells); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.grid, beforeGrid) || !reflect.DeepEqual(w.structureSlots, beforeSlots) || w.terrainOpen(DomainGhost, 1, 0) {
		t.Fatal("Building metadata changed fresh grid/aliases/Ghost pathing")
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if w.Hash() != cold.Hash() || cold.terrainOpen(DomainGhost, 1, 0) || !reflect.DeepEqual(cold.structureSlots, beforeSlots) {
		t.Fatal("constructor provenance changed after cold form")
	}
	occupancy := w.StructureOccupancy()
	if !reflect.DeepEqual(occupancy, []StructureCellBinding{{Cell: 0, ID: 7}, {Cell: 1, ID: 7}}) {
		t.Fatal("constructor did not preserve first-collision alias prefix", occupancy)
	}
	occupancy[0].ID = 99
	if w.StructureOccupancy()[0].ID != 7 {
		t.Fatal("occupancy getter aliases current state")
	}
}

func TestStructureConstructorGeneratedAndMixedPlanePolicy(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		grid := make([]byte, 4)
		grid[1] = blockAir
		w, err := NewWorld(3, Bounds{Width: 2, Height: 2}, ModeCanonical, grid, nil)
		if err != nil {
			t.Fatal(err)
		}
		live := []Structure{{ID: 1, Kind: 1}, {ID: 2, Kind: 1}}
		source := []SavedStructure{{ID: 1, Class: GeneratedBuilding, SourceKey: 11}, {ID: 2, Class: GeneratedBuilding, SourceKey: 12}}
		if mixed {
			source[1].Class, source[1].ArchiveIndex = SavedBuilding, 31
		}
		if err := w.ImportOriginalStructures(live, source, nil, grid); err != nil {
			t.Fatal(err)
		}
		if w.terrainOpen(DomainGhost, 1, 0) != mixed {
			t.Fatal("Generated and mixed provenance merged their Ghost masks", mixed)
		}
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold World
		if err := cold.UnmarshalBinary(raw); err != nil || cold.terrainOpen(DomainGhost, 1, 0) != mixed {
			t.Fatal("cold form changed mixed provenance", mixed, err)
		}
		policy := w.CurrentPolicy()
		policy.StructureCarrier = false
		if err := w.RestoreCurrentContinuation(&policy, nil, w.Actions(), nil); err != nil {
			t.Fatal(err)
		}
		if w.terrainOpen(DomainGhost, 1, 0) {
			t.Fatal("absent legacy carrier retained native Ghost mask", mixed)
		}
	}
}

func TestStructureConstructorEmptyNativeCarrierKeepsNativePlanes(t *testing.T) {
	w, err := NewWorld(3, Bounds{Width: 2, Height: 2}, ModeCanonical, []byte{0, blockAir, 0, 0}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.terrainOpen(DomainGhost, 1, 0) {
		t.Fatal("fresh fixture ignores air")
	}
	if err := w.ImportOriginalStructures(nil, nil, nil, w.CurrentPolicy().Terrain.Block); err != nil {
		t.Fatal(err)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || !cold.terrainOpen(DomainGhost, 1, 0) {
		t.Fatal("empty imported carrier was treated as Generated", err)
	}
}
