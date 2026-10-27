package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/shopstock"
	"againrom/pkg/sim"
)

// secondShop is the second game's shop as the campaign holds it: each
// location's assortment record, the shop's deal core with the one stock the
// town shops share (R2-ENGINE-374) as its four shelves, and the location
// that stock was drawn for. The customer's places hold stacks of the hero's
// bag, the party's one pack. A capture writes the deal settled, and writes
// neither the stock nor its location (DIV-3045).
type secondShop struct {
	records [4]secondshop.Record
	// Shop holds the stock and the open deal as the one shop item, traded
	// by the second game's rules.
	Shop
	// filled is the location the stock was drawn for; zero before the first
	// fill.
	filled int
	// tables price and code what the deal moves into the bag; the last
	// fill's.
	tables secondshop.Tables

	category int
}

// rules are the second game's buying and selling values (R2-ENGINE-266,
// R2-ENGINE-333, R2-ENGINE-375): an unbounded deal, a buy that takes the
// items before the first one the gold cannot pay (DIV-3065), an unpriced
// item offered returned to the hero, an emptied entry kept as a placeholder
// until the hero leaves, a returned or sold item filed by its record's
// placement and merged by the shelf insert under secondshop.Stacking, and a
// lot joining an equal place by that rule. They become the Shop's rules, so
// the Shop's own deal methods trade by them.
func (s *secondShop) rules() shopRules[ShopItem, ShopShelf] {
	record, tables := s.records[s.filled], s.tables
	place := func(i ShopItem) ShopShelf { return ShopShelf(secondshop.Place(record, secondItem(i))) }
	r := shopRules[ShopItem, ShopShelf]{
		unpricedBack: true, keepEmptied: true, fallback: ShopShelf(len(secondshop.Record{}) - 1),
		count: func(i ShopItem) int64 { return int64(i.Count) },
		price: func(i ShopItem) int64 { return int64(i.Price) },
		withCount: func(i ShopItem, n int64) ShopItem {
			i.Count = int32(n)
			return i
		},
		clone: ShopItem.Clone,
		split: func(i ShopItem, n int64) (ShopItem, ShopItem) {
			moved, rest := i, i
			moved.Count, rest.Count = int32(n), i.Count-int32(n)
			return moved, rest
		},
		home:   func(p ShopPlace) ShopShelf { return place(p.ShopItem) },
		shelve: place,
		// put files i on the shelf; an entry whose bag unit has no table
		// identity to read back leaves the shelf here.
		put: func(items []ShopItem, i ShopItem) []ShopItem {
			var shelf []ShopItem
			for _, e := range items {
				if _, ok := secondShopItem(tables, secondItem(e)); ok {
					shelf = append(shelf, e.Clone())
				}
			}
			if it, ok := secondShopItem(tables, secondItem(i)); ok {
				shelf = shopstock.Insert(shelf, secondshop.Stacking, it)
			}
			return shelf
		},
		join: func(a, b ShopItem) bool {
			return secondshop.Stacking.Merges(secondItem(a), secondItem(b))
		},
	}
	s.trade = &r
	return r
}

// secondShopItem is it as a shelf entry: its bag unit, with the table
// identity that unit reads back as; false when it has no bag form.
func secondShopItem(t secondshop.Tables, it secondshop.Item) (ShopItem, bool) {
	in, ok := t.Instance(it)
	out := shopItemFromInstance(in, it.Count)
	back, _ := secondshop.FromInstance(in)
	out.Table, out.Row, out.Class, out.Material, out.Spell, out.Level = back.Table, back.Row, back.Class, back.Material, back.Spell, back.Level
	return out, ok
}

// secondShopItems are items as shop items; one with no bag form, which the
// hero can never carry, is left out.
func secondShopItems(t secondshop.Tables, items []secondshop.Item) []ShopItem {
	var out []ShopItem
	for _, it := range items {
		if i, ok := secondShopItem(t, it); ok {
			out = append(out, i)
		}
	}
	return out
}

// secondItem is i read back from its bag unit, the only identity a place
// made from the pack carries.
func secondItem(i ShopItem) secondshop.Item {
	it, _ := secondshop.FromInstance(i.Instance())
	it.Count = i.Count
	return it
}

func secondItems(items []ShopItem) []secondshop.Item {
	out := make([]secondshop.Item, 0, len(items))
	for _, i := range items {
		out = append(out, secondItem(i))
	}
	return out
}

