package sim

import "testing"

// mageInRecovery holds a mage's retained spell-1 order in its recovery.
func mageInRecovery(t *testing.T, other SpellRule) (*World, uint8) {
	t.Helper()
	w := engageMageWorld(t, 1<<1|1<<other.ID, 100, hlArrow(), other)
	w.entities[0].KnownSpells = 1 << 1
	if !w.orderAttack(0, 2) {
		t.Fatal("the engage refused")
	}
	for range 400 {
		Step(w, nil)
		if k, ok := w.bookCastIndex(1); ok && w.bookCasts[k].Phase == bookRelaxing && w.bookCasts[k].Retained {
			if w.bookCasts[k].Spell != 1 {
				t.Fatalf("the retained order holds spell %d, want 1", w.bookCasts[k].Spell)
			}
			return w, w.bookCasts[k].Remaining
		}
	}
	t.Fatal("the mage's cast never reached a retained recovery")
	return nil, 0
}

// A mage's draw with a retained cast pending rewrites or ends it (AI-554, AI-376).
func TestAComputerMageDrawReplacesItsRetainedCastOrder(t *testing.T) {
	other := hlArrow()
	other.ID = 3
	rewritten, ended := 0, 0
	for seed := uint64(1); seed <= 60; seed++ {
		w, remaining := mageInRecovery(t, other)
		w.entities[0].KnownSpells = 1 << 3
		w.entities[0].Mind = 60
		w.rng = rng{state: seed}
		if w.orderAttack(0, 3) {
			t.Fatalf("seed %d: an order was written during recovery", seed)
		}
		k, ok := w.bookCastIndex(1)
		switch {
		case !ok:
			if got := w.entities[0].CastWait; got != remaining {
				t.Fatalf("seed %d: the hand-back left a cast wait of %d, want %d", seed, got, remaining)
			}
			ended++
		case w.bookCasts[k].Spell == 3 && w.bookCasts[k].Target == 3 && !w.bookCasts[k].AtCell && w.bookCasts[k].Retained &&
			w.entities[0].AdmittedBookSpell == 3:
			rewritten++
		default:
			t.Fatalf("seed %d: the retained order is %+v, want spell 3 at victim 3 or no order", seed, w.bookCasts[k])
		}
	}
	if rewritten == 0 || ended == 0 {
		t.Fatalf("over 60 seeds %d retained orders were rewritten and %d ended", rewritten, ended)
	}
}

// A mage's area cast armed out of reach lands at its victim's cell, not (0,0).
func TestAComputerMageArmsAnAreaCastAtItsVictimsCell(t *testing.T) {
	area := SpellRule{ID: 2, ManaCost: 3, School: 1, MaxRange: 7, Area: true, Distribution: 3, Radius: 1}
	w := engageMageWorld(t, 1<<2, 100, area)
	if !w.orderAttack(0, 3) {
		t.Fatal("the engage refused")
	}
	k, ok := w.bookCastIndex(1)
	if !ok || w.bookCasts[k].Phase != bookApproach || w.bookCasts[k].AtCell {
		t.Fatalf("the mage armed %+v, want an approach at victim 3", w.bookCasts)
	}
	if c := w.bookCasts[k]; c.X != 13 || c.Y != 8 {
		t.Fatalf("the armed area cast aims at %d,%d, want the victim's cell 13,8", c.X, c.Y)
	}
}
