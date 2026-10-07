package data

import "encoding/binary"

// The shop's candidate pool and what a candidate is worth (SHOP-POOL-006,
// SHOP-POOL-021, SHOP-PRICE-011, SHOP-MISSION-019).
//
// THIS FILE PRICES AND ADMITS; IT DOES NOT DRAW. Which candidates a shelf takes,
// how many, and with what generator is pkg/game's business. Nothing here reads a
// customer, a purse or a shop.
//
// THE TWO VOCABULARIES ARE INVERTED AND THIS IS THE ONE PLACE THAT SAYS SO.
// SHOP-POOL-006 calls the row of the Armors, Shields or Weapons collection the
// *shape* and the index into the Shapes collection the *tier*. This tree has
// called them the other way round since 0110: an item code's field C is the
// `shape` (the Shapes entry, the quality word a name begins with) and field D is
// the `row`. Every name below is this tree's. Read against the claim:
//
//	claim "shape" = row here      (Armors / Shields / Weapons entry, field D)
//	claim "tier"  = shape here    (Shapes entry 0..4, field C)
//	claim "material bit" = material here (Materials entry 0..15, field A)

// ShopShapes is how many 16-bit material masks a row carries: one per Shapes
// entry, and the Shapes collection ships exactly this many.
//
// It is a property of the RECORD and not of the shipped file. The raw block an
// armour, shield or weapon entry carries is ten bytes, which is these five masks
// end to end, so a sixth would have nowhere in the entry to live.
const ShopShapes = 5

// ShopMaterials is how many bits of a mask name a material, which is also how
// many entries the Materials collection ships. Field A of an item code is four
// bits wide, so a seventeenth material could not be named by a code either.
const ShopMaterials = 16

const itemPriceColumn = 2

// scalePriceSlot is the slot of a shape's or a material's nine-double record
// that scales a price. The image reads it at `+0x30` of a record based at
// `+0x20`, i.e. eight bytes times two — the same arithmetic that puts damage at
// slot 4 and defence at slot 6 in weapon.go.
const scalePriceSlot = 2

// scaleMagCapSlot is the final scale double used by all three equipment fills.
const scaleMagCapSlot = 8

// shopMaskStride is the byte distance between one shape's mask and the next
// inside the raw block. Shape 0's mask is at offset 0. Written as a stride
// rather than as five offsets so the block's layout is stated once.
const shopMaskStride = 2

// MaskTable is a Collection whose entries also carry their raw record bytes.
//
// IT IS A SECOND INTERFACE AND NOT A WIDER Collection: the raw block is read
// by this file alone, and every hand-built Collection in the tree would
// otherwise have had to grow a method for it. A caller holding a Collection
// asserts to this at the point of use and treats a failed assertion as a
// table that admits nothing, which is the answer a nil table already gives.
type MaskTable interface {
	Collection

	// EntryRaw is entry i's undecoded record bytes, or nil where the entry
	// carries none. For an armour, shield or weapon row it is the ten-byte
	// block holding ShopShapes 16-bit material masks, little-endian, shape s
	// at byte offset shopMaskStride*s.
	EntryRaw(i int) []byte
}

// ShopCandidate is one admitted (row, shape, material) triple: the item code it
// composes and what one unit of it costs.
//
// Price IS THE ITEM'S, NOT THE WINDOW'S. The two differ by the rounding addend
// alone — see ItemValue and ItemPrice — and a consumer that used the window's
// value as a price would undercharge by a coin on every fractional product.
//
// The three indices are carried because the draw that consumes a candidate has
// them already; re-deriving them from the code would read back the same three
// fields the code was composed from.
type ShopCandidate struct {
	Code                 ItemCode
	Price                int32
	Row, Shape, Material int
	MagCap               int32
	ItemKind             uint8
	EffectSlot           int
	Fighter              bool
	ForcedCast           bool
}

// ItemMagCap is the base item's enchantment capacity.
func ItemMagCap(shapes, materials ScaleTable, shape, material int) int32 {
	return ftol(scale(shapes, shape, scaleMagCapSlot) * scale(materials, material, scaleMagCapSlot))
}

// ItemValue is what the window compares: `trunc(price x shape x material)`
// (SHOP-POOL-021). No rounding addend — that is ItemPrice's, and the difference
// between the two is exactly the `+0.5` SHOP-POOL-006's retraction turned on.
func ItemValue(price int32, shapes, materials ScaleTable, shape, material int) int32 {
	return ftol(float64(price) * scale(shapes, shape, scalePriceSlot) *
		scale(materials, material, scalePriceSlot))
}

