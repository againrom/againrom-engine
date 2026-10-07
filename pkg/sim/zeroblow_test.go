package sim

import "testing"

// zeroBlowFight runs an attack order of entity 1 on entity 2 for ticks ticks
// and returns every damage event the report named.
func zeroBlowFight(t *testing.T, a, v Entity, ticks int) (events []DamageEvent, w *World) {
	t.Helper()
	w = cbWorld(t, 5, a, v)
	report := StepReported(w, []Command{Attack(1, 2)})
	events = append(events, report.Damages...)
	for i := 0; i < ticks; i++ {
		events = append(events, StepReported(w, nil).Damages...)
	}
	return events, w
}

func zeroBlowAttacker(damage int32) Entity {
	a := cbEnt(1, 1, 1)
	a.ToHit, a.DamageBase, a.AttackCharge, a.AttackRelax = 2000, damage, 2, 2
	return a
}

// A unit strike whose total is zero is reported with the victim's level
// unchanged, once per strike (ANIM-125). A strike that takes health is
// reported as the decrease alone.
func TestAnAbsorbedStrikeIsReportedWithHealthUnchanged(t *testing.T) {
	v := cbEnt(2, 2, 1)
	v.Absorption = 5
	events, w := zeroBlowFight(t, zeroBlowAttacker(3), v, 60)
	if len(events) < 2 {
		t.Fatalf("the fixture struck %d times in 60 ticks", len(events))
	}
	for _, e := range events {
		if e != (DamageEvent{Target: 2, BeforeHP: 100, AfterHP: 100}) {
			t.Fatalf("event %+v, want the unchanged level 100", e)
		}
	}
	if got := cbAt(t, w, 2).HP; got != 100 {
		t.Fatalf("victim at %d", got)
	}

	v.Absorption = 0
	events, _ = zeroBlowFight(t, zeroBlowAttacker(3), v, 60)
	for _, e := range events {
		if e.BeforeHP == e.AfterHP {
			t.Fatalf("a strike that took health reported %+v", e)
		}
	}
	if len(events) < 2 {
		t.Fatalf("the fixture struck %d times in 60 ticks", len(events))
	}
}

// A strike on a body still above the finished-body floor is reported; a body at
// the floor is no longer a target.
func TestAZeroStrikeOnAFallenBodyIsReportedAboveTheFloor(t *testing.T) {
	v := cbEnt(2, 2, 1)
	v.HP, v.Absorption = -5, 5
	events, _ := zeroBlowFight(t, zeroBlowAttacker(3), v, 60)
	if len(events) == 0 {
		t.Fatal("no strike reported on a body at -5")
	}
	for _, e := range events {
		if e != (DamageEvent{Target: 2, BeforeHP: -5, AfterHP: -5}) {
			t.Fatalf("event %+v", e)
		}
	}
}

// A report changes no state: a world stepped with the sink and one stepped
// without it hash alike.
func TestReportingAZeroStrikeChangesNoWorldState(t *testing.T) {
	v := cbEnt(2, 2, 1)
	v.Absorption = 5
	_, observed := zeroBlowFight(t, zeroBlowAttacker(3), v, 60)
	plain := cbWorld(t, 5, zeroBlowAttacker(3), v)
	Step(plain, []Command{Attack(1, 2)})
	for i := 0; i < 60; i++ {
		Step(plain, nil)
	}
	if plain.Hash() != observed.Hash() {
		t.Fatal("the report moved the world")
	}
}

// A strike on a structure sends the structure message and never the actor's
// damage message, so it reports nothing, whether it takes health or not
// (ANIM-125).
func TestAStructureStrikeReportsNoDamageEvent(t *testing.T) {
	for _, base := range []uint8{3, 20} {
		a, s := structureCombatActor(), structureCombatTarget()
		a.SecondaryDamage.Base, a.SecondaryDamage.Spread = base, 1
		w := structureCombatWorld(t, 0, a, s)
		report := StepReported(w, []Command{AttackStructure(0, 0)})
		var events []DamageEvent
		events = append(events, report.Damages...)
		for i := 0; i < 40; i++ {
			events = append(events, StepReported(w, nil).Damages...)
		}
		if base == 20 && w.structures[0].Field42 >= 100 {
			t.Fatalf("base %d: the fixture did not strike the structure", base)
		}
		if len(events) != 0 {
			t.Fatalf("base %d: structure strike reported %+v", base, events)
		}
	}
}
