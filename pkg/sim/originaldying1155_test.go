package sim

import (
	"reflect"
	"testing"
)

func TestOriginalDying1155LoadRetainsGroupAndSavedOrder(t *testing.T) {
	for _, admission := range []bool{false, true} {
		t.Run(map[bool]string{false: "source-tuple", true: "actor-admission"}[admission], func(t *testing.T) {
			w := sourceBindingWorld1111(t)
			e := &w.entities[0]
			e.TokenSize = 1
			e.Owner, e.HP, e.Decay, e.Dwell, e.Defence = 3, -8, DecayFallen, 7, 20
			group := SavedGroup{ID: 71, Selector: 41,
				Owner: SavedGroupReference{Key: 0x10000002, Archive: 2, Class: 1, Owner: 9},
				Words: []uint16{7, 8, 7}, Path: []uint16{0x0203, 0x0405},
				Members: []SavedGroupMember{{Archive: 4, Entity: e.ID, Bound: true}}}
			for i := range group.AI {
				group.AI[i] = byte(i + 1)
			}
			group.AI[0x20] = 0xff
			order := SavedActorOrder{Entity: e.ID, State: 0xb, RepairStage: 1, Patrol: []uint16{0x0203, 0x0405, 0x0203}}
			order.Raw[0], order.Raw[1], order.Raw[0x21] = 5, 6, 77
			if err := w.ImportSavedGroups([]SavedGroup{group}, []SavedActorOrder{order}); err != nil {
				t.Fatal(err)
			}
			before, rng := *e, w.rng.state
			var err error
			if admission {
				err = w.ImportOriginalLivingActors([]OriginalLivingActor{{Entity: *e}})
			} else {
				err = w.ImportOriginalDyingActors([]OriginalDyingActor{{ID: e.ID, HP: -8, Timer: 7}})
			}
			if err != nil {
				t.Fatal(err)
			}
			groups, orders, present := w.SavedGroups()
			if !present || !reflect.DeepEqual(groups, []SavedGroup{group}) || !reflect.DeepEqual(orders, []SavedActorOrder{order}) {
				t.Fatal("source LOAD changed Group identity/owner/membership or stored order", groups, orders)
			}
			e = &w.entities[0]
			if e.SourceBinding != before.SourceBinding || e.Owner != before.Owner || e.HP != before.HP || e.Decay != before.Decay || e.Dwell != before.Dwell || e.Defence != before.Defence || w.rng.state != rng || len(w.sacks) != 0 {
				t.Fatal("source LOAD replayed a death transition", *e)
			}
			w.savedActorDispatch(0)
			if e.HasTarget || e.X != before.X || e.Y != before.Y || !reflect.DeepEqual(w.savedOrder(e.ID), &order) {
				t.Fatal("retained dying order executed or was cleared")
			}
			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var back World
			if err := back.UnmarshalBinary(form); err != nil || back.Hash() != w.Hash() {
				t.Fatal("native wire lost restored dying Group/order", err)
			}
			for range 20 {
				Step(w, nil)
				Step(&back, nil)
				if back.Hash() != w.Hash() {
					t.Fatal("dying Group continuation differs")
				}
			}
		})
	}
}

func TestNewDamageKeepsSavedGroupUntilTeardown(t *testing.T) {
	w := savedGroupWorld(t)
	g := &w.savedGroups.Groups[0]
	g.AI[0x20] = 0xff
	Step(w, []Command{Damage(10, 108)})
	if w.entities[0].Alive() || w.entities[0].Decay != DecayFallen || w.savedGroupFor(10) == nil || len(g.Members) != 3 {
		t.Fatal("stage-one member left its saved Group", g)
	}
	w.script = mustScript(t, []ScriptCheck{{Op: ScriptCheckGroupCount, HasGroup: true, Group: 19, Register: 0}}, nil, nil)
	w.scriptPass(nil)
	if got := w.ScriptRegister(0); got != 3 {
		t.Fatalf("stage-one saved Group count=%d, want 3", got)
	}
	Step(w, []Command{Damage(10, 2)})
	if w.entities[0].Decay < DecayBones || w.savedGroupFor(10) != nil || len(g.Members) != 2 || g.AI[0x44] != 7 {
		t.Fatal("teardown did not detach only its member", g)
	}
	w.scriptPass(nil)
	if got := w.ScriptRegister(0); got != 2 {
		t.Fatalf("torn-down saved Group count=%d, want 2", got)
	}
}