// stockOf is a drawn stock on the shelves.
func stockOf(t secondshop.Tables, stock secondshop.Stock) [shopShelfCount][]ShopItem {
	var out [shopShelfCount][]ShopItem
	for k := range stock {
		out[k] = secondShopItems(t, stock[k])
	}
	return out
}

// stock is the shelves as the second game's items.
func (s *secondShop) stock() secondshop.Stock {
	var out secondshop.Stock
	for k := range s.shelves {
		out[k] = secondItems(s.shelves[k])
	}
	return out
}

// buying and selling are the deal's shop items and the hero's offered
// stacks, each in the order they were put down.
func (s *secondShop) buying() []secondshop.Item  { return secondItems(s.side(false)) }
func (s *secondShop) selling() []secondshop.Item { return secondItems(s.side(true)) }

func (s *secondShop) side(mine bool) []ShopItem {
	var out []ShopItem
	for _, p := range s.table {
		if p.Mine == mine {
			out = append(out, p.ShopItem)
		}
	}
	return out
}

func newSecondShop() secondShop { return secondShop{records: secondshop.NewGameRecords} }

// entry is auxiliary entry i of the departure: record 2's categories 0..3,
// then record 3's (R2-ENGINE-330, R2-SESSION-050).
func (s *secondShop) entry(i int) *secondshop.Category { return &s.records[2+i/4][i%4] }

// secondShopMasks are the departures' mask writes (R2-ENGINE-330): by
// mission, the record and each written category's mask.
var secondShopMasks = map[int]struct {
	record int
	masks  map[int]uint32
}{
	30: {2, map[int]uint32{0: 0x1383c22c, 1: 0x2bc3c324, 3: 0x10438324}},
	40: {2, map[int]uint32{0: 0x1387c26c, 1: 0x2bc7c324, 3: 0x10478264}},
	50: {3, map[int]uint32{0: 0x13c7d318, 1: 0x2bc7d318, 2: 0x04000000, 3: 0x2bc042f0}},
	60: {2, map[int]uint32{0: 0x1387c268, 1: 0x2bc7c268, 3: 0x10478260}},
	80: {2, map[int]uint32{0: 0x138752e0, 1: 0x2bc752e0, 3: 0x104702e0}},
	90: {2, map[int]uint32{0: 0x138772c0, 1: 0x2bc772c0, 3: 0x104722c0}},
}

// depart makes the records' writes of the departure from mission n: record
// 3's minima, draws and bounds after mission 50 (R2-SESSION-050) and the
// masks (R2-ENGINE-330).
func (s *secondShop) depart(n int) {
	defer s.staleIfChanged(s.records)
	if n == 50 {
		for i := 4; i < 8; i++ {
			e := s.entry(i)
			e.Min, e.Draws, e.Bound = 499, 100, 2
			if i == 5 || i == 6 {
				e.Draws, e.Bound = 20, 1
			}
		}
		s.entry(6).Min = 0
	}
	if w, ok := secondShopMasks[n]; ok {
		for c, mask := range w.masks {
			s.records[w.record][c].Mask = mask
		}
	}
}

// secondStageMaxima are the maxima each stage case stores, and how many
// auxiliary entries from 0 they cover (R2-SESSION-050, R2-ENGINE-330).
var secondStageMaxima = map[int32]struct {
	value   int32
	entries int
}{
	30: {10000, 4}, 40: {22000, 4}, 50: {60000, 8}, 60: {150000, 8}, 70: {400000, 8},
	80: {800000, 8}, 90: {1500000, 8}, 100: {5000000, 8}, 110: {10000000, 8},
}

// stage makes the records' writes of stage case st: the maxima, and record
// 2's minima of categories 0, 1 and 3 at stages 90 and 100.
func (s *secondShop) stage(st int32) {
	defer s.staleIfChanged(s.records)
	maxima, ok := secondStageMaxima[st]
	if !ok {
		return
	}
	for i := 0; i < maxima.entries; i++ {
		s.entry(i).Max = maxima.value
	}
	if least := map[int32]int32{90: 12000, 100: 40000}[st]; least != 0 {
		for _, i := range []int{0, 1, 3} {
			s.entry(i).Min = least
		}
	}
}

// secondShopClass4 is the class-4 bit the druid inn's NPC 675 topic 78 adds
// to record 3's category 3 (R2-ENGINE-331).
const secondShopClass4 = 1 << 19

// addClass4 is the druid inn conversation's write to record 3's category 3.
func (s *secondShop) addClass4() {
	defer s.staleIfChanged(s.records)
	s.records[3][3].Mask |= secondShopClass4
}

