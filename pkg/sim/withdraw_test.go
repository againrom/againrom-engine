package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func withdrawalFighter(id EntityID, owner uint32, x, y, hp int32) Entity {
	return Entity{ID: id, Owner: owner, X: x, Y: y, HP: hp, MaxHP: 100,
		ScanRange: 5, Reach: 1, DamageBase: 1, AlwaysHits: true}
}

func TestWithdrawalTailReplacesTheDecisionMadeOnTheProductionFullTick(t *testing.T) {
	self := withdrawalFighter(1, 2, 20, 20, 30)
	self.Withdraw = 30
	hostile := withdrawalFighter(2, 3, 21, 20, 100)
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)

	engRun(w, 1)
	got := w.entities[indexOfEntity(w.entities, self.ID)]
	if got.HasAttackTarget {
		t.Fatalf("ordinary group decision survived the post-dispatch tail: victim %d", got.AttackTarget)
	}
	if !got.HasTarget || got.TargetX != 17 || got.TargetY != 20 {
		t.Fatalf("post-dispatch destination = (%d,%d,%t), want (17,20,true)", got.TargetX, got.TargetY, got.HasTarget)
	}
	// Movement runs after the tail in the same tick. Seeing the first step is
	// what proves the production call was reached, rather than only the helper.
	if got.X != 19 || got.Y != 20 {
		t.Errorf("position after the withdrawal tick = (%d,%d), want first step (19,20)", got.X, got.Y)
	}
}

func TestWimpyPrecedesWithdrawAndEachUsesItsOwnRadius(t *testing.T) {
	t.Run("Wimpy reaches the actor radius", func(t *testing.T) {
		self := withdrawalFighter(1, 2, 20, 20, 8)
		self.Wimpy, self.Withdraw, self.ScanRange = 8, 30, 5
		hostile := withdrawalFighter(2, 3, 24, 20, 100)
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
		w.withdrawalPass()
		if got := w.entities[0]; !got.HasTarget || got.TargetX != 17 || got.TargetY != 20 {
			t.Fatalf("Wimpy destination = (%d,%d,%t), want (17,20,true)", got.TargetX, got.TargetY, got.HasTarget)
		}
	})

	t.Run("Withdraw does not inherit that radius", func(t *testing.T) {
		self := withdrawalFighter(1, 2, 20, 20, 9)
		self.Wimpy, self.Withdraw, self.ScanRange = 8, 30, 5
		hostile := withdrawalFighter(2, 3, 24, 20, 100)
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
		w.withdrawalPass()
		if got := w.entities[0]; got.HasTarget || got.HasAttackTarget {
			t.Fatalf("fixed-radius Withdraw reached a hostile four cells away: %+v", got)
		}
	})

	t.Run("a nonempty corpse block consumes Wimpy", func(t *testing.T) {
		self := withdrawalFighter(1, 2, 20, 20, 5)
		self.Wimpy, self.Withdraw, self.ScanRange = 10, 10, 1
		corpse := withdrawalFighter(2, 3, 19, 20, -1)
		live := withdrawalFighter(3, 3, 22, 20, 100)
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, corpse, live)
		w.withdrawalPass()
		got := w.entities[0]
		if !got.HasAttackTarget || got.AttackTarget != corpse.ID || got.HasTarget {
			t.Fatalf("dead-only Wimpy fallback = target(%d,%t) move(%d,%d,%t), want ordinary corpse acquisition and no Withdraw retreat",
				got.AttackTarget, got.HasAttackTarget, got.TargetX, got.TargetY, got.HasTarget)
		}
	})
}

