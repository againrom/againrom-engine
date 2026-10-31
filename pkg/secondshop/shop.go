// Package secondshop holds the second game's shop rules: the assortment
// record, candidate admission, the stock fill, prices, the sale credit and the
// buy, sell and undo settlements. It reads the second game's data.bin
// collections through a narrow interface and draws through a rand() source the
// caller supplies, so one seed replays a fill exactly.
package secondshop

import (
	"math"

	"againrom/pkg/shopitem"
)

// Collection is the slice of a data.bin collection the rules read.
type Collection interface {
	Len() int
	EntryParams(i int) []int32
	EntryRaw(i int) []byte
	EntryDoubles(i int) []float64
}

// Tables are the second game's data.bin collections the shop reads. Shapes
// are the item classes, Materials the materials and Magic the effect table,
// data.bin group B (R2-ENGINE-264, R2-ENGINE-711).
type Tables struct {
	Shapes, Materials, Armors, Shields, Weapons, MagicItems, Spells, Magic Collection
	// Stocked are the items mods put in the shop: one unit each, filed by
	// placement into every stock a fill draws.
	Stocked []Item
}

// Rand answers one rand() value in 0..32767.
type Rand func() int

// Category is one 0x14-byte category entry of an assortment record
// (R2-ENGINE-263).
type Category struct {
	Min, Max, Draws, Bound int32
	Mask                   uint32
}

// Record is one location's assortment record: four categories.
type Record [4]Category

// Mask bits (R2-ENGINE-264).
const (
	kindWeapon   = 0x400000
	kindShield   = 0x800000
	kindArmor    = 0x1000000
	kindMagic    = 0x4000000
	kindStaff    = 0x8000000
	bitEnchanted = 1 << 29
	classShift   = 15
	classCount   = 7
	materialBits = 15
)

// The data.bin columns the rules read, as parameter-array slots: the price of
// an equipment row, its body part, its use word, whose bit 0 is the fighter's,
// a magic item's price, a spell's enchant base and its book price, and an
// effect's point cost (R2-ENGINE-264, R2-ENGINE-265, R2-ENGINE-267,
// R2-ENGINE-327, R2-ENGINE-710, R2-ENGINE-711).
const (
	colPrice     = 2
	colPart      = 4
	colWeaponUse = 15
	colMagic     = 0
	colEnchant   = 20
	colBook      = 21
	colCost      = 0
)

// The nine-double slots of a class or material record: the price factor and
// the level factor: column 2 and column 8 (R2-ENGINE-332). The focused test
// reproduces every admitted population and price the claims give with them.
const (
	factorPrice = 2
	factorLevel = 8
)

// materialOfBit maps a mask material bit to its material: bits 0..12 are
// materials 0..12, bit 13 material 14 and bit 14 material 15.
func materialOfBit(bit int) int {
	if bit >= 13 {
		return bit + 1
	}
	return bit
}

// Kind is an item's table.
type Kind = shopitem.Table

const (
	Armor Kind = iota + 1
	Shield
	Weapon
	Magic
	Book
)

// Item is one stock entry or one item in a deal; Price is the unit price P
// (R2-ENGINE-267).
type Item = shopitem.Item

// stackable is the merge rule: a magic item, or an item with no effect
// (R2-ENGINE-265). A book carries its spell and is not.
func stackable(a Item) bool {
	return a.Table == Magic || a.Table != Book && len(a.Effects) == 0
}

// SaleCredit is the gold a sale of q units at unit price P credits:
// floor(q*P/2+0.5) (R2-ENGINE-266, R2-ENGINE-267).
func SaleCredit(price int64, q int) int64 { return (price*int64(q) + 1) / 2 }

// SaleCell is the unit price a shop cell shows for an item offered for sale:
// (P+1)/2 (R2-ENGINE-267).
func SaleCell(price int64) int64 { return (price + 1) / 2 }

// priceCap is the total price ceiling (R2-ENGINE-267).
const priceCap = 19999999

