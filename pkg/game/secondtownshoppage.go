package game

import (
	"fmt"
	"image"
	"image/draw"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/random"
	"againrom/pkg/secondshop"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// The lists a shop page's grids show: the shown category with the items
// offered for sale after it, the hero's items with the purchase after them,
// and the information panel's chosen item.
const (
	secondShopStock = "stock"
	secondShopHero  = "hero"
	secondShopInfo  = "info"
)

// secondTownLists are the grid lists the second game's town pages show.
var secondTownLists = []string{secondShopStock, secondShopHero, secondShopInfo}

// The events a shop page raises for its scene's actors (R2-ENGINE-269): a
// changed category with its number, a buy the gold covers, a buy it does not,
// and a sale that credits gold.
const (
	secondShopCategoryChanged = "category-changed"
	secondShopBought          = "bought"
	secondShopRefused         = "refused"
	secondShopSold            = "sold"
)

// secondShopEvents are the events the second game's town pages raise.
var secondShopEvents = []string{secondShopCategoryChanged, secondShopBought, secondShopRefused, secondShopSold}

// secondShopControl is one row of the shop page: its text and its action.
type secondShopControl struct {
	text string
	act  func(t *secondCampaignScreen, c *secondCampaign)
}

// secondShopButtons bind the page's button names to its rows.
var secondShopButtons = map[string]string{"undo": "UNDO", "buy": "BUY", "sell": "SELL", "exit": "EXIT",
	"stock-back": "SHELF <", "stock-on": "SHELF >", "hero-back": "MINE <", "hero-on": "MINE >",
	"shelf/0": "SHELF 1", "shelf/1": "SHELF 2", "shelf/2": "SHELF 3", "shelf/3": "SHELF 4"}

// secondShopOpen is the page entry: the deal starts empty, the stock is drawn
// when it must be (R2-ENGINE-266) and both grids show their first entries.
func secondShopOpen(t *secondCampaignScreen, _ string) {
	c := t.state()
	tables, ok := secondShopTables(t.install.Table)
	if !ok {
		return
	}
	stream := t.composed.draws.stream(random.ShopStock)
	c.shop.enter(tables, c.current.id, stream.Raw)
	t.composed.shopOffsets, t.composed.shopInfo = [2]int{}, nil
}

// visibleShelf answers the shown category's entries the page lists: those
// with units left.
func (s *secondShop) visibleShelf() []int {
	var out []int
	for i, it := range s.shelf() {
		if it.Quantity > 0 {
			out = append(out, i)
		}
	}
	return out
}

// shopEvent raises a page event in the shown room's scene.
func (t *secondCampaignScreen) shopEvent(name string, args ...int) {
	if d, room, ok := t.composedRoom(); ok {
		t.scene(d, room).Event(name, args...)
	}
}

// shelfOpen reports whether category k exists on the shown page: a shelf
// button with a condition admits its category only while the condition
// holds (R2-ENGINE-259).
func (t *secondCampaignScreen) shelfOpen(k int) bool {
	d, room, ok := t.composedRoom()
	if !ok {
		return true
	}
	r := d.Room(room)
	if r == nil {
		return true
	}
	name := fmt.Sprintf("shelf/%d", k)
	for _, b := range r.Buttons {
		if b.Name == name {
			return t.scene(d, room).ButtonShown(b)
		}
	}
	return true
}

// shopControls are the shop page's rows: the open categories, each listed
// shelf entry and hero item, the two grids' scroll rows and the four
// buttons in their index order (R2-ENGINE-249). A changed category, a buy
// and a credited sale raise the page's events (R2-ENGINE-269).
func (t *secondCampaignScreen) shopControls(c *secondCampaign) []secondShopControl {
	var out []secondShopControl
	for k := range c.shop.stock {
		if !t.shelfOpen(k) {
			continue
		}
		out = append(out, secondShopControl{fmt.Sprintf("SHELF %d", k+1), func(t *secondCampaignScreen, c *secondCampaign) {
			if c.shop.choose(k) {
				t.composed.shopOffsets[0] = 0
				t.shopEvent(secondShopCategoryChanged, k)
			}
		}})
	}
	for n, i := range c.shop.visibleShelf() {
		out = append(out, secondShopControl{fmt.Sprintf("ITEM %d", n+1), func(t *secondCampaignScreen, c *secondCampaign) {
			it := c.shop.shelf()[i]
			it.Quantity = 1
			if c.shop.take(i) {
				t.composed.shopInfo = &it
			}
		}})
	}
	for i := range c.shop.hero {
		out = append(out, secondShopControl{fmt.Sprintf("MINE %d", i+1), func(t *secondCampaignScreen, c *secondCampaign) {
			it := c.shop.hero[i]
			if c.shop.offer(i) {
				t.composed.shopInfo = &it
			}
		}})
	}
	scroll := func(grid, by int) func(*secondCampaignScreen, *secondCampaign) {
		return func(t *secondCampaignScreen, c *secondCampaign) {
			n := len(t.shopList(c, []string{secondShopStock, secondShopHero}[grid]))
			t.composed.shopOffsets[grid] = max(0, min(t.composed.shopOffsets[grid]+by, n-1))
		}
	}
	buy := func(t *secondCampaignScreen, c *secondCampaign) {
		gold := &t.session.Town.gold
		if cost := c.shop.cost(); cost > 0 {
			event := secondShopBought
			if cost > int64(*gold) {
				event = secondShopRefused
			}
			t.shopEvent(event)
		}
		c.shop.buy(gold)
	}
	sell := func(t *secondCampaignScreen, c *secondCampaign) {
		if c.shop.credit() > 0 {
			t.shopEvent(secondShopSold)
		}
		c.shop.sell(&t.session.Town.gold)
	}
	return append(out,
		secondShopControl{"SHELF <", scroll(0, -1)}, secondShopControl{"SHELF >", scroll(0, 1)},
		secondShopControl{"MINE <", scroll(1, -1)}, secondShopControl{"MINE >", scroll(1, 1)},
		secondShopControl{"UNDO", func(_ *secondCampaignScreen, c *secondCampaign) { c.shop.undo() }},
		secondShopControl{"BUY", buy},
		secondShopControl{"SELL", sell},
		secondShopControl{secondInnExit, nil})
}

// cost is the open purchase's cost, sum P*q (R2-ENGINE-267).
func (s *secondShop) cost() int64 {
	var cost int64
	for _, it := range s.buying {
		cost += it.Price * int64(it.Quantity)
	}
	return cost
}

// credit is the open sale's credit as the page shows it, sum ((P+1)/2)*q
// (R2-ENGINE-267).
func (s *secondShop) credit() int64 {
	var credit int64
	for _, it := range s.selling {
		credit += secondshop.SaleCell(it.Price) * int64(it.Quantity)
	}
	return credit
}

// shopList answers a grid list's entries in display order.
func (t *secondCampaignScreen) shopList(c *secondCampaign, list string) []secondshop.Item {
	s := &c.shop
	switch list {
	case secondShopStock:
		var out []secondshop.Item
		for _, i := range s.visibleShelf() {
			out = append(out, s.shelf()[i])
		}
		return append(out, s.selling...)
	case secondShopHero:
		return append(append([]secondshop.Item(nil), s.hero...), s.buying...)
	case secondShopInfo:
		if t.composed.shopInfo != nil {
			return []secondshop.Item{*t.composed.shopInfo}
		}
	}
	return nil
}

func (t *secondCampaignScreen) shopRows(c *secondCampaign) []ui.TownRow {
	var rows []ui.TownRow
	for _, ctl := range t.shopControls(c) {
		rows = append(rows, ui.TownRow{Text: ctl.text, Choosable: true})
	}
	return rows
}

// chooseShop runs row i; the exit row leaves the page for the square.
func (t *secondCampaignScreen) chooseShop(d *town.Description, c *secondCampaign, room string, i int) ui.TownAction {
	controls := t.shopControls(c)
	if i < 0 || i >= len(controls) {
		return ui.TownAction{}
	}
	if controls[i].act == nil {
		t.scene(d, d.Square.Name).LeaveRoom(room)
		return ui.TownAction{}
	}
	controls[i].act(t, c)
	return ui.TownAction{}
}

// shopControlAt answers the row a click at p reaches on the shop page: a
// listed shelf entry or hero item in its grid, or a button.
func (v secondTownView) shopControlAt(r *town.RoomSpec, p image.Point) (ui.TownSquareControl, bool) {
	c := v.t.state()
	want := ""
	for gi := range r.Grids {
		g := &r.Grids[gi]
		cell := g.CellAt(p)
		if cell < 0 {
			continue
		}
		switch g.List {
		case secondShopStock:
			if n := v.t.composed.shopOffsets[0] + cell; n < len(c.shop.visibleShelf()) {
				want = fmt.Sprintf("ITEM %d", n+1)
			}
		case secondShopHero:
			if n := v.t.composed.shopOffsets[1] + cell; n < len(c.shop.hero) {
				want = fmt.Sprintf("MINE %d", n+1)
			}
		}
	}
	buttons := v.t.scene(v.d, v.room).ShownButtons(r.Buttons)
	if i := town.ButtonAt(buttons, p); want == "" && i >= 0 {
		want = secondShopButtons[buttons[i].Name]
	}
	for i, ctl := range v.t.shopControls(c) {
		if want != "" && ctl.text == want {
			return ui.TownSquareControl{Kind: ui.TownSquareControlDoor, Door: i}, true
		}
	}
	return ui.TownSquareControl{}, false
}

// paintShopGrids draws each grid: a cell's back art, then its entry's
// quantity, name and unit price; the information panel's cell adds the
// hero's gold and the open deal's totals (R2-ENGINE-267).
func (v secondTownView) paintShopGrids(dst *image.RGBA, r *town.RoomSpec) {
	font := v.t.install.Font.Value()
	page := v.t.roomArt(v.d, v.room).page
	c := v.t.state()
	for _, g := range r.Grids {
		list := v.t.shopList(c, g.List)
		offset := 0
		switch g.List {
		case secondShopStock:
			offset = v.t.composed.shopOffsets[0]
		case secondShopHero:
			offset = v.t.composed.shopOffsets[1]
		}
		for k := 0; k < g.Count; k++ {
			cell := g.Cell(k)
			drawAt(dst, page.Picture(g.Back), cell.Min, draw.Over)
			if font == nil {
				continue
			}
			var lines []string
			if offset+k < len(list) {
				lines = v.t.shopItemLines(c, g.List, list[offset+k], g.List == secondShopStock && c.shop.offeredAt(offset+k))
			}
			if g.List == secondShopInfo {
				lines = append(lines, v.t.shopTotals(c)...)
			}
			h := (cell.Dy() - 4) / max(1, len(lines))
			for n, line := range lines {
				at := image.Rect(cell.Min.X+2, cell.Min.Y+2+n*h, cell.Max.X-2, cell.Min.Y+2+(n+1)*h)
				ui.DrawTownCaption(dst, font, line, at)
			}
		}
	}
}

// shopItemLines are a cell's text: the units, the item's name one word a line
// (at most three lines) and the unit price, (P+1)/2 for an item offered for sale (R2-ENGINE-267); the
// information panel adds the other parts of the name.
func (t *secondCampaignScreen) shopItemLines(c *secondCampaign, list string, it secondshop.Item, offered bool) []string {
	price := it.Price
	if offered {
		price = secondshop.SaleCell(price)
	}
	parts := t.shopItemName(it)
	name := strings.Fields(parts[0])
	if len(name) > 3 {
		name = append(name[:2], strings.Join(name[2:], " "))
	}
	if list == secondShopInfo {
		name = append(name, parts[1:]...)
	}
	return append(append([]string{fmt.Sprintf("%d", it.Quantity)}, name...), fmt.Sprintf("%d", price))
}

// offeredAt reports whether entry n of the stock grid's list is an offered
// item: the offers follow the visible shelf, so the place decides, not the value.
func (s *secondShop) offeredAt(n int) bool { return n >= len(s.visibleShelf()) }

// shopTotals are the panel's gold and the open deal's cost and credit:
// sum P*q of the purchase and sum ((P+1)/2)*q of the sale (R2-ENGINE-267).
func (t *secondCampaignScreen) shopTotals(c *secondCampaign) []string {
	return []string{fmt.Sprintf("%d", t.session.Town.gold), fmt.Sprintf("-%d", c.shop.cost()), fmt.Sprintf("+%d", c.shop.credit())}
}

// shopItemName is an item's name parts, the cell's name first (DIV-3046):
// the item's line in the shipped item-name table, else its data.bin row,
// material and class names; then an enchanted item's spell and level. A book
// is its spell's data.bin name.
func (t *secondCampaignScreen) shopItemName(it secondshop.Item) []string {
	tab := t.install.Table
	if tab == nil {
		return []string{""}
	}
	name := func(c interface{ Len() int }, named func(int) string, i int) string {
		if c == nil || i < 0 || i >= c.Len() {
			return ""
		}
		return named(i)
	}
	collection := func(c data.Collection, i int) string {
		if c == nil {
			return ""
		}
		return name(c, c.EntryName, i)
	}
	scale := func(c data.ScaleTable, i int) string {
		if c == nil {
			return ""
		}
		return name(c, c.EntryName, i)
	}
	shipped := ""
	if tables, ok := secondShopTables(tab); ok {
		if key, ok := tables.NameKey(it); ok {
			shipped, _ = tab.Names.NameFor(data.ItemCode(key))
		}
	}
	var parts []string
	switch {
	case it.Kind == secondshop.Book:
		return []string{collection(tab.Spells, it.Spell)}
	case shipped != "":
		parts = []string{shipped}
	case it.Kind == secondshop.Magic:
		parts = []string{collection(tab.MagicItems, it.Row)}
	default:
		rows := map[secondshop.Kind]data.Collection{secondshop.Armor: tab.Armors, secondshop.Shield: tab.Shields, secondshop.Weapon: tab.Weapons}
		parts = []string{collection(rows[it.Kind], it.Row), scale(tab.Materials, it.Material), scale(tab.Shapes, it.Class)}
	}
	if it.Level > 0 {
		parts = append(parts, fmt.Sprintf("%s %d", collection(tab.Spells, it.Spell), it.Level))
	}
	return parts
}
