package sim

import "testing"

func flierVetoRel(t *testing.T) Relations {
	return engRel(t, [3]uint32{1, 2, 1}, [3]uint32{2, 1, 1})
}

func flierVetoVictim(domain Domain) Entity {
	e := engFighter(2, 2, 6, 5)
	e.Domain = domain
	return e
}

func flierVetoAttacker(domain Domain, reach uint8) Entity {
	e := engFighter(1, 1, 5, 5)
	e.Domain = domain
	e.Reach = reach
	return e
}

func TestMeleeAttackOrderRefusesAFlierOnly(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		attacker Entity
		victim   Entity
		want     bool
	}{
		{"melee ground vs flier", flierVetoAttacker(DomainGround, 1), flierVetoVictim(DomainAir), false},
		{"melee spirit vs flier", flierVetoAttacker(DomainGhost, 1), flierVetoVictim(DomainAir), false},
		{"melee ground vs spirit", flierVetoAttacker(DomainGround, 1), flierVetoVictim(DomainGhost), true},
		{"melee ground vs ground", flierVetoAttacker(DomainGround, 1), flierVetoVictim(DomainGround), true},
		{"ranged ground vs flier", flierVetoAttacker(DomainGround, 4), flierVetoVictim(DomainAir), true},
		{"flier vs melee ground", flierVetoVictim(DomainAir), flierVetoAttacker(DomainGround, 1), true},
	} {
		w := engWorld(t, flierVetoRel(t), tc.attacker, tc.victim)
		a, v := tc.attacker.ID, tc.victim.ID
		if tc.attacker.ID == tc.victim.ID {
			t.Fatal("fixture ids collide")
		}
		Step(w, []Command{Attack(a, v)})
		got := laEnt(t, w, a)
		if got.HasAttackTarget != tc.want {
			t.Errorf("%s: holds attack = %v, want %v", tc.name, got.HasAttackTarget, tc.want)
		}
	}
}

func TestGuardPostEngagePassesOverAFlier(t *testing.T) {
	t.Parallel()
	melee := flierVetoAttacker(DomainGround, 1)
	flier := flierVetoVictim(DomainAir)
	ground := engFighter(3, 2, 7, 5)
	w := engWorld(t, flierVetoRel(t), melee, flier, ground)
	if !w.postEngage(0, cell{x: 5, y: 5}) {
		t.Fatal("guard found nothing, want the ground foe")
	}
	if got := laEnt(t, w, 1); got.AttackTarget != 3 {
		t.Errorf("guard victim = %d, want the ground foe 3", got.AttackTarget)
	}
	lone := engWorld(t, flierVetoRel(t), melee, flier)
	if lone.postEngage(0, cell{x: 5, y: 5}) || laEnt(t, lone, 1).HasAttackTarget {
		t.Error("guard took a flier as its only candidate")
	}
	ranged := melee
	ranged.Reach = 4
	rw := engWorld(t, flierVetoRel(t), ranged, flier)
	if !rw.postEngage(0, cell{x: 5, y: 5}) {
		t.Error("a ranged guard refused a flier")
	}
}

func TestMeleeWithOnlyAFlierInReachIdlesWithoutOscillating(t *testing.T) {
	t.Parallel()
	melee := flierVetoAttacker(DomainGround, 1)
	flier := flierVetoVictim(DomainAir)
	flier.Reach = 4
	w := engWorld(t, flierVetoRel(t), melee, flier)
	for tick := 0; tick < 200; tick++ {
		Step(w, nil)
		m := laEnt(t, w, 1)
		if m.HasAttackTarget || m.X != 5 || m.Y != 5 {
			t.Fatalf("tick %d: melee holds %v or moved to (%d,%d)", tick, m.HasAttackTarget, m.X, m.Y)
		}
	}
	if w.targetVetoed(1, 0) || !w.targetVetoed(0, 1) {
		t.Fatal("pair vetoes are wrong")
	}
}

func TestLoadedMeleeOrderOnAFlierIsDropped(t *testing.T) {
	t.Parallel()
	melee := flierVetoAttacker(DomainGround, 1)
	flier := flierVetoVictim(DomainAir)
	w := engWorld(t, flierVetoRel(t), melee, flier)
	e := &w.entities[0]
	e.AttackTarget, e.HasAttackTarget = 2, true
	Step(w, nil)
	if laEnt(t, w, 1).HasAttackTarget {
		t.Error("a loaded melee order on a flier survived a tick")
	}
	e = &w.entities[0]
	e.PendingAttackTarget, e.HasPendingAttackTarget = 2, true
	Step(w, nil)
	if got := laEnt(t, w, 1); got.HasPendingAttackTarget || got.HasAttackTarget {
		t.Error("a pending melee order on a flier survived a tick")
	}
}
