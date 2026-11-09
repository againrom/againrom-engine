package sim

import "testing"

func TestAcquisitionActivityUsesTheSaveProjection(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1043: direct actor and Group acquisition prepare the same byte.
	for _, second := range []bool{false, true} {
		w := orderFallbackScene(t, second, "saved acquisition")
		o := *w.savedOrder(asHero)
		want := byte(5)
		if second {
			want = 6
		}
		if o.Raw[8] != want {
			t.Fatalf("second=%v current activity=%d, want %d", second, o.Raw[8], want)
		}
		projected := ProjectActorAcquireOrder(w.entities[0], o)
		if projected.Raw[8] != 6 {
			t.Fatal("save projection lost activity 6")
		}
		projected.Raw[8] = o.Raw[8]
		if projected.Raw != o.Raw {
			t.Fatal("acquisition projection changed another byte")
		}
	}
}

func TestIdleBodyPromotionUsesTheMainFilter(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-502: an empty live list promotes bodies before idle ranking.
	self := asFighter(asHero, SelfSlot, 30, 40)
	self.Facing = 64
	a, b := asFoe(asFoeA, 31, 40), asFoe(asFoeB, 30, 41)
	a.HP, a.Domain, b.HP = -1, DomainAir, -1
	w := asWorld(t, true, self, a, b)
	w.acquireStanding(0)
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != asFoeB {
		t.Fatalf("promoted body bypassed the main veto: target=%v/%d", e.HasAttackTarget, e.AttackTarget)
	}
}

func TestIdleBodyPromotionKeepsTheFlierAdjustment(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-502 and R2-ENGINE-501: promoted bodies use the main rank.
	w := orderFallbackScene(t, true, "body flier")
	if w.entities[0].HasAttackTarget {
		t.Fatal("promoted flier at the range boundary bypassed the extra cell")
	}
}

func TestIdleReserveBodyKeepsItsSeparateRank(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-501: a live-list miss still tries the separate body population.
	self := asFighter(asHero, SelfSlot, 30, 40)
	self.Reach, self.Facing = 2, 64
	body := asFoe(asFoeA, 32, 40)
	body.HP, body.Domain = -1, DomainAir
	w := asWorld(t, true, self, body, asFoe(asFoeB, 34, 40))
	if live, _ := w.actorCandidates(0); len(live) != 1 {
		t.Fatal("reserve fixture requires a visible living candidate outside reach")
	}
	w.acquireStanding(0)
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != asFoeA {
		t.Fatalf("reserve body lost its separate rank: target=%v/%d", e.HasAttackTarget, e.AttackTarget)
	}
}

func orderFallbackScene(t *testing.T, second bool, kind string) *World {
	t.Helper()
	self := asFighter(asHero, SelfSlot, 30, 40)
	self.Facing = 64
	body := asFoe(asFoeA, 31, 40)
	body.HP, body.Domain = -1, DomainAir
	switch kind {
	case "empty guard", "empty pursuit", "empty none", "repick range":
		self.Reach = 3
		foe := asFoe(asFoeA, 31, 40)
		if kind == "empty none" {
			foe.X = 40
		}
		w := asWorld(t, second, self, foe)
		w.commandGroup([]int{0}, orderNone, cell{})
		w.entities[0].ActorState = actorStateGuard
		o := w.syncSavedActorCommand(0)
		o.Raw[8] = 1
		w.walkToCell(0, 45, 40, false)
		if kind == "empty pursuit" {
			w.orderAttack(0, asFoeA)
			w.entities[0].ActorState, o.State = actorStateEngage, uint32(actorStateEngage)
			w.answerRefusedPursuit(0, searchStatic)
		} else if kind == "repick range" {
			w.entities[1].X = 33
			w.reacquisitionVictim(0)
		} else {
			w.answerRefusedWalk(0, searchStatic)
		}
		return w
	case "idle escort":
		w := asWorld(t, second, self, asFighter(asFoeA, SelfSlot, 32, 40))
		w.commandDefend([]int{0, 1}, asHero)
		w.entities[1].EscortOrder, w.entities[1].EscortTurnPending = escortOrderIdle, true
		w.stepEscortOrder(nil, 1)
		return w
	case "computer support":
		self.Owner = asSide
		w := asWorld(t, second, self)
		w.commandGroup([]int{0}, orderNone, cell{})
		w.savedOrder(asHero).Raw[8], w.savedOrder(asHero).Raw[0x54] = 0xb, 1
		wakeGroups(w)
		Step(w, nil)
		return w
	case "body veto", "body flier", "body reserve":
		if kind == "body flier" {
			self.Reach, body.X = 2, 32
		}
		ents := []Entity{self, body}
		if kind == "body reserve" {
			ents = append(ents, asFoe(asFoeB, 36, 40))
		}
		w := asWorld(t, second, ents...)
		w.acquireStanding(0)
		return w
	case "saved acquisition":
		w := asWorld(t, second, self, asFoe(asFoeA, 31, 40))
		w.acquireInPlace(0)
		w.savedAcquire(0, w.savedOrder(asHero))
		return w
	case "joined score":
		self.Owner, self.Reach = asSide, 4
		a, b := asFighter(asFoeA, SelfSlot, 30, 39), asFighter(asFoeB, SelfSlot, 31, 40)
		for _, e := range []*Entity{&a, &b} {
			e.HasAttackTarget, e.AttackTarget, e.AttackPhase = true, asHero, AttackCharging
		}
		w := asWorld(t, second, self, a, b)
		w.joinEngagedGroup(asSide, orderSwarm, noticeCircle{}, []int{0}, nil, nil)
		if !w.entities[0].HasAttackTarget {
			t.Fatal("join fixture did not assign a target")
		}
		return w
	case "cover preference":
		self.Reach = 4
		leader := asFighter(asFoeA, SelfSlot, 30, 41)
		a, b := asFoe(asFoeB, 31, 40), asFoe(4, 32, 41)
		b.Domain = DomainAir
		w := asWorld(t, second, self, leader, a, b)
		w.coverEngage(0, 1)
		if !w.entities[0].HasAttackTarget || w.entities[0].AttackTarget != 4 {
			t.Fatal("cover fixture did not prefer the flier")
		}
		return w
	}
	t.Fatalf("unknown order scene %q", kind)
	return nil
}
