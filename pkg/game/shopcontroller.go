package game

import (
	"image"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// shopController is the shop page of both games: it trades the customer
// party's packs against one deal and holds the page's presentation state.
// A game gives it its deal, its party and its room as data. The first
// game's city graph is the session's own state, absent in the second game,
// so a page move runs as the city transaction only where the graph exists.
type shopController struct {
	// sess and in are the campaign the page trades in and its install.
	sess *CampaignSession
	in   *InstallResources
	// dealOf selects the deal of a session; nil selects the first game's,
	// the session's Shop. The customers are the session's Carried party. A
	// city transaction runs them over its candidate session.
	dealOf func(*CampaignSession) *Shop
	// bag is the one pack the strip shows and the deal trades with, the same
	// whichever member the panel shows, and bagUnit the form an item takes
	// on entering it; a nil bag is the shown customer's own pack.
	bag     func(*CampaignSession) *[]sim.ItemInstance
	bagUnit func(sim.ItemInstance) sim.ItemInstance
	// shop is the room the page stands in.
	shop shopHost
	// racks answers the room's four racks now; nil is the first game's
	// (shopRoomShelves).
	racks func() [4]shopRack
	// value prices a customer's item put on the table; nil is the first
	// game's table price (shopItemValue).
	value func(sim.ItemInstance) int32
	// packUnits bounds the units the customer's pack and his offered places
	// hold together, what the game's SAVE writes; zero is no bound.
	packUnits int
	// label names an item as a cell's text, for a page with no item
	// pictures; nil leaves the text empty.
	label func(sim.ItemInstance) []string
	// The shop screen's own state. shopChosen is which of the four rectangles
	// in the merchant's room has been clicked, shopNoShelf while none has;
	// shopShelf is the shelf that pairs with it; the two bases are the first
	// item each scrolling grid shows. All four are PRESENTATION state and live
	// here rather than on Shop for the reason the conversation does: the model
	// is decidable with no screen anywhere near it.
	shopChosen int
	shopShelf  ShopShelf
	shelfBase  int
	packBase   int
	// tableBase is the first table place the strip shows; it moves only on
	// a deal whose table holds more places than the strip.
	tableBase int
	// shopMember is which roster entry the character panel shows and whose
	// container the bottom strip is bound to (SHOP-PICKER-043's view+0x130).
	// It is presentation for the same reason the four above are.
	shopMember int
	shopBook   bool
	// shopFigures is every party member composed in his own equipment, in
	// roster order, filled by composeShopFaces when a room is entered. It is
	// the composition cache SHOP-LIMIT-049 names as the borrowed panel's own
	// carried state, shared here between the character panel and the town
	// dialogue's speaker figures rather than composed twice.
	shopFigures []*image.RGBA
	// shopFigureMasks is composeShopFaces' own SlotMask beside each entry of
	// shopFigures, in the same roster order (1005 round 2, `SHOP-FIGURE-041`,
	// `DIV-085` restated for the shop): the per-pixel slot lookup the doll's
	// hover, tap and drag all read, built beside the figure by the same
	// composeInventorySubject call rather than derived from it afterwards.
	shopFigureMasks []*ui.SlotMask
	// shopSuppressSlot is the 1-based slot the shop's drag machine last lifted
	// off the shown member's doll, or 0. shopSuppressFigure and
	// shopSuppressMask are the figure and mask composed with that slot cleared,
	// and shopSuppressMember is the roster index they were composed for:
	// ShopScreen substitutes them only while the panel still shows that member.
	shopSuppressSlot   int
	shopSuppressMember int
	shopSuppressFigure *image.RGBA
	shopSuppressMask   *ui.SlotMask
	// shopIconCache is the base item pictures the screen has resolved, keyed
	// by item code and held across frames. Animated stars are a cell-paint
	// layer and never mutate or duplicate these pictures.
	shopIconCache map[uint16]*image.RGBA
	// shopSpellAtlasImg/shopSpellAtlasTried/shopSpellIcons are the shop's
	// install-owned cache for the same spellbook strip the mission view uses.
	shopSpellAtlasImg   *bmp.Image
	shopSpellAtlasTried bool
	shopSpellIcons      map[uint16]*image.RGBA
}

// shopHost is what the page asks of the room that shows it. A nil hook is a
// room without that part: no sound, no keeper, no character panel.
type shopHost struct {
	// shown reports whether the page is the open room; nil is always.
	shown func() bool
	// sound requests the room's sound bound to an interface event.
	sound func(on string)
	// scene is the room's scene; events name its keeper events, in
	// shopKeeperEvent order.
	scene  func() *town.Scene
	events [shopKeeperEvents]string
	// speak is the keeper's response to an item put on the table.
	speak func(item ShopItem)
	// fresh is the party a page with no customer yet trades for.
	fresh func() []mapload.PartyMember
	// dress fills the room's parts of a frame: art, scene, tip panel and
	// character panel.
	dress func(v *ui.ShopScreenView)
	// stepped follows the character picker from one member to another;
	// stats toggles the character panel's mode; leave closes the room.
	stepped func(from, to mapload.PartyMember)
	stats   func()
	leave   func() bool
	// chose follows a rack the player chose; nil keeps the choice on the page.
	chose func(i int)
	// page is the room's shop drawing rules; nil keeps the zero rules.
	page func() *town.ShopSpec
	// covered reports a modal dialogue standing over the page: the shelf grid
	// then shows no stock. Nil is never covered.
	covered func() bool
	// faces is the room's part of a recomposition of the party's figures.
	faces func(src entrySource, party []mapload.PartyMember, player *image.RGBA, players map[int]*image.RGBA)
	// face is member i's pane figure, slot map and worn item text.
	face func(i int) (*image.RGBA, *ui.SlotMask, [12][]string)
	// pane is the room's character pane; the book control is the pane's.
	pane func() *town.PaneSpec
}

// paneBook reports whether the room's pane has the spellbook corner.
func (t *shopController) paneBook() bool {
	if t.shop.pane == nil {
		return false
	}
	p := t.shop.pane()
	return p != nil && p.Book
}

// shopKeeperEvent is a keeper event the page raises in the room's scene.
type shopKeeperEvent int

const (
	// shopKeeperBought is a buy the purse covers, shopKeeperRefused one it
	// does not, shopKeeperSold a sale with a non-zero value and
	// shopKeeperRack a chosen rack with its index.
	shopKeeperBought shopKeeperEvent = iota
	shopKeeperRefused
	shopKeeperSold
	shopKeeperRack
	shopKeeperEvents
)

func (t *shopController) rack() *[4]shopRack {
	if t.racks != nil {
		r := t.racks()
		return &r
	}
	return &shopRoomShelves
}

func (t *shopController) rackName(i int) string {
	if t.racks == nil {
		return shopRoomShelfName(i, t.in.Table)
	}
	if i < 0 || i >= len(t.rack()) {
		return ""
	}
	return t.rack()[i].name
}

func (t *shopController) labelled(c ui.ShopCell, item sim.ItemInstance) ui.ShopCell {
	if t.label != nil {
		c.Text = t.label(item)
	}
	return c
}

// deal is the shop the page trades with.
func (t *shopController) deal() *Shop {
	if t.dealOf != nil {
		return t.dealOf(t.sess)
	}
	return t.sess.Shop
}

// customers is the customer party's slot in the session.
func (t *shopController) customers() *[]mapload.PartyMember {
	return &t.sess.Carried
}

func (t *shopController) shownRoom() bool {
	return t != nil && (t.shop.shown == nil || t.shop.shown())
}

func (t *shopController) play(on string) {
	if t.shop.sound != nil {
		t.shop.sound(on)
	}
}

func (t *shopController) keeper(e shopKeeperEvent, args ...int) {
	if t.shop.scene == nil {
		return
	}
	if s := t.shop.scene(); s != nil {
		s.Event(t.shop.events[e], args...)
	}
}

func (t *shopController) say(item ShopItem) {
	if t.shop.speak != nil {
		t.shop.speak(item)
	}
}

func (t *shopController) toggleStats() {
	if t.shop.stats != nil {
		t.shop.stats()
	}
}

func (t *shopController) leave() {
	if t.shop.leave != nil {
		t.shop.leave()
	}
}

// bindShop binds the first game's town room to the page.
func (t *townScreen) bindChapterShop() {
	t.shop = shopHost{
		shown: func() bool { return t.room == roomShop },
		sound: func(on string) { t.roomSound("shop", on) },
		scene: t.shopPage,
		events: [shopKeeperEvents]string{shopKeeperBought: "merchant-yes", shopKeeperRefused: "merchant-no",
			shopKeeperSold: "merchant-yes", shopKeeperRack: "rack"},
		speak: t.shopItemResponse,
		fresh: t.nextParty,
		dress: func(v *ui.ShopScreenView) {
			v.Art = t.shopArt()
			v.Scene = roomScene{t.shopPage()}
			v.TipPanel = t.roomTipView(roomShop, t.shopTip)
			v.Character, _, _ = t.heroPane(t.chapterPane("shop"))
			v.ButtonStyle = t.in.roomButtonStyle(t.townDescription())
		},
		face: func(i int) (*image.RGBA, *ui.SlotMask, [12][]string) {
			mask, info := t.shopDollHover(i)
			return t.shopFigure(i), mask, info
		},
		pane: func() *town.PaneSpec { return t.chapterPane("shop") },
		stepped: func(from, to mapload.PartyMember) {
			t.clearSchoolSelection()
			t.retargetSchoolColumn(from.Mage, to.Mage)
		},
		stats: func() { t.townStats = !t.townStats },
		leave: t.Back,
		faces: t.shopFaces,
		page: func() *town.ShopSpec {
			if d := t.townDescription(); d != nil && d.Room("shop") != nil {
				return d.Room("shop").Shop
			}
			return nil
		},
		covered: func() bool { return t.room == roomTalk && t.dialogueBuilding == TownShop },
	}
}

// pageRules is the room's shop drawing rules, the first game's without any.
func (t *shopController) pageRules() town.ShopSpec {
	if t.shop.page != nil {
		if p := t.shop.page(); p != nil {
			return *p
		}
	}
	return town.ShopSpec{}
}

// packStack is the stack of the pack's list that strip cell i shows, -1 for
// the money element. The list is the money element and the stacks: the
// money first, or after the stacks when the room's rules say so. An index
// past the stacks names no stack; its caller refuses it.
func (t *shopController) packStack(i int) int {
	k := t.packBase + i
	if !t.pageRules().MoneyLast {
		return k - 1
	}
	if k == len(t.shopPackStacks()) {
		return -1
	}
	return k
}

// packCell is the strip cell that shows stack k of the pack's list, the
// inverse of packStack.
func (t *shopController) packCell(k int) int {
	if t.pageRules().MoneyLast {
		return k - t.packBase
	}
	return k + 1 - t.packBase
}
