package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// TestShopMageWeightedCastTakesAHundredTimesThePrice is SHOP-156: a weighted
// castSpell draw on a mage's item takes the price times 100 as its budget, a
// fighter's item the price times 10. At price 10 and spell base 50 the
// fighter's budget 100 leaves no power, so its first effect fails, and the
// mage's budget 1000 gives powers 1..5.
func TestShopMageWeightedCastTakesAHundredTimesThePrice(t *testing.T) {
	table := &mapload.Table{Magic: enchantMagic{41: {weight: 1}},
		Spells: enchantSpells{1: 50, 11: 50, 13: 50, 14: 50, 20: 50}}
	for _, fighter := range []bool{false, true} {
		candidate := data.ShopCandidate{Code: 0x0301, Price: 10, MagCap: 100, ItemKind: 1, EffectSlot: 3, Fighter: fighter}
		draws := &fixedDraws{values: []int{0, 0, 0}}
		got, ok := shopEnchantedItem(candidate, 100000, table, draws)
		if fighter {
			if ok {
				t.Errorf("fighter's item: weighted cast at ten times the price gave %+v, want no effect", got)
			}
			continue
		}
		if !ok || len(got.Effects) != 1 || got.Effects[0].Kind != 41 || got.Effects[0].Operand != 1|1<<16 {
			t.Fatalf("mage's item: shopEnchantedItem = %+v, %v, want one cast of spell 1 at power 1", got, ok)
		}
		if want := []int{1, 5, 5}; len(draws.calls) != len(want) || draws.calls[1] != 5 || draws.calls[2] != 5 {
			t.Errorf("mage's item: draw calls = %v, want %v (power drawn in 1..5)", draws.calls, want)
		}
	}
}
