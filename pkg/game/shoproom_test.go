package game

import (
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// The shop room end to end, driven through ShopClick exactly as pkg/ui
// drives it.

// shopRoom is a front end standing in the stocked shop, with the party carrying
// what carried names.
func shopRoom(t *testing.T, carried []uint16) (*FrontEnd, *townScreen) {
	t.Helper()
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: shopTable()}, CampaignSession: CampaignSession{Town: NewTown(c), Carried: []mapload.PartyMember{{Carry: &mapload.Carry{Items: carried}}}}}
	f.Town.Arrive()
	f.Town.gold = 500
	f.Shop = NewShop(1000)
	f.Shop.Generate(f.Table, 11)
	s := f.TownScreen().(*townScreen)
	s.room = roomShop
	return f, s
}

func TestShopButtonsShowPurseAndProjectedTableBalance(t *testing.T) {
	f, s := shopRoom(t, nil)
	check := func(want [4]int32) {
		t.Helper()
		v := s.ShopScreen()
		if got := [4]int32{v.Purse, v.Buy, v.Sell, v.Total}; got != want {
			t.Fatalf("Undo/Buy/Sell/Exit numbers = %v, want %v", got, want)
		}
	}
	check([4]int32{500, 0, 0, 500})
	f.Shop.shelves[ShelfWeapons] = []ShopItem{{Code: 0x0101, Price: 80, Count: 10}}
	if !f.Shop.TakeFromShelf(ShelfWeapons, 0, 1) {
		t.Fatal("could not stage a purchase")
	}
	check([4]int32{500, 80, 0, 420})
	if !f.Shop.PutOnTable(ShopItem{Code: 0x0101, Price: 40, Count: 1}) {
		t.Fatal("could not stage a sale")
	}
	check([4]int32{500, 80, 20, 440})
	if !f.Shop.TakeFromShelf(ShelfWeapons, 0, 6) {
		t.Fatal("could not stage an unaffordable purchase")
	}
	check([4]int32{500, 560, 20, -40})
	click(s, ui.ShopControlButton, 0)
	check([4]int32{500, 0, 0, 500})
}

// click presses one control, the way pkg/ui does after a hit test.
func click(s *townScreen, kind ui.ShopControlKind, i int) ui.TownAction {
	return s.ShopClick(ui.ShopControl{Kind: kind, Index: i})
}

// clickShift is click with the shift+click convention held (owner, DIV-046,
// DIV-047): a table or pack cell moves its whole stack rather than one unit.
// Every other control ignores it, exactly as ShopClick does.
func clickShift(s *townScreen, kind ui.ShopControlKind, i int) ui.TownAction {
	return s.ShopClick(ui.ShopControl{Kind: kind, Index: i, Shift: true})
}

// The four rectangles in the merchant's room, by what this build stocks behind
// each of them. THE INDEX IS THE HIT RECT'S INDEX (SHOP-SHELF-047), which runs
// lower right, lower left, upper right, upper left.
const (
	roomArmour  = 0
	roomWeapons = 1
	roomMagic   = 2
	roomBooks   = 3
)

// takeFromShelf opens one of the room's shelves and puts its first cell on the
// table, which is the two presses a player makes.
func takeFromShelf(s *townScreen, room int) ui.TownAction {
	click(s, ui.ShopControlShelfPick, room)
	return click(s, ui.ShopControlShelfCell, 0)
}

// fillTable puts five places on the table, one unit from each of five
// DISTINCT shelf cells, walking the stocked rooms in order.
//
// FIVE CLICKS ON ONE CELL MAKE ONE PLACE since `DIV-322`, not five: the room
// joins a second unit of the same code and price to the place it already
// opened (shopFindHisPlace). A setup that wants a full table therefore has to
// spend distinct cells, and one shelf of this fixture does not hold five.
func fillTable(tb *testing.T, f *FrontEnd, s *townScreen) {
	tb.Helper()
	for _, room := range []int{roomArmour, roomWeapons, roomMagic} {
		click(s, ui.ShopControlShelfPick, room)
		shelf := shopRoomShelves[room].shelf
		for i := 0; i < len(f.Shop.Shelf(shelf)); i++ {
			if len(f.Shop.Table()) >= ShopTablePlaces {
				return
			}
			click(s, ui.ShopControlShelfCell, i)
		}
	}
}

// AC-4 and AC-6, driven through the room: take two of his, buy them, and find
// them in the pack with the purse down by exactly the total.
func TestTakingFromTheShelfAndBuyingPutsTheGoodsInThePack(t *testing.T) {
	f, s := shopRoom(t, nil)
	shelf := shelfUnits(f.Shop, ShelfArmour)

	// TWO CLICKS ON ONE CELL MAKE ONE PLACE OF TWO UNITS since `DIV-322`:
	// the room joins a second unit to the merchant's place it came from
	// (shopFindHisPlace), the way it already joined a second unit from the
	// pack. Two units left the shelf either way, which is what the purse and
	// the pack are checked against below.
	takeFromShelf(s, roomArmour)
	click(s, ui.ShopControlShelfCell, 0)
	if len(f.Shop.Table()) != 1 {
		t.Fatalf("the table holds %d places, want one joined place", len(f.Shop.Table()))
	}
	if got := f.Shop.Table()[0].Count; got != 2 {
		t.Fatalf("the joined place holds %d units, want two", got)
	}
	if got := shelfUnits(f.Shop, ShelfArmour); got != shelf-2 {
		t.Fatalf("the shelf holds %d units, want %d", got, shelf-2)
	}

	total := f.Shop.BuyTotal()
	if total <= 0 {
		t.Fatal("two items off the shelf cost nothing")
	}
	act := click(s, ui.ShopControlButton, 1)
	if act.Msg == "" {
		t.Error("the purchase said nothing")
	}
	if got := f.Town.Gold(); got != 500-int(total) {
		t.Errorf("gold = %d, want %d", got, 500-int(total))
	}
	if got := len(f.Carried[0].Carry.Items); got != 2 {
		t.Errorf("the pack holds %d units, want the two bought", got)
	}
	if len(f.Shop.Table()) != 0 {
		t.Errorf("the table still holds %+v after the buy", f.Shop.Table())
	}
}

