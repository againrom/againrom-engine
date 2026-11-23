package sim

import "testing"

// castRecoveryWant is the expected count from the claims' expression: the
// relax, the draw the world's generator will give next, the weapon term and
// the Complication a book cast adds.
func castRecoveryWant(w *World, relax, term, complication int64) (draw, sum int64) {
	probe := w.rng
	draw = int64(probe.uniform(relaxJitter))
	return draw, relax + draw + term + complication
}

func TestCastRecoveryCountPerGameAndSource(t *testing.T) {
	const complication = 20
	rule := SpellRule{ID: 1, Complication: complication}
	cases := []struct {
		name      string
		rule      castRecoveryRule
		relax     int32
		src       castOrigin
		withBook  bool
		wantClamp bool
	}{
		{"first game book", castRecoveryClamped, 4, originBook, true, false},
		{"first game staff", castRecoveryClamped, 4, originStaff, false, false},
		{"first game scroll", castRecoveryClamped, 4, originScroll, false, false},
		{"second game book", castRecoveryLowByte, 4, originBook, false, false},
		{"second game staff", castRecoveryLowByte, 4, originStaff, false, false},
		{"second game scroll", castRecoveryLowByte, 4, originScroll, false, false},
		{"first game book over a byte", castRecoveryClamped, 250, originBook, false, true},
		{"first game scroll over a byte", castRecoveryClamped, 252, originScroll, false, true},
		{"second game book over a byte", castRecoveryLowByte, 250, originBook, false, false},
		{"second game staff over a byte", castRecoveryLowByte, 254, originStaff, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := spEnt(1, 3, 3)
			e.Humanoid, e.Reaction, e.AttackRelax = true, 30, tc.relax
			w := spWorld(t, 7, []SpellRule{rule}, e)
			cadenceEquipWeightedWeapon(t, w, 1, 24)
			w.game.castRecovery = tc.rule
			var comp int64
			if tc.src == originBook {
				comp = complication
			}
			_, sum := castRecoveryWant(w, int64(tc.relax), 2, comp)
			want := sum
			switch {
			case tc.rule == castRecoveryLowByte:
				want = sum & 0xff
			case (tc.src == originBook || tc.src == originScroll) && sum > 255:
				want = 255
			}
			if got := w.castRecoveryTicks(0, tc.src, rule); got != want {
				t.Fatalf("count %d, want %d (sum %d)", got, want, sum)
			}
		})
	}
}

func TestCastRecoveryEstimateDrawsNothing(t *testing.T) {
	e := spEnt(1, 3, 3)
	e.AttackRelax = 300
	w := spWorld(t, 7, nil, e)
	before := w.rng
	if got := w.castRecoveryTicks(0, originEstimate, SpellRule{Complication: 25}); got != 255 {
		t.Fatalf("estimate %d, want 255", got)
	}
	if w.rng != before {
		t.Fatal("the estimate consumed a draw")
	}
}
