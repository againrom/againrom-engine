package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
)

// secondShopItem is it as a shelf entry: its bag unit, with the table
// identity that unit reads back as; false when it has no bag form.
func secondShopItem(t secondshop.Tables, it secondshop.Item) (ShopItem, bool) {
	in, ok := t.Instance(it)
	out := shopItemFromInstance(in, it.Count)
	back, _ := secondshop.FromInstance(in)
	out.Table, out.Row, out.Class, out.Material, out.Spell = back.Table, back.Row, back.Class, back.Material, back.Spell
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
func (s *Shop) stock() secondshop.Stock {
	var out secondshop.Stock
	for k := range s.shelves {
		out[k] = secondItems(s.shelves[k])
	}
	return out
}

// buying and selling are the deal's shop items and the hero's offered
// stacks, each in the order they were put down.
func (s *Shop) buying() []secondshop.Item  { return secondItems(s.side(false)) }
func (s *Shop) selling() []secondshop.Item { return secondItems(s.side(true)) }

func (s *Shop) side(mine bool) []ShopItem {
	var out []ShopItem
	for _, p := range s.table {
		if p.Mine == mine {
			out = append(out, p.ShopItem)
		}
	}
	return out
}

func newSecondShop() Shop {
	return newShopState(&bankShopProfile, shopRecordRows(secondshop.NewGameRecords))
}

const secondShopClass4 = 1 << 19

func (s *Shop) depart(n int)   { s.patch(s.profile().departures, int32(n)) }
func (s *Shop) stage(st int32) { s.patch(s.profile().stages, st) }
func (s *Shop) addClass4()     { s.patch(s.profile().unlocks, 0) }

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
	return s.nativeRecords()
}

// secondShopTables answers the shop's data.bin collections from the
// install's table, and the mods' stocked armour and shields.
func secondShopTables(t *mapload.Table) (secondshop.Tables, bool) {
	if t == nil {
		return secondshop.Tables{}, false
	}
	var out secondshop.Tables
	out.ItemRecords = t.ItemRecords
	out.TemplateCode = func(code uint16) uint16 { return uint16(itemTemplateCode(t, code)) }
	ok := true
	for _, x := range []struct {
		dst *secondshop.Collection
		src any
	}{{&out.Shapes, t.Shapes}, {&out.Materials, t.Materials}, {&out.Armors, t.Armors}, {&out.Shields, t.Shields},
		{&out.Weapons, t.Weapons}, {&out.MagicItems, t.MagicItems}, {&out.Spells, t.Spells}, {&out.Magic, t.Magic}} {
		c, is := x.src.(secondshop.Collection)
		*x.dst, ok = c, ok && is && c != nil
	}
	for _, m := range modStock(t, secondModStock) {
		out.Stocked = append(out.Stocked, m.Item)
	}
	return out, ok
}

// secondDeathItems is the second game's death item draw over t's tables:
// the shop draw of the death's category (R2-ENGINE-941). Tables without the
// item collections draw no list.
func secondDeathItems(t *mapload.Table) sim.DeathItemDraw {
	return func(roll sim.DeathItemRoll, rnd func() int) []sim.ItemInstance {
		tables, ok := secondShopTables(t)
		if !ok {
			return nil
		}
		return tables.DeathItems(roll.Min, roll.Max, roll.Mask, rnd)
	}
}

// ShopTables exposes the installed shop profile to headless instruments.
func (f *FrontEnd) ShopTables() (secondshop.Tables, bool) { return secondShopTables(f.Table) }

// enter opens the page at location: the stock is drawn from the location's
// record when its first category is empty (R2-ENGINE-266) or when it was
// drawn for another location (DIV-3045); the first category shows.
func (s *Shop) enter(tables secondshop.Tables, location int, rnd secondshop.Rand, bag *[]sim.ItemInstance) {
	p := s.profile()
	if p.undoOnEntry {
		s.undo(bag)
	}
	s.category, s.tables, s.location = 0, tables, location
	if location < 0 || s.records[location] == nil {
		return
	}
	if p.refillOnEntry && (len(s.shelves[0]) == 0 || s.filled != location) {
		s.generate(nil, shopRawDraw(rnd))
	}
}

func (s *Shop) shownShelf() []secondshop.Item {
	return secondItems(s.Shelf(ShopShelf(s.category)))
}