func TestWithdrawalThresholdsAreAbsoluteInclusiveAndZeroIsInert(t *testing.T) {
	for _, tc := range []struct {
		name      string
		hp, limit int32
		want      bool
	}{
		{"equal", 15, 15, true},
		{"one above", 16, 15, false},
		{"zero default", 15, 0, false},
		{"negative authored", 15, -7, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			self := withdrawalFighter(1, 2, 20, 20, tc.hp)
			self.Withdraw = tc.limit
			hostile := withdrawalFighter(2, 3, 22, 20, 100)
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
			w.withdrawalPass()
			if got := w.entities[0].HasTarget; got != tc.want {
				t.Errorf("HasTarget = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestWithdrawalUsesTheMeanOfTheFilteredListAndDecodedIntegerGeometry(t *testing.T) {
	self := withdrawalFighter(1, 2, 20, 20, 10)
	self.Wimpy = 10
	a := withdrawalFighter(2, 3, 18, 18, 100)
	b := withdrawalFighter(3, 3, 18, 24, 100)
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, a, b)
	w.withdrawalPass()
	got := w.entities[0]
	// Fine mean (18.5,21.5) cells, fine self (20.5,20.5): delta (2,-1): +3 on X
	// and -1.5 truncated on Y from the centred self, then the signed
	// fixed-point cell extraction yields (23,19). A whole-cell self would give
	// (23,18).
	if !got.HasTarget || got.TargetX != 23 || got.TargetY != 19 {
		t.Fatalf("mean-hostile destination = (%d,%d,%t), want (23,19,true)", got.TargetX, got.TargetY, got.HasTarget)
	}
}

// A mover between cells is measured at the paid part of its accepted stride,
// not at the centre of the cell it is entering (AI-WITHDRAW-028).
func TestFleeCellMeasuresAMoverBetweenCellsAtItsFinePosition(t *testing.T) {
	self := withdrawalFighter(1, 2, 21, 20, 10)
	hostile := withdrawalFighter(2, 3, 24, 20, 100)
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
	x, y, ok := w.fleeCell(0, []int{1})
	if !ok || x != 18 || y != 20 {
		t.Fatalf("centred mover flees to (%d,%d,%t), want (18,20,true)", x, y, ok)
	}
	e := &w.entities[0]
	e.Stride = NativeStride{Present: true, FromX: 20, FromY: 20, ToX: 21, ToY: 20, Rate: 1, StepX: 16, StepY: 0}
	e.Transit, e.TransitTotal = 13, 16
	x, y, ok = w.fleeCell(0, []int{1})
	if !ok || x != 17 || y != 20 {
		t.Fatalf("mover three ticks into a stride from (20,20) flees to (%d,%d,%t), want (17,20,true)", x, y, ok)
	}
}

func TestWithdrawalZeroAxesAndPlayableEdgeClamps(t *testing.T) {
	for _, tc := range []struct {
		name         string
		sx, sy       int32
		hx, hy       int32
		wantX, wantY int32
	}{
		{"zero x", 20, 20, 20, 18, 20, 23},
		{"both zero", 20, 20, 20, 20, 23, 23},
		{"low edge", 10, 10, 12, 12, 8, 8},
		{"high edge", 38, 38, 36, 36, 39, 39},
	} {
		t.Run(tc.name, func(t *testing.T) {
			self := withdrawalFighter(1, 2, tc.sx, tc.sy, 10)
			self.Withdraw = 10
			hostile := withdrawalFighter(2, 3, tc.hx, tc.hy, 100)
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
			w.withdrawalPass()
			got := w.entities[0]
			if !got.HasTarget || got.TargetX != tc.wantX || got.TargetY != tc.wantY {
				t.Errorf("destination = (%d,%d,%t), want (%d,%d,true)", got.TargetX, got.TargetY, got.HasTarget, tc.wantX, tc.wantY)
			}
		})
	}
}

func TestWithdrawalDropsVictimRouteAndGroupSpeedBeforeWritingTheMove(t *testing.T) {
	self := withdrawalFighter(1, 2, 20, 20, 10)
	self.Withdraw, self.GroupSpeed = 10, 7
	self.AttackTarget, self.HasAttackTarget = 2, true
	self.TargetX, self.TargetY, self.HasTarget = 21, 20, true
	hostile := withdrawalFighter(2, 3, 21, 20, 100)
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
	w.routes[0] = []cell{{21, 20}}
	w.withdrawalPass()
	got := w.entities[0]
	if got.HasAttackTarget || got.GroupSpeed != 0 || len(w.routes[0]) != 0 {
		t.Fatalf("replacement retained victim=%t groupSpeed=%d route=%v", got.HasAttackTarget, got.GroupSpeed, w.routes[0])
	}
	if !got.HasTarget || got.TargetX != 17 || got.TargetY != 20 {
		t.Errorf("replacement move = (%d,%d,%t), want (17,20,true)", got.TargetX, got.TargetY, got.HasTarget)
	}
}

func TestEveryGroupAndActorDecisionReachesTheCommonWithdrawalTail(t *testing.T) {
	orders := []struct {
		name  string
		order uint8
	}{
		{"None", orderNone},
		{"Guard", orderGuard},
		{"Swarm", orderSwarm},
		{"Stand Ground", orderStandGround},
		{"Move", orderMove},
		{"Swarm 2", orderSwarm2},
	}
	for _, tc := range orders {
		t.Run("group "+tc.name, func(t *testing.T) {
			self := withdrawalFighter(1, 2, 20, 20, 10)
			self.Group, self.Withdraw = 7, 10
			self.TargetX, self.TargetY, self.HasTarget = 28, 28, true
			hostile := withdrawalFighter(2, 3, 21, 20, 100)
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
			setGroupOrder(t, w, self.Owner, self.Group, tc.order)
			w.engagementPass()
			ordinary := w.entities[0]
			var obs withdrawalObs
			w.withdrawalPassObserved(&obs)
			assertWithdrawalReplacement(t, obs.decisions, ordinary, 17, 20)
		})
	}

	actors := []struct {
		name  string
		state uint8
	}{
		{"Patrol", actorStatePatrol},
		{"Defend", actorStateDefend},
		{"Follow", actorStateFollow},
	}
	for _, tc := range actors {
		t.Run("actor "+tc.name, func(t *testing.T) {
			self := withdrawalFighter(1, 2, 20, 20, 10)
			self.Group, self.Withdraw, self.ActorState = 7, 10, tc.state
			self.PatrolHeadX, self.PatrolHeadY = 20, 20
			self.PatrolTailX, self.PatrolTailY, self.PatrolLeg = 30, 20, patrolLegTail
			if tc.state != actorStatePatrol {
				self.PatrolHeadX, self.PatrolHeadY = 0, 0
				self.PatrolTailX, self.PatrolTailY, self.PatrolLeg = 0, 0, patrolLegHead
				self.EscortTarget, self.HasEscortTarget, self.EscortRange = 3, true, 2
			}
			hostile := withdrawalFighter(2, 3, 21, 20, 100)
			subject := withdrawalFighter(3, 2, 30, 20, 100)
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile, subject)
			setGroupOrder(t, w, self.Owner, self.Group, orderNone)
			w.actorPass()
			ordinary := w.entities[0]
			var obs withdrawalObs
			w.withdrawalPassObserved(&obs)
			assertWithdrawalReplacement(t, obs.decisions, ordinary, 17, 20)
		})
	}
}

func assertWithdrawalReplacement(t *testing.T, got []WithdrawalDecision, ordinary Entity, x, y int32) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("withdrawal decisions = %d, want 1: %+v", len(got), got)
	}
	if !reflect.DeepEqual(got[0].Before, ordinary) {
		t.Fatalf("observed pre-tail decision differs:\n got  %+v\n want %+v", got[0].Before, ordinary)
	}
	after := got[0].After
	if after.HasAttackTarget || !after.HasTarget || after.TargetX != x || after.TargetY != y {
		t.Fatalf("tail replacement = attack(%d,%t) move(%d,%d,%t), want move (%d,%d)",
			after.AttackTarget, after.HasAttackTarget, after.TargetX, after.TargetY, after.HasTarget, x, y)
	}
}

