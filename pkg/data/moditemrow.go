package data

import "math"

// Mod item rows state their numbers for the reference item: common quality,
// made of iron (shape 0 and material 0). The table row holds the column that
// the shape and material factors of that item scale to the stated number; every
// other shape and material scales the same column differently, as for every
// shipped row.

// ItemAttr names a number a mod states for an item.
type ItemAttr int

// The numbers a mod states.
const (
	AttrPrice ItemAttr = iota
	AttrWeight
	AttrDefence
	AttrAbsorption
)

// ModItemShape and ModItemMaterial are the shape and material of the reference
// item and of every item a mod adds.
const (
	ModItemShape    = 0
	ModItemMaterial = 0
)

func attrColumn(a ItemAttr) (column, slot int) {
	switch a {
	case AttrPrice:
		return itemPriceColumn, scalePriceSlot
	case AttrWeight:
		return itemWeightColumn, scaleWeightSlot
	case AttrDefence:
		return armorDefenceColumn, scaleDefenceSlot
	default:
		return armorAbsorptionColumn, scaleAbsorptionSlot
	}
}

// ReferenceColumn is the column value whose reference item carries v for the
// attribute: v divided by the product of the reference shape's and material's
// factor, rounded to the nearest integer.
func ReferenceColumn(shapes, materials ScaleTable, a ItemAttr, v int32) int32 {
	_, slot := attrColumn(a)
	f := scale(shapes, ModItemShape, slot) * scale(materials, ModItemMaterial, slot)
	if f <= 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return v
	}
	return int32(math.Round(float64(v) / f))
}

// modRowWidth is the width of the shipped armour, shield and weapon rows.
const modRowWidth = 17

// WithReferenceColumn returns a copy of the row parameters p with attribute a
// set so that the reference item carries v. A short row is widened with the
// empty cell.
func WithReferenceColumn(p []int32, shapes, materials ScaleTable, a ItemAttr, v int32) []int32 {
	col, _ := attrColumn(a)
	out := append([]int32(nil), p...)
	for len(out) <= col {
		out = append(out, emptyCell)
	}
	out[col] = ReferenceColumn(shapes, materials, a, v)
	return out
}

// NewModRowParams is the parameter array of an armour or shield row a mod adds:
// the shipped rows' width, the slot, the suitability bits, and the four
// numbers as columns of the reference item. Every cell the mod does not state
// is the empty cell, or zero where the shipped armour rows hold zero.
func NewModRowParams(shapes, materials ScaleTable, slot, suitable int, price, weight, defence, absorption int32) []int32 {
	p := make([]int32, modRowWidth)
	for i := range p {
		p[i] = emptyCell
	}
	for _, i := range []int{0, 1, 6, 7, 8} {
		p[i] = 0
	}
	p[armorSlotColumn] = int32(slot)
	p[SutableForColumn] = int32(suitable)
	for _, s := range []struct {
		a ItemAttr
		v int32
	}{{AttrPrice, price}, {AttrWeight, weight}, {AttrDefence, defence}, {AttrAbsorption, absorption}} {
		col, _ := attrColumn(s.a)
		p[col] = ReferenceColumn(shapes, materials, s.a, s.v)
	}
	return p
}

// ModRowRaw is the raw record of a mod row: five material masks, none set, so
// the shop's random draw never picks the item. A mod puts it on a shelf itself.
func ModRowRaw() []byte { return make([]byte, 2*ShopShapes) }
