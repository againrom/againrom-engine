package ui

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"againrom/pkg/render/text"
)

// The shop screen (second round).
//
// THIS FILE HOLDS THE GEOMETRY AND NOTHING ELSE ABOUT THE TRADE. Every
// rectangle below is decoded — SHOP-SCREEN-030 for the five regions,
// SHOP-SCREEN-031 for the three cell formulas, SHOP-SCREEN-033 for the two
// arrows, SHOP-SCREEN-034 for the four shelf rectangles in the room picture
// and SHOP-SCREEN-035 for the four buttons. What an item costs, whether the
// purse covers it and which shelf it came off are all decided on the far
// side of the seam and arrive here as a picture, a number and a small
// enumeration, exactly as ChargenPresentation already crosses it.
//
// THE SCREEN IS COMPOSED INTO AN *image.RGBA AND UPLOADED ONCE. Every control
// is resolvable and every pixel is composable with no window and no archive, so
// the whole of this file is reachable from a test on a machine with no game
// installed (golden rule 2).

// The frame the whole screen occupies. Six regions tile it: the five the shop
// view builds and the character panel it borrows (SHOP-FIGURE-041).
const (
	shopScreenW = 640
	shopScreenH = 480
)

// The five regions the shop view builds, left, top, right, bottom
// (SHOP-SCREEN-030).
var (
	shopShelfRegion    = image.Rect(0, 0, 164, 303)
	shopMerchantRegion = image.Rect(164, 0, 480, 303)
	shopTableRegion    = image.Rect(0, 303, 480, 390)
	shopPackRegion     = image.Rect(0, 390, 480, 480)
	shopButtonRegion   = TownWideUpperRegion

	// shopCharRegion is the sixth region: the mission screen's own character
	// panel, offset by 640 - width and re-parented to the shop view for the
	// visit (SHOP-FIGURE-041). Its rect is (0,238,160,480) at the main frame
	// plus the literal 0x280.
	shopCharRegion = TownCharacterRegion

	// The composed figure and SlotMask retain their 160x240 source coordinate
	// system. The shared shell paints all of townCharacterFigure's 240
	// rows; shopDollSlotAt still subtracts this source origin before asking the
	// mask, then shopDollAreaAt restricts the answer to what was painted.
	shopFigureRect = image.Rect(480, 240, 640, 480)

	// The party picker's two 32x32 rects, panel-relative (1,205,33,237) and
	// (119,205,151,237), which at the shipped 640x480 are these
	// (SHOP-PICKER-043). The left one posts the previous member, the right one
	// the next; both wrap.
	shopPickerPrevRect    = townCharacterPrev
	shopPickerNextRect    = townCharacterNext
	shopCharacterModeRect = townCharacterMode
)

// The cell grid. Every cell is 80x80 and the three grids differ only in their
// origins and their shapes (SHOP-SCREEN-031).
const (
	shopCellSize  = 80
	shopShelfCols = 2
	shopShelfRows = 3
	shopShelfN    = shopShelfCols * shopShelfRows
	shopStripN    = 5
)

// The price figure on a plaque (ITEM-PRICETAG-144, MISSION-MSGLINE-056):
// shopPriceRightInset is how far left of the cell's right edge the figure's pen
// ends, shopPriceTop how far below the cell's top its glyph cells start, and
// shopPriceShadow how far right and down the flat shadow behind it stands.
const (
	shopPriceRightInset = 6
	shopPriceTop        = 1
	shopPriceShadow     = 1
)

// shopPriceInk is the colour whose text.Shade ramp is the grid's price ramp:
// level k is (185k/15, 159k/15, 73k/15). The shadow is the text renderer's own
// flat colour, messageShadowColor.
var shopPriceInk = color.RGBA{R: 185, G: 159, B: 73, A: 255}

// The two scroll arrows on the shelf grid (SHOP-SCREEN-033).
var (
	shopArrowUpRect   = image.Rect(46, 0, 118, 32)
	shopArrowDownRect = image.Rect(46, 271, 118, 303)
	shopPackLeftRect  = image.Rect(0, 391, 32, 479)
	shopPackRightRect = image.Rect(432, 391, 464, 479)
)

// The merchant panel carries TWO four-rect arrays and they are not the same
// (SHOP-SHELF-047, which corrects SHOP-SCREEN-034): one is where a shelf is
// clicked, the other is where its animation is drawn. Both are indexed by the
// same i, and the shipped animation folder for i is `4 - i`.
//
// UNTIL THIS ROUND THIS BUILD USED THE DRAW RECTS AS HIT RECTS, listed in the
// reverse of the index order, which is exactly the consumer error the
// retraction of SHOP-SCREEN-034 names: four click regions on the shelf art
// rather than on the larger room areas, opening the wrong shelf for three of
// the four clicks that did register.
var shopShelfPickRects = [4]image.Rectangle{
	image.Rect(354, 110, 459, 295),
	image.Rect(169, 110, 274, 295),
	image.Rect(314, 5, 454, 105),
	image.Rect(172, 5, 314, 105),
}

// shopShelfDrawRects is where shelf i's own animation is drawn, index for index
// with the hit rects above (SHOP-SHELF-047). The town description draws each
// rack here at its own natural size; the chosen shelf is also marked here.
var shopShelfDrawRects = [4]image.Rectangle{
	image.Rect(353, 108, 433, 220),
	image.Rect(197, 108, 277, 220),
	image.Rect(313, 20, 445, 108),
	image.Rect(201, 20, 313, 108),
}

// shopMerchantRect is where the merchant himself stands: his sprite is drawn at
// (panel.left + 113, panel.top + 112) and measures 76x176 (SHOP-MERCHANT-046),
// so he occupies (277,112)-(353,288). The four shelf hit rects leave that column
// free — they stop at x 274 and resume at x 354 — so he shares no pixel with a
// pressable rectangle. A press on him does nothing (TOWN-478); the rectangle
// serves his hover tooltip.
var shopMerchantRect = image.Rect(277, 112, 353, 288)

// The four command buttons, top to bottom (SHOP-SCREEN-035). Each carries one
// number and one command: purse, buy total, sell total, and the projected purse
// after those trades (owner-directed, DIV-170).
var shopButtonRects = [4]image.Rectangle{
	image.Rect(494, 15, 614, 67),
	image.Rect(483, 67, 623, 113),
	image.Rect(483, 114, 623, 160),
	image.Rect(494, 160, 614, 212),
}

// ShopButtonRect answers one command button's screen rectangle. Exported for
// cmd/buttonframecheck (1017), which correlates each shipped button bitmap
// against shopmenu.bmp and compares the winner with this rectangle, offset by
// shopButtonRegion.Min (TownWideUpperRegion.Min, already exported).
func ShopButtonRect(i int) image.Rectangle {
	if i < 0 || i >= len(shopButtonRects) {
		return image.Rectangle{}
	}
	return shopButtonRects[i]
}

