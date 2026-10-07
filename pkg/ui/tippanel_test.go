package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

// The floating tip panel's own composition and hit test (1018 spec
// behaviours 1, 3, 4).

// tipTestArt is a fill and a border each large enough to tile once across a
// panel this file's rects use, filled with a distinct solid colour so a
// composed frame can be told apart from the bare background.
func tipTestArt() *TipPanelArt {
	fill := image.NewRGBA(image.Rect(0, 0, 40, 40))
	draw.Draw(fill, fill.Bounds(), &image.Uniform{C: color.RGBA{10, 20, 30, 255}}, image.Point{}, draw.Src)
	border := image.NewRGBA(image.Rect(0, 0, 88, 108))
	draw.Draw(border, border.Bounds(), &image.Uniform{C: color.RGBA{200, 180, 90, 255}}, image.Point{}, draw.Src)
	gemOff := image.NewRGBA(image.Rect(0, 0, tipPanelGemSize, tipPanelGemSize))
	draw.Draw(gemOff, gemOff.Bounds(), &image.Uniform{C: color.RGBA{60, 60, 60, 255}}, image.Point{}, draw.Src)
	gemOn := image.NewRGBA(image.Rect(0, 0, tipPanelGemSize, tipPanelGemSize))
	draw.Draw(gemOn, gemOn.Bounds(), &image.Uniform{C: color.RGBA{240, 220, 40, 255}}, image.Point{}, draw.Src)
	return &TipPanelArt{Fill: fill, Border: border, GemOff: gemOff, GemOn: gemOn}
}

// A panel is Showing only with every one of Art, Font, Text and a non-empty
// Rect present. Each is checked in isolation so a future field that is
// forgotten in the guard is caught by whichever case exercises it, not
// masked by the other three.
func TestTipPanelViewShowingRequiresEveryField(t *testing.T) {
	font := shopTipTestFont()
	art := tipTestArt()
	rect := image.Rect(0, 0, 100, 100)
	cases := []struct {
		name string
		v    TipPanelView
		want bool
	}{
		{"complete", TipPanelView{Rect: rect, Text: "hi", Art: art, Font: font}, true},
		{"no art", TipPanelView{Rect: rect, Text: "hi", Font: font}, false},
		{"no font", TipPanelView{Rect: rect, Text: "hi", Art: art}, false},
		{"no text", TipPanelView{Rect: rect, Art: art, Font: font}, false},
		{"empty rect", TipPanelView{Text: "hi", Art: art, Font: font}, false},
	}
	for _, tc := range cases {
		if got := tc.v.Showing(); got != tc.want {
			t.Errorf("%s: Showing() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// ComposeTipPanel draws nothing at all when the view is not Showing — no
// fill, no border, no text — and something inside the rect when it is.
func TestComposeTipPanelDrawsNothingWhenNotShowing(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 200, 200))
	blank := image.NewRGBA(image.Rect(0, 0, 200, 200))
	ComposeTipPanel(dst, TipPanelView{})
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			if got, want := dst.RGBAAt(x, y), blank.RGBAAt(x, y); got != want {
				t.Fatalf("(%d,%d) = %+v, want the untouched destination %+v", x, y, got, want)
			}
		}
	}
}

func TestComposeTipPanelPaintsTheFillInsideTheRect(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 200, 200))
	rect := image.Rect(20, 20, 180, 180)
	v := TipPanelView{Rect: rect, Text: "the shop centre is where the merchant trades", Art: tipTestArt(), Font: shopTipTestFont()}
	ComposeTipPanel(dst, v)
	// A point just inside the corner, past the border's own corner patch and
	// before the text rect, lands on the tiled fill.
	at := image.Pt(rect.Min.X+2, rect.Min.Y+2)
	if got := dst.RGBAAt(at.X, at.Y); got == (color.RGBA{}) {
		t.Fatalf("fill corner (%d,%d) is still transparent black; want the tiled fill", at.X, at.Y)
	}
}

// ComposeTipPanel must consume the two install-selected captions carried by
// the view. A view-only assertion would stay green if the composer kept its
// former English literals, so this compares the actual pixels in each control
// while every other field is identical.
func TestComposeTipPanelDrawsTheViewCaptions(t *testing.T) {
	rect := image.Rect(0, 0, 312, 200)
	base := TipPanelView{Rect: rect, Text: "x", ToggleOn: true, Art: tipTestArt(), Font: shopTipTestFont()}
	render := func(v TipPanelView) *image.RGBA {
		dst := image.NewRGBA(rect)
		ComposeTipPanel(dst, v)
		return dst
	}
	different := func(a, b *image.RGBA, r image.Rectangle) bool {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
					return true
				}
			}
		}
		return false
	}

	fallback := render(base)
	custom := base
	custom.CloseLabel, custom.ToggleLabel = "X", "Y"
	installed := render(custom)
	if !different(fallback, installed, TipPanelCloseRect(rect)) {
		t.Fatal("changing CloseLabel changed no pixels in the close control")
	}
	if !different(fallback, installed, TipPanelToggleRect(rect)) {
		t.Fatal("changing ToggleLabel changed no pixels in the toggle control")
	}
}

// The close and toggle controls sit inside TipPanelCloseRect/TipPanelToggleRect
// and inside the panel's own Rect (TipPanelTextRect's own doc: the button
// row is inset from the panel's own edges).
func TestTipControlRectsSitInsideThePanel(t *testing.T) {
	rect := image.Rect(0, 0, 312, 200)
	closeRect := TipPanelCloseRect(rect)
	toggleRect := TipPanelToggleRect(rect)
	if !closeRect.In(rect) {
		t.Errorf("TipPanelCloseRect %v not inside panel %v", closeRect, rect)
	}
	if !toggleRect.In(rect) {
		t.Errorf("TipPanelToggleRect %v not inside panel %v", toggleRect, rect)
	}
	if closeRect.Overlaps(toggleRect) {
		t.Errorf("close %v and toggle %v overlap", closeRect, toggleRect)
	}
}

