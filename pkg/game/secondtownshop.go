package game

import (
	"fmt"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
)

// secondShop is the second game's shop as the campaign holds it: each
// location's assortment record, the one stock the town shops share
// (R2-ENGINE-374), the location that stock was drawn for, the hero's items
// the shop deals in (DIV-3043), and the open deal. The deal is the page's:
// a capture writes it undone (DIV-3045).
type secondShop struct {
	records [4]secondshop.Record
	stock   secondshop.Stock
	// filled is the location the stock was drawn for; zero before the first
	// fill.
	filled int
	hero   []secondshop.Item

	category        int
	buying, selling []secondshop.Item
}

func newSecondShop() secondShop { return secondShop{records: secondshop.NewGameRecords} }

// secondShopTables answers the shop's data.bin collections from the
// install's table.
func secondShopTables(t *mapload.Table) (secondshop.Tables, bool) {
	if t == nil {
		return secondshop.Tables{}, false
	}
	var out secondshop.Tables
	ok := true
	for _, x := range []struct {
		dst *secondshop.Collection
		src any
	}{{&out.Shapes, t.Shapes}, {&out.Materials, t.Materials}, {&out.Armors, t.Armors}, {&out.Shields, t.Shields},
		{&out.Weapons, t.Weapons}, {&out.MagicItems, t.MagicItems}, {&out.Spells, t.Spells}} {
		c, is := x.src.(secondshop.Collection)
		*x.dst, ok = c, ok && is && c != nil
	}
	return out, ok
}

// enter opens the page at location: the stock is drawn from the location's
// record when its first category is empty (R2-ENGINE-266) or when it was
// drawn for another location (DIV-3045); the first category shows.
func (s *secondShop) enter(tables secondshop.Tables, location int, rnd secondshop.Rand) {
	s.undo()
	s.category = 0
	if location < 0 || location >= len(s.records) {
		return
	}
	if len(s.stock[0]) == 0 || s.filled != location {
		s.stock, s.filled = tables.Fill(s.records[location], rnd), location
	}
}

// shelf is the category the page shows.
func (s *secondShop) shelf() []secondshop.Item { return s.stock[s.category] }

// choose shows category c.
func (s *secondShop) choose(c int) bool {
	if c < 0 || c >= len(s.stock) || c == s.category {
		return false
	}
	s.category = c
	return true
}

// take moves one unit of entry i of the shown category into the purchase
// (R2-ENGINE-375); a unit already taken stays there.
func (s *secondShop) take(i int) bool {
	stock, it, ok := secondshop.Take(s.stock, s.category, i, 1)
	if !ok {
		return false
	}
	s.stock = stock
	s.buying = secondshop.Insert(s.buying, it)
	return true
}

// offer moves the hero's item i, its whole stack, into the sale.
func (s *secondShop) offer(i int) bool {
	if i < 0 || i >= len(s.hero) {
		return false
	}
	it := s.hero[i]
	s.hero = append(s.hero[:i:i], s.hero[i+1:]...)
	s.selling = append(s.selling, it)
	return true
}

// buy settles the purchase against gold (R2-ENGINE-266): the bought items
// enter the hero's items, the unaffordable rest stays in the deal.
func (s *secondShop) buy(gold *int) bool {
	if len(s.buying) == 0 {
		return false
	}
	bought, left, after := secondshop.Buy(s.buying, int64(*gold))
	for _, it := range bought {
		s.hero = secondshop.Insert(s.hero, it)
	}
	s.buying, *gold = left, int(after)
	return len(bought) > 0
}

// sell settles the sale (R2-ENGINE-266): each priced item credits its sale
// credit and goes to its category (R2-ENGINE-333).
func (s *secondShop) sell(gold *int) bool {
	if len(s.selling) == 0 {
		return false
	}
	var unpriced []secondshop.Item
	for _, it := range s.selling {
		if it.Price == 0 {
			unpriced = append(unpriced, it)
		}
	}
	stock, after := secondshop.Sell(s.stock, s.records[s.filled], s.selling, int64(*gold))
	s.stock, *gold, s.selling = stock, int(after), nil
	for _, it := range unpriced {
		s.hero = secondshop.Insert(s.hero, it)
	}
	return true
}

// undo ends the deal (R2-ENGINE-266): the hero's items return to the hero,
// the shop's go to their category by placement.
func (s *secondShop) undo() {
	for _, it := range s.selling {
		s.hero = secondshop.Insert(s.hero, it)
	}
	for _, it := range s.buying {
		c := secondshop.Place(s.records[s.filled], it)
		s.stock[c] = secondshop.Insert(s.stock[c], it)
	}
	s.buying, s.selling = nil, nil
}

// leave ends the deal and drops the emptied entries (R2-ENGINE-266).
func (s *secondShop) leave() {
	s.undo()
	s.stock = secondshop.Leave(s.stock)
}

// currentSecondShop is the shop as the SAV writes it: the records, the
// stock, the location it was drawn for and the hero's items, with the open
// deal undone.
type currentSecondShop struct {
	Records [4]secondshop.Record
	Stock   secondshop.Stock
	Filled  int
	Hero    []secondshop.Item
}

// maxSecondShopItems bounds each saved list.
const maxSecondShopItems = 4096

func captureSecondShop(s secondShop) *currentSecondShop {
	s.stock = cloneSecondStock(s.stock)
	s.hero = append([]secondshop.Item(nil), s.hero...)
	s.undo()
	if s.records == secondshop.NewGameRecords && s.filled == 0 && len(s.hero) == 0 && s.empty() {
		return nil
	}
	return &currentSecondShop{Records: s.records, Stock: s.stock, Filled: s.filled, Hero: s.hero}
}

// empty reports whether no category holds an entry.
func (s secondShop) empty() bool {
	for _, list := range s.stock {
		if len(list) > 0 {
			return false
		}
	}
	return true
}

func cloneSecondStock(s secondshop.Stock) secondshop.Stock {
	for c := range s {
		s[c] = append([]secondshop.Item(nil), s[c]...)
	}
	return s
}

func (c *currentSecondShop) validate() error {
	if c.Filled < 0 || c.Filled >= len(c.Records) || len(c.Hero) > maxSecondShopItems {
		return fmt.Errorf("current second campaign shop is out of range")
	}
	lists := append([][]secondshop.Item{c.Hero}, c.Stock[:]...)
	for _, list := range lists {
		if len(list) > maxSecondShopItems {
			return fmt.Errorf("current second campaign shop list is too long")
		}
		for _, it := range list {
			if it.Kind < secondshop.Armor || it.Kind > secondshop.Book || it.Quantity < 0 || it.Price < 0 {
				return fmt.Errorf("current second campaign shop holds an invalid item")
			}
		}
	}
	return nil
}

// restore answers the campaign's shop; an absent one is a new game's.
func (c *currentSecondShop) restore() secondShop {
	if c == nil {
		return newSecondShop()
	}
	return secondShop{records: c.Records, stock: cloneSecondStock(c.Stock), filled: c.Filled,
		hero: append([]secondshop.Item(nil), c.Hero...)}
}

func (c *currentSecondShop) clone() *currentSecondShop {
	if c == nil {
		return nil
	}
	out := *c
	out.Stock = cloneSecondStock(c.Stock)
	out.Hero = append([]secondshop.Item(nil), c.Hero...)
	return &out
}
