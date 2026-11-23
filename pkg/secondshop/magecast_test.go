package secondshop

import (
	"testing"

	"againrom/pkg/shopstock"
)

type zeroDraws struct{}

func (zeroDraws) Intn(int) int { return 0 }

// TestSecondShopMageWeightedCastKeepsTenTimesThePrice pins the second game's
// weighted cast budget, ten times the price on a mage's item too (DIV-4329):
// at price 10 and spell base 50 the budget 100 leaves no power, so the first
// effect fails. The first game's hundred times (SHOP-156) would give powers
// 1..5.
func TestSecondShopMageWeightedCastKeepsTenTimesThePrice(t *testing.T) {
	spells := map[int][]int32{}
	for _, id := range castSpells[0] {
		p := make([]int32, 22)
		p[colEnchant] = 50
		spells[int(id)] = p
	}
	tb := Tables{Magic: fixtureRows{n: 50, params: map[int][]int32{tagCastSpell: effectRow(1, 1, 1, 100, nil)}},
		Spells: fixtureRows{n: 30, params: spells}}
	base := shopstock.Base{Price: 10, Capacity: 100, Ceiling: 100000, Slot: 3}
	if effects, _, ok := tb.enchant().Apply(base, zeroDraws{}); ok {
		t.Fatalf("mage's item: weighted cast at ten times the price gave %+v, want no effect", effects)
	}
}
