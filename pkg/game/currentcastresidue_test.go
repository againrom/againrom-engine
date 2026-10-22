package game

import (
	"encoding/json"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentFirstGameBookCompletionIsNotAnIdleOrderResidue(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		current, book, second bool
		wantResidue           bool
	}{
		{"current book", true, true, false, false},
		{"current idle", true, false, false, true},
		{"original", false, false, false, true},
		{"second game book", true, true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := sav.DocumentData{Objects: []sav.DocumentRecordData{
				{Class: "Human", Values: []sav.DocumentValueData{{Name: "U136", Value: 1}, {Name: "U6C", Value: 0}},
					Raw:      []sav.DocumentRawData{{Name: "U58", Bytes: make([]byte, 4)}, {Name: "U158", Bytes: make([]byte, 148)}},
					RefSlots: []sav.DocumentRefsData{{Name: "Spells", Objects: []uint16{2}}}},
				{Class: "Spell", Values: []sav.DocumentValueData{{Name: "This", Value: 123}}},
			}}
			if tc.current {
				a := currentActionData{Version: 1, Bindings: []currentActionBinding{{ID: 1, Object: 1}}}
				if tc.book {
					a.Actions.Books = []sim.BookContinuation{{Caster: 1, Spell: 1, Complete: true}}
				}
				data, err := json.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				if err := sav.SetNativeActions(&doc.State, data); err != nil {
					t.Fatal(err)
				}
			}
			var script *sim.Script
			if tc.second {
				var err error
				script, err = sim.NewROM2Script(nil, nil, nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
				[]sim.Entity{{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100}}, script)
			if err != nil {
				t.Fatal(err)
			}
			state := SnapshotSAVDocument{Document: &doc, Actors: []SnapshotSAVActor{{EntityID: 1, ObjectIndex: 1}}}
			held := map[sim.EntityID]sim.HeldOrder{}
			if err := holdOrderResidues(&state, w, held); err != nil {
				t.Fatal(err)
			}
			_, residue := held[1]
			if residue != tc.wantResidue || residue && held[1] != (sim.HeldOrder{Kind: sim.HeldOrderResidue, Complete: 1}) {
				t.Fatalf("held orders %+v, want residue %v", held, tc.wantResidue)
			}
		})
	}
}
