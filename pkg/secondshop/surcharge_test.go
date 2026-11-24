package secondshop

import (
	"testing"

	"againrom/pkg/sim"
)

// An equipment price adds one surcharge on the sum of its ordinary effects'
// sizes, not one per effect (R2-ENGINE-1764).
func TestSecondShopPriceAddsOneSurchargeOnTheSummedSize(t *testing.T) {
	tb := enchantTables()
	effect := func(n int) sim.ItemEffect { return sim.ItemEffect{Kind: 5, Operand: uint32(n)} }
	for _, c := range []struct {
		name    string
		effects []sim.ItemEffect
		want    int64
	}{
		{"one of 70", []sim.ItemEffect{effect(70)}, 7000},
		{"two of 70", []sim.ItemEffect{effect(70), effect(70)}, 26798},
		{"two of 35", []sim.ItemEffect{effect(35), effect(35)}, 7000},
	} {
		if got := tb.effectsPrice(c.effects); got != c.want {
			t.Errorf("%s: effects price %d, want %d", c.name, got, c.want)
		}
	}
	it := Item{Table: Weapon, Row: 1, Effects: []sim.ItemEffect{effect(1 << 30), effect(1 << 30)}}
	if got := tb.Price(it); got != priceCap {
		t.Errorf("huge sum: price %d, want the cap %d", got, priceCap)
	}
}
