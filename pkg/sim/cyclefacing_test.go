package sim

import "testing"

// A cycle ends with the facing it was loaded with (AI-637): from the charge
// load to the return to AttackReady the attacker writes no facing, though its
// victim steps to another side of it. The turn toward the victim's new cell
// comes after the cycle, which is the control that the victim's step would
// turn the attacker at all.
func TestACycleEndsWithTheFacingItWasLoadedWith(t *testing.T) {
	a := cbFighter(1, 5, 5, 6, 4)
	a.RotationSpeed = 32
	v := cbEnt(2, 5, 4)
	v.HP, v.MaxHP = 1_000_000, 1_000_000
	w := cbWorld(t, 3, a, v)
	Step(w, []Command{cbOrder(1, 2)})
	for n := 0; cbAt(t, w, 1).AttackPhase == AttackReady; n++ {
		if n == 40 {
			t.Fatal("fixture: no cycle loaded in 40 ticks")
		}
		Step(w, nil)
	}
	loaded := cbAt(t, w, 1).Facing
	if loaded != 0 {
		t.Fatalf("fixture: loaded facing %#x, want north 0 toward the victim", loaded)
	}
	Step(w, []Command{MoveTo(2, CellPoint{X: 6, Y: 5})})
	stepped := false
	for n := 0; cbAt(t, w, 1).AttackPhase != AttackReady; n++ {
		if n == 40 {
			t.Fatal("the cycle did not return to AttackReady in 40 ticks")
		}
		if e := cbAt(t, w, 1); e.Facing != loaded || e.Turning() {
			t.Fatalf("tick %d of the cycle: facing %#x turning %v, want the loaded %#x", n, e.Facing, e.Turning(), loaded)
		}
		vv := cbAt(t, w, 2)
		stepped = stepped || vv.X == 6 && vv.Y == 5
		Step(w, nil)
	}
	if !stepped {
		t.Fatal("fixture: the victim did not reach the east cell inside the cycle")
	}
	for n := 0; cbAt(t, w, 1).Facing != 64; n++ {
		if n == 40 {
			t.Fatalf("after the cycle the attacker never faced east: facing %#x", cbAt(t, w, 1).Facing)
		}
		Step(w, nil)
	}
}
