package game

import (
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentTerminalBindingsImportWithoutSnapshotActorEntity(t *testing.T) {
	const id sim.EntityID = 34
	const object uint16 = 1
	const identity uint32 = 0x62000020
	terminal := sim.CurrentTerminalActor{ID: id, Cell: 0x0504, HP: -601, Stage: 4}
	doc := sav.DocumentData{Version: sav.DocumentDataVersion, Head: sav.DocumentHeadData{Mission: 130}, World: &sav.DocumentWorldData{}, DeadActors: []uint16{object},
		Objects: []sav.DocumentRecordData{{Class: "Unit", Values: []sav.DocumentValueData{{Name: "Identity", Value: identity}}}}}
	stateData := sav.NewCityStateData("terminal binding")
	doc.State = sav.DocumentStateData{RootKind: stateData.RootKind, DirectoryRecords: stateData.DirectoryRecords, ValueRecords: stateData.ValueRecords}
	a := currentActionData{Version: 1, Bindings: []currentActionBinding{{ID: id, Object: object}},
		Values: map[sim.EntityID]sim.ActorValues{id: {CurrentTerminal: &terminal}}}
	encoded, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, encoded); err != nil {
		t.Fatal(err)
	}
	state := &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &doc}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 2, X: 2, Y: 2, HP: 10, MaxHP: 10}, {ID: id, X: 1, Y: 1, HP: 10, MaxHP: 10, MapUnitID: 7}})
	if err != nil {
		t.Fatal(err)
	}
	ms := &Mission{World: w, savedDocument: state}
	if err := restoreCurrentActorIdentities(ms, &a); err != nil {
		t.Fatalf("terminal identity restore without a snapshot actor row: %v", err)
	}
	if got := w.CurrentTerminalActors(); !reflect.DeepEqual(got, []sim.CurrentTerminalActor{terminal}) {
		t.Fatalf("imported terminal tuple = %+v", got)
	}
	if got := w.Entities(); len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("terminal restore did not retire only the constructor actor at its exact ID: %+v", got)
	}
	byID, err := currentTerminalActorBindings(state, w)
	if err != nil || !reflect.DeepEqual(byID, map[sim.EntityID]uint16{id: object}) {
		t.Fatalf("terminal SAV bindings = %v, err %v", byID, err)
	}
	byObject, err := currentTerminalActorObjects(state, w)
	if err != nil || !reflect.DeepEqual(byObject, map[uint16]sim.EntityID{object: id}) {
		t.Fatalf("terminal actor objects = %v, err %v", byObject, err)
	}
	keys, err := currentTerminalActorKeys(state)
	if err != nil || !reflect.DeepEqual(keys, map[uint32]bool{identity: true}) {
		t.Fatalf("terminal identity keys = %v, err %v", keys, err)
	}
}

func TestCurrentTerminalBindingRejectsMissingDeadRootBeforeRetiringConstructor(t *testing.T) {
	const id sim.EntityID = 34
	const object uint16 = 1
	terminal := sim.CurrentTerminalActor{ID: id, Cell: 0x0504, HP: -601, Stage: 4}
	doc := sav.DocumentData{Version: sav.DocumentDataVersion, Head: sav.DocumentHeadData{Mission: 130}, World: &sav.DocumentWorldData{},
		Objects: []sav.DocumentRecordData{{Class: "Unit", Values: []sav.DocumentValueData{{Name: "Identity", Value: 0x62000020}}}}}
	stateData := sav.NewCityStateData("unrooted terminal")
	doc.State = sav.DocumentStateData{RootKind: stateData.RootKind, DirectoryRecords: stateData.DirectoryRecords, ValueRecords: stateData.ValueRecords}
	a := currentActionData{Version: 1, Bindings: []currentActionBinding{{ID: id, Object: object}},
		Values: map[sim.EntityID]sim.ActorValues{id: {CurrentTerminal: &terminal}}}
	encoded, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, encoded); err != nil {
		t.Fatal(err)
	}
	state := &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &doc}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: id, X: 1, Y: 1, HP: 10, MaxHP: 10}})
	if err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	ms := &Mission{World: w, savedDocument: state}
	if err := restoreCurrentActorIdentities(ms, &a); err == nil || w.Hash() != before || len(w.Entities()) != 1 {
		t.Fatal("terminal import without an exact DeadActors root removed a constructed actor")
	}
}
