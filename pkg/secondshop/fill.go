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

// R2-ENGINE-941: stop above ten failures per draw or at a book timeout.
// Potions enter at 51..100 and books at quantity 2 (R2-ENGINE-265).
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

// Fill draws a stock from record r (R2-ENGINE-265) through the shared
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
// drops a price outside the range. Failures above ten times its draws end
// it (R2-ENGINE-941); Sort orders it (R2-ENGINE-710). Mods' stocked items then
// join the stock by placement; they take no draw. With Rolls above one, each
// category that took items rolls again that many items Factor-1 times on
// Rolls' own generator, ends each roll by the same failure bound and joins
// the further items into its list; the lead items, mods' stocked items and a
// death's list are not rolled again.
func (t Tables) Profile(r Record, rnd Rand) shopstock.Profile[Item, Item] {
	p := shopstock.Profile[Item, Item]{Repeat: effectRepeat}
	for _, cat := range r {
		p.Categories = append(p.Categories, t.category(cat, rnd))
	}
	p.Place = func(stock [][]Item) {
		for _, it := range t.Stocked {
			c := Place(r, it)
			stock[c] = insert(stock[c], it)
		}
	}
	p.Multiplier = shopstock.Multiplier[Item]{
		Factor: t.Rolls.Factor,
		Source: t.Rolls.Source,
		Join: func(_ int, list, more []Item) []Item {
			for _, it := range more {
				list = insert(list, it)
			}
			t.sortFill(list)
			return list
		},
	}
	return p
}

// category is cat as the shared generator's category: the one draw of a
// shop's category and of a death's item list.
func (t Tables) category(cat Category, rnd Rand) shopstock.Category[Item, Item] {
	return shopstock.Category[Item, Item]{
		Candidates: func(shopstock.Source) []Item { return t.Candidates(cat, rnd) },
		Lead:       func(cands []Item, src shopstock.Source) []Item { return t.lead(cat, cands, src) },
		Count:      func(shopstock.Source) int { return int(cat.Draws) },
		Bound:      shopstock.Bound{Failures: func(n int) int { return failuresFactor * n }},
		Again:      shopstock.Bound{Failures: func(n int) int { return failuresFactor * n }},
		Redraw:     func(it Item) bool { return it.Table == Book },
		Redraws:    bookRetries,
		Make:       func(it Item, src shopstock.Source) (Item, bool) { return t.make(it, cat, src) },
		Insert:     insert,
		Finish: func(list []Item, _ shopstock.Source) []Item {
			t.sortFill(list)
			return list
		},
	}
}

// The death item draw's record (R2-ENGINE-940): 100 draws at bound 1.
const (
	deathDraws = 100
	deathBound = 1
)

// DeathItems is the list a death's item draw generates (R2-ENGINE-941): the
// shop draw of one category {least, most, 100 draws, bound 1, mask} on rnd,
// without the mods' stocked items, each entry one unit as the bag holds it.
// An entry with no bag form is a zero item in its place.
func (t Tables) DeathItems(least, most int32, mask uint32, rnd Rand) []sim.ItemInstance {
	cat := Category{Min: least, Max: most, Draws: deathDraws, Bound: deathBound, Mask: mask}
	p := shopstock.Profile[Item, Item]{Repeat: effectRepeat, Categories: []shopstock.Category[Item, Item]{t.category(cat, rnd)}}
	list := shopstock.Generate(p, picks(rnd))[0]
	out := make([]sim.ItemInstance, len(list))
	for i, it := range list {
		if unit, ok := t.Instance(it); ok {
			out[i] = unit
		}
	}
	return out
}

// lead preloads a nonempty magic category: books at 2, six potions at 51..100.
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

// make is one drawn unit of it (R2-ENGINE-941). An enchanted category
// enchants it, with bit 28 too only on a rand(100) above 50, and refuses a
// price outside the range, a unit left plain too. A kept unit that stacks
// draws a quantity of 1..Bound+1.
func (t Tables) make(it Item, cat Category, src shopstock.Source) (Item, bool) {
	checked := cat.Mask&bitEnchanted != 0
	enchant := checked
	if checked && cat.Mask&bitHalfEnchant != 0 {
		enchant = src.Intn(halfEnchantDraw) > halfEnchantPass
	}
	it.Count = 1
	if enchant {
		effects, _, ok := t.enchant().Apply(t.enchantBase(it, cat), src)
		if !ok {
			return it, false
		}
		it.Effects = shopstock.EffectsAs[sim.ItemEffect](effects)
	}
	p := t.Price(it)
	it.Price = int32(p)
	if checked && (p < int64(cat.Min) || p > int64(cat.Max)) {
		return it, false
	}
	if stackable(it) {
		it.Count = int32(1 + src.Intn(int(cat.Bound+1)))
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

// sortFill supplies the server order that the client permutes among equal
// keys. Magic lists retain draw order where the server's slots are Unknown.
func (t Tables) sortFill(list []Item) {
	for _, it := range list {
		if it.Table == Magic || it.Table == Book {
			return
		}
	}
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
	}, true)
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
