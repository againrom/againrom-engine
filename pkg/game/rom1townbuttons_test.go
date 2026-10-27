package game

import (
	"image"
	"image/color"
	"testing"
)

// The first game's room buttons draw lit in the gold rest ink, brighten to
// yellow under the pointer, and join Buy and Sell with their numbers on one
// line (SHOP-050, TOWN-535); a two-line caption sits as the EN original draws it.
func TestFirstGameRoomButtonsAreLitGoldWithBuyAndSellOnOneLine(t *testing.T) {
	s := (&InstallResources{}).roomButtonStyle(ROM1TownDescription())
	if s == nil {
		t.Fatal("the first game's town names no room button style")
	}
	if !s.Lit || s.OneLine != [4]bool{false, true, true, false} {
		t.Fatalf("lit %t one-line %v", s.Lit, s.OneLine)
	}
	if s.Ink != (color.RGBA{185, 159, 73, 255}) || s.Hover != (color.RGBA{255, 255, 116, 255}) {
		t.Fatalf("ink %v hover %v", s.Ink, s.Hover)
	}
	if s.LabelShift != image.Pt(0, 6) || s.ValueShift != image.Pt(0, 1) {
		t.Fatalf("label shift %v value shift %v", s.LabelShift, s.ValueShift)
	}
}

// The first game's hover help draws gold-orange with a dark shadow one
// pixel down and right, the second game's ink (MENU-128, R2-ENGINE-385).
func TestFirstGameHoverHelpIsGoldWithAShadow(t *testing.T) {
	h := generatorDescriptions["rom1"].Hover
	if h == nil || h.Shadow == nil {
		t.Fatal("the first game's generator names no hover style with a shadow")
	}
	if h.Ink.RGBA() != (color.RGBA{160, 120, 50, 255}) || h.Shadow.RGBA() != (color.RGBA{8, 8, 8, 255}) || h.Lines != "wrapped" || h.Font != "text" {
		t.Fatalf("hover ink %v shadow %v lines %q font %q", h.Ink.RGBA(), h.Shadow.RGBA(), h.Lines, h.Font)
	}
}
