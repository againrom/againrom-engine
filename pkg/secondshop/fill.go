package secondshop

import (
	"fmt"

	"againrom/pkg/shopstock"
	"againrom/pkg/sim"
)

// Stock is the four category lists of one shop.
type Stock [4][]Item

// castSpells are the spells an enchanted cast draws from, a mage's item's and
// a fighter's (R2-ENGINE-265, R2-ENGINE-711).
var castSpells = [2][]uint16{{1, 10, 11, 18, 26, 5, 16}, {18, 26}}

// Fill limits (R2-ENGINE-265): a book pick is redrawn up to 1000 times and
// failures end a category after ten times its draws. Potions enter at 51..100
// and books at quantity 2.
const (
	bookRetries    = 1000
	failuresFactor = 10
	potionLeast    = 51
	potionSpan     = 50
	bookQuantity   = 2
)

// picks is rnd as the generator's draw source: Intn is (rand()*n)>>15, the
// draw's index over n candidates.
type picks Rand

func (r picks) Intn(n int) int { return (r() * n) >> 15 }

// Fill draws a whole stock from record r (R2-ENGINE-265) through the shared
// generator.
func (t Tables) Fill(r Record, rnd Rand) Stock {
	var s Stock
	copy(s[:], shopstock.Generate(t.Profile(r, rnd), picks(rnd)))
	return s
}

// Profile is record r as the shared generator's profile (R2-ENGINE-265).
// Each category admits its candidates; the magic category first takes each
// admitted book at quantity 2 and the six potions; each draw then picks a
// candidate, redrawing a book; an enchanted category enchants its pick and
// drops a price outside the range. Failures end a category at ten times its
// draws, and Sort orders it (R2-ENGINE-710). The mods' stocked items then
// join the stock by placement; they take no draw. The second game has no
// count multiplier.
func (t Tables) Profile(r Record, rnd Rand) shopstock.Profile[Item, Item] {
	p := shopstock.Profile[Item, Item]{Repeat: effectRepeat}
	for _, cat := range r {
		p.Categories = append(p.Categories, shopstock.Category[Item, Item]{
			Candidates: func(shopstock.Source) []Item { return t.Candidates(cat, rnd) },
			Lead:       func(cands []Item, src shopstock.Source) []Item { return t.lead(cat, cands, src) },
			Count:      func(shopstock.Source) int { return int(cat.Draws) },
			Bound:      shopstock.Bound{Failures: func(n int) int { return failuresFactor * n }},
			Redraw:     func(it Item) bool { return it.Table == Book },
			Redraws:    bookRetries,
			Make:       func(it Item, src shopstock.Source) (Item, bool) { return t.make(it, cat, src) },
			Insert:     insert,
			Finish: func(list []Item, _ shopstock.Source) []Item {
				t.Sort(list)
				return list
			},
		})
	}
	p.Place = func(stock [][]Item) {
		for _, it := range t.Stocked {
			c := Place(r, it)
			stock[c] = insert(stock[c], it)
		}
	}
	return p
}

// lead is what a magic category holds before its draws: each admitted book
// at quantity 2, then the six potions at 51..100.
func (t Tables) lead(cat Category, cands []Item, src shopstock.Source) []Item {
	if cat.Mask&kindMagic == 0 {
		return nil
	}
	var list []Item
	for _, it := range cands {
		if it.Table == Book {
			it.Count, it.Price = bookQuantity, int32(t.Price(it))
			list = append(list, it)
		}
	}
	for row := potionFirst; row <= potionLast; row++ {
		it := Item{Table: Magic, Row: row, Count: int32(potionLeast + src.Intn(potionSpan))}
		it.Price = int32(t.Price(it))
		list = append(list, it)
	}
	return list
}

// make is one drawn unit of it: an enchanted category enchants it and
// refuses a price outside the range; another draws a stackable quantity of
// 1..Bound+1.
func (t Tables) make(it Item, cat Category, src shopstock.Source) (Item, bool) {
	enchanted := cat.Mask&bitEnchanted != 0
	it.Count = 1
	if enchanted {
		effects, _, ok := t.enchant().Apply(t.enchantBase(it, cat), src)
		if !ok {
			return it, false
		}
		it.Effects = shopstock.EffectsAs[sim.ItemEffect](effects)
	} else if stackable(it) {
		it.Count = int32(1 + src.Intn(int(cat.Bound+1)))
	}
	p := t.Price(it)
	it.Price = int32(p)
	if enchanted && (p < int64(cat.Min) || p > int64(cat.Max)) {
		return it, false
	}
	return it, true
}

