package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/render/text"
)

// The shop screen's geometry and composition (0157 AC-11, AC-14, AC-18, AC-19).
//
// EVERY NUMBER ASSERTED HERE IS THE DECODED ONE, spelt out rather than read back
// out of the constant it is testing: a test that computes its expectation the
// same way the code does cannot fail when both are wrong.

// AC-11: every control lies inside the frame, and no two of different kinds
// overlap.
func TestEveryShopControlIsInsideTheFrameAndDisjoint(t *testing.T) {
	frame := image.Rect(0, 0, 640, 480)
	type named struct {
		name string
		r    image.Rectangle
	}
	var all []named
	add := func(name string, r image.Rectangle) {
		if !r.In(frame) {
			t.Errorf("%s = %v, outside the 640x480 frame", name, r)
		}
		all = append(all, named{name, r})
	}
	for i, r := range shopButtonRects {
		add("button", r)
		if i > 0 && r.Min.Y < shopButtonRects[i-1].Max.Y-1 {
			t.Errorf("button %d starts at y=%d, above button %d's end", i, r.Min.Y, i-1)
		}
	}
	add("arrow up", shopArrowUpRect)
	add("arrow down", shopArrowDownRect)
	for i := 0; i < shopShelfN; i++ {
		add("shelf cell", ShopShelfCellRect(i))
	}
	for i := 0; i < shopStripN; i++ {
		add("table cell", ShopTableCellRect(i))
		add("pack cell", ShopPackCellRect(i))
	}
	for _, r := range shopShelfPickRects {
		add("shelf pick", r)
	}
	add("character panel", shopCharRegion)
	add("merchant", shopMerchantRect)
	add("picker prev", shopPickerPrevRect)
	add("picker next", shopPickerNextRect)

	// ONE PAIR OF DECODED RECTANGLES OVERLAPS AND THE HIT TEST'S FIXED ORDER
	// RESOLVES IT: the up arrow's bottom edge, y=32, is one row past the first
	// cell row's top, y=31. That is the original's own pair of numbers.
	//
	// The picker rectangles lie inside the character panel by construction, so
	// that pair is allowed too.
	//
	// SINCE THE SHELF RECTANGLES BECAME THE HIT ARRAY (SHOP-SHELF-047) THE
	// BUTTON PANEL NO LONGER OVERLAPS THEM: the hit rectangles stop at x 459
	// and the buttons start at 483. The draw rectangles the round-2 build used
	// reached x 445 and were allowed against the buttons; that allowance is
	// gone, so the disjointness is now asserted rather than excused.
	allowed := map[string]bool{
		"arrow up|shelf cell":         true,
		"character panel|picker prev": true,
		"character panel|picker next": true,
	}
	for i, a := range all {
		for _, b := range all[i+1:] {
			if allowed[a.name+"|"+b.name] || allowed[b.name+"|"+a.name] {
				continue
			}
			if !a.r.Intersect(b.r).Empty() {
				t.Errorf("%s %v overlaps %s %v", a.name, a.r, b.name, b.r)
			}
		}
	}

	// The one shared pixel row belongs to the arrow, which is tested first.
	if c, ok := ShopControlAt(image.Pt(60, 31)); !ok || c.Kind != ShopControlArrowUp {
		t.Errorf("the shared row at y=31 resolved to %+v, %v, want the up arrow", c, ok)
	}
}

// AC-11: the decoded cell formulas, checked at their own corners, and the
// gutters between cells that hit nothing.
func TestTheCellFormulasAreTheDecodedOnes(t *testing.T) {
	if got := ShopShelfCellRect(0); got != image.Rect(1, 31, 81, 111) {
		t.Errorf("shelf cell 0 = %v, want (1,31)-(81,111)", got)
	}
	if got := ShopShelfCellRect(5); got != image.Rect(81, 191, 161, 271) {
		t.Errorf("shelf cell 5 = %v, want the second column's third row", got)
	}
	if got := ShopTableCellRect(4); got != image.Rect(352, 303, 432, 383) {
		t.Errorf("table cell 4 = %v, want (352,303)-(432,383)", got)
	}
	if got := ShopPackCellRect(0); got != image.Rect(32, 395, 112, 475) {
		t.Errorf("pack cell 0 = %v, want (32,395)-(112,475)", got)
	}

	for i := 0; i < shopStripN; i++ {
		p := ShopTableCellRect(i).Min.Add(image.Pt(40, 40))
		if c, ok := ShopControlAt(p); !ok || c.Kind != ShopControlTableCell || c.Index != i {
			t.Errorf("the middle of table cell %d resolved to %+v, %v", i, c, ok)
		}
	}
	// The two-pixel gutter between two table cells belongs to neither.
	if c, ok := ShopControlAt(image.Pt(432, 340)); ok {
		t.Errorf("the gap past the last table cell resolved to %+v", c)
	}
	// The shelf grid's own top strip is the up arrow, not a cell.
	if c, ok := ShopControlAt(image.Pt(80, 16)); !ok || c.Kind != ShopControlArrowUp {
		t.Errorf("the top of the shelf grid resolved to %+v, %v, want the up arrow", c, ok)
	}
	// The four rectangles in the room are clicked in the room's own areas, in
	// the HIT array's index order (SHOP-SHELF-047): 0 lower right, 1 lower
	// left, 2 upper right, 3 upper left.
	for _, tc := range []struct {
		at   image.Point
		want int
	}{
		{image.Pt(400, 200), 0},
		{image.Pt(220, 200), 1},
		{image.Pt(380, 50), 2},
		{image.Pt(250, 60), 3},
	} {
		c, ok := ShopControlAt(tc.at)
		if !ok || c.Kind != ShopControlShelfPick || c.Index != tc.want {
			t.Errorf("the room at %v resolved to %+v, %v, want shelf %d", tc.at, c, ok, tc.want)
		}
	}
	// The merchant stands in the column the four hit rectangles leave free.
	if c, ok := ShopControlAt(image.Pt(310, 200)); !ok || c.Kind != ShopControlMerchant {
		t.Errorf("the merchant resolved to %+v, %v", c, ok)
	}
	// The character panel's own two picker rectangles.
	if c, ok := ShopControlAt(image.Pt(490, 450)); !ok || c.Kind != ShopControlPickerPrev {
		t.Errorf("the previous-member arrow resolved to %+v, %v", c, ok)
	}
	if c, ok := ShopControlAt(image.Pt(610, 450)); !ok || c.Kind != ShopControlPickerNext {
		t.Errorf("the next-member arrow resolved to %+v, %v", c, ok)
	}
	// The button panel wins where it overlaps the room.
	if c, ok := ShopControlAt(image.Pt(500, 30)); !ok || c.Kind != ShopControlButton || c.Index != 0 {
		t.Errorf("the first button resolved to %+v, %v", c, ok)
	}
}

