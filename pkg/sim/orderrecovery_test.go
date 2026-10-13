package sim

import "testing"

// A move order given during a cast's recovery waits for the recovery's end,
// then walks; an armed automatic row with a target in range does not keep
// the mage standing.
func TestAMoveOrderInTheRecoveryWaitsThenWalks(t *testing.T) {
	for _, armed := range []bool{false, true} {
		w := osWorld(t, [2]int32{8, 4})
		if armed {
			w.entities[0].AutoSpell = 1
		}
		Step(w, []Command{Cast(osMage, osEnemy, 1)})
		for k := 0; k < 60 && w.entities[0].CastWait == 0; k++ {
			Step(w, nil)
		}
		if w.entities[0].CastWait == 0 {
			t.Fatal("fixture: the cast never reached its recovery")
		}
		to := CellPoint{X: 4, Y: 20}
		Step(w, []Command{MoveTo(osMage, to)})
		for w.entities[0].CastWait != 0 {
			if e := w.entities[0]; e.X != 4 || e.Y != 4 || e.Transit != 0 {
				t.Fatalf("armed %v: the mage stepped during its recovery: %d,%d transit %d", armed, e.X, e.Y, e.Transit)
			}
			Step(w, nil)
		}
		for k := 0; k < 400 && (w.entities[0].X != to.X || w.entities[0].Y != to.Y); k++ {
			Step(w, nil)
		}
		if e := w.entities[0]; e.X != to.X || e.Y != to.Y {
			t.Fatalf("armed %v: the mage stands at %d,%d after its recovery, ordered to %v", armed, e.X, e.Y, to)
		}
	}
}
