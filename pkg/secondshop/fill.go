package secondshop

import "math"

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

// pick is (rand()*n)>>15, the draw's index over n candidates.
func pick(rnd Rand, n int) int { return (rnd() * n) >> 15 }

// Fill draws a whole stock from record r (R2-ENGINE-265). The magic category
// first takes each admitted book at quantity 2 and the six potions; each draw
// then picks a candidate, redrawing a book; an enchanted category enchants its
// pick and drops a price outside the range.
func (t Tables) Fill(r Record, rnd Rand) Stock {
	var s Stock
	for c, cat := range r {
		cands := t.Candidates(cat, rnd)
		var list []Item
		if cat.Mask&kindMagic != 0 {
			for _, it := range cands {
				if it.Kind == Book {
					it.Quantity, it.Price = bookQuantity, t.Price(it)
					list = append(list, it)
				}
			}
			for row := potionFirst; row <= potionLast; row++ {
				it := Item{Kind: Magic, Row: row, Quantity: potionLeast + pick(rnd, potionSpan)}
				it.Price = t.Price(it)
				list = append(list, it)
			}
		}
		if len(cands) > 0 {
			list = t.draw(list, cands, cat, rnd)
		}
		s[c] = list
	}
	return s
}

func (t Tables) draw(list, cands []Item, cat Category, rnd Rand) []Item {
	enchanted := cat.Mask&bitEnchanted != 0
	for done, failed := 0, 0; done < int(cat.Draws) && failed < failuresFactor*int(cat.Draws); {
		it := cands[pick(rnd, len(cands))]
		for i := 0; it.Kind == Book && i < bookRetries; i++ {
			it = cands[pick(rnd, len(cands))]
		}
		if it.Kind == Book {
			failed++
			continue
		}
		it.Quantity = 1
		if enchanted {
			if !t.enchant(&it, cat, rnd) {
				failed++
				continue
			}
		} else if it.Stackable() {
			it.Quantity = 1 + (rnd()*int(cat.Bound+1))>>15
		}
		it.Price = t.Price(it)
		if enchanted && (it.Price < int64(cat.Min) || it.Price > int64(cat.Max)) {
			failed++
			continue
		}
		list = Insert(list, it)
		done++
	}
	return list
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
func (t Tables) enchant(it *Item, cat Category, rnd Rand) bool {
	for i := 0; i < spellRetries; i++ {
		s := enchantSpells[pick(rnd, len(enchantSpells))]
		if cap := t.LevelCap(*it, s, cat); cap > 0 {
			it.Spell, it.Level = s, 1+pick(rnd, cap)
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