// AC-14: the plaque a price selects is its decimal digit count minus one,
// clamped to the seventh (SHOP-SCREEN-037). A price of 0 or less takes the first
// (ITEM-PRICETAG-144).
func TestThePlaqueIsChosenByTheDigitCount(t *testing.T) {
	for _, tc := range []struct {
		price int32
		want  int
	}{
		{-2147483648, 0}, {-1250, 0}, {-1, 0},
		{0, 0}, {1, 0}, {5, 0}, {9, 0},
		{10, 1}, {99, 1},
		{100, 2}, {1000, 3}, {10000, 4}, {100000, 5},
		{1000000, 6}, {9999999, 6},
		// The clamp holds past the price ceiling, so no index can leave the
		// seven-entry array.
		{99999999, 6},
	} {
		if got := shopPlaqueIndex(tc.price); got != tc.want {
			t.Errorf("shopPlaqueIndex(%d) = %d, want %d", tc.price, got, tc.want)
		}
	}
}

// AC-19: with no artwork and no font the whole screen still composes, at the
// frame's own size, and nothing panics.
func TestTheScreenComposesWithNoInstallAtAll(t *testing.T) {
	v := ShopScreenView{Purse: 100, Buy: 40, Sell: 9, Chosen: 2}
	v.Shelf[0] = ShopCell{Back: ShopBackAffordable, Price: 40, Count: 1}
	v.Table[0] = ShopCell{Back: ShopBackItem, Price: 20, Mine: true, Count: 3}
	v.Pack[1] = ShopCell{Back: ShopBackItem, Price: 7, Mine: true, Count: 1,
		Info: []string{"a sword", "Damage 1-3"}}

	pic := ComposeShopScreen(v, image.Pt(80, 430), true, nil, false)
	if pic == nil {
		t.Fatal("composed nothing")
	}
	if got := pic.Bounds(); got != image.Rect(0, 0, 640, 480) {
		t.Fatalf("the screen is %v, want the 640x480 frame", got)
	}
}

func TestComposeShopScreenDrawsTheButtonCaptionFromWords(t *testing.T) {
	font := &text.Font{Glyphs: make([]text.Glyph, 224)}
	font.Glyphs['A'-text.FirstChar] = text.Glyph{Width: 3, Height: 3, Advance: 4,
		Pixels: []text.Pixel{
			{}, {}, {},
			{}, {Level: text.MaxLevel, Painted: true}, {},
			{}, {}, {},
		}}

	withWords := ShopScreenView{Font: font, Words: Words{ShopUndo: "A", ShopBuy: "A", ShopSell: "A", ShopExit: "A"}}
	withoutWords := ShopScreenView{Font: font}

	withPic := ComposeShopScreen(withWords, image.Point{}, false, nil, false)
	withoutPic := ComposeShopScreen(withoutWords, image.Point{}, false, nil, false)

	for i, r := range shopButtonRects {
		region := image.Rect(r.Min.X, r.Min.Y+2, r.Max.X, r.Min.Y+24)
		differs := false
		for y := region.Min.Y; y < region.Max.Y && !differs; y++ {
			for x := region.Min.X; x < region.Max.X; x++ {
				if withPic.RGBAAt(x, y) != withoutPic.RGBAAt(x, y) {
					differs = true
					break
				}
			}
		}
		if !differs {
			t.Errorf("button %d: the caption region is identical with and without Words", i)
		}
	}
}

// TestComposeShopScreenDrawsTheDragIconUnderTheCursor is Counterexample 2's
// own witness (round-2 adversarial review): `contract.md`'s "carried on the
// cursor" clause names the shop grid explicitly, and this is the one call
// the shop's own software compositor makes that draws it — centred on
// hover, drawn last, and only while a drag genuinely carries a picture
// (`DIV-090` retired).
func TestComposeShopScreenDrawsTheDragIconUnderTheCursor(t *testing.T) {
	icon := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			icon.SetRGBA(x, y, color.RGBA{R: 0xff, A: 0xff})
		}
	}
	hover := image.Pt(300, 100) // clear of every furniture rect at art == nil

	pic := ComposeShopScreen(ShopScreenView{Chosen: -1}, hover, true, icon, true)
	if got := pic.RGBAAt(hover.X, hover.Y); got.R != 0xff || got.A != 0xff {
		t.Errorf("(%d,%d) = %+v, want the icon's own opaque red — centred on hover", hover.X, hover.Y, got)
	}

	// hasHover false, dragIcon nil, and hasDrag false each independently
	// suppress the blit: the background fill is unchanged at hover.
	fill := color.RGBA{R: 0x08, G: 0x09, B: 0x0c, A: 0xff}
	for _, tc := range []struct {
		name              string
		hasHover, hasDrag bool
		icon              *image.RGBA
	}{
		{"no hover", false, true, icon},
		{"nil icon", true, true, nil},
		{"drag not armed", true, false, icon},
	} {
		p := ComposeShopScreen(ShopScreenView{Chosen: -1}, hover, tc.hasHover, tc.icon, tc.hasDrag)
		if got := p.RGBAAt(hover.X, hover.Y); got != fill {
			t.Errorf("%s: (%d,%d) = %+v, want the plain background fill %+v", tc.name, hover.X, hover.Y, got, fill)
		}
	}
}

// AC-18: the hover answers the cell under the pointer and nothing anywhere
// else.
//
// SHOP-SCREEN-039
func TestTheHoverAnswersTheCellUnderThePointer(t *testing.T) {
	v := ShopScreenView{Chosen: 2}
	v.Pack[1] = ShopCell{Back: ShopBackItem, Info: []string{"a sword", "Damage 1-3"}}
	v.Table[0] = ShopCell{Back: ShopBackItem}

	inside := ShopPackCellRect(1).Min.Add(image.Pt(40, 40))
	lines, ok := ShopHoverLines(v, inside)
	if !ok || len(lines) != 2 || lines[0] != "a sword" {
		t.Errorf("the hover over pack cell 1 answered %v, %v", lines, ok)
	}
	// An occupied cell with no lines, an empty cell and a button all answer
	// nothing.
	for _, p := range []image.Point{
		ShopTableCellRect(0).Min.Add(image.Pt(40, 40)),
		ShopTableCellRect(3).Min.Add(image.Pt(40, 40)),
		shopButtonRects[1].Min.Add(image.Pt(10, 10)),
	} {
		if lines, ok := ShopHoverLines(v, p); ok {
			t.Errorf("the hover at %v answered %v", p, lines)
		}
	}
}