// choose shows category c.
func (s *Shop) choose(c int) bool {
	if c < 0 || c >= len(s.shelves) || c == s.category {
		return false
	}
	s.category = c
	return true
}

// take moves one unit of entry i of the shown category into the purchase
// (R2-ENGINE-375); a unit already taken stays there.
func (s *Shop) take(i int) bool {
	return s.takeFromShelf(s.rules(), ShopShelf(s.category), i, 1)
}

// offer moves the bag's stack i, every unit of it, into the sale.
func (s *Shop) offer(bag *[]sim.ItemInstance, i int) bool {
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
func (s *Shop) toBag(bag *[]sim.ItemInstance, items []ShopItem) {
	for _, it := range items {
		if units, ok := s.tables.Units(secondItem(it)); ok {
			*bag = append(*bag, units...)
		}
	}
}

// buyBag settles an affordable basket into the bag. A short purse or an item
// with no bag form refuses the whole purchase.
func (s *Shop) buyBag(bag *[]sim.ItemInstance, gold *int) bool {
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
func (s *Shop) sellBag(bag *[]sim.ItemInstance, gold *int) bool {
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
func (s *Shop) undo(bag *[]sim.ItemInstance) {
	s.toBag(bag, s.clearTable(s.rules()))
}

// leave ends the deal and drops the emptied entries (R2-ENGINE-266).
func (s *Shop) leave(bag *[]sim.ItemInstance) {
	s.undo(bag)
	s.dropEmptied(s.rules())
}

// drawStock draws the stock of location anew from its record. A SAV carries
// no stock.
func (s *Shop) drawStock(tables secondshop.Tables, location int, rnd secondshop.Rand) {
	s.category, s.tables, s.location = 0, tables, location
	s.generate(nil, shopRawDraw(rnd))
}

// shopArrival is what a town arrival's refill reads: the install's shop
// tables, the launch's stock multiplier, the town arrived at and the shop
// stock stream.
type shopArrival struct {
	tables     secondshop.Tables
	multiplier int32
	location   int
	rnd        secondshop.Rand
}

// refill is the second game's arrival refill (R2-ENGINE-470,
// R2-ENGINE-471): every category is emptied and drawn anew from the record
// of the town arrived at; an open deal keeps the stock.
func (s *Shop) refill(a shopArrival) bool {
	if s.dealHolds() {
		return false
	}
	s.stockMultiplier = a.multiplier
	s.drawStock(a.tables, a.location, a.rnd)
	return true
}

// refillShop sends a town arrival's refill to the campaign's shop: the
// travel's end at a town and a LOAD in a town, before the shop can be
// entered. An arrival at a mission sends none. The refill is read from the
// shop's own policy, whose bindings reach the world map that calls this.
func (c *secondCampaign) refillShop(a shopArrival) bool {
	p := c.shop.policy
	if c.current.kind != placeTown || p == nil || p.arrivalRefill == nil {
		return false
	}
	a.location = c.current.id
	return p.arrivalRefill(&c.shop, a)
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

// item is h; the bag codes a price at most the cap. An equipment item's
// spell at a level above 0 is its castspell effect.
func (h secondHeroItem) item() secondshop.Item {
	it := secondshop.Item{Table: h.Kind, Row: h.Row, Class: h.Class, Material: h.Material,
		Count: int32(h.Quantity), Price: int32(min(h.Price, math.MaxInt32))}
	switch h.Kind {
	case secondshop.Book:
		it.Spell = h.Spell
	case secondshop.Armor, secondshop.Shield, secondshop.Weapon:
		if h.Level > 0 {
			it.Effects = []sim.ItemEffect{castSpellEffect(uint16(h.Spell), int32(h.Level))}
		}
	}
	return it
}

func captureSecondShop(s Shop, legacy []secondHeroItem) *currentSecondShop {
	if s.nativeRecords() == secondshop.NewGameRecords && len(legacy) == 0 {
		return nil
	}
	return &currentSecondShop{Records: s.nativeRecords(), Hero: slices.Clone(legacy)}
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
func (c *currentSecondShop) restore() Shop {
	if c == nil {
		return newSecondShop()
	}
	return newShopState(&bankShopProfile, shopRecordRows(c.Records))
}

func (c *currentSecondShop) clone() *currentSecondShop {
	if c == nil {
		return nil
	}
	out := *c
	out.Hero = append([]secondHeroItem(nil), c.Hero...)
	return &out
}
