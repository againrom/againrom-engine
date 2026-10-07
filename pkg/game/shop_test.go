package game

import (
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// shopRow is one row of a synthetic item collection: a name, a price cell, an
// equipment slot for the armour class rule, and one material mask per shape.
type shopRow struct {
	name  string
	price int32
	slot  int32
	masks [data.ShopShapes]uint16

	// suit is the row's sutableFor cell, 0 meaning unstated — see sutableFor
	// below.
	suit int32
}

type shopCollection []shopRow

func (c shopCollection) Len() int                    { return len(c) }
func (c shopCollection) EntryName(i int) string      { return c[i].name }
func (c shopCollection) EntryStrings(i int) []string { return nil }

func (c shopCollection) EntryParams(i int) []int32 {
	p := make([]int32, data.SutableForColumn+1)
	for j := range p {
		p[j] = -1
	}
	p[2] = c[i].price
	p[4] = c[i].slot
	p[data.SutableForColumn] = c[i].sutableFor()
	return p
}

// sutableFor is the row's own wear-rule cell. THE ZERO VALUE IS 3, usable by
// fighter and mage alike: a fixture row that does not state the column is
// one whose subject is not the wear rule, and 3 leaves it drawn and worn
// exactly as it was before 0162. A row shorter than the column, or one
// holding 0, reads as usable by NEITHER, so the default cannot be silence.
func (r shopRow) sutableFor() int32 {
	if r.suit == 0 {
		return 3
	}
	return r.suit
}

func (c shopCollection) EntryRaw(i int) []byte {
	if c[i].name == "" {
		return nil
	}
	raw := make([]byte, 2*data.ShopShapes)
	for s, m := range c[i].masks {
		binary.LittleEndian.PutUint16(raw[2*s:], m)
	}
	return raw
}

// unitScale is a ScaleTable whose every factor is 1, so a row's price cell is
// also the item's price and a test states one number instead of three.
type unitScale int

func (t unitScale) Len() int               { return int(t) }
func (t unitScale) EntryName(i int) string { return "" }

func (t unitScale) EntryDoubles(i int) []float64 {
	d := make([]float64, 9)
	for j := range d {
		d[j] = 1
	}
	return d
}

// shopMagicCollection makes kind 2 a one-point ordinary effect for every
// equipment slot and both classes. The legacy shop fixture can therefore
// exercise the now-enchanted Magic shelf without depending on Data.bin.
type shopMagicCollection int

func (c shopMagicCollection) Len() int                    { return int(c) }
func (c shopMagicCollection) EntryName(i int) string      { return "effect" }
func (c shopMagicCollection) EntryStrings(i int) []string { return nil }
func (c shopMagicCollection) EntryParams(i int) []int32 {
	p := make([]int32, 4+24)
	if i == 2 {
		p[0], p[1], p[2] = 1, 1, 1
		for k := 4; k < len(p); k++ {
			p[k] = 1
		}
	}
	return p
}

// shopTable is an item table holding one weapon row and one armour row, each
// admitting two materials at shape 0.
func shopTable() *mapload.Table {
	return &mapload.Table{
		Shapes:    unitScale(data.ShopShapes),
		Materials: unitScale(data.ShopMaterials),
		Weapons:   shopCollection{{}, {name: "axe", price: 40, masks: [data.ShopShapes]uint16{1<<0 | 1<<1}}},
		Armors:    shopCollection{{}, {name: "helm", price: 25, slot: 5, masks: [data.ShopShapes]uint16{1<<2 | 1<<3}}},
		Shields:   shopCollection{{}, {name: "targe", price: 15, slot: 2, masks: [data.ShopShapes]uint16{1 << 4}}},
		Magic:     shopMagicCollection(3),
	}
}

// AC-3: one seed gives one assortment, two seeds give two, and a second
// generation replaces the shelves rather than appending to them.
func TestShopGenerationIsSeededAndClearsBeforeItFills(t *testing.T) {
	table := shopTable()

	a := NewShop(1000)
	a.Generate(table, 7)
	b := NewShop(1000)
	b.Generate(table, 7)
	if !reflect.DeepEqual(a.Shelf(ShelfWeapons), b.Shelf(ShelfWeapons)) {
		t.Error("one seed gave two different weapon shelves")
	}

	c := NewShop(1000)
	c.Generate(table, 8)
	if reflect.DeepEqual(a.Shelf(ShelfWeapons), c.Shelf(ShelfWeapons)) {
		t.Error("two seeds gave the same weapon shelf")
	}

	// THE DRAW COUNT IS IN UNITS, NOT IN ELEMENTS, since `DIV-322`. A shelf
	// folds its repeats, so the number of elements depends on how many
	// distinct code+price pairs the pool happened to yield; what the
	// generator promises is that it drew shopShelfDraws[shelf] times, and
	// that is the sum of the counts.
	want := [numShopShelves]int{100, 100, 20}
	for _, gen := range []int{1, 2} {
		for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
			var units int
			for _, item := range a.Shelf(shelf) {
				units += int(item.Count)
			}
			if units != want[shelf] {
				t.Errorf("generation %d: shelf %v holds %d units, want %d", gen, shelf, units, want[shelf])
			}
			if len(a.Shelf(shelf)) > units {
				t.Errorf("generation %d: shelf %v holds %d elements over %d units",
					gen, shelf, len(a.Shelf(shelf)), units)
			}
		}
		a.Generate(table, 7)
	}

	// Every drawn element carries the row's own price and at least one unit,
	// and the shelf really did fold: this fixture's weapon pool is far
	// narrower than the 100 draws, so a shelf of 100 elements would mean
	// shopStackShelf never ran.
	weapons := a.Shelf(ShelfWeapons)
	var units int
	for _, item := range weapons {
		if item.Count < 1 || item.Price != 40 {
			t.Fatalf("drawn weapon = %+v, want at least one unit at price 40", item)
		}
		units += int(item.Count)
	}
	if len(weapons) >= units {
		t.Fatalf("the weapon shelf holds %d elements over %d units, want repeats folded", len(weapons), units)
	}
}

