package sim

import "testing"

// shooterDeathWorld is an archer ordered onto a victim five cells away.
func shooterDeathWorld(t *testing.T) *World {
	t.Helper()
	ents := []Entity{
		{ID: 1, X: 2, Y: 4, HP: 100, MaxHP: 100, Owner: 1, Reach: 6, TokenSize: 1,
			AttackCharge: 12, AttackRelax: 6, DamageBase: 5, AlwaysHits: true},
		{ID: 2, X: 7, Y: 4, HP: 100, MaxHP: 100, Owner: 2, TokenSize: 1},
	}
	w := shotWorld(t, ents...)
	Step(w, []Command{Attack(1, 2)})
	return w
}

func victimHP(w *World) int32 {
	return w.entities[indexOfEntity(w.entities, 2)].HP
}

// SAV-1192: a shooter whose hit points fall to 0 or below while its blow is
// counting down never applies that blow. The control run, with the shooter
// alive, lands the blow inside the same window.
func TestAShooterKilledDuringItsCountdownAppliesNoDamage(t *testing.T) {
	control := shooterDeathWorld(t)
	hit := -1
	for n := 0; n < 60 && hit < 0; n++ {
		Step(control, nil)
		if victimHP(control) < 100 {
			hit = n
		}
	}
	if hit < 2 {
		t.Fatalf("the control blow landed at step %d; the witness needs a countdown to interrupt", hit)
	}
	for _, hp := range []int32{0, -3} {
		w := shooterDeathWorld(t)
		for n := 0; n < hit-1; n++ {
			Step(w, nil)
		}
		i := indexOfEntity(w.entities, 1)
		if w.entities[i].AttackCountdown <= 0 {
			t.Fatalf("hp %d: no countdown pending one step before the blow", hp)
		}
		w.entities[i].setCurrentHealth(hp)
		w.clearFelled(i)
		for n := 0; n < 60; n++ {
			Step(w, nil)
		}
		if got := victimHP(w); got != 100 {
			t.Errorf("shooter at hp %d: the victim fell to %d, want the pending blow never applied", hp, got)
		}
	}
}

// SAV-1194: an actor taken off the map is not ticked, so its pending
// countdown is not advanced while it is away.
func TestAnOffMapAttackerKeepsItsCountdown(t *testing.T) {
	w := shooterDeathWorld(t)
	Step(w, nil)
	i := indexOfEntity(w.entities, 1)
	before := w.entities[i]
	if before.AttackCountdown <= 0 {
		t.Fatal("no countdown pending")
	}
	w.takeOffMap(i)
	for n := 0; n < 20; n++ {
		Step(w, nil)
	}
	after := w.entities[indexOfEntity(w.entities, 1)]
	if after.AttackCountdown != before.AttackCountdown || after.AttackPhase != before.AttackPhase {
		t.Fatalf("off the map the countdown moved from %d/%d to %d/%d", before.AttackPhase, before.AttackCountdown, after.AttackPhase, after.AttackCountdown)
	}
	if victimHP(w) != 100 {
		t.Fatal("an off-map attacker struck")
	}
}
