package game

import (
	"cmp"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// The town after save 666's mission 20, through App: every shelf opened by its
// own rectangle and paged with the down arrow paints its stock in groups, class
// then kind, and cheapest first inside one class and kind (owner, `DIV-319`). A
// unit bought from the middle of a shelf leaves the rest in order and, sold
// back or returned from the table, stands in its old cell again; a unit of a
// code the shop lacks, sold from the pack, lands in its own group by its price,
// not at the end. An F2 SAVE writes SAV, and its cold LOAD in a fresh front end
// opens every shelf in order again. Every chapter's generated shop, magic shelf
// included, is in the same order.
func TestReleaseShopShelvesRunCheapestFirstInsideEachGroup(t *testing.T) {
	source := os.Getenv("AGAINROM_SAVE_666")
	if source == "" {
		t.Skip("AGAINROM_SAVE_666 is not set")
	}
	f := shopOrderFront(t)
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("shop order")
	app.Layout(640, 480)
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(source)}, nil))
	homecoming := HeadlessScenario{Version: 7, Steps: []HeadlessStep{
		{Command: "load", Target: "666 - mission 20"},
		{Command: "activate", Target: "notice"},
		{Command: "wait_until", Until: &HeadlessUntil{Control: "SHOP"}, Ticks: 4000},
	}}
	if err := RunHeadlessScenario(f, app, homecoming, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	s := shopOrderEnter(t, f, app)
	first := shopOrderShelves(t, app, s, "homecoming")

	bought, room, at := shopOrderBuy(t, app, s, first)
	from := shopRoomShelves[room].shelf
	rest := shopOrderShelf(t, app, s, room, "after the purchase")
	if !shopOrderSame(rest, slices.Delete(slices.Clone(first[from]), at, at+1)) {
		t.Fatalf("buying cell %d of shelf %d moved the elements left on it", at, from)
	}
	shopOrderSell(t, app, s, shopOrderPackStack(t, s, bought))
	if back := shopOrderShelf(t, app, s, room, "after selling it back"); !shopOrderSame(back, first[from]) {
		t.Fatalf("%s sold back left shelf %d unlike the homecoming's", shopOrderLabel(f, bought), from)
	}
	t.Logf("bought %s from cell %d of %d on shelf %d and sold it back into the same cell",
		shopOrderLabel(f, bought), at, len(first[from]), from)

	// A return: the same unit staged from its cell and sent back off the table
	// by a click on its place and by the Clear button.
	for _, way := range []struct {
		name  string
		clear bool
	}{{"its table place", false}, {"Clear", true}} {
		shopOrderReturn(t, app, s, room, at, way.clear)
		if back := shopOrderShelf(t, app, s, room, "after returning it by "+way.name); !shopOrderSame(back, first[from]) {
			t.Fatalf("%s returned by %s left shelf %d unlike the homecoming's", shopOrderLabel(f, bought), way.name, from)
		}
	}

	sold := shopOrderSell(t, app, s, shopOrderNewStock(t, s))
	into := shopShelfForItem(sold, f.Shop.tbl)
	// The unit stands after every element it ties with and before every one that
	// stands after it: at the count of those that do not stand after it.
	want := 0
	for _, e := range first[into] {
		if shopOrderCompare(f.Table, e, sold) <= 0 {
			want++
		}
	}
	shelf := shopOrderShelf(t, app, s, shopOrderRoom(t, into), "after the sale")
	at = slices.IndexFunc(shelf, func(e ShopItem) bool { return e.Code == sold.Code && e.Price == sold.Price })
	if at != want || at == 0 || at == len(shelf)-1 || len(shelf) != len(first[into])+1 {
		t.Fatalf("the sold unit stands at %d of %d elements, want a new element at %d inside the shelf", at, len(shelf), want)
	}
	t.Logf("sold %s: cell %d of %d, between %s and %s", shopOrderLabel(f, sold), at, len(shelf),
		shopOrderLabel(f, shelf[at-1]), shopOrderLabel(f, shelf[at+1]))

	name := shopOrderF2Save(t, app, store)
	g := shopOrderFront(t)
	cold := g.App("shop order cold")
	cold.Layout(640, 480)
	save, list, load := g.SaveSeams(SaveStore{Dir: store.Dir}, OriginalStore{}, nil)
	cold.SetSaveSeams(save, list, load)
	shopOrderLoad(t, cold, list, name)
	reloaded := shopOrderShelves(t, cold, shopOrderEnter(t, g, cold), "cold LOAD")
	for shelf := range reloaded {
		t.Logf("cold LOAD shelf %d: %d elements; the homecoming's %d, same sequence %v", shelf,
			len(reloaded[shelf]), len(first[shelf]), shopOrderSame(reloaded[shelf], first[shelf]))
	}

	// Check every main chapter's shelf at its own installed ceiling.
	camp := f.Campaign.Value()
	magic, held := 0, []string{}
	for _, n := range camp.Main {
		ceiling := int32(camp.Chapters[n].ShopMax)
		shop := NewShop(ceiling)
		shop.Generate(f.Table, shopSeed(n, 0, ceiling))
		var counts [numShopShelves]int
		for shelf := range counts {
			items := shop.Shelf(ShopShelf(shelf))
			counts[shelf] = len(items)
			for k := 1; k < len(items); k++ {
				if !shopOrderInOrder(f.Table, items[k-1], items[k]) {
					t.Errorf("chapter %d shelf %d cell %d %s stands after %s", n, shelf, k,
						shopOrderLabel(f, items[k]), shopOrderLabel(f, items[k-1]))
				}
			}
		}
		magic += counts[ShelfMagic]
		held = append(held, fmt.Sprintf("%d at %d: %v", n, ceiling, counts))
	}
	if magic == 0 {
		t.Fatal("no chapter's shop holds magic stock")
	}
	t.Logf("generated shelves by chapter: %s", strings.Join(held, "; "))
}

