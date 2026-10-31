package game

import (
	"math"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/shopstock"
	"againrom/pkg/sim"
)

type shopDrawSource = shopstock.Source

func shopDrawInclusive(r shopDrawSource, lo, hi int) (int, bool) {
	return shopstock.Inclusive(r, lo, hi)
}

type generatedEffect struct {
	effect sim.ItemEffect
	points int32
	cast   int32
}

func generatedOf(g shopstock.EffectDraw, ok bool) (generatedEffect, bool) {
	return generatedEffect{effect: sim.ItemEffect(g.Effect), points: g.Points, cast: g.Cast}, ok
}

// shopCastSpells are the first game's cast spells, a mage's item's and a
// fighter's (SHOP-121).
var shopCastSpells = [2][]uint16{{1, 13, 14, 20, 11}, {20, 11}}

// shopEnchantRules are the first game's enchant rules over t: the Magic and
// Spells tables, its cast spells, the price of the summed effect points, the
// price cap 9,999,999 and the effect-repeat rule repeat.
func shopEnchantRules(t *mapload.Table, repeat shopstock.Repeat) shopstock.Enchant {
	e := shopstock.Enchant{
		CastSpells: shopCastSpells,
		PointsPrice: func(points []int32) int64 {
			sum := int32(0)
			for _, p := range points {
				sum += p
			}
			return int64(shopNonCastPrice(sum))
		},
		PriceCap: 9_999_999,
		Repeat:   repeat,
	}
	if t != nil {
		if t.Magic != nil {
			e.Effects = t.Magic
		}
		if t.Spells != nil {
			e.Spells = t.Spells
		}
	}
	return e
}

// shopEnchantedItem is candidate c enchanted under the first game's
// effect-repeat rule.
func shopEnchantedItem(c data.ShopCandidate, ceiling int32, t *mapload.Table, r shopDrawSource) (ShopItem, bool) {
	return shopEnchant(c, ceiling, t, r, shopEffectRepeat)
}

// shopEnchant is candidate c with its effects by the one enchant: the first,
// then up to two optional ones, joined under the effect-repeat rule repeat.
func shopEnchant(c data.ShopCandidate, ceiling int32, t *mapload.Table, r shopDrawSource, repeat shopstock.Repeat) (ShopItem, bool) {
	if t == nil || t.Magic == nil || r == nil {
		return ShopItem{}, false
	}
	base := shopstock.Base{Price: c.Price, Capacity: c.MagCap, Ceiling: ceiling, Slot: c.EffectSlot, Fighter: c.Fighter, Forced: c.ForcedCast}
	effects, price, ok := shopEnchantRules(t, repeat).Apply(base, r)
	if !ok {
		return ShopItem{}, false
	}
	return shopItemFromInstance(sim.ItemInstance{Code: uint16(c.Code), Kind: c.ItemKind, Price: price, Effects: shopstock.EffectsAs[sim.ItemEffect](effects)}, 1), true
}

func shopOrdinaryEffect(magic data.Collection, kind uint8, budget int64, capacity int32, r shopDrawSource) (generatedEffect, bool) {
	e := shopstock.Enchant{}
	if magic != nil {
		e.Effects = magic
	}
	return generatedOf(e.Ordinary(kind, budget, capacity, r))
}

func shopCastEffect(fighter bool, budget int64, t *mapload.Table, r shopDrawSource) (generatedEffect, bool) {
	return generatedOf(shopEnchantRules(t, shopEffectRepeat).Cast(fighter, budget, 0, r))
}

func shopNonCastPrice(points int32) int32 {
	if points <= 0 {
		return 0
	}
	n := float64(points)
	return int32((math.Pow(1.5, n/70) + 1) * n * 50)
}