// enchant is the second game's rules for the one enchant (R2-ENGINE-711):
// the group B table, whose columns hold a slot's weights as the first
// game's Magic table does, the spells a cast draws from, each effect's
// points priced on their own, the price cap, and a cast's level within the
// item's level field (R2-ENGINE-265). The value of an ordinary effect, the
// add rule, the weighted cast's budget and the effect-repeat rule are the
// first game's (DIV-4329).
func (t Tables) enchant() shopstock.Enchant {
	e := shopstock.Enchant{
		CastSpells: castSpells,
		PointsPrice: func(points []int32) int64 {
			p := int64(0)
			for _, n := range points {
				p += EffectPrice(int(n))
			}
			return p
		},
		PriceCap:       priceCap,
		CastInCapacity: true,
		Repeat:         effectRepeat,
	}
	if t.Magic != nil {
		e.Effects = t.Magic
	}
	if t.Spells != nil {
		e.Spells = t.Spells
	}
	return e
}

// effectRepeat is the effect-repeat rule the enchant applies: the first
// game's, a repeated kind taking the held entry's place (DIV-4329).
const effectRepeat = shopstock.RepeatReplace

// enchantBase is it as the enchant starts it (R2-ENGINE-711): price P, the
// level field as its capacity, the category's maximum as the ceiling, its
// body part as the slot and its row's fighter bit; a weapon no fighter
// wields takes its one cast.
func (t Tables) enchantBase(it Item, cat Category) shopstock.Base {
	fighter := t.fighter(it)
	return shopstock.Base{Price: int32(t.Price(it)), Capacity: int32(t.levelProduct(it)), Ceiling: cat.Max,
		Slot: t.Part(it), Fighter: fighter, Forced: !fighter && it.Table == Weapon}
}

// fighter is whether an equipment row sets the fighter bit, bit 0 of its use
// word (R2-ENGINE-710, R2-ENGINE-711).
func (t Tables) fighter(it Item) bool {
	use := param(t.equipment(it.Table), it.Row, colWeaponUse)
	return use > 0 && use&1 != 0
}

// The elemental damage effects, whose value is a base and a spread.
const (
	damageFirst = 44
	damageLast  = 48
)

// effectsPrice is what effects add to an equipment price: a cast its spell's
// price at its level (R2-ENGINE-265), an ordinary effect EffectPrice of its
// points, its value times its group B cost (DIV-4329).
func (t Tables) effectsPrice(effects []sim.ItemEffect) int64 {
	p := int64(0)
	var points []int32
	for _, e := range effects {
		if e.Kind == tagCastSpell {
			p += SpellEffectPrice(param(t.Spells, int(e.Operand&0xffff), colEnchant), int(e.Operand>>16))
			continue
		}
		value := int32(e.Operand)
		if e.Kind >= damageFirst && e.Kind <= damageLast {
			value = int32(uint8(e.Operand)) + int32(uint8(e.Operand>>8))
		}
		points = append(points, value*param(t.Magic, int(e.Kind), colCost))
	}
	return p + t.enchant().PointsPrice(points)
}

// Sort orders a category list as the fill does (R2-ENGINE-710): an
// equipment row with the fighter bit first, then the row ascending, then the
// price ascending. A magic item or a book sorts by price alone, the first
// game's order (DIV-4328); items equal on every key keep their order.
func (t Tables) Sort(list []Item) {
	shopstock.SortBy(list, func(it Item) []int64 {
		switch it.Table {
		case Armor, Shield, Weapon:
			fighter := int64(1)
			if t.fighter(it) {
				fighter = 0
			}
			return []int64{fighter, int64(it.Row), int64(it.Price)}
		}
		return []int64{1, 0, int64(it.Price)}
	})
}

// stackKey is the item but its quantity and price.
type stackKey struct {
	table                       Kind
	row, class, material, spell int
	effects                     string
}

// Stacking is the second game's stack-key rule (R2-ENGINE-265,
// R2-ENGINE-266): the key is the item but its quantity and price; an equal
// entry takes a stackable item, a quantity-0 placeholder any; no order.
var Stacking = shopstock.Stack[Item, stackKey]{
	Key: func(it Item) stackKey {
		return stackKey{it.Table, it.Row, it.Class, it.Material, it.Spell, fmt.Sprint(it.Effects)}
	},
	Takes: func(entry, it Item) bool { return stackable(it) || entry.Count == 0 },
	Add: func(entry, it Item) Item {
		entry.Count += it.Count
		return entry
	},
}

// insert files it on list by Stacking.
func insert(list []Item, it Item) []Item { return shopstock.Insert(list, Stacking, it) }
