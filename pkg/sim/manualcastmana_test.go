package sim

import "testing"

func replacementManaWorld(t *testing.T) *World {
	t.Helper()
	caster := effectMage(1, 1, 1, 1<<1|1<<3)
	caster.Mana, caster.MaxMana, caster.AttackCharge = 100, 100, 8
	caster.Mind = 0
	target := spEnt(2, 5, 1)
	target.Owner, target.HP, target.MaxHP = 2, 1000, 1000
	w := spWorld(t, 42, []SpellRule{
		{ID: 1, ManaCost: 10, School: 1, TargetsUnit: true, Damaging: true, MaxRange: 15, DamageMin: 5, DamageMax: 5, Delivery: 2, EffectSpeed: 128},
		{ID: 3, ManaCost: 10, School: 1, TargetsUnit: true, Damaging: true, MaxRange: 15, DamageMin: 5, DamageMax: 5, Delivery: 2, EffectSpeed: 128},
	}, caster, target)
	w.relations.turnHostile(SelfSlot, 2)
	return w
}

// Once the mana is spent the cast completes: another cast clicked during the
// paid wind-up waits as the next order and runs after the release.
func TestManualCastDuringAPaidWindUpRunsAfterTheRelease(t *testing.T) {
	w := replacementManaWorld(t)
	Step(w, []Command{spCast(1, 2, 1)})
	before := w.bookCasts[0]
	Step(w, []Command{spCast(1, 2, 3)})
	c := w.bookCasts[0]
	if c.Spell != 1 || !c.Paid || c.Remaining+1 != before.Remaining || w.entities[0].Mana != 90 ||
		w.entities[0].PendingOrder.Kind != PendingActorCast || w.entities[0].PendingOrder.Spell != 3 {
		t.Fatalf("queued click changed the wind-up: %+v pending %+v mana %d", c, w.entities[0].PendingOrder, w.entities[0].Mana)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 64 && (len(w.bookCasts) == 0 || w.bookCasts[0].Spell != 3); tick++ {
		Step(w, nil)
		Step(&back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("LOAD with a queued cast diverged", tick)
		}
	}
	if len(w.bookCasts) != 1 || w.bookCasts[0].Spell != 3 || w.entities[0].Mana != 80 || w.entities[0].PendingOrder.Kind != PendingNone {
		t.Fatalf("second order: casts %+v mana %d pending %+v", w.bookCasts, w.entities[0].Mana, w.entities[0].PendingOrder)
	}
}

// The same click repeated on a charging cast leaves the record, its payment
// and its countdown as one click left them.
func TestRepeatedManualCastLeavesTheChargingRecord(t *testing.T) {
	w := replacementManaWorld(t)
	Step(w, []Command{spCast(1, 2, 1)})
	before := w.bookCasts[0]
	Step(w, []Command{spCast(1, 2, 1)})
	after := w.bookCasts[0]
	if !after.Paid || after.Phase != bookCharging || after.Remaining+1 != before.Remaining || w.entities[0].Mana != 90 {
		t.Fatalf("before %+v after %+v mana %d", before, after, w.entities[0].Mana)
	}
}

// A rule that pays at release: clicks never delay the wind-up, and the same
// click on the release tick lets the cast release and spend once. Arriving
// after that release, it starts the next animation as a recovery click does.
func TestRepeatedManualCastOnTheReleaseTickReleasesOnce(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	w := spWorld(t, 42, []SpellRule{spell}, spMage(1, 0, 0, 60, 50, 50, 1<<1), spEnt(2, 3, 0))
	Step(w, []Command{spCast(1, 2, 1)})
	for range 255 {
		if len(w.bookCasts) == 0 || w.bookCasts[0].Remaining <= 1 {
			break
		}
		Step(w, []Command{spCast(1, 2, 1)})
	}
	if len(w.bookCasts) != 1 || w.bookCasts[0].Remaining != 1 || w.entities[0].Mana != 50 {
		t.Fatalf("repeated clicks changed the wind-up: %+v mana %d", w.bookCasts, w.entities[0].Mana)
	}
	events := StepObserved(w, []Command{spCast(1, 2, 1)})
	if len(events) != 1 || w.entities[0].Mana != 45 || len(w.bookCasts) != 1 || w.bookCasts[0].Remaining != castWindupTicks(w.entities[0]) {
		t.Fatalf("release tick: %d events, mana %d, casts %+v", len(events), w.entities[0].Mana, w.bookCasts)
	}
}

// An attack order during the paid wind-up neither cancels nor restarts the
// cast: it releases on its own clock, and the victim is held afterwards. An
// attack order is the newest order, so it drops a cast queued before it.
func TestAttackOrderDuringAPaidWindUpLetsTheCastRelease(t *testing.T) {
	w := replacementManaWorld(t)
	Step(w, []Command{spCast(1, 2, 1)})
	remaining := w.bookCasts[0].Remaining
	Step(w, []Command{spCast(1, 2, 3)})
	Step(w, []Command{Attack(1, 2)})
	if len(w.bookCasts) != 1 || w.bookCasts[0].Spell != 1 || w.bookCasts[0].Remaining+2 != remaining || w.entities[0].Mana != 90 {
		t.Fatalf("attack order changed the wind-up: %+v mana %d", w.bookCasts, w.entities[0].Mana)
	}
	if w.entities[0].PendingOrder.Kind != PendingNone || !w.entities[0].HasAttackTarget {
		t.Fatalf("attack order: pending %+v victim %v", w.entities[0].PendingOrder, w.entities[0].HasAttackTarget)
	}
	for tick := 0; tick < 64 && w.PendingSpellDeliveries() == 0; tick++ {
		Step(w, nil)
	}
	if w.PendingSpellDeliveries() == 0 || w.entities[0].Mana != 90 {
		t.Fatalf("the cast did not release once: mana %d", w.entities[0].Mana)
	}
}
