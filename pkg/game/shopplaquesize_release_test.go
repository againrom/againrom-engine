package game

import (
	"fmt"
	"image"
	"image/draw"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The price plaque of a shop cell on the installed art and font
// (SHOP-SCREEN-037, ITEM-PRICETAG-144): on both sides of the deal it is the
// plaque of the stored unit price's decimal digit count, and the figure printed
// on it is the stored price on the merchant's side and (price+1)/2 on the
// player's.

// plaqueSizeStock is one item the witness stocks: the stored price it carries,
// the equipment code and kind it is drawn as, and whether that price is the one
// the installed tables give the code.
type plaqueSizeStock struct {
	price     int32
	code      data.ItemCode
	kind      uint8
	installed bool
}

// plaqueSizePopulation is what the witness stocks, cheapest first. The real
// items are the shop generator's own weapon, shield and armour candidates on the
// installed tables at an unbounded value window: every candidate priced 10 to 18
// or 100 to 198, the cheapest and dearest one of each higher decade below a
// million whose half has a digit fewer than the price, and the cheapest and
// dearest one priced below 10. The prices at the edge of every decade, k-1, k,
// 2k-2 and 2k-1 from k = 10 to k = 1000000, are stocked too: the halved figure
// and the stored price choose different plaques from k to 2k-2 and the same one
// at k-1 and 2k-1. A price no candidate carries is stocked on the first
// candidate's code, as an enchanted item carries any price.
func plaqueSizePopulation(t *testing.T, f *FrontEnd) []plaqueSizeStock {
	t.Helper()
	const unbounded = int32(1<<31 - 1)
	pool := append(shopWeaponPool(f.Table, unbounded), shopArmourPool(f.Table, unbounded)...)
	first := map[int32]data.ShopCandidate{}
	var prices []int32
	for _, c := range pool {
		if _, ok := first[c.Price]; !ok && c.Price > 0 {
			first[c.Price] = c
			prices = append(prices, c.Price)
		}
	}
	if len(prices) == 0 {
		t.Fatal("the installed tables give the shop no priced weapon, shield or armour")
	}
	slices.Sort(prices)

	chosen := map[int32]plaqueSizeStock{}
	real := func(p int32) {
		c := first[p]
		chosen[p] = plaqueSizeStock{price: p, code: c.Code, kind: c.ItemKind, installed: true}
	}
	edges := func(lo, hi int32) {
		var in []int32
		for _, p := range prices {
			if p >= lo && p <= hi {
				in = append(in, p)
			}
		}
		if len(in) > 0 {
			real(in[0])
			real(in[len(in)-1])
		}
	}
	for _, p := range prices {
		if p >= 10 && p <= 18 || p >= 100 && p <= 198 {
			real(p)
		}
	}
	edges(1, 9)
	for k := int32(1000); k <= 100000; k *= 10 {
		edges(k, 2*k-2)
	}
	fixture := first[prices[0]]
	for k := int32(10); k <= 1000000; k *= 10 {
		for _, p := range []int32{k - 1, k, 2*k - 2, 2*k - 1} {
			if _, ok := chosen[p]; ok {
				continue
			}
			if _, ok := first[p]; ok {
				real(p)
				continue
			}
			chosen[p] = plaqueSizeStock{price: p, code: fixture.Code, kind: fixture.ItemKind}
		}
	}
	out := make([]plaqueSizeStock, 0, len(chosen))
	for _, it := range chosen {
		out = append(out, it)
	}
	slices.SortFunc(out, func(a, b plaqueSizeStock) int { return int(a.price) - int(b.price) })
	return out
}

// plaqueSizeCell is one occupied cell the witness holds to the claim: where it
// stands, which side it is on and the stored price of its item.
type plaqueSizeCell struct {
	grid   string // shelf, table or pack
	index  int
	mine   bool
	stored int32
}

// figure is the number the claim has the cell print: the stored price, or on
// the player's side (price+1)/2.
func (c plaqueSizeCell) figure() int32 {
	if c.mine {
		return (c.stored + 1) / 2
	}
	return c.stored
}

// plaqueSizeIndex is the plaque a number takes: its decimal digit count minus
// one, at most the seventh.
func plaqueSizeIndex(n int32) int {
	d := 0
	for ; n >= 10; n /= 10 {
		d++
	}
	return min(d, 6)
}

// plaqueSizeVariant states one way of painting a cell differently from the claim.
// The zero variant is the claim.
type plaqueSizeVariant struct {
	fromFigure bool // the plaque of the printed figure's digit count
	other      bool // the other side's plaque
	shift      int  // added to the plaque index, within the seven plaques
	noPlaque   bool
	noFigure   bool
	font       *priceOracle
	move       priceVariant
}

// paintPlaqueSize paints the claim's plaque and figure for c on pic, which holds
// the screen composed without either: the plaque of the cell's side at its
// stored price's digit count, right-aligned to the cell's right edge one pixel
// below its top, then the figure an oracle paints from font2's nodes.
func paintPlaqueSize(pic *image.RGBA, art *ui.ShopScreenArt, oracle priceOracle, r image.Rectangle, c plaqueSizeCell, o plaqueSizeVariant) {
	if !o.noPlaque {
		side := 0
		if c.mine != o.other {
			side = 1
		}
		index := plaqueSizeIndex(c.stored)
		if o.fromFigure {
			index = plaqueSizeIndex(c.figure())
		}
		plaque := art.Plaque[side][min(max(index+o.shift, 0), 6)]
		at := image.Pt(r.Max.X-plaque.Bounds().Dx(), r.Min.Y+1)
		draw.Draw(pic, image.Rectangle{Min: at, Max: at.Add(plaque.Bounds().Size())}, plaque, plaque.Bounds().Min, draw.Over)
	}
	if !o.noFigure {
		if o.font != nil {
			oracle = *o.font
		}
		oracle.paint(pic, r, c.figure(), o.move)
	}
}

// Items priced 10 to 18 and 100 to 198, and at the edges of every other decade,
// draw the plaque of their stored price in every grid, through App input on the
// installed art. The merchant's armour shelf is stocked with the population
// (the merchant's stock and the purse are the fixture); the witness pages the
// shelf, clicks the items onto the merchant's table five at a time and presses
// Buy, pages the pack, drags each item from the pack onto the player's side of
// the table and presses Undo. Each frame's cells then equal the same screen
// composed without plaques and figures with the claim's plaque of the cell's
// side, chosen from the stored price, and the figure an oracle paints, over the
// whole cell. Each of a missing plaque, the other side's plaque, a plaque up or
// down, the plaque of the printed figure, a missing figure, font1, a pixel aside
// and a missing shadow leaves the frame different from that.
func TestReleaseShopPlaqueIsChosenFromTheStoredPrice(t *testing.T) {
	app, s := releaseShopApp(t)
	f := frontOf(s)
	s.CloseTip()
	art := f.shopArt()
	if art == nil {
		t.Fatal("shop art did not resolve")
	}
	oracle := newPriceOracle(t, f, "font2")
	wrongFont := newPriceOracle(t, f, "font1")

	pop := plaqueSizePopulation(t, f)
	installed, inTens, inHundreds := 0, 0, 0
	for _, it := range pop {
		if it.installed {
			installed++
		}
		if it.installed && it.price >= 10 && it.price <= 18 {
			inTens++
		}
		if it.installed && it.price >= 100 && it.price <= 198 {
			inHundreds++
		}
	}
	if inTens == 0 || inHundreds == 0 {
		t.Fatalf("the installed candidates hold %d prices in 10..18 and %d in 100..198", inTens, inHundreds)
	}
	t.Logf("%d stored prices stocked: %d installed (%d in 10..18, %d in 100..198), %d fixture prices at decade edges",
		len(pop), installed, inTens, inHundreds, len(pop)-installed)

	shelf := make([]ShopItem, len(pop))
	for i, it := range pop {
		shelf[i] = shopItemFromInstance(sim.ItemInstance{Code: uint16(it.code), Kind: it.kind, Price: it.price}, 1)
	}
	f.Shop.shelves[ShelfArmour] = shelf
	f.Town.gold = 30_000_000

	point := shopPointer(t, app)
	down := shopOrderControl(t, ui.ShopControlArrowDown)
	right := shopOrderControl(t, ui.ShopControlPackRight)
	left := shopOrderControl(t, ui.ShopControlPackLeft)
	ix, iy, err := app.HeadlessShopIdlePoint()
	if err != nil {
		t.Fatal(err)
	}

	// frame is the App's frame with the pointer on no control, and the same view
	// composed without plaques and figures.
	frame := func() (got, bare *image.RGBA, view ui.ShopScreenView) {
		t.Helper()
		if err := app.HeadlessPointer("hover", ix, iy); err != nil {
			t.Fatal(err)
		}
		if tip, _ := app.HeadlessTooltip(); tip.Visible {
			t.Fatal("a tooltip stands over the screen")
		}
		got, _, err := app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		view = s.ShopScreen()
		bareView := view
		bareView.Font, bareView.PriceFont, bareView.Character.Font = nil, nil, nil
		bareArt := *view.Art
		bareArt.Plaque = [2][7]*image.RGBA{}
		bareView.Art = &bareArt
		return got, ui.ComposeShopScreen(bareView, image.Point{}, false, nil, false), view
	}
	frames, cellsChecked, oneSmaller := 0, 0, 0
	verify := func(state string, cells []plaqueSizeCell) {
		t.Helper()
		got, bare, view := frame()
		rects := make([]image.Rectangle, len(cells))
		affected := 0
		for i, c := range cells {
			var cell ui.ShopCell
			switch c.grid {
			case "shelf":
				cell, rects[i] = view.Shelf[c.index], ui.ShopShelfCellRect(c.index)
			case "table":
				cell, rects[i] = view.Table[c.index], ui.ShopTableCellRect(c.index)
			case "pack":
				cell, rects[i] = view.Pack[c.index], ui.ShopPackCellRect(c.index)
			}
			if !cell.Occupied() || cell.Money || cell.Count != 1 || cell.Star || cell.Mine != c.mine || cell.Price != c.figure() || cell.PlaquePrice != c.stored {
				t.Fatalf("%s: %s cell %d: occupied %v money %v count %d star %v mine %v prints %d chosen from %d; want an occupied cell of one, no star, mine %v, printing %d chosen from %d",
					state, c.grid, c.index, cell.Occupied(), cell.Money, cell.Count, cell.Star, cell.Mine, cell.Price, cell.PlaquePrice, c.mine, c.figure(), c.stored)
			}
			if plaqueSizeIndex(c.figure()) != plaqueSizeIndex(c.stored) {
				affected++
			}
		}
		differences := func(o plaqueSizeVariant) (int, string) {
			want := image.NewRGBA(bare.Bounds())
			copy(want.Pix, bare.Pix)
			for i, c := range cells {
				paintPlaqueSize(want, art, oracle, rects[i], c, o)
			}
			bad, first := 0, ""
			for i, c := range cells {
				n, at := cellDifferences(got, want, rects[i])
				if bad == 0 && n > 0 {
					first = fmt.Sprintf("%s cell %d stored %d at %s", c.grid, c.index, c.stored, at)
				}
				bad += n
			}
			return bad, first
		}
		if bad, first := differences(plaqueSizeVariant{}); bad != 0 {
			t.Fatalf("%s: the frame differs from the claimed plaques and figures at %d pixel(s), first %s", state, bad, first)
		}
		controls := map[string]plaqueSizeVariant{
			"no plaque":               {noPlaque: true},
			"the other side's plaque": {other: true},
			"one plaque up":           {shift: 1},
			"one plaque down":         {shift: -1},
			"no figure":               {noFigure: true},
			"font1":                   {font: &wrongFont},
			"one pixel left":          {move: priceVariant{dx: -1}},
			"no shadow":               {move: priceVariant{noShadow: true}},
		}
		if affected > 0 {
			controls["the plaque of the printed figure"] = plaqueSizeVariant{fromFigure: true}
		}
		// A shift past the first or the seventh plaque changes nothing on a frame
		// whose cells all stand there.
		canUp, canDown := false, false
		for _, c := range cells {
			canUp = canUp || plaqueSizeIndex(c.stored) < 6
			canDown = canDown || plaqueSizeIndex(c.stored) > 0
		}
		if !canUp {
			delete(controls, "one plaque up")
		}
		if !canDown {
			delete(controls, "one plaque down")
		}
		for name, o := range controls {
			if bad, _ := differences(o); bad == 0 {
				t.Errorf("%s: the check cannot tell the claimed cells from %s", state, name)
			}
		}
		frames++
		cellsChecked += len(cells)
		oneSmaller += affected
	}

	// The shelf: its first page and its last, each cell at the stored price of
	// the item stocked there.
	point("shelf_pick", 0, "press", "release")
	if s.shopChosen != 0 || s.shelfBase != 0 {
		t.Fatalf("shelf pick 0 chose %d at base %d", s.shopChosen, s.shelfBase)
	}
	shelfPage := func(state string) {
		t.Helper()
		var cells []plaqueSizeCell
		base := s.ShopScreen().ShelfOffset
		for i := 0; i < len(ui.ShopScreenView{}.Shelf) && base+i < len(pop); i++ {
			cells = append(cells, plaqueSizeCell{"shelf", i, false, pop[base+i].price})
		}
		verify(state, cells)
	}
	shelfPage("shelf, first page")
	for n := 0; n < len(pop); n++ {
		shopTap(t, app, down, "press", "release")
	}
	shelfPage("shelf, last page")

	// The merchant's table: five items at a time, then Buy. The merchant's side
	// draws as it did before, so the first, a middle and the last batch stand
	// for it.
	point("shelf_pick", 0, "press", "release")
	packBefore := len(s.shopPackStacks())
	const perScreen = 5
	for from := 0; from < len(pop); from += perScreen {
		batch := pop[from:min(from+perScreen, len(pop))]
		var cells []plaqueSizeCell
		var total int
		for i, it := range batch {
			point("shelf", 0, "press", "release")
			cells = append(cells, plaqueSizeCell{"table", i, false, it.price})
			total += int(it.price)
		}
		if table := f.Shop.Table(); len(table) != len(batch) {
			t.Fatalf("the table holds %d places after %d shelf clicks", len(table), len(batch))
		}
		if from == 0 || from == len(pop)/2/perScreen*perScreen || from+perScreen >= len(pop) {
			verify("merchant's table", cells)
		}
		gold := f.Town.Gold()
		point("button", 1, "press", "release")
		if got := gold - f.Town.Gold(); got != total || len(f.Shop.Table()) != 0 {
			t.Fatalf("buying prices %d..%d cost %d with %d places left, want %d and none", batch[0].price, batch[len(batch)-1].price, got, len(f.Shop.Table()), total)
		}
	}
	if got := len(s.shopPackStacks()) - packBefore; got != len(pop) {
		t.Fatalf("the pack gained %d stacks, want %d", got, len(pop))
	}

	// The pack: each stack of the population in the first cell of the strip and
	// the four after it.
	stackOf := func(it plaqueSizeStock) int {
		return slices.IndexFunc(s.shopPackStacks(), func(st sim.ItemStack) bool {
			return st.Code == uint16(it.code) && st.Price == it.price
		})
	}
	priceAt := func(stacks []sim.ItemStack, k int) (int32, bool) {
		if k >= len(stacks) {
			return 0, false
		}
		for _, it := range pop {
			if stacks[k].Code == uint16(it.code) && stacks[k].Price == it.price {
				return it.price, true
			}
		}
		return 0, false
	}
	packTo := func(base int) {
		t.Helper()
		for n := 0; s.packBase != base && n < 1000; n++ {
			p := right
			if s.packBase > base {
				p = left
			}
			shopTap(t, app, p, "press", "release")
		}
		if s.packBase != base {
			t.Fatalf("the pack strip stands at %d, want %d", s.packBase, base)
		}
	}
	seen := map[int32]bool{}
	for _, it := range pop {
		if seen[it.price] {
			continue
		}
		k := stackOf(it)
		if k < 0 {
			t.Fatalf("the pack holds no stack priced %d", it.price)
		}
		packTo(k + 1)
		stacks := s.shopPackStacks()
		var cells []plaqueSizeCell
		for i := 0; i < len(ui.ShopScreenView{}.Pack); i++ {
			if price, ok := priceAt(stacks, k+i); ok && !seen[price] {
				seen[price] = true
				cells = append(cells, plaqueSizeCell{"pack", i, true, price})
			}
		}
		verify("pack", cells)
	}

	// The player's side of the table: five items at a time out of the pack,
	// then Undo.
	for from := 0; from < len(pop); from += perScreen {
		batch := pop[from:min(from+perScreen, len(pop))]
		var cells []plaqueSizeCell
		for i, it := range batch {
			k := stackOf(it)
			if k < 0 {
				t.Fatalf("the pack holds no stack priced %d", it.price)
			}
			packTo(k + 1)
			before := len(f.Shop.Table())
			// A drag stages one unit at once; a single click on equipment waits
			// out the use-tap window first.
			point("pack", 0, "press")
			point("table", 0, "move", "release")
			if len(f.Shop.Table()) != before+1 {
				t.Fatalf("a drag of the pack stack priced %d put nothing on the table", it.price)
			}
			cells = append(cells, plaqueSizeCell{"table", i, true, it.price})
		}
		verify("player's table", cells)
		point("button", 0, "press", "release")
		if len(f.Shop.Table()) != 0 {
			t.Fatalf("Undo left %d places on the table", len(f.Shop.Table()))
		}
	}
	t.Logf("%d frames, %d cells checked; %d player's-side cells take a plaque one size or more above the halved figure's",
		frames, cellsChecked, oneSmaller)
}
