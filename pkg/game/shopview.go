package game

import (
	"fmt"
	"image"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The shop screen's own state and the view it hands across the seam (second
// round).
//
// THE SCREEN OWNS NO TRADE STATE. Which shelf is chosen and how far the two
// grids have been scrolled are presentation, and they live on the town screen
// beside the conversation's own; the shelves, the table and the purse are the
// model's and are read through Shop exactly as the row list read them.

// shopNoShelf is shopChosen's value while no shelf has been clicked, which is
// the state the room is entered in (SHOP-SCREEN-034: view+0x132 is 0x64, outside
// the four, until a shelf is opened).
const shopNoShelf = -1

// shopShelfCols is how many columns the shelf grid has, and so how many cells
// one scroll step moves (SHOP-SCREEN-031's two columns of three, and
// SHOP-SCREEN-033's doubled step). It is spelt here as well as in pkg/ui
// because the two tiers may not share a constant: one owns the geometry, the
// other owns the scroll base.
const shopShelfCols = 2

// shopRoomShelves pairs each of the four rectangles in the merchant's room with
// the shelf this build stocks behind it.
//
// WHICH RECTANGLE HOLDS WHICH KIND IS NOT DECODED. SHOP-SHELF-047 gives four
// hit rectangles and says nothing about their contents.
//
// THE INDEX IS THE HIT RECT'S INDEX (SHOP-SHELF-047), which is not the order
// the four rectangles are stored in and not the order they are drawn in. Hit
// rect 0 is the room's lower right, 1 the lower left, 2 the upper right and 3
// the upper left; the shipped animation folder for i is `4 - i`. The pairing
// below is read off the shipped room picture by what is drawn inside each
// rectangle: consumables upper left, magic upper right, weapons lower left,
// armour lower right. The owner-directed fourth shelf currently contains
// books; scrolls and potions remain outside its model.
var shopRoomShelves = [4]struct {
	shelf   ShopShelf
	stocked bool
	name    string
}{
	{ShelfArmour, true, "armour"},
	{ShelfWeapons, true, "weapons"},
	{ShelfMagic, true, "magic"},
	{ShelfBooks, true, ""},
}

func shopRoomShelfName(room int, table *mapload.Table) string {
	if room < 0 || room >= len(shopRoomShelves) {
		return ""
	}
	if shopRoomShelves[room].shelf == ShelfBooks {
		return itemName(shopBookLabelCode, table)
	}
	return shopRoomShelves[room].name
}

// Open the first stocked shelf on entry: armour, weapons, magic, then books.
func (t *townScreen) openStockedShopShelf() {
	t.shopChosen, t.shopShelf, t.shelfBase = shopNoShelf, ShelfArmour, 0
	for i, room := range shopRoomShelves {
		if room.stocked && len(t.sess.Shop.Shelf(room.shelf)) > 0 {
			t.shopChosen, t.shopShelf = i, room.shelf
			return
		}
	}
}

// AtTownShop is on townscreen.go; ShopScreen and ShopClick complete the seam.

// ShopScreen composes the whole screen for one frame.
func (t *townScreen) ShopScreen() ui.ShopScreenView {
	v := ui.ShopScreenView{Chosen: shopNoShelf}
	if t == nil {
		return v
	}
	v.Art = t.art.shopScreen()
	v.Interior = t.shopInteriorFrame(v.Art)
	v.Font = t.in.Font.Value()
	v.PriceFont = t.in.tipFont()
	v.Words = t.in.Words
	// TipPanelShrinkRect, not the raw ui.ShopTipRect() (round-2 adversarial
	// review, owner item: the shop tip is too tall). The other three rooms
	// already route their own tip rect through this shrink
	// (townshell.go:169, :184; townscreen.go:455); the shop was the one
	// call site this story's own B2 table called unchanged, and
	// cmd/tippanelcheck measured its own spare at the shipped font as 66
	// rows on en, 78 on ru, covering table cell 2 at 46.2% (2960 of 6400px).
	v.TipPanel = t.tipView(roomShop, t.shopTip, ui.TipPanelShrinkRect(ui.ShopTipRect(), t.in.tipFont(), t.shopTip))

	shop := t.sess.Shop
	if shop != nil {
		v.InputEpoch = shop.inputEpoch
	}
	purse := int32(t.sess.Town.Gold())
	buy, sell := shop.BuyTotal(), shop.SellTotal()
	v.Purse, v.Buy, v.Sell, v.Total = purse, buy, sell, purse+sell-buy

	// Buy and Sell accept a press even on an empty table. The transaction
	// owns affordability checks and the empty-table no-op.
	table := shop.Table()
	v.Live = [4]bool{len(table) > 0, true, true, true}

	// The shelf grid. Nothing is shown until a rectangle in the room has been
	// clicked, which is the original's own no-shelf-open state.
	v.Chosen = t.shopChosen
	v.ShelfOffset = t.shelfBase
	t.packBase = min(t.packBase, t.packScrollMax())
	v.PackOffset = t.packBase
	v.PackBack, v.PackForward = t.packBase > 0, t.packBase < t.packScrollMax()
	if t.shopChosen >= 0 && t.shopChosen < len(shopRoomShelves) {
		room := shopRoomShelves[t.shopChosen]
		v.ShelfName = shopRoomShelfName(t.shopChosen, t.in.Table)
		if room.stocked {
			items := shop.Shelf(room.shelf)
			for i := 0; i < len(v.Shelf); i++ {
				k := t.shelfBase + i
				if k < 0 || k >= len(items) {
					continue
				}
				v.Shelf[i] = t.shopShelfCell(items[k], purse)
			}
		}
	}

	for i, place := range table {
		if i >= len(v.Table) {
			break
		}
		v.Table[i] = t.shopPlaceCell(place)
	}

	// THE BOTTOM STRIP IS THE SHOWN MEMBER'S OWN CONTAINER, not a party-wide
	// one (SHOP-PICKER-043), and its first element is the money element: an
	// element whose own +0x6 is 0xffff, whose quantity is re-read from the
	// campaign's gold and which draws the coin over backinv.bmp
	// (SHOP-SCREEN-036 arm a, SHOP-MONEY-048). It scrolls with the list, so it
	// leaves the strip when the strip is turned past it.
	stacks := t.shopPackStacks()
	for i := 0; i < len(v.Pack); i++ {
		k := t.packBase + i
		switch {
		case k == 0:
			v.Pack[i] = ui.ShopCell{Money: true, Count: uint32(purse)}
		case k >= 1 && k-1 < len(stacks):
			v.Pack[i] = t.shopPackCell(stacks[k-1])
			if t.hasCityShopTopology() && v.Pack[i].UseItemKey != "" {
				if member := t.shopPartyMember(t.shopMemberIndex()); member != nil {
					if root, err := cityMutationParty(t.sess.Town.cityObjects, member.ID); err == nil && k-1 < len(root.Pack) {
						v.Pack[i].UseItemKey = fmt.Sprintf("city:%d/%s", root.Pack[k-1], v.Pack[i].UseItemKey)
					}
				}
			}
		}
	}

	party := t.shopParty()
	v.Member, v.MemberCount = townPickerPosition(party, t.shopMemberIndex())
	if i := t.shopMemberIndex(); i < len(party) {
		v.Figure = t.shopFigure(i)
		if i < len(t.shopFigureMasks) {
			v.SlotMask = t.shopFigureMasks[i]
			// OrdinaryDollMask NEVER TAKES THE SUPPRESSED SUBSTITUTION BELOW (round-2
			// adversarial review, tenth pass, counterexample 1): shopReleaseIsOrigin
			// (pkg/ui) reads this to ask whether a release names the drag's own
			// origin slot, a question the suppressed mask can never answer true for
			// that slot once the drag has armed.
			v.OrdinaryDollMask = t.shopFigureMasks[i]
		}
		for n := 0; n < len(v.SlotInfo); n++ {
			if item, ok := t.shopEquippedItem(i, n+1); ok && !item.Empty() {
				v.SlotInfo[n] = itemInstanceInfoLines(item, t.in.Table, t.in.Words)
				v.SlotIcon[n] = t.shopItemIcon(item)
			}
		}
		// THE SUPPRESSED PICTURE STANDS IN FOR THE ORDINARY ONE (1005 round
		// 2, world.go's own dollSubject restated): only for the SAME member
		// the drag machine is lifting a slot off (shopSuppressMember == i,
		// dollSuppressOwner's own check restated), and only while
		// refreshShopDrag has actually composed one.
		//
		// shopStepMember DOES NOT call composeShopFaces (round-2 adversarial
		// review, item 3 — this comment previously claimed it does, which was
		// false, composeShopFaces runs once, on room entry and on every mutation
		// that changes what is worn, never on a bare picker step). The member
		// guard below is NOT REACHABLE THROUGH TODAY'S OWN INPUT WIRING — a
		// picker press and a held doll drag both need the same physical button, so
		// app.go's drive can never interleave them — but IS reachable at this
		// model's own API, one call at a time (ShopSuppressDoll then
		// shopStepMember with no composeShopFaces between), which is how
		// TestShopScreenNeverSubstitutesAnotherMembersSuppressedFigure
		// (shopdrag_test.go) exercises it directly and mutation-witnesses the
		// guard rather than merely asserting it is unreachable.
		if i == t.shopSuppressMember && t.shopSuppressSlot != 0 && t.shopSuppressFigure != nil {
			v.Figure = t.shopSuppressFigure
			v.SlotMask = t.shopSuppressMask
		}
	}
	v.Character = t.townCharacterView()
	// The drag preview changes only the picture. townCharacterView resolves
	// canonical statistics and equipment, so copying the suppressed figure
	// into that view makes the doll look unequipped while its numbers remain
	// unchanged until ShopDrag actually commits the move.
	v.Character.Figure = v.Figure
	t.fillTownBook(&v)
	return v
}

func (t *townScreen) fillTownBook(v *ui.ShopScreenView) {
	party := t.shopParty()
	v.Book = t.shopBook
	if v.Book && len(party) > 0 {
		member := party[t.shopMemberIndex()]
		d, _, _ := mapload.PartySpawnWithTable(member, t.in.Table)
		e := sim.Entity{KnownSpells: member.KnownSpells, Book: member.Book, Mind: d.Mind, Spirit: d.Spirit,
			Skill: d.Skill, SkillXP: d.SkillXP}
		names := spellNamesFrom(t.in.Table, t.in.Words.SpellBookNames)
		v.Spells, v.SpellCatalog = selectedSpellbook(mapload.TableRules(t.in.Table), []sim.Entity{e}, mapload.SpellRules(t.in.Table), names, t.in.Words, t.shopSpellIcon)
		if !v.SpellCatalog {
			v.Spells = spellbookOf(mapload.TableRules(t.in.Table), e, mapload.SpellRules(t.in.Table), names, t.in.Words, t.shopSpellIcon)
		}
	}
}

// shopEquippedCode is member i's own code in equipment slot n (1..12),
// shopWornSlots' own read restated as a value rather than a pointer, so
// ShopScreen's own SlotInfo loop above can build itemInfoLines' own
// characteristics for whatever the doll — the ordinary one, not the
// suppressed picture — currently wears.
//
// SLOT 1 GOES THROUGH shopSlot1Code (round-2 adversarial review,
// counterexample 1), not the raw array alone: composeShopFaces already draws
// slot 1 with member.Weapon's own fallback whenever the array itself reads
// empty, and this was the one reader that disagreed with the picture —
// SlotInfo and SlotIcon both answered nothing for a slot the figure and mask
// both named, so a shop doll drawn wearing a weapon through the fallback had
// no hover text and no drag icon over that slot.
func (t *townScreen) shopEquippedCode(i, n int) (data.ItemCode, bool) {
	item, ok := t.shopEquippedItem(i, n)
	return data.ItemCode(item.Code), ok
}

func (t *townScreen) shopEquippedItem(i, n int) (sim.ItemInstance, bool) {
	if n < 1 || n > sim.EquipSlots {
		return sim.ItemInstance{}, false
	}
	worn := t.shopWornItemSlots(i)
	if worn == nil {
		return sim.ItemInstance{}, false
	}
	item := worn[n-1].Clone()
	if n == 1 && item.Empty() {
		if code, ok := t.shopWeaponFallbackCode(i); ok {
			item = mapload.ItemInstanceFromCode(code, t.in.Table)
		}
	}
	return item, true
}

// ShopDrag completes a drag between two of the screen's own surfaces (spec
// bullet 1 and 2, restated for the shop; `DIV-087`, `DIV-089`, `DIV-091`). A
// pair this switch does not recognise — released back on the same surface,
// or on nothing the shop names at all — is a no-op: nothing was ever taken
// from either side to begin with (the model is never mid-drag, command.go's
// own invariant restated for a screen with no world underneath it).
//
// THE FIFTH, table-to-doll, has no click precedent: a click on a table cell
// takes it off, so this pair reaches `shopEquipFromTable`, the take-and-wear
// gesture over `Shop.TakeOffTable`, for a customer-owned place only.
//
// The three staging pairs pass from.Shift, the release frame's modifier, to
// the act ShopClick reaches with c.Shift (DIV-1463).
func (t *townScreen) ShopDrag(from, to ui.ShopControl) ui.TownAction {
	if t == nil || t.room != roomShop {
		return ui.TownAction{}
	}
	switch {
	case from.Kind == ui.ShopControlPackCell && to.Kind == ui.ShopControlDoll:
		return t.shopEquipFromPack(from.Index)
	case from.Kind == ui.ShopControlShelfCell && to.Kind == ui.ShopControlDoll:
		// Shelf stamp: the release returns it (SHOP-098, DIV-1667).
		return ui.TownAction{}
	case from.Kind == ui.ShopControlDoll && to.Kind == ui.ShopControlPackCell:
		return t.shopUnequipDoll(from.Index + 1)
	case from.Kind == ui.ShopControlDoll && (to.Kind == ui.ShopControlShelfCell || to.Kind == ui.ShopControlTableCell):
		return t.shopUnequipToTable(from.Index + 1)
	case from.Kind == ui.ShopControlPackCell && (to.Kind == ui.ShopControlTableCell || to.Kind == ui.ShopControlShelfCell):
		// PACK-TO-SHELF JOINS PACK-TO-TABLE (counterexample 5, round-2 adversarial
		// review, second pass, `ITEM-CMD-007`), on doll-to-shelf's own precedent
		// above: `ITEM-CMD-007`'s own opcode space pairs the container code (2)
		// with a shop code (4..8) in either direction, table and shelf alike, so a
		// shelf destination is no narrower a target than the table this build
		// already stages every pack-origin move on. This build has no shelf-only
		// stock mutation to reach beyond staging — a shelf's own contents are
		// server-authored inventory, not a place the model writes player goods
		// into — so the staging function already proven for the table is the
		// SAME act a shelf drop names, not a second one invented for destination
		// alone.
		return t.shopFromPack(t.packBase+from.Index-1, from.Shift)
	case from.Kind == ui.ShopControlShelfCell && (to.Kind == ui.ShopControlTableCell || to.Kind == ui.ShopControlPackCell):
		// SHELF-TO-PACK JOINS SHELF-TO-TABLE, the same reasoning mirrored:
		// `ITEM-CMD-007`'s shop code (4..8) pairs with the container code (2)
		// symmetrically, and shopClickShelfCell is already this build's own "take
		// a shelf item toward the player's side" act — staging on the table,
		// where a price is asked before anything leaves the pack for good
		// (shopBuy's own commit). A pack destination reaches the SAME act rather
		// than an unpriced direct pack insert, which would let a shelf item enter
		// the pack with no purse ever consulted.
		return t.shopClickShelfCell(from.Index, from.Shift)
	case from.Kind == ui.ShopControlTableCell && (to.Kind == ui.ShopControlPackCell || to.Kind == ui.ShopControlShelfCell):
		return t.shopOffTable(from.Index, from.Shift)
	case from.Kind == ui.ShopControlTableCell && to.Kind == ui.ShopControlDoll:
		// A merchant-owned place keeps its shelf stamp and stays (DIV-1668).
		if table := t.sess.Shop.Table(); from.Index < 0 || from.Index >= len(table) || !table[from.Index].Mine {
			return ui.TownAction{}
		}
		// COUNTEREXAMPLE 3 (round-2 adversarial review): the one pair among the
		// five-place grid's own eight this switch left unrecognised.
		// shopEquipFromTable is the table's own take-and-wear gesture,
		// shopEquipFromPack's and shopEquipFromShelf's own pattern restated over
		// TakeOffTable.
		return t.shopEquipFromTable(from.Index)
	}
	return ui.TownAction{}
}

// ShopSuppressDoll is the shop's own per-frame push (1005 round 2,
// world.go's own refreshDollDrag): slot is 1-based, 0 for none, and a call
// naming the same slot already in force costs one compare and no recompose,
// refreshDollDrag's own guard restated.
func (t *townScreen) ShopSuppressDoll(slot int) {
	if t == nil || t.room != roomShop {
		return
	}
	t.refreshShopDrag(slot)
}

// refreshShopDrag is ShopSuppressDoll's own act: it recomposes the shown
// member's figure with slot's code cleared, caches it, and does nothing at
// all where the slot named has not changed since the last call.
//
// THE EQUIPMENT IT COMPOSES FROM MUST MATCH composeShopFaces' OWN BUILD
// (round-2 adversarial review, item 4): the `member.Weapon` slot-1 fallback
// below is composeShopFaces' own line, restated. Its absence was a
// divergence between the ordinary figure (built with the fallback) and the
// suppressed one (built without it) for any member whose worn set leaves
// slot 1 empty and who carries a weapon anyway — `rosterTemplate`
// (pkg/mapload/spawn.go) builds exactly such a companion. A drag on any
// OTHER slot of that member would have suppressed a figure missing the
// weapon layer the ordinary figure still shows, `DIV-085`'s own disagreement
// between a hit test and a drawn pixel.
func (t *townScreen) refreshShopDrag(want int) {
	if want == t.shopSuppressSlot {
		return
	}
	t.shopSuppressSlot = want
	if want == 0 {
		t.shopSuppressFigure, t.shopSuppressMask = nil, nil
		return
	}
	i := t.shopMemberIndex()
	party := t.shopParty()
	if i < 0 || i >= len(party) {
		return
	}
	t.shopSuppressMember = i
	// eq IS mapload.EquipmentFromParty, NOT THE TWO-LINE Carry-over-Worn IDIOM
	// THIS FILE USED TO STATE INLINE (round-2 adversarial review, twelfth pass,
	// C1 addendum): the mission doll restated the same rule a fourth time and
	// forgot the Carry half, so the two remaining inline copies are collapsed
	// onto the one named reader they were always supposed to agree with, rather
	// than left as a second place the rule could still be half-copied from.
	// party[i] is read by VALUE here (EquipmentFromParty takes a PartyMember,
	// not a pointer), which changes nothing: this call only extracts eq and
	// never writes back through party[i] the way shopWornSlots' own pointer
	// result does.
	eq := mapload.EquipmentFromParty(party[i])
	// SLOT 1'S FALLBACK GOES THROUGH shopWeaponFallbackCode, composeShopFaces'
	// own line restated (round-2 adversarial review, third pass, counterexample
	// 2's own duplication finding applied here too): the unconditional
	// member.Weapon != nil test this replaced disagreed with shopSlot1Code's
	// own discriminator (shoproom.go) for a member who had already unequipped
	// his starting weapon for real and finished the mission — the code sits
	// in his own pack, and drawing it again on the suppressed doll would have
	// shown a weapon shopUnequipDoll and its siblings no longer treat as
	// fallback-backed.
	if occupied, _ := eq.Occupied(1); !occupied {
		if code, ok := t.shopWeaponFallbackCode(i); ok {
			eq.SetCode(1, data.ItemCode(code))
		}
	}
	eq.SetCode(want, 0)
	var src entrySource
	if t.in.Archives != nil {
		src = t.in.Archives.Containers
	}
	composed, _ := composeInventoryPortraitLayered(src, uint32(i+1), eq,
		figureLayersFor(mapload.MemberLayers(party[i], t.in.Table), t.in.Table), memberFigureID(party[i]))
	t.shopSuppressFigure, t.shopSuppressMask = composed.Figure, composed.SlotMask
}

// shopMemberIndex is which member the character panel shows and whose container
// the bottom strip is bound to. It is clamped to the roster on every read, so a
// party that shrank between two frames cannot leave it pointing past the end,
// and it never rests on a tavern hire while the party holds anyone else
// (townPickerIndex).
func (t *townScreen) shopMemberIndex() int {
	return townPickerIndex(t.shopParty(), t.shopMember)
}

// clampTownMember keeps the one town-wide character selection valid after a
// roster mutation. The historical field name predates the shared shell; the
// value now deliberately survives movement between every town room.
func (t *townScreen) clampTownMember() {
	if t == nil {
		return
	}
	t.shopMember = t.shopMemberIndex()
}

// shopStepMember moves the picker by one, wrapping at both ends
// (SHOP-PICKER-043: next resets to 0 at the roster's upper bound, previous takes
// the last index when already at 0). The roster is the members a picker stands
// on, so a tavern hire in the party is stepped over (townPickerMembers).
//
// ONE PRESS MOVES THE FIGURE AND THE STRIP TOGETHER, which is the original's own
// single routine: it binds the backpack grid to the new member's own container
// and marks the figure for recomposition in the same eight steps. Here that is
// the index, the strip's own base returned to the first place, and the composed
// figure taken from the cache the room filled on the way in.
func (t *townScreen) shopStepMember(step int) ui.TownAction {
	party := t.shopParty()
	roster := townPickerMembers(party)
	n := len(roster)
	if n <= 1 {
		return ui.TownAction{}
	}
	previous := t.shopMemberIndex()
	at := slices.Index(roster, previous)
	if at < 0 {
		at = 0
	}
	to := (at + step%n + n) % n
	i := roster[to]
	t.shopMember, t.packBase = i, 0
	t.clearSchoolSelection()
	t.retargetSchoolColumn(party[previous].Mage, party[i].Mage)
	name := party[i].Name
	if name == "" {
		name = fmt.Sprintf("member %d", to+1)
	}
	return ui.TownInfo(fmt.Sprintf("showing %s", name))
}

// shopFigure is member i composed in his own equipment, 160x240, from the cache
// the room filled when it was entered (composeShopFaces).
//
// THE CACHE IS THE SEAM SHOP-LIMIT-049 NAMES. The original shows the SAME panel
// object in the mission screen and in the shop, and what that object carries
// across the boundary is its composition cache. This build cannot re-parent one
// widget into two screens, so what it shares instead is the composition: one
// pass over the party fills the map, and the town dialogue's own speaker figures
// and this panel both read it.
func (t *townScreen) shopFigure(i int) *image.RGBA {
	if i < 0 || i >= len(t.shopFigures) {
		return nil
	}
	return t.shopFigures[i]
}

// shopUsable is the wear rule asked of the shown party member: whether the
// class he is may use the item code names.
//
// THE SHOWN MEMBER IS THE ONE THE PICKER STANDS ON. The original passes
// view+0x10c[view+0x130], the roster entry the picker steps and the backpack
// grid is bound to (SHOP-SCREEN-036, SHOP-PICKER-043); here that is the same
// index the character panel draws and the bottom strip reads.
//
// AN ITEM WHOSE ROW THIS BUILD CANNOT READ IS USABLE, on spec FR-2a's polarity:
// the column was not read, so the rule does not apply, and the cell keeps the
// background it had before this rule existed. A missing table answers the same
// way, for the same reason.
func (t *townScreen) shopUsable(code data.ItemCode) bool {
	table := t.in.Table
	if table == nil {
		return true
	}
	party := t.shopParty()
	if len(party) == 0 {
		return true
	}
	allowed, _ := data.AllowsItem(code, party[t.shopMemberIndex()].Mage,
		table.Weapons, table.Shields, table.Armors)
	return allowed
}

// shopCellBack is the background arm one occupied cell takes, given whether the
// shown member can use it and whether it is a shelf item the purse covers.
//
// THE AFFORDABLE ARM IS NESTED INSIDE THE USABLE ONE, which is the
// original's own order: its painter reaches backinvs.bmp only when the price
// test AND the usability test both pass, reaches backinv.bmp when usability
// alone passes, and reaches backinvg.bmp otherwise. So an unusable shelf
// item is never drawn affordable, however large the purse, by construction
// rather than by a second test.
func shopCellBack(usable, affordable bool) ui.ShopCellBack {
	switch {
	case !usable:
		return ui.ShopBackUnusable
	case affordable:
		return ui.ShopBackAffordable
	default:
		return ui.ShopBackItem
	}
}

// shopShelfCell is one of the merchant's items as a cell. Its background states
// whether the shown member can use it and whether the purse covers one unit of
// it (SHOP-SCREEN-036 arms c, d and e) and its plaque is the merchant's family
// at the full price (SHOP-SCREEN-037).
func (t *townScreen) shopShelfCell(item ShopItem, purse int32) ui.ShopCell {
	// THE PURSE TEST IS AGAINST ONE UNIT AND NOT AGAINST THE STACK. A cell
	// can hold several units since `DIV-322`, and its background answers
	// whether the member can use it and whether the purse covers a unit of
	// it, which is what a default click buys.
	back := shopCellBack(t.shopUsable(item.Code), item.Price > 0 && purse >= item.Price)
	instance := item.Instance()
	return ui.ShopCell{
		Icon:        t.shopItemIcon(instance),
		Count:       uint32(item.Count),
		Price:       shopGridPrice(item.Code, item.Price, false),
		PlaquePrice: shopClientPrice(item.Code, item.Price),
		Back:        back,
		Star:        itemShowsStarTrail(instance),
		Info:        itemInstanceInfoLines(instance, t.in.Table, t.in.Words),
	}
}

// shopClientPrice is the unit price the client's copy of an item carries, the
// number the grid chooses the item's plaque from on both sides of the deal
// (SHOP-SCREEN-037, ITEM-PRICETAG-144). The copy of a plain Item, which every
// class 14 item is, carries 0 where the stored price is the -1 sentinel
// (SHOP-MISSION-019); every other kind carries its price unchanged.
func shopClientPrice(code data.ItemCode, price int32) int32 {
	if price == -1 && code.B() == data.ItemClassCarried {
		return 0
	}
	return price
}

// shopGridPrice is the number a grid cell prints for an item whose stored unit
// price is price, on the merchant's side or the player's (SHOP-SCREEN-037,
// ITEM-PRICETAG-144): the client's price, which the player's side halves to
// (price+1)/2 with the original's truncating division, so a price of 0 or -1
// prints 0 and -1250 prints -624. Nothing is clamped: a price of 0 or less is
// drawn like any other. The plaque under the number is chosen from
// shopClientPrice, not from this figure.
func shopGridPrice(code data.ItemCode, price int32, mine bool) int32 {
	price = shopClientPrice(code, price)
	if mine {
		return int32((int64(price) + 1) / 2)
	}
	return price
}

// shopPlaceCell is one table place. The side its stamp names selects the plaque
// family and the figure: the player's side prints ceil(price/2), which is what
// the merchant pays a unit, on the plaque of the whole price (SHOP-SCREEN-037,
// SHOP-TRAY-027).
func (t *townScreen) shopPlaceCell(place ShopPlace) ui.ShopCell {
	instance := place.Instance()
	useKey := ""
	if place.Mine && t.shopDollTakes(instance) {
		useKey = fmt.Sprintf("city:%d/from:%d/%#v", place.cityID, place.From, instance)
	}
	return ui.ShopCell{
		UseItemKey:  useKey,
		Icon:        t.shopItemIcon(instance),
		Count:       uint32(place.Count),
		Price:       shopGridPrice(place.Code, place.Price, place.Mine),
		PlaquePrice: shopClientPrice(place.Code, place.Price),
		Back:        shopCellBack(t.shopUsable(place.Code), false),
		Mine:        place.Mine,
		Star:        itemShowsStarTrail(instance),
		Info:        itemInstanceInfoLines(instance, t.in.Table, t.in.Words),
	}
}

// shopPackCell is one stack of the player's own pack, always on his side.
func (t *townScreen) shopPackCell(st sim.ItemStack) ui.ShopCell {
	code := data.ItemCode(st.Code)
	instance := st.Instance()
	useKey := ""
	if t.shopDollTakes(instance) {
		useKey = fmt.Sprintf("%#v", instance)
	}
	stored := t.shopItemValue(instance)
	return ui.ShopCell{
		UseItemKey:  useKey,
		Icon:        t.shopItemIcon(instance),
		Count:       st.Count,
		Price:       shopGridPrice(code, stored, true),
		PlaquePrice: shopClientPrice(code, stored),
		Back:        shopCellBack(t.shopUsable(code), false),
		Mine:        true,
		Star:        itemShowsStarTrail(instance),
		Info:        itemInstanceInfoLines(instance, t.in.Table, t.in.Words),
	}
}

// shopDollTakes reports whether a double click on this pack item goes to the
// doll's drop route, as ITEM-USE-112 sends the shop backpack's double click:
// a potion or scroll is used there (DIV-584), and an item with an equipment
// slot is worn there or refused by the doll's wear rule (ITEM-WEAR-057,
// DIV-1465). Any other item keeps the immediate single click.
func (t *townScreen) shopDollTakes(item sim.ItemInstance) bool {
	if item.Kind == 3 || item.Kind == 4 {
		return true
	}
	_, ok := EquipTarget(data.ItemCode(item.Code), t.in.Table)
	return ok
}

// shopItemIcon returns only the code-keyed base icon. The measured shelf,
// trade-tray and shown-member pack painters add the trail from ShopCell.Star;
// worn dolls and the held drag cursor reuse this base and remain unmarked.
func (t *townScreen) shopItemIcon(item sim.ItemInstance) *image.RGBA {
	return t.shopIcon(data.ItemCode(item.Code))
}

// shopIcon is one item's own picture, through the inventory's own loader and
// the front end's own cache, so an item shows the same picture in the shop as
// it does in the pack.
func (t *townScreen) shopIcon(code data.ItemCode) *image.RGBA {
	cache := t.shopIcons()
	if icon, tried := cache[uint16(code)]; tried {
		return icon
	}
	var src entrySource
	if t.in.Archives != nil {
		src = t.in.Archives.Containers
	}
	icons, _, _ := buildInventoryPack(src, []sim.ItemStack{{Code: uint16(code), Count: 1}}, cache)
	if len(icons) == 0 {
		return nil
	}
	return icons[0]
}

// shopIcons is the icon cache the screen resolves through, allocated on first
// use. It is the front end's, not the frame's: a screen that rebuilt it every
// frame would re-read every picture sixty times a second.
func (t *townScreen) shopIcons() map[uint16]*image.RGBA {
	if t.shopIconCache == nil {
		t.shopIconCache = map[uint16]*image.RGBA{}
	}
	return t.shopIconCache
}

// ShopClick acts on one control.
//
// A CONTROL THE MODEL REFUSES ANSWERS A REASON AND MOVES NOTHING. An index
// outside its own family is a miss, not a panic.
func (t *townScreen) ShopClick(c ui.ShopControl) ui.TownAction {
	if t == nil || t.room != roomShop {
		return ui.TownAction{}
	}
	switch c.Kind {
	case ui.ShopControlButton:
		return t.shopButton(c.Index)
	case ui.ShopControlArrowUp:
		t.scrollShelf(-1)
	case ui.ShopControlArrowDown:
		t.scrollShelf(+1)
	case ui.ShopControlPackLeft:
		t.scrollPack(-1)
	case ui.ShopControlPackRight:
		t.scrollPack(+1)
	case ui.ShopControlShelfCell:
		return t.shopClickShelfCell(c.Index, c.Shift)
	case ui.ShopControlTableCell:
		if t.shopBook {
			return ui.TownAction{}
		}
		return t.shopOffTable(c.Index, c.Shift)
	case ui.ShopControlPackCell:
		// The strip's first element is the money element, so a cell's own
		// index is one past it in the stack list.
		return t.shopFromPack(t.packBase+c.Index-1, c.Shift)
	case ui.ShopControlShelfPick:
		return t.chooseRoomShelf(c.Index)
	case ui.ShopControlPickerPrev:
		return t.shopStepMember(-1)
	case ui.ShopControlPickerNext:
		return t.shopStepMember(+1)
	case ui.ShopControlCharacterMode:
		t.townStats = !t.townStats
	case ui.ShopControlBook:
		t.shopBook = !t.shopBook
	case ui.ShopControlSpell:
		// The shop's borrowed book is inspection-only. Its cells swallow the
		// click so the trade table hidden under them cannot receive it.
		return ui.TownAction{}
	case ui.ShopControlDoll:
		// A PRESS THAT NEVER CROSSED TapSlop is round 1's own tap-to-unequip
		// (1005 round 2, spec bullet 2), reached through the one mutation
		// door ShopClick already is rather than a second one for the doll.
		return t.shopUnequipDoll(c.Index + 1)
	case ui.ShopControlMerchant:
		// The painted merchant takes no press (TOWN-478); flow.clickShop does
		// not send one. The shop's offer opens on entry only.
		return ui.TownAction{}
	}
	return ui.TownAction{}
}

// ShopScroll turns the region the pointer stands in (owner, DIV-005). The
// rack pages by whole rows, which is the arrows' own step; the strip moves
// by one place.
func (t *townScreen) ShopScroll(region ui.ShopWheelRegion, rows int) {
	if t == nil || t.room != roomShop {
		return
	}
	switch region {
	case ui.ShopWheelShelf:
		t.scrollShelf(rows)
	case ui.ShopWheelPack:
		t.scrollPack(rows)
	}
}

// shopButton runs one of the four commands (SHOP-SCREEN-035).
func (t *townScreen) shopButton(i int) ui.TownAction {
	switch i {
	case 0:
		return t.shopClear()
	case 1:
		if total := t.sess.Shop.BuyTotal(); total != 0 {
			if int32(t.sess.Town.Gold()) >= total {
				t.requestShopMerchantYes()
			} else {
				t.requestShopMerchantNo()
			}
		}
		return t.shopBuy()
	case 2:
		if t.sess.Shop.SellTotal() != 0 {
			t.requestShopMerchantYes()
		}
		return t.shopSell()
	case 3:
		// Clear the table, then leave. Back already clears it on the way out, so
		// leaving is the whole command. It posts no line: the original's leave
		// sends its own message and no text (SHOP-SCREEN-035).
		t.Back()
	}
	return ui.TownAction{}
}

// shopClickShelfCell puts the item in that cell on the table. whole is the
// shift+click convention, carried to shopFromShelf: a cell can hold several
// units since `DIV-322`.
func (t *townScreen) shopClickShelfCell(i int, whole bool) ui.TownAction {
	if t.shopChosen < 0 || t.shopChosen >= len(shopRoomShelves) {
		return ui.TownAction{}
	}
	room := shopRoomShelves[t.shopChosen]
	if !room.stocked {
		return ui.TownAction{}
	}
	t.shopShelf = room.shelf
	return t.shopFromShelf(t.shelfBase+i, whole)
}

// chooseRoomShelf makes one of the four room rectangles the shown shelf and
// returns the grid to its first row (SHOP-SCREEN-034).
func (t *townScreen) chooseRoomShelf(i int) ui.TownAction {
	if i < 0 || i >= len(shopRoomShelves) {
		return ui.TownAction{}
	}
	t.selectShopInteriorRack(i, true)
	t.shopChosen, t.shelfBase = i, 0
	room := shopRoomShelves[i]
	t.shopShelf = room.shelf
	if !room.stocked {
		return ui.TownAction{}
	}
	return ui.TownAction{}
}

// scrollShelf moves the shelf grid by whole rows (SHOP-SCREEN-033's doubled
// step over a grid two columns wide).
//
// THE BASE NEVER LEAVES THE SHELF. Scrolling down past the last row stops at
// the last row that still holds an item, so an arrow press can never leave a
// stocked shelf showing nothing.
func (t *townScreen) scrollShelf(rows int) {
	if t.shopChosen < 0 || t.shopChosen >= len(shopRoomShelves) {
		return
	}
	room := shopRoomShelves[t.shopChosen]
	n := 0
	if room.stocked {
		n = len(t.sess.Shop.Shelf(room.shelf))
	}
	t.shelfBase = clampGridBase(t.shelfBase+rows*shopShelfCols, n, shopShelfCols)
}

// scrollPack moves the backpack grid by one place (spec D-9). The original's own
// control for this is not decoded; the wheel is this build's.
//
// THE LIST IS ONE LONGER THAN THE STACK COUNT because its first element is the
// money element (SHOP-SCREEN-036 arm a), which scrolls with everything else.
func (t *townScreen) scrollPack(places int) {
	t.packBase = min(max(t.packBase+places, 0), t.packScrollMax())
}

// packScrollMax is the furthest the strip turns: the last element is in the
// last slot and nothing scrolls into empty slots. The list is the money element
// and the stacks, so five or fewer never scroll.
func (t *townScreen) packScrollMax() int {
	return max(0, len(t.shopPackStacks())+1-len(ui.ShopScreenView{}.Pack))
}

// clampGridBase keeps a grid's first index inside a list of n items, on a
// multiple of step. A list shorter than one step is always shown from 0.
func clampGridBase(base, n, step int) int {
	if base < 0 || n <= 0 {
		return 0
	}
	last := ((n - 1) / step) * step
	if base > last {
		return last
	}
	return base
}
