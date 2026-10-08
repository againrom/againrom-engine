package game

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func participantFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := purseFront1115(t)
	open, town, err := f.RestoreOriginal(purseLiteral1115(t, f, [2]uint32{1, 7}, [2]uint32{50, 75}))
	if err != nil || town {
		t.Fatal("Participant source import failed", town, err)
	}
	if err := f.App("current Participant").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestCurrentParticipantCaptureOverridesStaleDocumentAndColdSave(t *testing.T) {
	f := participantFront(t)
	want := []sim.PlayerParticipant{{PlayerID: 1, Value: 0}, {PlayerID: 2, Value: 1}}
	if got, present := f.live.world.PlayerParticipants(); !present || !reflect.DeepEqual(got, want) {
		t.Fatal("ordinary exact Player words have no current carrier", got, present)
	}
	if err := f.live.world.SetPlayerParticipant(2, 0xf1234567); err != nil {
		t.Fatal(err)
	}
	want[1].Value = 0xf1234567
	s := groupDocumentSnapshot(t, f)
	// The captured World owns this word, even when the caller supplies old DTOs.
	for i := range s.SavedDocument.Document.Objects {
		r := &s.SavedDocument.Document.Objects[i]
		if r.Class == "Player" && actorProjectionValue(t, *r, "This") == newGroupRight1115 {
			mustSetValue(r, "Participant", 31)
		}
	}
	raw, err := f.ExportCurrentSave(s, "current Participant")
	if err != nil {
		t.Fatal(err)
	}
	for cycle := range 2 {
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range doc.Objects {
			if r.Class == "Player" && actorProjectionValue(t, r, "This") == newGroupRight1115 && actorProjectionValue(t, r, "Participant") != want[1].Value {
				t.Fatal("current Participant did not reach ordinary SAV word")
			}
		}
		cold := purseFront1115(t)
		open, town, err := cold.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatal("current Participant cold LOAD failed", cycle, town, err)
		}
		if err := cold.App("current Participant continuation").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		if got, present := cold.live.world.PlayerParticipants(); !present || !reflect.DeepEqual(got, want) {
			t.Fatal("cold LOAD lost exact Player Participant", got, present)
		}
		cold.live.tick()
		s = groupDocumentSnapshot(t, cold)
		raw, err = cold.ExportCurrentSave(s, "following Participant")
		if err != nil {
			t.Fatal(err)
		}
		if cycle == 0 {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			leaf, _, err := sav.NativeActions(doc.State)
			if err != nil {
				t.Fatal(err)
			}
			want[1].Value = 0x87654321
			for i := range doc.Objects {
				r := &doc.Objects[i]
				if r.Class == "Player" && actorProjectionValue(t, *r, "This") == newGroupRight1115 {
					mustSetValue(r, "Participant", want[1].Value)
				}
			}
			raw, err = sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			back, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			unchanged, _, err := sav.NativeActions(back.State)
			if err != nil || !bytes.Equal(leaf, unchanged) {
				t.Fatal("ordinary Participant loss control changed current supplement", err)
			}
		}
	}
}

func TestCurrentParticipantPresenceKeepsZeroAndAbsentDocument(t *testing.T) {
	f := participantFront(t)
	if err := f.live.world.SetPlayerParticipant(2, 0); err != nil {
		t.Fatal(err)
	}
	s := groupDocumentSnapshot(t, f)
	for _, r := range s.SavedDocument.Document.Objects {
		if r.Class == "Player" && actorProjectionValue(t, r, "This") == newGroupRight1115 && actorProjectionValue(t, r, "Participant") != 0 {
			t.Fatal("explicit zero retained stale Document word")
		}
	}
	if err := f.live.world.RestorePlayerParticipants(nil, false); err != nil {
		t.Fatal(err)
	}
	state := f.live.mission.state.savedDocument
	for i := range state.Document.Objects {
		r := &state.Document.Objects[i]
		if r.Class == "Player" && actorProjectionValue(t, *r, "This") == newGroupRight1115 {
			mustSetValue(r, "Participant", 0x12345678)
		}
	}
	s = groupDocumentSnapshot(t, f)
	for _, r := range s.SavedDocument.Document.Objects {
		if r.Class == "Player" && actorProjectionValue(t, r, "This") == newGroupRight1115 && actorProjectionValue(t, r, "Participant") != 0x12345678 {
			t.Fatal("absent carrier invented a current word")
		}
	}
	raw, err := f.ExportCurrentSave(s, "absent Participant")
	if err != nil {
		t.Fatal(err)
	}
	cold := purseFront1115(t)
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("absent Participant cold LOAD failed", town, err)
	}
	if err := cold.App("absent Participant continuation").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if got, present := cold.live.world.PlayerParticipants(); present || len(got) != 0 {
		t.Fatal("ordinary word synthesized an absent current carrier", got)
	}
}

