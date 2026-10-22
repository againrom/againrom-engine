package sim

import "testing"

// A computer mage whose engage draws a cast in reach while it is still
// crossing into a cell finishes the crossing first and starts the cast at the
// cell centre: the executor runs a step in progress to the centre before it
// reads the cast order (AI-350, AI-352, MAGIC-239). Before, the cast began at
// once and owned the body, so the crossing stood still through the wind-up and
// the recovery while the cast was drawn.
func TestAMageCrossingIntoACellCastsAtItsCentre(t *testing.T) {
	w := engageMageWorld(t, 1<<1, 100, hlArrow())
	w.entities[0].Transit, w.entities[0].TransitTotal = 4, 8
	if !w.orderAttack(0, 2) {
		t.Fatal("the engage refused")
	}
	k, ok := w.bookCastIndex(1)
	if !ok || w.bookCasts[k].Phase != bookApproach || w.bookCasts[k].Target != 2 {
		t.Fatalf("a mage mid-crossing began its cast at once: casts %+v", w.bookCasts)
	}
	for want := uint16(3); ; want-- {
		StepObserved(w, nil)
		mage := spAt(t, w, 1)
		k, ok = w.bookCastIndex(1)
		if !ok {
			t.Fatal("the armed cast was dropped during the crossing")
		}
		if mage.Transit != want {
			t.Fatalf("crossing owes %d ticks, want %d (cast phase %d)", mage.Transit, want, w.bookCasts[k].Phase)
		}
		if want == 0 {
			if w.bookCasts[k].Phase != bookApproach {
				t.Fatalf("the cast started on the tick the crossing ended, phase %d", w.bookCasts[k].Phase)
			}
			break
		}
		if w.bookCasts[k].Phase != bookApproach {
			t.Fatalf("the cast started mid-crossing with %d ticks owed", mage.Transit)
		}
	}
	StepObserved(w, nil)
	k, ok = w.bookCastIndex(1)
	if mage := spAt(t, w, 1); !ok || w.bookCasts[k].Phase != bookCharging || mage.X != 2 || mage.Y != 8 || mage.HasTarget {
		t.Fatalf("at the cell centre: casts %+v, mage at %d,%d walking %v", w.bookCasts, mage.X, mage.Y, mage.HasTarget)
	}
	if ev, ok := runCast(w, 60); !ok || ev.Target != 2 {
		t.Fatalf("the cast never released: %+v", ev)
	}
}

// No claim states the second game's executor, so a second-game mage still
// begins the cast mid-crossing.
func TestASecondGameMageCrossingIntoACellCastsAtOnce(t *testing.T) {
	w := engageMageWorld(t, 1<<1, 100, hlArrow())
	w.rom2 = &rom2ScriptState{}
	w.entities[0].Transit, w.entities[0].TransitTotal = 4, 8
	if !w.orderAttack(0, 2) {
		t.Fatal("the engage refused")
	}
	if k, ok := w.bookCastIndex(1); !ok || w.bookCasts[k].Phase != bookCharging {
		t.Fatalf("a second-game mage mid-crossing deferred its cast: casts %+v", w.bookCasts)
	}
}
