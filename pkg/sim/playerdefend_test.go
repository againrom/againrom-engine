package sim

import (
	"bytes"
	"testing"
)

func defendOrder1087(w *World, subject EntityID, ids ...EntityID) {
	var cmds []Command
	for _, id := range ids {
		cmds = append(cmds, Command{Kind: KindGroupDefend, Entity: id, X: int32(subject), Group: 91})
	}
	if len(cmds) > 0 {
		w.groupOrder(cmds, 0, make([]bool, len(cmds)))
	}
}

func TestPlayerDefendFreshGroupSelectedSubjectAndAdmission1087(t *testing.T) {
	w := engWorld(t, engRel(t), laFighter(1, 1, 7, 10, 10), laFighter(2, 1, 8, 20, 10), laFighter(3, 1, 7, 12, 10))
	outside := laEnt(t, w, 3)
	defendOrder1087(w, 1, 2, 1, 2, 999)
	g := commandGroupOf(t, w, 1, 2)
	if order, _, ok := w.groupState(1, g); !ok || order != 0 {
		t.Fatalf("new group order = %d/%v, want none", order, ok)
	}
	a, b := laEnt(t, w, 1), laEnt(t, w, 2)
	if a.ActorState != 0xc || a.HasEscortTarget || a.PostX != 10 || a.PostY != 10 || b.ActorState != 8 || b.EscortTarget != 1 || !b.HasEscortTarget || b.EscortRange != 3 {
		t.Fatalf("selected subject/defender = %+v / %+v", a, b)
	}
	if laEnt(t, w, 3) != outside {
		t.Fatal("authored group peer outside selection changed")
	}
	before := mustMarshal(t, w)
	for _, missing := range []EntityID{999, ^EntityID(0)} {
		defendOrder1087(w, missing, 1, 2)
		if !bytes.Equal(before, mustMarshal(t, w)) {
			t.Fatalf("missing subject %d changed world", missing)
		}
	}
	defendOrder1087(w, 3, 2)
	if laEnt(t, w, 3) != outside {
		t.Fatal("unselected protected subject changed")
	}
	if laEnt(t, w, 2).EscortTarget != 3 {
		t.Fatal("replacement did not name new subject")
	}
}

func TestPlayerDefendCloseCoverAndAcquire1087(t *testing.T) {
	// Only the protected owner is hostile to 3: cover must use that row.
	w := engWorld(t, engRel(t, [3]uint32{1, 3, 1}),
		laFighter(1, 1, 7, 20, 20), laFighter(2, 2, 8, 10, 20), laFighter(3, 3, 9, 23, 20))
	defendOrder1087(w, 1, 2)
	w.actorPass()
	if e := laEnt(t, w, 2); !e.HasTarget || e.TargetX != 20 || e.HasAttackTarget {
		t.Fatalf("distant defender must close before cover: %+v", e)
	}
	w.entities[1].X = 18
	w.actorPass()
	if e := laEnt(t, w, 2); !e.HasAttackTarget || e.AttackTarget != 3 {
		t.Fatalf("near defender did not cover protected owner's enemy: %+v", e)
	}
	// The subject is now selected too. Its own nearby enemy is within reach;
	// farther enemies may be covered by a defender but cannot pull it away.
	w.entities[2].X = 21
	defendOrder1087(w, 1, 1, 2)
	w.actorPass()
	if e := laEnt(t, w, 1); !e.HasAttackTarget || e.AttackTarget != 3 || e.HasTarget {
		t.Fatalf("selected protected member is inert: %+v", e)
	}
	hp := laEnt(t, w, 3).HP
	// Remove the other attack source: only the protected actor can hit now.
	w.commandStance([]int{1}, int32(orderStandGround))
	w.entities[1].clearAttack()
	w.clearOrder(1)
	for n := 0; n < 64 && laEnt(t, w, 3).HP == hp; n++ {
		Step(w, nil)
	}
	if laEnt(t, w, 3).HP >= hp {
		t.Fatal("acquisition never delivered damage")
	}
	w.entities[2].X, w.entities[2].Y = 35, 35
	w.actorPass()
	// The lost target is released beside a loaded cycle: the actor never chases
	// it, and the attack target clears once the cycle completes.
	if e := laEnt(t, w, 1); e.HasTarget || e.X != 20 || e.Y != 20 {
		t.Fatalf("acquire chased a lost/out-of-reach target: %+v", e)
	}
	for n := 0; n < 64 && laEnt(t, w, 1).HasAttackTarget; n++ {
		Step(w, nil)
	}
	if e := laEnt(t, w, 1); e.HasAttackTarget || e.HasTarget || e.X != 20 || e.Y != 20 {
		t.Fatalf("lost target kept after the cycle completed: %+v", e)
	}
}

