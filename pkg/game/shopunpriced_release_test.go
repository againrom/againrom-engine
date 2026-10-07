package game

import (
	"fmt"
	"image"
	"image/draw"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The plaque and figure of a shop cell whose item is priced 0 or below
// (ITEM-PRICETAG-144, SHOP-SCREEN-037), on the installed art and font. The grid
// skips only the money element and a quantity of 0, so such an item draws the
// first plaque of its side with its number in every grid.

// installedUnpriced lists what the installed tables price at 0 or below: the
// class 14 rows, the equipment rows whose own price is 0 or below, and the
// equipment codes of a row priced above 0 whose shape and material scales bring
// the price to 0 or below.
func installedUnpriced(f *FrontEnd) (magic []data.ItemCode, rows []string, combos []data.ItemCode) {
	tb := f.Table
	for row := 1; row < tb.MagicItems.Len(); row++ {
		code := data.ItemCode(0x0e00 | row)
		if tb.MagicItems.EntryName(row) != "" && mapload.ItemInstanceFromCode(uint16(code), tb).Price <= 0 {
			magic = append(magic, code)
		}
	}
	for _, c := range []struct {
		name  string
		coll  data.Collection
		class data.ShopClass
	}{{"weapon", tb.Weapons, data.WeaponShopClass}, {"shield", tb.Shields, data.ShieldShopClass}, {"armour", tb.Armors, data.ArmorShopClass}} {
		for row := 1; row < c.coll.Len(); row++ {
			p := c.coll.EntryParams(row)
			if c.coll.EntryName(row) == "" || len(p) <= 2 {
				continue
			}
			if p[2] <= 0 {
				rows = append(rows, fmt.Sprintf("%s row %d %q at %d", c.name, row, c.coll.EntryName(row), p[2]))
				continue
			}
			for shape := 0; shape < data.ShopShapes; shape++ {
				for material := 0; material < data.ShopMaterials; material++ {
					if data.ItemPrice(p[2], tb.Shapes, tb.Materials, shape, material) <= 0 {
						combos = append(combos, data.ComposeItemCode(material, c.class(p), shape, row))
					}
				}
			}
		}
	}
	return magic, rows, combos
}

// unpricedWant states what the claim has one cell draw: the plaque family of
// its side and the number on it.
type unpricedWant struct {
	grid   string // shelf, table or pack
	index  int
	mine   bool  // the player's plaques, costm
	figure int32 // the number printed, already halved on the player's side
}

// unpricedVariant states one way of drawing a cell differently from the claim.
type unpricedVariant struct {
	plaque   int // added to the plaque index the digit count gives
	other    bool
	noPlaque bool
	noFigure bool
	figure   *int32
	font     *priceOracle
	shift    priceVariant
}

// paintUnpriced paints the claim's plaque and figure for w on pic, which holds
// the screen composed without either.
func paintUnpriced(pic *image.RGBA, art *ui.ShopScreenArt, oracle priceOracle, r image.Rectangle, w unpricedWant, o unpricedVariant) {
	figure := w.figure
	if o.figure != nil {
		figure = *o.figure
	}
	if !o.noPlaque {
		side := 0
		if w.mine != o.other {
			side = 1
		}
		index := 0
		for n := w.figure / 10; n > 0; n /= 10 {
			index++
		}
		plaque := art.Plaque[side][min(index+o.plaque, 6)]
		at := image.Pt(r.Max.X-plaque.Bounds().Dx(), r.Min.Y+1)
		draw.Draw(pic, image.Rectangle{Min: at, Max: at.Add(plaque.Bounds().Size())}, plaque, plaque.Bounds().Min, draw.Over)
	}
	if !o.noFigure {
		if o.font != nil {
			oracle = *o.font
		}
		oracle.paint(pic, r, figure, o.shift)
	}
}

// cellDifferences counts the pixels of r at which got and want differ and
// describes the first.
func cellDifferences(got, want *image.RGBA, r image.Rectangle) (int, string) {
	bad, first := 0, ""
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if got.RGBAAt(x, y) != want.RGBAAt(x, y) {
				if bad == 0 {
					first = fmt.Sprintf("(%d,%d): got %+v, want %+v", x, y, got.RGBAAt(x, y), want.RGBAAt(x, y))
				}
				bad++
			}
		}
	}
	return bad, first
}

