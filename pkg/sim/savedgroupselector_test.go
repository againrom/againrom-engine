package sim

import (
	"reflect"
	"strings"
	"testing"
)

// selectorWorld holds the saved Groups of two Players as a LOAD restores
// them: Groups in file order, each in its Player's list. owners[k] is the slot
// of Player k+1; groups[k] names the Player and selector of Group k+1, whose
// members are the entities whose Group field equals k+1.
func selectorWorld(t *testing.T, owners []uint32, groups [][2]uint32, ents []Entity) *World {
	t.Helper()
	w := mustWorld(t, 113, Bounds{Width: 32, Height: 32}, ents)
	var saved []SavedGroup
	var containers []SavedGroupContainer
	for k, g := range groups {
		id := uint32(k + 1)
		sg := SavedGroup{ID: id, Selector: g[1], Owner: SavedGroupReference{Class: 1, Owner: owners[g[0]-1]}}
		sg.AI[0x45] = 1
		for _, e := range ents {
			if e.Group == id {
				sg.Members = append(sg.Members, SavedGroupMember{Archive: uint16(e.ID), Entity: e.ID, Bound: true})
			}
		}
		saved = append(saved, sg)
		containers = append(containers, SavedGroupContainer{GroupID: id, PlayerID: g[0]})
	}
	if err := w.ImportOriginalGroups(saved, nil); err != nil {
		t.Fatal(err)
	}
	var players []SavedGroupPlayer
	for k, slot := range owners {
		players = append(players, SavedGroupPlayer{ID: uint32(k + 1), Slot: slot})
	}
	if err := w.ImportSavedGroupPlayers(players, containers); err != nil {
		t.Fatal(err)
	}
	return w
}

// The count a script reads and the Group a command reaches, for selector 5.
func selectorOutcome(t *testing.T, w *World) (int, []SavedGroup) {
	t.Helper()
	before, _, _ := w.SavedGroups()
	w.cmdSavedGroupOrder(ScriptInstant{HasGroup: true, Group: 5, Args: [10]int32{4, 3, 4}})
	after, _, _ := w.SavedGroups()
	var reached []SavedGroup
	for k := range after {
		if !reflect.DeepEqual(before[k], after[k]) {
			reached = append(reached, after[k])
		}
	}
	return len(w.groupMembers(5)), reached
}

// One Player's two Groups carry selector 5: the later Group in file order is
// the one a script reaches (SAV-1338, AI-366). The earlier Group holds two
// members and the later one, so the count tells first from last.
func TestSavedSelectorOneOwnerLaterGroupWins(t *testing.T) {
	ents := []Entity{
		{ID: 10, Owner: 2, Group: 1, X: 4, Y: 4, HP: 10, MaxHP: 10},
		{ID: 20, Owner: 2, Group: 1, X: 5, Y: 4, HP: 10, MaxHP: 10},
		{ID: 30, Owner: 2, Group: 2, X: 6, Y: 4, HP: 10, MaxHP: 10},
	}
	w := selectorWorld(t, []uint32{1, 2}, [][2]uint32{{2, 5}, {2, 5}}, ents)
	for _, issue := range w.SavedGroupIssues() {
		if strings.Contains(issue, "selector") {
			t.Fatal("a selector one owner's two Groups carry is reported:", issue)
		}
	}
	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	for _, current := range []*World{w, &cold} {
		count, reached := selectorOutcome(t, current)
		if count != 1 || len(reached) != 1 || reached[0].ID != 2 || reached[0].AI[10] != 3 || reached[0].AI[11] != 4 {
			t.Fatalf("selector 5 reached %d members and Groups %+v; want the later Group 2 alone", count, reached)
		}
	}
	// Loss control: with the selector on the earlier Group alone the same
	// command reaches Group 1, so the answer above is the order rule's.
	w = selectorWorld(t, []uint32{1, 2}, [][2]uint32{{2, 5}, {2, 6}}, ents)
	if count, reached := selectorOutcome(t, w); count != 2 || len(reached) != 1 || reached[0].ID != 1 {
		t.Fatalf("control: selector 5 reached %d members and %+v", count, reached)
	}
}

// Two Players carry selector 5: the later Player's Group wins, whatever the
// slot numbers, as before (AI-366). Player 1 here is slot 3 and Player 2 slot
// 2, so a rule keyed on the slot would pick the other Group.
func TestSavedSelectorLaterPlayerWins(t *testing.T) {
	ents := []Entity{
		{ID: 10, Owner: 3, Group: 1, X: 4, Y: 4, HP: 10, MaxHP: 10},
		{ID: 20, Owner: 3, Group: 1, X: 5, Y: 4, HP: 10, MaxHP: 10},
		{ID: 30, Owner: 2, Group: 2, X: 6, Y: 4, HP: 10, MaxHP: 10},
	}
	w := selectorWorld(t, []uint32{3, 2}, [][2]uint32{{1, 5}, {2, 5}}, ents)
	if count, reached := selectorOutcome(t, w); count != 1 || len(reached) != 1 || reached[0].ID != 2 {
		t.Fatalf("selector 5 reached %d members and %+v; want the later Player's Group 2", count, reached)
	}
	w = selectorWorld(t, []uint32{3, 2}, [][2]uint32{{1, 5}, {2, 6}}, ents)
	if count, reached := selectorOutcome(t, w); count != 2 || len(reached) != 1 || reached[0].ID != 1 {
		t.Fatalf("control: selector 5 reached %d members and %+v", count, reached)
	}
}

