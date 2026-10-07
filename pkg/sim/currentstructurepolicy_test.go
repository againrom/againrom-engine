package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func currentStructureSources(structures []Structure) []SavedStructure {
	sources := make([]SavedStructure, len(structures))
	for i, structure := range structures {
		sources[i] = SavedStructure{ID: structure.ID, Class: SavedBuilding, SourceKey: uint32(900 + i), ArchiveIndex: uint16(3 + i),
			Position: [12]byte{byte(structure.Col), byte(structure.Row)}, Kind: 1, Blocking: structure.Blocking}
		sources[i].Base52[14], sources[i].Base52[15] = structure.Width, structure.Height
		binary.LittleEndian.PutUint32(sources[i].Base52[18:], structure.Blocking)
	}
	return sources
}

func TestCurrentContinuationKeepsStructuresImportedFromDocument(t *testing.T) {
	w := structureAreaWorld(t, nil)
	policy := w.CurrentPolicy()
	if policy.StructureCarrier {
		t.Fatal("fresh policy unexpectedly owns saved structures")
	}
	grid := append([]byte(nil), w.grid...)
	grid[0] |= blockStaticObject
	structures := []Structure{
		{ID: 7, Col: 10, Row: 10, Width: 1, Height: 2, Blocking: 5, Field42: 60, MaxHealth: 80},
		{ID: 9, Col: 12, Row: 11, Width: 2, Height: 1, Blocking: 5, Field42: 70, MaxHealth: 90},
	}
	sources := currentStructureSources(structures)
	cells := []SavedStructureCell{{Cell: 0x0a0a, ID: 7, HasStructure: true}, {Cell: 0x0b0c, ID: 9, HasStructure: true}}
	if err := w.ImportOriginalStructures(structures, sources, cells, grid); err != nil {
		t.Fatal(err)
	}
	if err := w.RestoreCurrentContinuation(&policy, nil, w.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	gotSources, gotCells, present := w.SavedStructures()
	if !present || !reflect.DeepEqual(w.Structures(), structures) || !reflect.DeepEqual(gotSources, sources) || !reflect.DeepEqual(gotCells, cells) {
		t.Fatal("document structure roster or source state was lost")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var loaded World
	if err := loaded.UnmarshalBinary(form); err != nil {
		t.Fatal("native LOAD changed the restored structure state", err)
	}
	loadedSources, loadedCells, loadedPresent := loaded.SavedStructures()
	if !loadedPresent || !reflect.DeepEqual(loaded.Structures(), structures) || !reflect.DeepEqual(loadedSources, sources) || !reflect.DeepEqual(loadedCells, cells) {
		t.Fatal("native LOAD lost the document structure roster")
	}
}

func TestCurrentContinuationDropsUncarriedStructureImport(t *testing.T) {
	w := structureAreaWorld(t, nil)
	policy := w.CurrentPolicy()
	structures := []Structure{{ID: 7, Col: 10, Row: 10, Width: 1, Height: 2, Blocking: 5}}
	if err := w.ImportOriginalStructures(structures, currentStructureSources(structures), nil, w.grid); err != nil {
		t.Fatal(err)
	}
	if err := w.RestoreCurrentContinuation(&policy, nil, w.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	sources, cells, present := w.SavedStructures()
	if present || len(sources) != 0 || len(cells) != 0 || !reflect.DeepEqual(w.Structures(), structures) {
		t.Fatal("uncarried source rows changed the fresh-world policy")
	}
}
