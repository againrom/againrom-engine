package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestTranslucentDialogueKeepsReachedHUDSmoothing(t *testing.T) {
	v := backdropViewer(t)
	v.Layout(1024, 768)
	v.ShowReadout(true)
	v.SetTextSmoothing(true)
	v.DeferPointer(true)
	layout := NoticeLayout{Box: image.Rect(-200, -200, 1200, 1000), Fill: color.RGBA{A: 48}, Border: color.RGBA{A: 48}}
	v.SetNoticeLayouts(layout, layout)
	v.SetDialogue(Dialogue{})
	v.SetNoticeBackdrop(color.RGBA{})
	v.Draw(ebiten.NewImage(1024, 768))
	baseline := v.textKept
	if baseline < 100 || v.textSettleFallbacks != 0 {
		t.Fatalf("HUD control kept %d glyphs with %d framebuffer fallbacks", baseline, v.textSettleFallbacks)
	}
	v.SetDialogueBackdrop(DialogueBackdrop{})
	v.Draw(ebiten.NewImage(1024, 768))
	if v.textCaptured < baseline || v.textKept < baseline || v.textSettleFallbacks != 0 {
		t.Fatalf("translucent dialogue kept %d/%d glyphs, control %d, framebuffer fallbacks %d", v.textKept, v.textCaptured, baseline, v.textSettleFallbacks)
	}
}

func TestRemappedGlyphLaterWashColorsAndUnderlay(t *testing.T) {
	g := &text.Glyph{Width: 1, Height: 1, Pixels: []text.Pixel{{Painted: true, Level: 16}}}
	ink := color.RGBA{220, 175, 70, 255}
	under := color.RGBA{124, 90, 201, 255}
	calls := []text.DrawCall{{Glyph: g, Color: ink, Flat: true, Under: []color.RGBA{under}}}
	lookup, _ := backdrop.New(backdrop.RGB565, backdrop.Full)
	bounds := image.Rect(0, 0, 1, 1)
	remapCaptured(calls, bounds, lookup, 1)
	pic := image.NewRGBA(bounds)
	pic.SetRGBA(0, 0, lookup.Color(ink))
	var log pixelLog
	log.reset(bounds)
	log.upload(pic)
	wash := color.RGBA{A: 48}
	log.overSolid(bounds, wash)
	kept, certain := textsmooth.DecideWith(calls, bounds, log.oracle())
	if !certain || len(kept) != 1 {
		t.Fatalf("later wash lost remapped glyph: certain %v, kept %d", certain, len(kept))
	}
	want := color.RGBA{140, 111, 39, 255}
	if got := kept[0].NativeColor(16, 0); got != want {
		t.Errorf("native remapped color under later wash: got %v want %v", got, want)
	}
	if got := kept[0].Under[0]; got != text.Tinted(wash, lookup.Color(under)) {
		t.Errorf("later wash did not reach restored underlay: %v", got)
	}
	out := image.NewRGBA(bounds)
	textsmooth.Composite(out, kept, 1, 0, 0)
	if got := out.RGBAAt(0, 0); got != want {
		t.Errorf("smoothed remapped color under later wash: got %v want %v", got, want)
	}
	remapCaptured(kept, bounds, lookup, 1)
	if got := kept[0].RasterColors[0]; got != lookup.Color(want) || kept[0].Tint != (color.RGBA{}) {
		t.Fatalf("second remap failed to fold the intervening wash once: color %v, tint %v", got, kept[0].Tint)
	}
}
