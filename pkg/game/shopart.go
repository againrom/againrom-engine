package game

import (
	"fmt"
	"image"
	"image/draw"

	"againrom/pkg/formats/spr256"
	"againrom/pkg/ui"
)

// The shop screen's artwork.
//
// EVERY ADDRESS HERE IS DECODED. SHOP-SCREEN-032 names the literal path each
// widget loads its own art by, and this file resolves those paths against the
// container filesystem: the first component of the original's path names the
// archive, so `graphics\interface\ShopTable.bmp` is `graphics/interface/
// shoptable.bmp` here. SHOP-SCREEN-038 establishes no shop-screen graphic is
// localised, so the same addresses serve the English and the Russian install.
//
// NOTHING IS READ UNTIL THE PLAYER OPENS THE SHOP, and it is read once. A load
// is attempted exactly once whether it succeeds or fails, which is sheetCache's
// own "tried" convention (statics.go): an install missing a file must not be
// re-read on every frame.
//
// EVERY PICTURE IS OPTIONAL. loadShopArt answers a non-nil ShopScreenArt even
// with no archive at all, holding nil for every picture, and pkg/ui draws
// nothing for a nil one. So the screen is composable in a test with no install
// present (golden rule 2) and an install that lost one file loses one picture.

const shopArtPrefix = graphicsPrefix + "interface/"

// shopMoviesMerchant is where the merchant's own static picture lives:
// moviesPrefix (archives.go) plus the shipped `shopanim\Pose2-3` folder
// SHOP-MERCHANT-046 names (`movies\shopanim\Pose2-3\1.bmp`, 76x176 on both
// roots). It is the one shop asset MoviesArchive carries; everything else the
// shop paints comes out of graphics.res.
const shopMoviesMerchant = moviesPrefix + "shopanim/pose2-3/1.bmp"

// shopRackFolders is folder `4-i`, index for index with the researched hit and
// draw rectangles. Each family has eleven members. The controller decides
// which of those members reaches a paint; this loader only owns immutable art.
var shopRackFolders = [4]string{"04", "03", "02", "01"}

const (
	shopRackFrameCount     = 11
	shopMerchantIdleCount  = 28 // Pose2-3 files 2..29; file 29 is terminal-before-draw.
	shopMerchantReactCount = 11 // Yes/No files 2..12; base pose is slot zero.
)

// The two arrow pictures the grid draws. SHOP-SCREEN-032 names four —
// ShopArrow1 and ShopArrow3 into one pair of slots, ShopArrow2 and ShopArrow4
// into the other — and does not say which of a pair is the resting state. The
// resting pair is taken to be 1 and 2: shoparrow1.bmp draws a chevron pointing
// up and shoparrow2.bmp one pointing down, matching the upper and lower arrow
// rectangles. The other two are the pressed states and this build has no pressed
// state to draw.
var shopArrowFiles = [4]string{"shoparrow1.bmp", "shoparrow2.bmp", "shoparrow3.bmp", "shoparrow4.bmp"}

var shopButtonFiles = [4]string{"shopbutton1.bmp", "shopbutton2.bmp", "shopbutton3.bmp", "shopbutton4.bmp"}

// The cell backgrounds, indexed by ui.ShopCellBack (SHOP-SCREEN-036).
// backinvg.bmp appears twice because the original draws it for two different
// arms — an empty cell and a cell the shown member's class cannot use —
// and this build has to tell those two apart everywhere but at the blit.
var shopBackFiles = [4]string{"backinvg.bmp", "backinv.bmp", "backinvs.bmp", "backinvg.bmp"}

// loadShopArt resolves every picture the screen paints with.
func loadShopArt(src entrySource) *ui.ShopScreenArt {
	art := &ui.ShopScreenArt{
		Shelf:    loadShopBMP(src, "shopinv.bmp"),
		Table:    loadShopBMP(src, "shoptable.bmp"),
		Room:     loadShopBMP(src, "shopanim/shopmain.bmp"),
		Frame:    loadShopSprite(src, "shopframe.256"),
		Menu:     loadShopBMP(src, "shopmenu.bmp"),
		Merchant: loadShopBMPAddr(src, shopMoviesMerchant),
	}
	for i, name := range shopArrowFiles {
		art.Arrow[i] = loadShopBMP(src, name)
		art.PackArrow[i] = loadShopBMP(src, fmt.Sprintf("invarrow%d.bmp", i+1))
	}
	if frame := loadShopBMP(src, "invframe.bmp"); frame != nil {
		for i, x := range [2]int{0, 432} {
			corner := image.NewRGBA(image.Rect(0, 0, 32, 88))
			draw.Draw(corner, corner.Bounds(), frame, image.Pt(x, 0), draw.Src)
			art.PackCap[i] = corner
		}
	}
	for i, name := range shopButtonFiles {
		art.Button[i] = loadShopBMP(src, name)
	}
	for i, name := range shopBackFiles {
		art.Back[i] = loadShopBMP(src, name)
	}
	for i, folder := range shopRackFolders {
		frames, complete := loadShopAnimationFamily(src, shopArtPrefix+"shopanim/"+folder+"/%d.bmp", 1, shopRackFrameCount)
		art.ShelfAnim[i] = frames[0]
		if complete {
			art.RackAnimation[i] = frames
		}
	}
	if frames, complete := loadShopAnimationFamily(src, moviesPrefix+"shopanim/pose2-3/%d.bmp", 2, shopMerchantIdleCount); complete {
		art.MerchantIdle = frames
	}
	if frames, complete := loadShopAnimationFamily(src, moviesPrefix+"shopanim/yes/%d.bmp", 2, shopMerchantReactCount); complete {
		art.MerchantYes = frames
	}
	if frames, complete := loadShopAnimationFamily(src, moviesPrefix+"shopanim/no/%d.bmp", 2, shopMerchantReactCount); complete {
		art.MerchantNo = frames
	}
	art.Coin = loadShopIcon(src, shopMoneyPath)
	// The seven plaques a side, one per decimal digit count (SHOP-SCREEN-037).
	// Index 0 is the merchant's family and index 1 the player's, which is
	// ui.ShopCell.Mine's own order.
	for i := 0; i < 7; i++ {
		art.Plaque[0][i] = loadShopBMP(src, fmt.Sprintf("costs%d.bmp", i+1))
		art.Plaque[1][i] = loadShopBMP(src, fmt.Sprintf("costm%d.bmp", i+1))
	}
	return art
}

