package game

import (
	"image"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// shopBinding is one row of what the shop page is bound to: the deal it
// trades with, the pack the strip shows, how its racks, cells and prices read,
// how a gesture settles, and the room that shows it. A game, and a mission
// visit over the town's row, are rows; bindShop is the one binder.
type shopBinding struct {
	// adopt readies the town host the row's room stands in.
	adopt func()
	// fresh drops the page's presentation state with the binding.
	fresh bool
	// dealOf selects the deal of a session. The customers are the session's
	// Carried party. A city transaction runs them over its candidate
	// session. cityGraph reports that the deal is the session's own Shop,
	// which the city object graph holds, so a trade mutates that graph.
	dealOf    func(*CampaignSession) *Shop
	cityGraph bool
	// bag is the one pack the strip shows and the deal trades with, the same
	// whichever member the panel shows, and bagUnit the form an item takes
	// on entering it; a nil bag is the shown customer's own pack, which
	// takes its items as they are.
	bag     func(*CampaignSession) *[]sim.ItemInstance
	bagUnit func(sim.ItemInstance) sim.ItemInstance
	// racks answers the room's four racks now and rackLabel names rack i.
	racks     func() [4]shopRack
	rackLabel func(i int) string
	// value prices a customer's item put on the table.
	value      func(sim.ItemInstance) int32
	shownPrice func(sim.ItemInstance) int32
	// packUnits bounds the units the customer's pack and his offered places
	// hold together, what the game's SAVE writes; zero is no bound.
	packUnits int
	// label names an item as a cell's text, for a page with no item
	// pictures; it answers no text for a page that has pictures.
	label func(sim.ItemInstance) []string
	// described is the form of an item a cell's caption describes.
	described func(sim.ItemInstance) sim.ItemInstance
	// settle runs one gesture and commits what it changed to the place the
	// customer's goods live. wornFixed refuses every gesture that touches
	// the doll.
	settle    func(act func() ui.TownAction) ui.TownAction
	wornFixed bool
	speech    shopSpeechRules
	// host is the room the page stands in.
	host shopHost
}

// bindShop binds the page to row.
func (t *townScreen) bindShop(row shopBinding) {
	row.adopt()
	if row.fresh {
		t.shopController = shopController{sess: t.sess, in: t.in}
	}
	t.row = row
	t.shop = row.host
}

// hostShopRow is the row the host's rules choose for the town's shop room.
func (t *townScreen) hostShopRow() shopBinding {
	if t.hostRules().Shop == town.ShopBank {
		return t.bankShopRow()
	}
	return t.chapterShopRow()
}

// chapterShopRow is the first game's town room as the page's row: the
// session's own deal, the fixed racks and the table's prices.
func (t *townScreen) chapterShopRow() shopBinding {
	return shopBinding{
		adopt:     func() {},
		speech:    firstShopSpeech,
		dealOf:    func(s *CampaignSession) *Shop { return s.Shop },
		cityGraph: true,
		bagUnit:   sim.ItemInstance.Clone,
		racks:     func() [4]shopRack { return shopRoomShelves },
		rackLabel: func(i int) string { return shopRoomShelfName(i, t.in.Table) },
		value:     t.tableItemValue,
		label:     func(sim.ItemInstance) []string { return nil },
		described: func(in sim.ItemInstance) sim.ItemInstance { return in },
		settle:    func(act func() ui.TownAction) ui.TownAction { return act() },
		host: shopHost{
			shown: func() bool { return t.room == roomShop },
			sound: t.shopRoomSound,
			scene: t.shopPage,
			events: [shopKeeperEvents]string{shopKeeperBought: "merchant-yes", shopKeeperRefused: "merchant-no",
				shopKeeperSold: "merchant-yes", shopKeeperRack: "rack"},
			speak: t.shopItemResponse,
			fresh: t.nextParty,
			dress: func(v *ui.ShopScreenView) {
				v.Art = t.shopArt()
				v.Scene = roomScene{t.shopPage()}
				v.TipPanel = t.roomTipView("shop")
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
		},
	}
}