func TestPlayerDefendRefusesUncommandableMembersAndPoints1087(t *testing.T) {
	a, b, c := laFighter(1, 1, 7, 10, 10), laFighter(2, 1, 7, 12, 10), laFighter(3, 0, 8, 14, 10)
	b.HP = 0
	w := engWorld(t, engRel(t), a, b, c)
	w.entities[0].OffMap = true
	before := mustMarshal(t, w)
	defendOrder1087(w, 2, 1, 2, 3, 999)
	if !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("off-map/dead/unowned/absent members changed")
	}
	w.entities[0].OffMap = false
	w.entities[1].OffMap = true
	before = mustMarshal(t, w)
	defendOrder1087(w, 2, 1)
	if !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("off-map subject changed group")
	}
}

func TestPlayerDefendNativeContinuationAndInterrupt1087(t *testing.T) {
	w := engWorld(t, engRel(t), laFighter(1, 1, 7, 28, 20), laFighter(2, 1, 8, 10, 20))
	checkpoint := func(stage string) {
		t.Helper()
		var restored World
		if err := restored.UnmarshalBinary(mustMarshal(t, w)); err != nil {
			t.Fatal(stage, err)
		}
		for n := 0; n < 33; n++ {
			Step(w, nil)
			Step(&restored, nil)
			if !bytes.Equal(mustMarshal(t, w), mustMarshal(t, &restored)) {
				t.Fatalf("%s continuation diverged at %d", stage, n)
			}
		}
	}
	checkpoint("before")
	Step(w, []Command{{Kind: KindGroupDefend, Entity: 2, X: 1, Group: 9}})
	checkpoint("following")
	if laEnt(t, w, 2).X <= 10 {
		t.Fatal("defender never accompanied subject")
	}
	Step(w, []Command{{Kind: KindGroupMoveTo, Entity: 2, X: 12, Y: 30, Group: 10}})
	if e := laEnt(t, w, 2); e.ActorState == 8 || e.HasEscortTarget || e.EscortRange != 0 {
		t.Fatalf("move retained escort: %+v", e)
	}
	checkpoint("replaced")
	defendOrder1087(w, 1, 2)
	Step(w, []Command{{Kind: KindAttack, Entity: 2, X: 1}})
	if e := laEnt(t, w, 2); e.ActorState == 8 || e.HasEscortTarget {
		t.Fatalf("attack retained escort: %+v", e)
	}
}

func TestPlayerDefendScrollAdmissionAndRefund1087(t *testing.T) {
	w := scrollWorld1090(t, 12, 2)
	w.entities[0].Owner = 1
	Step(w, []Command{{Kind: KindUseScroll, Entity: 1, X: 2}})
	if len(w.scrollCasts) != 1 {
		t.Fatal("scroll was not reserved")
	}
	before := mustMarshal(t, w)
	defendOrder1087(w, 999, 1)
	if !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("missing Defend subject canceled a reserved scroll")
	}
	w.entities[1].OffMap = true
	before = mustMarshal(t, w)
	defendOrder1087(w, 2, 1)
	if !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("off-map Defend subject canceled a reserved scroll")
	}
	w.entities[1].OffMap = false
	defendOrder1087(w, 2, 1)
	stock, _ := w.CarriedStacks(1)
	if len(w.scrollCasts) != 0 || len(stock) != 1 || stock[0].Count != 2 {
		t.Fatalf("admitted Defend did not refund scroll: stock=%+v casts=%+v", stock, w.scrollCasts)
	}
	if e := laEnt(t, w, 1); e.ActorState != actorStateDefend || !e.HasEscortTarget || e.EscortTarget != 2 {
		t.Fatalf("admitted Defend did not replace scroll approach: %+v", e)
	}
}
