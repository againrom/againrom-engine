package sim

import (
	"reflect"
	"testing"
)

// OffMap is presence, not structural membership. The hidden member is the
// CURRENT head, deliberately ahead of a visible successor with a lower ID.
// Both must remain counted, but only the visible successor may decide.
func TestSavedGroupsStandingOffMapRetainsOrderAndCannotAcquireAcrossSave(t *testing.T) {
	for _, hostile := range []bool{false, true} {
		name := "peaceful-retained-move"
		owner := uint32(1)
		if hostile {
			name, owner = "nearby-hostile", 2
		}
		t.Run(name, func(t *testing.T) {
			w, err := NewRelatedWorld(1113, Bounds{40, 40}, ModeCanonical, Terrain{}, []Entity{
				{ID: 10, Owner: 1, X: 10, Y: 10, HP: 100, MaxHP: 100, Speed: 20, ScanRange: 10, Reach: 2, DamageBase: 5, AlwaysHits: true, TargetX: 17, TargetY: 17, HasTarget: true},
				{ID: 20, Owner: 1, X: 11, Y: 10, HP: 100, MaxHP: 100, Speed: 20, ScanRange: 10, Reach: 2, TargetX: 17, TargetY: 17, HasTarget: true},
				{ID: 30, Owner: owner, X: 12, Y: 10, HP: 100, MaxHP: 100, Speed: 20},
			}, nil, acEnemies(t))
			if err != nil {
				t.Fatal(err)
			}
			g := SavedGroup{ID: 91, Selector: 101, Members: []SavedGroupMember{{Archive: 2, Entity: 20, Bound: true}, {Archive: 1, Entity: 10, Bound: true}}}
			g.AI[0x20], g.AI[0x45] = 0xff, 1
			orders := []SavedActorOrder{{Entity: 10, State: 1}, {Entity: 20, State: 1}}
			for i := range orders {
				orders[i].Raw[8], orders[i].Raw[10], orders[i].Raw[11] = 1, 17, 17
			}
			if err := w.ImportSavedGroups([]SavedGroup{g}, orders); err != nil {
				t.Fatal(err)
			}
			w.script = mustScript(t, []ScriptCheck{{Op: ScriptCheckGroupCount, HasGroup: true, Group: 101, Register: 0}}, nil, nil)
			w.takeOffMap(indexOfEntity(w.entities, 20))
			w.tick = scriptPassPhase
			beforeEntity, beforeOrder := w.entities[1], *w.savedOrder(20)
			beforeGroups, _, _ := w.SavedGroups()
			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var fresh World
			if err := fresh.UnmarshalBinary(form); err != nil {
				t.Fatal(err)
			}
			for tick := range 35 {
				Step(w, nil)
				Step(&fresh, nil)
				if w.Hash() != fresh.Hash() {
					t.Fatalf("native continuation changed at tick%d", tick)
				}
				for _, current := range []*World{w, &fresh} {
					if current.entities[1] != beforeEntity || !reflect.DeepEqual(*current.savedOrder(20), beforeOrder) {
						t.Fatalf("hidden ff member changed retained state at tick%d", tick)
					}
					groups, _, _ := current.SavedGroups()
					if !reflect.DeepEqual(groups, beforeGroups) || current.ScriptRegister(0) != 2 {
						t.Fatalf("presence changed current membership/order/count at tick%d", tick)
					}
				}
				if tick == 0 {
					active := w.entities[0]
					if hostile && (!active.HasAttackTarget || active.AttackTarget != 30) {
						t.Fatal("visible successor did not acquire the hostile", active)
					}
					if !hostile && (active.HasTarget || w.savedOrder(10).Raw[8] != 0xb) {
						t.Fatal("visible successor did not execute peaceful acquire", active)
					}
				}
			}
		})
	}
}
