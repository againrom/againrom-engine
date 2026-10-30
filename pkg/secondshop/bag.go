package secondshop

import (
	"slices"

	"againrom/pkg/shopstock"
	"againrom/pkg/sim"
)

// The hero's bag holds the engine's one item instance, one per unit. The
// item ID is material<<12 | part<<8 | class<<5 | row (R2-ENGINE-403,
// R2-ENGINE-327); a magic item takes part 14 and a book part 14 with row
// bit 7 set (DIV-3043's chosen codes). Effects are the item's Effect list:
// tag, flags and the u32 at +0x40 (R2-ENGINE-403, R2-ENGINE-414).
const (
	partWeapon   = 1
	partShield   = 2
	partMagic    = 0xe
	bookRow      = 0x80
	tagCastSpell = 41
	tagTeach     = 42
	// typeArmor and typeWeapon are the +0x44 type bytes placement reads
	// (R2-ENGINE-333); the other kinds' bytes are Unknown and stay zero.
	typeArmor  = 1
	typeWeapon = 2
)

// Instance is one unit of it as the bag holds it, or false when the item
// has no ID form: a part outside the equipment range or a row past 5 bits.
func (t Tables) Instance(it Item) (sim.ItemInstance, bool) {
	part := 0
	switch it.Table {
	case Armor, Shield, Weapon:
		part = t.Part(it)
	}
	return unit(it, part)
}

// unit is Instance over the item's equipment part, which the tables answer
// for a shop item and the ID for a worn one.
func unit(it Item, part int) (sim.ItemInstance, bool) {
	out := sim.ItemInstance{Price: min(it.Price, priceCap), Kind: typeOf(it.Table)}
	switch it.Table {
	case Armor, Shield, Weapon:
		if part < partWeapon || part >= partMagic || it.Row < 0 || it.Row > 0x1f || it.Class < 0 || it.Class > 7 || it.Material < 0 || it.Material > 0xf {
			return sim.ItemInstance{}, false
		}
		out.Code = uint16(it.Material)<<12 | uint16(part)<<8 | uint16(it.Class)<<5 | uint16(it.Row)
		if it.Level > 0 {
			out.Effects = []sim.ItemEffect{{Kind: tagCastSpell, Operand: uint32(it.Spell&0xffff) | uint32(it.Level&0xffff)<<16}}
		}
	case Magic:
		if it.Row < 0 || it.Row >= bookRow {
			return sim.ItemInstance{}, false
		}
		out.Code = partMagic<<8 | uint16(it.Row)
	case Book:
		if it.Spell < 0 || it.Spell >= bookRow {
			return sim.ItemInstance{}, false
		}
		out.Code = partMagic<<8 | bookRow | uint16(it.Spell)
		out.Effects = []sim.ItemEffect{{Kind: tagTeach, Operand: uint32(it.Spell)}}
	default:
		return sim.ItemInstance{}, false
	}
	return out, true
}

// typeOf is the type byte of a table's units.
func typeOf(table Kind) uint8 {
	switch table {
	case Armor:
		return typeArmor
	case Weapon:
		return typeWeapon
	}
	return 0
}

// idItem reads an item ID as the shop item it names, quantity 1 with no
// price or effect, or false when it names none: a part outside 1..14, or a
// magic item or book with material bits.
func idItem(code uint16) (Item, bool) {
	it := Item{Count: 1}
	switch part := int(code>>8) & 0xf; {
	case part == partMagic && code&bookRow != 0:
		it.Table, it.Spell = Book, int(code&(bookRow-1))
		return it, code>>12 == 0
	case part == partMagic:
		it.Table, it.Row = Magic, int(code&0xff)
		return it, code>>12 == 0
	case part == partWeapon:
		it.Table = Weapon
	case part == partShield:
		it.Table = Shield
	case part > partShield && part < partMagic:
		it.Table = Armor
	default:
		return Item{}, false
	}
	it.Material, it.Class, it.Row = int(code>>12), int(code>>5)&7, int(code)&0x1f
	return it, true
}

// BookSpell is the spell of a book ID, or false when the ID is not a book's:
// the one reading of a book's ID that the picture, which the spell's school
// owns, asks.
func BookSpell(code uint16) (int, bool) {
	it, ok := idItem(code)
	return it.Spell, ok && it.Table == Book
}

// MagicRow is the magic table row a magic item's ID names, the row an Item
// holds at +0xc (R2-ENGINE-403), or false when the ID names no magic item.
func MagicRow(code uint16) (int, bool) {
	it, ok := idItem(code)
	return it.Row, ok && it.Table == Magic
}

// Unit is a worn item as the bag holds it: the shop item its ID names, at
// its price and with its castspell effect, written by Instance's rule. A
// unit Instance writes is its own Unit; an ID that names no shop item is
// kept as it is.
func Unit(in sim.ItemInstance) sim.ItemInstance {
	if _, ok := FromInstance(in); ok {
		return in.Clone()
	}
	it, ok := idItem(in.Code)
	if !ok {
		return in.Clone()
	}
	it.Price = max(0, in.Price)
	if spell, level, cast := in.CastSpell(); cast && level > 0 && it.Table != Magic && it.Table != Book {
		it.Spell, it.Level = int(spell), int(level)
	}
	if out, ok := unit(it, int(in.Code>>8)&0xf); ok {
		return out
	}
	return in.Clone()
}