// Where the merchant's room picture is drawn inside its region, and where the
// room frame is drawn (SHOP-SCREEN-032: ShopMain.bmp at (169, 8), ShopFrame.256
// at the panel's own origin).
var (
	shopRoomOrigin  = image.Pt(169, 8)
	shopFrameOrigin = image.Pt(164, 0)
)

// ShopCellAt is cell i of one grid, in frame coordinates (SHOP-SCREEN-031).
func ShopShelfCellRect(i int) image.Rectangle {
	col, row := i%shopShelfCols, i/shopShelfCols
	x, y := 1+shopCellSize*col, 31+shopCellSize*row
	return image.Rect(x, y, x+shopCellSize, y+shopCellSize)
}

func ShopTableCellRect(i int) image.Rectangle {
	x := 32 + shopCellSize*i
	return image.Rect(x, 303, x+shopCellSize, 303+shopCellSize)
}

func ShopPackCellRect(i int) image.Rectangle {
	x := 32 + shopCellSize*i
	return image.Rect(x, 395, x+shopCellSize, 395+shopCellSize)
}

// ShopCellBack is which cell background a cell takes (SHOP-SCREEN-036, whose
// per-hero usability arm 0162 added).
type ShopCellBack uint8

const (
	// ShopBackEmpty is backinvg.bmp: no item in the cell.
	ShopBackEmpty ShopCellBack = iota
	// ShopBackItem is backinv.bmp: an item the cell can hold.
	ShopBackItem
	// ShopBackAffordable is backinvs.bmp: a shelf item the purse covers.
	ShopBackAffordable

	// ShopBackUnusable is backinvg.bmp again: an item the shown party member's
	// class cannot use. The ORIGINAL DRAWS THE SAME PICTURE HERE AS FOR AN
	// EMPTY CELL, which is why the state is separate rather than folded onto
	// ShopBackEmpty: Occupied is derived from this field, and every click,
	// price plaque and count on the screen is gated on it, so an unusable item
	// folded onto the empty state would have stopped being buyable, sellable
	// and clickable as well as grey.
	ShopBackUnusable
)

// ShopCell is one 80x80 place of one grid, as this package draws it.
//
// Price IS ALREADY THE FIGURE TO PRINT. The halving on the player's side is the
// far side's arithmetic (SHOP-SCREEN-037), not this package's. THE PLAQUE IS
// NOT CHOSEN FROM IT but from PlaquePrice, the stored price, of which the
// player's side prints half.
type ShopCell struct {
	UseItemKey string // set only when a double click uses or wears it
	Icon       *image.RGBA
	Count      uint32
	Price      int32

	// PlaquePrice is the unit price the item carries, the number whose decimal
	// digit count selects the plaque on both sides of the deal
	// (SHOP-SCREEN-037). Price is this number on the merchant's side and
	// (PlaquePrice+1)/2 on the player's.
	PlaquePrice int32

	Back ShopCellBack

	// Star is ITEM-STARFLAG-096's display bit after the item-owning tier has
	// applied the Potion exception. StarPhase belongs to this visible grid
	// slot for the current paint, not to the item. Held cursors and SlotIcon
	// intentionally carry neither field (ITEM-STARSURF-097).
	Star      bool
	StarPhase uint32

	// Money marks the money element: the cell whose own +0x6 is 0xffff in the
	// original, whose quantity is re-read from the campaign's gold, and which
	// draws backinv.bmp with the coin sprite over it (SHOP-SCREEN-036 arm a,
	// SHOP-MONEY-048). It carries no icon and no price; Count is the purse.
	Money bool

	// Mine selects which of the two plaque families the price is drawn on:
	// costm on the player's side, costs on the merchant's.
	Mine bool

	// Info is the characteristics hover, and it is this build's own addition to
	// a screen the original draws no item text on.
	Info []string
}

// Occupied reports whether the cell holds an item. An empty cell draws NOTHING
// AT ALL — its grid's own picture shows through — and no click on it moves
// anything.
//
// AN UNUSABLE CELL IS OCCUPIED. It is drawn on the empty cell's own picture
// but it holds an item, and the shop sells, buys and moves it exactly as
// before: the class rule reaches the background and nothing else, on the
// original's own reading, where the purchase path never consults it.
func (c ShopCell) Occupied() bool { return c.Back != ShopBackEmpty }

// ShopScreenArt is every picture the screen paints with, resolved from one
// install and handed over whole.
//
// EVERY FIELD MAY BE NIL. A picture that would not read leaves its own region
// unpainted and changes nothing else, which is what makes the screen composable
// with no install at all.
type ShopScreenArt struct {
	Book      *BottomHUDArt
	Shelf     *image.RGBA // interface/shopinv.bmp, 164x303
	Table     *image.RGBA // interface/shoptable.bmp, 472x87
	Room      *image.RGBA // interface/shopanim/shopmain.bmp, 288x288
	Frame     *image.RGBA // interface/shopframe.256, 316x303
	Menu      *image.RGBA // interface/shopmenu.bmp, 176x238
	Arrow     [4]*image.RGBA
	PackArrow [4]*image.RGBA
	// PackCap is the strip's resting left and right corner: the frame's own
	// edge columns, which an arrow overlays while the strip can turn that way.
	PackCap [2]*image.RGBA
	Button  [4]*image.RGBA

	// Back is indexed by ShopCellBack: backinvg, backinv, backinvs, and
	// backinvg again for the unusable state.
	Back [4]*image.RGBA

	// Plaque is the two families of seven, [0] costs (the merchant's side),
	// [1] costm (the player's), one per decimal digit count 1..7.
	Plaque [2][7]*image.RGBA

	// Coin is graphics\interface\money\money.16a frame 0, one 80x80 frame —
	// the same size as a cell and as backinv.bmp (SHOP-MONEY-048). It carries
	// a per-pixel level and composites over the background already in the
	// cell.
	Coin *image.RGBA

	// Scene is the shop interior's racks and merchant by the town
	// description's entry names. The game's room page draws them.
	Scene map[string][]image.Image
}

