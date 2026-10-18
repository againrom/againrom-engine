package game

// The one deal core both games' shops trade through: four shelves of stock
// and the table the shop's items and the customer's items are put on. What
// differs between two shops is data, a shopRules value: the table's places,
// the buy guard, where an unpriced sale and an emptied shelf entry go, and
// how an item is counted, priced, shelved and merged.

// shopShelfCount is the shelves a shop holds in both games: the first
// game's four shelves, the second game's four categories (R2-ENGINE-374).
const shopShelfCount = 4

// shopPlace is one place of the table: an item, whether the customer owns
// it, and the shelf a shop-owned place came off.
type shopPlace[T any, S ~int] struct {
	ShopItem T
	Mine     bool
	From     S
}

// shopDeal is a shop's stock and open deal.
type shopDeal[T any, S ~int] struct {
	shelves [shopShelfCount][]T
	table   []shopPlace[T, S]
}

// shopRules are one shop's buying and selling values.
type shopRules[T any, S ~int] struct {
	// places bounds the table; zero leaves it unbounded.
	places int
	// wholeBasket is the client's buy guard: a buy is sent only when its
	// total is non-zero and the purse covers all of it. Without it a buy
	// takes the shop's places before the first one the purse cannot pay.
	wholeBasket bool
	// unpricedBack returns an unpriced customer place to the customer when a
	// sale settles; without it the place stays on the table.
	unpricedBack bool
	// keepEmptied keeps a shelf entry taken down to no units as a
	// placeholder until the customer leaves; without it the entry leaves
	// the shelf.
	keepEmptied bool
	// fallback is the shelf an item goes to when its own is out of range.
	fallback S

	count     func(T) int64
	price     func(T) int64
	withCount func(T, int64) T
	clone     func(T) T
	// split divides a place's item into n units moved and the rest.
	split func(T, int64) (moved, rest T)
	// home is the shelf a shop-owned place returns to when the deal is
	// undone; shelve is the shelf a sold item goes to.
	home   func(shopPlace[T, S]) S
	shelve func(T) S
	// put files an item on a shelf: its merge and its order.
	put func([]T, T) []T
	// join, when set, lets a lot put on the table join an equal place of
	// the same side instead of opening a new one.
	join func(a, b T) bool
}

func (d *shopDeal[T, S]) full(r shopRules[T, S]) bool {
	return r.places > 0 && len(d.table) >= r.places
}

// removeFromShelf takes n units of entry i off shelf. The returned item
// holds what moved: n, or the whole entry, which then leaves the shelf or,
// under keepEmptied, stays with no units. An entry with no units refuses
// under keepEmptied.
func (d *shopDeal[T, S]) removeFromShelf(r shopRules[T, S], shelf S, i int, n int64) (T, bool) {
	var zero T
	if n <= 0 || shelf < 0 || int(shelf) >= shopShelfCount || i < 0 || i >= len(d.shelves[shelf]) {
		return zero, false
	}
	items := d.shelves[shelf]
	item := r.clone(items[i])
	have := r.count(item)
	if r.keepEmptied && have < 1 {
		return zero, false
	}
	if n >= have {
		if r.keepEmptied {
			items = append([]T(nil), items...)
			items[i] = r.withCount(items[i], 0)
			d.shelves[shelf] = items
			return item, true
		}
		d.shelves[shelf] = append(items[:i:i], items[i+1:]...)
		return item, true
	}
	items = append([]T(nil), items...)
	items[i] = r.withCount(items[i], have-n)
	d.shelves[shelf] = items
	return r.withCount(item, n), true
}

// takeFromShelf moves n units of entry i onto the table in a new place. A
// full table refuses.
func (d *shopDeal[T, S]) takeFromShelf(r shopRules[T, S], shelf S, i int, n int64) bool {
	if d.full(r) || n <= 0 {
		return false
	}
	moved, ok := d.removeFromShelf(r, shelf, i, n)
	if !ok {
		return false
	}
	d.addPlace(r, shopPlace[T, S]{ShopItem: moved, From: shelf})
	return true
}

