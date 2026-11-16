package ui

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
)

// rimFont is a face of glyphs 7 wide and 9 high: a ring of edge cells at edge
// level around an interior at the top level. With coverage the levels are
// alpha and the glyph is drawn by DrawOver; without they are shades of the ink.
func rimFont(coverage bool, edge uint8) *text.Font {
	glyphs := make([]text.Glyph, 224)
	for i := range glyphs {
		g := text.Glyph{Width: 7, Height: 9, Advance: 8, CoverageLevels: coverage, Pixels: make([]text.Pixel, 63)}
		if i > 0 {
			for n := range g.Pixels {
				x, y := n%7, n/7
				level := uint8(text.MaxLevel)
				if x == 0 || y == 0 || x == 6 || y == 8 {
					level = edge
				}
				g.Pixels[n] = text.Pixel{Level: level, Painted: (x+y+i)%5 != 0 || level == text.MaxLevel}
			}
		}
		glyphs[i] = g
	}
	return &text.Font{Glyphs: glyphs}
}

// rimWood is a textured brown panel.
func rimWood(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(10 + (x*7+y*13)%17)
			img.SetRGBA(x, y, color.RGBA{R: v + 40, G: v + 15, B: v, A: 255})
		}
	}
	return img
}

// rimCapture composes a screen inside a capture window and answers its frame
// and the glyphs it captured.
func rimCapture(compose func() *image.RGBA) (*image.RGBA, []text.DrawCall) {
	text.ResetCapture()
	text.SetCapture(false)
	frame := compose()
	text.StopCapture()
	calls := append([]text.DrawCall(nil), text.Captured()...)
	text.ResetCapture()
	return frame, calls
}

// rimWindow is the window the text path shows for a native frame and its
// captured glyphs at scale s: the frame with the kept glyphs erased, scaled by
// the window scaler, and the kept glyphs drawn over it at window resolution.
// It answers the window, the same window with every glyph erased and the
// glyphs kept.
func rimWindow(frame *image.RGBA, calls []text.DrawCall, s float64) (win, art *image.RGBA, kept []text.DrawCall) {
	work := &image.RGBA{Pix: append([]uint8(nil), frame.Pix...), Stride: frame.Stride, Rect: frame.Rect}
	kept = textsmooth.Settle(work, calls)
	size := image.Rect(0, 0, int(float64(frame.Rect.Dx())*s), int(float64(frame.Rect.Dy())*s))
	win = image.NewRGBA(size)
	referenceFinalFrame(win, work, s, 0, 0, filterSmooth)
	textsmooth.Composite(win, kept, s, 0, 0)

	bare := &image.RGBA{Pix: append([]uint8(nil), frame.Pix...), Stride: frame.Stride, Rect: frame.Rect}
	textsmooth.Erase(bare, calls)
	art = image.NewRGBA(size)
	referenceFinalFrame(art, bare, s, 0, 0, filterSmooth)
	return win, art, kept
}

func rimLum(c color.RGBA) int { return (299*int(c.R) + 587*int(c.G) + 114*int(c.B)) / 1000 }

// rimExcess is the rim measure: over the window rectangle r, the most any
// window pixel's luminance falls below the darker of the scaled art at that
// pixel and the darkest cell any glyph paints. A glyph drawn at window
// resolution over the scaled art never goes below that floor; a glyph pixel
// the scaler rang does.
func rimExcess(win, art *image.RGBA, calls []text.DrawCall, r image.Rectangle) int {
	ink := 1 << 30
	for _, c := range calls {
		for _, p := range c.Glyph.Pixels {
			if p.Painted {
				ink = min(ink, rimLum(c.ShownColor(p.Level)))
			}
		}
	}
	worst := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			floor := min(ink, rimLum(art.RGBAAt(x, y)))
			worst = max(worst, floor-rimLum(win.RGBAAt(x, y)))
		}
	}
	return worst
}