// EffectPrice is the price of an effect of size S other than a spell:
// trunc(50*S*(1+(S/70)^1.5)) (R2-ENGINE-267).
func EffectPrice(s int) int64 {
	v := float64(s)
	return int64(50 * v * (1 + math.Pow(v/70, 1.5)))
}

// SpellEffectPrice is the price of an enchanted spell of base c20 at level L:
// trunc(10*c20*2^(ln(1+L/30)/ln 1.2)) (R2-ENGINE-265).
func SpellEffectPrice(c20 int32, level int) int64 {
	return int64(10 * float64(c20) * math.Pow(2, math.Log(1+float64(level)/30)/math.Log(1.2)))
}

func param(c Collection, row, col int) int32 {
	if c == nil || row <= 0 || row >= c.Len() {
		return -1
	}
	p := c.EntryParams(row)
	if col >= len(p) {
		return -1
	}
	return p[col]
}

func factor(c Collection, row, slot int) float64 {
	if c == nil || row < 0 || row >= c.Len() {
		return 0
	}
	d := c.EntryDoubles(row)
	if slot >= len(d) {
		return 0
	}
	return d[slot]
}

func (t Tables) equipment(k Kind) Collection {
	switch k {
	case Armor:
		return t.Armors
	case Shield:
		return t.Shields
	case Weapon:
		return t.Weapons
	}
	return nil
}

// baseProduct is column2 x material factor x class factor.
func (t Tables) baseProduct(it Item) float64 {
	return float64(param(t.equipment(it.Table), it.Row, colPrice)) * factor(t.Materials, it.Material, factorPrice) * factor(t.Shapes, it.Class, factorPrice)
}

// Part is an equipment row's body part, the item code's slot field.
func (t Tables) Part(it Item) int { return int(param(t.equipment(it.Table), it.Row, colPart)) }

// NameKey is the key of the item's line in the shipped item-name table
// (R2-ENGINE-267; the key form is DIV-3046's chosen join): equipment
// material<<12 | part<<8 | class<<5 | row, a magic item 0x0e00 | row. A book
// has none.
func (t Tables) NameKey(it Item) (uint16, bool) {
	switch it.Table {
	case Armor, Shield, Weapon:
		return uint16(it.Material&0xf)<<12 | uint16(t.Part(it)&0xf)<<8 | uint16(it.Class&0x7)<<5 | uint16(it.Row&0x1f), true
	case Magic:
		return 0x0e00 | uint16(it.Row&0xff), true
	}
	return 0, false
}

// Price is the unit price P of an item (R2-ENGINE-267): equipment
// trunc(base product + 0.5) plus its effects' prices, a magic item its column
// 0, a book its spell's column 21; capped at 19999999.
func (t Tables) Price(it Item) int64 {
	var p int64
	switch it.Table {
	case Armor, Shield, Weapon:
		p = int64(t.baseProduct(it)+0.5) + t.effectsPrice(it.Effects)
	case Magic:
		p = int64(param(t.MagicItems, it.Row, colMagic))
	case Book:
		p = int64(param(t.Spells, it.Spell, colBook))
	}
	return min(p, priceCap)
}

// levelProduct is trunc(material level x class level), the enchant level
// field (R2-ENGINE-265, R2-ENGINE-327).
func (t Tables) levelProduct(it Item) int {
	return int(factor(t.Materials, it.Material, factorLevel) * factor(t.Shapes, it.Class, factorLevel))
}

// armorExcluded are the (class, material, row) armour triples admission
// refuses (R2-ENGINE-264, R2-ENGINE-327).
var armorExcluded = [][3]int{{0, 0, 2}, {0, 1, 2}, {6, 4, 6}}

// bookSkipped are the spells admission builds no book for (R2-ENGINE-264).
var bookSkipped = map[int]bool{9: true, 14: true, 15: true, 24: true, 28: true, 29: true}

// Book and scroll row arithmetic (R2-ENGINE-264, R2-ENGINE-327): spells
// 1..29, phase-two MagicItems row i+5 or i+34, potions rows 69..74.
const (
	bookSpells   = 29
	scrollOffset = 5
	superOffset  = 34
	potionFirst  = 69
	potionLast   = 74
)

