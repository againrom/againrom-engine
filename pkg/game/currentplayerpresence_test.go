package game

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentPlayerPresenceAnchorKeepsByteStrings(t *testing.T) {
	first, second := mustNewRecord("Player"), mustNewRecord("Player")
	mustSetText(&first, "Name", string([]byte{0xff}))
	mustSetText(&second, "Name", string([]byte{0xfe}))
	left, _ := json.Marshal(first)
	right, _ := json.Marshal(second)
	if !bytes.Equal(left, right) {
		t.Fatal("negative control no longer exercises invalid UTF-8 replacement")
	}
	a, err := currentPlayerPresenceAnchor(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := currentPlayerPresenceAnchor(second)
	if err != nil || a == b {
		t.Fatal("distinct ordinary byte strings shared a presence anchor", err)
	}
}

func TestCurrentMixedPlayersKeepTwoSAVCyclesAndOrdinaryEdits(t *testing.T) {
	for _, change := range []string{"none", "money", "name bytes", "formation", "slot", "Group AI", "Group selector"} {
		t.Run(change, func(t *testing.T) {
			f, snapshot := currentMixedPlayerFixture(t)
			players, _ := f.live.world.SavedGroupPlayers()
			formations, _ := f.live.world.SavedPlayerFormations()
			groups, _, _ := f.live.world.SavedGroups()
			raw, err := f.ExportCurrentSave(snapshot, "mixed current Players")
			if err != nil {
				t.Fatal(err)
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || len(a.AbsentPlayers) != 1 || a.GroupPlayers == nil || !*a.GroupPlayers {
				t.Fatal("mixed Player lacks sparse presence", a, err)
			}
			object := a.AbsentPlayers[0].Object
			player := &doc.Objects[object-1]
			if len(player.Groups) != 1 {
				t.Fatal("constructed Player lost its one actor root")
			}
			leaf, _, _ := sav.NativeActions(doc.State)
			name := string([]byte{0xff, 0x80, 'P'})
			switch change {
			case "money":
				mustSetValue(player, "Money", 123456)
			case "name bytes":
				mustSetText(player, "Name", name)
			case "formation":
				crossingRawField(t, player, "PRaw32")[sav.PlayerTailFormationByte] = 23
			case "slot":
				mustSetValue(player, "Slot", 15)
				mustSetValue(player, "SlotAgain", 15)
			case "Group AI":
				crossingRawField(t, &player.Groups[0], "G3C")[0x12] = 47
			case "Group selector":
				mustSetValue(&player.Groups[0], "G1C", 93)
			}
			raw, err = sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			back, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			unchanged, _, _ := sav.NativeActions(back.State)
			if !bytes.Equal(leaf, unchanged) {
				t.Fatal("ordinary Player edit altered its presence policy")
			}
			var previous *sim.World
			if change == "none" {
				previous = f.live.world
			}
			for cycle := range 2 {
				cold := groupDocumentFront1115(t)
				cold.SetDeterministicFrames(true)
				open, town, err := cold.RestoreOriginal(raw)
				if err != nil || town {
					t.Fatal(cycle, town, err)
				}
				if err := cold.App("mixed Player cold LOAD").OpenMission(open); err != nil {
					t.Fatal(err)
				}
				w := cold.live.world
				if previous != nil && previous.Hash() != w.Hash() {
					currentMenuWorldDiagnostics(t, previous, w)
					t.Fatal("current Player presence changed full World", cycle)
				}
				got, present := w.SavedGroupPlayers()
				wantCount := 2
				if change == "none" {
					wantCount = 1
				}
				if !present || len(got) != wantCount || got[0] != players[0] || len(w.Entities()) != 5 {
					t.Fatal("Player promotion changed identities or actor population", got)
				}
				modes, hasModes := w.SavedPlayerFormations()
				if !hasModes || len(modes) != wantCount || modes[0] != formations[0] {
					t.Fatal("mixed Player formation presence changed", modes)
				}
				if change != "none" {
					wantSlot := uint32(0)
					if change == "slot" {
						wantSlot = 15
					}
					if got[1].ID != 2 || got[1].Slot != wantSlot || change == "formation" && modes[1].Mode != 23 || change == "money" && w.Purse(0) != 123456 {
						t.Fatal("ordinary Player edit was replaced by absence", got, modes, w.Purse(0))
					}
				}
				currentGroups, _, _ := w.SavedGroups()
				groupEdit := change == "Group AI" || change == "Group selector"
				wantGroups := len(groups)
				if groupEdit {
					wantGroups++
				}
				if len(currentGroups) != wantGroups || !reflect.DeepEqual(currentGroups[:len(groups)], groups) {
					t.Fatal("ordinary Player policy changed actual Groups", currentGroups)
				}
				if groupEdit {
					g := currentGroups[len(groups)]
					if g.ContainerID != 2 || change == "Group AI" && g.AI[0x12] != 47 || change == "Group selector" && g.Selector != 93 {
						t.Fatal("ordinary Group edit did not promote its exact Player", g)
					}
				}
				if change == "name bytes" {
					b := cold.live.mission.state.savedDocument
					for _, p := range b.GroupBindings.Players {
						if p.ID == 2 {
							found := false
							for _, value := range b.Document.Objects[p.ObjectIndex-1].Texts {
								found = found || value.Name == "Name" && value.Value == name
							}
							// A Player name is written from the map or the hero;
							// an edit to the file's bytes is not World state, so
							// only the edited file itself holds it.
							if found != (cycle == 0) {
								t.Fatal("Player name edit carried outside the World", cycle)
							}
						}
					}
				}
				if previous != nil {
					for range 20 {
						sim.Step(previous, nil)
						sim.Step(w, nil)
						if previous.Hash() != w.Hash() {
							t.Fatal("mixed Player successor diverged", cycle)
						}
					}
				}
				next, label, err := cold.Snapshot(true)
				if err != nil {
					t.Fatal("current mixed Player metadata invalid", err)
				}
				raw, err = cold.ExportCurrentSave(next, label)
				if err != nil {
					t.Fatal("mixed Player second SAVE", cycle, err)
				}
				previous = w
			}
		})
	}
}

func TestCurrentMixedPlayerMalformedPresenceIsAtomic(t *testing.T) {
	f, snapshot := currentMixedPlayerFixture(t)
	raw, err := f.ExportCurrentSave(snapshot, "mixed presence")
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"duplicate", "null", "non-player", "out of range", "empty anchor", "absent carrier", "native Group conflict"} {
		t.Run(fault, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			switch fault {
			case "duplicate":
				a.AbsentPlayers = append(a.AbsentPlayers, a.AbsentPlayers[0])
			case "null":
				a.AbsentPlayers[0].Object = 0
			case "non-player":
				a.AbsentPlayers[0].Object = a.Bindings[0].Object
			case "out of range":
				a.AbsentPlayers[0].Object = uint16(len(doc.Objects) + 1)
			case "empty anchor":
				a.AbsentPlayers[0].Anchor = [32]byte{}
			case "absent carrier":
				*a.GroupPlayers, *a.GroupFormations = false, false
			case "native Group conflict":
				a.AbsentPlayers[0].Object = a.Groups[0].Object
			}
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
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
				err = f.App("malformed mixed Player").OpenMission(open)
			}
			if err == nil {
				t.Fatal("malformed current Player presence accepted")
			}
			after, _, snapshotErr := f.Snapshot(true)
			if snapshotErr != nil || f.live.world.Hash() != hash || !reflect.DeepEqual(before, after) {
				t.Fatal("rejected Player presence changed the current session", snapshotErr)
			}
		})
	}
}
