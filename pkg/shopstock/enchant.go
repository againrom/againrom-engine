package shopstock

import "math"

// Effect is one item effect as the enchant draws it, field for field the
// simulation's item effect, so a caller converts it.
type Effect struct {
	Kind    uint8
	Mode    uint8
	Operand uint32
}

// EffectsAs is effects as the caller's item effect type, nil for none.
func EffectsAs[T ~struct {
	Kind    uint8
	Mode    uint8
	Operand uint32
}](effects []Effect) []T {
	if effects == nil {
		return nil
	}
	out := make([]T, len(effects))
	for i, x := range effects {
		out[i] = T(x)
	}
	return out
}

// Params is a data.bin table as the enchant reads it: rows of parameters.
type Params interface {
	Len() int
	EntryParams(i int) []int32
}

// Inclusive draws lo..hi on src, or false when src is nil or the range is
// empty.
func Inclusive(src Source, lo, hi int) (int, bool) {
	if src == nil || hi < lo {
		return 0, false
	}
	return lo + src.Intn(hi-lo+1), true
}

// The effect tag that casts a spell, and the tries a cast or a weighted pick
// takes before it fails; a cast found on the last try fails too.
const (
	castSpell = 41
	tries     = 100
)

// Enchant is one game's item-enchant rules. Effects is the effect table:
// row k holds the effect's point cost, minimum, maximum, the UsableBy mask
// and from column 4 twenty-four cumulative weight columns, the fighter's
// slots 1..12 then the others'. Spells is the spell table, whose column 20 is
// a spell's enchant base. CastSpells are the spells a cast draws from for a
// mage's item and for a fighter's. PointsPrice prices the ordinary effects'
// points and PriceCap caps the item's price. CastInCapacity keeps a cast's
// power within the item's capacity left. Repeat joins an effect of a kind the
// item holds.
type Enchant struct {
	Effects, Spells Params
	CastSpells      [2][]uint16
	PointsPrice     func(points []int32) int64
	PriceCap        int64
	CastInCapacity  bool
	Repeat          Repeat
}

// Base is the item an enchant starts from: its price, its capacity for effect
// points, the category's price ceiling, the slot whose weight column it
// draws from, whether a fighter wears it, and whether its one effect is a
// cast the item is made for.
type Base struct {
	Price, Capacity, Ceiling int32
	Slot                     int
	Fighter, Forced          bool
}

// EffectDraw is one effect an enchant made: the effect, the points an
// ordinary effect costs and the price a cast adds.
type EffectDraw struct {
	Effect       Effect
	Points, Cast int32
}

// Apply enchants b on src and answers the effects and the price, or false
// when the first effect fails. A forced item takes one cast. Another takes a
// weighted first effect; a cast ends it there. Otherwise a second and a third
// pick are each attempted on a draw of 0..100 below 50 and below 25; a failed
// pick, or a cast, which is dropped, ends the chain. An effect whose UsableBy
// mask shares no bit with a held effect's is dropped.
func (e Enchant) Apply(b Base, src Source) ([]Effect, int32, bool) {
	if e.Effects == nil || src == nil {
		return nil, 0, false
	}
	var effects []Effect
	var points []int32
	price, remaining := b.Price, b.Capacity
	castPrice := int64(0)

	reprice := func() {
		sum := int32(0)
		for _, p := range points {
			sum += p
		}
		remaining = b.Capacity - sum
		total := int64(b.Price) + e.PointsPrice(points) + castPrice
		if total >= e.PriceCap {
			price = int32(e.PriceCap)
		} else {
			price = int32(total)
		}
	}
	add := func(g EffectDraw) {
		mask := e.usableBy(g.Effect.Kind)
		if mask != 3 {
			for _, held := range effects {
				if e.usableBy(held.Kind)&mask == 0 {
					reprice()
					return
				}
			}
		}
		if i := Held(effects, func(x Effect) uint8 { return x.Kind }, g.Effect.Kind, e.Repeat); i >= 0 {
			effects[i].Operand = g.Effect.Operand
			points[i] = g.Points
			reprice()
			return
		}
		effects = append(effects, g.Effect)
		points = append(points, g.Points)
		castPrice += int64(g.Cast)
		reprice()
	}
	construct := func(forced bool) (EffectDraw, bool) {
		budget := int64(2)*int64(b.Ceiling) - int64(price)
		if forced {
			budget = min(budget, int64(price)*100)
			for attempt := 1; attempt <= tries; attempt++ {
				if g, ok := e.Cast(false, budget, remaining, src); ok {
					return g, attempt < tries
				}
			}
			return EffectDraw{}, false
		}
		kind, ok := e.pick(b.Fighter, b.Slot, src)
		if !ok {
			return EffectDraw{}, false
		}
		if kind == castSpell {
			// The weighted route's cast takes ten times the price as its
			// budget.
			return e.Cast(b.Fighter, int64(price)*10, remaining, src)
		}
		return e.Ordinary(kind, budget, remaining, src)
	}

	first, ok := construct(b.Forced)
	if !ok {
		return nil, 0, false
	}
	add(first)
	if first.Effect.Kind == castSpell {
		return effects, price, true
	}
	for _, threshold := range []int{50, 25} {
		if gate, _ := Inclusive(src, 0, 100); gate >= threshold {
			continue
		}
		g, ok := construct(false)
		if !ok || g.Effect.Kind == castSpell {
			return effects, price, true
		}
		add(g)
	}
	return effects, price, true
}