// Candidates are the items category c admits from the tables
// (R2-ENGINE-264). The magic kind's phase-two rows take one of two rows per
// spell, chosen per fill by rnd (DIV-3042); a nil rnd takes the first.
func (t Tables) Candidates(c Category, rnd Rand) []Item {
	var out []Item
	enchanted := c.Mask&bitEnchanted != 0
	admit := func(it Item, price float64) bool {
		if enchanted {
			return math.Pow(float64(c.Min), 0.4) <= float64(t.levelProduct(it))
		}
		p := int64(price)
		return p >= int64(c.Min) && p <= int64(c.Max)
	}
	type table struct {
		kind Kind
		bit  uint32
		keep func(row int) bool
	}
	tables := []table{
		{Armor, kindArmor, nil},
		{Shield, kindShield, nil},
		{Weapon, kindWeapon, func(row int) bool { return param(t.Weapons, row, colWeaponUse)&1 != 0 }},
	}
	if enchanted {
		tables = append(tables, table{Weapon, kindStaff, func(row int) bool { return param(t.Weapons, row, colWeaponUse)&1 == 0 }})
	}
	for _, tb := range tables {
		coll := t.equipment(tb.kind)
		if c.Mask&tb.bit == 0 || coll == nil {
			continue
		}
		for row := 1; row < coll.Len(); row++ {
			if tb.keep != nil && !tb.keep(row) {
				continue
			}
			raw := coll.EntryRaw(row)
			for class := 0; class < classCount; class++ {
				if c.Mask&(1<<(classShift+class)) == 0 || 2*class+1 >= len(raw) {
					continue
				}
				word := int(raw[2*class]) | int(raw[2*class+1])<<8
				for bit := 0; bit < materialBits; bit++ {
					m := materialOfBit(bit)
					if c.Mask&(1<<bit) == 0 || word&(1<<m) == 0 {
						continue
					}
					if tb.kind == Armor && excluded(class, m, row) {
						continue
					}
					it := Item{Table: tb.kind, Row: row, Class: class, Material: m}
					if admit(it, t.baseProduct(it)) {
						out = append(out, it)
					}
				}
			}
		}
	}
	if c.Mask&kindMagic != 0 {
		for s := 1; s <= bookSpells; s++ {
			it := Item{Table: Book, Spell: s}
			if !bookSkipped[s] && admit(it, float64(t.Price(it))) {
				out = append(out, it)
			}
		}
		for i := 1; i <= bookSpells; i++ {
			row := i + scrollOffset
			if rnd != nil && (rnd()*2)>>15 != 0 {
				row = i + superOffset
			}
			it := Item{Table: Magic, Row: row}
			if admit(it, float64(t.Price(it))) {
				out = append(out, it)
			}
		}
	}
	return out
}

func excluded(class, material, row int) bool {
	for _, x := range armorExcluded {
		if x == [3]int{class, material, row} {
			return true
		}
	}
	return false
}

// NewGameRecords are the assortment records of locations 0..3 a new game
// writes: record 1 for town 1 (R2-ENGINE-263) and record 2 for town 2
// (R2-ENGINE-330); records 0 and 3 stay zero (R2-ENGINE-329) and admit nothing.
var NewGameRecords = [4]Record{
	1: {
		{Min: 0, Max: 1500, Draws: 100, Bound: 2, Mask: 0x1381cc03},
		{Min: 0, Max: 1500, Draws: 100, Bound: 2, Mask: 0x10418103},
		{Min: 0, Max: 1500, Draws: 20, Bound: 1, Mask: 0x28018100},
		{Min: 0, Max: 1500, Draws: 20, Bound: 1, Mask: 0x04000000},
	},
	2: {
		{Min: 0, Max: 5000, Draws: 100, Bound: 2, Mask: 0x1383c305},
		{Min: 0, Max: 5000, Draws: 20, Bound: 1, Mask: 0x2bc3c304},
		{Min: 0, Max: 5000, Draws: 20, Bound: 1, Mask: 0x04000000},
		{Min: 0, Max: 5000, Draws: 100, Bound: 2, Mask: 0x10438305},
	},
}
