package game

import (
	"bytes"
	"encoding/json"
	"fmt"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
)

// secondShop is the second game's shop as the campaign holds it: each
// location's assortment record, the one stock the town shops share
// (R2-ENGINE-374), the location that stock was drawn for, the hero's items
// the shop deals in (DIV-3043), and the open deal. The deal is the page's:
// a capture writes it undone, and writes neither the stock nor its location
// (DIV-3045).
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

// restock draws the stock of location anew from its record: the draw a LOAD
// of a save made in a town makes before the shop can be entered. A SAV
// carries no stock; reloading is what restocks the shop.
func (s *secondShop) restock(tables secondshop.Tables, location int, rnd secondshop.Rand) {
	s.buying, s.selling, s.category = nil, nil, 0
	if location <= 0 || location >= len(s.records) {
		s.stock, s.filled = secondshop.Stock{}, 0
		return
	}
	s.stock, s.filled = tables.Fill(s.records[location], rnd), location
}

// currentSecondShop is the shop as the SAV writes it: the records and the
// hero's items, with the open deal undone. The stock is never written: a
// LOAD draws it anew.
type currentSecondShop struct {
	Records [4]secondshop.Record
	Hero    []secondshop.Item
}

// UnmarshalJSON reads the shop. A document this engine wrote earlier also
// holds the stock and the location it was drawn for; both are read past and
// dropped.
func (c *currentSecondShop) UnmarshalJSON(raw []byte) error {
	var fields struct {
		Records [4]secondshop.Record
		Hero    []secondshop.Item
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

// maxSecondShopItems bounds the saved list.
const maxSecondShopItems = 4096

func captureSecondShop(s secondShop) *currentSecondShop {
	hero := append([]secondshop.Item(nil), s.hero...)
	for _, it := range s.selling {
		hero = secondshop.Insert(hero, it)
	}
	if s.records == secondshop.NewGameRecords && len(hero) == 0 {
		return nil
	}
	return &currentSecondShop{Records: s.records, Hero: hero}
}

func (c *currentSecondShop) validate() error {
	if len(c.Hero) > maxSecondShopItems {
		return fmt.Errorf("current second campaign shop list is too long")
	}
	for _, it := range c.Hero {
		if it.Kind < secondshop.Armor || it.Kind > secondshop.Book || it.Quantity < 0 || it.Price < 0 {
			return fmt.Errorf("current second campaign shop holds an invalid item")
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
	return secondShop{records: c.Records, hero: append([]secondshop.Item(nil), c.Hero...)}
}

func (c *currentSecondShop) clone() *currentSecondShop {
	if c == nil {
		return nil
	}
	out := *c
	out.Hero = append([]secondshop.Item(nil), c.Hero...)
	return &out
}