// solid is an n-by-n picture of one opaque colour, for the composition tests
// below. It reads no archive (golden rule 2).
func solid(n int, c color.RGBA) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, n, n))
	for i := 0; i+3 < len(pic.Pix); i += 4 {
		pic.Pix[i], pic.Pix[i+1], pic.Pix[i+2], pic.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return pic
}

// testArt is a whole art set of flat colours, one per slot, so a composed pixel
// names which picture painted it.
func testArt() *ShopScreenArt {
	art := &ShopScreenArt{Coin: solid(shopCellSize, color.RGBA{R: 9, A: 0xff})}
	for i := range art.Back {
		art.Back[i] = solid(shopCellSize, color.RGBA{R: uint8(i + 1), A: 0xff})
	}
	return art
}

// A CELL WITH NO ELEMENT PAINTS NOTHING where its grid has its own picture
// under it, and a place of the bottom strip paints backinvg.bmp.
func TestAnEmptyPlacePaintsOnlyWhereTheStripHasNoPictureOfItsOwn(t *testing.T) {
	v := ShopScreenView{Chosen: -1, Art: testArt()}
	pic := ComposeShopScreen(v, image.Point{}, false, nil, false)

	for _, tc := range []struct {
		name string
		at   image.Point
		want uint8
	}{
		// The art set below carries no region picture, so what shows through
		// an unpainted place is the screen's own fill.
		{"an empty shelf place", ShopShelfCellRect(0).Min.Add(image.Pt(40, 40)), shopScreenFill.R},
		{"an empty table place", ShopTableCellRect(0).Min.Add(image.Pt(40, 40)), shopScreenFill.R},
		{"an empty strip place", ShopPackCellRect(1).Min.Add(image.Pt(40, 40)), 1},
	} {
		if got := pic.RGBAAt(tc.at.X, tc.at.Y).R; got != tc.want {
			t.Errorf("%s at %v painted R=%d, want %d", tc.name, tc.at, got, tc.want)
		}
	}
}

// The money element draws backinv.bmp and the coin over it, and carries the
// purse as its quantity (SHOP-SCREEN-036 arm a, SHOP-MONEY-048).
func TestTheMoneyElementDrawsTheCoinOverTheItemBackground(t *testing.T) {
	v := ShopScreenView{Chosen: -1, Art: testArt()}
	v.Pack[0] = ShopCell{Money: true, Count: 600}
	pic := ComposeShopScreen(v, image.Point{}, false, nil, false)

	at := ShopPackCellRect(0).Min.Add(image.Pt(40, 40))
	if got := pic.RGBAAt(at.X, at.Y).R; got != 9 {
		t.Errorf("the money cell painted R=%d, want the coin's 9", got)
	}
	// A money cell holds no item, so nothing on it is clickable as one and the
	// hover has nothing to describe.
	if lines, ok := ShopHoverLines(v, at); ok {
		t.Errorf("the money cell answered a hover: %v", lines)
	}
}

// opaqueBounds finds the plaque's own tablet inside its 60x10 canvas, which is
// what puts the digits on it rather than beside it.
func TestOpaqueBoundsIsTheInkAndNotTheCanvas(t *testing.T) {
	pic := image.NewRGBA(image.Rect(0, 0, 60, 10))
	for y := 2; y < 8; y++ {
		for x := 33; x < 58; x++ {
			pic.SetRGBA(x, y, color.RGBA{R: 0x80, A: 0xff})
		}
	}
	if got, want := opaqueBounds(pic), image.Rect(33, 2, 58, 8); got != want {
		t.Errorf("opaqueBounds = %v, want %v", got, want)
	}
	if got := opaqueBounds(image.NewRGBA(image.Rect(0, 0, 4, 4))); !got.Empty() {
		t.Errorf("a wholly transparent picture has ink at %v", got)
	}
	if got := opaqueBounds(nil); !got.Empty() {
		t.Errorf("a nil picture has ink at %v", got)
	}
}

func TestTheWheelNamesTheRegionUnderThePointer(t *testing.T) {
	for _, tc := range []struct {
		at   image.Point
		want ShopWheelRegion
	}{
		{image.Pt(80, 150), ShopWheelShelf},
		{image.Pt(200, 430), ShopWheelPack},
		{image.Pt(200, 340), ShopWheelNone},
		{image.Pt(300, 150), ShopWheelNone},
		{image.Pt(560, 300), ShopWheelNone},
	} {
		if got := ShopWheelAt(tc.at); got != tc.want {
			t.Errorf("ShopWheelAt(%v) = %d, want %d", tc.at, got, tc.want)
		}
	}
}

// The character panel is the sixth region. The shared shell keeps the
// complete 160x240 figure source and mask, with controls over its corners.
func TestTheCharacterPanelUsesTheSharedFigureCropAndControls(t *testing.T) {
	if shopCharRegion != image.Rect(480, 238, 640, 480) {
		t.Errorf("the character panel is %v, want (480,238)-(640,480)", shopCharRegion)
	}
	if shopFigureRect != image.Rect(480, 240, 640, 480) {
		t.Errorf("the figure source box is %v, want (480,240)-(640,480)", shopFigureRect)
	}
	if townCharacterFigure != image.Rect(480, 240, 640, 480) {
		t.Errorf("the painted figure is %v, want (480,240)-(640,480)", townCharacterFigure)
	}
	if shopPickerPrevRect != image.Rect(481, 443, 513, 475) {
		t.Errorf("the previous arrow is %v, want (481,443)-(513,475)", shopPickerPrevRect)
	}
	if shopPickerNextRect != image.Rect(599, 443, 631, 475) {
		t.Errorf("the next arrow is %v, want (599,443)-(631,475)", shopPickerNextRect)
	}

	v := ShopScreenView{Chosen: -1, MemberCount: 2, Figure: solid(240, color.RGBA{B: 77, A: 0xff})}
	pic := ComposeShopScreen(v, image.Point{}, false, nil, false)
	if got := pic.RGBAAt(560, 300).B; got != 77 {
		t.Errorf("the panel's middle painted B=%d, want the figure's 77", got)
	}
	if got := pic.RGBAAt(560, townCharacterFigure.Max.Y-1).B; got != 77 {
		t.Errorf("the figure's last row painted B=%d, want 77", got)
	}
	if got := pic.RGBAAt(500, townCharacterFigure.Max.Y).B; got == 77 {
		t.Errorf("the first cropped row painted the figure: %+v", got)
	}
	// The two rows above the figure are the panel's own furniture, never the
	// figure and never black.
	if got := pic.RGBAAt(560, 238); got.B == 77 || (got.R|got.G|got.B) == 0 {
		t.Errorf("the panel's top row is %+v, want the furniture", got)
	}
}

