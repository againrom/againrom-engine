package sim

import "testing"

func TestGroundStepUphillRoundingByGame(t *testing.T) {
	for _, tc := range []struct {
		name          string
		dialect       ScriptDialect
		rate, transit int32
	}{
		{"first game", ScriptROM1, 7, 37},
		{"second game", ScriptROM2, 8, 32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cost, height := srPlane(1), srPlane(0)
			height[srAt(2, 1)] = 1
			w := srWorld(t, cost, height, []Entity{{ID: 1, Owner: SelfSlot, X: 1, Y: 1, Speed: 1}})
			if tc.dialect == ScriptROM2 {
				w.rom2, w.game = &rom2ScriptState{}, newGameProfile(tc.dialect)
			}
			rate, transit, adjacent, ok := w.StepRate(1, 2, 1)
			if !ok || !adjacent || rate != tc.rate || transit != tc.transit {
				t.Errorf("step rate and transit = %d/%d, adjacent %t, ok %t, want %d/%d", rate, transit, adjacent, ok, tc.rate, tc.transit)
			}
			Step(w, []Command{MoveTo(1, CellPoint{X: 2, Y: 1})})
			if e := w.entities[0]; e.X != 2 || e.Y != 1 || int32(e.TransitTotal) != tc.transit {
				t.Errorf("mover = (%d,%d) with transit %d, want (2,1) with transit %d", e.X, e.Y, e.TransitTotal, tc.transit)
			}
		})
	}
}

func TestBlockedFleeAnswerByGame(t *testing.T) {
	intruder := withdrawalFighter(2, SelfSlot, 22, 20, 100)
	first := pocketedArcher(t, ScriptROM1, intruder)
	first.entities[0].Wimpy = first.entities[0].HP
	wakeGroups(first)
	regWithdrawalPass(first, nil)
	if e := first.entities[0]; e.HasTarget || !e.HasAttackTarget || e.AttackTarget != intruder.ID {
		t.Errorf("first game: move %t, attack %t on %d, want reacquisition of %d", e.HasTarget, e.HasAttackTarget, e.AttackTarget, intruder.ID)
	}

	second := pocketedArcher(t, ScriptROM2, intruder)
	second.entities[0].Wimpy = second.entities[0].HP
	wakeGroups(second)
	pass := 0
	for tick := 0; tick < 2*scriptCycle; tick++ {
		decisions := StepWithdrawalTraced(second, nil)
		if len(decisions) == 0 {
			continue
		}
		pass++
		if len(decisions) != 1 || decisions[0].After.ID != 1 {
			t.Fatalf("second game, pass %d: want one withdrawal decision for actor 1", pass)
		}
		e, order := decisions[0].After, second.savedOrder(1)
		if e.HasTarget || e.HasAttackTarget || e.Transit != 0 || e.ActorState != 0xc || e.PostX != 20 || e.PostY != 20 {
			t.Errorf("second game, pass %d: state %#x, post (%d,%d), move %t, attack %t, transit %d; want order 0xc at (20,20) standing", pass, e.ActorState, e.PostX, e.PostY, e.HasTarget, e.HasAttackTarget, e.Transit)
		}
		if order == nil {
			t.Fatalf("second game, pass %d: missing saved order", pass)
		}
		if order.State != 0xc || order.Raw[0] != 20 || order.Raw[1] != 20 || order.Raw[8] != 0 || order.Raw[0x50] != 1 {
			t.Errorf("second game, pass %d: saved order %#x at (%d,%d), prepared byte %d and command latch %d, want 0xc at (20,20), 0 and 1", pass, order.State, order.Raw[0], order.Raw[1], order.Raw[8], order.Raw[0x50])
		}
	}
	if pass != 2 {
		t.Fatalf("second game: saw %d withdrawal passes, want two", pass)
	}
	var restored World
	if err := restored.UnmarshalBinary(laForm(t, second)); err != nil {
		t.Fatal(err)
	}
	if restored.Hash() != second.Hash() {
		t.Error("the stopped order changed on reload")
	}
}

func TestRetreatModeThresholdsByGame(t *testing.T) {
	melee := Entity{ID: 1, Owner: SelfSlot, X: 8, Y: 8, HP: 79, MaxHP: 79, Humanoid: true, Reach: 1, Withdraw: 63, Wimpy: 19}
	ranged, mage, flyer, foreign := melee, melee, melee, melee
	ranged.ID, ranged.X, ranged.Reach = 2, 9, 4
	mage.ID, mage.X, mage.Book.State = 3, 10, BookPresent
	flyer.ID, flyer.X, flyer.Reach, flyer.Domain = 4, 11, 4, DomainAir
	foreign.ID, foreign.X, foreign.Owner = 5, 12, SelfSlot+1
	foreign.Humanoid = false
	actors := []Entity{melee, ranged, mage, flyer, foreign}

	for _, tc := range []struct {
		name   string
		second bool
		mode   int32
		want   [4][2]int32
	}{
		{"first game high", false, RetreatModeHigh, [4][2]int32{{23, 0}, {23, 0}, {23, 0}, {23, 0}}},
		{"first game unknown", false, -1, [4][2]int32{{63, 19}, {63, 19}, {63, 19}, {63, 19}}},
		{"second game off", true, RetreatModeOff, [4][2]int32{}},
		{"second game low", true, RetreatModeLow, [4][2]int32{{0, 7}, {15, 7}, {23, 7}, {0, 7}}},
		{"second game high", true, RetreatModeHigh, [4][2]int32{{0, 23}, {47, 23}, {71, 23}, {0, 23}}},
		{"second game unknown", true, -1, [4][2]int32{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var w *World
			if tc.second {
				w = secondWorld(t, 1, engBounds, nil, actors...)
			} else {
				w = engWorld(t, engRel(t), actors...)
			}
			before := w.entities[4]
			Step(w, []Command{SetPlayerParameter(SelfSlot, PlayerParameterRetreat, tc.mode)})
			for i, want := range tc.want {
				e := w.entities[i]
				if got := [2]int32{e.Withdraw, e.Wimpy}; got != want {
					t.Errorf("actor %d thresholds = %v, want %v", e.ID, got, want)
				}
			}
			if e := w.entities[4]; e.Withdraw != before.Withdraw || e.Wimpy != before.Wimpy {
				t.Errorf("another owner's thresholds = %d/%d, want %d/%d", e.Withdraw, e.Wimpy, before.Withdraw, before.Wimpy)
			}
		})
	}
}
