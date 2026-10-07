package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentPlayerSlotFixture(t *testing.T, last uint32) (*FrontEnd, Snapshot, *sim.World) {
	t.Helper()
	f, s, _ := partialCurrentGraph(t)
	var entities []sim.Entity
	for owner := uint32(0); owner <= last; owner++ {
		entities = append(entities, sim.Entity{ID: sim.EntityID(owner), X: int32(owner) + 5, Y: 5, Owner: owner,
			HP: 31, MaxHP: 31, TypeID: 1, Speed: 1, TokenSize: 1})
	}
	w, err := sim.NewWorld(31, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	for owner := uint32(0); owner <= last; owner++ {
		w.SetPurse(owner, 0xf1234500+owner)
	}
	s.Party, s.CurrentPartyIDs, s.CurrentRoster, s.SavedDocument = nil, nil, nil, nil
	s.Residue.VisualIdentities = nil
	s.Residue.VisualNext = sim.EntityID(last + 1)
	s.World, err = w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return f, s, w
}

func TestCurrentPlayerSlotsKeepExactOwnerAndOrdinaryMoney(t *testing.T) {
	for _, last := range []uint32{1, 16} {
		f, s, original := currentPlayerSlotFixture(t, last)
		raw, err := f.ExportCurrentSave(s, "native slot zero")
		if err != nil {
			t.Fatal(err)
		}
		for _, change := range []string{"none", "money", "slot"} {
			if last == 16 && change == "slot" {
				continue
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || len(a.PlayerSlots) != 1 || a.PlayerSlots[0].Native != 0 {
				t.Fatalf("missing exact native slot policy: %+v %v", a, err)
			}
			p := a.PlayerSlots[0]
			if p.Wire == 1 || last == 16 && p.Wire != 16 {
				t.Fatal("transport slot borrowed the real self slot or changed full-population policy", p)
			}
			leaf, _, _ := sav.NativeActions(doc.State)
			r := &doc.Objects[p.Object-1]
			switch change {
			case "money":
				mustSetValue(r, "Money", 0x87654321)
			case "slot":
				mustSetValue(r, "Slot", 15)
				mustSetValue(r, "SlotAgain", 15)
			}
			changed, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			back, err := sav.DecodeDocumentData(changed)
			if err != nil {
				t.Fatal(err)
			}
			unchanged, _, _ := sav.NativeActions(back.State)
			if !bytes.Equal(leaf, unchanged) {
				t.Fatal("ordinary Player edit changed supplement")
			}
			for cycle := 0; cycle < 2; cycle++ {
				cold := cellStateFront(t)
				cold.Campaign, cold.Table = f.Campaign, f.Table
				open, town, err := cold.RestoreOriginal(changed)
				if err == nil && !town {
					err = cold.App("current owner slot").OpenMission(open)
				}
				if err != nil || town {
					t.Fatalf("last=%d change=%s cycle=%d: %v", last, change, cycle, err)
				}
				w := cold.live.world
				for _, e := range w.Entities() {
					want := uint32(e.ID)
					if change == "slot" && e.ID == 0 {
						want = 15
					}
					if e.Owner != want {
						t.Fatalf("last=%d change=%s cycle=%d actor%d owner=%d want%d", last, change, cycle, e.ID, e.Owner, want)
					}
				}
				for owner := uint32(0); owner <= last; owner++ {
					want := original.Purse(owner)
					if owner == 0 && change == "money" {
						want = 0x87654321
					}
					if owner == 0 && change == "slot" {
						want = 0
					}
					if w.Purse(owner) != want {
						t.Fatalf("last=%d change=%s cycle=%d purse%d=%x want%x", last, change, cycle, owner, w.Purse(owner), want)
					}
				}
				if change == "slot" && w.Purse(15) != original.Purse(0) {
					t.Fatal("ordinary Slot edit did not move its current Money")
				}
				if cycle == 0 {
					next, label, err := cold.Snapshot(true)
					if err != nil {
						t.Fatal(err)
					}
					changed, err = cold.ExportCurrentSave(next, label)
					if err != nil {
						t.Fatal("second SAVE", err)
					}
				}
			}
		}
	}
}

func sharedCurrentPlayerFixture(t *testing.T, slot uint32) (*FrontEnd, Snapshot, []sim.SavedGroup, []sim.SavedGroupPlayer) {
	t.Helper()
	f, s, source := currentPlayerSlotFixture(t, 1)
	state, err := f.materializeCurrentWorld(s, source)
	if err != nil {
		t.Fatal(err)
	}
	entities := source.Entities()
	for i := range entities {
		entities[i].Owner = slot
	}
	w, err := sim.NewWorld(31, source.Bounds(), sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	var groups []sim.SavedGroup
	var players []sim.SavedGroupPlayer
	var containers []sim.SavedGroupContainer
	for i, player := range state.GroupBindings.Players {
		r := &state.Document.Objects[player.ObjectIndex-1]
		mustSetValue(r, "Slot", slot)
		mustSetValue(r, "SlotAgain", slot)
		mustSetValue(r, "Money", 7)
		players = append(players, sim.SavedGroupPlayer{ID: player.ID, Slot: slot})
		for j := range state.GroupBindings.Groups {
			b := &state.GroupBindings.Groups[j]
			if b.PlayerObject != player.ObjectIndex {
				continue
			}
			b.RootOnly, b.Authored, b.Owner.Owner = false, false, slot
			refs, present := savedObjectRefs(&r.Groups[b.InlineIndex], "Actors")
			if !present || len(refs) != 1 {
				t.Fatal("shared fixture needs exact singleton Groups", refs, present)
			}
			var member sim.SavedGroupMember
			for _, actor := range state.Actors {
				if actor.ObjectIndex == refs[0] {
					member = sim.SavedGroupMember{Entity: actor.EntityID, Bound: true}
				}
			}
			if !member.Bound {
				t.Fatal("fixture lost exact actor root")
			}
			group := sim.SavedGroup{ID: b.ID, Selector: uint32(i + 7), Owner: sim.SavedGroupReference{Class: 1, Key: b.Owner.Key, Owner: slot}, Members: []sim.SavedGroupMember{member}}
			groups = append(groups, group)
			containers = append(containers, sim.SavedGroupContainer{GroupID: b.ID, PlayerID: player.ID})
		}
	}
	if len(groups) != 2 || len(players) != 2 {
		t.Fatal("fixture merged distinct native Player/Group identities", groups, players)
	}
	if err := w.ImportSavedGroups(groups, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedGroupPlayers(players, containers); err != nil {
		t.Fatal(err)
	}
	w.SetPurse(slot, 0xf1234567)
	rows, err := savedPlayerPurseRows(state.Document, state.GroupBindings)
	if err != nil {
		t.Fatal(err)
	}
	state.PlayerPurses = &SnapshotSAVPlayerPurses{Version: 1, Players: rows}
	state.Document.Players = append(state.Document.Players, 0, state.Document.Players[0])
	s.SavedDocument = state
	s.World, err = w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	groups, _, _ = w.SavedGroups()
	return f, s, groups, players
}

func TestCurrentPlayerSlotsSharePurseWithoutMergingIdentities(t *testing.T) {
	for _, slot := range []uint32{0, 1} {
		for _, edit := range []string{"none", "money", "slot", "both slots"} {
			t.Run(fmt.Sprintf("native%d/%s", slot, edit), func(t *testing.T) {
				f, s, groups, players := sharedCurrentPlayerFixture(t, slot)
				raw, err := f.ExportCurrentSave(s, "shared current purse")
				if err != nil {
					t.Fatal(err)
				}
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				a, err := readCurrentActions(&doc)
				if err != nil || len(a.PlayerSlots) != 2 || !a.PlayerSlots[0].Shared || !a.PlayerSlots[1].Shared || a.PlayerSlots[0].Object == a.PlayerSlots[1].Object {
					t.Fatal("shared purse lost exact Player policy", a, err)
				}
				leaf, _, _ := sav.NativeActions(doc.State)
				for _, row := range a.PlayerSlots {
					value, err := savedStructureValue(&doc.Objects[row.Object-1], "Money")
					if err != nil || value != 0xf1234567 || row.Native != slot {
						t.Fatal("SAVE kept stale per-root Money instead of current shared purse", value, row, err)
					}
				}
				wantMoney := uint32(0xf1234567)
				if edit == "money" {
					wantMoney = 0x87654321
					for _, row := range a.PlayerSlots {
						mustSetValue(&doc.Objects[row.Object-1], "Money", wantMoney)
					}
				}
				if edit == "slot" {
					mustSetValue(&doc.Objects[a.PlayerSlots[1].Object-1], "Slot", 15)
					mustSetValue(&doc.Objects[a.PlayerSlots[1].Object-1], "SlotAgain", 15)
					players[1].Slot = 15
				}
				if edit == "both slots" {
					for i, row := range a.PlayerSlots {
						mustSetValue(&doc.Objects[row.Object-1], "Slot", 15)
						mustSetValue(&doc.Objects[row.Object-1], "SlotAgain", 15)
						players[i].Slot = 15
					}
				}
				raw, err = sav.EncodeDocumentData(doc)
				if err != nil {
					t.Fatal(err)
				}
				back, _ := sav.DecodeDocumentData(raw)
				unchanged, _, _ := sav.NativeActions(back.State)
				if !bytes.Equal(leaf, unchanged) {
					t.Fatal("ordinary shared Player edit changed supplement")
				}
				for cycle := 0; cycle < 2; cycle++ {
					cold := cellStateFront(t)
					cold.Campaign, cold.Table = f.Campaign, f.Table
					open, town, err := cold.RestoreOriginal(raw)
					if err == nil && !town {
						err = cold.App("shared purse LOAD").OpenMission(open)
					}
					if err != nil || town {
						t.Fatalf("cycle%d: %v", cycle, err)
					}
					w := cold.live.world
					gotPlayers, present := w.SavedGroupPlayers()
					if !present || !reflect.DeepEqual(gotPlayers, players) {
						t.Fatal("equal native slots merged, reordered or remapped exact Players", gotPlayers, players)
					}
					gotGroups, _, present := w.SavedGroups()
					if !present || len(gotGroups) != len(groups) {
						t.Fatal("equal native slots changed Group population", gotGroups)
					}
					for i, g := range gotGroups {
						want := groups[i]
						owner := slot
						if edit == "both slots" || edit == "slot" && g.ContainerID == players[1].ID {
							owner = 15
						}
						if g.ID != want.ID || g.ContainerID != want.ContainerID || g.Selector != want.Selector || g.Owner.Owner != owner || len(g.Members) != 1 || g.Members[0].Entity != want.Members[0].Entity {
							t.Fatal("shared purse changed exact Group ownership or membership", g, want)
						}
						for _, actor := range w.Entities() {
							if actor.ID == g.Members[0].Entity && actor.Owner != owner {
								t.Fatal("Slot edit affected a different Player's actor", actor.ID, actor.Owner, owner)
							}
						}
					}
					oldMoney := wantMoney
					if edit == "both slots" {
						oldMoney = 0
					}
					if w.Purse(slot) != oldMoney || (edit == "slot" || edit == "both slots") && w.Purse(15) != wantMoney {
						t.Fatal("ordinary Money did not supply exact shared/split purses", w.Purse(slot), w.Purse(15), wantMoney)
					}
					roots := cold.live.mission.state.savedDocument.Document.Players
					if len(roots) != 4 || roots[0] == roots[1] || roots[2] != 0 || roots[3] != roots[0] {
						t.Fatal("LOAD changed null/repeated/distinct Player roots", roots)
					}
					if cycle == 0 {
						wantMoney += 31
						if edit != "both slots" {
							w.SetPurse(slot, wantMoney)
						}
						if edit == "slot" || edit == "both slots" {
							w.SetPurse(15, wantMoney)
						}
						next, label, err := cold.Snapshot(true)
						if err != nil {
							t.Fatal(err)
						}
						raw, err = cold.ExportCurrentSave(next, label)
						if err != nil {
							t.Fatal("second SAVE", err)
						}
					}
				}
			})
		}
	}
}

func TestCurrentPlayerSlotsRetainUnmodeledDistinctMoney(t *testing.T) {
	for _, slot := range []uint32{0, 1} {
		f, s, _, _ := sharedCurrentPlayerFixture(t, slot)
		raw, err := f.ExportCurrentSave(s, "unmodeled distinct Money")
		if err != nil {
			t.Fatal(err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		want := []uint32{0xf1234567, 0x13579}
		mustSetValue(&doc.Objects[doc.Players[1]-1], "Money", want[1])
		raw, err = sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		for cycle := 0; cycle < 2; cycle++ {
			cold := cellStateFront(t)
			cold.Campaign, cold.Table = f.Campaign, f.Table
			open, town, err := cold.RestoreOriginal(raw)
			if err == nil && !town {
				err = cold.App("distinct Money debt").OpenMission(open)
			}
			if err != nil || town {
				t.Fatalf("slot%d cycle%d: %v", slot, cycle, err)
			}
			// The native model has one scalar for this slot. Keep the distinct
			// ordinary values and leave the import uncovered, without a winner.
			if got := cold.live.world.Purse(slot); got != 0 {
				t.Fatal("unmodeled ordinary Money silently selected a native purse", got)
			}
			next, label, err := cold.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			raw, err = cold.ExportCurrentSave(next, label)
			if err != nil {
				t.Fatal("unmodeled distinct Money refused SAVE", err)
			}
			doc, err = sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			for i, value := range want {
				got, err := savedStructureValue(&doc.Objects[doc.Players[i]-1], "Money")
				if err != nil || got != value {
					t.Fatal("unmodeled ordinary Money was overwritten", slot, cycle, i, got, value, err)
				}
			}
		}
	}
}

func TestCurrentPlayerSlotMalformedPolicyIsAtomic(t *testing.T) {
	f, s, _ := currentPlayerSlotFixture(t, 1)
	raw, err := f.ExportCurrentSave(s, "slot policy validation")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"duplicate", "actor", "native", "wire", "unshared"} {
		doc, _ := sav.DecodeDocumentData(raw)
		a, _ := readCurrentActions(&doc)
		switch kind {
		case "duplicate":
			a.PlayerSlots = append(a.PlayerSlots, a.PlayerSlots[0])
		case "actor":
			a.PlayerSlots[0].Object = a.Bindings[0].Object
		case "native":
			a.PlayerSlots[0].Native = 1
		case "wire":
			a.PlayerSlots[0].Wire = 0
		case "unshared":
			a.PlayerSlots[0].Shared = true
		}
		leaf, _ := json.Marshal(a)
		if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
			t.Fatal(err)
		}
		bad, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		prior, hash := f.live, f.live.world.Hash()
		if _, _, err := f.RestoreOriginal(bad); err == nil || f.live != prior || f.live.world.Hash() != hash {
			t.Fatal("malformed native Player mapping changed active state", kind, err)
		}
	}
}

func TestCurrentPlayerSlotsKeepDistinctNativeTriggerAcrossTwoCycles(t *testing.T) {
	for _, slot := range []uint32{0, 1} {
		f, s, old := partialCurrentGraph(t)
		entities := old.Entities()
		for i := range entities {
			entities[i].Owner = slot
		}
		w, err := sim.NewWorld(11, old.Bounds(), sim.ModeCanonical, nil, entities)
		if err != nil {
			t.Fatal(err)
		}
		groups, orders, _ := old.SavedGroups()
		groups[0].Owner.Owner, groups[0].ContainerID, groups[0].OwnerID = slot, 0, 0
		if err := w.ImportSavedGroups(groups, orders); err != nil {
			t.Fatal(err)
		}
		state := s.SavedDocument
		player := state.GroupBindings.Players[0]
		if err := w.ImportSavedGroupPlayers([]sim.SavedGroupPlayer{{ID: player.ID, Slot: slot}}, []sim.SavedGroupContainer{{GroupID: groups[0].ID, PlayerID: player.ID}}); err != nil {
			t.Fatal(err)
		}
		trigger := uint32(0xf1234567)
		formations := []sim.SavedPlayerFormation{{PlayerID: player.ID, CommandID: int16(slot), TriggerID: trigger, Mode: 5}}
		if err := w.ImportSavedPlayerFormations(formations, []sim.SavedGroupOwner{{GroupID: groups[0].ID, PlayerID: player.ID}}); err != nil {
			t.Fatal(err)
		}
		state.GroupBindings.FormationsPresent = true
		state.GroupBindings.Groups[0].Owner.Owner = slot
		r := &state.Document.Objects[player.ObjectIndex-1]
		mustSetValue(r, "Slot", slot)
		mustSetValue(r, "SlotAgain", trigger)
		purses, err := savedPlayerPurseRows(state.Document, state.GroupBindings)
		if err != nil {
			t.Fatal(err)
		}
		state.PlayerPurses = &SnapshotSAVPlayerPurses{Version: 1, Players: purses}
		tail, err := savedActorRaw(r, "PRaw32", sav.PlayerTailLen)
		if err != nil {
			t.Fatal(err)
		}
		tail[sav.PlayerTailFormationByte] = 5
		s.World, err = w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(s, "distinct native trigger")
		if err != nil {
			t.Fatal(err)
		}
		for _, edit := range []bool{false, true} {
			input, owner := raw, slot
			expected := append([]sim.SavedPlayerFormation(nil), formations...)
			if edit {
				doc, err := sav.DecodeDocumentData(input)
				if err != nil {
					t.Fatal(err)
				}
				a, err := readCurrentActions(&doc)
				if err != nil || len(a.PlayerSlots) != 1 || a.PlayerSlots[0].Trigger == nil || *a.PlayerSlots[0].Trigger != trigger {
					t.Fatal("distinct trigger lacks exact anchored policy", err)
				}
				leaf, _, _ := sav.NativeActions(doc.State)
				r := &doc.Objects[a.PlayerSlots[0].Object-1]
				mustSetValue(r, "Slot", 15)
				mustSetValue(r, "SlotAgain", 15)
				input, err = sav.EncodeDocumentData(doc)
				if err != nil {
					t.Fatal(err)
				}
				back, err := sav.DecodeDocumentData(input)
				if err != nil {
					t.Fatal(err)
				}
				unchanged, _, _ := sav.NativeActions(back.State)
				if !bytes.Equal(leaf, unchanged) {
					t.Fatal("ordinary Slot edit changed native trigger policy")
				}
				owner, expected[0].CommandID, expected[0].TriggerID = 15, 15, 15
			}
			for cycle := 0; cycle < 2; cycle++ {
				cold := cellStateFront(t)
				cold.Campaign, cold.Table = f.Campaign, f.Table
				open, town, err := cold.RestoreOriginal(input)
				if err == nil && !town {
					err = cold.App("distinct trigger LOAD").OpenMission(open)
				}
				if err != nil || town {
					t.Fatalf("slot%d cycle%d: %v", slot, cycle, err)
				}
				got, present := cold.live.world.SavedPlayerFormations()
				if !present || !reflect.DeepEqual(got, expected) {
					t.Fatalf("slot%d cycle%d lost exact native trigger/command/mode: %+v", slot, cycle, got)
				}
				for _, e := range cold.live.world.Entities() {
					if e.Owner != owner {
						t.Fatal("trigger transport changed native actor owner", e.ID, e.Owner)
					}
				}
				if cycle == 0 {
					next, label, err := cold.Snapshot(true)
					if err != nil {
						t.Fatal(err)
					}
					input, err = cold.ExportCurrentSave(next, label)
					if err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
}