func TestShopRestockRefusesToDestroyAStagedTrade(t *testing.T) {
	table := shopTable()
	shop := NewShop(1000)
	shop.Generate(table, 7)
	if !shop.TakeFromShelf(ShelfWeapons, 0, 1) {
		t.Fatal("fixture could not stage a merchant weapon")
	}
	beforeTable := shop.Table()
	beforeShelf := shop.Shelf(ShelfWeapons)
	if shop.Restock(table, 99) {
		t.Fatal("restock accepted a live trade table")
	}
	if shop.restocks != 0 || !reflect.DeepEqual(shop.Table(), beforeTable) || !reflect.DeepEqual(shop.Shelf(ShelfWeapons), beforeShelf) {
		t.Fatalf("refused restock mutated shop: ordinal %d table %+v shelf %+v", shop.restocks, shop.Table(), shop.Shelf(ShelfWeapons))
	}
}

func TestShopWithNoTablesStocksNothing(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *mapload.Table
	}{
		{"nil table", nil},
		{"no collections", &mapload.Table{}},
		{"collections that carry no raw block", &mapload.Table{
			Shapes:    unitScale(data.ShopShapes),
			Materials: unitScale(data.ShopMaterials),
			Weapons:   plainCollection{},
		}},
	} {
		s := NewShop(1000)
		s.Generate(tc.table, 1)
		for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
			if got := len(s.Shelf(shelf)); got != 0 {
				t.Errorf("%s: shelf %v holds %d items", tc.name, shelf, got)
			}
		}
	}
}

// plainCollection satisfies data.Collection and NOT data.MaskTable, which is
// plan R-1's own risk: the assertion fails and the shelf goes empty.
type plainCollection struct{}

func (plainCollection) Len() int                    { return 2 }
func (plainCollection) EntryName(i int) string      { return "axe" }
func (plainCollection) EntryParams(i int) []int32   { return []int32{1, 2, 3} }
func (plainCollection) EntryStrings(i int) []string { return nil }

// shelfUnits is the shelf's total in units rather than in elements. Since
// `DIV-322` an element can hold several units, so a test about what a take
// removed counts units; an assertion on len() would be answered by the
// element the take only shortened.
func shelfUnits(s *Shop, shelf ShopShelf) int {
	var n int
	for _, item := range s.Shelf(shelf) {
		n += int(item.Count)
	}
	return n
}

