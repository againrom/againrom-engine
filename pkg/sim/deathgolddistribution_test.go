package sim

import "testing"

// A unit whose Units row pays 24200 + U[0,36300] at chance 90 draws gold at
// 90/101 of its deaths, uniformly over the inclusive range, from the world's
// own seeded generator.
func TestDeathGoldDrawsTheRowLawFromTheSeededGenerator(t *testing.T) {
	const draws = 20000
	w, err := NewLootWorld(7, ddBounds, ModeCanonical, Terrain{}, []Entity{{
		ID: 1, X: 4, Y: 5, TypeID: 0x44, GoldChance: 90, TreasureMin: 24200, TreasureMax: 36300,
	}}, nil, Relations{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	paid, zero := 0, 0
	var sum float64
	low, high := uint32(1<<31), uint32(0)
	for i := 0; i < draws; i++ {
		g := w.deathGold(0)
		if g == 0 {
			zero++
			continue
		}
		paid++
		sum += float64(g)
		if g < low {
			low = g
		}
		if g > high {
			high = g
		}
	}
	if low < 24200 || high > 60500 {
		t.Fatalf("paid range %d..%d, want within 24200..60500", low, high)
	}
	if low > 25500 || high < 59000 {
		t.Fatalf("paid range %d..%d does not span the row's range", low, high)
	}
	if frac := float64(paid) / draws; frac < 0.88 || frac > 0.90 {
		t.Fatalf("paid fraction %.4f, want 90/101 = 0.8911", frac)
	}
	if mean := sum / float64(paid); mean < 41700 || mean > 43000 {
		t.Fatalf("mean payout %.0f, want 24200+36300/2 = 42350", mean)
	}
	if zero == 0 {
		t.Fatal("no death paid nothing")
	}
}