// A merchant with work to offer is marked, because nothing decoded says how the
// original invites the press and his box is otherwise part of the room picture.
// TestComposeShopScreenDrawsTheMerchantAndTheShelfAnimations is 1009's own
// witness: DIV-020's static merchant picture and DIV-018's four shelf
// animations (their first frame; DIV-127 covers the missing cadence) are
// blit at the addresses SHOP-MERCHANT-046 and SHOP-SHELF-047 give, not left
// as an empty region.
func TestComposeShopScreenDrawsTheMerchantAndTheShelfAnimations(t *testing.T) {
	art := &ShopScreenArt{Merchant: solid(4, color.RGBA{R: 0x11, A: 0xff})}
	for i := range art.ShelfAnim {
		art.ShelfAnim[i] = solid(4, color.RGBA{R: uint8(0x20 + i), A: 0xff})
	}
	pic := ComposeShopScreen(ShopScreenView{Chosen: -1}, image.Point{}, false, nil, false)
	blank := pic.RGBAAt(shopMerchantRect.Min.X, shopMerchantRect.Min.Y)

	pic = ComposeShopScreen(ShopScreenView{Chosen: -1, Art: art}, image.Point{}, false, nil, false)
	corner := shopMerchantRect.Min
	if got := pic.RGBAAt(corner.X, corner.Y); got != (color.RGBA{R: 0x11, A: 0xff}) {
		t.Errorf("merchant corner = %+v, want the art's own picture", got)
	}
	if got := pic.RGBAAt(corner.X, corner.Y); got == blank {
		t.Error("the merchant corner is unchanged with art present")
	}
	for i, r := range shopShelfDrawRects {
		if got := pic.RGBAAt(r.Min.X, r.Min.Y); got != (color.RGBA{R: uint8(0x20 + i), A: 0xff}) {
			t.Errorf("shelf %d corner = %+v, want its own folder's first frame", i, got)
		}
	}
}

// The shop's own doll, its hit test and hover (1005 round 2: shopDollSlotAt,
// shopGridControlAt, ShopHoverLines' new doll arm). The figure source and mask
// use shopFigureRect with no centring (SHOP-FIGURE-042), so a mask pixel at
// figure-local (x,y) sits at shopFigureRect.Min plus (x,y); the visibility
// gate then restricts it to townCharacterFigure and its uncovered pixels.

// shopTestMask is a SlotMask the size of shopFigureRect, slot 4 (1-based,
// zero-based 3) at figure-local (50,100) and nowhere else — one marked pixel is
// enough to tell "the mask answered" from "the mask answered nothing".
func shopTestMask() *SlotMask {
	w, h := shopFigureRect.Dx(), shopFigureRect.Dy()
	m := &SlotMask{W: w, H: h, Slot: make([]uint8, w*h)}
	m.Slot[100*w+50] = 4
	return m
}

// shopFullDollTestView paints a solid synthetic figure over the complete
// visible crop, including the rows where the removed doll-name overlay used
// to cover equipment pixels. Its subject still has a drawable name, so a
// regression that paints the card name over DOLL mode changes those pixels.
func shopFullDollTestView(mask *SlotMask) ShopScreenView {
	font := &text.Font{Glyphs: make([]text.Glyph, 224)}
	font.Glyphs['A'-text.FirstChar] = text.Glyph{Width: 3, Height: 3, Advance: 4,
		Pixels: []text.Pixel{
			{}, {}, {},
			{}, {Level: text.MaxLevel, Painted: true}, {},
			{}, {}, {},
		}}
	figure := image.NewRGBA(image.Rect(0, 0, 160, 240))
	draw.Draw(figure, figure.Bounds(), &image.Uniform{C: color.RGBA{R: 0x03, G: 0x51, B: 0xa7, A: 0xff}}, image.Point{}, draw.Src)
	return ShopScreenView{SlotMask: mask, OrdinaryDollMask: mask, Character: TownCharacterView{
		Subject: PanelSubject{Name: "A"}, HasSubject: true, Figure: figure, MemberCount: 1, Font: font,
	}}
}

func TestShopDollSlotAtReadsTheMask(t *testing.T) {
	mask := shopTestMask()
	v := ShopScreenView{SlotMask: mask}
	marked := shopFigureRect.Min.Add(image.Pt(50, 100))
	if n, ok := shopDollSlotAt(v, mask, marked); !ok || n != 3 {
		t.Errorf("shopDollSlotAt at the marked pixel = (%d,%v), want (3,true)", n, ok)
	}

	// A point inside the figure's own box but off the marked pixel answers no
	// slot: the mask, not the box, decides.
	unmarked := shopFigureRect.Min.Add(image.Pt(50, 50))
	if n, ok := shopDollSlotAt(v, mask, unmarked); ok {
		t.Errorf("shopDollSlotAt over the unmarked ground = (%d,true), want none", n)
	}

	// A point outside shopFigureRect entirely answers no slot even where the
	// mask, read at the translated coordinate, would have held one.
	outside := shopFigureRect.Min.Sub(image.Pt(1, 0))
	if n, ok := shopDollSlotAt(v, mask, outside); ok {
		t.Errorf("shopDollSlotAt outside the figure's box = (%d,true), want none", n)
	}

	// A nil mask (no member shown, or no archive to compose one) answers no
	// slot for every point, including the one a non-nil mask marks.
	if n, ok := shopDollSlotAt(v, nil, marked); ok {
		t.Errorf("shopDollSlotAt with a nil mask = (%d,true), want none", n)
	}
}