// AC-4: a take moves units off the shelf and stamps them, and the table
// refuses a sixth place.
func TestTheTableHoldsFivePlacesAndStampsEachSide(t *testing.T) {
	s := NewShop(1000)
	s.Generate(shopTable(), 3)
	before := shelfUnits(s, ShelfWeapons)

	if !s.TakeFromShelf(ShelfWeapons, 0, 1) {
		t.Fatal("taking the first weapon off the shelf was refused")
	}
	if got := shelfUnits(s, ShelfWeapons); got != before-1 {
		t.Errorf("shelf holds %d units after one take, want %d", got, before-1)
	}
	if place := s.Table()[0]; place.Mine {
		t.Error("an item off the shelf was stamped as the player's")
	}

	if !s.PutOnTable(ShopItem{Code: 0x1234, Price: 9, Count: 2}) {
		t.Fatal("putting a pack item on the table was refused")
	}
	if place := s.Table()[1]; !place.Mine || place.Count != 2 {
		t.Errorf("player's place = %+v, want his, count 2", place)
	}

	for len(s.Table()) < ShopTablePlaces {
		if !s.TakeFromShelf(ShelfWeapons, 0, 1) {
			t.Fatal("the table refused a place below five")
		}
	}
	shelf := shelfUnits(s, ShelfWeapons)
	if s.TakeFromShelf(ShelfWeapons, 0, 1) {
		t.Error("the table took a sixth place")
	}
	if len(s.Table()) != ShopTablePlaces || shelfUnits(s, ShelfWeapons) != shelf {
		t.Error("the refused take changed the table or the shelf")
	}
	if s.PutOnTable(ShopItem{Code: 1, Price: 1, Count: 1}) {
		t.Error("the table took a sixth place from the pack")
	}
}

// TakeFromShelf IS APPEND-ONLY AND STAYS THAT WAY: two direct takes of one
// code open two places, because the primitive never calls a find rule at all.
// The merge decision belongs to the caller, exactly as PutOnTable's does —
// shopFromShelf is the caller that makes it, through shopFindHisPlace
// (shoproom.go, `DIV-322`), and TestTheRoomJoinsASecondUnitToTheSamePlace
// witnesses that arm. This test witnesses the primitive underneath it, so
// that the room's rule can be changed without the primitive silently
// acquiring one.
//
// Each place keeps its own From, which is what lets ClearTable return each to
// its own shelf; a merged place could carry only one From and would return
// the wrong count, or none, to whichever shelf lost its stamp.
func TestTwoShelfElementsOfTheSameCodeDoNotMerge(t *testing.T) {
	s := &Shop{ceiling: 1000}
	s.shelves[ShelfWeapons] = []ShopItem{
		{Code: 0x0101, Price: 40, Count: 1},
		{Code: 0x0101, Price: 40, Count: 1},
	}
	s.shelves[ShelfArmour] = []ShopItem{{Code: 0x0201, Price: 25, Count: 1}}

	if !s.TakeFromShelf(ShelfWeapons, 0, 1) {
		t.Fatal("the first take off the weapon shelf was refused")
	}
	if !s.TakeFromShelf(ShelfWeapons, 0, 1) {
		t.Fatal("the second take off the weapon shelf was refused")
	}
	places := s.Table()
	if len(places) != 2 {
		t.Fatalf("the table holds %+v, want two SEPARATE places of the identical code, not one merged", places)
	}
	for i, p := range places {
		if p.Mine || p.Code != 0x0101 || p.Price != 40 || p.Count != 1 || p.From != ShelfWeapons {
			t.Errorf("place %d = %+v, want the merchant's own one-unit weapon-shelf stamp", i, p)
		}
	}

	// Each keeps its OWN From: clearing returns both to the weapon shelf, not
	// one of them to whatever a merged place's single stamp would have named.
	// The two units come back to ONE element, because returnToShelf merges
	// into an equal one (`DIV-322`, and SHOP-DUP-028 says the original's
	// return path does the same); the count is what carries them, so this
	// asks for units.
	back := s.ClearTable()
	if len(back) != 0 {
		t.Fatalf("the clear gave back %+v, want nothing — both places are the merchant's", back)
	}
	if got := shelfUnits(s, ShelfWeapons); got != 2 {
		t.Errorf("the weapon shelf holds %d units after the clear, want both takes returned", got)
	}
	if got := len(s.Shelf(ShelfArmour)); got != 1 {
		t.Errorf("the armour shelf holds %d, want its own untouched stock", got)
	}
}