func TestPlayerCommandedMoveSucceedsOrSurvivesWithTheWithdrawalTail(t *testing.T) {
	for _, tc := range []struct {
		name        string
		hostileX    int32
		wantRetreat bool
	}{
		{"successful tail replaces the command", 22, true},
		{"failed tail preserves the command", 24, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			self := withdrawalFighter(1, SelfSlot, 20, 20, 10)
			self.Withdraw = 10
			hostile := withdrawalFighter(2, 2, tc.hostileX, 20, 100)
			w := engWorld(t, engRel(t, [3]uint32{SelfSlot, 2, 1}), self, hostile)
			w.tick = scriptPassPhase
			decisions := StepWithdrawalTraced(w, []Command{{Kind: KindMoveTo, Entity: self.ID, X: 30, Y: 20}})
			got := w.entities[0]
			if tc.wantRetreat {
				if len(decisions) != 1 || decisions[0].Before.TargetX != 30 || decisions[0].Before.TargetY != 20 ||
					!decisions[0].Before.HasTarget || decisions[0].After.TargetX != 17 || decisions[0].After.TargetY != 20 {
					t.Fatalf("command/tail boundary = %+v", decisions)
				}
				return
			}
			if len(decisions) != 0 {
				t.Fatalf("failed tail reported decisions: %+v", decisions)
			}
			if !got.HasTarget || got.TargetX != 30 || got.TargetY != 20 || got.X != 21 || got.Y != 20 {
				t.Fatalf("failed tail did not preserve commanded move: %+v", got)
			}
		})
	}
}

