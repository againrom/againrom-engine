package game

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Each carrier the set took in from its left list is a row whose steps sit at
// their points: the kept loaded document, the identity remap and its terminal
// actors, the builder's reads of the loaded record, the Token Reference
// misses, the LOAD matchers, the settled application state and the town's
// carried Diary.
func TestTheMovedCarriersAreStepsOfTheSet(t *testing.T) {
	loaded := worldCarrierRow(t, "the loaded document")
	if loaded.decode[materialRecord] == nil || loaded.build[seatBuild] == nil || loaded.build[joinBuild] == nil {
		t.Error("the loaded document row does not keep the document, seat it in the builder and join the actors")
	}
	if worldCarrierRow(t, "World actor identities").decode[identityRecord] == nil {
		t.Error("the identity remap is not the set's identity step")
	}
	if worldCarrierRow(t, "World terminal actors").decode[terminalRecord] == nil {
		t.Error("the terminal actor row does not decode inside the identity remap")
	}
	if dead := worldCarrierRow(t, "World original dead bodies"); dead.build[keysBuild] == nil || dead.build[deadRecordBuild] == nil {
		t.Error("the dead body row does not claim its retained keys and retain its loaded records in the builder")
	}
	if worldCarrierRow(t, "the loaded file's Token Reference misses").decode[fileRecord] == nil {
		t.Error("the Token Reference misses are not decoded from the file record")
	}
	for prefix, at := range map[string]carrierRecord{"spell book selection": selectionRecord, "order spell absence": orderRecord,
		"World spell-effect Tokens": tokenRecord, "cell plane residue": cellRecord} {
		if worldCarrierRow(t, prefix).decode[at] == nil {
			t.Errorf("the %s row has no LOAD matcher at its actions leaf point", prefix)
		}
	}
	if worldCarrierRow(t, "the application state").seat[settleSeat] == nil {
		t.Error("the application state is not seated at the mission settle")
	}
	diary := worldCarrierRow(t, "World saved Diaries")
	for _, at := range []carrierTownPoint{townCapture, townRestore, townArchive, townCarry, townWrite} {
		if diary.town[at] == nil {
			t.Errorf("the Diary row has no town step at %d", at)
		}
	}
}

// The material point keeps the loaded document for the mission writers, and
// the file point captures the loaded file's Token Reference misses.
func TestTheSetKeepsTheLoadedDocumentAndItsMisses(t *testing.T) {
	state := &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &sav.DocumentData{}}
	s := Snapshot{SavedDocument: state}
	if err := decodeWorldCarriersFrom(materialRecord, carrierDecode{s: &s}); err != nil || s.loadedDocument != state {
		t.Fatalf("the material point kept %p, want %p (%v)", s.loadedDocument, state, err)
	}
	doc := sav.DocumentData{Objects: []sav.DocumentRecordData{{Class: "Item", Values: []sav.DocumentValueData{{Name: "Identity", Value: 0x24}, {Name: "Reference", Value: 0x99}}}}}
	file := &SnapshotSAVDocument{Document: &doc}
	if err := decodeWorldCarriersFrom(fileRecord, carrierDecode{state: file}); err != nil || !reflect.DeepEqual(file.ReferenceMisses, loadedTokenReferenceMisses(&doc)) || len(file.ReferenceMisses) != 1 {
		t.Fatalf("the file point captured misses %+v (%v)", file.ReferenceMisses, err)
	}
}