// ShopScreenView is the whole screen's state for one frame.
type ShopScreenView struct {
	SuppressHover bool  // App paints the shared delayed overlay after widening.
	InputEpoch    *byte // transient stock identity, never written to a save
	Shelf         [shopShelfN]ShopCell
	Table         [shopStripN]ShopCell
	Pack          [shopStripN]ShopCell
	// Scene is the shop's composed interior; nil draws none.
	Scene RoomScene

	// Purse, Buy, Sell and Total are the four numbers printed in button order.
	// Total is Purse + Sell - Buy, including a negative projected balance.
	Purse, Buy, Sell, Total int32

	// Live is whether each button's command can run. A button that is not live
	// is drawn unlit and its click moves nothing.
	Live  [4]bool
	Press ShopControl

	// Shelf selection: which of the four room rectangles is chosen, -1 for
	// none, and what the chosen one is called.
	Chosen    int
	ShelfName string
	// ShelfOffset and PackOffset let the paint-owned star phase arrays apply
	// the decoded asymmetric scroll without learning item identity.
	ShelfOffset int
	PackOffset  int
	// PackBack and PackForward say whether the strip can turn that way; a
	// corner that cannot shows its resting cap instead of the arrow.
	PackBack, PackForward bool

	// The character panel (SHOP-FIGURE-041 … SHOP-PICKER-043): the shown
	// member composed in his own equipment as a 160x240 canvas, his index in
	// the roster and the roster's size. The panel shows the same
	// member whose container the bottom strip is bound to; one press of the
	// picker moves both.
	Figure       *image.RGBA
	Member       int
	MemberCount  int
	Character    TownCharacterView
	Book         bool
	Spells       []SpellEntry
	SpellCatalog bool

	// SlotMask and SlotInfo are the shown member's own doll, restated for the
	// shop's own figure (1005 round 2, `SHOP-FIGURE-041`, `SHOP-FIGURE-042`):
	// SlotMask is the mask composeInventorySubject built beside Figure — nil
	// whenever Figure carries none, on SlotMask.At's own nil handling — and
	// SlotInfo[n] is the characteristics of whatever equipment slot n+1 (1-based,
	// SlotMask's own numbering) holds, itemInfoLines' own line set, the shelf
	// and pack cells' own Info field restated per slot instead of per cell.
	// SlotMask stands substituted for the SUPPRESSED mask while a doll drag
	// on this member is armed (`ShopScreen`'s own suppression arm below).
	SlotMask *SlotMask
	SlotInfo [12][]string

	OrdinaryDollMask *SlotMask

	// SlotIcon[n] is slot n+1's own item picture, the doll's own counterpart of
	// ShopCell.Icon (round-2 adversarial review, item 2): the shop's own drag
	// machine reads this once, at the moment a doll-origin drag first crosses
	// TapSlop, exactly as command.go's own dragIcon reads invSubject.Slots[idx]
	// — the icon a released gesture carries on the cursor is the SAME picture
	// the worn box already draws for that slot, never a second load of its own.
	SlotIcon [12]*image.RGBA

	// Msg is the front-end's own last line, filled by this package before the
	// screen is composed: it is the answer to the press that just happened and
	// belongs to the flow, not to the shop.
	Msg string

	// TipPanel is the bordered tip widget for this room (1018 spec behaviour
	// 5, superseding the bare-text `Tip` field this replaces): SHOP-TIP-045's
	// own node loaded whole, never split into the game's own text-table
	// lines. A zero TipPanelView draws nothing (TipPanelView.Showing).
	TipPanel TipPanelView

	Art  *ShopScreenArt
	Font *text.Font
	// PriceFont is the grid painter's font2, the card font: it draws the
	// grouped figure on a price plaque (ITEM-PRICETAG-144) and every quantity
	// and purse (SHOP-106). The mission font does not fit the plaques (DIV-1405).
	// Nil falls back to Font.
	PriceFont *text.Font

	// Words carries the four button captions this build resolves from the
	// install (SHOP-050): ShopUndo, ShopBuy, ShopSell and ShopExit, in button
	// order. The caller always sets this from the front end's own resolved
	// word set (AuthoredWords or an install's own line), so the four fields
	// are never empty; a zero-value ShopScreenView built by hand (a test)
	// draws no caption, which is that test's own choice.
	Words Words
}

// ShopControlKind names one family of clickable rectangle on the screen.
type ShopControlKind uint8

const (
	ShopControlNone ShopControlKind = iota
	// ShopControlButton is one of the four command buttons, index 0..3.
	ShopControlButton
	// ShopControlArrowUp and ShopControlArrowDown scroll the shelf grid.
	ShopControlArrowUp
	ShopControlArrowDown
	// ShopControlShelfCell, ShopControlTableCell and ShopControlPackCell are
	// one cell of the named grid.
	ShopControlShelfCell
	ShopControlTableCell
	ShopControlPackCell
	// ShopControlShelfPick is one of the four shelf rectangles in the room
	// picture, index 0..3.
	ShopControlShelfPick
	// ShopControlMerchant is the merchant himself. He answers the hover
	// tooltip only: the original tests no press on him (TOWN-478), so a press
	// on him is not dispatched.
	ShopControlMerchant
	// ShopControlPickerPrev and ShopControlPickerNext are the character
	// panel's own two 32x32 rects (SHOP-PICKER-043). They step the shown
	// member, and the same press rebinds the bottom strip to his container.
	ShopControlPickerPrev
	ShopControlPickerNext
	ShopControlCharacterMode
	ShopControlBook
	// ShopControlSpell is an occupied or empty cell of the opened spellbook.
	// It deliberately carries no action: the shop reuses the book for reading
	// and hover information, not for casting.
	ShopControlSpell
	// ShopControlDoll is a slot of the shown member's own figure, Index the
	// zero-based slot shopDollSlotAt answers — dollFigureSlotAt's own
	// indexing (pkg/ui/inventory.go), restated for the shop's figure (1005
	// round 2). It names both a drag ORIGIN (a worn item) and a drag
	// DESTINATION (equip here); ShopControlAt never answers it — the doll is
	// not one of the geometry's three grids — so a caller that wants it
	// tests shopDollSlotAt itself, which shopGridControlAt below does first.
	ShopControlDoll
	ShopControlPackLeft
	ShopControlPackRight
)

// ShopControl is one control, named by its family and its index within it.
//
// Shift is the grid cells' quantity modifier (owner, DIV-046, DIV-047,
// DIV-1463): without it one unit of the cell's stack moves, with it the whole
// stack. ShopClick reads it on a shelf, table or pack cell and ShopDrag on
// such an origin; every other control ignores it. ShopControlAt never sets it
// — it is a hit test with no notion of a held key — so the caller fills it in
// from the release frame's input snapshot.
type ShopControl struct {
	Kind  ShopControlKind
	Index int
	Shift bool
}