// shopOrderFront is a release front end with deterministic frames and with
// every audio device removed before its App is built, so nothing plays.
func shopOrderFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SoundPlayer, f.SpeechPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer = nil, nil, nil, nil, nil
	f.SetDeterministicFrames(true)
	return f
}

// shopOrderEnter walks from the town into the merchant's room through App.
func shopOrderEnter(t *testing.T, f *FrontEnd, app *ui.App) *townScreen {
	t.Helper()
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	s := f.TownScreen().(*townScreen)
	for n := 0; s.room == roomTalk && n < 32; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if s.room != roomShop {
		t.Fatalf("App entered room %d, want the shop", s.room)
	}
	closeShopTip(t, app, s)
	return s
}

// shopOrderShelves reads all four of the room's shelves with shopOrderShelf.
func shopOrderShelves(t *testing.T, app *ui.App, s *townScreen, stage string) [numShopShelves][]ShopItem {
	t.Helper()
	var out [numShopShelves][]ShopItem
	for room, r := range shopRoomShelves {
		out[r.shelf] = shopOrderShelf(t, app, s, room, stage)
	}
	return out
}

// shopOrderShelf opens the room's shelf rectangle room and pages through it
// with the down arrow. Every element must be painted once, at its model index,
// and the shelf must stand in its order: class, then kind, then price cheapest
// first (shopOrderInOrder). It returns the shelf's elements.
func shopOrderShelf(t *testing.T, app *ui.App, s *townScreen, room int, stage string) []ShopItem {
	t.Helper()
	shopPointer(t, app)("shelf_pick", room, "press", "release")
	down := shopOrderControl(t, ui.ShopControlArrowDown)
	items := s.sess.Shop.Shelf(shopRoomShelves[room].shelf)
	painted := make([]bool, len(items))
	for v := s.ShopScreen(); ; v = s.ShopScreen() {
		if v.Chosen != room {
			t.Fatalf("%s: rectangle %d shows shelf %d", stage, room, v.Chosen)
		}
		for i, cell := range v.Shelf {
			k := v.ShelfOffset + i
			if k >= len(items) {
				break
			}
			if cell.Price != items[k].Price || cell.Count != uint32(items[k].Count) {
				t.Fatalf("%s: shelf %d cell %d paints %d x%d, model %d x%d", stage, room, k,
					cell.Price, cell.Count, items[k].Price, items[k].Count)
			}
			painted[k] = true
		}
		if v.ShelfOffset+len(v.Shelf) >= len(items) {
			break
		}
		shopTap(t, app, down, "press", "release")
		if s.ShopScreen().ShelfOffset <= v.ShelfOffset {
			t.Fatalf("%s: the down arrow left shelf %d at offset %d", stage, room, v.ShelfOffset)
		}
	}
	for k := range items {
		if !painted[k] {
			t.Fatalf("%s: shelf %d element %d was never painted", stage, room, k)
		}
		if k > 0 && !shopOrderInOrder(s.in.Table, items[k-1], items[k]) {
			t.Errorf("%s: shelf %d cell %d %s stands after %s", stage, room, k,
				shopOrderLabelIn(s.in, items[k]), shopOrderLabelIn(s.in, items[k-1]))
		}
	}
	t.Logf("%s shelf %d (%s): %d elements%s", stage, room, shopOrderShelfNames[room], len(items), shopOrderEndsIn(s.in, items))
	return items
}