// addPlace opens place p, or grows the first place of its side its item
// joins.
func (d *shopDeal[T, S]) addPlace(r shopRules[T, S], p shopPlace[T, S]) {
	if r.join != nil {
		for i := range d.table {
			if d.table[i].Mine == p.Mine && r.join(d.table[i].ShopItem, p.ShopItem) {
				d.growPlace(r, i, r.count(p.ShopItem))
				return
			}
		}
	}
	d.table = append(d.table, p)
}

// putOnTable puts a customer's item on the table in a new place. A full
// table and an empty item refuse.
func (d *shopDeal[T, S]) putOnTable(r shopRules[T, S], item T) bool {
	if d.full(r) || r.count(item) <= 0 {
		return false
	}
	d.addPlace(r, shopPlace[T, S]{ShopItem: r.clone(item), Mine: true})
	return true
}

// growPlace adds n units to place i.
func (d *shopDeal[T, S]) growPlace(r shopRules[T, S], i int, n int64) bool {
	if i < 0 || i >= len(d.table) || n <= 0 {
		return false
	}
	d.table[i].ShopItem = r.withCount(d.table[i].ShopItem, r.count(d.table[i].ShopItem)+n)
	return true
}

// takeOffTable moves up to n units off place i; the returned place holds
// what moved.
func (d *shopDeal[T, S]) takeOffTable(r shopRules[T, S], i int, n int64) (shopPlace[T, S], bool) {
	if i < 0 || i >= len(d.table) || n <= 0 {
		return shopPlace[T, S]{}, false
	}
	place := d.table[i]
	if n >= r.count(place.ShopItem) {
		place.ShopItem = r.clone(place.ShopItem)
		d.table = append(d.table[:i:i], d.table[i+1:]...)
		return place, true
	}
	moved, rest := r.split(place.ShopItem, n)
	place.ShopItem, d.table[i].ShopItem = r.clone(moved), rest
	return place, true
}

// returnToShelf files item on shelf by the shop's put.
func (d *shopDeal[T, S]) returnToShelf(r shopRules[T, S], shelf S, item T) {
	if shelf < 0 || int(shelf) >= shopShelfCount {
		shelf = r.fallback
	}
	d.shelves[shelf] = r.put(d.shelves[shelf], item)
}

// clearTable undoes the deal: the shop's places return to their home
// shelves and the customer's come back to the caller. No coin moves.
func (d *shopDeal[T, S]) clearTable(r shopRules[T, S]) []T {
	var mine []T
	for _, place := range d.table {
		if place.Mine {
			mine = append(mine, r.clone(place.ShopItem))
			continue
		}
		d.returnToShelf(r, r.home(place), place.ShopItem)
	}
	d.table = nil
	return mine
}

// buyTotal is the shop's places' cost, sum of quantity x unit price.
func (d *shopDeal[T, S]) buyTotal(r shopRules[T, S]) int64 {
	var total int64
	for _, place := range d.table {
		if !place.Mine {
			total += r.count(place.ShopItem) * r.price(place.ShopItem)
		}
	}
	return total
}

// sellTotal is the customer's places as the cells show them: sum of
// quantity x (P+1)/2.
func (d *shopDeal[T, S]) sellTotal(r shopRules[T, S]) int64 {
	var total int64
	for _, place := range d.table {
		if place.Mine {
			total += ((r.price(place.ShopItem) + 1) / 2) * r.count(place.ShopItem)
		}
	}
	return total
}

// sellPayout is what a sale credits: ceil(quantity x price / 2) a place.
func (d *shopDeal[T, S]) sellPayout(r shopRules[T, S]) int64 {
	var total int64
	for _, place := range d.table {
		if place.Mine {
			total += shopHalfUp64(r.count(place.ShopItem) * r.price(place.ShopItem))
		}
	}
	return total
}

