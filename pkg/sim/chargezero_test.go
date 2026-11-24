package sim

import "testing"

// A charge byte of zero lasts 256 steps in the second game and one step in
// the first (R2-ENGINE-1763); the second game's blow falls one advance later.
func TestChargeByteOfZeroLastsTheGamesOwnCount(t *testing.T) {
	for _, tc := range []struct {
		name   string
		second bool
		charge int32
		want   int
	}{
		{"first game zero", false, 0, 1},
		{"second game zero", true, 0, 257},
		{"second game one", true, 1, 2},
		{"second game five", true, 5, 6},
	} {
		fighter := asFighter(asHero, SelfSlot, 30, 40)
		w := asWorld(t, tc.second, fighter, asFoe(asFoeA, 31, 40))
		w.entities[0].AttackCharge = tc.charge
		Step(w, []Command{Attack(asHero, asFoeA)})
		first := 0
		if w.entities[1].HP != 1000 {
			first = 1
		}
		for k := 2; first == 0 && k <= 300; k++ {
			Step(w, nil)
			if w.entities[1].HP != 1000 {
				first = k
			}
		}
		if first != tc.want {
			t.Errorf("%s: first blow on advance %d, want %d", tc.name, first, tc.want)
		}
	}
}