func TestFailedWithdrawalTailPreservesThePriorOrderByteForByte(t *testing.T) {
	self := withdrawalFighter(1, 2, 20, 20, 10)
	self.Withdraw, self.TargetX, self.TargetY, self.HasTarget = 10, 30, 27, true
	hostile := withdrawalFighter(2, 3, 24, 20, 100)
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
	w.routes[0] = []cell{{21, 21}, {22, 22}}
	before, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var obs withdrawalObs
	w.withdrawalPassObserved(&obs)
	after, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(obs.decisions) != 0 || !bytes.Equal(after, before) {
		t.Fatalf("failed tail changed the world: decisions=%+v equal=%t", obs.decisions, bytes.Equal(after, before))
	}
}

func TestWithdrawalEligibilityAndHostileVisibilityPopulation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*World)
	}{
		{"dead actor", func(w *World) { w.entities[0].HP = -1 }},
		{"off-map actor", func(w *World) { w.entities[0].OffMap = true }},
		{"owner-zero actor", func(w *World) { w.entities[0].Owner = 0 }},
		{"Stone actor", func(w *World) {
			w.attached = []attachedEffect{{Target: 1, Spell: 20, Kind: EffectAbsorption, Mode: EffectDuration, Remaining: 20}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			self := withdrawalFighter(1, 2, 20, 20, 10)
			self.Withdraw, self.TargetX, self.TargetY, self.HasTarget = 10, 30, 30, true
			hostile := withdrawalFighter(2, 3, 21, 20, 100)
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
			tc.mutate(w)
			before := w.entities[0]
			w.withdrawalPass()
			if !reflect.DeepEqual(w.entities[0], before) {
				t.Fatalf("excluded actor changed:\n before %+v\n after  %+v", before, w.entities[0])
			}
		})
	}

	t.Run("zero sight does not bypass the invisibility filter", func(t *testing.T) {
		self := withdrawalFighter(1, 2, 20, 20, 10)
		self.Withdraw, self.ScanRange, self.SeeInvisible = 10, 0, 0
		hostile := withdrawalFighter(2, 3, 21, 20, 100)
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
		w.attached = []attachedEffect{{Target: hostile.ID, Spell: 15, Kind: EffectInvisible,
			Mode: EffectDuration, Remaining: 20}}
		w.withdrawalPass()
		if got := w.entities[0]; got.HasTarget {
			t.Fatal("invisible hostile triggered withdrawal")
		}
		w.attached = nil
		w.withdrawalPass()
		if got := w.entities[0]; !got.HasTarget || got.TargetX != 17 || got.TargetY != 20 {
			t.Fatal("zero sight prevented spatial withdrawal")
		}
	})

	t.Run("off-map hostile does not count", func(t *testing.T) {
		self := withdrawalFighter(1, 2, 20, 20, 10)
		self.Withdraw = 10
		hostile := withdrawalFighter(2, 3, 21, 20, 100)
		hostile.OffMap = true
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
		before := w.entities[0]
		w.withdrawalPass()
		if !reflect.DeepEqual(w.entities[0], before) {
			t.Fatalf("off-map hostile triggered withdrawal: %+v", w.entities[0])
		}
	})
}