// Buy remains clickable with an insufficient purse; the transaction refuses it.
func TestAPurseBelowTheTotalKeepsBuyClickableAndRefusesTheTransaction(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Town.gold = 0

	takeFromShelf(s, roomArmour)
	if f.Shop.BuyTotal() <= 0 {
		t.Fatal("the item taken off the shelf costs nothing")
	}
	if !s.ShopScreen().Live[1] {
		t.Fatal("BUY must remain clickable with an empty purse")
	}
	if act := click(s, ui.ShopControlButton, 1); act.Msg == "" {
		t.Error("pressing unaffordable BUY said nothing")
	}
	if _, _, ok := f.Shop.Buy(0); ok {
		t.Error("the model sold on an empty purse")
	}
	if len(f.Shop.Table()) != 1 || f.Town.Gold() != 0 {
		t.Error("the refusal changed the table or the purse")
	}
}

// AC-5: a pack item put on the table leaves the pack, and clearing brings it
// back with no coin moved. Shift+click moves the whole stack in one call
// (owner, DIV-046, DIV-047), which is what this test needs to exercise a
// two-unit round trip.
func TestAPackItemOnTheTableComesBackWhenTheTableIsCleared(t *testing.T) {
	f, s := shopRoom(t, []uint16{0x0155, 0x0155, 0x0122})
	gold := f.Town.Gold()

	clickShift(s, ui.ShopControlPackCell, 1)
	if got := len(f.Carried[0].Carry.Items); got != 1 {
		t.Fatalf("the pack holds %d units after one stack moved, want 1", got)
	}
	if places := f.Shop.Table(); len(places) != 1 || !places[0].Mine || places[0].Count != 2 {
		t.Fatalf("the table holds %+v, want his customer's stack of two", places)
	}

	click(s, ui.ShopControlButton, 0)
	if got := len(f.Carried[0].Carry.Items); got != 3 {
		t.Errorf("the pack holds %d units after the clear, want 3", got)
	}
	if len(f.Shop.Table()) != 0 {
		t.Errorf("the table still holds %+v", f.Shop.Table())
	}
	if f.Town.Gold() != gold {
		t.Errorf("clearing moved a coin: gold %d, want %d", f.Town.Gold(), gold)
	}
}

// AC-8 through the room: selling credits the payout, the goods are on a
// shelf, and the pack no longer holds them. Shift+click moves the whole
// stack in one call (owner, DIV-046, DIV-047), which is what this test needs
// to exercise the stack payout rather than a single unit's.
func TestSellingCreditsThePayoutAndPutsTheGoodsOnAShelf(t *testing.T) {
	// Weapons row 1 of the fixture is priced 40 at every shape and material,
	// so the code below is worth 40 and a stack of two pays 40.
	f, s := shopRoom(t, []uint16{0x0101, 0x0101})
	// Units, not elements: the two sold units land on the weapon shelf and
	// returnToShelf folds them into an element already holding that code
	// (`DIV-322`), so the element count need not move at all.
	shelf := shelfUnits(f.Shop, ShelfWeapons)

	clickShift(s, ui.ShopControlPackCell, 1)
	if got := f.Shop.SellPayout(); got != 40 {
		t.Fatalf("payout = %d, want ceil(2*40/2)", got)
	}
	click(s, ui.ShopControlButton, 2)

	if got := f.Town.Gold(); got != 540 {
		t.Errorf("gold = %d, want 540", got)
	}
	if got := len(f.Carried[0].Carry.Items); got != 0 {
		t.Errorf("the pack still holds %d units", got)
	}
	// TWO units were sold, so the shelf gains two. The old assertion asked
	// for one because it counted ELEMENTS, and the sold place arrived as a
	// single appended element carrying both units.
	if got := shelfUnits(f.Shop, ShelfWeapons); got != shelf+2 {
		t.Errorf("the weapon shelf holds %d units, want %d", got, shelf+2)
	}
}

// shopFindMinePlace's own identity rule, tested directly (seat): Code and
// Price both agreeing with an existing Mine place is a match; either
// disagreeing, or the place not being Mine at all, is not.
func TestShopFindMinePlaceRequiresCodeAndPriceAndMine(t *testing.T) {
	table := []ShopPlace{
		{ShopItem: ShopItem{Code: 0x0101, Price: 40, Count: 2}, Mine: true},
		{ShopItem: ShopItem{Code: 0x0101, Price: 40, Count: 1}, Mine: false, From: ShelfWeapons},
		{ShopItem: ShopItem{Code: 0x0155, Price: 9, Count: 1}, Mine: true},
	}

	if idx, ok := shopFindMinePlace(table, 0x0101, 40); !ok || idx != 0 {
		t.Errorf("code and price agreeing with a Mine place: (%d, %v), want (0, true)", idx, ok)
	}
	if _, ok := shopFindMinePlace(table, 0x0101, 41); ok {
		t.Error("a price one coin off still matched")
	}
	if _, ok := shopFindMinePlace(table, 0x0201, 40); ok {
		t.Error("a different code still matched")
	}
	// Table place 1 carries the SAME code and price as place 0 but is not
	// Mine (a merchant's own shelf-origin place): it must not match, or a
	// pack lot would join the merchant's own goods.
	if idx, ok := shopFindMinePlace(table[1:2], 0x0101, 40); ok {
		t.Errorf("a non-Mine place of the identical code and price matched at %d", idx)
	}
	if _, ok := shopFindMinePlace(table, 0x0155, 8); ok {
		t.Error("the wrong Mine place's price still matched")
	}
	if _, ok := shopFindMinePlace(nil, 0x0101, 40); ok {
		t.Error("an empty table matched something")
	}
}