// shopOrderRoom is the room's shelf rectangle that opens shelf.
func shopOrderRoom(t *testing.T, shelf ShopShelf) int {
	t.Helper()
	for room, r := range shopRoomShelves {
		if r.shelf == shelf {
			return room
		}
	}
	t.Fatalf("no shelf rectangle opens shelf %d", shelf)
	return -1
}

// shopOrderShelfNames is what each shelf rectangle opens, by rectangle.
var shopOrderShelfNames = [...]string{"armour", "weapons", "magic", "books"}

// shopOrderCompare is below, at or above zero as a stands before, ties with or
// stands after b: a later class stands after, inside a class a later kind, and
// inside a class and kind a dearer price.
func shopOrderCompare(tbl *mapload.Table, a, b ShopItem) int {
	x, y := shopOrderOf(a.Code, tbl), shopOrderOf(b.Code, tbl)
	return cmp.Or(cmp.Compare(x.class, y.class), cmp.Compare(x.kind, y.kind), cmp.Compare(a.Price, b.Price))
}

// shopOrderInOrder is whether b may stand after a in a shelf.
func shopOrderInOrder(tbl *mapload.Table, a, b ShopItem) bool {
	return shopOrderCompare(tbl, a, b) <= 0
}

// shopOrderBuy buys one unit through the shelf grid and the Buy button. It
// takes the single unit nearest the middle of its shelf that the purse covers,
// that the pack holds none of, and whose code, and whose price with its class
// and kind, no other element of the shelf shares, so selling it back has
// exactly one cell the order allows. It returns the unit, the shelf rectangle
// that opens its shelf and the cell it stood in.
func shopOrderBuy(t *testing.T, app *ui.App, s *townScreen, shelves [numShopShelves][]ShopItem) (ShopItem, int, int) {
	t.Helper()
	carried := map[uint16]bool{}
	for _, st := range s.shopPackStacks() {
		carried[st.Code] = true
	}
	room, at, sides := -1, -1, 0
	for r, sh := range shopRoomShelves {
		items := shelves[sh.shelf]
		for k, item := range items {
			if item.Count != 1 || item.Price <= 0 || int(item.Price) > s.sess.Town.Gold() ||
				carried[uint16(item.Code)] || min(k, len(items)-1-k) <= sides {
				continue
			}
			key := shopOrderOf(item.Code, s.in.Table)
			if slices.ContainsFunc(slices.Delete(slices.Clone(items), k, k+1), func(e ShopItem) bool {
				return e.Code == item.Code || e.Price == item.Price && shopOrderOf(e.Code, s.in.Table) == key
			}) {
				continue
			}
			room, at, sides = r, k, min(k, len(items)-1-k)
		}
	}
	if room < 0 {
		t.Fatal("no single unit inside a shelf can be bought and sold back into one cell")
	}
	item := shelves[shopRoomShelves[room].shelf][at]
	point := shopPointer(t, app)
	cell := shopOrderShow(t, app, s, room, at)
	if v := s.ShopScreen(); v.Shelf[cell].Price != item.Price {
		t.Fatalf("shelf %d cell %d paints %d at offset %d, model %d", room, at, v.Shelf[cell].Price, v.ShelfOffset, item.Price)
	}
	gold := s.sess.Town.Gold()
	point("shelf", cell, "press", "release")
	point("button", 1, "press", "release")
	if got := gold - s.sess.Town.Gold(); got != int(item.Price) {
		t.Fatalf("buying %s cost %d", shopOrderLabelIn(s.in, item), got)
	}
	return item, room, at
}

