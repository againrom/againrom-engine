package game

import (
	"fmt"
	"image"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// The events the shop room raises for its scene's actors (R2-ENGINE-269): a
// changed category with its number, a buy the gold covers, a buy it does not,
// and a sale that credits gold.
const (
	secondShopCategoryChanged = "category-changed"
	secondShopBought          = "bought"
	secondShopRefused         = "refused"
	secondShopSold            = "sold"
)

// secondShopEvents are the events the second game's shop rooms raise.
var secondShopEvents = []string{secondShopCategoryChanged, secondShopBought, secondShopRefused, secondShopSold}

// secondShopOpen is the room entry: the deal starts empty, the stock is drawn
// when it must be (R2-ENGINE-266) and the first category shows.
func secondShopOpen(t *townScreen, _ string) {
	c := t.state()
	tables, ok := secondShopTables(t.in.Table)
	if !ok {
		return
	}
	stream := t.draws.stream(random.ShopStock)
	c.adoptLegacyHero(tables)
	c.shop.enter(tables, c.current.id, stream.Raw, c.heroBag())
	t.shopChosen, t.shopShelf, t.shelfBase, t.packBase, t.tableBase, t.shopMember = 0, 0, 0, 0, 0, 0
}

// bindShop binds the composed town's shop room to the one shop page: the
// second campaign's deal and bag, the campaign party as the customers whose
// doll the pane shows, the room's art, scene and rectangles from
// the town description and its keeper events (R2-ENGINE-269); the room's
// pane spec adds the character panel and its book (dressShop).
func (t *townScreen) bindBankShop() {
	if t.adapter == nil {
		t.adapter = destinationTownCampaign{t}
		t.defaults = townDescriptions["rom2-town1"]
	}
	t.shopController = shopController{sess: t.sess, in: t.in,
		dealOf: func(s *CampaignSession) *Shop {
			c := secondOf(s)
			if c == nil {
				return nil
			}
			c.shop.rules()
			return &c.shop.Shop
		},
		bag: func(s *CampaignSession) *[]sim.ItemInstance {
			c := secondOf(s)
			if c == nil {
				return new([]sim.ItemInstance)
			}
			return c.heroBag()
		},
		bagUnit:   secondshop.Unit,
		racks:     t.shopRacks,
		packUnits: maxSecondBagUnits,
		value:     func(in sim.ItemInstance) int32 { return in.Price },
		label: func(in sim.ItemInstance) []string {
			it, _ := secondshop.FromInstance(in)
			return t.shopItemName(it)
		},
		shop: shopHost{
			shown: t.bankShopShown,
			scene: t.shopScene,
			events: [shopKeeperEvents]string{shopKeeperBought: secondShopBought, shopKeeperRefused: secondShopRefused,
				shopKeeperSold: secondShopSold, shopKeeperRack: secondShopCategoryChanged},
			dress: t.dressShop,
			leave: func() bool {
				d, room, ok := t.composedRoom()
				if ok {
					t.scene(d, d.Square.Name).LeaveRoom(room)
				}
				return ok
			},
			chose: func(i int) {
				if c := t.state(); c != nil {
					c.shop.choose(i)
				}
			},
			stats: func() { t.townStats = !t.townStats },
			face:  t.heroFaceView,
			faces: func(entrySource, []mapload.PartyMember, *image.RGBA, map[int]*image.RGBA) {
				if t.composed != nil {
					t.composed.hero = nil
				}
			},
			page: func() *town.ShopSpec {
				if d, room, ok := t.composedRoom(); ok && d.Room(room) != nil {
					return d.Room(room).Shop
				}
				return nil
			},
			pane: func() *town.PaneSpec {
				if d, room, ok := t.composedRoom(); ok && d.Room(room) != nil {
					return d.Room(room).Pane
				}
				return nil
			},
		},
	}
}

// secondOf is a session's second campaign, nil without one.
func secondOf(s *CampaignSession) *secondCampaign {
	if s.Town == nil {
		return nil
	}
	return s.Town.second
}

// AtTownShop reports whether the composed town's shop room is shown. The
// page's chosen rack follows the deal's category, which a LOAD restores.
func (t *townScreen) bankShopShown() bool {
	c := t.state()
	if c == nil || c.room != secondTownShop {
		return false
	}
	if _, _, ok := t.composedRoom(); !ok {
		return false
	}
	if t.shopChosen != c.shop.category {
		t.shopChosen, t.shopShelf, t.shelfBase = c.shop.category, ShopShelf(c.shop.category), 0
	}
	return true
}

// shopScene is the shown shop room's scene, nil when none is shown.
func (t *townScreen) shopScene() *town.Scene {
	d, room, ok := t.composedRoom()
	if !ok {
		return nil
	}
	return t.scene(d, room)
}

// shopRacks are the four categories as the page's racks: rack k shows
// category k, and a category whose shelf button has a condition is stocked
// only while it holds (R2-ENGINE-259). Their names are not published.
func (t *townScreen) shopRacks() [4]shopRack {
	var racks [4]shopRack
	for k := range racks {
		racks[k] = shopRack{ShopShelf(k), t.shelfOpen(k), ""}
	}
	return racks
}

// shelfOpen reports whether category k exists in the shown room.
func (t *townScreen) shelfOpen(k int) bool {
	d, room, ok := t.composedRoom()
	if !ok {
		return true
	}
	if b := shopButton(d.Room(room), fmt.Sprintf("shelf/%d", k)); b != nil {
		return t.scene(d, room).ButtonShown(*b)
	}
	return true
}

// shopButton is the room's button named name, nil without one.
func shopButton(r *town.RoomSpec, name string) *town.ButtonSpec {
	if r == nil {
		return nil
	}
	for i := range r.Buttons {
		if r.Buttons[i].Name == name {
			return &r.Buttons[i]
		}
	}
	return nil
}

// dressShop fills the room's parts of a frame: the town's shop art
// (R2-ENGINE-249), its scene's interior, and the first layout with the
// town's category and keeper rectangles (R2-ENGINE-354, DIV-3064); a closed
// category has no rectangle; the room's shop rules move the plaque labels.
// A room with a pane spec shows the character
// panel (R2-SESSION-204, R2-SESSION-217).
func (t *townScreen) dressShop(v *ui.ShopScreenView) {
	l := ui.FirstShopLayout()
	l.Character, l.Book, l.TableScroll = false, false, true
	v.Layout = &l
	d, room, ok := t.composedRoom()
	if !ok {
		return
	}
	r := d.Room(room)
	v.ButtonStyle = t.in.roomButtonStyle(d)
	v.BuyDebit = d.RoomButtons != nil && d.RoomButtons.BuyDebit
	racks := t.shopRacks()
	for k := range l.ShelfPick {
		l.ShelfPick[k] = image.Rectangle{}
		if b := shopButton(r, fmt.Sprintf("shelf/%d", k)); b != nil && racks[k].stocked {
			l.ShelfPick[k] = b.Rect.Rectangle()
		}
	}
	if b := shopButton(r, "merchant"); b != nil {
		l.Merchant = b.Rect.Rectangle()
	}
	if s := r.Shop; s != nil {
		for i, o := range s.LabelOffset {
			l.LabelOffset[i] = o.Pt()
		}
	}
	if p := r.Pane; p != nil {
		// The pane and its doll are the campaign party's, the shop's
		// customers; the strip is the campaign's bag (DIV-4081, DIV-3043).
		l.Character, l.Book, l.BookOffset = true, p.Book, p.BookOffset.Pt()
		hero, mask, info := t.heroPane(p)
		v.Character = hero
		v.Figure, v.SlotMask, v.OrdinaryDollMask, v.SlotInfo = hero.Figure, mask, mask, info
	}
	a := t.roomArt(d, room)
	if a.shop == nil {
		// The pane's book draws on the mission bar's book art, as the first
		// game's shop does.
		a.shop = shopArtOf(a.page)
		a.shop.Book = t.in.BottomHUDArt.Value()
	}
	v.Art = a.shop
	v.Scene = roomScene{t.scene(d, room)}
	v.TipPanel = t.roomTipView(room)
}

// shopItemName is an item's name parts, the cell's name first (DIV-3046):
// the item's line in the shipped item-name table, else its data.bin row,
// material and class names; then an enchanted item's spell and level. A book
// is its spell's data.bin name.
func (t *townScreen) shopItemName(it secondshop.Item) []string {
	tab := t.in.Table
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
			shipped = mapload.ItemDisplayName(tab, data.ItemCode(key), "", "")
		}
	}
	var parts []string
	switch {
	case it.Table == secondshop.Book:
		return []string{collection(tab.Spells, it.Spell)}
	case shipped != "":
		parts = []string{shipped}
	case it.Table == secondshop.Magic:
		parts = []string{collection(tab.MagicItems, it.Row)}
	default:
		rows := map[secondshop.Kind]data.Collection{secondshop.Armor: tab.Armors, secondshop.Shield: tab.Shields, secondshop.Weapon: tab.Weapons}
		parts = []string{collection(rows[it.Table], it.Row), scale(tab.Materials, it.Material), scale(tab.Shapes, it.Class)}
	}
	if it.Level > 0 {
		parts = append(parts, fmt.Sprintf("%s %d", collection(tab.Spells, it.Spell), it.Level))
	}
	return parts
}
