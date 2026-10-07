package game

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The price plaque of a shop cell (SHOP-SCREEN-037, ITEM-PRICETAG-144) is the
// plaque of the stored unit price's decimal digit count on both sides of the
// deal; only the figure printed on it is halved on the player's side. Every
// case below is a literal of the claim's arithmetic: floor(log10(price)) picks
// the plaque, (price+1)/2 is the player's figure. The pairs at a decade edge, 9
// and 10, 18 and 19, 99 and 100, 198 and 199 and so on, are where the halved
// figure and the stored price choose different plaques.

// plaqueSizeCases are stored unit prices, the figure the player's side prints
// for each and the index of the plaque both sides take (digit count minus one).
var plaqueSizeCases = []struct {
	stored, figure int32
	plaque         int
}{
	{1, 1, 0}, {9, 5, 0},
	{10, 5, 1}, {14, 7, 1}, {18, 9, 1}, {19, 10, 1}, {99, 50, 1},
	{100, 50, 2}, {150, 75, 2}, {198, 99, 2}, {199, 100, 2}, {999, 500, 2},
	{1000, 500, 3}, {1250, 625, 3}, {1998, 999, 3}, {1999, 1000, 3}, {9999, 5000, 3},
	{10000, 5000, 4}, {19998, 9999, 4}, {19999, 10000, 4}, {99999, 50000, 4},
	{100000, 50000, 5}, {199998, 99999, 5}, {199999, 100000, 5}, {999999, 500000, 5},
	{1000000, 500000, 6}, {1999998, 999999, 6}, {1999999, 1000000, 6}, {9999999, 5000000, 6},
}

// plaqueSizeArt is a plaque set of the right size with a colour of its own for
// every side and digit count, so a plaque of the wrong size differs in every
// pixel of its strip.
func plaqueSizeArt() *ui.ShopScreenArt {
	art := &ui.ShopScreenArt{}
	for side := range art.Plaque {
		for i := range art.Plaque[side] {
			pic := image.NewRGBA(image.Rect(0, 0, 60, 10))
			draw.Draw(pic, pic.Bounds(), image.NewUniform(color.RGBA{R: uint8(60 + 100*side), G: uint8(30 + 30*i), B: 180, A: 255}), image.Point{}, draw.Src)
			art.Plaque[side][i] = pic
		}
	}
	return art
}

// plaqueSizeWant is one cell the claim gives a plaque: where it stands, whose
// side it is on, the plaque index and the figure it prints.
type plaqueSizeWant struct {
	grid   string
	index  int
	mine   bool
	plaque int
	figure int32
}

// The shelf, the merchant's places on the table, the player's places on the
// table and the party pack draw the plaque of the stored price: the price 10
// takes the second plaque on the player's side as on the merchant's, and its
// figure is 5. The composed screen equals the same screen without plaques with
// the claimed plaque of each cell's side blitted at the cell's right edge one
// pixel below its top.
func TestShopGridPlaqueIsChosenFromTheStoredPrice(t *testing.T) {
	f, s := shopRoom(t, nil)
	art := plaqueSizeArt()
	bareArt := *art
	bareArt.Plaque = [2][7]*image.RGBA{}
	rects := map[string]func(int) image.Rectangle{
		"shelf": ui.ShopShelfCellRect, "table": ui.ShopTableCellRect, "pack": ui.ShopPackCellRect,
	}

	verify := func(state string, want []plaqueSizeWant) {
		t.Helper()
		v := s.ShopScreen()
		v.Art, v.Font, v.PriceFont, v.Character.Font = art, nil, nil, nil
		got := ui.ComposeShopScreen(v, image.Point{}, false, nil, false)
		bare := v
		bare.Art = &bareArt
		claimed := ui.ComposeShopScreen(bare, image.Point{}, false, nil, false)
		for _, w := range want {
			var cell ui.ShopCell
			switch w.grid {
			case "shelf":
				cell = v.Shelf[w.index]
			case "table":
				cell = v.Table[w.index]
			case "pack":
				cell = v.Pack[w.index]
			}
			if !cell.Occupied() || cell.Money || cell.Mine != w.mine || cell.Price != w.figure {
				t.Fatalf("%s: %s cell %d: occupied %v money %v mine %v prints %d, want an occupied item cell, mine %v, printing %d",
					state, w.grid, w.index, cell.Occupied(), cell.Money, cell.Mine, cell.Price, w.mine, w.figure)
			}
			side := 0
			if w.mine {
				side = 1
			}
			pic, r := art.Plaque[side][w.plaque], rects[w.grid](w.index)
			at := image.Pt(r.Max.X-pic.Bounds().Dx(), r.Min.Y+1)
			draw.Draw(claimed, image.Rectangle{Min: at, Max: at.Add(pic.Bounds().Size())}, pic, pic.Bounds().Min, draw.Over)
		}
		for _, w := range want {
			if bad, first := cellDifferences(got, claimed, priceStrip(rects[w.grid](w.index))); bad != 0 {
				t.Errorf("%s: %s cell %d printing %d differs from plaque %d of the %s family at %d pixel(s), first %s",
					state, w.grid, w.index, w.figure, w.plaque+1, map[bool]string{false: "merchant's", true: "player's"}[w.mine], bad, first)
			}
		}
	}

	const perScreen = 5
	for from := 0; from < len(plaqueSizeCases); from += perScreen {
		chunk := plaqueSizeCases[from:min(from+perScreen, len(plaqueSizeCases))]
		stock := make([]ShopItem, len(chunk))
		pack := make([]sim.ItemInstance, len(chunk))
		var shelf, merchant, table, packed []plaqueSizeWant
		for i, c := range chunk {
			stock[i] = ShopItem{Code: gridWeapon, Price: c.stored, Count: 1}
			pack[i] = sim.ItemInstance{Code: uint16(gridWeapon), Price: c.stored}
			shelf = append(shelf, plaqueSizeWant{"shelf", i, false, c.plaque, c.stored})
			merchant = append(merchant, plaqueSizeWant{"table", i, false, c.plaque, c.stored})
			table = append(table, plaqueSizeWant{"table", i, true, c.plaque, c.figure})
			packed = append(packed, plaqueSizeWant{"pack", i, true, c.plaque, c.figure})
		}

		// The merchant's side: his shelf, then the same items on his side of
		// the table.
		f.Shop.ClearTable()
		f.Shop.shelves[ShelfArmour] = append([]ShopItem(nil), stock...)
		click(s, ui.ShopControlShelfPick, roomArmour)
		verify("shelf", shelf)
		for range stock {
			if !f.Shop.TakeFromShelf(ShelfArmour, 0, 1) {
				t.Fatal("could not stage a merchant's place")
			}
		}
		verify("merchant's table", merchant)

		// The player's side: his places on the table and the pack.
		f.Shop.ClearTable()
		for _, item := range stock {
			if !f.Shop.PutOnTable(item) {
				t.Fatal("could not stage a player's place")
			}
		}
		for len(pack) < 5 {
			pack = append(pack, sim.ItemInstance{Code: uint16(0x0f00 + len(pack))})
		}
		f.Carried[0].Carry.ItemInstances = pack
		s.packBase = 1
		verify("player's table", table)
		verify("pack", packed)
	}
}
