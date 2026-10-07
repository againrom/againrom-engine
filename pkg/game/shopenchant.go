package game

import (
	"math"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type shopDrawSource interface {
	Intn(int) int
}

func shopDrawInclusive(r shopDrawSource, lo, hi int) (int, bool) {
	if r == nil || hi < lo {
		return 0, false
	}
	return lo + r.Intn(hi-lo+1), true
}

type generatedEffect struct {
	effect sim.ItemEffect
	points int32
	cast   int32
}

func shopEnchantedItem(c data.ShopCandidate, ceiling int32, t *mapload.Table, r shopDrawSource) (ShopItem, bool) {
	if t == nil || t.Magic == nil || r == nil {
		return ShopItem{}, false
	}
	item := sim.ItemInstance{Code: uint16(c.Code), Kind: c.ItemKind, Price: c.Price}
	remaining := c.MagCap
	nonCastPoints := int32(0)
	var castPrice int32

	appendEffect := func(g generatedEffect) {
		item.Effects = append(item.Effects, g.effect)
		nonCastPoints += g.points
		castPrice += g.cast
		price := int64(c.Price) + int64(shopNonCastPrice(nonCastPoints)) + int64(castPrice)
		if price >= 9_999_999 {
			item.Price = 9_999_999
		} else {
			item.Price = int32(price)
		}
	}

	construct := func(forced bool) (generatedEffect, bool) {
		budget := int64(2)*int64(ceiling) - int64(item.Price)
		if forced {
			capBudget := int64(item.Price) * 100
			if capBudget < budget {
				budget = capBudget
			}
			for attempt := 1; attempt <= 100; attempt++ {
				g, ok := shopCastEffect(false, budget, t, r)
				if ok {
					return g, attempt < 100
				}
			}
			return generatedEffect{}, false
		}
		kind, ok := shopSelectEffectKind(t.Magic, c.Fighter, c.EffectSlot, r)
		if !ok {
			return generatedEffect{}, false
		}
		if kind == 41 {
			// The weighted fighter route substitutes price*10 for the ordinary
			// interaction budget.
			return shopCastEffect(c.Fighter, int64(item.Price)*10, t, r)
		}
		return shopOrdinaryEffect(t.Magic, kind, budget, remaining, r)
	}

	first, ok := construct(c.ForcedCast)
	if !ok {
		return ShopItem{}, false
	}
	appendEffect(first)
	if first.effect.Kind == 41 {
		return shopItemFromInstance(item, 1), true
	}
	remaining -= first.points

	for _, threshold := range []int{50, 25} {
		gate, _ := shopDrawInclusive(r, 0, 100)
		if gate >= threshold {
			continue
		}
		g, ok := construct(false)
		if !ok {
			return shopItemFromInstance(item, 1), true
		}
		if g.effect.Kind == 41 {
			// An optional cast is constructed and discarded, then ends the
			// orchestrator before the capacity debit.
			return shopItemFromInstance(item, 1), true
		}
		appendEffect(g)
		remaining -= g.points
	}
	return shopItemFromInstance(item, 1), true
}

func shopSelectEffectKind(magic data.Collection, fighter bool, slot int, r shopDrawSource) (uint8, bool) {
	if magic == nil || slot < 1 || slot > 12 {
		return 0, false
	}
	column := slot - 1
	if !fighter {
		column += 12
	}
	final := int32(0)
	for kind := 1; kind < magic.Len(); kind++ {
		p := magic.EntryParams(kind)
		if len(p) > 4+column {
			final = p[4+column]
		}
	}
	if final <= 0 {
		return 0, false
	}
	for attempt := 1; attempt < 100; attempt++ {
		draw, _ := shopDrawInclusive(r, 1, int(final))
		previous := int32(0)
		for kind := 1; kind < magic.Len(); kind++ {
			p := magic.EntryParams(kind)
			endpoint := previous
			if len(p) > 4+column {
				endpoint = p[4+column]
			}
			if int32(draw) > previous && int32(draw) <= endpoint {
				return uint8(kind), true
			}
			previous = endpoint
		}
	}
	return 0, false
}

func shopOrdinaryEffect(magic data.Collection, kind uint8, budget int64, capacity int32, r shopDrawSource) (generatedEffect, bool) {
	if magic == nil || int(kind) >= magic.Len() || budget <= 0 || capacity <= 0 {
		return generatedEffect{}, false
	}
	p := magic.EntryParams(int(kind))
	if len(p) < 3 || p[0] <= 0 {
		return generatedEffect{}, false
	}
	cost, minimum, maximum := p[0], p[1], p[2]
	capacityMax := capacity / cost
	x := float64(budget)/float64(50*int64(capacity)) - 1
	if x <= 0 {
		return generatedEffect{}, false
	}
	budgetMax := int32((70 / float64(cost)) * (math.Log(x) / math.Log(1.5)))
	payloadMax := capacityMax
	if budgetMax < payloadMax {
		payloadMax = budgetMax
	}
	if maximum < payloadMax {
		payloadMax = maximum
	}
	if payloadMax < minimum || payloadMax < 1 {
		return generatedEffect{}, false
	}
	draw, _ := shopDrawInclusive(r, 1, int(payloadMax))
	payload := int32(draw)
	if payload < minimum {
		payload = minimum
	}
	if kind >= 44 && kind <= 48 {
		max := payloadMax
		if max > 255 {
			max = 255
		}
		base, _ := shopDrawInclusive(r, 1, int(max))
		spread, ok := shopDrawInclusive(r, 1, int(max/2))
		if !ok {
			return generatedEffect{}, false
		}
		points := int32(base+spread) * cost
		return generatedEffect{effect: sim.ItemEffect{Kind: kind,
			Operand: uint32(uint8(base)) | uint32(uint8(spread))<<8}, points: points}, true
	}
	return generatedEffect{effect: sim.ItemEffect{Kind: kind, Operand: uint32(payload)}, points: payload * cost}, true
}

func shopCastEffect(fighter bool, budget int64, t *mapload.Table, r shopDrawSource) (generatedEffect, bool) {
	ids := []uint16{1, 13, 14, 20, 11}
	if fighter {
		ids = []uint16{20, 11}
	}
	at, _ := shopDrawInclusive(r, 0, len(ids)-1)
	id := ids[at]
	if t == nil || t.Spells == nil || int(id) >= t.Spells.Len() {
		return generatedEffect{}, false
	}
	p := t.Spells.EntryParams(int(id))
	if len(p) <= 20 || p[20] <= 0 {
		return generatedEffect{}, false
	}
	scalar := p[20]
	ratio := float64(budget) / float64(10*int64(scalar))
	if ratio <= 1 {
		return generatedEffect{}, false
	}
	rank := math.Log2(ratio)
	powerMax := int32(30 * (math.Pow(1.2, rank) - 1))
	if powerMax > 100 {
		powerMax = 100
	}
	if powerMax <= 0 {
		return generatedEffect{}, false
	}
	power, _ := shopDrawInclusive(r, 1, int(powerMax))
	cast := int32(10 * float64(scalar) * math.Pow(2, math.Log(float64(power)/30+1)/math.Log(1.2)))
	return generatedEffect{effect: sim.ItemEffect{Kind: 41,
		Operand: uint32(id) | uint32(uint16(int16(power)))<<16}, cast: cast}, true
}

func shopNonCastPrice(points int32) int32 {
	if points <= 0 {
		return 0
	}
	n := float64(points)
	return int32((math.Pow(1.5, n/70) + 1) * n * 50)
}
