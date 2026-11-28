package sim

import "testing"

func strideLawWorld(t *testing.T, d ScriptDialect, speed int32) *World {
	t.Helper()
	w := mustWorld(t, 1, Bounds{20, 20}, []Entity{{ID: 7, Owner: SelfSlot, X: 8, Y: 8, HP: 100, MaxHP: 100, Speed: speed}})
	if d == ScriptROM2 {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	return w
}

// R2-ENGINE-1213 and R2-ENGINE-1470: a step takes one off x in octant 1 or 5
// when both sub bytes reach 0. A rate-1 diagonal has no displacement.
func TestEveryGameStepsByTheSameStrideLaw(t *testing.T) {
	t.Parallel()
	for _, d := range []ScriptDialect{ScriptROM1, ScriptROM2} {
		// Speed 23 walks north-east at stride 16 for 16 calls; the eighth call
		// leaves both sub bytes 0, so x keeps cell 8 and both axes change cell
		// on the ninth.
		w := strideLawWorld(t, d, 23)
		Step(w, []Command{MoveTo(7, CellPoint{X: 9, Y: 7})})
		for paid := 2; paid <= 9; paid++ {
			Step(w, nil)
			e := w.Entities()[0]
			want := cell{8, 8}
			if paid == 9 {
				want = cell{9, 7}
			}
			if got := w.moverCell(e); int(e.TransitTotal-e.Transit) != paid || got != want {
				t.Fatalf("dialect %v call %d: cell %v of %d/%d, want %v", d, paid, got, e.TransitTotal-e.Transit, e.TransitTotal, want)
			}
		}

		w = strideLawWorld(t, d, 1)
		Step(w, []Command{MoveTo(7, CellPoint{X: 9, Y: 9})})
		e := w.Entities()[0]
		wantTotal := uint16(256)
		if d == ScriptROM2 {
			wantTotal = 0
		}
		if e.Stride.StepX != 0 || e.Stride.StepY != 0 || e.TransitTotal != wantTotal || strideFault(e) != nil {
			t.Fatalf("dialect %v rate-1 diagonal: stride %+v over %d calls, fault %v", d, e.Stride, e.TransitTotal, strideFault(e))
		}
	}
}
