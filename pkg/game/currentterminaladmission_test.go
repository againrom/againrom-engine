package game

import (
	"encoding/json"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentTerminalActorAdmissionRequiresExactDeadRootAndPureValue(t *testing.T) {
	actor := &poolFixtureActor{mapID: 91, cell: 0x100f, hp: uint16(65536 - 601), maxHP: 31, stage: 4}
	raw := savedContainer(poolFixtureBody(nil, []*poolFixtureActor{actor}))
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	const id sim.EntityID = 34
	terminal := sim.CurrentTerminalActor{ID: id, Cell: 0x100f, HP: -601, Stage: 4}

	for _, tc := range []struct {
		name       string
		marker     *sim.CurrentTerminalActor
		rooted     bool
		extraValue bool
		wantErr    bool
	}{
		{name: "exact terminal tombstone", marker: &terminal, rooted: true},
		{name: "missing terminal marker", rooted: true, wantErr: true},
		{name: "marker names another actor", marker: &sim.CurrentTerminalActor{ID: id + 1, Cell: terminal.Cell, HP: terminal.HP, Stage: terminal.Stage}, rooted: true, wantErr: true},
		{name: "terminal object is not a dead root", marker: &terminal, rooted: false, wantErr: true},
		{name: "terminal marker carries unrelated value", marker: &terminal, rooted: true, extraValue: true, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateData := sav.NewCityStateData("terminal admission")
			doc := sav.DocumentData{Version: sav.DocumentDataVersion, Head: sav.DocumentHeadData{Mission: 10},
				World: &sav.DocumentWorldData{}, Objects: []sav.DocumentRecordData{{Class: "Unit"}}, DeadActors: []uint16{1},
				State: sav.DocumentStateData{RootKind: stateData.RootKind, DirectoryRecords: stateData.DirectoryRecords, ValueRecords: stateData.ValueRecords}}
			origins := []sav.DocumentObjectOrigin{{ArchiveIndex: 1, ObjectIndex: 1}}
			object := doc.DeadActors[0]
			if !tc.rooted {
				doc.DeadActors = nil
			}
			value := sim.ActorValues{CurrentTerminal: tc.marker}
			if tc.extraValue {
				value.SuppressCorpseLoot = true
			}
			a := currentActionData{Version: 1, Bindings: []currentActionBinding{{ID: id, Object: object}}, Values: map[sim.EntityID]sim.ActorValues{id: value}}
			encoded, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, encoded); err != nil {
				t.Fatal(err)
			}
			state := &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &doc}
			graph, err := currentActorGraph(file, state, origins)
			if (err != nil) != tc.wantErr {
				t.Fatalf("current actor graph err = %v, wantErr %t", err, tc.wantErr)
			}
			if err == nil && !graph.CurrentPopulation {
				t.Fatal("terminal actor lost the explicit current-population boundary")
			}
			for _, actor := range graph.Actors {
				if actor.CurrentEntity {
					t.Fatalf("terminal actor entered current population: %+v", graph)
				}
			}
		})
	}
}
