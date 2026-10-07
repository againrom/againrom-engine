package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func TestCurrentActionWithoutImportedMotionReleasesTypedCell(t *testing.T) {
	key := uint32(0x0316a100)
	want := mustWorld(t, 91, Bounds{32, 32}, []Entity{{ID: 1, X: 15, Y: 16, HP: 10, MaxHP: 10}})
	want.entities[0].SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 1, Identity: key}
	want.entities[0].ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
	cell := SavedActorCell{Cell: 0x100f, Ground: SavedActorSlot{Key: key}}
	binary.LittleEndian.PutUint32(cell.Payload[4:], key)
	want.savedMotion = &savedActorMotionState{Cells: []SavedActorCell{cell}}
	want.savedCellRecords = []SavedCellRecord{{Cell: cell.Cell, Ground: SavedCellActorSlot{Key: key}}}
	a := actionCopy(t, want.Actions())
	cold := mustWorld(t, 91, Bounds{32, 32}, []Entity{{ID: 1, X: 15, Y: 16, HP: 10, MaxHP: 10}})
	cold.entities[0].SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 1, Identity: key}
	cold.entities[0].ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
	cell.Ground = SavedActorSlot{Key: key, Entity: 1, Bound: true}
	cold.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 1, Current: true, Position: SavedActorPosition{Cell: cell.Cell, PackedCell: cell.Cell, FineX: 128, FineY: 128}}}, Cells: []SavedActorCell{cell}}
	cold.savedCellRecords = []SavedCellRecord{{Cell: cell.Cell, Ground: SavedCellActorSlot{Key: key, Entity: 1, Bound: true}}}
	bad := *cold
	bad.savedMotion = cloneActorMotions(cold.savedMotion)
	bad.savedCellRecords = append([]SavedCellRecord(nil), cold.savedCellRecords...)
	bad.savedMotion.Cells[0].Payload[4] ^= 1
	beforeMotion := cloneActorMotions(bad.savedMotion)
	beforeRecords := bad.SavedCellRecords()
	if err := bad.RestoreActions(a, nil); err == nil || !reflect.DeepEqual(bad.savedMotion, beforeMotion) || !reflect.DeepEqual(bad.SavedCellRecords(), beforeRecords) {
		t.Fatal("corrupt actor key was accepted or partially restored", err)
	}
	badZero := *cold
	badZero.entities = append([]Entity(nil), cold.entities...)
	badZero.entities[0].SourceBinding.Identity = 0
	badZero.savedMotion = cloneActorMotions(cold.savedMotion)
	badZero.savedMotion.Cells[0].Ground.Key = 0
	binary.LittleEndian.PutUint32(badZero.savedMotion.Cells[0].Payload[4:], 0)
	beforeZero := cloneActorMotions(badZero.savedMotion)
	if err := badZero.RestoreActions(a, nil); err == nil || !reflect.DeepEqual(badZero.savedMotion, beforeZero) {
		t.Fatal("zero bound actor key was accepted or partially restored", err)
	}
	badSource := *cold
	badSource.entities = append([]Entity(nil), cold.entities...)
	badSource.entities[0].SourceBinding.Identity++
	badSource.savedMotion = cloneActorMotions(cold.savedMotion)
	beforeSource := cloneActorMotions(badSource.savedMotion)
	if err := badSource.RestoreActions(a, nil); err == nil || !reflect.DeepEqual(badSource.savedMotion, beforeSource) {
		t.Fatal("typed key was detached from a different source identity", err)
	}
	if err := cold.RestoreActions(a, nil); err != nil {
		t.Fatal(err)
	}
	got, err := cold.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	for comparison := 0; comparison < 2; comparison++ {
		expected, err := want.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, expected) || !reflect.DeepEqual(cold.SavedCellRecords(), want.SavedCellRecords()) {
			t.Fatal("cold action left a typed cell bound to absent imported motion")
		}
		Step(cold, nil)
		Step(want, nil)
		got, err = cold.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCurrentActionImportedMotionRestoresCurrentFlag(t *testing.T) {
	want := mustWorld(t, 91, Bounds{32, 32}, []Entity{{ID: 1, X: 15, Y: 16, HP: 10, MaxHP: 10}})
	want.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 1, Current: true, Issue: "original boundary speed callback is not executed", Position: SavedActorPosition{Cell: 0x100f, PackedCell: 0x100f, FineX: 128, FineY: 128}}}}
	a := actionCopy(t, want.Actions())
	if !a.Actors[0].ImportedMotion {
		t.Fatal("current source motion was not captured")
	}
	cold := *want
	cold.savedMotion = cloneActorMotions(want.savedMotion)
	cold.savedMotion.Motions[0].Current = false
	cold.savedMotion.Motions[0].Issue = "native movement continues the imported route"
	if err := cold.RestoreActions(a, nil); err != nil {
		t.Fatal(err)
	}
	wantRaw, wantErr := want.MarshalBinary()
	coldRaw, coldErr := cold.MarshalBinary()
	if wantErr != nil || coldErr != nil || !bytes.Equal(wantRaw, coldRaw) {
		t.Fatalf("current action did not restore the source motion flag: %v, %v", wantErr, coldErr)
	}
}
