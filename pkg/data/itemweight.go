package data

import "fmt"

// The weight column and the factor it is scaled by.
//
// itemWeightColumn IS RUNTIME COLUMN 3 of an Armors, Shields or Weapons row.
// The three collections share one eighteen-entry title array and carry
// seventeen cells, so `slot i = title i+1` exactly (ITEM-WEAPCOL-021, High,
// anchored four times at instruction level), and title 4 is `weight`. Two
// further readings land on the same number: HERO-EQUIP-017 states the weapon
// fill builds `w+0x4a` from column 3, and DAT-SCHEMA-007 lists that fill as
// reading slots 3/6/7/8/9 as weight/physicalMin/physicalMax/toHit/deIrnce.
// weapon.go's own ladder is the same numbering, one slot per attribute.
//
// scaleWeightSlot IS LADDER SLOT 3 of a shape's or a material's nine-double
// record. ITEM-LADDER-019 (High) fixes the ladder at `f64[j] = record + 0x20
// + 8j` by the record's own 0x68 size, gives `f64[j] = title[j+1]` for the
// eleven-title Shapes/Materials group, and enumerates the complete read
// population: `+0x30` (f64[2]) by the price routines, `+0x38` (f64[3]),
// `+0x50` (f64[6]) and `+0x60` (f64[8]) by ALL THREE item fills, `+0x40`
// (f64[4]) and `+0x48` (f64[5]) by the weapon fill alone, `+0x58` (f64[7])
// by the armour and shield fills alone. Title 4 of that group is `weight`,
// which puts the weight factor at f64[3] — one of the three read by all
// three fills, which is what a column every item class carries needs. The
// other two ladder slots this package already reads sit where the same
// correspondence puts them: scaleDefenceSlot 6 on title 7 `#.defence` and
// scaleAbsorptionSlot 7 on title 8 `#.absorption`.
//
// HERO-EQUIP-017 states the fill's five products as `ftol(column x shape x
// material)`. The rounding taken here is `ftol(x + 0.5)`, the same one
// ResolveWeapon's ToHit and Defence and fillArmor's Defence already take:
// ITEM-DMGFACT-020's own re-execution of that fill spells the damage byte as
// `ftol(23 x 0.2 + 0.5)`, so the half is inside the fill rather than a
// convention of one column.
const (
	itemWeightColumn = 3
	scaleWeightSlot  = 3
)

// itemWeight is the ONE arithmetic a carried item's per-unit weight is worth:
// the row's own weight column times the product of the shape's and the
// material's factor at scaleWeightSlot, rounded and truncated toward zero.
//
// A ROW TOO SHORT TO CARRY THE COLUMN WEIGHS ITS SCALED ZERO (armorColumn's
// own convention, reused rather than re-spelled). That is not a refusal:
// every shipped row on either root carries the column, and a row that did
// not would be an item whose table states no weight, which is exactly zero.
//
// THE RESULT IS SIGNED AND IS NOT CLAMPED. The original's field is a signed
// word (`item+0x4a`, ITEM-STACK-003), the shipped Weapons collection carries
// a `rem` row whose price and weight columns are both -1, and a negative
// product is therefore a state the shipped tables can produce. Clamping it
// here would state something the table does not. DIVERGENCES.md DIV-225
// carries the row and the shipped count.
func itemWeight(p []int32, shape, material int, shapes, materials ScaleTable) int32 {
	f := scale(shapes, shape, scaleWeightSlot) * scale(materials, material, scaleWeightSlot)
	return ftol(float64(armorColumn(p, itemWeightColumn))*f + 0.5)
}

// ItemCodeWeight is the per-unit weight of the item an item code names, for
// EVERY code, dispatched on the code's own class field.
//
// It is the one seam between an item's identity and its weight, and it is
// stated here rather than at each caller because the simulation tier holds no
// definition table at all: whoever builds a world resolves each code it can
// name through this function once, and the simulation then carries the
// answers rather than the tables.
//
// THE DISPATCH IS field B's, and it is the same three-way split the three
// FromCode functions already draw: weaponItemClass names a weapon,
// shieldItemClass a shield, and any other valid equipment slot an armour.
// A code naming no equipment slot at all -- ItemClassCarried among them --
// resolves to no item here and answers false, which is a code this function
// states no weight for rather than one it states zero for.
//
// A TABLE THE CALLER DOES NOT HAVE ANSWERS FALSE, not an error and not a
// fault. The two scale tables and the one collection the dispatched class
// needs are each checked for nil before the FromCode call, because a nil
// Collection or ScaleTable is an interface value with no method set and
// calling Len on it faults. A caller resolving a mission's codes over a
// partial install has exactly this shape: Shapes and Materials read, one of
// the three item collections missing, and every code of that one class then
// weighing nothing while the other two classes still resolve.
//
// IT REPORTS FALSE RATHER THAN ERRORING for a code that resolves to nothing,
// because the caller's question is "how much does this weigh" over a list of
// codes some of which are carried documents and quest tokens. The three
// FromCode functions' own errors are wrapped only when a code that DOES name
// an equipment class fails to resolve, which is a table the caller cannot
// read rather than a code it should not have asked about.
func ItemCodeWeight(c ItemCode, shapes, materials ScaleTable, armors, shields, weapons Collection) (int32, bool, error) {
	if shapes == nil || materials == nil {
		return 0, false, nil
	}
	switch b := c.B(); {
	case b == weaponItemClass:
		if weapons == nil {
			return 0, false, nil
		}
		w, err := WeaponFromCode(c, shapes, materials, weapons)
		if err != nil {
			return 0, false, fmt.Errorf("data: weight of item code 0x%04x: %w", uint16(c), err)
		}
		return w.Weight, true, nil
	case b == shieldItemClass:
		if shields == nil {
			return 0, false, nil
		}
		s, err := ShieldFromCode(c, shapes, materials, shields)
		if err != nil {
			return 0, false, fmt.Errorf("data: weight of item code 0x%04x: %w", uint16(c), err)
		}
		return s.Weight, true, nil
	case validEquipSlot(b):
		if armors == nil {
			return 0, false, nil
		}
		a, err := ArmorFromCode(c, shapes, materials, armors)
		if err != nil {
			return 0, false, fmt.Errorf("data: weight of item code 0x%04x: %w", uint16(c), err)
		}
		return a.Weight, true, nil
	default:
		return 0, false, nil
	}
}
