package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/text"
)

func TestTipCloseCaptionUsesSelectedFontAnchor(t *testing.T) {
	f := &text.Font{Glyphs: make([]text.Glyph, 224), Spacing: 2}
	for i := range f.Glyphs {
		g := text.Glyph{Width: 8, Height: 10, Advance: 6, Pixels: make([]text.Pixel, 80)}
		if i != 0 {
			for p := range g.Pixels {
				g.Pixels[p] = text.Pixel{Level: 15, Painted: true}
			}
		}
		f.Glyphs[i] = g
	}
	for _, tc := range []struct {
		name           string
		hover, pressed bool
		shadow         int
	}{
		{"idle", false, false, 2},
		{"hover", true, false, 2},
		{"pressed inside", true, true, 4},
		{"pressed outside", false, true, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := TipPanelView{Rect: image.Rect(328, 0, 640, 200), Text: "A", CloseLabel: "Q", ToggleLabel: "Z", Font: f, Art: tipTestArt(), CloseHover: tc.hover, ClosePressed: tc.pressed}
			pic := image.NewRGBA(image.Rect(0, 0, 640, 480))
			calls := text.Record(func() { ComposeTipPanel(pic, v) })
			ink, shadow := 0, 0
			for _, call := range calls {
				if call.Glyph != f.GlyphFor('Q') {
					continue
				}
				x, y := 556, 164
				if call.Flat {
					x, y = x+tc.shadow, y+tc.shadow
					shadow++
				} else {
					ink++
				}
				if call.X != x || call.Y != y {
					t.Errorf("caption flat=%v at (%d,%d), want (%d,%d)", call.Flat, call.X, call.Y, x, y)
				}
				if !image.Rect(call.X, call.Y, call.X+8, call.Y+10).In(image.Rect(520, 160, 600, 178)) {
					t.Error("caption cell extends outside Close")
				}
			}
			if ink != 1 || shadow != 1 {
				t.Fatalf("caption calls ink/shadow = %d/%d, want 1/1", ink, shadow)
			}
		})
	}
}