// DIV-046: a default click moves one unit. Repeated takes join a matching
// player table place, allowing quantities above the five-place table limit.
func TestRepeatedDefaultClicksOnOneStackJoinOneTablePlace(t *testing.T) {
	f, s := shopRoom(t, []uint16{0x0101, 0x0101, 0x0101})

	click(s, ui.ShopControlPackCell, 1)
	if got := len(f.Carried[0].Carry.Items); got != 2 {
		t.Fatalf("the pack holds %d units after one default click, want 2 left behind", got)
	}
	places := f.Shop.Table()
	if len(places) != 1 || !places[0].Mine || places[0].Count != 1 {
		t.Fatalf("the table holds %+v, want one place of one unit", places)
	}

	// A second and third default click reach the same stack (still cell 1,
	// the strip having closed up behind the money element) and join the SAME
	// place rather than opening a second and third one.
	click(s, ui.ShopControlPackCell, 1)
	click(s, ui.ShopControlPackCell, 1)
	if got := len(f.Carried[0].Carry.Items); got != 0 {
		t.Fatalf("the pack holds %d units after three default clicks, want the stack gone", got)
	}
	places = f.Shop.Table()
	if len(places) != 1 || !places[0].Mine || places[0].Count != 3 {
		t.Fatalf("the table holds %+v, want ONE place of three units, not three places of one", places)
	}
}

// The table side of the same identity rule: three default takes off ONE
// place must not fragment the pack into three cells of the same code (seat).
// The pack is a flat code list (townItemStacks, townscreen.go) that re-folds
// by code on every read regardless of insertion order, so this is expected
// to hold already; witnessed rather than assumed.
func TestRepeatedDefaultTakesOffOnePlaceDoNotFragmentThePack(t *testing.T) {
	f, s := shopRoom(t, nil)
	if !f.Shop.PutOnTable(ShopItem{Code: 0x0101, Price: 40, Count: 3}) {
		t.Fatal("setup: putting the fixture stack on the table was refused")
	}

	click(s, ui.ShopControlTableCell, 0)
	click(s, ui.ShopControlTableCell, 0)
	click(s, ui.ShopControlTableCell, 0)

	if len(f.Shop.Table()) != 0 {
		t.Errorf("the table still holds %+v after three takes of a three-unit place", f.Shop.Table())
	}
	got := s.shopPackStacks()
	if len(got) != 1 || got[0].Count != 3 {
		t.Fatalf("the pack folds to %+v, want one stack of three, not three cells", got)
	}
}

// The shift+click convention moves the whole stack in one call, which is the
// behaviour this hotfix keeps as an option rather than removes (owner,
// DIV-046, DIV-047 — the convention itself has no ROM1 claim).
func TestShiftClickMovesTheWholeStackToTheTableInOneCall(t *testing.T) {
	f, s := shopRoom(t, []uint16{0x0101, 0x0101, 0x0101, 0x0122})

	clickShift(s, ui.ShopControlPackCell, 1)
	if got := len(f.Carried[0].Carry.Items); got != 1 {
		t.Fatalf("the pack holds %d units after the shift-click, want the one item left", got)
	}
	places := f.Shop.Table()
	if len(places) != 1 || !places[0].Mine || places[0].Count != 3 {
		t.Fatalf("the table holds %+v, want one place of all three", places)
	}
}

// The table side of the same default: a plain click takes ONE unit off a
// table place and leaves the rest of it on the table (owner, DIV-046).
func TestDefaultClickTakesOneUnitOffTheTableAndLeavesTheRest(t *testing.T) {
	f, s := shopRoom(t, nil)
	if !f.Shop.PutOnTable(ShopItem{Code: 0x0101, Price: 40, Count: 3}) {
		t.Fatal("setup: putting the fixture stack on the table was refused")
	}

	if act := click(s, ui.ShopControlTableCell, 0); act.Msg != "back in your pack" {
		t.Errorf("message %q, want the pack's own sentence", act.Msg)
	}
	if got := len(f.Carried[0].Carry.Items); got != 1 {
		t.Fatalf("the pack holds %d units after one default take, want 1", got)
	}
	places := f.Shop.Table()
	if len(places) != 1 || places[0].Count != 2 {
		t.Fatalf("the table holds %+v, want the same place at count 2", places)
	}
}

// Shift+click on a table place takes the whole place off in one call, which
// is the whole-place behaviour TakeOffTable always had before this hotfix
// (owner, DIV-046, DIV-047).
func TestShiftClickTakesTheWholeTablePlaceOff(t *testing.T) {
	f, s := shopRoom(t, nil)
	if !f.Shop.PutOnTable(ShopItem{Code: 0x0101, Price: 40, Count: 3}) {
		t.Fatal("setup: putting the fixture stack on the table was refused")
	}

	clickShift(s, ui.ShopControlTableCell, 0)
	if got := len(f.Carried[0].Carry.Items); got != 3 {
		t.Fatalf("the pack holds %d units after the shift-take, want all 3", got)
	}
	if len(f.Shop.Table()) != 0 {
		t.Errorf("the table still holds %+v after the shift-take", f.Shop.Table())
	}
}

// AC-11's own edge case: a stack of one moves wholly whichever way it is
// clicked, because TakeOffTable's own "at or above" test takes it all and a
// stack of one IS its own whole (owner, DIV-046).
func TestAStackOfOneMovesWhollyByDefaultOrByShift(t *testing.T) {
	for _, shift := range []bool{false, true} {
		f, s := shopRoom(t, []uint16{0x0101})
		press := click
		if shift {
			press = clickShift
		}

		press(s, ui.ShopControlPackCell, 1)
		if got := len(f.Carried[0].Carry.Items); got != 0 {
			t.Fatalf("shift=%v: the pack holds %d units after moving its one item, want 0", shift, got)
		}
		places := f.Shop.Table()
		if len(places) != 1 || places[0].Count != 1 {
			t.Fatalf("shift=%v: the table holds %+v, want one place of one", shift, places)
		}

		press(s, ui.ShopControlTableCell, 0)
		if got := len(f.Carried[0].Carry.Items); got != 1 {
			t.Fatalf("shift=%v: the pack holds %d units after taking it back, want 1", shift, got)
		}
		if len(f.Shop.Table()) != 0 {
			t.Fatalf("shift=%v: the table still holds %+v", shift, f.Shop.Table())
		}
	}
}