// TestShopDollHitAreaMatchesEveryVisibleSyntheticFigurePixel enumerates the
// complete 160x240 source/mask area. The expected answer comes from the
// composed frame: a crop pixel is eligible exactly when the figure's pixel is
// still visible outside the Book control's own rectangle. It does not derive
// expected coverage from the hit helper.
//
// Rect B's own hit rectangle (shopBookCornerHitAt) is excluded here by
// geometry, not by paint (round-2 adversarial review, owner item: the
// near-black control boxes now sitting on shipped art, carried forward to
// round-2's own deletion of the authored Book plaque). Book no longer paints
// an opaque fill anywhere, so its own area shows the same figure pixels the
// doll area does, and only the hit test's own documented control precedence
// — Book wins inside its own rectangle, shopDollAreaAt — still separates
// them. That precedence is asserted here by name, the same way the loop
// below asserts the removed name row by literal points rather than by
// re-deriving it from a former layout helper.
func TestShopDollHitAreaMatchesEveryVisibleSyntheticFigurePixel(t *testing.T) {
	w, h := shopFigureRect.Dx(), shopFigureRect.Dy()
	mask := &SlotMask{W: w, H: h, Slot: make([]uint8, w*h)}
	for i := range mask.Slot {
		mask.Slot[i] = 5
	}
	v := shopFullDollTestView(mask)
	frame := ComposeShopScreen(v, image.Point{}, false, nil, false)
	figure := v.Character.Figure
	for y := shopFigureRect.Min.Y; y < shopFigureRect.Max.Y; y++ {
		for x := shopFigureRect.Min.X; x < shopFigureRect.Max.X; x++ {
			p := image.Pt(x, y)
			q := p.Sub(shopFigureRect.Min)
			bottomControl := p.In(image.Rect(480, 440, 508, 480)) || p.In(image.Rect(481, 443, 513, 475)) ||
				p.In(image.Rect(599, 443, 631, 475)) || p.In(image.Rect(606, 444, 638, 476))
			want := p.In(townCharacterFigure) && !shopBookCornerHitAt(p) && !bottomControl &&
				frame.RGBAAt(p.X, p.Y) == figure.RGBAAt(q.X, q.Y)
			n, got := shopDollSlotAt(v, mask, p)
			if got != want || got && n != 4 {
				t.Fatalf("shopDollSlotAt(%v) = (%d,%v), visible composed figure pixel = %v",
					p, n, got, want)
			}
		}
	}
	for name, p := range map[string]image.Point{
		"previous": shopPickerPrevRect.Min.Add(image.Pt(2, 2)),
		"next":     shopPickerNextRect.Min.Add(image.Pt(2, 2)),
		"border":   image.Pt(shopCharRegion.Min.X, shopCharRegion.Max.Y-1),
	} {
		if n, ok := shopDollSlotAt(v, mask, p); ok {
			t.Errorf("%s at %v = doll slot %d, want the bottom control", name, p, n)
		}
	}
}

// These literal points occupied the old doll-name glyph and a neighbouring
// glyph-cell gap. With the duplicate label removed, both underlying figure
// pixels belong to the same equipment slot and remain interactive.
func TestShopDollFormerNameRowsStayInteractive(t *testing.T) {
	mask := &SlotMask{W: 160, H: 240, Slot: make([]uint8, 160*240)}
	for i := range mask.Slot {
		mask.Slot[i] = 5
	}
	v := shopFullDollTestView(mask)
	for _, p := range []image.Point{image.Pt(559, 424), image.Pt(558, 423)} {
		if n, ok := shopDollSlotAt(v, mask, p); !ok || n != 4 {
			t.Fatalf("former member-name point %v = (%d,%v), want (4,true)", p, n, ok)
		}
	}
}

// shopGridControlAt: the doll wins over the grid it sits inside (there is
// none, since shopFigureRect stands entirely inside the merchant's room
// picture — SHOP-FIGURE-041's own doc), the shelf and pack cells still pass
// through unchanged, and every other family (button, arrow, table cell,
// picker, merchant, shelf pick) is a miss, because none of them is a drag
// surface this story's gestures use.
func TestShopGridControlAtRecognisesTheDollAndTheThreeGrids(t *testing.T) {
	v := ShopScreenView{SlotMask: shopTestMask()}

	marked := shopFigureRect.Min.Add(image.Pt(50, 100))
	if c, ok := shopGridControlAt(v, marked); !ok || c.Kind != ShopControlDoll || c.Index != 3 {
		t.Errorf("shopGridControlAt at the marked doll pixel = %+v,%v, want {ShopControlDoll 3},true", c, ok)
	}

	shelf := ShopShelfCellRect(0).Min.Add(image.Pt(4, 4))
	if c, ok := shopGridControlAt(v, shelf); !ok || c.Kind != ShopControlShelfCell || c.Index != 0 {
		t.Errorf("shopGridControlAt over shelf cell 0 = %+v,%v, want {ShopControlShelfCell 0},true", c, ok)
	}

	pack := ShopPackCellRect(1).Min.Add(image.Pt(4, 4))
	if c, ok := shopGridControlAt(v, pack); !ok || c.Kind != ShopControlPackCell || c.Index != 1 {
		t.Errorf("shopGridControlAt over pack cell 1 = %+v,%v, want {ShopControlPackCell 1},true", c, ok)
	}

	table := ShopTableCellRect(2).Min.Add(image.Pt(4, 4))
	if c, ok := shopGridControlAt(v, table); !ok || c.Kind != ShopControlTableCell || c.Index != 2 {
		t.Errorf("shopGridControlAt over table cell 2 = %+v,%v, want {ShopControlTableCell 2},true", c, ok)
	}

	for _, p := range []image.Point{
		shopButtonRects[0].Min.Add(image.Pt(4, 4)),
		shopPickerPrevRect.Min.Add(image.Pt(4, 4)),
		shopMerchantRect.Min.Add(image.Pt(4, 4)),
		shopShelfPickRects[0].Min.Add(image.Pt(4, 4)),
	} {
		if c, ok := shopGridControlAt(v, p); ok {
			t.Errorf("shopGridControlAt at %v answered %+v, want a miss (not a drag surface)", p, c)
		}
	}
}