// ItemPrice is what one unit costs: `trunc(price x shape x material + 0.5)`
// (SHOP-PRICE-011; SHOP-ROUND-017 sites (a)(b)(c) are three per-class
// constructors with byte-identical arithmetic, so this is ONE function and not
// three).
//
// IT READS NO CUSTOMER, which is the whole of SHOP-PRICE-011: there is no
// parameter here a hero, a party or a purse could reach.
func ItemPrice(price int32, shapes, materials ScaleTable, shape, material int) int32 {
	return ftol(float64(price)*scale(shapes, shape, scalePriceSlot)*
		scale(materials, material, scalePriceSlot) + 0.5)
}

// itemMask is row's own 16-bit material mask for shape, or 0 where the row
// carries no block, carries a short one, or the table cannot answer at all.
//
// A ZERO MASK ADMITS NOTHING, which is what keeps every refusal to one arm: a
// missing table, an unwritten entry and a row that names no material at this
// shape all reach the walk as "no bit is set".
func itemMask(c MaskTable, row, shape int) uint16 {
	if c == nil || row < 0 || row >= c.Len() || shape < 0 || shape >= ShopShapes {
		return 0
	}
	raw := c.EntryRaw(row)
	off := shopMaskStride * shape
	if len(raw) < off+2 {
		return 0
	}
	return binary.LittleEndian.Uint16(raw[off:])
}

// ShopClass answers what field B a row composes with, given the row's own
// parameters. The three item collections differ here and nowhere else in the
// walk.
//
// It is a FUNCTION VALUE rather than three walks because the walk itself is
// identical for all three (SHOP-POOL-006: kind 1 is Shields and Armors together,
// kind 2 is Weapons, and both take the same admission).
type ShopClass func(params []int32) int

// WeaponShopClass is a weapon's own constant class.
func WeaponShopClass([]int32) int { return weaponItemClass }

// ShieldShopClass is a shield's own constant class.
func ShieldShopClass([]int32) int { return shieldItemClass }

// ArmorShopClass is an armour row's Slot column, or 0 where the row is too short
// to carry one or where the cell names no equipment place. That is fillArmor's
// own rule, reused so a shop item and the same item resolved from its name
// compose the identical code.
func ArmorShopClass(p []int32) int {
	if len(p) <= armorSlotColumn {
		return 0
	}
	slot := int(p[armorSlotColumn])
	if !validEquipSlot(slot) {
		return 0
	}
	return slot
}

// ShopPool is every candidate collection c admits inside the value window
// [0, ceiling].
//
// THE FLOOR IS THE LITERAL 0 AND IS NOT A PARAMETER (SHOP-MISSION-019): all four
// pool constructions in the original push a zero, and the per-mission
// `ShopMinPrice` key is read, transmitted and never applied. Giving this
// function a floor would add a knob the original does not have. The floor is not
// inert — it is what excludes the negative-price sentinel rows, the deleted
// weapon row and the quest magic items, all at -1.
//
// EVERY SHAPE IS WALKED, which is SHOP-POOL-006's `tier == 5` reading: the
// generator's own calls pass the all-shapes value, so this function takes no
// shape argument at all.
//
// The walk starts at row 1. The three item collections are one-based and index 0
// is the entry their count allocates and never writes; a row with no name is
// skipped for the reason findByName skips it.
//
// A nil c, a nil class and a negative ceiling each answer nil, and so does a c
// whose entries carry no raw block: a shop with no table to draw from has an
// empty shelf, never a partial one.
func ShopPool(c MaskTable, class ShopClass, shapes, materials ScaleTable, ceiling int32) []ShopCandidate {
	if c == nil || class == nil || ceiling < 0 {
		return nil
	}
	var out []ShopCandidate
	for row := 1; row < c.Len(); row++ {
		if c.EntryName(row) == "" {
			continue
		}
		p := c.EntryParams(row)
		if len(p) <= itemPriceColumn {
			continue
		}
		base := p[itemPriceColumn]
		b := class(p)
		suit, _ := SuitabilityFromParams(p)
		for shape := 0; shape < ShopShapes; shape++ {
			mask := itemMask(c, row, shape)
			if mask == 0 {
				continue
			}
			for material := 0; material < ShopMaterials; material++ {
				if mask&(1<<uint(material)) == 0 {
					continue
				}
				v := ItemValue(base, shapes, materials, shape, material)
				if v < 0 || v > ceiling {
					continue
				}
				out = append(out, ShopCandidate{
					Code:     ComposeItemCode(material, b, shape, row),
					Price:    ItemPrice(base, shapes, materials, shape, material),
					MagCap:   ItemMagCap(shapes, materials, shape, material),
					Row:      row,
					Shape:    shape,
					Material: material,
					ItemKind: func() uint8 {
						if b == weaponItemClass {
							return 2
						}
						return 1
					}(),
					EffectSlot: b,
					Fighter:    suit.Fighter,
					ForcedCast: b == weaponItemClass && !suit.Fighter && suit.Mage,
				})
			}
		}
	}
	return out
}
