package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestBookSelectionUsesExactSparseNodeAndKeepsAliasSlot(t *testing.T) {
	for _, control := range []struct {
		name string
		refs []uint16
		key  uint32
		want uint16
	}{
		{"equal intrinsic ID distinct nodes", []uint16{2, 3}, 102, 2},
		{"shared node distinct slots", []uint16{3, 3}, 102, 2},
		{"ordinary pointer edit", []uint16{2, 3}, 101, 1},
		{"ordinary pointer removal", []uint16{2, 3}, 0, 0},
	} {
		t.Run(control.name, func(t *testing.T) {
			doc := sav.DocumentData{Objects: []sav.DocumentRecordData{
				{Class: "Human", Values: []sav.DocumentValueData{{Name: "U44", Value: control.key}}, RefSlots: []sav.DocumentRefsData{{Name: "Spells", Objects: control.refs}}},
				{Class: "Spell", Values: []sav.DocumentValueData{{Name: "S08", Value: 6}, {Name: "This", Value: 101}}},
				{Class: "Spell", Values: []sav.DocumentValueData{{Name: "S08", Value: 6}, {Name: "This", Value: 102}}},
			}}
			a := currentActionData{Bindings: []currentActionBinding{{ID: 7, Object: 1}}, Actions: sim.ActionContinuations{Actors: []sim.ActorContinuation{{Entity: 7, Current: &sim.ActorCurrentContinuation{AdmittedBookSpell: 2}}}}}
			if err := matchCurrentBookSelection(&doc, &a); err != nil || a.Actions.Actors[0].Current.AdmittedBookSpell != control.want {
				t.Fatal("sparse selection changed", err, a.Actions.Actors[0].Current.AdmittedBookSpell)
			}
		})
	}
}

func TestAbsentAdmittedBookNodeRetainsOrdinaryFieldWithoutClearingHistory(t *testing.T) {
	f := castOrderFront(t)
	a := f.live.world.Actions()
	a.Actors[0].Current.AdmittedBookSpell = 6
	if err := f.live.world.RestoreActions(a, nil); err != nil {
		t.Fatal(err)
	}
	for _, refs := range [][]uint16{nil, {0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 99}} {
		doc := sav.DocumentData{Objects: []sav.DocumentRecordData{{Class: "Human", Values: []sav.DocumentValueData{{Name: "U44", Value: 777}}, RefSlots: []sav.DocumentRefsData{{Name: "Spells", Objects: refs}}}}}
		state := SnapshotSAVDocument{Document: &doc, Objects: &SnapshotSAVObjectBindings{}, Actors: []SnapshotSAVActor{{EntityID: 1, ObjectIndex: 1}}}
		if err := projectCurrentCastWire(&state, f.live.world); err != nil || savedRecordValueForTest(t, doc.Objects[0], "U44") != 777 || f.live.world.Entities()[0].AdmittedBookSpell != 6 {
			t.Fatal("missing node refused SAVE, selected another node or invented a clear", err)
		}
	}
}
