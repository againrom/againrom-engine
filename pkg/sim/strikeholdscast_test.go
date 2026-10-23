package sim

import "testing"

// loadStrike gives caster 1 a plain attack on victim and steps until its
// strike cycle is loaded.
func loadStrike(t *testing.T, w *World, victim EntityID) {
	t.Helper()
	ci := indexOfEntity(w.entities, 1)
	if !w.orderAttack(ci, victim) || !w.entities[ci].HasAttackTarget {
		t.Fatal("the plain engage refused")
	}
	for n := 0; w.entities[ci].AttackPhase == AttackReady && w.entities[ci].AttackCountdown == 0; n++ {
		if n > 64 {
			t.Fatal("the strike never loaded")
		}
		Step(w, nil)
	}
}

// drawDuringStrike runs the engage draw for caster 1 while its strike is
// loaded and checks that the drawn cast waits for the strike's end and then
// begins (AI-554, MAGIC-239, AI-356).
func drawDuringStrike(t *testing.T, w *World, victim EntityID, spell uint16) {
	t.Helper()
	ci := indexOfEntity(w.entities, 1)
	before := w.entities[ci]
	if !w.orderAttack(ci, victim) {
		t.Fatal("the draw during the strike wrote no order")
	}
	k, ok := w.bookCastIndex(1)
	if !ok || w.bookCasts[k].Phase != bookApproach || !w.bookCasts[k].Retained ||
		w.bookCasts[k].Spell != spell {
		t.Fatalf("the draw during the strike left %+v, want a retained spell %d order", w.bookCasts, spell)
	}
	if e := w.entities[ci]; e.AttackPhase != before.AttackPhase || e.AttackCountdown != before.AttackCountdown {
		t.Fatal("the draw cut the loaded strike")
	}
	advanced, ready := false, false
	for range 200 {
		e := w.entities[ci]
		Step(w, nil)
		k, ok := w.bookCastIndex(1)
		if !ok {
			t.Fatal("the waiting cast order was dropped")
		}
		switch w.bookCasts[k].Phase {
		case bookApproach:
			now := w.entities[ci]
			advanced = advanced || now.AttackPhase != e.AttackPhase || now.AttackCountdown != e.AttackCountdown
			ready = now.AttackPhase == AttackReady && now.AttackCountdown == 0
		case bookCharging:
			if !advanced {
				t.Fatal("the strike stood still while the cast waited")
			}
			if !ready {
				t.Fatal("the cast began before the strike reached its end")
			}
			return
		default:
			t.Fatalf("the waiting cast order moved to phase %d", w.bookCasts[k].Phase)
		}
	}
	t.Fatal("the cast drawn during the strike never began")
}

func TestAComputerMageCastDrawnDuringAStrikeWaitsForItsEnd(t *testing.T) {
	w := engageMageWorld(t, 0, 100, hlArrow())
	w.entities[0].Mind = 0
	loadStrike(t, w, 2)
	w.entities[0].KnownSpells = 1 << 1
	drawDuringStrike(t, w, 2, 1)
}

func TestACreatureCastDrawnDuringAStrikeWaitsForItsEnd(t *testing.T) {
	w := creatureWorld(t, []SpellRule{hlArrow()}, [CreatureSpellSlots]CreatureSpell{}, 1<<1, creatureArrowBook())
	vi := indexOfEntity(w.entities, 2)
	w.entities[vi].X = 6
	loadStrike(t, w, 2)
	w.entities[indexOfEntity(w.entities, 1)].CreatureSpells = creatureCertain
	drawDuringStrike(t, w, 2, 1)
}

// The high-Mind hand-back writes the plain engage over a waiting cast order
// (AI-554, AI-376).
func TestAMageHandBackEndsItsWaitingCastOrder(t *testing.T) {
	ended, kept := 0, 0
	for seed := uint64(1); seed <= 40; seed++ {
		w := engageMageWorld(t, 0, 100, hlArrow())
		loadStrike(t, w, 2)
		w.entities[0].KnownSpells = 1 << 1
		w.entities[0].Mind = 0
		if !w.orderAttack(0, 2) {
			t.Fatal("the draw during the strike wrote no order")
		}
		w.entities[0].Mind = 99
		w.rng = rng{state: seed}
		w.orderAttack(0, 2)
		if _, ok := w.bookCastIndex(1); ok {
			kept++
		} else {
			ended++
		}
	}
	if ended == 0 || kept == 0 {
		t.Fatalf("over 40 seeds %d waiting orders ended and %d stayed", ended, kept)
	}
}
