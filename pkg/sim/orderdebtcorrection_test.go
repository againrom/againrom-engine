package sim

import "testing"

func TestAPlayersClickMoveWithAnEmptyRouteStandsDown(t *testing.T) {
	for _, state := range []uint8{0, actorStateGuard, actorStateAcquire} {
		self := asFighter(asHero, SelfSlot, 30, 40)
		self.Reach, self.Facing = 2, 64
		grid := make([]byte, 64*64)
		for dy := int32(-1); dy <= 1; dy++ {
			for dx := int32(-1); dx <= 1; dx++ {
				if dx != 0 || dy != 0 {
					grid[(40+dy)*64+30+dx] = blockGround
				}
			}
		}
		w, err := NewStockedSpelledWorld(1, Bounds{Width: 64, Height: 64}, ModeCanonical, Terrain{Block: grid},
			[]Entity{self, asFoe(asFoeA, 32, 40)}, secondGameScript(t, nil, nil, nil),
			engRel(t, [3]uint32{SelfSlot, asSide, 1}, [3]uint32{asSide, SelfSlot, 1}), nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		w.entities[0].ActorState = state
		Step(w, []Command{MoveTo(asHero, CellPoint{X: 45, Y: 40})})
		for k := range 160 {
			Step(w, nil)
			if e := w.entities[0]; e.HasAttackTarget {
				t.Fatalf("state=%d step %d: click move with an empty route acquired %d", state, k, e.AttackTarget)
			}
		}
	}
}

func TestAPlayersIdleHeroDoesNotLookAround(t *testing.T) {
	self := asFighter(asHero, SelfSlot, 30, 40)
	self.Facing, self.RotationSpeed = 64, 1
	w := asWorld(t, true, self, asFighter(asFoeA, asSide, 55, 40))
	w.commandGroup([]int{0}, orderNone, cell{})
	wakeGroups(w)
	for range 200 {
		Step(w, nil)
		if f := w.entities[0].Facing; f != 64 {
			t.Fatalf("a player's idle hero turned to %d by itself", f)
		}
	}
}

func TestAComputerUnitHoldingAnAttackTargetDoesNotLookAround(t *testing.T) {
	self := asFighter(asHero, asSide, 30, 40)
	self.Facing = 64
	w := asWorld(t, true, self, asFighter(asFoeA, SelfSlot, 36, 40))
	w.commandGroup([]int{0}, orderNone, cell{})
	w.savedOrder(asHero).Raw[8] = 0xb
	w.orderAttack(0, asFoeA)
	for range 120 {
		Step(w, nil)
	}
	if e := w.entities[0]; e.X == 30 && e.Y == 40 {
		t.Fatalf("the unit holding an attack target never moved (target %v)", e.HasAttackTarget)
	}
}
