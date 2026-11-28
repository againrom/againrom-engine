package sim

import "testing"

func TestFirstGameRefusedWimpyMoveStandsDownWithoutStrike(t *testing.T) {
	intruder := withdrawalFighter(2, SelfSlot, 21, 20, 100)
	intruder.ScanRange = 0

	t.Run("Move state 1 stands down on first refusal then reacquires", func(t *testing.T) {
		w := pocketedArcher(t, ScriptROM1, intruder)
		w.entities[0].Wimpy = w.entities[0].HP
		w.entities[0].Withdraw = 0
		wakeGroups(w)
		w.commandGroup([]int{0}, orderMove, cell{x: 30, y: 20})
		w.walkAtStateOne([]int{0})
		w.issueGroupDestination([]int{0}, cell{x: 30, y: 20})

		regWithdrawalPass(w, nil)
		got := w.entities[0]
		o := w.savedOrder(got.ID)
		if got.HasAttackTarget || got.HasTarget || got.ActorState != actorStateAcquire || got.PostX != 20 || got.PostY != 20 {
			t.Fatalf("first refusal under Move: state %#x post (%d,%d) move %t attack %t; want stand-down 0xc at (20,20) with no attack",
				got.ActorState, got.PostX, got.PostY, got.HasTarget, got.HasAttackTarget)
		}
		if o == nil || o.State != uint32(actorStateAcquire) || o.Raw[8] != 0 || !o.stoodDown {
			t.Fatalf("saved order after stand-down = %+v, want state 0xc kind 0 stoodDown", o)
		}

		twin := worldRoundTripForTest(t, w)
		regWithdrawalPass(w, nil)
		regWithdrawalPass(twin, nil)
		if w.Hash() != twin.Hash() {
			t.Fatal("reloaded stand-down diverged on the next withdrawal pass")
		}
		if got = w.entities[0]; !got.HasAttackTarget || got.AttackTarget != intruder.ID || got.HasTarget {
			t.Fatalf("second refusal after stand-down: attack %t on %d move %t; want reacquisition of %d",
				got.HasAttackTarget, got.AttackTarget, got.HasTarget, intruder.ID)
		}
	})

	t.Run("latched Move and non-Move orders reacquire on first refusal", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			setup func(*World)
		}{
			{"latched Move", func(w *World) {
				w.commandGroup([]int{0}, orderMove, cell{x: 20, y: 20})
				w.walkAtStateOne([]int{0})
				w.stopWalkOrder(0, true)
			}},
			{"Guard order", func(w *World) {
				w.commandGroup([]int{0}, orderGuard, cell{x: 20, y: 20})
			}},
			{"explicit Retreat", func(w *World) {
				w.commandGroup([]int{0}, orderNone, cell{})
				w.entities[0].ActorState = actorStateRetreat
				w.entities[0].Retreat = RetreatContinuation{Known: true}
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				w := pocketedArcher(t, ScriptROM1, intruder)
				w.entities[0].Wimpy = w.entities[0].HP
				w.entities[0].Withdraw = 0
				wakeGroups(w)
				tc.setup(w)
				regWithdrawalPass(w, nil)
				got := w.entities[0]
				if !got.HasAttackTarget || got.AttackTarget != intruder.ID || got.HasTarget {
					t.Fatalf("%s: attack %t on %d move %t; want reacquisition of %d",
						tc.name, got.HasAttackTarget, got.AttackTarget, got.HasTarget, intruder.ID)
				}
			})
		}
	})
}