func TestForm60ResumesWithTheSameNextWithdrawalDecision(t *testing.T) {
	self := withdrawalFighter(1, 2, 20, 20, 10)
	self.Withdraw = 10
	hostile := withdrawalFighter(2, 3, 21, 20, 100)
	original := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
	original.tick = scriptPassPhase
	form, err := original.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var loaded World
	if err := loaded.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	var unobserved World
	if err := unobserved.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	want := StepWithdrawalTraced(original, nil)
	got := StepWithdrawalTraced(&loaded, nil)
	Step(&unobserved, nil)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("loaded next withdrawal = %+v, want %+v", got, want)
	}
	wantForm, wantErr := original.MarshalBinary()
	gotForm, gotErr := loaded.MarshalBinary()
	plainForm, plainErr := unobserved.MarshalBinary()
	if wantErr != nil || gotErr != nil || plainErr != nil || !bytes.Equal(gotForm, wantForm) || !bytes.Equal(gotForm, plainForm) {
		t.Fatalf("resumed worlds differ: original err=%v loaded err=%v unobserved err=%v resumed=%t observation-neutral=%t",
			wantErr, gotErr, plainErr, bytes.Equal(gotForm, wantForm), bytes.Equal(gotForm, plainForm))
	}
}

func TestCanonicalInt32HealthDoesNotWrapAtTheOriginalSignedWordBoundary(t *testing.T) {
	self := withdrawalFighter(1, 2, 20, 20, 65566)
	self.MaxHP, self.Withdraw = 70000, 65565
	hostile := withdrawalFighter(2, 3, 21, 20, 100)
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
	if got := w.entities[0].HP; got != 65566 {
		t.Fatalf("constructor HP = %d, want 65566", got)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var loaded World
	if err := loaded.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if got := loaded.entities[0].HP; got != 65566 {
		t.Fatalf("form-60 HP = %d, want 65566", got)
	}
	before := loaded.entities[0]
	loaded.withdrawalPass()
	if !reflect.DeepEqual(loaded.entities[0], before) {
		t.Fatalf("int32 HP %d wrapped below threshold %d: %+v", before.HP, before.Withdraw, loaded.entities[0])
	}
}

func TestWithdrawalThresholdsRoundTripAndEachChangesTheDigest(t *testing.T) {
	base := withdrawalFighter(1, 2, 20, 20, 50)
	build := func(withdraw, wimpy int32) *World {
		e := base
		e.Withdraw, e.Wimpy = withdraw, wimpy
		return engWorld(t, Relations{}, e)
	}
	zero, withdraw, wimpy := build(0, 0), build(17, 0), build(0, 19)
	if zero.Hash() == withdraw.Hash() || zero.Hash() == wimpy.Hash() || withdraw.Hash() == wimpy.Hash() {
		t.Fatalf("distinct threshold states hash alike: zero=%#x withdraw=%#x wimpy=%#x",
			zero.Hash(), withdraw.Hash(), wimpy.Hash())
	}

	form, err := build(-17, 190).MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got := back.entities[0]; got.Withdraw != -17 || got.Wimpy != 190 {
		t.Fatalf("round trip Withdraw/Wimpy = %d/%d, want -17/190", got.Withdraw, got.Wimpy)
	}
	again, err := back.MarshalBinary()
	if err != nil || !bytes.Equal(again, form) {
		t.Fatalf("round trip bytes differ: err=%v", err)
	}
}

func TestBlockedWithdrawalTargetUsesTheOrdinaryReachableSubstitute(t *testing.T) {
	self := withdrawalFighter(1, 2, 20, 20, 10)
	self.Withdraw = 10
	hostile := withdrawalFighter(2, 3, 21, 20, 100)
	grid := make([]byte, engBounds.Width*engBounds.Height)
	grid[20*engBounds.Width+17] = blockGround
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{Block: grid},
		[]Entity{self, hostile}, nil, engRel(t, [3]uint32{2, 3, 1}))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	w.withdrawalPass()
	Step(w, nil)
	got := w.entities[0]
	if got.TargetX != 18 || got.TargetY != 20 || !got.HasTarget {
		t.Fatalf("route substitute = (%d,%d,%t), want nearest reachable (18,20,true)", got.TargetX, got.TargetY, got.HasTarget)
	}
	if got.X == 20 && got.Y == 20 {
		t.Fatal("blocked destination produced no reachable substitute step")
	}
}
