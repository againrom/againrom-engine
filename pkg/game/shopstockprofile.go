package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/shopstock"
	"againrom/pkg/sim"
)

// shopEffectRepeat is the first game's effect-repeat rule: an effect of a
// kind the item holds replaces that entry's value in place.
const shopEffectRepeat = shopstock.RepeatReplace

// shopStockCandidate is one candidate of a first-game shelf: a definition
// row the draw makes an item of, or a books-shelf item taken as it is.
type shopStockCandidate struct {
	row  data.ShopCandidate
	item ShopItem
}

func shopRowCandidates(pool []data.ShopCandidate) []shopStockCandidate {
	out := make([]shopStockCandidate, len(pool))
	for i, c := range pool {
		out[i].row = c
	}
	return out
}

func shopItemCandidates(items []ShopItem) []shopStockCandidate {
	out := make([]shopStockCandidate, len(items))
	for i, it := range items {
		out[i].item = it
	}
	return out
}

// shopFixedBound bounds every attempt of a roll at n.
func shopFixedBound(n int) shopstock.Bound {
	return shopstock.Bound{Attempts: func(int) int { return n }}
}

// stockProfile is the first game's profile of the shared stock generator for
// this shop (SHOP-GEN-005, SHOP-POOL-006, SHOP-MAGIC-007). The location is the
// shop's ceiling. Armour and weapons draw 100 plain units each from their
// filtered definition-table pools within 100 attempts; Magic Items draws 20
// enchanted units from both unfiltered pools within 201 attempts; the books
// shelf draws 1..8 picks from the books and scrolls, then takes the
// guaranteed books (DIV-1025) and the potions. Each shelf is stacked and
// sorted. The mods' stocked items join the armour shelf without a draw
// (DIV-1862). The stock multiplier (DIV-3960) rolls on its own generator
// from the shop's seed. A nil draw source stocks no books shelf.
func (s *Shop) stockProfile(t *mapload.Table, r shopDrawSource) shopstock.Profile[shopStockCandidate, ShopItem] {
	allArmour := shopArmourPool(t, s.ceiling)
	allWeapons := shopWeaponPool(t, s.ceiling)
	armour := shopPlainShelfPool(allArmour)
	weapons := shopPlainShelfPool(allWeapons)
	magic := append(append([]data.ShopCandidate{}, allArmour...), allWeapons...)
	ceiling := s.ceiling
	order := func(items []ShopItem) []ShopItem {
		items = shopStackShelf(items)
		shopSortShelf(items, t)
		return items
	}
	finish := func(items []ShopItem, _ shopstock.Source) []ShopItem { return order(items) }
	shelf := func(pool []data.ShopCandidate, n, attempts, again int, mk func(shopStockCandidate, shopstock.Source) (ShopItem, bool)) shopstock.Category[shopStockCandidate, ShopItem] {
		cands := shopRowCandidates(pool)
		return shopstock.Category[shopStockCandidate, ShopItem]{
			Candidates: func(shopstock.Source) []shopStockCandidate { return cands },
			Count:      func(shopstock.Source) int { return n },
			Bound:      shopFixedBound(attempts),
			Again:      shopFixedBound(again),
			Make:       mk,
			Finish:     finish,
		}
	}
	plain := func(c shopStockCandidate, _ shopstock.Source) (ShopItem, bool) {
		return shopItemFromInstance(sim.ItemInstance{Code: uint16(c.row.Code), Kind: c.row.ItemKind, Price: c.row.Price}, 1), true
	}
	p := shopstock.Profile[shopStockCandidate, ShopItem]{Repeat: shopEffectRepeat}
	enchanted := func(c shopStockCandidate, src shopstock.Source) (ShopItem, bool) {
		return shopEnchant(c.row, ceiling, t, src, p.Repeat)
	}
	plainDraws, magicDraws := shopShelfDraws[ShelfArmour], shopShelfDraws[ShelfMagic]
	p.Categories = []shopstock.Category[shopStockCandidate, ShopItem]{
		ShelfArmour:  shelf(armour, plainDraws, plainDraws, plainDraws, plain),
		ShelfWeapons: shelf(weapons, shopShelfDraws[ShelfWeapons], shopShelfDraws[ShelfWeapons], shopShelfDraws[ShelfWeapons], plain),
		ShelfMagic:   shelf(magic, magicDraws, 10*magicDraws+1, 100*(10*magicDraws+1), enchanted),
	}
	if r != nil {
		var books []ShopItem
		p.Categories = append(p.Categories, shopstock.Category[shopStockCandidate, ShopItem]{
			Candidates: func(src shopstock.Source) []shopStockCandidate {
				books = shopBookPool(t, ceiling)
				return shopItemCandidates(append(books, shopScrollPool(t, ceiling, src)...))
			},
			Count: func(src shopstock.Source) int {
				n, _ := shopDrawInclusive(src, 1, 8)
				return n
			},
			Make: func(c shopStockCandidate, _ shopstock.Source) (ShopItem, bool) { return c.item.Clone(), true },
			Finish: func(items []ShopItem, src shopstock.Source) []ShopItem {
				items = shopGuaranteedBooks(items, books)
				return order(append(items, shopPotionStock(t, src)...))
			},
		})
	}
	p.Place = func(stock [][]ShopItem) {
		if extra := modShopStock(t, s.drawSeed, s.restocks, len(armour)); len(extra) != 0 {
			armoury := stock[ShelfArmour]
			stock[ShelfArmour] = order(append(armoury[:len(armoury):len(armoury)], extra...))
		}
	}
	p.Multiplier = shopstock.Multiplier[ShopItem]{
		Factor: int(s.stockMultiplier),
		Source: func() shopstock.Source { return random.NewGo(shopExtraRollSeed(s.drawSeed)) },
		Join: func(_ int, list, more []ShopItem) []ShopItem {
			stack := newShopStacker(list)
			for _, item := range more {
				stack.add(item)
			}
			shopSortShelf(stack.out, t)
			return stack.out
		},
	}
	return p
}

// shopGuaranteedBooks is items with one book of each eligible spell the
// random picks missed, taking no draw (owner policy, DIV-1025).
func shopGuaranteedBooks(items, books []ShopItem) []ShopItem {
	for _, book := range books {
		spell, _ := book.Instance().BookSpell()
		found := false
		for _, item := range items {
			if got, ok := item.Instance().BookSpell(); ok && got == spell {
				found = true
				break
			}
		}
		if !found {
			items = append(items, book.Clone())
		}
	}
	return items
}