// The builder's seat point gives it the loaded record and its join point
// joins an actor to the identity its loaded record holds.
func TestTheSetSeatsTheLoadedRecordInTheBuilder(t *testing.T) {
	doc := sav.DocumentData{Objects: []sav.DocumentRecordData{{Class: "Unit", Values: []sav.DocumentValueData{{Name: "Identity", Value: 0x77}}}}}
	s := Snapshot{SavedDocument: &SnapshotSAVDocument{Document: &doc, Actors: []SnapshotSAVActor{{EntityID: 5, ObjectIndex: 1}}}}
	w := secondGameScriptWorld(t, nil, nil)
	b := &generatedDocumentBuilder{currentKeys: map[uint32]bool{}}
	if err := buildWorldCarriers(seatBuild, carrierBuild{b: b, s: &s, w: w}); err != nil || b.loaded != &doc {
		t.Fatalf("the seat point gave the builder %p, want %p (%v)", b.loaded, &doc, err)
	}
	if err := buildWorldCarriers(joinBuild, carrierBuild{b: b, s: &s, w: w}); err != nil || b.joined[5] != 0x77 || !b.currentKeys[0x77] {
		t.Fatalf("the join point joined %v (%v)", b.joined, err)
	}
}

// The identity step remaps the actions leaf's identities once and imports
// its terminal actor rows at the terminal point inside it.
func TestTheIdentityStepImportsTheTerminalActors(t *testing.T) {
	const id sim.EntityID = 34
	terminal := sim.CurrentTerminalActor{ID: id, Cell: 0x0504, HP: -601, Stage: 4, MaxHP: 31, MaxHPPresent: true}
	doc := sav.DocumentData{Version: sav.DocumentDataVersion, Head: sav.DocumentHeadData{Mission: 130}, World: &sav.DocumentWorldData{}, DeadActors: []uint16{1},
		Objects: []sav.DocumentRecordData{{Class: "Unit", Values: []sav.DocumentValueData{{Name: "Identity", Value: 0x62000020}, {Name: "HealthMax", Value: 31}}}}}
	stateData := sav.NewCityStateData("terminal step")
	doc.State = sav.DocumentStateData{RootKind: stateData.RootKind, DirectoryRecords: stateData.DirectoryRecords, ValueRecords: stateData.ValueRecords}
	a := currentActionData{Version: 1, Bindings: []currentActionBinding{{ID: id, Object: 1}}, Values: map[sim.EntityID]sim.ActorValues{id: {CurrentTerminal: &terminal}}}
	encoded, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, encoded); err != nil {
		t.Fatal(err)
	}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 2, X: 2, Y: 2, HP: 10, MaxHP: 10}, {ID: id, X: 1, Y: 1, HP: 10, MaxHP: 10, MapUnitID: 7}})
	if err != nil {
		t.Fatal(err)
	}
	ms := &Mission{World: w, savedDocument: &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &doc}}
	if err := decodeWorldCarriersFrom(identityRecord, carrierDecode{w: w, ms: ms, state: ms.savedDocument, actions: &a}); err != nil {
		t.Fatal(err)
	}
	if got := w.CurrentTerminalActors(); !reflect.DeepEqual(got, []sim.CurrentTerminalActor{terminal}) {
		t.Fatalf("the identity step imported %+v", got)
	}
}

// The Diary row's town steps move the carried Diary between the town, its
// Snapshot, an ended mission's World and a city document.
func TestTheDiaryRowCarriesTheTownsKnowledge(t *testing.T) {
	known := []sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: 3, Entries: []sim.SavedDiaryEntry{{Index: 1, Count: 2, Remaining: 3}}}}
	var s Snapshot
	if err := seatTownCarriers(townCapture, carrierTown{t: &Town{knowledge: known}, s: &s}); err != nil || !reflect.DeepEqual(s.knowledge, known) {
		t.Fatalf("the capture point copied %+v (%v)", s.knowledge, err)
	}
	restored := &Town{}
	if err := seatTownCarriers(townRestore, carrierTown{t: restored, s: &s}); err != nil || !reflect.DeepEqual(restored.knowledge, known) {
		t.Fatalf("the restore point put %+v (%v)", restored.knowledge, err)
	}
	w := secondGameScriptWorld(t, nil, nil)
	w.SetSavedDiaries(known)
	carried := &Town{}
	if err := seatTownCarriers(townCarry, carrierTown{t: carried, w: w}); err != nil || !reflect.DeepEqual(carried.knowledge, known) {
		t.Fatalf("the carry point kept %+v (%v)", carried.knowledge, err)
	}
	archived := &Town{knowledge: known}
	if err := seatTownCarriers(townArchive, carrierTown{t: archived}); err != nil || archived.knowledge != nil {
		t.Fatalf("the archive point kept %+v from a file with no Player Diary (%v)", archived.knowledge, err)
	}
	if err := seatTownCarriers(townWrite, carrierTown{s: &s, doc: &sav.DocumentData{}}); err != nil {
		t.Fatalf("the write point refused an empty city document: %v", err)
	}
}

