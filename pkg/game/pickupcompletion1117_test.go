package game

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

func pickupForm1117(t *testing.T, w *sim.World) []byte {
	t.Helper()
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestPickup1117UnderfootSupersedesOlderQueuedOrdersOnlyAfterSuccess(t *testing.T) {
	for _, succeeds := range []bool{false, true} {
		var sacks []sim.Sack
		sacks = append(sacks, sim.Sack{X: poSackX, Y: poSackY, Items: []uint16{eqSwordCode}})
		if succeeds {
			sacks = append(sacks, sim.Sack{X: poHeroX, Y: poHeroY, Gold: 17})
		}
		w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
			[]sim.Entity{{ID: poHeroID, X: poHeroX, Y: poHeroY}, {ID: 99, X: 20, Y: 20}}, nil, sim.Relations{}, sacks)
		if err != nil {
			t.Fatal(err)
		}
		mw := grabWorld(t, w, poHeroID, missionSource{})
		mw.orderPickup(poHeroID, poSackX, poSackY)
		kept := []sim.Command{{Kind: sim.KindEquip, Entity: poHeroID, X: 999, Y: 1}, {Kind: sim.KindMoveTo, Entity: 99, X: 21, Y: 20}}
		mw.pending = append(mw.pending, kept...)
		queued := append([]sim.Command(nil), mw.pending...)
		before := pickupForm1117(t, w)
		mw.grab(0, 0, 0, false)
		if !succeeds {
			if !bytes.Equal(before, pickupForm1117(t, w)) || !reflect.DeepEqual(queued, mw.pending) || !mw.pickup.set {
				t.Fatal("failed underfoot key changed earlier order")
			}
			continue
		}
		if mw.pickup.set || !reflect.DeepEqual(kept, mw.pending) {
			t.Fatalf("successful key retained earlier approach or discarded peer/inventory: %+v %+v", mw.pickup, mw.pending)
		}
		// A later order is not one of the superseded predecessors.
		mw.enqueue(poHeroID, poSackX, poSackY)
		for n := 0; n < 120; n++ {
			mw.tick()
		}
		if _, exists := mw.sackAt(poSackX, poSackY); !exists {
			t.Fatal("cancelled earlier pickup took second sack after later ordinary Move")
		}
		if e, _ := mw.entity(poHeroID); e.ActorState == 2 || e.ActorState == 0xc {
			t.Fatal("later Move lost to pickup completion")
		}
	}
}

func TestPickup1117DeadOffMapAndMissingActorsCannotTake(t *testing.T) {
	for _, which := range []string{"dead", "off-map", "missing"} {
		t.Run(which, func(t *testing.T) {
			e := sim.Entity{ID: 0, X: 3, Y: 3, HP: 10, MaxHP: 10}
			if which == "dead" {
				e.HP = 0
			}
			if which == "off-map" {
				e.OffMap = true
			}
			w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{e}, nil, sim.Relations{}, []sim.Sack{{X: 3, Y: 3, Gold: 17}})
			if err != nil {
				t.Fatal(err)
			}
			mw := grabWorld(t, w, 0, missionSource{})
			id := sim.EntityID(0)
			if which == "missing" {
				id = 123
			}
			before := pickupForm1117(t, w)
			mw.takeSackFor(id)
			mw.orderPickup(id, 3, 3)
			if !bytes.Equal(before, pickupForm1117(t, w)) || mw.pickup.set || len(mw.pending) != 0 {
				t.Fatal("invalid actor took or ordered a sack")
			}
		})
	}
}

func TestPickup1117ClickAndUnderfootCompleteOnNextStep(t *testing.T) {
	for _, click := range []bool{false, true} {
		name := "underfoot"
		if click {
			name = "click-walk"
		}
		t.Run(name, func(t *testing.T) {
			w, mw := poWorld(t)
			if click {
				mw.grab(poHeroID, poSackX, poSackY, true)
			} else {
				var err error
				w, err = sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
					[]sim.Entity{{ID: poHeroID, X: poSackX, Y: poSackY, HasTarget: true, TargetX: poEmptyX, TargetY: poEmptyY}},
					nil, sim.Relations{}, []sim.Sack{{X: poSackX, Y: poSackY, Items: []uint16{eqSwordCode}}})
				if err != nil {
					t.Fatal(err)
				}
				mw = grabWorld(t, w, poHeroID, missionSource{})
			}
			if click && poRun(mw, w) < 0 {
				t.Fatal("hero never reached sack")
			}
			if !click {
				mw.grab(0, 0, 0, false)
			}
			if len(w.Sacks()) != 0 {
				t.Fatal("successful pickup left sack")
			}
			e, _ := mw.entity(poHeroID)
			if e.ActorState != 2 || e.HasTarget || e.HasAttackTarget {
				t.Fatalf("transfer did not leave pending completion without stale order: %+v", e)
			}
			atTransfer := pickupForm1117(t, w)
			var restored sim.World
			if err := restored.UnmarshalBinary(atTransfer); err != nil {
				t.Fatal("transfer LOAD:", err)
			}
			for n := 0; n < 34; n++ {
				sim.Step(w, nil)
				sim.Step(&restored, nil)
				if w.Hash() != restored.Hash() || !bytes.Equal(pickupForm1117(t, w), pickupForm1117(t, &restored)) {
					t.Fatalf("native successor %d differs", n+1)
				}
				if n == 0 {
					e, _ = mw.entity(poHeroID)
					if e.ActorState != 0x0c || e.HasTarget || e.HasAttackTarget {
						t.Fatalf("next step did not complete into acquire: %+v", e)
					}
					var completion sim.World
					if err := completion.UnmarshalBinary(pickupForm1117(t, w)); err != nil {
						t.Fatal("completion LOAD:", err)
					}
					restored = completion
				}
			}
		})
	}
}
