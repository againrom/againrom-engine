package game

import (
	"fmt"
	"image"

	"againrom/pkg/screenart"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// The shop screen's artwork, read from the shop room of the room
// description: SHOP-SCREEN-032 names the literal path each widget loads its
// own art by, and SHOP-SCREEN-038 establishes no shop-screen graphic is
// localised, so the same entries serve the English and the Russian install.
//
// NOTHING IS READ UNTIL THE PLAYER OPENS THE SHOP, and it is read once. A load
// is attempted exactly once whether it succeeds or fails: an install missing
// a file must not be re-read on every frame.
//
// EVERY PICTURE IS OPTIONAL. loadShopArt answers a non-nil ShopScreenArt even
// with no archive at all, holding nil for every picture, and pkg/ui draws
// nothing for a nil one. So the screen is composable in a test with no install
// present and an install that lost one file loses one picture.
//
// PURE BLACK IS THE TRANSPARENT COLOUR of every bitmap here: each is a 24-bit
// DIB with no alpha channel, and the shape the engine blits is carried by
// leaving the rest of the canvas at (0,0,0). costs3.bmp is 60x10 and 56.7%
// pure black, drawing a tablet only right of column 33. The blit routine
// SHOP-SCREEN-036 reaches was not decoded; what IS established is that the
// seven plaques of a family are one canvas size with the tablet growing
// leftward by digit count (SHOP-SCREEN-037), which no opaque blit can produce.

// loadShopArt resolves every picture the screen paints with; the page art and
// the scene art are the shop room's of room description d.
func loadShopArt(d *town.Description, src entrySource) *ui.ShopScreenArt {
	art := &ui.ShopScreenArt{}
	if src == nil {
		return art
	}
	set, err := roomPageArt(d, "shop", src)
	if err != nil {
		return art
	}
	art = shopArtOf(set)
	art.Scene, _ = loadRoomSceneArt(d, "shop", src)
	return art
}

// shopArtOf is the screen's pictures from a shop room's loaded page art, by
// the entry names both games' descriptions give them.
func shopArtOf(set *screenart.Set) *ui.ShopScreenArt {
	art := &ui.ShopScreenArt{}
	if set == nil {
		return art
	}
	pic := func(name string) *image.RGBA { return rgbaOf(set.Picture(name)) }
	art.Shelf, art.Table, art.Room = pic("shelf"), pic("table"), pic("room")
	art.Frame, art.Menu, art.Coin = pic("frame"), pic("menu"), pic("coin")
	for i := range art.Arrow {
		art.Arrow[i] = pic(fmt.Sprintf("arrow/%d", i))
		art.PackArrow[i] = pic(fmt.Sprintf("pack-arrow/%d", i))
	}
	for i := range art.PackCap {
		art.PackCap[i] = pic(fmt.Sprintf("pack-cap/%d", i))
	}
	for i := range art.Button {
		art.Button[i] = pic(fmt.Sprintf("button/%d", i))
	}
	// The cell backgrounds, indexed by ui.ShopCellBack (SHOP-SCREEN-036):
	// backinvg.bmp appears twice because the original draws it for an empty
	// cell and for a cell the shown member's class cannot use.
	for i := range art.Back {
		art.Back[i] = pic(fmt.Sprintf("back/%d", i))
	}
	// The seven plaques a side, one per decimal digit count (SHOP-SCREEN-037).
	// Index 0 is the merchant's family and index 1 the player's, which is
	// ui.ShopCell.Mine's own order.
	for i := 0; i < 7; i++ {
		art.Plaque[0][i] = pic(fmt.Sprintf("plaque-merchant/%d", i))
		art.Plaque[1][i] = pic(fmt.Sprintf("plaque-player/%d", i))
	}
	return art
}

// shopArt is the front end's one copy, loaded on the first entry into the shop.
func (f *FrontEnd) shopArt() *ui.ShopScreenArt {
	if f == nil {
		return nil
	}
	return f.Presentation.shopArt(&f.InstallResources)
}

func (p *Presentation) shopArt(in *InstallResources) *ui.ShopScreenArt {
	if p.shopArtCache.Tried() {
		return p.shopArtCache.Value()
	}
	p.shopArtCache.begin()
	var src entrySource
	if s := in.Archives.artSource(); s != nil {
		src = s
	}
	art := p.shopArtCache.store(loadShopArt(TownDescription(in.profile()), src))
	art.Book = in.BottomHUDArt.Value()
	return art
}