func shopHalfUp64(n int64) int64 {
	if n < 0 {
		return 0
	}
	return (n + 1) / 2
}

// buy settles the shop's places in table order against purse: each is
// debited quantity x price, and the walk stops at the first one the purse
// left cannot pay, which stays on the table with every place after it. It
// answers the bought items and the coins spent; a refusal changes nothing.
func (d *shopDeal[T, S]) buy(r shopRules[T, S], purse int64) (bought []T, spend int64, ok bool) {
	total := d.buyTotal(r)
	if r.wholeBasket && (total == 0 || purse < total) {
		return nil, 0, false
	}
	merchant := false
	for _, place := range d.table {
		merchant = merchant || !place.Mine
	}
	if !merchant {
		return nil, 0, false
	}
	left := purse
	kept := make([]shopPlace[T, S], 0, len(d.table))
	for k, place := range d.table {
		if place.Mine {
			kept = append(kept, place)
			continue
		}
		cost := r.count(place.ShopItem) * r.price(place.ShopItem)
		if cost > left {
			kept = append(kept, d.table[k:]...)
			break
		}
		left -= cost
		bought = append(bought, place.ShopItem)
	}
	d.table = kept
	return bought, purse - left, true
}

// sell settles the customer's places: each priced place credits
// ceil(quantity x price / 2) and goes to the shelf shelve names; an unpriced
// one comes back to the caller under unpricedBack and otherwise stays.
// Without unpricedBack a sale with nothing priced refuses.
func (d *shopDeal[T, S]) sell(r shopRules[T, S]) (paid int64, back []T, ok bool) {
	mine := false
	for _, place := range d.table {
		mine = mine || place.Mine
	}
	if !mine || !r.unpricedBack && d.sellTotal(r) == 0 {
		return 0, nil, false
	}
	kept := make([]shopPlace[T, S], 0, len(d.table))
	for _, place := range d.table {
		if !place.Mine {
			kept = append(kept, place)
			continue
		}
		if r.price(place.ShopItem) <= 0 {
			if r.unpricedBack {
				back = append(back, r.clone(place.ShopItem))
			} else {
				kept = append(kept, place)
			}
			continue
		}
		paid += shopHalfUp64(r.count(place.ShopItem) * r.price(place.ShopItem))
		d.returnToShelf(r, r.shelve(place.ShopItem), place.ShopItem)
	}
	d.table = kept
	return paid, back, true
}

// dropEmptied removes the shelf entries left with no units, the customer's
// leaving under keepEmptied.
func (d *shopDeal[T, S]) dropEmptied(r shopRules[T, S]) {
	for c := range d.shelves {
		var kept []T
		for _, it := range d.shelves[c] {
			if r.count(it) > 0 {
				kept = append(kept, it)
			}
		}
		d.shelves[c] = kept
	}
}

// settle is the one SAVE policy for an open deal: a SAV is written with
// the deal settled and the live deal stays open. It runs on the capture's
// own copy of the deal and walks the table from its last place: a shop
// place goes back on the shelf it came off, a customer place to its owner
// through back, which may take the place off the table itself.
func (d *shopDeal[T, S]) settle(r shopRules[T, S], back func(index int, item T) error) error {
	for len(d.table) != 0 {
		index := len(d.table) - 1
		place := d.table[index]
		if !place.Mine {
			d.table = d.table[:index]
			shelf := place.From
			if shelf < 0 || int(shelf) >= shopShelfCount {
				shelf = r.fallback
			}
			d.shelves[shelf] = append(d.shelves[shelf], r.clone(place.ShopItem))
			continue
		}
		if err := back(index, place.ShopItem); err != nil {
			return err
		}
		d.table = d.table[:min(index, len(d.table))]
	}
	return nil
}