// ShopControlAt is which control the point stands on.
//
// THE ORDER IS FIXED AND IT MATTERS IN ONE PLACE: the button panel and the
// merchant's room overlap in x 464..480, and the panel is tested first because
// its buttons are drawn over that strip. Everywhere else the families are
// disjoint, so the order is only a convention.
//
// A POINT IN THE GAP BETWEEN TWO CELLS IS A MISS. The three grids are tested
// cell by cell rather than by their regions, so the two-pixel gutters the cell
// formulas leave hit nothing (spec AC-11).
func ShopControlAt(p image.Point) (ShopControl, bool) {
	if p.In(shopPackLeftRect) {
		return ShopControl{Kind: ShopControlPackLeft}, true
	}
	if p.In(shopPackRightRect) {
		return ShopControl{Kind: ShopControlPackRight}, true
	}
	for i, r := range shopButtonRects {
		if p.In(r) {
			return ShopControl{Kind: ShopControlButton, Index: i}, true
		}
	}
	if p.In(shopArrowUpRect) {
		return ShopControl{Kind: ShopControlArrowUp}, true
	}
	if p.In(shopArrowDownRect) {
		return ShopControl{Kind: ShopControlArrowDown}, true
	}
	for i := 0; i < shopShelfN; i++ {
		if p.In(ShopShelfCellRect(i)) {
			return ShopControl{Kind: ShopControlShelfCell, Index: i}, true
		}
	}
	for i := 0; i < shopStripN; i++ {
		if p.In(ShopTableCellRect(i)) {
			return ShopControl{Kind: ShopControlTableCell, Index: i}, true
		}
		if p.In(ShopPackCellRect(i)) {
			return ShopControl{Kind: ShopControlPackCell, Index: i}, true
		}
	}
	if p.In(shopPickerPrevRect) {
		return ShopControl{Kind: ShopControlPickerPrev}, true
	}
	if p.In(shopPickerNextRect) {
		return ShopControl{Kind: ShopControlPickerNext}, true
	}
	if p.In(shopCharacterModeRect) {
		return ShopControl{Kind: ShopControlCharacterMode}, true
	}
	if shopBookCornerHitAt(p) {
		return ShopControl{Kind: ShopControlBook}, true
	}
	if p.In(shopMerchantRect) {
		return ShopControl{Kind: ShopControlMerchant}, true
	}
	for i, r := range shopShelfPickRects {
		if p.In(r) {
			return ShopControl{Kind: ShopControlShelfPick, Index: i}, true
		}
	}
	return ShopControl{}, false
}

// shopDollSlotAt reports which equipment slot, if any, owns the shown
// member's own VISIBLE figure pixel under point p — dollFigureSlotAt's own
// lookup (pkg/ui/inventory.go), restated for the shared town character region.
// The source and mask use shopFigureRect's 160x240 coordinates without
// centring; DrawTownCharacterRegion paints all 240 rows.
//
// THE ANSWER IS ZERO-BASED (0..11), dollFigureSlotAt's own indexing (1005
// round 2).
//
// The pane bounds, Book and the bottom controls refuse before the mask read.
// The newly visible feet between those controls remain interactive.
func shopDollSlotAt(v ShopScreenView, mask *SlotMask, p image.Point) (int, bool) {
	if mask == nil || mask.W <= 0 || mask.H <= 0 {
		return 0, false
	}
	if !shopDollAreaAt(v, p) {
		return 0, false
	}
	q := p.Sub(shopFigureRect.Min)
	n, ok := mask.At(q.X, q.Y)
	if !ok {
		return 0, false
	}
	return n - 1, true
}

// shopPaneTakesItemAt reports whether a drag released at p lands on the
// character pane, which takes a dragged item over its whole rectangle. The
// held item is tested before every corner, so no corner claims the release,
// and the figure's own pixels are not required (TOWN-348). The mission's
// heldItemBox is the same question.
func shopPaneTakesItemAt(p image.Point) bool {
	return p.In(shopCharRegion)
}

// shopDollAreaAt is shopDollSlotAt's own geometry, less the mask lookup: a
// point the doll's own drawn area could show a marked pixel at, regardless
// of whether one is drawn there right now. It answers where the figure's
// slots can be pressed; where an item is taken is shopPaneTakesItemAt.
func shopDollAreaAt(v ShopScreenView, p image.Point) bool {
	if !p.In(townCharacterFigure) {
		return false
	}
	// Keep the existing Book and bottom-control precedence while exposing
	// the feet between the bottom controls. The mode corner still shares
	// its figure pixels through the gesture split (DIV-308).
	if shopBookCornerHitAt(p) {
		return false
	}
	for _, c := range []CharacterPaneCorner{CharacterPaneBackpack, CharacterPanePrev, CharacterPaneNext, CharacterPaneMenu} {
		if p.In(CharacterPaneCornerRect(TownCharacterRegion, c)) {
			return false
		}
	}
	return true
}

// shopBookCornerHitAt reports whether p is inside rect B's own decoded hit
// rectangle on the shop's pane (owner finding 3, round-2 adversarial return:
// the authored "Book" plaque this project drew before the six corner
// controls existed is deleted; DrawTownCharacterRegion's own rect B, drawn
// with the shipped BookOpened.bmp/BookClosed.bmp art, is now the shop's own
// book control and answers its own click here rather than through
// shopBookRect, which is gone). CharacterPaneCornersAt returns every corner a
// point reaches, TOWN-355's own overlap included; this asks only whether
// Book is among them.
func shopBookCornerHitAt(p image.Point) bool {
	for _, c := range CharacterPaneCornersAt(shopCharRegion, CharacterPaneShop, 0, p) {
		if c == CharacterPaneBook {
			return true
		}
	}
	return false
}

// shopCharacterView is the one projection ComposeShopScreen paints and the
// shop doll hit path reads. The fallback preserves callers predating the
// shared-shell view without creating a second figure or font selection rule.
func shopCharacterView(v ShopScreenView) TownCharacterView {
	character := v.Character
	if !character.HasSubject && v.Figure != nil {
		character = TownCharacterView{HasSubject: true, Figure: v.Figure,
			Member: v.Member, MemberCount: v.MemberCount, Font: v.Font, CardFont: v.Character.CardFont}
	}
	return character
}

// shopGridControlAt is ShopControlAt widened with the doll (1005 round 2):
// the drag machine's own hit test, for both an origin and a destination,
// over the four surfaces the shop's own drag gestures name — the doll, the
// shelf grid, the pack strip and the table. THE DOLL IS TESTED FIRST, because
// townCharacterFigure sits entirely inside the merchant's room picture and
// over no grid cell at all, so the order carries no ambiguity to
// resolve; ShopControlAt's own arms are reused unchanged for the other three,
// and every OTHER kind ShopControlAt might answer — a button, an arrow, the
// picker, the merchant, a room rectangle — answers a miss here, because none
// of them is a drag surface this story's gestures use. Statistics mode
// removes the doll and keeps the three grids, so the doll is the one arm
// below that misses there.
func shopGridControlAt(v ShopScreenView, p image.Point) (ShopControl, bool) {
	if !v.Character.Statistics {
		if slot, ok := shopDollSlotAt(v, v.SlotMask, p); ok {
			return ShopControl{Kind: ShopControlDoll, Index: slot}, true
		}
	}
	if v.Book && p.In(shopTableRegion) {
		return ShopControl{}, false
	}
	if c, ok := ShopControlAt(p); ok {
		switch c.Kind {
		case ShopControlShelfCell, ShopControlPackCell, ShopControlTableCell:
			return c, true
		}
	}
	return ShopControl{}, false
}