// shopOrderShow opens the room's shelf rectangle, pages with the down arrow
// until element at is painted and returns the cell that paints it.
func shopOrderShow(t *testing.T, app *ui.App, s *townScreen, room, at int) int {
	t.Helper()
	shopPointer(t, app)("shelf_pick", room, "press", "release")
	down := shopOrderControl(t, ui.ShopControlArrowDown)
	for n := 0; s.ShopScreen().ShelfOffset+len(ui.ShopScreenView{}.Shelf) <= at && n < 64; n++ {
		shopTap(t, app, down, "press", "release")
	}
	v := s.ShopScreen()
	cell := at - v.ShelfOffset
	if cell < 0 || cell >= len(v.Shelf) {
		t.Fatalf("shelf %d cell %d is not painted at offset %d", room, at, v.ShelfOffset)
	}
	return cell
}

// shopOrderReturn stages the unit in cell at of the shelf that rectangle room
// opens and sends it back off the table: with the Clear button when clear is
// set, otherwise with a click on its table place.
func shopOrderReturn(t *testing.T, app *ui.App, s *townScreen, room, at int, clear bool) {
	t.Helper()
	point := shopPointer(t, app)
	point("shelf", shopOrderShow(t, app, s, room, at), "press", "release")
	if got := len(s.sess.Shop.Table()); got != 1 {
		t.Fatalf("a click on shelf %d cell %d left %d places on the table, want 1", room, at, got)
	}
	if clear {
		point("button", 0, "press", "release")
	} else {
		point("table", 0, "press", "release")
	}
	for n := 0; len(s.sess.Shop.Table()) != 0 && n < 120; n++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(s.sess.Shop.Table()); got != 0 {
		t.Fatalf("returning the unit left %d places on the table", got)
	}
}

// shopOrderPackStack is the index of the pack stack holding item's code.
func shopOrderPackStack(t *testing.T, s *townScreen, item ShopItem) int {
	t.Helper()
	for k, st := range s.shopPackStacks() {
		if st.Code == uint16(item.Code) {
			return k
		}
	}
	t.Fatalf("the pack holds no %s", shopOrderLabelIn(s.in, item))
	return -1
}

// shopOrderNewStock is the pack stack to sell whose code its shelf lacks: the
// one with the most elements on both sides of it in the shelf order, so the unit
// opens a new element inside the shelf, where an unsorted append could not put
// it.
func shopOrderNewStock(t *testing.T, s *townScreen) int {
	t.Helper()
	best, sides := -1, 0
	for k, st := range s.shopPackStacks() {
		item := shopItemFromInstance(st.Instance(), 1)
		shelf := s.sess.Shop.Shelf(shopShelfForItem(item, s.sess.Shop.tbl))
		if item.Price <= 0 || slices.ContainsFunc(shelf, func(e ShopItem) bool { return e.Code == item.Code }) {
			continue
		}
		before, after := 0, 0
		for _, e := range shelf {
			if shopOrderCompare(s.in.Table, e, item) <= 0 {
				before++
			} else {
				after++
			}
		}
		if min(before, after) > sides {
			best, sides = k, min(before, after)
		}
	}
	if best < 0 {
		t.Fatal("no pack stack opens a new element inside its shelf")
	}
	return best
}

// shopOrderSell clicks one unit of pack stack k onto the table and presses
// Sell, after the pack's arrows bring the stack into the strip, and returns
// the unit sold.
func shopOrderSell(t *testing.T, app *ui.App, s *townScreen, k int) ShopItem {
	t.Helper()
	item := shopItemFromInstance(s.shopPackStacks()[k].Instance(), 1)
	strip := len(ui.ShopScreenView{}.Pack)
	right := shopOrderControl(t, ui.ShopControlPackRight)
	for n := 0; k+1-s.packBase >= strip && n < 64; n++ {
		shopTap(t, app, right, "press", "release")
	}
	left := shopOrderControl(t, ui.ShopControlPackLeft)
	for n := 0; k+1-s.packBase < 0 && n < 64; n++ {
		shopTap(t, app, left, "press", "release")
	}
	cell := k + 1 - s.packBase
	if cell < 0 || cell >= strip || s.ShopScreen().Pack[cell].Price != shopHalfUp(item.Price) {
		t.Fatalf("pack stack %d is not painted in the strip at offset %d", k, s.packBase)
	}
	point := shopPointer(t, app)
	gold := s.sess.Town.Gold()
	point("pack", cell, "press", "release")
	if len(s.sess.Shop.Table()) == 0 {
		t.Fatalf("a click on pack cell %d put nothing on the table", cell)
	}
	point("button", 2, "press", "release")
	if got, want := s.sess.Town.Gold()-gold, int(shopHalfUp(item.Price)); got != want {
		t.Fatalf("selling %s paid %d, want %d", shopOrderLabelIn(s.in, item), got, want)
	}
	return item
}