// SAVE drops the empty Groups of a Player whose participant word +28 is zero
// and keeps those of every other Player (SAV-1251, SAV-1234). Each case has an
// empty Group in both Players; which survives follows the word, not the slot.
func TestPruneGroupsForSaveFollowsTheParticipantWord(t *testing.T) {
	ents := []Entity{
		{ID: 10, Owner: 1, Group: 1, X: 4, Y: 4, HP: 10, MaxHP: 10},
		{ID: 30, Owner: 2, Group: 3, X: 6, Y: 4, HP: 10, MaxHP: 10},
	}
	layout := [][2]uint32{{1, 1}, {1, 7}, {2, 2}, {2, 8}}
	for _, tc := range []struct {
		name    string
		words   []uint32
		present bool
		want    []uint32
	}{
		{"human first", []uint32{0, 1}, true, []uint32{1, 3, 4}},
		{"human second", []uint32{1, 0}, true, []uint32{1, 2, 3}},
		{"no human", []uint32{1, 1}, true, []uint32{1, 2, 3, 4}},
		{"words absent: the human slot", nil, false, []uint32{1, 3, 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := selectorWorld(t, []uint32{1, 2}, layout, ents)
			players, _ := w.SavedGroupPlayers()
			var rows []PlayerParticipant
			for k, word := range tc.words {
				rows = append(rows, PlayerParticipant{PlayerID: players[k].ID, Value: word})
			}
			if err := w.RestoreCurrentPlayers(players, rows, tc.present); err != nil {
				t.Fatal(err)
			}
			pruned := w.PruneGroupsForSave()
			groups, _, _ := w.SavedGroups()
			var ids []uint32
			for _, g := range groups {
				ids = append(ids, g.ID)
			}
			if !reflect.DeepEqual(ids, tc.want) || pruned != (len(tc.want) != len(layout)) {
				t.Fatalf("kept Groups %v (pruned %t), want %v", ids, pruned, tc.want)
			}
			before := w.Hash()
			if w.PruneGroupsForSave() || w.Hash() != before {
				t.Fatal("a second SAVE pruned again")
			}
			data, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(data); err != nil || cold.Hash() != w.Hash() {
				t.Fatal("the pruned lists do not survive the byte form", err)
			}
		})
	}
}

// A World with no saved registry, or with the one its map start built, has
// nothing to prune. The control prunes the same lists once a LOAD set them.
func TestPruneGroupsForSaveNativeDialect(t *testing.T) {
	w := mustWorld(t, 113, Bounds{Width: 8, Height: 8}, []Entity{{ID: 1, Owner: 1, X: 1, Y: 1, HP: 1, MaxHP: 1}})
	w.savedGroups = nil
	before := w.Hash()
	if w.PruneGroupsForSave() || w.Hash() != before {
		t.Fatal("a native-dialect World changed at SAVE")
	}
	for _, fresh := range []bool{true, false} {
		w = selectorWorld(t, []uint32{1, 2}, [][2]uint32{{1, 1}, {1, 7}}, []Entity{{ID: 10, Owner: 1, Group: 1, X: 4, Y: 4, HP: 10, MaxHP: 10}})
		w.savedGroups.fresh = fresh
		if pruned := w.PruneGroupsForSave(); pruned == fresh {
			t.Fatalf("map-start registry %t: pruned %t", fresh, pruned)
		}
	}
}

// An actor whose source Group is dropped at SAVE takes the Group that holds
// it as its source, as a LOAD of the written file reads it. The control keeps
// the empty Group and the actor's source Group with it.
func TestPruneGroupsForSaveRebindsTheSourceGroup(t *testing.T) {
	source := SourceBinding{Class: 1, ArchiveIndex: 4, Identity: 9, GroupIndex: 2, GroupSelector: 7, GroupOwnerKey: 99, GroupOwnerSlot: 1, GroupOwnerResolved: true}
	ents := []Entity{{ID: 10, Owner: 1, Group: 1, X: 4, Y: 4, HP: 10, MaxHP: 10, SourceBinding: source}}
	for _, tc := range []struct {
		name  string
		words []uint32
		want  SourceBinding
	}{
		{"source dropped", []uint32{0, 1}, SourceBinding{Class: 1, ArchiveIndex: 4, Identity: 9, GroupIndex: 1, GroupSelector: 1, GroupOwnerSlot: 1, GroupOwnerResolved: true}},
		{"source kept", []uint32{1, 0}, source},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := selectorWorld(t, []uint32{1, 2}, [][2]uint32{{1, 1}, {1, 7}, {2, 8}}, ents)
			players, _ := w.SavedGroupPlayers()
			rows := []PlayerParticipant{{PlayerID: players[0].ID, Value: tc.words[0]}, {PlayerID: players[1].ID, Value: tc.words[1]}}
			if err := w.RestoreCurrentPlayers(players, rows, true); err != nil {
				t.Fatal(err)
			}
			w.PruneGroupsForSave()
			if got := w.Entities()[0].SourceBinding; got != tc.want {
				t.Fatalf("source binding %+v, want %+v", got, tc.want)
			}
		})
	}
}
