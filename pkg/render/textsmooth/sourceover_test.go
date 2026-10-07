package textsmooth

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/text"
)

func TestSourceOverShadowSettlesAndErasesItsActualNativeBlend(t *testing.T) {
	g := &text.Glyph{Width: 2, Height: 1, Advance: 2, Pixels: []text.Pixel{{Painted: true, Level: 15}, {Painted: true, Level: 15}}}
	back := color.RGBA{120, 90, 60, 255}
	pic := image.NewRGBA(image.Rect(0, 0, 2, 1))
	pic.SetRGBA(0, 0, back)
	pic.SetRGBA(1, 0, back)
	text.ResetCapture()
	t.Cleanup(text.ResetCapture)
	text.SetCapture(false)
	text.DrawGlyphOver(pic, g, 0, 0, color.RGBA{A: 128})
	text.StopCapture()
	if pic.RGBAAt(0, 0) != (color.RGBA{59, 44, 29, 255}) {
		t.Fatal("half-alpha shadow changed its native source-over blend", pic.RGBAAt(0, 0))
	}
	calls := append([]text.DrawCall(nil), text.Captured()...)
	kept := Settle(pic, calls)
	if len(kept) != 1 || pic.RGBAAt(0, 0) != back || pic.RGBAAt(1, 0) != back {
		t.Fatal("showing source-over shadow was left baked or erased its background")
	}
}

func TestRepeatedSourceOverShadowsRetainBothAlphaContributions(t *testing.T) {
	g := &text.Glyph{Width: 1, Height: 1, Advance: 1, Pixels: []text.Pixel{{Painted: true, Level: 15}}}
	pic := image.NewRGBA(image.Rect(0, 0, 1, 1))
	pic.SetRGBA(0, 0, color.RGBA{120, 90, 60, 255})
	text.ResetCapture()
	t.Cleanup(text.ResetCapture)
	text.SetCapture(false)
	text.DrawGlyphOver(pic, g, 0, 0, color.RGBA{A: 128})
	text.DrawGlyphOver(pic, g, 0, 0, color.RGBA{A: 128})
	text.StopCapture()
	if pic.RGBAAt(0, 0) != (color.RGBA{29, 21, 14, 255}) {
		t.Fatal("repeated shadow lost its second blend", pic.RGBAAt(0, 0))
	}
	calls := append([]text.DrawCall(nil), text.Captured()...)
	kept := Settle(pic, calls)
	if len(kept) != 2 || pic.RGBAAt(0, 0) != (color.RGBA{120, 90, 60, 255}) {
		t.Fatal("shadow was deduplicated despite different underlay")
	}
}