// rimRegion is the window rectangle the glyphs cover at scale s, grown by pad.
func rimRegion(calls []text.DrawCall, s float64, pad int) image.Rectangle {
	b := glyphBounds(calls)
	return image.Rect(int(float64(b.Min.X)*s)-pad, int(float64(b.Min.Y)*s)-pad, int(float64(b.Max.X)*s)+pad, int(float64(b.Max.Y)*s)+pad)
}

func rimNamePage(over bool, font *text.Font) func() *image.RGBA {
	return func() *image.RGBA {
		l := *testGenerator()
		l.PreCreate.Name.Over = over
		l.PreCreate.Name.Ink = GeneratorInk{255, 235, 90}
		l.PreCreate.Name.Prompt.Ink = GeneratorInk{255, 235, 90}
		art := &ChargenPresentation{Layout: &l, NameFont: font, Background: rimWood(640, 480)}
		c := NewChargen(ChargenSetup{Name: "Gildarius", PreCreate: &ChargenPreCreate{Prompt: "Name:", Art: art}})
		return composeChargenPage(c, chargenNone, chargenNone)
	}
}

// rimTownCard is the town Statistics card on a wood screen, as the town shell
// draws it.
func rimTownCard(font *text.Font) func() *image.RGBA {
	return func() *image.RGBA {
		dst := rimWood(640, 480)
		body := solidPic(160, 242, characterCardFill)
		view := TownCharacterView{
			HasSubject: true, Statistics: true, PaneRect: image.Rect(70, 90, 230, 332),
			Font: font, CardFont: font, Subject: PanelSubject{Name: "Gildarius"},
			StatsPane:  TownPane{Body: body},
			CardShadow: &TextShadow{Ink: color.RGBA{A: 255}, At: image.Pt(1, 1)},
		}
		drawCharacterPaneBody(dst, view)
		return dst
	}
}

// rimMissionPanel is the mission character panel as panelPresent composes it,
// laid on a frame where the viewer blits it.
func rimMissionPanel(t *testing.T, font *text.Font) func() *image.RGBA {
	v := missionPaneViewer(t)
	v.SetFont(font)
	v.SetEntities([]MapEntity{panelEntity(1, "Gildarius", 10, 10, 1, 1)})
	v.sel = selection{1}
	if !v.characterPaneView(true).Statistics {
		v.toggleHudPanel(hudPanelDoll)
	}
	return func() *image.RGBA {
		v.panelPic = nil
		start := markCapture()
		pic, at, ok := v.panelPresent()
		if !ok {
			t.Fatal("panelPresent composed no character panel")
		}
		shiftCapture(start, at)
		frame := rimWood(v.frameW, v.frameH)
		draw.Draw(frame, pic.Bounds().Add(at), pic, pic.Bounds().Min, draw.Src)
		return frame
	}
}

type rimScene struct {
	name    string
	compose func() *image.RGBA
}

func rimScenes(t *testing.T) []rimScene {
	shade, coverage := rimFont(false, 6), rimFont(true, 6)
	return []rimScene{
		{"chargen name, shade glyphs (first game)", rimNamePage(false, shade)},
		{"chargen name, coverage glyphs mixed in (second game)", rimNamePage(true, coverage)},
		{"town statistics card", rimTownCard(shade)},
		{"mission character panel", rimMissionPanel(t, shade)},
	}
}

// rimBaked counts the cells of every captured glyph that still hold the
// glyph's raster in the frame the scaler gets, after the settle kept and
// erased what it keeps.
func rimBaked(frame *image.RGBA, calls []text.DrawCall) int {
	work := &image.RGBA{Pix: append([]uint8(nil), frame.Pix...), Stride: frame.Stride, Rect: frame.Rect}
	textsmooth.Settle(work, calls)
	bare := &image.RGBA{Pix: append([]uint8(nil), frame.Pix...), Stride: frame.Stride, Rect: frame.Rect}
	textsmooth.Erase(bare, calls)
	n := 0
	for _, c := range calls {
		for i, p := range c.Glyph.Pixels {
			at := image.Pt(c.X+i%c.Glyph.Width, c.Y+i/c.Glyph.Width)
			if p.Painted && at.In(frame.Rect) && work.RGBAAt(at.X, at.Y) != bare.RGBAAt(at.X, at.Y) {
				n++
			}
		}
	}
	return n
}

