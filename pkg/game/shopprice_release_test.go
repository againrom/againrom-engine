package game

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"strconv"
	"testing"

	"againrom/pkg/formats/spr16"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The price figure on the town shop's grids (ITEM-PRICETAG-144,
// MISSION-MSGLINE-056), checked on the installed art and font against an oracle
// that shares no code with the shop painter, the text package's placement or
// the font loader: font2's two nodes are decoded here and the claim's numbers
// are literals.

// priceOracle draws the grid's price figure from one font's two nodes.
type priceOracle struct {
	frames   []spr16.FrameG
	advances []int
}

func newPriceOracle(t *testing.T, f *FrontEnd, base string) priceOracle {
	t.Helper()
	atlas, err := f.Archives.Containers.ReadFile(FontAtlasPath(base))
	if err != nil {
		t.Fatal(err)
	}
	sidecar, err := f.Archives.Containers.ReadFile(FontAdvancePath(base))
	if err != nil {
		t.Fatal(err)
	}
	o := priceOracle{}
	if o.frames, err = spr16.DecodeG(atlas); err != nil {
		t.Fatal(err)
	}
	if o.advances, err = spr16.Advances(sidecar); err != nil {
		t.Fatal(err)
	}
	if len(o.frames) != len(o.advances) {
		t.Fatalf("%s: %d glyph records, %d advances", base, len(o.frames), len(o.advances))
	}
	return o
}

// priceVariant states one way of drawing a figure differently from the claim:
// how far it stands from the claimed placement, whether its shadow is drawn and
// the flat colour the ramp's top entry takes. The zero variant is the claim's.
type priceVariant struct {
	dx, dy   int
	noShadow bool
	ink      *color.RGBA
}

// groupedFigure is a number grouped by threes with commas, its sign in front
// and no comma after it.
func groupedFigure(n int32) string {
	sign, s := "", strconv.Itoa(int(n))
	if n < 0 {
		sign, s = "-", s[1:]
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return sign + s
}

// paint draws the figure the claim gives for price in the cell: the text's pen
// ends 6 pixels left of the cell's right edge and its glyph cells start one
// pixel below the cell's top. The pen advances a glyph's own advance and the
// letter spacing of 2 (SPR16A-FONT-018) after every glyph. A shadow of the same
// text one pixel right and down is painted first in the flat colour 8 8 8, then
// each glyph pixel of level k in the ramp entry (185k/15, 159k/15, 73k/15).
func (o priceOracle) paint(pic *image.RGBA, cell image.Rectangle, price int32, v priceVariant) {
	s := groupedFigure(price)
	const spacing = 2
	width := 0
	for i := 0; i < len(s); i++ {
		width += o.advances[int(s[i])-32] + spacing
	}
	x0, y0 := cell.Max.X-6-width+v.dx, cell.Min.Y+1+v.dy
	pass := func(offset int, colour func(level uint8) color.RGBA) {
		pen := 0
		for i := 0; i < len(s); i++ {
			g := o.frames[int(s[i])-32]
			for row := 0; row < g.Height; row++ {
				for col := 0; col < g.Width; col++ {
					if p := g.Pixels[row*g.Width+col]; p.Painted {
						pic.SetRGBA(x0+pen+col+offset, y0+row+offset, colour(p.Value))
					}
				}
			}
			pen += o.advances[int(s[i])-32] + spacing
		}
	}
	if !v.noShadow {
		pass(1, func(uint8) color.RGBA { return color.RGBA{R: 8, G: 8, B: 8, A: 255} })
	}
	pass(0, func(level uint8) color.RGBA {
		k := int(level)
		if v.ink != nil {
			return color.RGBA{R: uint8(int(v.ink.R) * k / 15), G: uint8(int(v.ink.G) * k / 15), B: uint8(int(v.ink.B) * k / 15), A: 255}
		}
		return color.RGBA{R: uint8(185 * k / 15), G: uint8(159 * k / 15), B: uint8(73 * k / 15), A: 255}
	})
}

// priceCellsPerScreen is how many prices the population check places on one
// composed screen: one per place of the table and the pack, and the shelf's
// first five.
const priceCellsPerScreen = 5

// priceStrip is the rows of a cell that hold its plaque and figure, with two
// spare rows below the shadow.
func priceStrip(cell image.Rectangle) image.Rectangle {
	return image.Rect(cell.Min.X, cell.Min.Y, cell.Max.X, cell.Min.Y+14)
}

// priceSlot is one cell of a shop grid on the composed screen.
type priceSlot struct {
	grid  string
	index int
	rect  image.Rectangle
	cell  ui.ShopCell
}

// priced is whether the cell draws a figure: every occupied cell but the money
// element does, whatever its price's sign (ITEM-PRICETAG-144).
func (s priceSlot) priced() bool { return s.cell.Occupied() && !s.cell.Money }

func priceSlots(v ui.ShopScreenView) []priceSlot {
	var out []priceSlot
	for i, c := range v.Shelf {
		out = append(out, priceSlot{"shelf", i, ui.ShopShelfCellRect(i), c})
	}
	for i, c := range v.Table {
		out = append(out, priceSlot{"table", i, ui.ShopTableCellRect(i), c})
	}
	for i, c := range v.Pack {
		out = append(out, priceSlot{"pack", i, ui.ShopPackCellRect(i), c})
	}
	return out
}

// priceStripDifferences counts the pixels, over the strip of every cell, at which
// got differs from bare (the same screen composed without any text) with the
// oracle's figure painted on each priced cell, and describes the first.
func priceStripDifferences(got, bare *image.RGBA, slots []priceSlot, o priceOracle, v priceVariant) (int, string) {
	want := image.NewRGBA(bare.Bounds())
	copy(want.Pix, bare.Pix)
	for _, s := range slots {
		if s.priced() {
			o.paint(want, s.rect, s.cell.Price, v)
		}
	}
	bad, first := 0, ""
	for _, s := range slots {
		r := priceStrip(s.rect)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if got.RGBAAt(x, y) != want.RGBAAt(x, y) {
					if bad == 0 {
						first = fmt.Sprintf("%s %d at (%d,%d): got %+v, want %+v", s.grid, s.index, x, y, got.RGBAAt(x, y), want.RGBAAt(x, y))
					}
					bad++
				}
			}
		}
	}
	return bad, first
}