// shopScreenControlAt applies the current presentation mode to the shop's
// static geometry.
//
// BOOK NO LONGER TAKES ANYTHING AWAY (owner finding 3, round-2 adversarial
// return). Before this round Book was the one control Statistics mode
// removed: the authored "Book" plaque sat on the statistics card's own name
// row and ComposeShopScreen drew neither its outline nor its label there, so
// a control the player could not see had to stop answering a press. The
// plaque is deleted; rect B's own decoded art (DrawTownCharacterRegion)
// draws in both modes alike (characterPaneCornerDrawn's own Book gate is
// `s&3 != 0`, unconditional on Statistics), so there is nothing left to
// suppress.
func shopScreenControlAt(v ShopScreenView, p image.Point) (ShopControl, bool) {
	if v.Book && p.In(shopTableRegion) {
		i := -1
		if _, n, ok := shopSpellEntryAt(v, p); ok {
			i = n
		}
		return ShopControl{Kind: ShopControlSpell, Index: i}, true
	}
	return ShopControlAt(p)
}

// The same inspection book serves all three city rooms.
func drawTownBook(dst *image.RGBA, v ShopScreenView) {
	if !v.Book {
		return
	}
	var book *image.RGBA
	if v.Art != nil && v.Art.Book != nil && v.SpellCatalog {
		book = composeOriginalSpellBar(v.Art.Book, v.Spells, 0, image.Rect(0, 0, 480, 85), 0)
	} else {
		book = composeSpellBar(v.Font, v.Spells, 0, shopSpellColumns(), shopTableRegion, 0)
	}
	blit(dst, book, shopTableRegion.Min.X, shopTableRegion.Min.Y)
}

func shopSpellColumns() int {
	return (shopTableRegion.Dx() - 2*hudBarPad + bookCellGap) / (bookCellSize + bookCellGap)
}

func shopSpellEntryAt(v ShopScreenView, p image.Point) (SpellEntry, int, bool) {
	if !v.Book {
		return SpellEntry{}, 0, false
	}
	for i, r := range bookCellRects(shopTableRegion, shopSpellColumns()) {
		if p.In(r) && i < len(v.Spells) {
			return v.Spells[i], i, true
		}
	}
	return SpellEntry{}, 0, false
}

// shopReleaseIsOrigin answers whether release point p names the exact cell a
// drag began on — app.go's own tremor guard, "the release is judged
// against its own origin, not the picture currently on screen" (round-2
// adversarial review, tenth pass, counterexample 1).
//
// A DOLL ORIGIN IS READ AGAINST OrdinaryDollMask, NEVER SlotMask. Once this
// drag has crossed TapSlop, `ShopSuppressDoll` (this member's own per-frame
// push) has already recomposed SlotMask with the origin's own slot cleared,
// so `shopGridControlAt`'s own `shopDollSlotAt(v, v.SlotMask, p)` can never
// answer the origin's own index again for as long as the drag stays armed —
// a release back on the exact pixel the press began on found no slot there
// at all, and the whole gesture (`ShopClick` AND `ShopDrag` alike) was
// silently dropped: the shop's own restatement of the mission's counterexample
// 1. The origin's identity does not change because the drawn picture did;
// OrdinaryDollMask is the mask in force before the drag suppressed anything,
// which is the question this asks.
//
// A NON-DOLL ORIGIN needs no such widening: shelf, pack and table cells carry
// no suppression, so shopGridControlAt's ordinary (possibly doll-suppressed,
// but that is irrelevant off the doll) read already answers correctly.
func shopReleaseIsOrigin(v ShopScreenView, origin ShopControl, p image.Point) bool {
	if origin.Kind == ShopControlDoll {
		slot, ok := shopDollSlotAt(v, v.OrdinaryDollMask, p)
		return ok && slot == origin.Index
	}
	dest, ok := shopGridControlAt(v, p)
	return ok && dest == origin
}

// ShopWheelRegion names which scrolling region the pointer stands in, so the
// wheel turns what is under it rather than always the same grid (owner,
// docs/DIVERGENCES.md DIV-005).
type ShopWheelRegion uint8

const (
	// ShopWheelNone is a pointer over nothing that scrolls. The wheel then
	// moves nothing at all, which is what "the region under the cursor" means
	// where that region has no rows.
	ShopWheelNone ShopWheelRegion = iota
	// ShopWheelShelf is the merchant's rack, top left. It pages by whole rows
	// of two, which is the arrows' own step (SHOP-SCREEN-033).
	ShopWheelShelf
	// ShopWheelPack is the shown member's container, along the bottom. It
	// moves by one place.
	ShopWheelPack
)

// ShopWheelAt is which region the point stands in. The table does not scroll:
// it holds five places and no more, and the original gives it no scroll base
// at all (SHOP-SCREEN-033).
func ShopWheelAt(p image.Point) ShopWheelRegion {
	switch {
	case p.In(shopShelfRegion):
		return ShopWheelShelf
	case p.In(shopPackRegion):
		return ShopWheelPack
	}
	return ShopWheelNone
}

// ShopHoverLines is the characteristics of the item under the point, and
// whether there is one. It answers only the three grids; every other control
// has no item to describe.
func ShopHoverLines(v ShopScreenView, p image.Point) ([]string, bool) {
	if spell, _, ok := shopSpellEntryAt(v, p); ok {
		// The same availability bit tooltipTarget's own spellbook branch
		// requires (TEXT-HOVERTEXT-052): the Book toggle reuses this popup
		// (DIV-118), so a cell the shown member cannot cast states nothing
		// here either.
		if spell.Unavailable {
			return nil, false
		}
		if len(spell.Info) > 0 {
			return spell.Info, true
		}
		if spell.Name != "" {
			return []string{spell.Name}, true
		}
		return nil, false
	}
	if !v.Character.Statistics {
		if slot, ok := shopDollSlotAt(v, v.SlotMask, p); ok {
			if slot < 0 || slot >= len(v.SlotInfo) || len(v.SlotInfo[slot]) == 0 {
				return nil, false
			}
			return v.SlotInfo[slot], true
		}
	}
	c, ok := shopScreenControlAt(v, p)
	if !ok {
		return nil, false
	}
	var cell ShopCell
	switch c.Kind {
	case ShopControlShelfCell:
		cell = v.Shelf[c.Index]
	case ShopControlTableCell:
		if v.Book {
			return nil, false
		}
		cell = v.Table[c.Index]
	case ShopControlPackCell:
		cell = v.Pack[c.Index]
	default:
		return nil, false
	}
	if !cell.Occupied() || len(cell.Info) == 0 {
		return nil, false
	}
	return cell.Info, true
}

// The colours of everything this build paints that no shipped picture covers:
// the unpainted background, the authored corner and the text on both. They are
// inventory.go's own two, reused rather than a third palette invented for one
// more framed box over the same kind of screen.
var (
	shopScreenFill = color.RGBA{R: 0x08, G: 0x09, B: 0x0c, A: 0xff}
	shopTextColor  = color.RGBA{R: 0xe8, G: 0xdc, B: 0xc0, A: 0xff}
	shopDimColor   = color.RGBA{R: 0x70, G: 0x68, B: 0x58, A: 0xff}
	shopMarkColor  = color.RGBA{R: 0xff, G: 0xd0, B: 0x60, A: 0xff}

	// shopShadowColor is what a line drawn straight onto the room picture is
	// offset by one pixel in, so it reads against the floor without a box
	// behind it.
	shopShadowColor = color.RGBA{R: 0x08, G: 0x06, B: 0x04, A: 0xff}
)

