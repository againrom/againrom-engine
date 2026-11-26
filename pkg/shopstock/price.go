package shopstock

import "math"

// FixedEffect selects how effect kind 1 changes an equipment price.
type FixedEffect uint8

const (
	FixedAdd FixedEffect = iota
	FixedReplace
)

// ItemClass identifies an equipment class for cast pricing.
type ItemClass uint8

const (
	ItemWeapon ItemClass = 1 << iota
	ItemShield
	ItemArmor
)

// ItemPriceRow states one game's effect-price rules.
type ItemPriceRow struct {
	Fixed             FixedEffect
	OrdinaryCastKinds ItemClass
	SkipNegativePower bool
	Cap               int64
	SpellCol          int
	CostCol           int
}

// FirstItemPrice is the first game's effect-price row.
var FirstItemPrice = ItemPriceRow{
	Fixed:             FixedAdd,
	OrdinaryCastKinds: ItemArmor | ItemShield,
	SkipNegativePower: true,
	Cap:               9_999_999,
	SpellCol:          20,
	CostCol:           0,
}

// SecondItemPrice is the second game's effect-price row.
var SecondItemPrice = ItemPriceRow{
	Fixed:             FixedReplace,
	OrdinaryCastKinds: ItemArmor | ItemShield,
	SkipNegativePower: false,
	Cap:               19_999_999,
	SpellCol:          20,
	CostCol:           0,
}

const (
	fixedPriceKind = 1
	damageFirst    = 44
	damageLast     = 48
)

// PointsMarkup is trunc((1.5^(S/70)+1)*S*50), zero for S <= 0, capped at cap.
func PointsMarkup(points, cap int64) int64 {
	if points <= 0 {
		return 0
	}
	n := float64(points)
	v := (math.Pow(1.5, n/70) + 1) * n * 50
	if cap > 0 && v >= float64(cap) {
		return cap
	}
	return int64(v)
}

// PointsSumPrice is PointsMarkup of the sum of points, capped at cap.
func PointsSumPrice(points []int32, cap int64) int64 {
	var sum int64
	for _, p := range points {
		sum += int64(p)
	}
	return PointsMarkup(sum, cap)
}

// SpellPrice is trunc(10*c20*2^(ln(1+L/30)/ln 1.2)).
func SpellPrice(c20, level int32) int64 {
	return int64(10 * float64(c20) * math.Pow(2, math.Log(1+float64(level)/30)/math.Log(1.2)))
}

func effectScalar(mode uint8, operand uint32) int32 {
	if mode == 0 || mode == 8 {
		return int32(operand)
	}
	return int32(int16(operand))
}

// EffectsPrice prices base and effects under row.
func EffectsPrice[T ~struct {
	Kind    uint8
	Mode    uint8
	Operand uint32
}](base int64, effects []T, class ItemClass, magic, spells Params, row ItemPriceRow) int64 {
	castInPoints := class&row.OrdinaryCastKinds != 0
	var points, cast, direct int64
	for _, raw := range effects {
		e := Effect(raw)
		if e.Kind == fixedPriceKind {
			if row.Fixed == FixedReplace {
				return clampPrice(int64(int32(e.Operand)), row.Cap)
			}
			direct += int64(effectScalar(e.Mode, e.Operand))
			continue
		}
		if e.Kind == castSpell && !castInPoints {
			if spells == nil {
				continue
			}
			spell := int(uint16(e.Operand))
			if spell < 0 || spell >= spells.Len() {
				continue
			}
			params := spells.EntryParams(spell)
			if len(params) <= row.SpellCol || params[row.SpellCol] <= 0 {
				continue
			}
			var power int32
			if row.SkipNegativePower {
				power = int32(int16(e.Operand >> 16))
				if power < 0 {
					continue
				}
			} else {
				power = int32(e.Operand >> 16)
			}
			cast += SpellPrice(params[row.SpellCol], power)
			continue
		}
		if magic == nil || int(e.Kind) >= magic.Len() {
			continue
		}
		params := magic.EntryParams(int(e.Kind))
		if len(params) <= row.CostCol || params[row.CostCol] <= 0 {
			continue
		}
		magnitude := effectScalar(e.Mode, e.Operand)
		switch {
		case e.Kind == castSpell:
			if row.SkipNegativePower {
				magnitude = int32(int16(e.Operand >> 16))
			} else {
				magnitude = int32(e.Operand >> 16)
			}
		case e.Kind >= damageFirst && e.Kind <= damageLast:
			magnitude = int32(uint8(e.Operand)) + int32(uint8(e.Operand>>8))
		}
		points += int64(magnitude) * int64(params[row.CostCol])
	}
	price := base + direct + cast
	if points > 0 {
		limit := row.Cap
		if limit > 0 && price < 0 {
			limit -= price
		}
		price += PointsMarkup(points, limit)
	}
	return clampPrice(price, row.Cap)
}

func clampPrice(price, cap int64) int64 {
	if cap > 0 && price >= cap {
		return cap
	}
	if price < -1<<31 {
		return -1 << 31
	}
	return price
}
