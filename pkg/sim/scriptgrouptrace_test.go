package sim

import "testing"

func TestScriptGroupObservationIsACopyOfTheNamedRecords(t *testing.T) {
	ents := sttEnts()
	ents[0].Group, ents[0].Owner = 3, 1
	w := sttWorld(t, nil, ents)
	before := w.Hash()
	got := ObserveScriptGroups(w, 3)
	if len(got) == 0 {
		t.Fatal("the group fixture exposes no group 3")
	}
	got[0].Order = 0xff
	got[0].CommandedX = 0x7fffffff
	again := ObserveScriptGroups(w, 3)
	if again[0].Order == got[0].Order || again[0].CommandedX == got[0].CommandedX {
		t.Fatalf("mutating the observation reached the world: first=%+v again=%+v", got[0], again[0])
	}
	if w.Hash() != before {
		t.Fatalf("observing a group moved the world hash: %#x -> %#x", before, w.Hash())
	}
	w.savedGroups = &savedGroupState{PlayersPresent: true,
		Players: []SavedGroupPlayer{{ID: 9, Slot: 2}},
		Groups:  []SavedGroup{{ID: 1, Selector: 3, OwnerID: 9, AI: [76]byte{10: 11, 11: 16, 0x20: 4}}}}
	current := ObserveScriptGroups(w, 3)
	if len(current) != 1 || current[0] != (ScriptGroupState{Owner: 2, Group: 3, Order: 4, CommandedX: 11, CommandedY: 16}) {
		t.Fatalf("observation used dormant legacy groups: %+v", current)
	}
	current[0].Order = 1
	if ObserveScriptGroups(w, 3)[0].Order != 4 || len(ObserveScriptGroups(w, 99)) != 0 {
		t.Fatal("saved group observation aliases its owner or invents membership")
	}
}