// MissionUnit is bag unit u as a mission holds it (DIV-4191): the item the
// engine's decoder writes for u's ID, with u's price, and with u's castspell
// or teachspell effect when it has one. A bought potion thus takes the type
// and payload a potion the mission places has. The decoder is the caller's
// (mapload.ItemInstanceFromCode over the install's table), as this package
// reads no install table.
func MissionUnit(u sim.ItemInstance, decode func(code uint16) sim.ItemInstance) sim.ItemInstance {
	out := decode(u.Code)
	out.Code, out.Price = u.Code, u.Price
	if len(u.Effects) != 0 {
		out.Effects = slices.Clone(u.Effects)
	}
	return out
}

// KeptUnit is the bag unit a unit of a hero's pack comes home as, or false
// when the bag would lose part of it. A bag unit comes as it is. Any other
// unit comes as Unit(in) when MissionUnit turns that back into in's ID,
// type, price and effects; the weight and equipment a mission derives from
// the ID do not count. It refuses a unit bound to a saved object or a native
// record, one at a price below 0 or past the cap, and one whose effects or
// type the mission form does not reproduce.
func KeptUnit(in sim.ItemInstance, decode func(code uint16) sim.ItemInstance) (sim.ItemInstance, bool) {
	if in.ObjectID != 0 || in.NativeRecord != nil {
		return sim.ItemInstance{}, false
	}
	if _, ok := FromInstance(in); ok {
		return in.Clone(), true
	}
	out := Unit(in)
	if _, ok := FromInstance(out); !ok {
		return sim.ItemInstance{}, false
	}
	if back := MissionUnit(out, decode); back.Code != in.Code || back.Kind != in.Kind || back.Price != in.Price || !slices.Equal(back.Effects, in.Effects) {
		return sim.ItemInstance{}, false
	}
	return out, true
}

// FromInstance reads one bag unit back as a shop item of quantity 1, or
// false when the instance is not one Instance writes.
func FromInstance(in sim.ItemInstance) (Item, bool) {
	it, ok := idItem(in.Code)
	if !ok || in.Kind != typeOf(it.Table) || in.Price < 0 || in.ObjectID != 0 || in.NativeRecord != nil || in.WeightPresent || in.Weight != 0 || in.SourceEquipment != (sim.SourceEquipment{}) {
		return Item{}, false
	}
	it.Price = in.Price
	switch it.Table {
	case Book:
		return it, len(in.Effects) == 1 && in.Effects[0] == (sim.ItemEffect{Kind: tagTeach, Operand: uint32(it.Spell)})
	case Magic:
		return it, len(in.Effects) == 0
	}
	switch len(in.Effects) {
	case 0:
	case 1:
		e := in.Effects[0]
		if e.Kind != tagCastSpell || e.Mode != 0 || e.Operand>>16 == 0 {
			return Item{}, false
		}
		it.Spell, it.Level = int(e.Operand&0xffff), int(e.Operand>>16)
	default:
		return Item{}, false
	}
	return it, true
}

// Units is it as bag units: Quantity instances of one unit each.
func (t Tables) Units(it Item) ([]sim.ItemInstance, bool) {
	unit, ok := t.Instance(it)
	if !ok || it.Count < 0 {
		return nil, false
	}
	out := make([]sim.ItemInstance, it.Count)
	for i := range out {
		out[i] = unit.Clone()
	}
	return out, true
}

// Stacks is a bag as shop items: units merge into the first equal item by
// Stacking, in the order they first appear. It is false when a unit is
// not one Instance writes.
func Stacks(bag []sim.ItemInstance) ([]Item, bool) {
	out, _, ok := stacks(bag)
	return out, ok
}

// stacks is Stacks with the stack each unit went to.
func stacks(bag []sim.ItemInstance) ([]Item, []int, bool) {
	var out []Item
	at := make([]int, len(bag))
	for k, in := range bag {
		it, ok := FromInstance(in)
		if !ok {
			return nil, nil, false
		}
		at[k] = shopstock.Find(out, Stacking, it)
		if at[k] < 0 {
			at[k] = len(out)
		}
		out = insert(out, it)
	}
	return out, at, true
}

// TakeUnits removes every unit of stack i of bag, as Stacks numbers them,
// and answers the bag left, the units taken and the stack they make.
func TakeUnits(bag []sim.ItemInstance, i int) (left, taken []sim.ItemInstance, stack Item, ok bool) {
	all, at, ok := stacks(bag)
	if !ok || i < 0 || i >= len(all) {
		return bag, nil, Item{}, false
	}
	for k, in := range bag {
		if at[k] == i {
			taken = append(taken, in.Clone())
		} else {
			left = append(left, in.Clone())
		}
	}
	return left, taken, all[i], true
}

// Stocked reads a mod's armour or shield code as one unit of a shop item at
// price, or false when the code is neither in the bag's item form.
func Stocked(code uint16, price int32) (Item, bool) {
	for _, kind := range []uint8{typeArmor, 0} {
		if it, ok := FromInstance(sim.ItemInstance{Code: code, Kind: kind, Price: price}); ok && (it.Table == Armor || it.Table == Shield) {
			return it, true
		}
	}
	return Item{}, false
}
