package sim

import (
	"reflect"
	"testing"
)

func TestGroupRootAbsenceKeepsOrdinaryActors(t *testing.T) {
	w := savedGroupWorld(t)
	groups, _, _ := w.SavedGroups()
	actors := w.Entities()
	if err := w.RestoreGroupContinuations([]GroupContinuation{{Group: 71, ID: 71}, {Group: 72, RootOnly: true}}, 90); err != nil {
		t.Fatal(err)
	}
	got, _, _ := w.SavedGroups()
	if len(got) != 1 || !reflect.DeepEqual(got[0], groups[0]) || !reflect.DeepEqual(actors, w.Entities()) || w.GroupHighWater() != 90 {
		t.Fatal("absent topology removed ordinary actors or changed existing Group")
	}
}

func TestMixedPlayerPresenceRemovesOnlyUnreferencedContainers(t *testing.T) {
	fixture := func(t *testing.T) *World {
		w := savedGroupWorld(t)
		if err := w.ImportSavedGroupPlayers([]SavedGroupPlayer{{ID: 1, Slot: 1}, {ID: 2, Slot: 2}}, []SavedGroupContainer{{GroupID: 71, PlayerID: 1}, {GroupID: 72, PlayerID: 2}}); err != nil {
			t.Fatal(err)
		}
		if err := w.ImportSavedPlayerFormations([]SavedPlayerFormation{{PlayerID: 1, CommandID: 1, TriggerID: 1, Mode: 3}, {PlayerID: 2, CommandID: 2, TriggerID: 2, Mode: 9}}, []SavedGroupOwner{{GroupID: 71}, {GroupID: 72}}); err != nil {
			t.Fatal(err)
		}
		return w
	}
	rows := []GroupContinuation{{Group: 71, ID: 71}, {Group: 72, RootOnly: true}}
	for _, fault := range [][]uint32{{0}, {99}, {2, 2}, {1}, {1, 2}} {
		w := fixture(t)
		before := w.Hash()
		if err := w.RestoreGroupContinuations(rows, 72, fault...); err == nil || w.Hash() != before {
			t.Fatal("malformed or still-referenced Player absence was not atomic", fault, err)
		}
	}
	t.Run("formation owner still referenced", func(t *testing.T) {
		w := fixture(t)
		w.savedGroups.Groups[0].Owner = SavedGroupReference{Class: 1, Owner: 2}
		w.savedGroups.Groups[0].OwnerID = 2
		before := w.Hash()
		if err := w.RestoreGroupContinuations(rows, 72, 2); err == nil || w.Hash() != before {
			t.Fatal("absence removed a real Group's exact formation owner", err)
		}
	})
	w := fixture(t)
	groups, orders, _ := w.SavedGroups()
	actors := w.Entities()
	if err := w.RestoreGroupContinuations(rows, 72, 2); err != nil {
		t.Fatal(err)
	}
	players, present := w.SavedGroupPlayers()
	formations, hasFormations := w.SavedPlayerFormations()
	gotGroups, gotOrders, _ := w.SavedGroups()
	if !present || !hasFormations || !reflect.DeepEqual(players, []SavedGroupPlayer{{ID: 1, Slot: 1}}) || !reflect.DeepEqual(formations, []SavedPlayerFormation{{PlayerID: 1, CommandID: 1, TriggerID: 1, Mode: 3}}) || !reflect.DeepEqual(gotGroups, groups[:1]) || !reflect.DeepEqual(gotOrders, orders) || !reflect.DeepEqual(w.Entities(), actors) {
		t.Fatal("mixed absence changed current native state beyond its exact roots")
	}
}

func TestGroupCarrierAbsencePrecedesNativeDispatchOrder(t *testing.T) {
	w := savedGroupWorld(t)
	if err := w.ImportSavedGroupPlayers([]SavedGroupPlayer{{ID: 1, Slot: 1}, {ID: 2, Slot: 2}},
		[]SavedGroupContainer{{GroupID: 71, PlayerID: 1}, {GroupID: 72, PlayerID: 2}}); err != nil {
		t.Fatal(err)
	}
	rows := []GroupContinuation{{Group: 72, ID: 91}, {Group: 71, ID: 90}}
	before := w.Hash()
	if err := w.RestoreGroupContinuations(rows, 91); err == nil || w.Hash() != before {
		t.Fatal("real Player traversal accepted an invalid native order", err)
	}
	if err := w.RestoreGroupCarrierPresence(false, false); err != nil {
		t.Fatal(err)
	}
	if err := w.RestoreGroupContinuations(rows, 91); err != nil {
		t.Fatal("absent Player topology restricted native dispatch order", err)
	}
	groups, _, _ := w.SavedGroups()
	if groups[0].ID != 91 || groups[1].ID != 90 || groups[0].ContainerID != 0 || groups[1].ContainerID != 0 {
		t.Fatal("native order or absent container policy changed", groups)
	}
}

func TestActionOrderAbsenceAndOrdinaryOrderValues(t *testing.T) {
	for _, absent := range []bool{false, true} {
		w := savedGroupWorld(t)
		a := w.Actions()
		for i := range a.Actors {
			if absent {
				a.Actors[i].Order = nil
			}
		}
		for i := range w.savedGroups.Orders {
			w.savedGroups.Orders[i].Raw[0x28] = 93
		}
		if err := w.RestoreActions(a, nil); err != nil {
			t.Fatal(err)
		}
		_, orders, _ := w.SavedGroups()
		if absent {
			if len(orders) != 0 {
				t.Fatal("ordinary LOAD invented native orders", orders)
			}
		} else {
			if len(orders) == 0 {
				t.Fatal("present native orders disappeared")
			}
			for _, o := range orders {
				if o.Raw[0x28] != 93 {
					t.Fatal("unchanged continuation replaced ordinary order value")
				}
			}
		}
		before := w.Hash()
		a.Actors = a.Actors[:len(a.Actors)-1]
		if err := w.RestoreActions(a, nil); err == nil || w.Hash() != before {
			t.Fatal("missing action row was treated as order absence", err)
		}
	}
}

func TestCurrentAbsentPlanePolicyKeepsOrdinaryObstacles(t *testing.T) {
	w := savedGroupWorld(t)
	w.grid[0] = blockStaticObject | blockGround | blockAir
	p := w.CurrentPolicy()
	for _, ordinary := range []byte{blockStaticObject | blockGround | blockAir, 0} {
		w.grid[0] = ordinary
		if err := w.restoreCurrentPolicy(p); err != nil {
			t.Fatal(err)
		}
		if w.grid[0] != ordinary || w.savedCellPlanes != nil {
			t.Fatal("absent raw carrier erased an ordinary obstacle or its edit")
		}
	}
}
