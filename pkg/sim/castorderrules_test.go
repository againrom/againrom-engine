package sim

import "testing"

// areaCloud is a damaging area row whose landing stays on the map as a cloud,
// so a test can read the cell it landed on.
func areaCloud() SpellRule {
	return SpellRule{ID: 7, ManaCost: 3, School: 1, MaxRange: 7, Area: true, Distribution: distributionDiamond,
		AreaDuration: 1, DamageMin: 1, DamageMax: 1, Damaging: true, TargetsUnit: true}
}

// nextCloud steps w until one more cloud of spell id lands, for at most n
// ticks, and reports the cell it landed on.
func nextCloud(w *World, id uint16, n int) (x, y int32, ok bool) {
	count := func() (k int, last CellEffect) {
		for _, c := range w.CellEffects() {
			if c.Spell == id {
				k, last = k+1, c
			}
		}
		return k, last
	}
	before, _ := count()
	for range n {
		Step(w, nil)
		if k, last := count(); k > before {
			return last.X, last.Y, true
		}
	}
	return 0, 0, false
}

// An area row a computer mage casts at a unit lands where the unit stands at
// the release, and each repeat of the retained order follows it (MAGIC-428).
// Before, the cast landed on the cell the unit stood on when the order was
// written.
func TestAnAreaCastAtAUnitLandsWhereTheUnitStandsAtTheRelease(t *testing.T) {
	w := engageMageWorld(t, 1<<7, 100, areaCloud())
	w.entities[0].Mind = 59
	if !w.orderAttack(0, 2) {
		t.Fatal("the engage refused")
	}
	if k, ok := w.bookCastIndex(1); !ok || w.bookCasts[k].Phase != bookCharging || w.bookCasts[k].Target != 2 {
		t.Fatalf("the mage holds %+v, want a wind-up at unit 2", w.bookCasts)
	}
	vi := indexOfEntity(w.entities, 2)
	w.entities[vi].X, w.entities[vi].Y = 4, 10
	if x, y, ok := nextCloud(w, 7, 60); !ok || x != 4 || y != 10 {
		t.Fatalf("the release landed at %d,%d (%v), want the unit's cell 4,10", x, y, ok)
	}
	w.entities[vi].X, w.entities[vi].Y = 5, 6
	if x, y, ok := nextCloud(w, 7, 60); !ok || x != 5 || y != 6 {
		t.Fatalf("the repeat landed at %d,%d (%v), want the unit's new cell 5,6", x, y, ok)
	}
}

// A retained order that waits between casts while its caster still crosses
// into a cell begins its next wind-up at the cell centre: the repeat does not
// cut the step (MAGIC-429). Before, the wind-up began at once and the
// crossing stood still under it.
func TestARetainedRepeatWaitsForTheCellCentre(t *testing.T) {
	w := engageMageWorld(t, 1<<1, 100, hlArrow())
	w.entities[0].Transit, w.entities[0].TransitTotal = 4, 8
	if !w.queueBookCast(bookCast{Caster: 1, Target: 2, Spell: 1, X: 3, Y: 8, Retained: true, Phase: bookPending}) {
		t.Fatal("fixture: the retained order was not queued")
	}
	for owed := spAt(t, w, 1).Transit; owed > 0; owed = spAt(t, w, 1).Transit {
		Step(w, nil)
		k, ok := w.bookCastIndex(1)
		if !ok {
			t.Fatal("the retained order was dropped during the crossing")
		}
		if w.bookCasts[k].Phase != bookPending && spAt(t, w, 1).Transit > 0 {
			t.Fatalf("the repeat began with %d crossing ticks owed, phase %d", spAt(t, w, 1).Transit, w.bookCasts[k].Phase)
		}
		if spAt(t, w, 1).Transit >= owed {
			t.Fatalf("the crossing stood still at %d ticks owed", owed)
		}
	}
	Step(w, nil)
	if k, ok := w.bookCastIndex(1); !ok || w.bookCasts[k].Phase != bookCharging {
		t.Fatalf("at the cell centre the repeat did not begin: %+v", w.bookCasts)
	}
}

// The withdrawal's move replaces a retained cast order: a mage at its Wimpy
// threshold keeps the rest of its recovery and casts no more (AI-763). Before,
// the retained order outlived the withdrawal and cast again.
func TestAWithdrawalEndsTheRetainedCastOrder(t *testing.T) {
	other := hlArrow()
	other.ID = 3
	w, remaining := mageInRecovery(t, other)
	w.entities[0].Wimpy = w.entities[0].HP
	w.withdrawalActor(0, nil)
	if _, ok := w.bookCastIndex(1); ok {
		t.Fatalf("the retained order outlived the withdrawal: %+v", w.bookCasts)
	}
	if got := w.entities[0].CastWait; got != remaining {
		t.Fatalf("the withdrawal left a cast wait of %d, want the recovery's %d", got, remaining)
	}
}

// A cast drawn during a strike begins on the tick the same strike would load
// its next charge: the executor reads the drawn order at the first
// progress-0 entry after the recovery and its two boundary turns (AI-762,
// HERO-CADENCE-112).
func TestADrawnCastBeginsWhenTheStrikeWouldLoadAgain(t *testing.T) {
	next := func(draw bool) int {
		w := engageMageWorld(t, 0, 100, hlArrow())
		w.entities[0].Mind = 0
		loadStrike(t, w, 2)
		if draw {
			w.entities[0].KnownSpells = 1 << 1
			if !w.orderAttack(0, 2) {
				t.Fatal("the draw during the strike wrote no order")
			}
		}
		// Ticks are counted from the recovery's end, the first boundary turn:
		// the two worlds roll different jitters.
		ended := -1
		for n := 1; n <= 200; n++ {
			before := w.entities[0].AttackPhase
			Step(w, nil)
			if ended < 0 && w.entities[0].AttackPhase == AttackBoundaryOne {
				ended = n
			}
			if k, ok := w.bookCastIndex(1); draw && ok && w.bookCasts[k].Phase == bookCharging {
				return n - ended
			}
			if !draw && before != AttackCharging && w.entities[0].AttackPhase == AttackCharging {
				return n - ended
			}
		}
		t.Fatalf("draw %v: nothing began in 200 ticks", draw)
		return 0
	}
	if cast, strike := next(true), next(false); cast != strike {
		t.Fatalf("the drawn cast began %d ticks after the recovery, the next strike loads after %d", cast, strike)
	}
}
