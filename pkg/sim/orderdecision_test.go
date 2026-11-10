package sim

import "testing"

func TestIdleTargetRankUsesDistanceTurnAndLaterTie(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-501, R2-ENGINE-1018, R2-ENGINE-1043.
	for _, tc := range []struct {
		name string
		a, b Entity
		want EntityID
	}{
		{"nearest before preference", asFoe(asFoeA, 31, 40), asFoe(asFoeB, 32, 40), asFoeA},
		{"least turn", asFoe(asFoeA, 30, 39), asFoe(asFoeB, 31, 40), asFoeB},
		{"later full tie", asFoe(asFoeA, 31, 39), asFoe(asFoeB, 31, 41), asFoeB},
	} {
		t.Run(tc.name, func(t *testing.T) {
			self := asFighter(asHero, SelfSlot, 30, 40)
			self.Reach, self.Facing = 4, 64
			if tc.name == "nearest before preference" {
				tc.b.Domain = DomainGhost
			}
			w := asWorld(t, true, self, tc.a, tc.b)
			w.acquireStanding(0)
			if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != tc.want || !e.AcquirePursuit {
				t.Fatalf("idle target=%v/%d acquisition=%v, want %d", e.HasAttackTarget, e.AttackTarget, e.AcquirePursuit, tc.want)
			}
		})
	}
}

func TestStandGroundScoreUsesTurnAndUnconditionalPreference(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1111: scorer and assignment retain the first full tie.
	self := asFighter(asHero, asSide, 30, 40)
	self.Facing, self.Mind = 64, 0
	w := asWorld(t, true, self, asFighter(asFoeA, SelfSlot, 30, 39), asFighter(asFoeB, SelfSlot, 31, 40))
	if got := w.candidateCost(0, 1, orderStandGround); got != 320 {
		t.Fatalf("north score=%d, want 256+64", got)
	}
	if got := w.chooseTarget(0, []int{1, 2}, orderStandGround, targetGroupScore); got != 2 {
		t.Fatalf("group selected index %d, want east index 2", got)
	}
	w.entities[1].Domain = DomainAir
	w.entities[0].Reach, w.entities[1].Reach = 4, 4
	if got := w.candidateCost(0, 1, orderStandGround); got != 160 {
		t.Fatalf("ranged flier score=%d, want (256+64)/2", got)
	}
}

func TestSavedIdlePreparationKeepsMemberOrder(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1043, R2-ENGINE-1105: activity is not member order.
	self := asFighter(asHero, SelfSlot, 30, 40)
	self.Facing = 64
	w := asWorld(t, true, self, asFoe(asFoeA, 31, 40))
	w.acquireInPlace(0)
	o := w.savedOrder(asHero)
	w.savedAcquire(0, o)
	if o.State != uint32(actorStateAcquire) || o.Raw[8] != 6 {
		t.Fatalf("member order=%d activity=%d, want 12/6", o.State, o.Raw[8])
	}
}

func TestAttackAdmissionUsesLiveEdgeDistanceBeforeApplication(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1106 separates initiation from HP application.
	for _, second := range []bool{false, true} {
		self := asFighter(asHero, SelfSlot, 30, 40)
		self.Reach, self.AttackCharge, self.Facing = 2, 8, 0
		w := asWorld(t, second, self, asFoe(asFoeA, 33, 40))
		w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{
			Entity: asFoeA, Current: true, Position: SavedActorPosition{FineX: 64, FineY: 128},
		}}}
		if got := w.closedOn(0, 1); got != second {
			t.Fatalf("second=%v admission=%v, want %v at live distance 2", second, got, second)
		}
		if inReach(w.entities[0], w.entities[1]) {
			t.Fatal("application reused admission distance")
		}
		w.orderAcquire(0, asFoeA)
		w.advanceAttack(0, nil)
		if w.entities[0].AttackPhase != AttackReady {
			t.Fatal("strike started before facing admission")
		}
		w.entities[0].clearTurn()
		w.entities[0].Facing = 64
		w.advanceAttack(0, nil)
		if got := w.entities[0].AttackPhase; (got == AttackCharging) != second {
			t.Fatalf("second=%v phase=%d after facing admission", second, got)
		}
	}
}

func TestIdleTargetFallsBackAfterLivingCandidatesMissReach(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-501: fallback is also tried after a live-list miss.
	self := asFighter(asHero, SelfSlot, 30, 40)
	dead := asFoe(asFoeA, 31, 40)
	dead.HP = -1
	w := asWorld(t, true, self, dead, asFoe(asFoeB, 36, 40))
	w.acquireStanding(0)
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != asFoeA {
		t.Fatalf("fallback target=%v/%d, want %d", e.HasAttackTarget, e.AttackTarget, asFoeA)
	}
}