// blit draws src at (x, y) over dst, honouring src's alpha.
func blit(dst, src *image.RGBA, x, y int) {
	if dst == nil || src == nil {
		return
	}
	b := src.Bounds()
	draw.Draw(dst, image.Rect(x, y, x+b.Dx(), y+b.Dy()), src, b.Min, draw.Over)
}

func blitPicture(dst *image.RGBA, src image.Image, x, y int) {
	if dst == nil || src == nil {
		return
	}
	b := src.Bounds()
	draw.Draw(dst, image.Rect(x, y, x+b.Dx(), y+b.Dy()), src, b.Min, draw.Over)
}

// shopPlaqueIndex is which of the seven plaques a stored unit price takes: its
// decimal digit count minus one, clamped to the seventh (SHOP-SCREEN-037).
//
// IT IS A DIGIT COUNT AND NOT A LOGARITHM. The original computes
// floor(log10(price)) in the x87 unit over a value the price clamp holds to
// seven digits, which over that domain is exactly this count; and this package
// may not use floats (the determinism wall's own habit, kept here because an
// integer answer exists).
//
// A PRICE OF 0 OR LESS TAKES THE FIRST PLAQUE. log10 has no value there, and
// the x87 integer-indefinite result the original converts has the low dword 0
// (ITEM-PRICETAG-144, derived): the loop below never runs for such a price.
func shopPlaqueIndex(price int32) int {
	d := 0
	for n := price / 10; n > 0; n /= 10 {
		d++
	}
	if d > 6 {
		d = 6
	}
	return d
}

// drawShopQuantity paints a quantity or purse left-aligned in the ramp over a
// flat shadow. The caller passes font2, the grid painter's font (SHOP-106).
func drawShopQuantity(dst *image.RGBA, font *text.Font, count uint32, r image.Rectangle) {
	digits := GroupDigits(int64(count))
	x, y := r.Min.X+10, r.Max.Y-15
	font.DrawFlat(dst, digits, x+shopPriceShadow, y+shopPriceShadow, messageShadowColor)
	font.Draw(dst, digits, x, y, shopPriceInk)
}

// drawShopCell paints one cell: background, picture, quantity, plaque.
//
// A CELL WITH NO ELEMENT PAINTS NOTHING (ShopCell.Occupied's own doc). The
// money element paints backinv.bmp and the coin over it and carries no icon and
// no plaque (SHOP-SCREEN-036 arm a, SHOP-MONEY-048).
//
// EVERY OTHER ELEMENT PAINTS ITS PLAQUE AND FIGURE WHATEVER THE PRICE'S SIGN.
// The original's painter skips only the money element and a quantity of 0
// (ITEM-PRICETAG-144), so a price of 0 or -1 draws the first plaque and its
// number like any other.
func drawShopCell(dst *image.RGBA, art *ShopScreenArt, font, priceFont *text.Font, cell ShopCell, r image.Rectangle, fixedPlaces bool, field *itemStarField) {
	if !cell.Occupied() && !cell.Money && fixedPlaces && art != nil {
		// A CONTAINER'S PLACE IS AN ELEMENT OF QUANTITY ZERO, which is
		// SHOP-SCREEN-036 arm b and draws backinvg.bmp. Only the bottom strip
		// is a container: the shelf grid is a list that scrolls and the table
		// is five places on a painted table, and both of those have their own
		// picture under them. The strip's own region has none — the view fills
		// it flat (SHOP-VIEW-044) — so its places are what makes it read as a
		// row of cells rather than a black band.
		blit(dst, art.Back[ShopBackEmpty], r.Min.X, r.Min.Y)
		return
	}
	// The grid painter draws quantity and price in one font, font2 (SHOP-106).
	// Font is the fallback when no font2 resolved.
	if priceFont == nil {
		priceFont = font
	}
	if cell.Money {
		if art != nil {
			blit(dst, art.Back[ShopBackItem], r.Min.X, r.Min.Y)
			blit(dst, art.Coin, r.Min.X, r.Min.Y)
		}
		if cell.Count > 1 && priceFont != nil {
			drawShopQuantity(dst, priceFont, cell.Count, r)
		}
		return
	}
	if !cell.Occupied() {
		return
	}
	if art != nil && int(cell.Back) < len(art.Back) {
		blit(dst, art.Back[cell.Back], r.Min.X, r.Min.Y)
	}
	if cell.Icon != nil {
		b := cell.Icon.Bounds()
		blit(dst, cell.Icon, r.Min.X+(shopCellSize-b.Dx())/2, r.Min.Y+(shopCellSize-b.Dy())/2)
	}
	// Shop order is background, base icon, trail, quantity, then the later
	// price layers (ITEM-STARCOMP-100). The +1 y origin is the measured shop
	// painter's own placement; drawItemStarTrail applies only the global clip.
	if cell.Star && cell.Icon != nil {
		drawItemStarTrail(dst, image.Pt(r.Min.X, r.Min.Y+1), cell.StarPhase, field)
	}
	if cell.Count > 1 && priceFont != nil {
		drawShopQuantity(dst, priceFont, cell.Count, r)
	}
	drawShopPlaque(dst, art, priceFont, cell, r)
}

// drawShopPlaque paints the price plaque and the figure on it (SHOP-SCREEN-037,
// ITEM-PRICETAG-144): the plaque family is the side of the deal, the plaque
// within the family is the digit count of the stored price, which is not the
// figure's on the player's side, and the bitmap is right-aligned to the cell's
// right edge. The figure is painted after the plaque, right-aligned
// so that its pen ends shopPriceRightInset pixels left of the cell's right edge,
// one pixel below the cell's top, in the gold ramp over a flat shadow one pixel
// right and down, as the text renderer draws every line.
func drawShopPlaque(dst *image.RGBA, art *ShopScreenArt, font *text.Font, cell ShopCell, r image.Rectangle) {
	if pic := shopPlaquePicture(art, cell); pic != nil {
		blit(dst, pic, r.Max.X-pic.Bounds().Dx(), r.Min.Y+1)
	}
	if font == nil {
		return
	}
	_, figure := ShopPricePlacement(art, font, cell, r)
	digits := GroupDigits(int64(cell.Price))
	font.DrawFlat(dst, digits, figure.Min.X+shopPriceShadow, figure.Min.Y+shopPriceShadow, messageShadowColor)
	font.Draw(dst, digits, figure.Min.X, figure.Min.Y, shopPriceInk)
}