// Every game text reaches the window as glyphs drawn at window resolution:
// after the settle no glyph raster is left in the frame the scaler gets, and
// no window pixel of the text region falls below the darker of the scaled art
// there and the darkest cell a glyph paints.
func TestGameTextHasNoScalerRimAtAnyWindowScale(t *testing.T) {
	for _, sc := range rimScenes(t) {
		frame, calls := rimCapture(sc.compose)
		if len(calls) == 0 {
			t.Fatalf("%s: no glyph captured", sc.name)
		}
		if n := rimBaked(frame, calls); n != 0 {
			t.Errorf("%s: %d glyph cells stay in the frame the window scaler resamples", sc.name, n)
		}
		for _, s := range []float64{1, 1.25, 1.5, 2, 3} {
			win, art, _ := rimWindow(frame, calls, s)
			got := rimExcess(win, art, calls, rimRegion(calls, s, 4))
			t.Logf("%s scale %v excess %d", sc.name, s, got)
			if got > 6 {
				t.Errorf("%s at scale %v: a window pixel is %d below the art and the glyph ink", sc.name, s, got)
			}
		}
	}
}

// Text smoothing off opens no capture window, so nothing is recorded and the
// pixels a screen draws are the pixels it drew before; with a window open they
// are the same.
func TestGameTextCaptureLeavesTheFramePixelsAlone(t *testing.T) {
	for _, sc := range rimScenes(t) {
		text.ResetCapture()
		off := sc.compose()
		if len(text.Captured()) != 0 {
			t.Errorf("%s: a glyph was captured with no window open", sc.name)
		}
		on, _ := rimCapture(sc.compose)
		if string(on.Pix) != string(off.Pix) {
			t.Errorf("%s: the frame differs with a capture window open", sc.name)
		}
	}
}

// rimCrop writes the window rectangle r of win, enlarged by whole pixels, as a
// PNG at path.
func rimCrop(t *testing.T, path string, win *image.RGBA, r image.Rectangle, zoom int) {
	t.Helper()
	out := image.NewRGBA(image.Rect(0, 0, r.Dx()*zoom, r.Dy()*zoom))
	for y := 0; y < out.Rect.Dy(); y++ {
		for x := 0; x < out.Rect.Dx(); x++ {
			out.SetRGBA(x, y, win.RGBAAt(r.Min.X+x/zoom, r.Min.Y+y/zoom))
		}
	}
	witness.PNG(t, path, out)
}

// With AGAINROM_RIM_CROPS naming a directory, the second game's name line at
// window scale 2 is written there twice: as the window scaler resampled the
// glyph edges before, and as the overlay draws it now. Without it the test
// skips.
func TestGameTextNameLineCrops(t *testing.T) {
	dir := os.Getenv("AGAINROM_RIM_CROPS")
	if dir == "" {
		t.Skip("AGAINROM_RIM_CROPS names no output directory")
	}
	frame, calls := rimCapture(rimNamePage(true, rimFont(true, 6)))
	before := append([]text.DrawCall(nil), calls...)
	for i := range before {
		before[i].SourceOver = false
	}
	r := rimRegion(calls, 2, 6)
	win, _, _ := rimWindow(frame, before, 2)
	rimCrop(t, filepath.Join(dir, "name-line-before.png"), win, r, 4)
	win, _, _ = rimWindow(frame, calls, 2)
	rimCrop(t, filepath.Join(dir, "name-line-after.png"), win, r, 4)
}