func TestPlayerMageExceptionOnlySuppressesAcquisition(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1106: the activity-6 exception does not forbid order 3.
	self := asFighter(asHero, SelfSlot, 30, 40)
	self.MaxMana, self.Facing = 100, 64
	w := asWorld(t, true, self, asFoe(asFoeA, 31, 40))
	w.acquireInPlace(0)
	w.savedAcquire(0, w.savedOrder(asHero))
	if e := w.entities[0]; !e.PursuitIdle || w.savedOrder(asHero).Raw[8] != 0 {
		t.Fatal("short-range player mage did not idle its acquisition")
	}
	Step(w, []Command{Attack(asHero, asFoeA)})
	if e := w.entities[0]; e.AcquirePursuit || e.PursuitIdle || !e.HasAttackTarget {
		t.Fatal("explicit attack inherited the acquisition exception")
	}
}

func TestEmptyWalkStopAndArrivalKeepDifferentLatches(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1108, R2-ENGINE-1076: empty route and arrival are distinct.
	for _, arrival := range []bool{false, true} {
		w := asWorld(t, true, asFighter(asHero, SelfSlot, 30, 40))
		w.entities[0].ActorState = 1
		w.syncSavedActorCommand(0)
		if arrival {
			w.stopWalkOrder(0, true)
		} else {
			w.answerRefusedWalk(0, searchStatic)
		}
		o := w.savedOrder(asHero)
		if e := w.entities[0]; e.ActorState != actorStateAcquire || o.State != 12 || o.Raw[8] != 0 || e.PostX != 30 || e.PostY != 40 {
			t.Fatalf("arrival=%v did not stop at its own cell: entity=%+v order=%+v", arrival, e, o)
		}
		if (o.Raw[0x50] == 1) != arrival {
			t.Fatalf("arrival=%v latch=%d", arrival, o.Raw[0x50])
		}
	}
}

func TestFirstGameOrderCellsKeepTheirBytes(t *testing.T) {
	t.Parallel()
	for primary, want := range map[uint8]uint64{4: 5639754465014353473, 7: 14881767044725322728, 255: 2175796466901727936} {
		w := primaryCellScene(t, false, primary)
		for range 20 {
			Step(w, nil)
		}
		if got := w.Hash(); got != want {
			t.Errorf("first-game imported primary %d hash=%d, want pre-change %d", primary, got, want)
		}
	}
	for _, kind := range []string{"body veto", "body flier", "body reserve", "saved acquisition", "joined score", "cover preference", "idle escort", "computer support", "empty guard", "empty pursuit", "empty none", "repick range"} {
		t.Run(kind, func(t *testing.T) {
			w := orderFallbackScene(t, false, kind)
			want := map[string]uint64{
				"body veto": 17157284657881764089, "body flier": 3072154270610456809,
				"body reserve": 18085316621644716136, "saved acquisition": 16676042835147640960,
				"joined score": 8019895757929327814, "cover preference": 12196724387676937679,
				"idle escort": 16410490491839632455, "computer support": 4450317173693816472,
				"empty guard": 522980333038066288, "empty pursuit": 9994414873249793888,
				"empty none": 11314713690971926999, "repick range": 16341346082256859235,
			}[kind]
			if got := w.Hash(); got != want {
				t.Fatalf("first-game %s hash=%d, want pre-change %d", kind, got, want)
			}
		})
	}
	for _, kind := range []string{"idle", "score", "admission", "empty walk"} {
		t.Run(kind, func(t *testing.T) {
			self := asFighter(asHero, SelfSlot, 30, 40)
			self.Reach, self.Facing = 4, 64
			w := asWorld(t, false, self, asFoe(asFoeA, 30, 39), asFoe(asFoeB, 31, 40))
			switch kind {
			case "idle":
				w.acquireStanding(0)
			case "score":
				if got := w.candidateCost(0, 1, orderStandGround); got != 512 {
					t.Fatalf("first-game score=%d, want 512", got)
				}
			case "admission":
				w.orderAcquire(0, asFoeB)
				w.advanceAttack(0, nil)
			case "empty walk":
				w.walkToCell(0, 45, 40, false)
				w.entities[0].ActorState = 1
				w.syncSavedActorCommand(0)
				w.answerRefusedWalk(0, searchStatic)
			}
			want := map[string]uint64{
				"idle":       17586582087991034016,
				"score":      14172829896077500475,
				"admission":  12156141118182599328,
				"empty walk": 3291890885520310301,
			}[kind]
			if got := w.Hash(); got != want {
				t.Fatalf("first-game %s hash=%d, want pre-change %d", kind, got, want)
			}
		})
	}
}