// An item the installed tables price at 0 or -1 draws the first plaque of its
// side and its number in the shelf, on the merchant's table, on the player's
// place of the table and in the pack, as an item of any other price does; the
// money element and an empty place draw neither. Through App input the witness
// finds the created hero's access item in the pack (installed price -1), stages
// it on the table, stocks the armour shelf with the access item, an equipment
// code the installed scales price at 0 and an ordinary priced item (the merchant's
// stock is the fixture) and clicks the last of those onto the table. Each cell of
// each frame then equals the same screen composed without plaques and figures
// with the claim's plaque blitted at the cell's right edge one pixel below its
// top and the figure an oracle paints from font2's nodes, over the whole cell.
// Each of a missing plaque, the other side's plaque, the second plaque, a missing
// figure, the stored price -1 as the figure, font1, a pixel aside and a missing
// shadow leaves a cell different from that.
func TestReleaseShopGridDrawsThePlaqueOfAnUnpricedItem(t *testing.T) {
	app, s := releaseShopApp(t)
	f := frontOf(s)
	s.CloseTip()
	art := f.shopArt()
	if art == nil {
		t.Fatal("shop art did not resolve")
	}
	oracle := newPriceOracle(t, f, "font2")
	wrongFont := newPriceOracle(t, f, "font1")

	quest := data.QuestDocumentCode
	magic, rows, combos := installedUnpriced(f)
	for _, code := range magic {
		t.Logf("class 14 %#04x %q: price %d", uint16(code), decodeInstallText(itemName(code, f.Table)),
			mapload.ItemInstanceFromCode(uint16(code), f.Table).Price)
	}
	t.Logf("equipment rows priced 0 or below: %q", rows)
	for _, code := range combos {
		t.Logf("equipment code %#04x %q: price %d", uint16(code), decodeInstallText(itemName(code, f.Table)),
			mapload.ItemInstanceFromCode(uint16(code), f.Table).Price)
	}
	if !slices.Contains(magic, quest) || len(combos) == 0 {
		t.Fatalf("the installed tables price %d class 14 rows (the access item among them: %v) and %d equipment codes at 0 or below",
			len(magic), slices.Contains(magic, quest), len(combos))
	}
	access := mapload.ItemInstanceFromCode(uint16(quest), f.Table)
	zero := mapload.ItemInstanceFromCode(uint16(combos[0]), f.Table)
	if access.Price != -1 || zero.Price != 0 {
		t.Fatalf("the access item is priced %d and %#04x %d, want -1 and 0", access.Price, uint16(combos[0]), zero.Price)
	}
	pool := shopArmourPool(f.Table, f.Shop.Ceiling())
	if len(pool) == 0 || pool[0].Price <= 0 {
		t.Fatalf("the armour pool of ceiling %d holds no priced item", f.Shop.Ceiling())
	}
	priced := shopItemFromInstance(sim.ItemInstance{Code: uint16(pool[0].Code), Kind: pool[0].ItemKind, Price: pool[0].Price}, 1)

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

	// frame is the App's frame with the pointer on no control, and the same
	// view composed without plaques and figures.
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
	minusOne := int32(-1)
	verify := func(state string, w unpricedWant) {
		t.Helper()
		got, bare, view := frame()
		var cell ui.ShopCell
		var r image.Rectangle
		switch w.grid {
		case "shelf":
			cell, r = view.Shelf[w.index], ui.ShopShelfCellRect(w.index)
		case "table":
			cell, r = view.Table[w.index], ui.ShopTableCellRect(w.index)
		case "pack":
			cell, r = view.Pack[w.index], ui.ShopPackCellRect(w.index)
		}
		name := fmt.Sprintf("%s: %s cell %d", state, w.grid, w.index)
		if !cell.Occupied() || cell.Money || cell.Count != 1 || cell.Star || cell.Mine != w.mine || cell.Price != w.figure {
			t.Fatalf("%s: occupied %v money %v count %d star %v mine %v prints %d, want an occupied cell of one, no star, mine %v, printing %d",
				name, cell.Occupied(), cell.Money, cell.Count, cell.Star, cell.Mine, cell.Price, w.mine, w.figure)
		}
		expect := func(o unpricedVariant) *image.RGBA {
			want := image.NewRGBA(bare.Bounds())
			copy(want.Pix, bare.Pix)
			paintUnpriced(want, art, oracle, r, w, o)
			return want
		}
		if bad, first := cellDifferences(got, expect(unpricedVariant{}), r); bad != 0 {
			t.Fatalf("%s: the cell differs from the claimed plaque and figure at %d pixel(s), first %s", name, bad, first)
		}
		controls := map[string]unpricedVariant{
			"no plaque":               {noPlaque: true},
			"the other side's plaque": {other: true},
			"the second plaque":       {plaque: 1},
			"no figure":               {noFigure: true},
			"font1":                   {font: &wrongFont},
			"one pixel left":          {shift: priceVariant{dx: -1}},
			"one pixel down":          {shift: priceVariant{dy: 1}},
			"no shadow":               {shift: priceVariant{noShadow: true}},
		}
		if w.figure != -1 {
			controls["the stored price -1 as the figure"] = unpricedVariant{figure: &minusOne}
		}
		for variant, o := range controls {
			if bad, _ := cellDifferences(got, expect(o), r); bad == 0 {
				t.Errorf("%s: the check cannot tell the claimed cell from %s", name, variant)
			}
		}
	}
	// blank holds got to the same screen with nothing drawn on the cell: the
	// money element and an empty place carry neither plaque nor figure.
	blank := func(state, grid string, index int) {
		t.Helper()
		got, bare, view := frame()
		var cell ui.ShopCell
		var r image.Rectangle
		switch grid {
		case "shelf":
			cell, r = view.Shelf[index], ui.ShopShelfCellRect(index)
		case "table":
			cell, r = view.Table[index], ui.ShopTableCellRect(index)
		case "pack":
			cell, r = view.Pack[index], ui.ShopPackCellRect(index)
		}
		if cell.Occupied() && !cell.Money {
			t.Fatalf("%s: %s cell %d holds an item, want the money element or an empty place", state, grid, index)
		}
		if bad, first := cellDifferences(got, bare, priceStrip(r)); bad != 0 {
			t.Errorf("%s: %s cell %d draws %d pixel(s) more than its background over the plaque strip, first %s", state, grid, index, bad, first)
		}
	}

	// The pack: the created hero carries the access item.
	stacks := s.shopPackStacks()
	k := slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == uint16(quest) })
	if k < 0 || stacks[k].Price != -1 || stacks[k].Count != 1 {
		t.Fatalf("the shown member's pack holds %+v, want one access item at price -1", stacks)
	}
	packCell := k + 1 - s.packBase
	if packCell < 1 || packCell >= 5 {
		t.Fatalf("the access item stands at pack place %d, outside the strip", packCell)
	}
	blank("pack", "pack", 0)
	verify("pack", unpricedWant{"pack", packCell, true, 0})

	// The shelf: the access item, an equipment code priced 0 and a priced item.
	f.Shop.shelves[ShelfArmour] = []ShopItem{shopItemFromInstance(access, 1), shopItemFromInstance(zero, 1), priced}
	click("shelf_pick", 0)
	if s.shopChosen != 0 || s.shelfBase != 0 {
		t.Fatalf("shelf pick 0 chose %d at base %d", s.shopChosen, s.shelfBase)
	}
	verify("shelf", unpricedWant{"shelf", 0, false, 0})
	verify("shelf", unpricedWant{"shelf", 1, false, 0})
	verify("shelf", unpricedWant{"shelf", 2, false, priced.Price})
	blank("shelf", "shelf", 3)

	// A click on the zero-priced shelf cell puts it on the merchant's table; a
	// click on the access item in the pack puts it on the player's side.
	click("shelf", 1)
	if table := f.Shop.Table(); len(table) != 1 || table[0].Mine || table[0].Code != combos[0] {
		t.Fatalf("table after the shelf click %+v, want the merchant's zero-priced item", table)
	}
	click("pack", packCell)
	if table := f.Shop.Table(); len(table) != 2 || !table[1].Mine || table[1].Price != -1 {
		t.Fatalf("table after the pack click %+v, want the access item at price -1 on the player's side", table)
	}
	verify("table", unpricedWant{"table", 0, false, 0})
	verify("table", unpricedWant{"table", 1, true, 0})
	blank("table", "table", 2)
	t.Logf("%d class 14 rows, %d equipment rows and %d equipment codes priced 0 or below; access item %q, zero-priced %q",
		len(magic), len(rows), len(combos), decodeInstallText(itemName(quest, f.Table)), decodeInstallText(itemName(combos[0], f.Table)))
}