func TestSecondGameWithdrawalAndCoverDetectWideInvisibleByEdgeDistance(t *testing.T) {
	t.Run("withdrawal wide invisible enemy", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			detect uint8
			want   bool
		}{
			{"at edge distance 3", 3, true},
			{"one short of edge distance", 2, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				self := withdrawalFighter(1, 2, 20, 20, 8)
				self.Wimpy, self.Group = 8, 7
				mate := withdrawalFighter(2, 2, 20, 21, 100)
				mate.Group, mate.SeeInvisible = 7, tc.detect
				hostile := withdrawalFighter(3, 3, 15, 20, 100)
				hostile.TokenSize = 3
				script, err := newScript(ScriptROM2, nil, nil, nil)
				if err != nil {
					t.Fatal(err)
				}
				w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{},
					[]Entity{self, mate, hostile}, script, engRel(t, [3]uint32{2, 3, 1}))
				if err != nil {
					t.Fatal(err)
				}
				wakeGroups(w)
				hideEntity(t, w, 3)
				twin := worldRoundTripForTest(t, w)
				regWithdrawalPass(w, nil)
				regWithdrawalPass(twin, nil)
				if w.Hash() != twin.Hash() {
					t.Fatal("reloaded second-game withdrawal diverged")
				}
				if got := w.entities[0].HasTarget; got != tc.want {
					t.Fatalf("second-game withdrawal started = %t, want %t", got, tc.want)
				}
			})
		}
	})

	t.Run("cover wide invisible enemy", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			detect uint8
			want   EntityID
		}{
			{"at edge distance 3", 3, 3},
			{"one short of edge distance", 2, 4},
		} {
			t.Run(tc.name, func(t *testing.T) {
				defender := engFighter(1, 2, 18, 20)
				defender.Reach = 6
				defender.Group = 8
				defender.SeeInvisible = 3
				subject := engFighter(2, 2, 20, 20)
				subject.Reach = 1
				subject.Group = 7
				subject.SeeInvisible = tc.detect
				wide := engFighter(3, SelfSlot, 15, 20)
				wide.TokenSize = 3
				far := engFighter(4, SelfSlot, 24, 20)
				script, err := newScript(ScriptROM2, nil, nil, nil)
				if err != nil {
					t.Fatal(err)
				}
				w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{},
					[]Entity{defender, subject, wide, far}, script, engRel(t, [3]uint32{2, SelfSlot, 1}))
				if err != nil {
					t.Fatal(err)
				}
				hideEntity(t, w, 3)
				w.coverEngage(0, 1)
				if got := w.entities[0].AttackTarget; got != tc.want {
					t.Fatalf("second-game cover victim = %d, want %d", got, tc.want)
				}
			})
		}
	})
}

func TestFirstGameLoneCorpseInTheSquareAcquiresInsteadOfFleeing(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(w *World)
	}{
		{"explicit Retreat", func(w *World) {
			commandBeforePass(w, []Command{GroupRetreat(1, 2, 91)})
		}},
		{"Wimpy withdrawal", func(w *World) {
			w.entities[0].Wimpy = w.entities[0].HP
			regWithdrawalPass(w, nil)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			self := withdrawalFighter(1, 2, 20, 20, 100)
			corpse := withdrawalFighter(2, 3, 21, 20, -1)
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, corpse)
			tc.run(w)
			got := w.entities[0]
			if got.HasTarget {
				t.Fatalf("%s fled from a lone corpse to (%d,%d)", tc.name, got.TargetX, got.TargetY)
			}
			if !got.HasAttackTarget || got.AttackTarget != corpse.ID {
				t.Fatalf("%s on a lone corpse: attack %t on %d, want standing acquisition of %d",
					tc.name, got.HasAttackTarget, got.AttackTarget, corpse.ID)
			}
		})
	}
}

func TestSecondGamePocketFlightWalksToStandInAndStopsOnlyOnOwnCell(t *testing.T) {
	archer := withdrawalArcher(1, 2, 20, 20)
	archer.Wimpy = archer.HP
	intruder := withdrawalFighter(2, SelfSlot, 22, 20, 100)
	grid := make([]byte, engBounds.Width*engBounds.Height)
	for y := int32(18); y <= 22; y++ {
		for x := int32(15); x <= 19; x++ {
			if x == 19 && y == 20 {
				continue
			}
			grid[y*engBounds.Width+x] = blockGround
		}
	}
	script, err := newScript(ScriptROM2, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{Block: grid},
		[]Entity{archer, intruder}, script, engRel(t, [3]uint32{2, SelfSlot, 1}, [3]uint32{SelfSlot, 2, 2}))
	if err != nil {
		t.Fatal(err)
	}
	wakeGroups(w)
	regWithdrawalPass(w, nil)
	got := w.entities[0]
	if !got.HasTarget || got.TargetX != 17 || got.TargetY != 20 {
		t.Fatalf("pocket with another reachable cell: move %t to (%d,%d), want walk toward (17,20)",
			got.HasTarget, got.TargetX, got.TargetY)
	}
	Step(w, nil)
	if got = w.entities[0]; got.X != 19 || got.Y != 20 {
		t.Fatalf("pocket stand-in step ended at (%d,%d), want (19,20)", got.X, got.Y)
	}
}

func TestSecondGameTeleportNotCastBySupportOrIdleRouter(t *testing.T) {
	w := teleportWorld(t, true, true, 60, 100, 0, 30)
	w.entities[0].AutoSpell = 23
	w.entities[0].MaxMana = 60
	w.entities[1].X = 35 // far outside threat/flee range
	for range 32 {
		Step(w, nil)
	}
	if len(w.bookCasts) != 0 || w.entities[0].X != 20 || w.entities[0].Y != 20 {
		t.Fatalf("idle/support mage cast Teleport outside flee: casts %+v at (%d,%d)",
			w.bookCasts, w.entities[0].X, w.entities[0].Y)
	}
}