func shopPlaquePicture(art *ShopScreenArt, cell ShopCell) *image.RGBA {
	if art == nil {
		return nil
	}
	side := 0
	if cell.Mine {
		side = 1
	}
	return art.Plaque[side][shopPlaqueIndex(cell.PlaquePrice)]
}

// ShopPricePlacement is where a priced cell in r draws its plaque's ink and
// its grouped figure in font, in screen coordinates. The figure box begins at
// the text's draw origin, which lies the text's advance left of the pen's end
// (ITEM-PRICETAG-144), and holds every pixel the text paints; its shadow stands
// shopPriceShadow further right and down. The figure box is empty when font is
// nil.
func ShopPricePlacement(art *ShopScreenArt, font *text.Font, cell ShopCell, r image.Rectangle) (ink, figure image.Rectangle) {
	if pic := shopPlaquePicture(art, cell); pic != nil {
		ink = opaqueBounds(pic).Add(image.Pt(r.Max.X-pic.Bounds().Dx(), r.Min.Y+1))
	}
	if font == nil {
		return ink, image.Rectangle{}
	}
	digits := GroupDigits(int64(cell.Price))
	tw, th := font.Measure(digits)
	x, y := r.Max.X-shopPriceRightInset-font.Advance(digits), r.Min.Y+shopPriceTop
	return ink, image.Rect(x, y, x+tw, y+th)
}

// opaqueBounds is the smallest rectangle holding every pixel of pic that is not
// wholly transparent, in pic's own coordinates. An entirely transparent picture
// answers the empty rectangle.
func opaqueBounds(pic *image.RGBA) image.Rectangle {
	if pic == nil {
		return image.Rectangle{}
	}
	b := pic.Bounds()
	out := image.Rectangle{Min: b.Max, Max: b.Min}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if pic.RGBAAt(x, y).A == 0 {
				continue
			}
			if x < out.Min.X {
				out.Min.X = x
			}
			if y < out.Min.Y {
				out.Min.Y = y
			}
			if x >= out.Max.X {
				out.Max.X = x + 1
			}
			if y >= out.Max.Y {
				out.Max.Y = y + 1
			}
		}
	}
	if out.Empty() {
		return image.Rectangle{}
	}
	return out
}

// outline draws a one-pixel rectangle, which is how this build marks the chosen
// shelf and the live button under the pointer. Nothing decoded says how the
// original marks either.
func outline(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	if dst == nil || r.Empty() {
		return
	}
	for x := r.Min.X; x < r.Max.X; x++ {
		dst.SetRGBA(x, r.Min.Y, c)
		dst.SetRGBA(x, r.Max.Y-1, c)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		dst.SetRGBA(r.Min.X, y, c)
		dst.SetRGBA(r.Max.X-1, y, c)
	}
}

// ComposeShopScreen paints the whole 640x480 screen.
//
// hover is where the pointer stands and whether it is on the screen at all;
// the characteristics box is drawn beside it.
//
// DIV-090
func ComposeShopScreen(v ShopScreenView, hover image.Point, hasHover bool, dragIcon *image.RGBA, hasDrag bool) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, shopScreenW, shopScreenH))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: shopScreenFill}, image.Point{}, draw.Src)
	art := v.Art

	if art != nil {
		blit(dst, art.Shelf, shopShelfRegion.Min.X, shopShelfRegion.Min.Y)
		blit(dst, art.Room, shopRoomOrigin.X, shopRoomOrigin.Y)
		blit(dst, art.Frame, shopFrameOrigin.X, shopFrameOrigin.Y)
		blit(dst, art.Table, shopTableRegion.Min.X, shopTableRegion.Min.Y)
	}
	panel := shopButtonPanel(v, hover, hasHover)
	panel.drawBody(dst)
	panel.drawButtons(dst, v.Font)
	if art != nil {
		blit(dst, art.Arrow[0], shopArrowUpRect.Min.X, shopArrowUpRect.Min.Y)
		blit(dst, art.Arrow[1], shopArrowDownRect.Min.X, shopArrowDownRect.Min.Y)
		blit(dst, art.PackCap[0], shopPackLeftRect.Min.X, shopPackLeftRect.Min.Y)
		blit(dst, art.PackCap[1], shopPackRightRect.Min.X, shopPackRightRect.Min.Y)
		for i, on := range [2]bool{v.PackBack, v.PackForward} {
			r := [2]image.Rectangle{shopPackLeftRect, shopPackRightRect}[i]
			if on {
				n := i
				if hasHover && hover.In(r) {
					n += 2
				}
				blit(dst, art.PackArrow[n], r.Min.X, r.Min.Y+packArrowDY)
			}
		}
		if v.Scene != nil {
			v.Scene.Paint(dst, "interior")
		}
	}
	for i := 0; i < shopShelfN; i++ {
		drawShopCell(dst, art, v.Font, v.PriceFont, v.Shelf[i], ShopShelfCellRect(i), false, &itemStarFields[1])
	}
	for i := 0; i < shopStripN; i++ {
		if !v.Book {
			drawShopCell(dst, art, v.Font, v.PriceFont, v.Table[i], ShopTableCellRect(i), false, &itemStarFields[2])
		}
		drawShopCell(dst, art, v.Font, v.PriceFont, v.Pack[i], ShopPackCellRect(i), true, &itemStarFields[3])
	}
	drawTownBook(dst, v)

	// THE DOLL/STATS TOGGLE CHANGES ONLY THE CHARACTER PANE (owner, hotfix: the
	// toggle just toggles the doll and the stats). ComposeShopScreen called
	// drawTownStatisticsSurface here until that report, which filled
	// TownContentRegion, the whole 480x480 left of the screen, with the shell's
	// near-black panel colour and left only the 160-wide character column
	// drawn. 1022 spec B6 moved the statistics card out of that region and into
	// the character pane, and removed the same blanking from the tavern and the
	// school (ComposeTownSurface's own note), but left the shop's own grids
	// suppressed. The shelf, pack, table, buttons, merchant and room art now
	// draw identically in both modes. DrawTownCharacterRegion below is the only
	// thing the mode reaches. DIV-193.
	//
	// The upper widget's second paint went with it: art.Menu and the four
	// button bitmaps are drawn once above, and the repeat here existed only
	// to restore the x=464..480 overlap the blanking had covered.
	character := shopCharacterView(v)

	DrawTownCharacterRegion(dst, character)
	if !character.Statistics {
		drawShopMessage(dst, v)
	}

	if hasHover && !v.SuppressHover {
		if lines, ok := ShopHoverLines(v, hover); ok {
			drawShopHover(dst, v.Font, lines, hover)
		}
	}

	if hasDrag && dragIcon != nil && hasHover {
		b := dragIcon.Bounds()
		blit(dst, dragIcon, hover.X-b.Dx()/2, hover.Y-b.Dy()/2)
	}

	// The tip panel draws LAST, over everything this function has drawn
	// above including the statistics surface's own content and any drag
	// icon (1018 spec behaviour 1, "draws over the screen's other content
	// and takes input before it") — the opposite of the retired drawShopTip,
	// which the room's furniture blits and a statistics view could both
	// paint over with no gate of their own.
	ComposeTipPanel(dst, v.TipPanel)
	return dst
}

