package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/text"
)

var testRoomStyleInk = color.RGBA{185, 159, 73, 255}
var testRoomStyleHover = color.RGBA{255, 255, 116, 255}

// glyphsOf reports whether every recorded call draws a glyph of f.
func glyphsOf(f *text.Font, calls []text.DrawCall) bool {
	own := map[*text.Glyph]bool{}
	for i := range f.Glyphs {
		own[&f.Glyphs[i]] = true
	}
	for _, c := range calls {
		if !own[c.Glyph] {
			return false
		}
	}
	return len(calls) > 0
}

// A composed room page's buttons draw their captions through the panel
// builder in the style's font, in its ink at rest and its hover ink under
// the pointer; without a style they keep the page font and the town caption
// ink in both states.
func TestRoomButtonsDrawInTheStyle(t *testing.T) {
	page, styled := shippedShapeFont(), shippedShapeFont()
	style := &RoomButtonStyle{Font: styled, Ink: testRoomStyleInk, Hover: testRoomStyleHover}
	buttons := []RoomButton{
		{Rect: image.Rect(484, 91, 624, 137), Caption: "AB"},
		{Rect: image.Rect(484, 138, 624, 184), Caption: "BA", Hover: true},
	}
	for _, tc := range []struct {
		name       string
		style      *RoomButtonStyle
		font       *text.Font
		rest, over color.RGBA
	}{
		{"styled", style, styled, testRoomStyleInk, testRoomStyleHover},
		{"plain", nil, page, townShellText, townShellText},
	} {
		dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
		calls := text.Record(func() { DrawRoomButtons(dst, page, tc.style, buttons) })
		if !glyphsOf(tc.font, calls) {
			t.Errorf("%s: the captions are not drawn in the expected font", tc.name)
		}
		if got := recordedText(tc.font, calls); got != "ABBA" {
			t.Errorf("%s: captions %q, want ABBA", tc.name, got)
		}
		for i, c := range calls {
			want := tc.rest
			if i >= 2 {
				want = tc.over
			}
			if c.Color != want {
				t.Errorf("%s: glyph %d ink %v, want %v", tc.name, i, c.Color, want)
			}
		}
	}
}

// The second game's shop plaques draw in the room button style: Undo with
// nothing to undo keeps the rest ink, and a pending buy shows below zero;
// without a style Undo is greyed and the buy shows as a positive figure.
func TestShopPlaquesInTheRoomButtonStyle(t *testing.T) {
	page, styled := shippedShapeFont(), shippedShapeFont()
	v := ShopScreenView{Font: page, Purse: 1000, Buy: 3, Total: 997, Live: [4]bool{false, true, true, true}}
	v.Words.ShopUndo, v.Words.ShopBuy, v.Words.ShopSell, v.Words.ShopExit = "A", "B", "AA", "BB"
	draw := func(v ShopScreenView) []text.DrawCall {
		dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
		return text.Record(func() { shopButtonPanel(v, image.Point{}, false).drawButtons(dst, v.ButtonStyle.font(v.Font)) })
	}
	plain := draw(v)
	if plain[0].Color != townShellDisabled {
		t.Errorf("first game: Undo ink %v, want the disabled ink %v", plain[0].Color, townShellDisabled)
	}
	v.ButtonStyle, v.BuyDebit = &RoomButtonStyle{Font: styled, Ink: testRoomStyleInk, Hover: testRoomStyleHover}, true
	styledCalls := draw(v)
	if !glyphsOf(styled, styledCalls) {
		t.Error("the plaques are not drawn in the style's font")
	}
	if styledCalls[0].Color != testRoomStyleInk {
		t.Errorf("Undo with nothing to undo: ink %v, want the rest ink %v", styledCalls[0].Color, testRoomStyleInk)
	}
	if got, want := len(styledCalls)-len(plain), 1; got != want {
		t.Errorf("the styled plaques draw %d more glyphs than the plain ones, want %d: the buy's minus sign", got, want)
	}
	if v.Buy != 3 {
		t.Errorf("the view's pending buy is %d after drawing, want 3", v.Buy)
	}
}