// TestShopStatisticsPlacesTheCardInTheCharacterPaneAndDisablesTheDoll proves
// 1022 spec B6 for the shop: the card draws over the doll's own rectangle
// (TownCharacterRegion), so the doll is no longer a drag surface, and the
// pixel loop below requires the pane to be the card exactly.
//
// THE THREE GRIDS ARE NO LONGER PART OF IT (owner, DOLL/STATS hotfix). This
// test required shopScreenControlAt and shopGridControlAt to miss on
// shopDragShelfPoint while Statistics was on, which was consistent with a
// mode that blanked TownContentRegion. The blanking is gone and both must
// hit. TestShopStatisticsChangesOnlyTheCharacterPane carries the composition
// half of the same rule.
//
// BOOK NOW ANSWERS IN BOTH MODES (owner finding 3, round-2 adversarial
// return). The authored "Book" plaque this project drew before the six
// corner controls existed sat on the statistics card's own name row and was
// suppressed there (1027 B3); it is deleted, and rect B's own decoded art
// (DrawTownCharacterRegion) draws and answers in Statistics and Doll mode
// alike (characterPaneCornerDrawn's Book gate, `s&3 != 0`, does not read
// Statistics at all).
func TestShopStatisticsPlacesTheCardInTheCharacterPaneAndDisablesTheDoll(t *testing.T) {
	v := ShopScreenView{SlotMask: shopTestMask(), Character: TownCharacterView{
		HasSubject: true, Statistics: true, MemberCount: 2,
		Subject: panelWideSubjectFixture(), Font: panelWideFont()}, Font: panelWideFont()}

	if c, ok := shopScreenControlAt(v, shopDragShelfPoint); !ok || c.Kind != ShopControlShelfCell {
		t.Fatalf("shelf control during statistics = %+v,%v, want shelf cell 0", c, ok)
	}
	if c, ok := shopGridControlAt(v, shopDragShelfPoint); !ok || c.Kind != ShopControlShelfCell {
		t.Fatalf("shelf drag surface during statistics = %+v,%v, want shelf cell 0", c, ok)
	}
	if c, ok := shopScreenControlAt(v, shopButtonRects[0].Min.Add(image.Pt(4, 4))); !ok || c.Kind != ShopControlButton {
		t.Fatalf("persistent shop button = %+v,%v", c, ok)
	}
	bookPoint := TownCharacterRegion.Min.Add(image.Pt(4, 4))
	if c, ok := shopScreenControlAt(v, bookPoint); !ok || c.Kind != ShopControlBook {
		t.Fatalf("Book over the statistics card = %+v,%v, want a hit (rect B draws in both modes)", c, ok)
	}
	doll := v
	doll.Character.Statistics = false
	if c, ok := shopScreenControlAt(doll, bookPoint); !ok || c.Kind != ShopControlBook {
		t.Fatalf("Book in doll mode = %+v,%v, want a hit", c, ok)
	}
	if c, ok := shopGridControlAt(v, shopDragDollPoint); ok {
		t.Fatalf("doll drag surface answered %+v during statistics, want a miss (the card replaces it)", c)
	}
	if lines, ok := ShopHoverLines(v, shopDragDollPoint); ok {
		t.Fatalf("doll hover answered %v during statistics, want a miss (the card replaces it)", lines)
	}

	frame := ComposeShopScreen(v, image.Point{}, false, nil, false)
	refLayout := CompactPanelLayout(nil)
	card := RenderCharacterPanel(refLayout, v.Character.Font, v.Character.Subject)
	// Excluded: TownCharacterPersistentControls(), every corner drawn for
	// CharacterPaneShop (its own union of hit and art rectangles), which
	// DrawTownCharacterRegion paints after the card. The authored Book
	// plaque used to need a second, disjoint rectangle here; deleting it
	// (owner finding 3, round-2 adversarial return) removes that second
	// exclusion rather than replacing it, since rect B's own art was already
	// covered by TownCharacterPersistentControls() before this round.
	//
	// No separate doll-name rectangle exists: the statistics card owns the
	// subject name as its first centred row, and every other non-control
	// rectangle must match the card exactly.
	covers := TownCharacterPersistentControls()
	for y := 0; y < card.Bounds().Dy(); y++ {
		for x := 0; x < card.Bounds().Dx(); x++ {
			p := TownCharacterRegion.Min.Add(image.Pt(x, y))
			covered := false
			for _, r := range covers {
				if p.In(r) {
					covered = true
					break
				}
			}
			if covered {
				continue
			}
			if got, want := frame.RGBAAt(p.X, p.Y), card.RGBAAt(x, y); got != want {
				t.Fatalf("character pane statistics pixel %v = %#v, want native card %#v", p, got, want)
			}
		}
	}
}

// ShopHoverLines' new doll arm (1005 round 2): a point over a marked doll
// pixel answers SlotInfo[slot] before the three grids are ever tested, and a
// slot with no recorded lines answers nothing rather than falling through to
// whatever the grid test underneath it would have said.
func TestShopHoverLinesAnswersTheDollSlotFirst(t *testing.T) {
	v := ShopScreenView{SlotMask: shopTestMask()}
	v.SlotInfo[3] = []string{"a helm", "Armour 2"}

	marked := shopFigureRect.Min.Add(image.Pt(50, 100))
	lines, ok := ShopHoverLines(v, marked)
	if !ok || len(lines) != 2 || lines[0] != "a helm" {
		t.Errorf("ShopHoverLines at the marked doll pixel = %v,%v, want SlotInfo[3]", lines, ok)
	}

	// The same mask, an unmarked pixel of the figure's own box: no slot, so
	// no lines — and, critically, NOT the grid's answer for that point,
	// because there is none (shopFigureRect stands over no grid cell).
	unmarked := shopFigureRect.Min.Add(image.Pt(50, 50))
	if lines, ok := ShopHoverLines(v, unmarked); ok {
		t.Errorf("ShopHoverLines over the unmarked ground = %v,true, want none", lines)
	}

	// A marked pixel whose slot carries no recorded lines answers nothing,
	// not a slice of zero meaning.
	v2 := ShopScreenView{SlotMask: shopTestMask()}
	if lines, ok := ShopHoverLines(v2, marked); ok {
		t.Errorf("ShopHoverLines at a slot with no SlotInfo = %v,true, want none", lines)
	}
}

