package secondshop

// Buy settles a pending purchase list against gold (R2-ENGINE-266): it walks
// the list in order, stops at the first item whose q*P exceeds the gold left,
// and debits q*P for each item before it. It answers the bought items, the
// items left pending and the gold after the deal.
func Buy(pending []Item, gold int64) (bought, left []Item, after int64) {
	for i, it := range pending {
		cost := it.Price * int64(it.Quantity)
		if cost > gold {
			return bought, pending[i:], gold
		}
		gold -= cost
		bought = append(bought, it)
	}
	return bought, nil, gold
}

// Sell settles a pending sale list (R2-ENGINE-266): each item with a nonzero
// price credits floor(q*P/2+0.5) and goes to its shelf by Place.
func Sell(s Stock, r Record, sold []Item, gold int64) (Stock, int64) {
	for _, it := range sold {
		if it.Price == 0 {
			continue
		}
		gold += SaleCredit(it.Price, it.Quantity)
		c := Place(r, it)
		s[c] = Insert(s[c], it)
	}
	return s, gold
}

// Take moves n units of entry i of category c into a deal (R2-ENGINE-375):
// the shelf keeps Q-n and the taken clone holds n. n is clamped to 1..Q.
func Take(s Stock, c, i, n int) (Stock, Item, bool) {
	if c < 0 || c >= len(s) || i < 0 || i >= len(s[c]) || s[c][i].Quantity < 1 {
		return s, Item{}, false
	}
	q := s[c][i].Quantity
	n = max(1, min(n, q))
	it := s[c][i]
	it.Quantity = n
	s[c] = append([]Item(nil), s[c]...)
	s[c][i].Quantity = q - n
	return s, it, true
}

// Leave drops the emptied entries when the customer leaves (R2-ENGINE-266).
func Leave(s Stock) Stock {
	for c := range s {
		var kept []Item
		for _, it := range s[c] {
			if it.Quantity > 0 {
				kept = append(kept, it)
			}
		}
		s[c] = kept
	}
	return s
}

// kindBit is the mask kind bit of an item's table (R2-ENGINE-333).
func kindBit(it Item) uint32 {
	switch it.Kind {
	case Armor:
		return kindArmor
	case Shield:
		return kindShield
	case Weapon:
		return kindWeapon
	}
	return kindMagic
}

// Place is the category a sold or returned item goes to (R2-ENGINE-333): a
// non-stackable non-book to the first enchanted category, else category 3; a
// stackable item or a book to the first category of its kind without the
// enchanted bit, else the first of its kind, else category 3.
func Place(r Record, it Item) int {
	const last = len(Record{}) - 1
	if !it.Stackable() && it.Kind != Book {
		for c, cat := range r {
			if cat.Mask&bitEnchanted != 0 {
				return c
			}
		}
		return last
	}
	bit := kindBit(it)
	for c, cat := range r {
		if cat.Mask&bit != 0 && cat.Mask&bitEnchanted == 0 {
			return c
		}
	}
	for c, cat := range r {
		if cat.Mask&bit != 0 {
			return c
		}
	}
	return last
}
