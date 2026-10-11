package sim

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func handoverPostWorld(t *testing.T, saved bool, state uint8) *World {
	t.Helper()
	w := mustWorld(t, 1, Bounds{Width: 32, Height: 32}, []Entity{
		{ID: 10, Owner: 2, Group: 7, X: 20, Y: 20, HP: 100, MaxHP: 100, Speed: 20, Reach: 1, ScanRange: 5},
		{ID: 20, Owner: 2, Group: 8, X: 28, Y: 20, HP: 100, MaxHP: 100},
	})
	if saved {
		g := SavedGroup{ID: 71, Selector: 7, Owner: SavedGroupReference{Class: 1, Owner: 2}, Authored: true,
			Members: []SavedGroupMember{{Entity: 10, Bound: true}}}
		g.AI[0x20], g.AI[0x45] = 0, 1
		o := SavedActorOrder{Entity: 10, State: uint32(state), Authored: true}
		o.Raw[0], o.Raw[1], o.Raw[8], o.Raw[10], o.Raw[11] = 4, 4, 1, 28, 20
		if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{o}); err != nil {
			t.Fatal(err)
		}
	}
	e := &w.entities[0]
	e.ActorState, e.PostX, e.PostY = state, 4, 4
	e.TargetX, e.TargetY, e.HasTarget = 28, 20, true
	if state == actorStatePatrol {
		e.PatrolHeadX, e.PatrolHeadY, e.PatrolTailX, e.PatrolTailY, e.PatrolLeg = 4, 4, 28, 20, patrolLegTail
	}
	if state == actorStateFollow {
		e.EscortTarget, e.HasEscortTarget, e.EscortRange = 20, true, 3
	}
	return w
}

func TestHandOverStandsAtTheCurrentPost(t *testing.T) {
	for _, saved := range []bool{false, true} {
		for _, state := range []uint8{actorStateGuard, actorStatePatrol, actorStateFollow} {
			t.Run(fmt.Sprintf("saved=%t/state=%x", saved, state), func(t *testing.T) {
				w := handoverPostWorld(t, saved, state)
				w.runInstant(ScriptInstant{Op: ScriptInstantGiveUnit, Unit: 10, HasUnit: true, Player: 3, HasPlayer: true})
				e := w.entities[0]
				if e.Owner != 3 || e.ActorState != actorStateAcquire || e.PostX != 20 || e.PostY != 20 || e.HasTarget || e.HasEscortTarget || e.PatrolLeg != 0 {
					t.Fatalf("handover did not stand at the current post: %+v", e)
				}
				g, o := w.savedGroupFor(10), w.savedOrder(10)
				if g == nil || o == nil || g.Owner.Owner != 3 || g.AI[0x20] != orderStandGround || o.State != 0xc || o.Raw[8] != 0 || binary.LittleEndian.Uint16(o.Raw[:]) != 0x1414 || len(o.Patrol) != 0 {
					t.Fatalf("registry order did not stand: group=%+v order=%+v", g, o)
				}
				form, err := w.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				var cold World
				if err := cold.UnmarshalBinary(form); err != nil {
					t.Fatal(err)
				}
				for tick := range 512 {
					Step(w, nil)
					Step(&cold, nil)
					if w.Hash() != cold.Hash() {
						t.Fatalf("cold continuation differs at tick %d", tick+1)
					}
					if got := w.entities[0]; got.X != 20 || got.Y != 20 || got.HasTarget {
						t.Fatalf("stood actor moved at tick %d: %+v", tick+1, got)
					}
				}
			})
		}
	}
	for _, saved := range []bool{false, true} {
		t.Run(fmt.Sprintf("dead/saved=%t", saved), func(t *testing.T) {
			w := handoverPostWorld(t, saved, actorStateGuard)
			if err := w.HeadlessKill(10); err != nil {
				t.Fatal(err)
			}
			e := &w.entities[0]
			state := e.ActorState
			w.handOver(0, 3)
			if e.Owner != 3 || e.ActorState != state || e.PostX != 4 || e.PostY != 4 {
				t.Fatalf("dead handover changed the actor's order: %+v", e)
			}
			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(form); err != nil {
				t.Fatal(err)
			}
		})
		t.Run(fmt.Sprintf("loss-control/saved=%t", saved), func(t *testing.T) {
			w := handoverPostWorld(t, saved, actorStateGuard)
			w.handOver(0, 3)
			e := &w.entities[0]
			e.ActorState, e.PostX, e.PostY = actorStateGuard, 4, 4
			w.savedGroupFor(10).AI[0x20] = orderNone
			o := w.savedOrder(10)
			o.State, o.Raw[0], o.Raw[1] = 0xb, 4, 4
			for range 512 {
				Step(w, nil)
			}
			if got := w.entities[0]; got.X == 20 && got.Y == 20 {
				t.Fatal("loss of Stand Ground and current post did not move the actor")
			}
		})
	}
}