func TestShopHoverLinesMissesTheDollDuringStatistics(t *testing.T) {
	v := ShopScreenView{SlotMask: shopTestMask(), Character: TownCharacterView{Statistics: true}}
	v.SlotInfo[3] = []string{"a helm", "Armour 2"}

	marked := shopFigureRect.Min.Add(image.Pt(50, 100))
	if lines, ok := ShopHoverLines(v, marked); ok {
		t.Errorf("ShopHoverLines at the marked doll pixel during Statistics = %v,true, want none (the card occupies this rect, not the doll)", lines)
	}
}

// The tip widget (1011 spec; SHOP-TIP-045; DIV-132, DIV-133).

// shopTipTestFont is a 224-record font whose every glyph — including record 0,
// the space — is the SAME solid, painted 6x12 cell with a 24px advance, so a
// string's measured width is predictable from its length alone and every
// drawn byte leaves ink a pixel comparison can see.
func shopTipTestFont() *text.Font {
	f := &text.Font{Glyphs: make([]text.Glyph, 224)}
	for i := range f.Glyphs {
		pixels := make([]text.Pixel, 6*12)
		for p := range pixels {
			pixels[p] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		f.Glyphs[i] = text.Glyph{Width: 6, Height: 12, Advance: 24, Pixels: pixels}
	}
	return f
}

// wrapShopTip breaks on whitespace, greedily, measuring each candidate line
// against the font exactly as the widths it is asked to fit are measured —
// the same font, not a hand-derived pixel count, so this cannot pass by
// agreeing with an arithmetic mistake shared with the code under test.
func TestWrapShopTipBreaksGreedilyOnWhitespace(t *testing.T) {
	font := shopTipTestFont()
	twoWords, _ := font.Measure("ab cd")
	threeWords, _ := font.Measure("ab cd ef")
	oneWord, _ := font.Measure("ab")

	if got := wrapShopTip(font, "ab cd ef", threeWords); len(got) != 1 || got[0] != "ab cd ef" {
		t.Fatalf("width = all three words' own measure: wrapShopTip = %v, want one line", got)
	}
	if got := wrapShopTip(font, "ab cd ef", twoWords); len(got) != 2 || got[0] != "ab cd" || got[1] != "ef" {
		t.Fatalf(`width = the first two words' own measure: wrapShopTip = %v, want ["ab cd" "ef"]`, got)
	}
	if got := wrapShopTip(font, "ab cd ef", oneWord); len(got) != 3 || got[2] != "ef" {
		t.Fatalf("width = one word's own measure: wrapShopTip = %v, want three lines", got)
	}
}

// A word wider than the rect on its own is kept whole: this build has no
// hyphenation rule to reproduce or invent.
func TestWrapShopTipNeverSplitsAWordWiderThanTheRect(t *testing.T) {
	font := shopTipTestFont()
	if got := wrapShopTip(font, "abcdefghij", 1); len(got) != 1 || got[0] != "abcdefghij" {
		t.Fatalf("a lone word over width = %v, want it kept whole on one line", got)
	}
}

// The file's own CRLF ends a line even where both halves would fit together —
// a paragraph break is authored as the file's structure, not folded into the
// same whitespace test a plain space is.
func TestWrapShopTipTreatsCRLFAsAParagraphBreak(t *testing.T) {
	font := shopTipTestFont()
	wide, _ := font.Measure("ab cd")
	got := wrapShopTip(font, "ab\r\ncd", wide)
	if len(got) != 2 || got[0] != "ab" || got[1] != "cd" {
		t.Fatalf(`wrapShopTip("ab\r\ncd") = %v, want ["ab" "cd"]`, got)
	}
}

// longShopTip is n distinct words, enough that shopTipTestFont's own 24px
// advance wraps them well past ShopTipRect()'s own height at the rect's
// width.
func longShopTip(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += " "
		}
		s += fmt.Sprintf("word%d", i)
	}
	return s
}

// ShopTipRect NEVER REACHES THE MESSAGE STRIP. DIV-018 named this exact
// collision: the tip widget's decoded rectangle (shopTipRect) carries this
// build's own message line at its bottom. ShopTipRect() stops at
// shopMessageRect's own top rather than at shopTipRect's own bottom.
// ShopTipRect NOW EXTENDS PAST shopMessageRect, ON PURPOSE (DIV-162, this
// story's own landing): its own 136-row SHOP-TIP-045 height is kept whole as
// text capacity and tipPanelChromeH is added below it for the close/toggle
// row, rather than carved out of those 136 rows. Carving the row out left
// only 4 of the shipped EN tip's own 7 wrapped lines fitting, a regression
// against DIV-133's own "seven lines, no spare row" measurement of the build
// before this story. The overlap with shopMessageRect is intentional: see
// TestTipPanelOccludesTheMessageStripWhileShowing below for what that means
// for the strip itself.
func TestShopTipRectExtendsPastTheMessageStrip(t *testing.T) {
	if got, want := ShopTipRect().Max.Y, shopTipRect.Max.Y+tipPanelChromeH; got != want {
		t.Fatalf("ShopTipRect().Max.Y = %d, want shopTipRect.Max.Y + tipPanelChromeH = %d", got, want)
	}
	if !ShopTipRect().Overlaps(shopMessageRect) {
		t.Fatal("ShopTipRect() does not overlap shopMessageRect, want it to (DIV-162)")
	}
}