// loadShopAnimationFamily reads one family without letting a partial load
// masquerade as a sequence. It returns every attempted slot so callers can
// retain a first-frame static fallback, plus a complete bit that alone admits
// progression. One missing member affects no sibling family.
func loadShopAnimationFamily(src entrySource, pattern string, first, count int) ([]*image.RGBA, bool) {
	frames := make([]*image.RGBA, count)
	complete := count > 0
	for i := range frames {
		frames[i] = loadShopBMPAddr(src, fmt.Sprintf(pattern, first+i))
		if frames[i] == nil {
			complete = false
		}
	}
	return frames, complete
}

// shopMoneyPath is the money cell's coin: one 80x80 frame, the same size as a
// cell and as backinv.bmp (SHOP-MONEY-048).
const shopMoneyPath = shopArtPrefix + "money/money.16a"

// loadShopBMP reads one 24-bit bitmap under the interface directory, or nil.
// Every failure is an absence and none is an error, which is loadPortrait's own
// contract at a different leaf.
//
// PURE BLACK IS THE TRANSPARENT COLOUR OF THESE BITMAPS, and this is where
// that is applied. Each of them is a 24-bit DIB with no alpha channel, and
// the shape the engine blits is carried by leaving the rest of the canvas at
// (0,0,0): costs3.bmp is 60x10 and 56.7% pure black, drawing a tablet only
// right of column 33, and shopbutton1.bmp is 120x52 with black corners
// around a rounded plate.
//
// THIS IS READ OFF THE SHIPPED ART AND NOT OFF THE IMAGE. The blit routine
// SHOP-SCREEN-036 reaches (vt+0x38) was not decoded, so what the original's
// blitter does with a black pixel is not a claim; what IS established is that
// the seven plaques of a family are one canvas size with the tablet growing
// leftward by digit count (SHOP-SCREEN-037), which no opaque blit can produce.
// docs/DIVERGENCES.md carries the row.
func loadShopBMP(src entrySource, name string) *image.RGBA {
	return loadShopBMPAddr(src, shopArtPrefix+name)
}

// loadShopBMPAddr is loadShopBMP over a full address rather than one relative
// to shopArtPrefix, for the one piece the shop reads out of a different
// archive: the merchant's static picture lives in MoviesArchive, addressed
// through moviesPrefix rather than graphicsPrefix (archives.go).
func loadShopBMPAddr(src entrySource, addr string) *image.RGBA {
	if src == nil {
		return nil
	}
	pic, err := readChargenBMP(src, addr)
	if err != nil {
		return nil
	}
	return keyBlack(pic)
}

// keyBlack clears the alpha of every pure-black pixel, in place, and answers the
// picture it was given.
func keyBlack(pic *image.RGBA) *image.RGBA {
	if pic == nil {
		return nil
	}
	for i := 0; i+3 < len(pic.Pix); i += 4 {
		if pic.Pix[i] == 0 && pic.Pix[i+1] == 0 && pic.Pix[i+2] == 0 {
			pic.Pix[i+3] = 0
		}
	}
	return pic
}

// loadShopIcon reads one .16a sprite's frame 0 through the item icons' own
// reader, which carries the sheet's per-pixel level into the alpha channel
// (SPR16A-ALPHA-025). A failure is an absence.
func loadShopIcon(src entrySource, addr string) *image.RGBA {
	if src == nil {
		return nil
	}
	pic, err := loadItemIcon(src, addr)
	if err != nil {
		return nil
	}
	return pic
}

// loadShopSprite reads frame 0 of one .256 sheet under the interface directory
// as an RGBA with the sheet's structural transparency carried into the alpha
// channel — which is what the merchant's room frame needs: it is a frame with a
// hole in it, and an opaque blit of it would paint over the room picture.
func loadShopSprite(src entrySource, name string) *image.RGBA {
	if src == nil {
		return nil
	}
	raw, err := src.ReadFile(shopArtPrefix + name)
	if err != nil {
		return nil
	}
	sheet, err := spr256.Decode(raw)
	if err != nil || len(sheet.Frames) == 0 || !sheet.HasPalette {
		return nil
	}
	f := sheet.Frames[0]
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) < f.Width*f.Height {
		return nil
	}
	pic := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
	for i := 0; i < f.Width*f.Height; i++ {
		p := f.Pixels[i]
		if !p.Opaque || int(p.Index) >= len(sheet.Palette) {
			continue
		}
		e := sheet.Palette[p.Index]
		o := i * 4
		pic.Pix[o], pic.Pix[o+1], pic.Pix[o+2], pic.Pix[o+3] = e.R, e.G, e.B, 0xff
	}
	return pic
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
	if in.Archives != nil {
		src = in.Archives.Containers
	}
	art := p.shopArtCache.store(loadShopArt(src))
	art.Book = in.BottomHUDArt.Value()
	return art
}