// worldCarrierIndex is the place of the row whose home starts with prefix.
func worldCarrierIndex(t *testing.T, prefix string) int {
	t.Helper()
	worldCarrierRow(t, prefix)
	for i, c := range worldCarrierSet() {
		if strings.HasPrefix(c.home, prefix) {
			return i
		}
	}
	return -1
}

// Each site that kept a carrier from the loaded record outside the set is a
// step of the row that owns the carrier, in the order its call ran: the world
// effect keys before the dead body keys, the world effects once the
// projectile leaves are written, the Player name before the name is written,
// the actor actions as the first graph writer, the graft of the unknown
// spans, the session head absence before the carrier absence, the session
// entry the original LOAD publishes into the loaded record, and the dead
// body art's decode and write.
func TestTheLoadedRecordSitesAreStepsOfTheirRows(t *testing.T) {
	effects := worldCarrierRow(t, "World saved spell effects")
	if effects.build[keysBuild] == nil || effects.build[effectsBuild] == nil {
		t.Error("the World spell-effect row does not reserve its keys and write its records in the builder")
	}
	if worldCarrierIndex(t, "World saved spell effects") > worldCarrierIndex(t, "World original dead bodies") {
		t.Error("the world effect keys are reserved after the dead body keys")
	}
	loaded := worldCarrierRow(t, "the loaded document")
	if loaded.build[playerBuild] == nil || loaded.writer == nil || loaded.stage != graftStage {
		t.Error("the loaded document row does not keep the Player name and graft the unknown spans")
	}
	actions := worldCarrierRow(t, "World actor actions")
	if actions.writer == nil || actions.stage != graphStage {
		t.Error("the actor actions are not a graph-stage writer")
	}
	for _, later := range []string{"World saved Diaries", "World original dead bodies"} {
		if worldCarrierIndex(t, "World actor actions") > worldCarrierIndex(t, later) {
			t.Errorf("the actor actions are written after %s", later)
		}
	}
	head := worldCarrierRow(t, "World session head absence")
	if head.decode[actionsRecord] == nil || head.decode[resumeRecord] == nil {
		t.Error("the session head absence is not decoded at the SAV LOAD and the native resume")
	}
	if worldCarrierIndex(t, "World session head absence") > worldCarrierIndex(t, "World carrier absence") {
		t.Error("the session head absence is decoded after the carrier absence")
	}
	if worldCarrierRow(t, "World session entry").decode[entryRecord] == nil {
		t.Error("the session entry is not published at the original LOAD's entry point")
	}
	dead := worldCarrierRow(t, "World original dead bodies")
	if dead.decode[heldRecord] == nil || dead.build[absentDeadBuild] == nil {
		t.Error("the dead body row does not decode and write the dead body art")
	}
	r := mustNewRecord("Unit")
	body := sim.OriginalDeadRecord{OriginalDeadActor: sim.OriginalDeadActor{ID: 4}}
	if err := buildWorldCarriers(absentDeadBuild, carrierBuild{ms: &Mission{DeadArt: map[sim.EntityID]uint16{4: 42}}, record: &r, body: body}); err != nil {
		t.Fatal(err)
	}
	if got, err := savedStructureValue(&r, "T0E"); err != nil || got != 42 {
		t.Fatalf("the absent dead point wrote T0E %d (%v), want 42", got, err)
	}
}