// THE MESSAGE ANSWER TO THE LAST PRESS IS OCCLUDED WHILE THE PANEL SHOWS,
// AND UNAFFECTED ONCE IT DOES NOT (DIV-162, amending DIV-133's own
// "unaffected" reading for the bare-text widget this story replaces). The
// panel composes last and opaque (ComposeShopScreen's own doc) and its own
// rect now overlaps shopMessageRect (the test above), so a showing panel
// paints over the strip for as long as it is open; closing or suppressing it
// (Showing() false) restores the strip pixel-identical to the build before
// this story, which the second half of this test checks.
func TestTipPanelOccludesTheMessageStripWhileShowing(t *testing.T) {
	font := shopTipTestFont()
	art := &TipPanelArt{Fill: image.NewUniform(shopScreenFill), Border: image.NewUniform(shopScreenFill)}
	render := func(tipText, msgText string) *image.RGBA {
		v := ShopScreenView{Chosen: -1, Font: font, Msg: msgText}
		if tipText != "" {
			v.TipPanel = TipPanelView{Rect: ShopTipRect(), Text: tipText, Art: art, Font: font}
		}
		return ComposeShopScreen(v, image.Point{}, false, nil, false)
	}

	both := render(longShopTip(60), "on the table")
	msgOnly := render("", "on the table")

	diff := 0
	for y := shopMessageRect.Min.Y; y < shopMessageRect.Max.Y; y++ {
		for x := shopMessageRect.Min.X; x < shopMessageRect.Max.X; x++ {
			if both.RGBAAt(x, y) != msgOnly.RGBAAt(x, y) {
				diff++
			}
		}
	}
	if diff == 0 {
		t.Fatal("message strip is pixel-identical with a showing tip panel above it, want it occluded (DIV-162)")
	}

	// With no tip text (Showing() false, TipPanelView{}), the strip must be
	// pixel-identical to a render with no panel at all — the affordance this
	// story does not otherwise touch.
	notShowing := render("", "on the table")
	for y := shopMessageRect.Min.Y; y < shopMessageRect.Max.Y; y++ {
		for x := shopMessageRect.Min.X; x < shopMessageRect.Max.X; x++ {
			if got, want := notShowing.RGBAAt(x, y), msgOnly.RGBAAt(x, y); got != want {
				t.Fatalf("message strip differs at (%d,%d) with no tip panel showing: %+v, want %+v", x, y, got, want)
			}
		}
	}
}

func TestShopStatisticsChangesOnlyTheCharacterPane(t *testing.T) {
	base := ShopScreenView{SlotMask: shopTestMask(), Book: true, Character: TownCharacterView{
		HasSubject: true, MemberCount: 2,
		Subject: panelWideSubjectFixture(), Font: panelWideFont()}, Font: panelWideFont()}
	for i := range base.Shelf {
		base.Shelf[i] = ShopCell{Back: ShopBackAffordable, Count: 1, Price: 5, Info: []string{"shelf item"}}
	}
	for i := range base.Pack {
		base.Pack[i] = ShopCell{Back: ShopBackItem, Count: 2, Price: 7, Mine: true, Info: []string{"pack item"}}
	}
	base.Spells = []SpellEntry{{ID: 1, Name: "one"}, {ID: 2, Name: "two"}, {ID: 3, Name: "three"}}

	stats := base
	stats.Character.Statistics = true
	dollFrame := ComposeShopScreen(base, image.Point{}, false, nil, false)
	statsFrame := ComposeShopScreen(stats, image.Point{}, false, nil, false)

	diff, first := 0, image.Point{-1, -1}
	b := dollFrame.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			p := image.Pt(x, y)
			if p.In(TownCharacterRegion) {
				continue
			}
			if dollFrame.RGBAAt(x, y) != statsFrame.RGBAAt(x, y) {
				diff++
				if first.X < 0 {
					first = p
				}
			}
		}
	}
	if diff != 0 {
		t.Fatalf("the DOLL/STATS toggle changed %d pixels outside TownCharacterRegion, first at %v; it owns that rectangle and nothing else", diff, first)
	}

	// The pane itself must differ, or the comparison above proves nothing:
	// a compositor that ignored Statistics entirely would pass it.
	paneDiff := 0
	for y := TownCharacterRegion.Min.Y; y < TownCharacterRegion.Max.Y; y++ {
		for x := TownCharacterRegion.Min.X; x < TownCharacterRegion.Max.X; x++ {
			if dollFrame.RGBAAt(x, y) != statsFrame.RGBAAt(x, y) {
				paneDiff++
			}
		}
	}
	if paneDiff == 0 {
		t.Fatal("the two modes painted TownCharacterRegion identically; the toggle changed nothing at all")
	}
}

// TestTheShopBookIsTheSharedIconBookWithInspectionOnlyCells is the owner's
// shop-book hotfix at the presentation boundary. The shop uses the same icon
// cells as the mission book, exposes the same information popup, and captures
// both occupied and empty book cells so the hidden trade table cannot act.
func TestTheShopBookIsTheSharedIconBookWithInspectionOnlyCells(t *testing.T) {
	icon := image.NewRGBA(image.Rect(0, 0, 36, 36))
	draw.Draw(icon, icon.Bounds(), &image.Uniform{C: color.RGBA{R: 0xa4, B: 0xd8, A: 0xff}}, image.Point{}, draw.Src)
	v := ShopScreenView{Book: true, Spells: []SpellEntry{{
		ID: 7, Name: "Fire Arrow", Icon: icon, Info: []string{"Fire Arrow", "cost 4"},
	}}}
	cell := bookCellRects(shopTableRegion, shopSpellColumns())[0]
	point := cell.Min.Add(image.Pt(4, 4))

	lines, ok := ShopHoverLines(v, point)
	if !ok || fmt.Sprint(lines) != "[Fire Arrow cost 4]" {
		t.Fatalf("spell hover = %v,%v, want the entry's shared popup", lines, ok)
	}
	if c, ok := shopScreenControlAt(v, point); !ok || c.Kind != ShopControlSpell || c.Index != 0 {
		t.Fatalf("occupied spell cell = %#v,%v, want inspection-only spell 0", c, ok)
	}
	if c, ok := shopGridControlAt(v, point); ok {
		t.Fatalf("opened book leaked a drag target through to the trade table: %#v", c)
	}
	emptyPoint := bookCellRects(shopTableRegion, shopSpellColumns())[1].Min.Add(image.Pt(4, 4))
	if c, ok := shopScreenControlAt(v, emptyPoint); !ok || c.Kind != ShopControlSpell || c.Index != -1 {
		t.Fatalf("empty spell cell = %#v,%v, want captured inspection no-op", c, ok)
	}

	a := ComposeShopScreen(v, image.Point{}, false, nil, false)
	renamed := v
	renamed.Spells = append([]SpellEntry(nil), v.Spells...)
	renamed.Spells[0].Name = "A name must not replace the icon"
	b := ComposeShopScreen(renamed, image.Point{}, false, nil, false)
	if !imagesEqual(a, b) {
		t.Fatal("changing an icon-backed spell name changed the painted shop book; a text row replaced or overlaid the icon")
	}
	found := false
	for y := cell.Min.Y; y < cell.Max.Y && !found; y++ {
		for x := cell.Min.X; x < cell.Max.X; x++ {
			if a.RGBAAt(x, y) == (color.RGBA{R: 0xa4, B: 0xd8, A: 0xff}) {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("the shop's first book cell painted none of the supplied spell icon")
	}
}
