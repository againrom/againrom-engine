package secondshop

import (
	"math"

	"againrom/pkg/shopstock"
)

// Stock is the four category lists of one shop.
type Stock [4][]Item

// enchantSpells is the spell table an enchanted category draws from
// (R2-ENGINE-265).
var enchantSpells = [...]int{1, 10, 11, 18, 26, 5, 16}

// Fill limits (R2-ENGINE-265): a book pick is redrawn up to 1000 times, an
// unusable spell up to 100 times, and failures end a category after ten times
// its draws. Potions enter at 51..100 and books at quantity 2.
const (
	bookRetries    = 1000
	spellRetries   = 100
	failuresFactor = 10
	potionLeast    = 51
	potionSpan     = 50
	bookQuantity   = 2
	levelCeiling   = 100
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
// draws. The mods' stocked items then join the stock by placement; they take
// no draw. The second game has no count multiplier, and its enchant gives one
// spell, so no effect repeats.
func (t Tables) Profile(r Record, rnd Rand) shopstock.Profile[Item, Item] {
	var p shopstock.Profile[Item, Item]
	for _, cat := range r {
		p.Categories = append(p.Categories, shopstock.Category[Item, Item]{
			Candidates: func(shopstock.Source) []Item { return t.Candidates(cat, rnd) },
			Lead:       func(cands []Item, src shopstock.Source) []Item { return t.lead(cat, cands, src) },
			Count:      func(shopstock.Source) int { return int(cat.Draws) },
			Bound:      shopstock.Bound{Failures: func(n int) int { return failuresFactor * n }},
			Redraw:     func(it Item) bool { return it.Kind == Book },
			Redraws:    bookRetries,
			Make:       func(it Item, src shopstock.Source) (Item, bool) { return t.make(it, cat, src) },
			Insert:     Insert,
		})
	}
	p.Place = func(stock [][]Item) {
		for _, it := range t.Stocked {
			c := Place(r, it)
			stock[c] = Insert(stock[c], it)
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
		if it.Kind == Book {
			it.Quantity, it.Price = bookQuantity, t.Price(it)
			list = append(list, it)
		}
	}
	for row := potionFirst; row <= potionLast; row++ {
		it := Item{Kind: Magic, Row: row, Quantity: potionLeast + src.Intn(potionSpan)}
		it.Price = t.Price(it)
		list = append(list, it)
	}
	return list
}

// make is one drawn unit of it: an enchanted category enchants it and
// refuses a price outside the range; another draws a stackable quantity of
// 1..Bound+1.
func (t Tables) make(it Item, cat Category, src shopstock.Source) (Item, bool) {
	enchanted := cat.Mask&bitEnchanted != 0
	it.Quantity = 1
	if enchanted {
		if !t.enchant(&it, cat, src) {
			return it, false
		}
	} else if it.Stackable() {
		it.Quantity = 1 + src.Intn(int(cat.Bound+1))
	}
	it.Price = t.Price(it)
	if enchanted && (it.Price < int64(cat.Min) || it.Price > int64(cat.Max)) {
		return it, false
	}
	return it, true
}

// LevelCap is the highest level an enchant of spell s can take on item it in
// category cat: trunc((1.2^log2(budget/(10*c20)) - 1)*30), budget
// min(2*max - P, 100*P), capped by the item's level field and 100; -1 when
// none (R2-ENGINE-265).
func (t Tables) LevelCap(it Item, s int, cat Category) int {
	p := int64(t.baseProduct(it) + 0.5)
	budget := min(2*int64(cat.Max)-p, 100*p)
	c20 := param(t.Spells, s, colEnchant)
	if c20 <= 0 || budget <= 0 {
		return -1
	}
	cap := int((math.Pow(1.2, math.Log2(float64(budget)/(10*float64(c20)))) - 1) * 30)
	cap = min(cap, t.levelProduct(it), levelCeiling)
	if cap < 1 {
		return -1
	}
	return cap
}

// enchant gives it one spell at a uniform level 1..cap (R2-ENGINE-265). The
// spell index and level draw forms are chosen (DIV-3042).
func (t Tables) enchant(it *Item, cat Category, src shopstock.Source) bool {
	for i := 0; i < spellRetries; i++ {
		s := enchantSpells[src.Intn(len(enchantSpells))]
		if cap := t.LevelCap(*it, s, cat); cap > 0 {
			it.Spell, it.Level = s, 1+src.Intn(cap)
			return true
		}
	}
	return false
}

// Insert adds it to list, merging into an equal stackable entry or a
// quantity-0 placeholder of the same key, else appending (R2-ENGINE-266).
func Insert(list []Item, it Item) []Item {
	for i := range list {
		if list[i].Same(it) && (it.Stackable() || list[i].Quantity == 0) {
			list[i].Quantity += it.Quantity
			return list
		}
	}
	return append(list, it)
}