// The money arithmetic follows the quantity actually moved, not the stack
// the click was drawn from: a default click on a 3-unit stack prices the
// table place at one unit, never at three (owner, DIV-046). A partial move
// charging for the whole stack would be worse than the defect this hotfix
// fixes.
func TestMoneyFollowsTheQuantityActuallyMovedNotTheStack(t *testing.T) {
	// Weapons row 1 of the fixture is priced 40 at every shape and material.
	f, s := shopRoom(t, []uint16{0x0101, 0x0101, 0x0101})

	click(s, ui.ShopControlPackCell, 1)
	if got := f.Shop.SellTotal(); got != 20 {
		t.Errorf("SellTotal = %d, want ceil(40/2) for the ONE unit moved, not the stack of 3", got)
	}
	if got := f.Shop.SellPayout(); got != 20 {
		t.Errorf("SellPayout = %d, want ceil(1*40/2) for the ONE unit moved", got)
	}
}

// The counterparty's own capacity limit is the table's five places (the pack
// has none — sim.ItemStack's own slice has no cap). A shift+click that
// would move a whole stack onto a full table refuses exactly as PutOnTable
// always refused a sixth place, and the pack is unchanged: an all-or-nothing
// refusal, never a partial move (owner, DIV-046).
func TestShiftClickOntoAFullTableRefusesAndMovesNothing(t *testing.T) {
	f, s := shopRoom(t, []uint16{0x0101, 0x0101, 0x0101})
	fillTable(t, f, s)
	if len(f.Shop.Table()) != ShopTablePlaces {
		t.Fatalf("setup: the table holds %d places, want five", len(f.Shop.Table()))
	}

	act := clickShift(s, ui.ShopControlPackCell, 1)
	if !contains(act.Msg, "five and no more") {
		t.Errorf("message %q, want the table's own refusal", act.Msg)
	}
	if got := len(f.Carried[0].Carry.Items); got != 3 {
		t.Errorf("the pack holds %d units after the refusal, want all 3 still there", got)
	}
	if len(f.Shop.Table()) != ShopTablePlaces {
		t.Errorf("the table holds %d places after the refusal, want still five", len(f.Shop.Table()))
	}
}