// RemoveFromShelf takes one element straight off the shelf, without a table
// place: the shelf-to-doll purchase (1005 round 2, `DIV-087`) needs a shelf
// element gone and nothing else touched, and TakeFromShelf's own table-place
// side effect would be a second, unwanted mutation for that caller to undo.
func TestRemoveFromShelfTakesOneElementAndShiftsTheRest(t *testing.T) {
	s := &Shop{ceiling: 1000}
	s.shelves[ShelfWeapons] = []ShopItem{
		{Code: 0x0101, Price: 40, Count: 1},
		{Code: 0x0102, Price: 41, Count: 1},
		{Code: 0x0103, Price: 42, Count: 1},
	}

	item, ok := s.RemoveFromShelf(ShelfWeapons, 1, 1)
	if !ok || item.Code != 0x0102 {
		t.Fatalf("RemoveFromShelf(1) = %+v, %v, want the middle element", item, ok)
	}
	left := s.Shelf(ShelfWeapons)
	want := []ShopItem{{Code: 0x0101, Price: 40, Count: 1}, {Code: 0x0103, Price: 42, Count: 1}}
	if !reflect.DeepEqual(left, want) {
		t.Errorf("shelf after the remove = %+v, want %+v", left, want)
	}
	if len(s.Table()) != 0 {
		t.Error("RemoveFromShelf opened a table place — it must open none")
	}
}

// TestRemoveFromShelfRefusesAnOutOfRangeIndex is TakeFromShelf's own bounds
// refusal (shop.go), restated for the new method: an index outside the
// shelf's own length leaves it untouched and answers false.
func TestRemoveFromShelfRefusesAnOutOfRangeIndex(t *testing.T) {
	s := &Shop{ceiling: 1000}
	s.shelves[ShelfArmour] = []ShopItem{{Code: 0x0201, Price: 25, Count: 1}}

	for _, i := range []int{-1, 1, 5} {
		if _, ok := s.RemoveFromShelf(ShelfArmour, i, 1); ok {
			t.Errorf("RemoveFromShelf(%d) succeeded, want a refusal", i)
		}
	}
	if len(s.Shelf(ShelfArmour)) != 1 {
		t.Error("a refused remove changed the shelf")
	}
}

// AC-5: clearing puts every place back where its stamp says, and moves no coin.
func TestClearingTheTableSendsEveryPlaceHome(t *testing.T) {
	s := NewShop(1000)
	s.Generate(shopTable(), 4)
	shelf := len(s.Shelf(ShelfWeapons))

	s.TakeFromShelf(ShelfWeapons, 0, 1)
	s.TakeFromShelf(ShelfWeapons, 0, 1)
	mine := ShopItem{Code: 0x0155, Price: 9, Count: 2}
	s.PutOnTable(mine)

	back := s.ClearTable()
	if len(s.Table()) != 0 {
		t.Errorf("the table still holds %d places", len(s.Table()))
	}
	if len(s.Shelf(ShelfWeapons)) != shelf {
		t.Errorf("shelf holds %d after the clear, want %d", len(s.Shelf(ShelfWeapons)), shelf)
	}
	if !reflect.DeepEqual(back, []ShopItem{mine}) {
		t.Errorf("the clear gave back %+v, want the player's one element", back)
	}
}

// AC-6: the guard refuses a basket above the purse and changes nothing; a
// covered basket debits exactly the sum and leaves the player's places.
func TestBuyRefusesWhatThePurseCannotCoverAndDebitsExactlyWhatItCan(t *testing.T) {
	s := NewShop(1000)
	s.Generate(shopTable(), 5)
	s.TakeFromShelf(ShelfWeapons, 0, 1)
	s.TakeFromShelf(ShelfWeapons, 0, 1)
	s.PutOnTable(ShopItem{Code: 0x0155, Price: 9, Count: 1})

	total := s.BuyTotal()
	if total != 80 {
		t.Fatalf("buy total = %d, want two 40-coin weapons", total)
	}
	shelf, places := len(s.Shelf(ShelfWeapons)), len(s.Table())

	if _, _, ok := s.Buy(total - 1); ok {
		t.Error("the merchant sold a basket the purse could not cover")
	}
	if len(s.Table()) != places || len(s.Shelf(ShelfWeapons)) != shelf || s.BuyTotal() != total {
		t.Error("the refused buy changed the table or the shelf")
	}

	bought, spend, ok := s.Buy(total)
	if !ok || spend != total || len(bought) != 2 {
		t.Fatalf("Buy = (%+v, %d, %v), want both weapons for %d", bought, spend, ok, total)
	}
	if len(s.Table()) != 1 || !s.Table()[0].Mine {
		t.Errorf("after the buy the table holds %+v, want the player's place alone", s.Table())
	}
	if _, _, ok := s.Buy(1000); ok {
		t.Error("an empty merchant side still committed")
	}
}