// shopPanel is the shop's four-button composition (SHOP-SCREEN-035):
// shopmenu.bmp over TownWideUpperRegion and four plaques whose bitmaps draw
// only while pressed and hovered (TOWN-260). The four-command tavern takes it
// whole (DIV-483).
var shopPanel = panelComposition{Body: shopButtonRegion, BodyOver: true, Plaques: shopButtonRects[:],
	FitLine: true, Ink: plaqueCommandInk}

// shopButtonPanel is the shop's command panel; Buy and Sell join caption and
// number on one line (SHOP-050).
func shopButtonPanel(v ShopScreenView, hover image.Point, hasHover bool) buttonPanel {
	var art panelArt
	if v.Art != nil {
		if v.Art.Menu != nil {
			art.Body = v.Art.Menu
		}
		art.Plaques = make([][2]image.Image, len(shopButtonRects))
		for i, pic := range v.Art.Button {
			if pic != nil {
				art.Plaques[i][1] = pic
			}
		}
	}
	labels := [4]string{v.Words.ShopUndo, v.Words.ShopBuy, v.Words.ShopSell, v.Words.ShopExit}
	numbers := [4]int32{v.Purse, v.Buy, v.Sell, v.Total}
	buttons := make([]panelButton, len(shopButtonRects))
	for i, r := range shopButtonRects {
		n := GroupDigits(int64(numbers[i]))
		b := &buttons[i]
		if i == 1 || i == 2 {
			b.Captions = []string{labels[i] + " " + n}
		} else {
			b.Captions = []string{labels[i], n}
		}
		b.Hover = hasHover && hover.In(r)
		b.Inside, b.Disabled = b.Hover, !v.Live[i]
		b.Pressed = v.Press == (ShopControl{Kind: ShopControlButton, Index: i})
	}
	return buildButtonPanel(shopPanel, art, buttons)
}

// shopMessageRect is where the answer to the last press is written: the bottom
// of the tip widget's own rect, which is panel-relative (0,162,312,298) inside
// the merchant panel and so (164,162)-(476,298) on the frame (SHOP-TIP-045).
// The room picture's stone floor is what lies under it.
var shopMessageRect = image.Rect(169, 283, 476, 298)

// drawShopMessage writes the front end's own last line over the room's floor,
// with a one-pixel shadow rather than a box: a filled box here would be a new
// dark rectangle on a screen whose whole complaint was dark rectangles.
func drawShopMessage(dst *image.RGBA, v ShopScreenView) {
	if v.Font == nil || v.Msg == "" {
		return
	}
	s := v.Msg
	for {
		w, _ := v.Font.Measure(s)
		if w <= shopMessageRect.Dx() || len(s) <= 1 {
			break
		}
		s = s[:len(s)-1]
	}
	w, _ := v.Font.Measure(s)
	x := shopMessageRect.Min.X + (shopMessageRect.Dx()-w)/2
	v.Font.Draw(dst, s, x+1, shopMessageRect.Min.Y+1, shopShadowColor)
	v.Font.Draw(dst, s, x, shopMessageRect.Min.Y, shopTextColor)
}

// wrapShopTip splits s into lines that each measure no wider than width under
// font, breaking only between words. A run with no spaces at all, or a single
// word wider than width on its own, is returned as one line rather than cut
// mid-glyph: this build has no hyphenation rule to reproduce or author.
func wrapShopTip(font *text.Font, s string, width int) []string {
	var lines []string
	s = strings.ReplaceAll(s, "\r\n", "\n")
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		line := words[0]
		for _, w := range words[1:] {
			candidate := line + " " + w
			if cw, _ := font.Measure(candidate); cw <= width {
				line = candidate
				continue
			}
			lines = append(lines, line)
			line = w
		}
		lines = append(lines, line)
	}
	return lines
}

// drawShopChevron paints one picker rect: a framed box with a triangle pointing
// the way it steps. A picker with one member to show is drawn dim and its press
// moves nothing.
// drawShopChevron draws the prev/next control's own outline and its
// direction triangle. fill draws the outline's own opaque interior first
// (invFill), matching this function's original behaviour; a caller whose
// control now sits over shipped art passes false instead (round-2
// adversarial review, owner item: the near-black control boxes now sitting
// on shipped art — townCharacterPrev/townCharacterNext, drawn over
// DrawTownCharacterRegion's own shipped body/seam since this story, no
// longer over the authored townShellPanel fill that used to sit under
// everything there).
func drawShopChevron(dst *image.RGBA, r image.Rectangle, dir int, live, fill bool) {
	if fill {
		draw.Draw(dst, r, &image.Uniform{C: invFill}, image.Point{}, draw.Src)
	}
	outline(dst, r, invBorder)
	c := shopDimColor
	if live {
		c = shopMarkColor
	}
	box := r.Inset(8)
	mid := (box.Min.Y + box.Max.Y) / 2
	for i := 0; i < box.Dx(); i++ {
		// i counts away from the apex, which stands at the edge the press
		// moves towards, so the triangle points the way it steps.
		x := box.Max.X - 1 - i
		if dir < 0 {
			x = box.Min.X + i
		}
		for y := mid - i/2; y <= mid+i/2; y++ {
			if image.Pt(x, y).In(box) {
				dst.SetRGBA(x, y, c)
			}
		}
	}
}

// drawShopHover paints the characteristics box beside the pointer, kept
// inside the frame on both axes.
func drawShopHover(dst *image.RGBA, font *text.Font, lines []string, at image.Point) {
	if font == nil || len(lines) == 0 {
		return
	}
	w, line := 0, font.Height()+2
	for _, s := range lines {
		if m, _ := font.Measure(s); m > w {
			w = m
		}
	}
	box := image.Rect(0, 0, w+10, len(lines)*line+8).Add(at.Add(image.Pt(shopHoverGap, shopHoverGap)))
	if box.Max.X > shopScreenW {
		box = box.Sub(image.Pt(box.Max.X-shopScreenW, 0))
	}
	if box.Max.Y > shopScreenH {
		box = box.Sub(image.Pt(0, box.Max.Y-shopScreenH))
	}
	draw.Draw(dst, box, &image.Uniform{C: shopScreenFill}, image.Point{}, draw.Src)
	outline(dst, box, invBorder)
	for i, s := range lines {
		font.Draw(dst, s, box.Min.X+5, box.Min.Y+4+i*line, shopTextColor)
	}
}

// shopHoverGap is how far the hover box stands from the pointer, on both axes.
// It is itemPopupOffset's own number for itemPopupOffset's own reason: far
// enough that the pointer itself does not sit on the first glyph.
const shopHoverGap = 14