// usableBy is an effect row's UsableBy mask; a row without one restricts
// nothing.
func (e Enchant) usableBy(kind uint8) int32 {
	if e.Effects == nil || int(kind) >= e.Effects.Len() {
		return 3
	}
	p := e.Effects.EntryParams(int(kind))
	if len(p) <= 3 {
		return 3
	}
	return p[3]
}

// pick draws an effect kind by the cumulative weights of the slot's column:
// slot−1, plus 12 for an item no fighter wears.
func (e Enchant) pick(fighter bool, slot int, src Source) (uint8, bool) {
	if e.Effects == nil || slot < 1 || slot > 12 {
		return 0, false
	}
	column := slot - 1
	if !fighter {
		column += 12
	}
	final := int32(0)
	for kind := 1; kind < e.Effects.Len(); kind++ {
		if p := e.Effects.EntryParams(kind); len(p) > 4+column {
			final = p[4+column]
		}
	}
	if final <= 0 {
		return 0, false
	}
	for attempt := 1; attempt < tries; attempt++ {
		draw, _ := Inclusive(src, 1, int(final))
		previous := int32(0)
		for kind := 1; kind < e.Effects.Len(); kind++ {
			endpoint := previous
			if p := e.Effects.EntryParams(kind); len(p) > 4+column {
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

// Ordinary is an effect of kind within budget and capacity: a value of
// 1..min(capacity/cost, budget's payload, maximum), raised to the minimum; a
// damage kind 44..48 draws a base and a spread instead.
func (e Enchant) Ordinary(kind uint8, budget int64, capacity int32, src Source) (EffectDraw, bool) {
	if e.Effects == nil || int(kind) >= e.Effects.Len() || budget <= 0 || capacity <= 0 {
		return EffectDraw{}, false
	}
	p := e.Effects.EntryParams(int(kind))
	if len(p) < 3 || p[0] <= 0 {
		return EffectDraw{}, false
	}
	cost, minimum, maximum := p[0], p[1], p[2]
	capacityMax := capacity / cost
	x := float64(budget)/float64(50*int64(capacity)) - 1
	if x <= 0 {
		return EffectDraw{}, false
	}
	budgetMax := int32((70 / float64(cost)) * (math.Log(x) / math.Log(1.5)))
	payloadMax := min(capacityMax, budgetMax, maximum)
	if payloadMax < minimum || payloadMax < 1 {
		return EffectDraw{}, false
	}
	draw, _ := Inclusive(src, 1, int(payloadMax))
	payload := max(int32(draw), minimum)
	if kind >= 44 && kind <= 48 {
		base, _ := Inclusive(src, 1, int(min(payloadMax, 255)))
		spread, ok := Inclusive(src, 1, int(min(payloadMax, 255)/2))
		if !ok {
			return EffectDraw{}, false
		}
		return EffectDraw{Effect: Effect{Kind: kind, Operand: uint32(uint8(base)) | uint32(uint8(spread))<<8},
			Points: int32(base+spread) * cost}, true
	}
	return EffectDraw{Effect: Effect{Kind: kind, Operand: uint32(payload)}, Points: payload * cost}, true
}

// Cast is a castspell effect within budget: a spell drawn from the mage's or
// the fighter's list, at a power of 1..trunc(30*(1.2^log2(budget/(10*base))
// − 1)), at most 100 and, under CastInCapacity, at most capacity. Its price is
// trunc(10*base*2^(ln(1+power/30)/ln 1.2)).
func (e Enchant) Cast(fighter bool, budget int64, capacity int32, src Source) (EffectDraw, bool) {
	ids := e.CastSpells[0]
	if fighter {
		ids = e.CastSpells[1]
	}
	if len(ids) == 0 {
		return EffectDraw{}, false
	}
	at, _ := Inclusive(src, 0, len(ids)-1)
	id := ids[at]
	if e.Spells == nil || int(id) >= e.Spells.Len() {
		return EffectDraw{}, false
	}
	p := e.Spells.EntryParams(int(id))
	if len(p) <= 20 || p[20] <= 0 {
		return EffectDraw{}, false
	}
	scalar := p[20]
	ratio := float64(budget) / float64(10*int64(scalar))
	if ratio <= 1 {
		return EffectDraw{}, false
	}
	powerMax := min(int32(30*(math.Pow(1.2, math.Log2(ratio))-1)), 100)
	if e.CastInCapacity {
		powerMax = min(powerMax, capacity)
	}
	if powerMax <= 0 {
		return EffectDraw{}, false
	}
	power, _ := Inclusive(src, 1, int(powerMax))
	cast := int32(10 * float64(scalar) * math.Pow(2, math.Log(float64(power)/30+1)/math.Log(1.2)))
	return EffectDraw{Effect: Effect{Kind: castSpell,
		Operand: uint32(id) | uint32(uint16(int16(power)))<<16}, Cast: cast}, true
}