// staleIfChanged marks the stock stale when a write changed the record of
// the location it was drawn for: location 0 is no shop, so the next entry to
// any shop draws again (DIV-3045). before is the records as the write found
// them.
func (s *secondShop) staleIfChanged(before [4]secondshop.Record) {
	if s.filled != 0 && s.records[s.filled] != before[s.filled] {
		s.filled = 0
	}
}

// replayRecords rebuilds the records from a bank: the new game's, then each
// completed mission's departure in ID order, each stage case up to the
// current stage, and the class-4 bit when bank slot 770 holds 2. It reads a
// current SAV this engine wrote before the campaign carried the records.
func replayRecords(bank [1024]int32) [4]secondshop.Record {
	s := newSecondShop()
	for n := 1; n < 128; n++ {
		if bank[896+n] != 0 {
			s.depart(n)
		}
	}
	for st := int32(20); st <= bank[768] && st <= 110; st += 10 {
		s.stage(st)
	}
	if bank[770] == 2 {
		s.records[3][3].Mask |= secondShopClass4
	}
	return s.records
}

// secondShopTables answers the shop's data.bin collections from the
// install's table, and the mods' stocked armour and shields.
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
	for _, m := range t.Mods.Items {
		if it, is := secondshop.Stocked(m.Code, m.Price); m.Stock && is {
			out.Stocked = append(out.Stocked, it)
		}
	}
	return out, ok
}

// enter opens the page at location: the stock is drawn from the location's
// record when its first category is empty (R2-ENGINE-266) or when it was
// drawn for another location (DIV-3045); the first category shows.
func (s *secondShop) enter(tables secondshop.Tables, location int, rnd secondshop.Rand, bag *[]sim.ItemInstance) {
	s.undo(bag)
	s.category, s.tables = 0, tables
	if location < 0 || location >= len(s.records) {
		return
	}
	if len(s.shelves[0]) == 0 || s.filled != location {
		s.shelves, s.filled = stockOf(tables, tables.Fill(s.records[location], rnd)), location
	}
}

// shelf is the category the page shows.
func (s *secondShop) shelf() []secondshop.Item { return secondItems(s.shelves[s.category]) }

// choose shows category c.
func (s *secondShop) choose(c int) bool {
	if c < 0 || c >= len(s.shelves) || c == s.category {
		return false
	}
	s.category = c
	return true
}

// take moves one unit of entry i of the shown category into the purchase
// (R2-ENGINE-375); a unit already taken stays there.
func (s *secondShop) take(i int) bool {
	return s.takeFromShelf(s.rules(), ShopShelf(s.category), i, 1)
}

// offer moves the bag's stack i, every unit of it, into the sale.
func (s *secondShop) offer(bag *[]sim.ItemInstance, i int) bool {
	left, _, stack, ok := secondshop.TakeUnits(*bag, i)
	if !ok {
		return false
	}
	item, ok := secondShopItem(s.tables, stack)
	if !ok || !s.putOnTable(s.rules(), item) {
		return false
	}
	*bag = left
	return true
}

// toBag puts items into the bag as units.
func (s *secondShop) toBag(bag *[]sim.ItemInstance, items []ShopItem) {
	for _, it := range items {
		if units, ok := s.tables.Units(secondItem(it)); ok {
			*bag = append(*bag, units...)
		}
	}
}

// buy settles the purchase against gold (R2-ENGINE-266): the bought items
// enter the bag, the unaffordable rest stays in the deal. A purchase holding
// an item with no bag form is refused whole.
func (s *secondShop) buy(bag *[]sim.ItemInstance, gold *int) bool {
	for _, it := range s.side(false) {
		if _, ok := s.tables.Units(secondItem(it)); !ok {
			return false
		}
	}
	bought, spent, ok := s.shopDeal.buy(s.rules(), int64(*gold))
	if !ok {
		return false
	}
	s.toBag(bag, bought)
	*gold -= int(spent)
	return len(bought) > 0
}

// sell settles the sale (R2-ENGINE-266): each priced item credits its sale
// credit and goes to its category (R2-ENGINE-333); an unpriced one returns
// to the bag.
func (s *secondShop) sell(bag *[]sim.ItemInstance, gold *int) bool {
	paid, back, ok := s.shopDeal.sell(s.rules())
	if !ok {
		return false
	}
	s.toBag(bag, back)
	*gold += int(paid)
	return true
}

// undo ends the deal (R2-ENGINE-266): the offered units return to the bag,
// the shop's items go to their category by placement.
func (s *secondShop) undo(bag *[]sim.ItemInstance) {
	s.toBag(bag, s.clearTable(s.rules()))
}

