package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/text"
)

// TestShopQuantityAndPurseDrawInTheGridFontLeftAligned is SHOP-106: the grid
// painter draws quantity and price in one font, font2, and the quantity run is
// left-aligned at the cell's left plus 10, shadow first, one pixel right and
// down. The purse cell goes through the quantity run.
func TestShopQuantityAndPurseDrawInTheGridFontLeftAligned(t *testing.T) {
	missionFont, gridFont := townShellRosterTestFont(), numeralFont()
	r := image.Rect(100, 50, 180, 130)
	for _, cell := range []ShopCell{
		{Back: ShopBackItem, Count: 4321},
		{Money: true, Count: 4321},
	} {
		dst := image.NewRGBA(image.Rect(0, 0, 300, 300))
		calls := text.Record(func() { drawShopCell(dst, nil, missionFont, gridFont, cell, r, false, nil) })
		var faces, shadows []text.DrawCall
		for _, call := range calls {
			if call.Flat {
				shadows = append(shadows, call)
			} else {
				faces = append(faces, call)
			}
		}
		for _, call := range calls {
			if !glyphOf(gridFont, call.Glyph) {
				t.Fatalf("cell %+v drew a glyph that is not font2's", cell)
			}
		}
		// An occupied cell then draws its price, 0, through the same font.
		if got := recordedText(gridFont, faces); got != "4,321" && got != "4,3210" {
			t.Fatalf("cell %+v drew %q, want its grouped quantity first", cell, got)
		}
		if len(shadows) == 0 || len(faces) == 0 {
			t.Fatalf("cell %+v drew %d shadow and %d face calls", cell, len(shadows), len(faces))
		}
		if faces[0].X != r.Min.X+10 || faces[0].Y != r.Max.Y-15 {
			t.Fatalf("cell %+v quantity starts at (%d,%d), want (%d,%d): flag 0 is left", cell, faces[0].X, faces[0].Y, r.Min.X+10, r.Max.Y-15)
		}
		if shadows[0].X != r.Min.X+11 || shadows[0].Y != r.Max.Y-14 {
			t.Fatalf("cell %+v quantity shadow at (%d,%d), want one pixel right and down", cell, shadows[0].X, shadows[0].Y)
		}
	}
}

func glyphOf(f *text.Font, g *text.Glyph) bool {
	for c := 0; c < 256; c++ {
		if f.GlyphFor(byte(c)) == g {
			return true
		}
	}
	return false
}

// TestPackReadoutCountsDrawInTheGoldRampOverAFlatShadow is SHOP-108: the pack
// readout draws the purse and every stack count above one in the ramp the shop
// grid uses, over a flat 8 8 8 shadow one pixel right and down.
func TestPackReadoutCountsDrawInTheGoldRampOverAFlatShadow(t *testing.T) {
	font := townShellRosterTestFont()
	icon := image.NewRGBA(image.Rect(0, 0, 18, 18))
	s := InventorySubject{Pack: []*image.RGBA{icon, icon}, PackCount: []uint32{12, 3400}, PackPurse: []bool{false, true}}
	pic := renderPackBarArt(s, 0, 9, image.Rect(0, 678, 864, 768), font, nil, -1, nil)
	var face, shadow, black int
	for y := 0; y < pic.Bounds().Dy(); y++ {
		for x := 0; x < pic.Bounds().Dx(); x++ {
			switch pic.RGBAAt(x, y) {
			case shopPriceInk:
				face++
			case messageShadowColor:
				shadow++
			case color.RGBA{A: 0xff}:
				black++
			}
		}
	}
	if face == 0 || shadow == 0 || black != 0 {
		t.Fatalf("pack counts drew %d ramp-top, %d flat-shadow and %d black pixels", face, shadow, black)
	}
}

// TestShopPressedAndHoveredCaptionsDrawOnePixelLower is SHOP-107's offset: a
// pressed-and-hovered button redraws both captions one pixel lower and no pixel
// to either side, and a hovered button that is not pressed does not move.
func TestShopPressedAndHoveredCaptionsDrawOnePixelLower(t *testing.T) {
	font := townShellRosterTestFont()
	base := ShopScreenView{Font: font, Live: [4]bool{true, true, true, true}, Purse: 10, Buy: 20, Sell: 30, Total: 40,
		Words: Words{ShopUndo: "ab", ShopBuy: "cd", ShopSell: "ef", ShopExit: "gh"}}
	for i, r := range shopButtonRects {
		hover := r.Min.Add(r.Max).Div(2)
		inRect := func(calls []text.DrawCall) []text.DrawCall {
			var out []text.DrawCall
			for _, c := range calls {
				if !c.Flat && image.Pt(c.X, c.Y).In(r.Inset(-8)) {
					out = append(out, c)
				}
			}
			return out
		}
		rest := inRect(text.Record(func() { ComposeShopScreen(base, hover, true, nil, false) }))
		pressedView := base
		pressedView.Press = ShopControl{Kind: ShopControlButton, Index: i}
		held := inRect(text.Record(func() { ComposeShopScreen(pressedView, hover, true, nil, false) }))
		if len(rest) == 0 || len(held) != len(rest) {
			t.Fatalf("button %d drew %d caption glyphs hovered and %d pressed", i, len(rest), len(held))
		}
		for n := range rest {
			if held[n].X != rest[n].X || held[n].Y != rest[n].Y+1 {
				t.Fatalf("button %d glyph %d moved from (%d,%d) to (%d,%d), want one pixel down", i, n, rest[n].X, rest[n].Y, held[n].X, held[n].Y)
			}
		}
	}
}
