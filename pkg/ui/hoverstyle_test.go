package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// A styled hover box draws its text in the style's ink over the style's
// shadow one pixel right and down; the shared box keeps popupTextColor and
// draws no shadow.
func TestHoverStyleDrawsItsInkOverItsShadow(t *testing.T) {
	f := shippedShapeFont()
	shadow := color.RGBA{8, 8, 8, 255}
	style := &hoverStyle{ink: color.RGBA{160, 120, 50, 255}, shadow: &shadow, authored: true}
	pic := composeHoverBox([]string{"A"}, f, nil, style)
	if pic == nil {
		t.Fatal("no styled box")
	}
	if got := pic.RGBAAt(hoverTextX, hoverTextY); got != style.ink {
		t.Errorf("text pixel %v, want the style ink %v", got, style.ink)
	}
	if got := pic.RGBAAt(hoverTextX+1, hoverTextY+1); got != shadow {
		t.Errorf("shadow pixel %v, want %v", got, shadow)
	}
	shared := composeHoverBox([]string{"A"}, f, nil, nil)
	if got := shared.RGBAAt(hoverTextX, hoverTextY); got != popupTextColor {
		t.Errorf("shared text pixel %v, want %v", got, popupTextColor)
	}
	if got := shared.RGBAAt(hoverTextX+1, hoverTextY+1); got != hoverFill {
		t.Errorf("shared box drew %v right and down of the text, want the fill %v", got, hoverFill)
	}
}

// Authored lines keep every '#' segment, blank ones included, and wrap none;
// the shared box drops blank segments and wraps to its column.
func TestHoverStyleAuthoredLinesKeepBlankLinesAndDoNotWrap(t *testing.T) {
	f := shippedShapeFont()
	long := strings.Repeat("A", 40)
	target := tooltipTarget{tooltipText, "help", tooltipLines("AA##" + long), f}
	frame := image.Rect(0, 0, 640, 480)
	authored, _, ok := tooltipPicture(target, image.Pt(0, 470), frame, nil, &hoverStyle{ink: popupTextColor, authored: true})
	if !ok {
		t.Fatal("no authored box")
	}
	if want := hoverPitch*3 + hoverExtraH; authored.Bounds().Dy() != want {
		t.Errorf("authored box %d tall, want %d: three lines, the blank one kept", authored.Bounds().Dy(), want)
	}
	if w := f.Advance(long) + hoverExtraW; authored.Bounds().Dx() < w {
		t.Errorf("authored box %d wide, want at least %d: the long line unwrapped", authored.Bounds().Dx(), w)
	}
	shared, _, ok := tooltipPicture(target, image.Pt(0, 470), frame, nil, nil)
	if !ok {
		t.Fatal("no shared box")
	}
	if shared.Bounds().Dx() > tooltipMaxWidth {
		t.Errorf("shared box %d wide, want at most %d", shared.Bounds().Dx(), tooltipMaxWidth)
	}
	blank := tooltipTarget{tooltipText, "blank", tooltipLines("AA##AA"), f}
	if pic, _, _ := tooltipPicture(blank, image.Pt(0, 470), frame, nil, nil); pic == nil || pic.Bounds().Dy() != hoverPitch*2+hoverExtraH {
		t.Error("the shared box kept a blank line")
	}
}

// A description's hover style decodes into the box style, and a style the
// builder cannot draw is refused.
func TestGeneratorHoverStyleDecodes(t *testing.T) {
	shadow := GeneratorInk{8, 8, 8}
	h := &GeneratorHover{Font: "text", Ink: GeneratorInk{160, 120, 50}, Shadow: &shadow, Lines: "authored"}
	s := h.style()
	if s == nil || s.ink != (color.RGBA{160, 120, 50, 255}) || s.shadow == nil || *s.shadow != (color.RGBA{8, 8, 8, 255}) || !s.authored {
		t.Fatalf("style %+v", s)
	}
	if (*GeneratorHover)(nil).style() != nil {
		t.Error("a description without a hover style named one")
	}
	d := *testGenerator()
	d.Hover = &GeneratorHover{Font: "install", Lines: "authored"}
	if err := d.validate(); err == nil {
		t.Error("a hover style in the install font validated")
	}
	d.Hover = &GeneratorHover{Font: "text", Lines: "wrapped"}
	if err := d.validate(); err != nil {
		t.Errorf("a wrapped hover style: %v", err)
	}
}