func TestTheScreenStatesBothSellFigures(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Shop.PutOnTable(ShopItem{Code: 0x0101, Price: 5, Count: 3})

	v := s.ShopScreen()
	if v.Sell != 9 {
		t.Errorf("the SELL button prints %d, want ceil(5/2)*3", v.Sell)
	}
	if got := v.Table[0].Price; got != 3 {
		t.Errorf("the table cell prints %d a unit, want ceil(5/2)", got)
	}
	if !v.Table[0].Mine {
		t.Error("the player's own place is not on his side of the deal")
	}
	// THE BUTTON'S FIGURE AND THE PAYOUT DISAGREE, AND THAT IS THE ORIGINAL'S
	// OWN ARITHMETIC, not this build's: the button prints the per-unit halving
	// summed over the stack (SHOP-SCREEN-037's per-cell ceil(price/2)) and the
	// merchant pays the halving of the whole place (SHOP-SELL-010). Both are
	// reproduced, so there is nothing here to disclose.
	if got := f.Shop.SellPayout(); got != 8 {
		t.Errorf("the merchant pays %d for the place, want ceil(5*3/2)", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestTheRoomRefusesASixthPlace(t *testing.T) {
	f, s := shopRoom(t, nil)
	fillTable(t, f, s)
	if len(f.Shop.Table()) != ShopTablePlaces {
		t.Fatalf("the table holds %d, want five", len(f.Shop.Table()))
	}
	// THE SIXTH TAKE COMES OFF A SHELF THE TABLE HOLDS NOTHING FROM. fillTable
	// spends the armour and weapon shelves, and a second take of a code
	// already on the table would JOIN its place rather than open a sixth
	// (shopFindHisPlace) — so taking from either of those would witness the
	// join, not the refusal. shopFindHisPlace compares From as well as code
	// and price, so a magic-shelf take can never join an armour-shelf place.
	click(s, ui.ShopControlShelfPick, roomMagic)
	act := click(s, ui.ShopControlShelfCell, 0)
	if act.Msg == "" {
		t.Error("the sixth take said nothing")
	}
	if len(f.Shop.Table()) != ShopTablePlaces {
		t.Errorf("the table took a sixth: %d places", len(f.Shop.Table()))
	}
}

// The mouse's own door: a click on a place sends it home, exactly as the
// take-back-off row does.
func TestClickingATablePlaceSendsItHome(t *testing.T) {
	f, s := shopRoom(t, nil)
	shelf := len(f.Shop.Shelf(ShelfArmour))
	takeFromShelf(s, roomArmour)

	if act := click(s, ui.ShopControlTableCell, 0); act.Msg == "" {
		t.Error("the click said nothing")
	}
	if len(f.Shop.Table()) != 0 {
		t.Errorf("the table still holds %+v", f.Shop.Table())
	}
	if got := len(f.Shop.Shelf(ShelfArmour)); got != shelf {
		t.Errorf("the shelf holds %d, want the item back at %d", got, shelf)
	}

	// A click from another room reaches nothing.
	s.room = roomSquare
	if act := click(s, ui.ShopControlTableCell, 0); act.Msg != "" {
		t.Errorf("a click outside the shop acted: %+v", act)
	}
}

func TestTheRoomRectanglesChooseTheShownShelf(t *testing.T) {
	f, s := shopRoom(t, nil)
	for _, cell := range s.ShopScreen().Shelf {
		if cell.Occupied() {
			t.Fatal("the shelf grid holds an item before any shelf was clicked")
		}
	}

	click(s, ui.ShopControlShelfPick, roomWeapons)
	if s.shopShelf != ShelfWeapons {
		t.Fatalf("the lower-left rectangle opened %v, want the weapon shelf", s.shopShelf)
	}
	if !s.ShopScreen().Shelf[0].Occupied() {
		t.Fatal("the chosen shelf shows no first cell")
	}

	click(s, ui.ShopControlShelfPick, roomArmour)
	if s.shopShelf != ShelfArmour {
		t.Fatalf("the lower-right rectangle opened %v, want the armour shelf", s.shopShelf)
	}

	// A cell addresses the shown shelf: taking the first one off removes it
	// from THAT shelf and no other.
	// Units, not elements: a cell holding more than one unit stays on the
	// shelf when a single unit is taken off it (`DIV-322`), so an assertion
	// on len() would be answered by the element the take only shortened.
	before := shelfUnits(f.Shop, ShelfArmour)
	weapons := shelfUnits(f.Shop, ShelfWeapons)
	click(s, ui.ShopControlShelfCell, 0)
	if got := shelfUnits(f.Shop, ShelfArmour); got != before-1 {
		t.Errorf("the shown shelf holds %d units, want %d", got, before-1)
	}
	if got := shelfUnits(f.Shop, ShelfWeapons); got != weapons {
		t.Errorf("a take off one shelf changed another: %d units, want %d", got, weapons)
	}
}

func TestTheShelfArrowsScrollByWholeRows(t *testing.T) {
	f, s := shopRoom(t, nil)

	// THE SHELF IS STOCKED BY HAND HERE, with eight DISTINCT codes. The
	// generator's own armour pool is narrow enough that folding its repeats
	// (`DIV-322`) leaves fewer cells than a scroll needs, and this test is
	// about the grid's paging rather than about what the generator drew.
	stock := make([]ShopItem, 0, 8)
	for i := 0; i < 8; i++ {
		stock = append(stock, ShopItem{Code: data.ItemCode(0x0201 + i), Price: int32(90 - 10*i), Count: 1})
	}
	f.Shop.shelves[ShelfArmour] = stock

	click(s, ui.ShopControlShelfPick, roomArmour)
	items := f.Shop.Shelf(ShelfArmour)
	if len(items) < 8 {
		t.Fatalf("the fixture stocked %d armour items, too few to scroll", len(items))
	}

	if got := s.ShopScreen().Shelf[0].Price; got != items[0].Price {
		t.Fatalf("the first cell shows price %d, want the first item's %d", got, items[0].Price)
	}
	click(s, ui.ShopControlArrowUp, 0)
	if s.shelfBase != 0 {
		t.Errorf("the up arrow at the first row scrolled to %d", s.shelfBase)
	}
	click(s, ui.ShopControlArrowDown, 0)
	if s.shelfBase != 2 {
		t.Fatalf("one press of the down arrow moved to %d, want one row of two", s.shelfBase)
	}
	if got := s.ShopScreen().Shelf[0].Price; got != items[2].Price {
		t.Errorf("after one row the first cell shows %d, want item 2's %d", got, items[2].Price)
	}

	// The base never leaves the shelf, however many presses it takes.
	for i := 0; i < 200; i++ {
		click(s, ui.ShopControlArrowDown, 0)
	}
	if s.shelfBase >= len(items) {
		t.Errorf("the down arrow scrolled past the shelf: base %d of %d", s.shelfBase, len(items))
	}
	if !s.ShopScreen().Shelf[0].Occupied() {
		t.Error("scrolling to the end left the grid empty")
	}
}

// AC-15 and SHOP-SCREEN-036: the cell background states whether the purse
// covers the item in it.
func TestTheCellBackgroundStatesAffordability(t *testing.T) {
	f, s := shopRoom(t, nil)
	click(s, ui.ShopControlShelfPick, roomArmour)
	price := f.Shop.Shelf(ShelfArmour)[0].Price
	if price <= 1 {
		t.Fatalf("the fixture's first armour item costs %d, too little to test", price)
	}

	f.Town.gold = int(price)
	if got := s.ShopScreen().Shelf[0].Back; got != ui.ShopBackAffordable {
		t.Errorf("background = %v at exactly the price, want the affordable one", got)
	}
	f.Town.gold = int(price) - 1
	if got := s.ShopScreen().Shelf[0].Back; got != ui.ShopBackItem {
		t.Errorf("background = %v one coin short, want the plain one", got)
	}
	if got := s.ShopScreen().Table[0].Back; got != ui.ShopBackEmpty {
		t.Errorf("an empty table place has background %v, want the empty one", got)
	}
}

func TestEveryOccupiedCellCarriesItsCharacteristics(t *testing.T) {
	_, s := shopRoom(t, []uint16{0x0101})
	click(s, ui.ShopControlShelfPick, roomWeapons)
	v := s.ShopScreen()

	if len(v.Shelf[0].Info) == 0 {
		t.Error("a shelf cell states nothing on hover")
	}
	// The lines are the inventory popup's own, so an item states the same
	// characteristics wherever it is seen. Pack cell 0 is the money element;
	// the first stack stands in cell 1.
	if !v.Pack[0].Money {
		t.Error("the strip's first place is not the money element")
	}
	if want := itemInfoLines(0x0101, s.in.Table); !reflect.DeepEqual(v.Pack[1].Info, want) {
		t.Errorf("the pack cell states %v, want the inventory's own %v", v.Pack[1].Info, want)
	}
	if len(v.Table[0].Info) != 0 {
		t.Error("an empty table place states something on hover")
	}
}

func TestTheShopScreenWithNoTablesOffersNothingAndCanBeLeft(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Shop = NewShop(0)
	f.Shop.Generate(nil, 1)
	s := f.TownScreen().(*townScreen)
	s.room = roomShop

	v := s.ShopScreen()
	if v.Live != [4]bool{false, true, true, true} {
		t.Errorf("empty shop button state = %v", v.Live)
	}
	if !v.Live[3] {
		t.Error("the shop cannot be left")
	}
	for _, cell := range v.Shelf {
		if cell.Occupied() {
			t.Error("an empty shop shows an item")
		}
	}

	// Every control, pressed, changes nothing and panics on none.
	for _, kind := range []ui.ShopControlKind{
		ui.ShopControlArrowUp, ui.ShopControlArrowDown, ui.ShopControlShelfCell,
		ui.ShopControlTableCell, ui.ShopControlPackCell, ui.ShopControlShelfPick,
		ui.ShopControlMerchant, ui.ShopControlPickerPrev, ui.ShopControlPickerNext,
	} {
		for i := -1; i < 7; i++ {
			click(s, kind, i)
		}
	}
	for i := 0; i < 3; i++ {
		click(s, ui.ShopControlButton, i)
	}
	if len(f.Shop.Table()) != 0 || f.Town.Gold() != initialPlayerPurse {
		t.Error("pressing the empty screen's controls moved something")
	}
	if !s.Back() {
		t.Error("the empty shop room could not be left")
	}
}

func TestTheFourthButtonClearsTheTableAndLeaves(t *testing.T) {
	f, s := shopRoom(t, []uint16{0x0101})
	takeFromShelf(s, roomArmour)
	click(s, ui.ShopControlPackCell, 1)
	if len(f.Shop.Table()) != 2 {
		t.Fatalf("the table holds %d, want one of each side", len(f.Shop.Table()))
	}

	if act := click(s, ui.ShopControlButton, 3); act.Msg != "" {
		t.Errorf("leaving posted %q; the original's leave posts no line", act.Msg)
	}
	if s.room == roomShop {
		t.Error("the fourth button did not leave the room")
	}
	if len(f.Shop.Table()) != 0 {
		t.Errorf("leaving left %+v on the table", f.Shop.Table())
	}
	if got := len(f.Carried[0].Carry.Items); got != 1 {
		t.Errorf("the pack holds %d units after leaving, want the one he walked in with", got)
	}
}

func TestAnUnpriceableItemIsStatedAsHavingNoPrice(t *testing.T) {
	// Class 14 is the carried class — quest documents and what a mission
	// script hands out. Field D 3, field C 0, field A 0.
	const questItem uint16 = 14<<8 | 3
	f, s := shopRoom(t, []uint16{questItem})

	act := click(s, ui.ShopControlPackCell, 1)
	if act.Msg == "" {
		t.Error("putting an unpriced item on the table said nothing")
	}
	if got := f.Shop.SellTotal(); got != 0 {
		t.Errorf("SellTotal = %d, want 0 for an item with no price", got)
	}
	if !s.ShopScreen().Live[2] {
		t.Error("SELL is not clickable for an item with no price")
	}
	f.originalCity = &originalCitySaveState{trade: &originalCityTrade{}}
	binding := f.originalCity.trade
	before, gold := f.Shop.Table(), f.Town.Gold()
	click(s, ui.ShopControlButton, 2)
	if !reflect.DeepEqual(before, f.Shop.Table()) || f.Town.Gold() != gold || f.originalCity.trade != binding {
		t.Fatal("refused sale changed holdings or source transaction")
	}

	// It is still the player's and clearing the table gives it back.
	click(s, ui.ShopControlButton, 0)
	if got := f.Carried[0].Carry.Items; !reflect.DeepEqual(got, []uint16{questItem}) {
		t.Errorf("the pack holds %v, want the quest item back", got)
	}
}

func TestArrivingInTheTownStocksTheMerchant(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: shopTable()}, CampaignSession: CampaignSession{Town: NewTown(c)}}

	f.arriveInTown()
	if !f.Town.Open() {
		t.Fatal("arriving did not open the town")
	}
	if got := f.Shop.Ceiling(); got != 1000 {
		t.Errorf("ceiling = %d, want chapter 30's own ShopMaxPrice", got)
	}
	// A HUNDRED UNITS, IN AS MANY ELEMENTS AS THE POOL YIELDED. The generator
	// draws shopShelfDraws[shelf] times and the shelf folds its repeats
	// (`DIV-322`), so the promise is the unit total.
	first := f.Shop.Shelf(ShelfWeapons)
	if got := shelfUnits(f.Shop, ShelfWeapons); got != 100 {
		t.Fatalf("the weapon shelf holds %d units, want 100", got)
	}

	// The same state stocks the same shop, which is provenance A-1's
	// disclosed divergence: a reload shows the same goods.
	f.arriveInTown()
	if !reflect.DeepEqual(first, f.Shop.Shelf(ShelfWeapons)) {
		t.Error("one campaign state stocked two different shops")
	}

	// Winning the chapter moves both the ceiling and the assortment.
	f.Town.Won(30)
	f.Town.Won(31)
	f.arriveInTown()
	if got := f.Shop.Ceiling(); got != 5000 {
		t.Errorf("ceiling = %d, want chapter 40's own", got)
	}
	if reflect.DeepEqual(first, f.Shop.Shelf(ShelfWeapons)) {
		t.Error("a new chapter stocked the same shop")
	}
}

// A fourth selector is a stocked model shelf even when an incomplete fixture
// supplies no eligible Spells rows. It names the existing item-name fallback
// and opens an empty list rather than reporting an unsupported consumable arm.
func TestPressingTheBookShelfOpensTheFourthModelShelf(t *testing.T) {
	_, s := shopRoom(t, nil)
	act := click(s, ui.ShopControlShelfPick, roomBooks)
	if s.shopShelf != ShelfBooks || s.shopChosen != roomBooks || act.Msg != "" {
		t.Errorf("clicking the book shelf left shelf=%v chosen=%d message=%q", s.shopShelf, s.shopChosen, act.Msg)
	}
}

func TestLeavingTheShopClearsTheTableSoARestockDestroysNothing(t *testing.T) {
	f, s := shopRoom(t, []uint16{0x0101, 0x0101})
	shelf := len(f.Shop.Shelf(ShelfArmour))

	click(s, ui.ShopControlPackCell, 1)
	takeFromShelf(s, roomArmour)
	if len(f.Shop.Table()) != 2 {
		t.Fatalf("the table holds %d places, want one of each side", len(f.Shop.Table()))
	}

	if !s.Back() {
		t.Fatal("Back in the shop reported it left no room")
	}
	if len(f.Shop.Table()) != 0 {
		t.Fatalf("leaving left %+v on the table", f.Shop.Table())
	}
	if got := len(f.Carried[0].Carry.Items); got != 2 {
		t.Errorf("the pack holds %d units after leaving, want the two he walked in with", got)
	}
	if got := len(f.Shop.Shelf(ShelfArmour)); got != shelf {
		t.Errorf("the armour shelf holds %d, want %d", got, shelf)
	}

	// And the restock that follows a homecoming cannot reach anything of his.
	f.arriveInTown()
	if got := len(f.Carried[0].Carry.Items); got != 2 {
		t.Errorf("the restock left the pack at %d units, want 2", got)
	}
}

// The party picker moves the character panel and the bottom strip TOGETHER
// (SHOP-PICKER-043: one routine binds the backpack grid to the new member's own
// container and marks the figure for recomposition).
func TestThePickerMovesTheFigureAndTheStripTogether(t *testing.T) {
	f, s := shopRoom(t, []uint16{0x0101})
	f.Carried = append(f.Carried, mapload.PartyMember{
		Name: "second", Carry: &mapload.Carry{Items: []uint16{0x0155, 0x0155}}})

	v := s.ShopScreen()
	if v.Member != 0 || v.MemberCount != 2 {
		t.Fatalf("the panel opens on member %d of %d, want the first of two", v.Member, v.MemberCount)
	}
	if !v.Pack[0].Money || v.Pack[1].Count != 1 {
		t.Fatalf("the first member's strip is %+v", v.Pack[:2])
	}

	if act := click(s, ui.ShopControlPickerNext, 0); act.Msg == "" {
		t.Error("stepping the picker said nothing")
	}
	v = s.ShopScreen()
	if v.Member != 1 || v.Character.Subject.Name != "second" {
		t.Fatalf("the panel shows member %d (%q), want the second", v.Member, v.Character.Subject.Name)
	}
	if v.Pack[1].Count != 2 {
		t.Errorf("the strip holds %d units, want the second member's stack of two", v.Pack[1].Count)
	}

	// Both ends wrap, which is the original's own reset at the roster bound.
	click(s, ui.ShopControlPickerNext, 0)
	if got := s.ShopScreen().Member; got != 0 {
		t.Errorf("stepping past the last member showed %d, want the first", got)
	}
	click(s, ui.ShopControlPickerPrev, 0)
	if got := s.ShopScreen().Member; got != 1 {
		t.Errorf("stepping back from the first showed %d, want the last", got)
	}
}

func TestBookIsIndependentAndKeepsTheTradeTableAcrossMembers(t *testing.T) {
	f, s := shopRoom(t, nil)
	spellParams := make([]int32, 19)
	spellParams[1], spellParams[2], spellParams[4], spellParams[6] = 3, 1, 1, 7
	f.Table.Spells = dbCollection{{}, {name: "First Spell", params: spellParams}, {name: "Second Spell", params: spellParams}}
	f.Carried[0].KnownSpells = 1 << 1
	f.Carried = append(f.Carried, mapload.PartyMember{Name: "second", KnownSpells: 1 << 2,
		Carry: &mapload.Carry{}})
	takeFromShelf(s, roomArmour)
	before := append([]ShopPlace(nil), f.Shop.Table()...)

	click(s, ui.ShopControlBook, 0)
	v := s.ShopScreen()
	if !v.Book || len(v.Spells) != 1 || v.Spells[0].Name != "First Spell" {
		t.Fatalf("first book = on %v spells %+v", v.Book, v.Spells)
	}
	click(s, ui.ShopControlCharacterMode, 0)
	if !s.ShopScreen().Character.Statistics || !s.ShopScreen().Book {
		t.Fatal("statistics mode changed the independent Book toggle")
	}
	if act := click(s, ui.ShopControlSpell, 0); act.Msg != "" || !reflect.DeepEqual(f.Shop.Table(), before) {
		t.Fatalf("inspection-only spell click returned %+v or changed the table", act)
	}
	click(s, ui.ShopControlPickerNext, 0)
	v = s.ShopScreen()
	if len(v.Spells) != 1 || v.Spells[0].Name != "Second Spell" {
		t.Fatalf("second book = %+v", v.Spells)
	}
	if !reflect.DeepEqual(f.Shop.Table(), before) {
		t.Fatal("member change altered the trade table")
	}
	click(s, ui.ShopControlBook, 0)
	if s.ShopScreen().Book || !reflect.DeepEqual(f.Shop.Table(), before) {
		t.Fatal("closing Book did not restore the unchanged table")
	}
}

// DIV-326
func TestBookOpensEmptyForAMemberWithNoKnownSpells(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Carried[0].KnownSpells = 0

	click(s, ui.ShopControlBook, 0)
	v := s.ShopScreen()
	if !v.Book {
		t.Fatal("Book toggled on for a member with no known spells produced v.Book = false, want true (the same empty panel a caster's book opens)")
	}
	if len(v.Spells) != 0 {
		t.Fatalf("Spells = %+v for a member with no known spells, want none", v.Spells)
	}
}

// TestEveryTradeTableProducerRevealsTheTableBeforeItStagesAnything closes the
// full-table regression the owner observed. Book owns the table's pixels, so
// pack, shelf and doll producers must close it before they mutate the hidden
// model. A real full table is also revealed before its refusal is reported.
func TestEveryTradeTableProducerRevealsTheTableBeforeItStagesAnything(t *testing.T) {
	t.Run("new visit starts closed", func(t *testing.T) {
		_, s := shopRoom(t, nil)
		s.shopBook = true
		s.room = roomSquare
		shopDoor := -1
		for i, door := range townDoors {
			if door.room == roomShop {
				shopDoor = i
				break
			}
		}
		if shopDoor < 0 {
			t.Fatal("fixture has no shop door")
		}
		s.Choose(shopDoor)
		if s.shopBook {
			t.Fatal("a new shop visit retained the previous visit's open book")
		}
	})

	t.Run("pack", func(t *testing.T) {
		f, s := shopRoom(t, []uint16{0x0101})
		s.shopBook = true
		act := click(s, ui.ShopControlPackCell, 1)
		if act.Msg == "the table holds five and no more" || s.shopBook || len(f.Shop.Table()) != 1 ||
			!s.ShopScreen().Table[0].Occupied() || len(f.Carried[0].Carry.Items) != 0 {
			t.Fatalf("pack stage = msg %q book %v table %+v pack %v", act.Msg, s.shopBook,
				f.Shop.Table(), f.Carried[0].Carry.Items)
		}
	})

	t.Run("shelf", func(t *testing.T) {
		f, s := shopRoom(t, nil)
		click(s, ui.ShopControlShelfPick, roomArmour)
		s.shopBook = true
		act := click(s, ui.ShopControlShelfCell, 0)
		if act.Msg == "the table holds five and no more" || s.shopBook || len(f.Shop.Table()) != 1 ||
			!s.ShopScreen().Table[0].Occupied() {
			t.Fatalf("shelf stage = msg %q book %v table %+v", act.Msg, s.shopBook, f.Shop.Table())
		}
	})

	t.Run("genuinely full", func(t *testing.T) {
		f, s := shopRoom(t, []uint16{0x0101})
		fillTable(t, f, s)
		beforeTable := f.Shop.Table()
		beforePack := append([]uint16(nil), f.Carried[0].Carry.Items...)
		s.shopBook = true
		act := click(s, ui.ShopControlPackCell, 1)
		if act.Msg != "the table holds five and no more" || s.shopBook {
			t.Fatalf("full stage = msg %q book %v, want a visible full-table refusal", act.Msg, s.shopBook)
		}
		if !reflect.DeepEqual(f.Shop.Table(), beforeTable) || !reflect.DeepEqual(f.Carried[0].Carry.Items, beforePack) {
			t.Fatal("full-table refusal changed the table or pack")
		}
		v := s.ShopScreen()
		for i := range v.Table {
			if !v.Table[i].Occupied() {
				t.Fatalf("revealed full table cell %d is visually empty", i)
			}
		}
	})
}

// Stepping the picker returns the strip's base to the first place: the new
// member's list is a different list, so a base carried over from the old one
// would show his container from an arbitrary offset. The money element back
// in the first place is the observable half of the same reset.
func TestSteppingThePickerReturnsTheStripToItsFirstPlace(t *testing.T) {
	f, s := shopRoom(t, []uint16{0x0101, 0x0122, 0x0155, 0x0102, 0x0103, 0x0104})
	f.Carried = append(f.Carried, mapload.PartyMember{
		Name: "second", Carry: &mapload.Carry{Items: []uint16{0x0101}}})

	s.ShopScroll(ui.ShopWheelPack, 2)
	if s.packBase != 2 {
		t.Fatalf("the strip turned to %d, want two places", s.packBase)
	}
	if s.ShopScreen().Pack[0].Money {
		t.Fatal("the money element is still in the first place after the strip turned")
	}

	click(s, ui.ShopControlPickerNext, 0)

	if s.packBase != 0 {
		t.Errorf("after the picker the strip's base is %d, want the first place", s.packBase)
	}
	if v := s.ShopScreen(); !v.Pack[0].Money {
		t.Errorf("the first place holds %+v, want the money element", v.Pack[0])
	}
}

// A purchase lands in the SHOWN member's container, not always the first one's.
func TestBuyingLandsInTheShownMembersOwnContainer(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Carried = append(f.Carried, mapload.PartyMember{Name: "second", Carry: &mapload.Carry{}})
	click(s, ui.ShopControlPickerNext, 0)

	takeFromShelf(s, roomArmour)
	click(s, ui.ShopControlButton, 1)

	if got := len(f.Carried[0].Carry.Items); got != 0 {
		t.Errorf("the first member's pack holds %d units, want none", got)
	}
	if got := len(f.Carried[1].Carry.Items); got != 1 {
		t.Errorf("the shown member's pack holds %d units, want the one bought", got)
	}
}

// The strip's first element is the money element and it carries the purse
// (SHOP-SCREEN-036 arm a: the quantity is re-read from the campaign's gold).
func TestTheStripsFirstPlaceIsTheMoneyElement(t *testing.T) {
	f, s := shopRoom(t, []uint16{0x0101, 0x0122, 0x0155, 0x0102, 0x0103, 0x0104})
	v := s.ShopScreen()
	if !v.Pack[0].Money || v.Pack[0].Count != uint32(f.Town.Gold()) {
		t.Errorf("the money element is %+v, want the purse %d", v.Pack[0], f.Town.Gold())
	}
	if v.Pack[0].Occupied() {
		t.Error("the money element reads as an item")
	}
	// Turning the strip past it leaves it behind, because it is an element of
	// the list rather than a fixture of the grid.
	s.ShopScroll(ui.ShopWheelPack, 1)
	if s.ShopScreen().Pack[0].Money {
		t.Error("the money element is still in the first place after the strip turned")
	}
}

// The wheel turns the region it is given and nothing else (DIV-005).
func TestTheWheelTurnsOnlyTheRegionItIsGiven(t *testing.T) {
	f, s := shopRoom(t, []uint16{0x0101, 0x0122})
	click(s, ui.ShopControlShelfPick, roomArmour)
	if len(f.Shop.Shelf(ShelfArmour)) <= 6 {
		t.Skip("the fixture's armour shelf is too short to scroll")
	}
	s.ShopScroll(ui.ShopWheelShelf, 1)
	if s.shelfBase != shopShelfCols {
		t.Errorf("the rack paged to %d, want one row of %d", s.shelfBase, shopShelfCols)
	}
	if s.packBase != 0 {
		t.Errorf("turning the rack moved the strip to %d", s.packBase)
	}
	s.ShopScroll(ui.ShopWheelPack, 1)
	if s.packBase != 1 {
		t.Errorf("the strip moved to %d, want one place", s.packBase)
	}
	if s.shelfBase != shopShelfCols {
		t.Errorf("turning the strip moved the rack to %d", s.shelfBase)
	}
	s.ShopScroll(ui.ShopWheelNone, 1)
	if s.packBase != 1 || s.shelfBase != shopShelfCols {
		t.Error("a wheel over nothing moved a grid")
	}
}

// Pure black is the transparent colour of the shop's 24-bit bitmaps, and
// keyBlack is where that is applied.
func TestKeyBlackClearsOnlyPureBlack(t *testing.T) {
	pic := image.NewRGBA(image.Rect(0, 0, 3, 1))
	pic.SetRGBA(0, 0, color.RGBA{A: 0xff})
	pic.SetRGBA(1, 0, color.RGBA{R: 1, A: 0xff})
	pic.SetRGBA(2, 0, color.RGBA{R: 0x80, G: 0x40, B: 0x20, A: 0xff})
	keyBlack(pic)
	if got := pic.RGBAAt(0, 0).A; got != 0 {
		t.Errorf("a pure-black pixel kept alpha %d", got)
	}
	if got := pic.RGBAAt(1, 0).A; got != 0xff {
		t.Errorf("a near-black pixel lost alpha: %d", got)
	}
	if got := pic.RGBAAt(2, 0).A; got != 0xff {
		t.Errorf("a coloured pixel lost alpha: %d", got)
	}
	if keyBlack(nil) != nil {
		t.Error("keyBlack invented a picture")
	}
}