// leave ends the deal and drops the emptied entries (R2-ENGINE-266).
func (s *secondShop) leave(bag *[]sim.ItemInstance) {
	s.undo(bag)
	s.dropEmptied(s.rules())
}

// restock draws the stock of location anew from its record: the draw a LOAD
// of a save made in a town makes before the shop can be entered. A SAV
// carries no stock; reloading is what restocks the shop.
func (s *secondShop) restock(tables secondshop.Tables, location int, rnd secondshop.Rand) {
	s.table, s.category, s.tables = nil, 0, tables
	if location <= 0 || location >= len(s.records) {
		s.shelves, s.filled = [shopShelfCount][]ShopItem{}, 0
		return
	}
	s.shelves, s.filled = stockOf(tables, tables.Fill(s.records[location], rnd)), location
}

// currentSecondShop is the shop as the SAV writes it: the records. The
// stock is never written: a LOAD draws it anew. Hero is the hero's items as
// a document written before the bag held them; it loads into the bag and is
// written only while that load waits for the install's tables.
type currentSecondShop struct {
	Records [4]secondshop.Record
	Hero    []secondHeroItem `json:",omitempty"`
}

// UnmarshalJSON reads the shop. A document this engine wrote earlier also
// holds the stock and the location it was drawn for; both are read past and
// dropped.
func (c *currentSecondShop) UnmarshalJSON(raw []byte) error {
	var fields struct {
		Records [4]secondshop.Record
		Hero    []secondHeroItem
		Stock   json.RawMessage
		Filled  json.RawMessage
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&fields); err != nil {
		return err
	}
	*c = currentSecondShop{Records: fields.Records, Hero: fields.Hero}
	return nil
}

// maxSecondShopItems bounds the older hero list's entries; its units share
// the bag's bound.
const maxSecondShopItems = 4096

// secondHeroItem is an older hero list item as its document writes it.
type secondHeroItem struct {
	Kind     secondshop.Kind
	Row      int
	Class    int
	Material int
	Spell    int
	Level    int
	Quantity int
	Price    int64
}

// item is h; the bag codes a price at most the cap.
func (h secondHeroItem) item() secondshop.Item {
	return secondshop.Item{Table: h.Kind, Row: h.Row, Class: h.Class, Material: h.Material, Spell: h.Spell, Level: h.Level,
		Count: int32(h.Quantity), Price: int32(min(h.Price, math.MaxInt32))}
}

func captureSecondShop(s secondShop, legacy []secondHeroItem) *currentSecondShop {
	if s.records == secondshop.NewGameRecords && len(legacy) == 0 {
		return nil
	}
	return &currentSecondShop{Records: s.records, Hero: slices.Clone(legacy)}
}

// maxSecondShopDraws and maxSecondShopBound bound a record's draws and
// quantity bound: the stock fill loops over the draws and draws a stack of
// one to the bound plus one. The records the original writes hold at most
// 100 and 2 (R2-ENGINE-263, R2-ENGINE-330).
const (
	maxSecondShopDraws = 1000
	maxSecondShopBound = 100
)

func (c *currentSecondShop) validate() error {
	for _, record := range c.Records {
		for _, cat := range record {
			if cat.Draws < 0 || cat.Draws > maxSecondShopDraws || cat.Bound < 0 || cat.Bound > maxSecondShopBound {
				return fmt.Errorf("current second campaign shop record draws %d with quantity bound %d", cat.Draws, cat.Bound)
			}
		}
	}
	if len(c.Hero) > maxSecondShopItems {
		return fmt.Errorf("current second campaign shop list is too long")
	}
	units := 0
	for _, it := range c.Hero {
		if it.Kind < secondshop.Armor || it.Kind > secondshop.Book || it.Quantity < 0 || it.Price < 0 {
			return fmt.Errorf("current second campaign shop holds an invalid item")
		}
		if units += it.Quantity; units > maxSecondBagUnits {
			return fmt.Errorf("current second campaign shop list holds too many units")
		}
	}
	return nil
}

// restore answers the campaign's shop with no stock; an absent one is a new
// game's.
func (c *currentSecondShop) restore() secondShop {
	if c == nil {
		return newSecondShop()
	}
	return secondShop{records: c.Records}
}

func (c *currentSecondShop) clone() *currentSecondShop {
	if c == nil {
		return nil
	}
	out := *c
	out.Hero = append([]secondHeroItem(nil), c.Hero...)
	return &out
}
