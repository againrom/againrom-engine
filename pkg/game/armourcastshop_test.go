package game

import (
	"math"
	"testing"

	"againrom/pkg/mapload"
)

// A forced cast of spell 1 at power 1 on a candidate of price 167. A weapon
// adds the cast's own price; an armour or a shield sums the power times Magic
// row 41's cost 1 into its points and pays the one surcharge (SHOP-158).
func TestShopArmourCastPaysTheSurchargeOnItsPower(t *testing.T) {
	table := &mapload.Table{Magic: enchantMagic{41: {cost: 1}}, Spells: enchantSpells{1: 50, 13: 1000}}
	for _, c := range []struct {
		name string
		kind uint8
		want int32
	}{
		{"weapon", 2, 733},
		{"armour", 1, 167 + int32((math.Pow(1.5, 1.0/70)+1)*50)},
	} {
		candidate := ordinaryCandidate()
		candidate.Price, candidate.ForcedCast, candidate.ItemKind = 167, true, c.kind
		item, ok := shopEnchantedItem(candidate, 1000, table, &fixedDraws{values: []int{1, 0, 0}})
		spell, power, cast := item.Instance().CastSpell()
		if !ok || !cast || spell != 1 || power != 1 || item.Price != c.want {
			t.Errorf("%s: ok=%v cast=%v spell=%d power=%d price=%d, want price %d",
				c.name, ok, cast, spell, power, item.Price, c.want)
		}
	}
}
