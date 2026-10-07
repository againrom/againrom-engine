package sim

import "testing"

func TestCurrentPlayerSlotOwnsExactGroupAndKeepsOrdinaryMoney(t *testing.T) {
	w := savedGroupWorld(t)
	groups, orders, _ := w.SavedGroups()
	groups[1].Members = groups[0].Members[2:]
	groups[0].Members = groups[0].Members[:2]
	groups[0].Owner = SavedGroupReference{Class: 1, Key: 101, Owner: 1}
	groups[1].Owner = SavedGroupReference{Class: 1, Key: 202, Owner: 2}
	if err := w.ImportSavedGroups(groups, orders); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedGroupPlayers([]SavedGroupPlayer{{1, 1}, {2, 2}}, []SavedGroupContainer{{71, 1}, {72, 2}}); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedPlayerFormations([]SavedPlayerFormation{{1, 1, 1, 0}, {2, 2, 2, 7}}, []SavedGroupOwner{{71, 1}, {72, 2}}); err != nil {
		t.Fatal(err)
	}
	w.SetPurse(1, 17)
	w.SetPurse(2, 29)
	row := CurrentPlayerSlot{Player: 2, Key: 202, Wire: 2, Native: 0}
	before := w.Hash()
	if err := w.RestoreCurrentPlayerSlots([]CurrentPlayerSlot{row, row}, nil); err == nil || before != w.Hash() {
		t.Fatal("duplicate Player mapping changed native state", err)
	}
	if err := w.RestoreCurrentPlayerSlots([]CurrentPlayerSlot{row}, map[uint32]uint32{0: 0xfedcba98, 1: 17, 50: 1}); err == nil || before != w.Hash() {
		t.Fatal("late invalid purse changed native state", err)
	}
	if err := w.RestoreCurrentPlayerSlots([]CurrentPlayerSlot{row}, map[uint32]uint32{0: 0xfedcba98, 1: 17}); err != nil {
		t.Fatal(err)
	}
	got, _, _ := w.SavedGroups()
	players, _ := w.SavedGroupPlayers()
	formations, _ := w.SavedPlayerFormations()
	if got[0].Owner.Owner != 1 || got[1].Owner.Owner != 0 || got[1].Owner.Key != 202 || got[1].ContainerID != 2 || players[1].Slot != 0 ||
		formations[1].CommandID != 0 || formations[1].TriggerID != 0 || formations[1].Mode != 7 {
		t.Fatal("slot mapping lost exact Group or formation ownership", got, players, formations)
	}
	if w.entities[0].Owner != 1 || w.entities[1].Owner != 1 || w.entities[2].Owner != 0 || w.Purse(0) != 0xfedcba98 || w.Purse(1) != 17 || w.Purse(2) != 0 {
		t.Fatal("mapping changed unrelated owner or ignored ordinary Money")
	}
}

func TestCurrentPlayerSlotsPermitSharedNativePurse(t *testing.T) {
	w := savedGroupWorld(t)
	groups, orders, _ := w.SavedGroups()
	groups[1].Members, groups[0].Members = groups[0].Members[2:], groups[0].Members[:2]
	groups[0].Owner = SavedGroupReference{Class: 1, Key: 101, Owner: 1}
	groups[1].Owner = SavedGroupReference{Class: 1, Key: 202, Owner: 2}
	if err := w.ImportSavedGroups(groups, orders); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedGroupPlayers([]SavedGroupPlayer{{1, 1}, {2, 2}}, []SavedGroupContainer{{71, 1}, {72, 2}}); err != nil {
		t.Fatal(err)
	}
	w.SetPurse(1, 17)
	w.SetPurse(2, 17)
	rows := []CurrentPlayerSlot{{Player: 1, Key: 101, Wire: 1, Native: 0, Shared: true}, {Player: 2, Key: 202, Wire: 2, Native: 0, Shared: true}}
	if err := w.RestoreCurrentPlayerSlots(rows, map[uint32]uint32{0: 0xf1234567}); err != nil {
		t.Fatal(err)
	}
	players, _ := w.SavedGroupPlayers()
	if len(players) != 2 || players[0] != (SavedGroupPlayer{1, 0}) || players[1] != (SavedGroupPlayer{2, 0}) || w.Purse(0) != 0xf1234567 || w.Purse(1) != 0 || w.Purse(2) != 0 {
		t.Fatal("shared native purse changed exact Player identities or Money", players)
	}
	got, _, _ := w.SavedGroups()
	for i, g := range got {
		if g.ID != groups[i].ID || g.ContainerID != uint32(i+1) || g.Owner.Key != groups[i].Owner.Key || g.Owner.Owner != 0 {
			t.Fatal("shared purse merged distinct Group owners", got)
		}
	}
	for _, e := range w.entities {
		if e.Owner != 0 {
			t.Fatal("shared native slot was not applied to exact Group actor", e.ID, e.Owner)
		}
	}
}