// priceLossControls are the ways of drawing a figure that the strip check must
// tell apart from the claim's: a pixel aside in each direction, no shadow and
// the shop's own text colour for the ramp.
func priceLossControls() map[string]priceVariant {
	cream := color.RGBA{R: 0xe8, G: 0xdc, B: 0xc0, A: 0xff}
	return map[string]priceVariant{
		"one pixel left":  {dx: -1},
		"one pixel right": {dx: 1},
		"one pixel up":    {dy: -1},
		"one pixel down":  {dy: 1},
		"no shadow":       {noShadow: true},
		"the shop's ink":  {ink: &cream},
	}
}

// The town shop draws every grid's priced cell as the original does: the
// grouped figure in font2, its pen ending 6 pixels left of the cell's right
// edge, its glyph cells one pixel below the cell's top, in the ramp (185k/15,
// 159k/15, 73k/15) over a flat 8 8 8 shadow one pixel right and down, on the
// plaque the digit count and the side of the deal select. Through App input the
// witness picks each of the four shelves, puts a shelf item on the table and
// takes a worn item off into the pack; each frame the App composes then equals
// the same screen without text with the oracle's figures painted on every cell
// strip. Then every digit count from one to seven, on both plaque families and
// in all three grids, is composed on the installed art and checked the same way.
// Every check fails for the same figures in font1, a pixel aside, without their
// shadow and in the shop's text colour.
func TestReleaseShopGridPriceFigureIsRightAlignedWithAShadow(t *testing.T) {
	app, s := releaseShopApp(t)
	f := frontOf(s)
	s.CloseTip()
	oracle := newPriceOracle(t, f, "font2")
	wrongFont := newPriceOracle(t, f, "font1")

	found := map[string]image.Point{}
	point := func(kind string, index int) image.Point {
		t.Helper()
		key := fmt.Sprintf("%s %d", kind, index)
		if p, ok := found[key]; ok {
			return p
		}
		x, y, err := app.HeadlessShopPoint(kind, index)
		if err != nil {
			t.Fatal(err)
		}
		found[key] = image.Pt(x, y)
		return found[key]
	}
	click := func(kind string, index int) {
		t.Helper()
		p := point(kind, index)
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	ix, iy, err := app.HeadlessShopIdlePoint()
	if err != nil {
		t.Fatal(err)
	}

	// verify holds got to the claim: it differs from the oracle's screen nowhere,
	// and from each loss control's screen somewhere.
	verify := func(name string, got *image.RGBA, view ui.ShopScreenView) int {
		t.Helper()
		bareView := view
		bareView.Font, bareView.PriceFont, bareView.Character.Font = nil, nil, nil
		bare := ui.ComposeShopScreen(bareView, image.Point{}, false, nil, false)
		if got.Bounds() != bare.Bounds() {
			t.Fatalf("%s: the screen is %v, want %v", name, got.Bounds(), bare.Bounds())
		}
		slots := priceSlots(view)
		if bad, first := priceStripDifferences(got, bare, slots, oracle, priceVariant{}); bad != 0 {
			t.Fatalf("%s: the screen differs from the claimed figures at %d pixel(s), first %s", name, bad, first)
		}
		for variant, v := range priceLossControls() {
			if bad, _ := priceStripDifferences(got, bare, slots, oracle, v); bad == 0 {
				t.Errorf("%s: the check cannot tell the claimed figures from %s", name, variant)
			}
		}
		if bad, _ := priceStripDifferences(got, bare, slots, wrongFont, priceVariant{}); bad == 0 {
			t.Errorf("%s: the check cannot tell font2 from font1", name)
		}
		priced := 0
		for _, slot := range slots {
			if slot.priced() {
				priced++
			}
		}
		return priced
	}
	pricedIn := map[string]int{}
	check := func(state string, grid string) {
		t.Helper()
		if err := app.HeadlessPointer("hover", ix, iy); err != nil {
			t.Fatal(err)
		}
		if tip, _ := app.HeadlessTooltip(); tip.Visible {
			t.Fatalf("%s: a tooltip stands over the screen", state)
		}
		frame, _, err := app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		view := s.ShopScreen()
		if verify(state, frame, view) == 0 {
			t.Fatalf("%s: no priced cell on the screen", state)
		}
		for _, slot := range priceSlots(view) {
			if slot.grid == grid && slot.priced() {
				pricedIn[grid]++
			}
		}
	}

	// The four shelves, as a player picks them in the room.
	for pick := range shopRoomShelves {
		click("shelf_pick", pick)
		if s.shopChosen != pick {
			t.Fatalf("shelf pick %d chose %d", pick, s.shopChosen)
		}
		check(fmt.Sprintf("shelf %d", pick), "shelf")
	}

	// A click on a shelf cell puts the item on the merchant's table.
	click("shelf_pick", 0)
	shelfView := s.ShopScreen()
	c := slices.IndexFunc(shelfView.Shelf[:], func(cell ui.ShopCell) bool { return cell.Occupied() && cell.Price > 0 })
	if c < 0 {
		t.Fatal("shelf 0 has no priced cell in view")
	}
	click("shelf", c)
	if table := f.Shop.Table(); len(table) == 0 || table[0].Mine || table[0].Price <= 0 {
		t.Fatalf("table after the shelf click %+v, want a merchant item at a positive price", table)
	}
	check("table", "table")

	// A tap on the doll takes a worn item off into the pack.
	member := s.shopMemberIndex()
	slot := 0
	for n := 1; n <= sim.EquipSlots && slot == 0; n++ {
		if item, ok := s.shopEquippedItem(member, n); ok && !item.Empty() && item.Price > 0 {
			slot = n
		}
	}
	if slot == 0 {
		t.Fatal("no worn item with a positive price could be taken off")
	}
	click("doll", slot)
	check("pack", "pack")
	for _, grid := range []string{"shelf", "table", "pack"} {
		if pricedIn[grid] == 0 {
			t.Errorf("no priced %s cell was checked through the App", grid)
		}
	}

	// Every digit count on both plaque families, in all three grids, on the
	// installed plaques and font: five prices to a screen, each placed in the
	// same place of every grid.
	art, font := f.shopArt(), f.tipFont()
	if art == nil || font == nil {
		t.Fatal("shop art or price font did not resolve")
	}
	prices := []int32{1, 9, 10, 99, 100, 999, 1000, 9999, 10000, 99999, 100000, 999999, 1000000, 1999998, 9999999}
	for _, mine := range []bool{false, true} {
		for from := 0; from < len(prices); from += priceCellsPerScreen {
			v := ui.ShopScreenView{Chosen: -1, Art: art, PriceFont: font}
			for i, price := range prices[from : from+priceCellsPerScreen] {
				cell := ui.ShopCell{Back: ui.ShopBackItem, Count: 1, Price: price, Mine: mine}
				v.Shelf[i], v.Table[i], v.Pack[i] = cell, cell, cell
			}
			name := fmt.Sprintf("population of prices %v, player's plaques %v", prices[from:from+priceCellsPerScreen], mine)
			verify(name, ui.ComposeShopScreen(v, image.Point{}, false, nil, false), v)
		}
	}
	t.Logf("priced cells checked through the App: %v", pricedIn)
}