// shopOrderF2Save presses F2 in the shop, which saves at once and opens the
// game menu, and returns the one save it wrote, which must be a SAV. Escape
// returns to the shop.
func shopOrderF2Save(t *testing.T, app *ui.App, store SaveStore) string {
	t.Helper()
	before, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenGameMenu {
		t.Fatalf("F2: screen %v, err %v", app.Screen(), err)
	}
	after, err := store.List()
	if err != nil || len(after) != len(before)+1 {
		t.Fatalf("F2 left %d saves after %d: %v", len(after), len(before), err)
	}
	for _, e := range after {
		if slices.ContainsFunc(before, func(b SaveEntry) bool { return b.Name == e.Name }) {
			continue
		}
		raw, err := store.Read(e.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sav.Open(raw); !IsOriginal(e.Name) || err != nil {
			t.Fatalf("F2 wrote %s, not a SAV: %v", e.Name, err)
		}
		if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenTown {
			t.Fatalf("Escape from the game menu: screen %v, err %v", app.Screen(), err)
		}
		t.Logf("F2 wrote %s, %d bytes", e.Name, len(raw))
		return e.Name
	}
	t.Fatal("F2 wrote no new save")
	return ""
}

// shopOrderControl is the centre of the frame pixels ui.ShopControlAt answers
// with kind; the layout is 640x480, where window and frame pixels coincide.
func shopOrderControl(t *testing.T, kind ui.ShopControlKind) image.Point {
	t.Helper()
	var r image.Rectangle
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			if c, ok := ui.ShopControlAt(image.Pt(x, y)); ok && c.Kind == kind {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	p := r.Min.Add(r.Size().Div(2))
	if c, ok := ui.ShopControlAt(p); !ok || c.Kind != kind {
		t.Fatalf("no frame pixel answers shop control %d", kind)
	}
	return p
}

// shopOrderLoad activates the load window's row for file through App, which
// must bring the player to the town.
func shopOrderLoad(t *testing.T, app *ui.App, list ui.SaveList, file string) {
	t.Helper()
	label := ""
	for _, row := range list() {
		if row.Name == file || row.Name == localOriginalSaveToken(file) {
			label = row.Label
		}
	}
	if label == "" {
		t.Fatalf("the load window lists no %s", file)
	}
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(label); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown {
		t.Fatalf("LOAD of %s stayed at %s: %s", file, app.Screen(), app.HeadlessMessage())
	}
}

func shopOrderSame(a, b []ShopItem) bool {
	return slices.EqualFunc(a, b, func(x, y ShopItem) bool {
		return x.Code == y.Code && x.Price == y.Price && x.Count == y.Count
	})
}

func shopOrderLabel(f *FrontEnd, item ShopItem) string {
	return shopOrderLabelIn(&f.InstallResources, item)
}

func shopOrderLabelIn(f *InstallResources, item ShopItem) string {
	selector := 0
	if font := f.Font.Value(); font != nil {
		selector = font.Selector
	}
	return fmt.Sprintf("%#04x %q at %d", uint16(item.Code),
		asciiLabel([]byte(itemName(item.Code, f.Table)), selector), item.Price)
}

// shopOrderEnds names a shelf's first three and last two elements.
func shopOrderEnds(f *FrontEnd, items []ShopItem) string {
	return shopOrderEndsIn(&f.InstallResources, items)
}

func shopOrderEndsIn(f *InstallResources, items []ShopItem) string {
	var parts []string
	for k, item := range items {
		if k < 3 || k >= len(items)-2 {
			parts = append(parts, fmt.Sprintf("%d: %s x%d", k, shopOrderLabelIn(f, item), item.Count))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "; " + strings.Join(parts, "; ")
}