// AC-7 and AC-8: the payout halves the stack once, the screen halves each unit,
// and a sold item is on a shelf.
func TestSellPaysTheStackAndTheScreenShowsThePerUnitFigure(t *testing.T) {
	s := NewShop(1000)
	s.PutOnTable(ShopItem{Code: data.ComposeItemCode(0, 1, 0, 3), Price: 5, Count: 3})

	if got := s.SellTotal(); got != 9 {
		t.Errorf("SellTotal = %d, want ceil(5/2)*3 = 9", got)
	}
	if got := s.SellPayout(); got != 8 {
		t.Errorf("SellPayout = %d, want ceil(3*5/2) = 8", got)
	}

	paid, ok := s.Sell()
	if !ok || paid != 8 {
		t.Fatalf("Sell = (%d, %v), want 8", paid, ok)
	}
	if len(s.Table()) != 0 {
		t.Errorf("the table still holds %+v", s.Table())
	}
	if got := s.Shelf(ShelfWeapons); len(got) != 1 || got[0].Count != 3 {
		t.Errorf("weapon shelf = %+v, want the sold stack of three", got)
	}
	if _, ok := s.Sell(); ok {
		t.Error("an empty player side still committed")
	}
}

// SHOP-SELL-010's return path by class: a weapon goes to the weapon shelf, a
// shield and an armour to the armour shelf, and anything else to Magic Items.
func TestASoldItemLandsOnTheShelfItsClassNames(t *testing.T) {
	for _, tc := range []struct {
		class int
		want  ShopShelf
	}{
		{1, ShelfWeapons},
		{2, ShelfArmour},
		{5, ShelfArmour},
		{12, ShelfArmour},
		{14, ShelfMagic},
		{0, ShelfMagic},
	} {
		got := shopShelfFor(data.ComposeItemCode(0, tc.class, 0, 1))
		if got != tc.want {
			t.Errorf("class %d lands on %v, want %v", tc.class, got, tc.want)
		}
	}
}

func TestNoCoinIsCreatedOrDestroyed(t *testing.T) {
	s := NewShop(1000)
	s.Generate(shopTable(), 6)
	purse := int32(500)

	s.TakeFromShelf(ShelfWeapons, 0, 1)
	s.PutOnTable(ShopItem{Code: data.ComposeItemCode(0, 1, 0, 1), Price: 7, Count: 4})

	spendWant := s.BuyTotal()
	paidWant := s.SellPayout()

	_, spend, ok := s.Buy(purse)
	if !ok {
		t.Fatal("the buy was refused")
	}
	purse -= spend
	paid, ok := s.Sell()
	if !ok {
		t.Fatal("the sell was refused")
	}
	purse += paid

	if spend != spendWant || paid != paidWant {
		t.Errorf("moved (%d, %d), want (%d, %d)", spend, paid, spendWant, paidWant)
	}
	if want := 500 - spendWant + paidWant; purse != want {
		t.Errorf("purse = %d, want %d", purse, want)
	}
}

func TestTheSeedMovesWithEveryCampaignInput(t *testing.T) {
	base := shopSeed(30, 1, 1000)
	for _, tc := range []struct {
		name string
		seed int64
	}{
		{"a later chapter", shopSeed(40, 1, 1000)},
		{"one more mission finished", shopSeed(30, 2, 1000)},
		{"a higher ceiling", shopSeed(30, 1, 3000)},
	} {
		if tc.seed == base {
			t.Errorf("%s gave the same seed %d", tc.name, tc.seed)
		}
		if tc.seed < 0 {
			t.Errorf("%s gave a negative seed %d", tc.name, tc.seed)
		}
	}
	if shopSeed(30, 1, 1000) != base {
		t.Error("the same three inputs gave two seeds")
	}
}