// TipPanelControlAt: outside the rect is no hit at all; inside the rect but
// off both controls is a consumed hit with TipControlNone (spec behaviour 1,
// "takes input before it" — the whole panel swallows the press, not only its
// two named controls); on each control is that control, consumed.
func TestTipPanelControlAt(t *testing.T) {
	rect := image.Rect(100, 100, 300, 300)
	v := TipPanelView{Rect: rect, Text: "x", Art: tipTestArt(), Font: shopTipTestFont()}

	if kind, consumed := TipPanelControlAt(v, image.Pt(0, 0)); consumed || kind != TipControlNone {
		t.Fatalf("outside the rect: kind=%v consumed=%v, want TipControlNone, false", kind, consumed)
	}
	closeRect := TipPanelCloseRect(rect)
	if kind, consumed := TipPanelControlAt(v, closeRect.Min.Add(image.Pt(1, 1))); !consumed || kind != TipControlClose {
		t.Fatalf("on the close control: kind=%v consumed=%v, want TipControlClose, true", kind, consumed)
	}
	toggleRect := TipPanelToggleRect(rect)
	if kind, consumed := TipPanelControlAt(v, toggleRect.Min.Add(image.Pt(1, 1))); !consumed || kind != TipControlToggle {
		t.Fatalf("on the toggle control: kind=%v consumed=%v, want TipControlToggle, true", kind, consumed)
	}
	// Inside the rect, off both controls (the panel's own top-left, above
	// the text — never inside either button's own row).
	if kind, consumed := TipPanelControlAt(v, rect.Min.Add(image.Pt(2, 2))); !consumed || kind != TipControlNone {
		t.Fatalf("inside the panel off both controls: kind=%v consumed=%v, want TipControlNone, true", kind, consumed)
	}
}

// A panel that is not Showing consumes nothing, so a caller cannot swallow
// input behind a panel with nothing drawn on screen.
func TestTipPanelControlAtNotShowingConsumesNothing(t *testing.T) {
	rect := image.Rect(0, 0, 300, 300)
	if kind, consumed := TipPanelControlAt(TipPanelView{Rect: rect}, image.Pt(10, 10)); consumed || kind != TipControlNone {
		t.Fatalf("not showing: kind=%v consumed=%v, want TipControlNone, false", kind, consumed)
	}
}

// The toggle draws the lit gem when ToggleOn and the unlit one otherwise —
// the two art pictures this file gives distinct colours, so the pixel at
// the toggle's own draw origin tells them apart.
func TestComposeTipPanelDrawsTheGemForToggleState(t *testing.T) {
	rect := image.Rect(0, 0, 312, 200)
	art := tipTestArt()
	render := func(on bool) *image.RGBA {
		dst := image.NewRGBA(image.Rect(0, 0, 312, 200))
		ComposeTipPanel(dst, TipPanelView{Rect: rect, Text: "x", ToggleOn: on, Art: art, Font: shopTipTestFont()})
		return dst
	}
	toggleRect := TipPanelToggleRect(rect)
	gy := toggleRect.Min.Y + (toggleRect.Dy()-tipPanelGemSize)/2
	at := image.Pt(toggleRect.Min.X, gy)

	off := render(false).RGBAAt(at.X, at.Y)
	on := render(true).RGBAAt(at.X, at.Y)
	if off == on {
		t.Fatalf("the toggle drew the same pixel %+v regardless of ToggleOn; want the off and on gems to differ", off)
	}
}

// TipPanelFits reports true for a panel with no text to lose (DIV-162's own
// "there is no text to lose" reading of a not-Showing view).
func TestTipPanelFitsNotShowingIsTrue(t *testing.T) {
	if !TipPanelFits(TipPanelView{}) {
		t.Fatal("TipPanelFits(TipPanelView{}) = false, want true (nothing to show)")
	}
}

// TipPanelFits tells a rect that holds every wrapped line apart from one
// that drops the last (DIV-162): the same text and font, only the rect's own
// height differs, one line's worth apart on either side of the boundary
// ComposeTipPanel's own draw loop already stops at.
func TestTipPanelFitsTellsAFullRectFromATruncatingOne(t *testing.T) {
	font := shopTipTestFont()
	art := tipTestArt()
	text := "one two three four five six seven eight nine ten"
	width := 60 // narrow enough that this wraps to several lines under a 24px advance
	lines := wrapShopTip(font, text, width)
	if len(lines) < 3 {
		t.Fatalf("fixture text wrapped to %d lines at width %d, want at least 3 to test a partial fit", len(lines), width)
	}
	pitch := font.Height() + 2

	full := image.Rect(0, 0, width+2*tipPanelInset, tipPanelInset*2+tipPanelButtonRowH+4+font.Height()+(len(lines)-1)*pitch)
	if !TipPanelFits(TipPanelView{Rect: full, Text: text, Art: art, Font: font}) {
		t.Fatalf("TipPanelFits = false for a rect sized to hold all %d wrapped lines, want true", len(lines))
	}

	short := full
	short.Max.Y -= pitch // one line's worth shorter: the last line no longer fits
	if TipPanelFits(TipPanelView{Rect: short, Text: text, Art: art, Font: font}) {
		t.Fatal("TipPanelFits = true for a rect one line shorter than the full fit, want false")
	}
}
