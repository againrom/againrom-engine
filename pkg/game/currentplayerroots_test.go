package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/sim"
)

func TestCurrentPlayerSourceImportWithoutGroupCarrier(t *testing.T) {
	f := participantFront(t)
	state, err := cloneSavedDocument(f.live.mission.state.savedDocument)
	if err != nil {
		t.Fatal(err)
	}
	state.GroupBindings, state.PlayerPurses, state.PlayerRoots = nil, nil, nil
	state.Actors, state.Objects, state.ActorEffects, state.WorldEffects = nil, nil, nil, nil
	state.Document.Head.CounterA, state.Document.Head.CounterB = 0, 0
	w, err := sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var objects []uint16
	for _, object := range state.Document.Players {
		if object != 0 && !slices.Contains(objects, object) {
			objects = append(objects, object)
		}
	}
	state.Document.Players = append(state.Document.Players, objects[0])
	for i, object := range objects {
		mustSetValue(&state.Document.Objects[object-1], "Slot", 7)
		mustSetValue(&state.Document.Objects[object-1], "Participant", []uint32{0, 0xf1234567}[i])
	}
	ms := &Mission{World: w}
	if err := importSavedDocument(ms, state, nil); err != nil {
		t.Fatal(err)
	}
	state = ms.savedDocument
	players := []sim.SavedGroupPlayer{{ID: 1, Slot: 7}, {ID: 2, Slot: 7}}
	want := []sim.PlayerParticipant{{PlayerID: 1}, {PlayerID: 2, Value: 0xf1234567}}
	if got, present := w.CurrentPlayers(); !present || !reflect.DeepEqual(got, players) {
		t.Fatal("source roots disappeared with absent Group carrier", got, present)
	}
	if got, present := w.PlayerParticipants(); !present || !reflect.DeepEqual(got, want) {
		t.Fatal("source aliases or shared Slot merged Participant words", got, present)
	}
	if err := w.SetPlayerParticipant(1, 0x87654321); err != nil {
		t.Fatal(err)
	}
	if err := projectSavedPlayerParticipants(state, w); err != nil {
		t.Fatal(err)
	}
	if actorProjectionValue(t, state.Document.Objects[objects[0]-1], "Participant") != 0x87654321 || actorProjectionValue(t, state.Document.Objects[objects[1]-1], "Participant") != want[1].Value {
		t.Fatal("independent source root did not own its ordinary current word")
	}
	if _, _, present := w.SavedGroups(); present {
		t.Fatal("source Player import created command dispatch state")
	}
	copy, err := cloneSavedDocument(state)
	if err != nil || copy.PlayerRoots == nil || len(*copy.PlayerRoots) != 2 {
		t.Fatal("source roots lost their independent snapshot receipts", err)
	}
	for i, root := range *copy.PlayerRoots {
		got := actorProjectionValue(t, copy.Document.Objects[root.ObjectIndex-1], "This")
		want := actorProjectionValue(t, state.Document.Objects[(*state.PlayerRoots)[i].ObjectIndex-1], "This")
		if root.ID != uint32(i+1) || got != want {
			t.Fatal("document reindex moved exact source Player ownership", root, got, want)
		}
	}
	(*copy.PlayerRoots)[0].ObjectIndex = (*copy.PlayerRoots)[1].ObjectIndex
	if _, err := cloneSavedDocument(copy); err == nil {
		t.Fatal("duplicate exact Player source receipt was accepted")
	}
}