func TestCurrentParticipantProjectsAllExactPlayersWithSharedSlot(t *testing.T) {
	f := participantFront(t)
	state, err := cloneSavedDocument(f.live.mission.state.savedDocument)
	if err != nil {
		t.Fatal(err)
	}
	w, err := sim.NewWorld(1, f.live.world.Bounds(), sim.ModeCanonical, nil, nil)
	if err != nil || w.ImportSavedGroups(nil, nil) != nil || w.ImportSavedGroupPlayers([]sim.SavedGroupPlayer{{ID: 1, Slot: 7}, {ID: 2, Slot: 7}}, nil) != nil {
		t.Fatal("shared-slot exact Player fixture failed", err)
	}
	want := []sim.PlayerParticipant{{PlayerID: 1, Value: 0xf1234567}, {PlayerID: 2, Value: 0}}
	if err := w.RestorePlayerParticipants(want, true); err != nil {
		t.Fatal(err)
	}
	if err := projectSavedPlayerParticipants(state, w); err != nil {
		t.Fatal(err)
	}
	for i, binding := range nativeSavedPlayerBindings(state.GroupBindings) {
		if got := actorProjectionValue(t, state.Document.Objects[binding.ObjectIndex-1], "Participant"); got != want[i].Value {
			t.Fatal("shared semantic Slot merged distinct Participant words", binding.ID, got)
		}
	}
}

func TestCurrentPlayerIdentityAndParticipantGeneratedColdContinuation(t *testing.T) {
	f, s, w := currentPlayerSlotFixture(t, 1)
	players := []sim.SavedGroupPlayer{{ID: 101, Slot: 1}, {ID: 205, Slot: 0}}
	want := []sim.PlayerParticipant{{PlayerID: 101, Value: 0}, {PlayerID: 205, Value: 0xf1234567}}
	if err := w.RestoreCurrentPlayers(players, want, true); err != nil {
		t.Fatal(err)
	}
	var err error
	s.World, err = w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "generated current Player")
	if err != nil {
		t.Fatal(err)
	}
	for cycle := range 2 {
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || a.PlayerIdentities == nil || len(*a.PlayerIdentities) != 2 || a.GroupParticipants == nil || !*a.GroupParticipants {
			t.Fatal("generated exact Player identities have no current transport", err)
		}
		for i, row := range *a.PlayerIdentities {
			if row.ID != players[i].ID || actorProjectionValue(t, doc.Objects[row.Object-1], "Participant") != want[i].Value {
				t.Fatal("current identity or word did not reach its exact ordinary root", row)
			}
		}
		if cycle == 0 {
			leaf, _, _ := sav.NativeActions(doc.State)
			want[1].Value = 0x87654321
			mustSetValue(&doc.Objects[(*a.PlayerIdentities)[1].Object-1], "Participant", want[1].Value)
			raw, err = sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			back, err := sav.DecodeDocumentData(raw)
			unchanged, _, _ := sav.NativeActions(back.State)
			if err != nil || !bytes.Equal(leaf, unchanged) {
				t.Fatal("ordinary Participant control changed current identity metadata", err)
			}
		}
		cold := cellStateFront(t)
		cold.Campaign, cold.Table = f.Campaign, f.Table
		open, town, err := cold.RestoreOriginal(raw)
		if err == nil && !town {
			err = cold.App("current Player continuation").OpenMission(open)
		}
		if err != nil || town {
			t.Fatal("generated current Player cold LOAD failed", cycle, town, err)
		}
		if got, present := cold.live.world.CurrentPlayers(); !present || !reflect.DeepEqual(got, players) {
			t.Fatal("current identity/Slot changed after ordinary LOAD", got)
		}
		if got, present := cold.live.world.PlayerParticipants(); !present || !reflect.DeepEqual(got, want) {
			t.Fatal("ordinary current Player words were replaced by stale metadata", got)
		}
		cold.live.tick()
		s = groupDocumentSnapshot(t, cold)
		raw, err = cold.ExportCurrentSave(s, "following generated Player")
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCurrentPlayerIdentityMalformedJoinKeepsSession(t *testing.T) {
	f := participantFront(t)
	raw, err := f.ExportCurrentSave(groupDocumentSnapshot(t, f), "exact Player identity")
	if err != nil {
		t.Fatal(err)
	}
	for _, edit := range []func(*currentActionData){
		func(a *currentActionData) { (*a.PlayerIdentities)[0].ID = 0 },
		func(a *currentActionData) { (*a.PlayerIdentities)[1].ID = (*a.PlayerIdentities)[0].ID },
		func(a *currentActionData) { (*a.PlayerIdentities)[0].Object = (*a.PlayerIdentities)[1].Object },
		func(a *currentActionData) { *a.PlayerIdentities = (*a.PlayerIdentities)[:1] },
	} {
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || a.PlayerIdentities == nil {
			t.Fatal("exact current identity transport absent", err)
		}
		edit(a)
		leaf, err := json.Marshal(a)
		if err != nil || sav.SetNativeActions(&doc.State, leaf) != nil {
			t.Fatal(err)
		}
		edited, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		before, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		hash := f.live.world.Hash()
		open, town, err := f.RestoreOriginal(edited)
		if err == nil && !town {
			err = f.App("invalid current Player identity").OpenMission(open)
		}
		if err == nil {
			t.Fatal("invalid exact current Player identity was accepted")
		}
		after, _, err := f.Snapshot(true)
		if err != nil || f.live.world.Hash() != hash || !reflect.DeepEqual(before, after) {
			t.Fatal("rejected current Player join changed the session", err)
		}
	}
}
