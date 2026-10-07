package ui

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
)

func TestDebugPickerCapturesEveryShowingFaceAndShadow(t *testing.T) {
	a := newTestApp(t, appRows(3), okLoader(t))
	a.flow.screen = ScreenPicker
	a.Layout(1920, 1080)
	a.SetTextSmoothing(true)
	t.Cleanup(text.ResetCapture)
	a.Draw(ebiten.NewImage(1920, 1080))
	captured, kept, fallbacks := a.TextSettle()
	if captured == 0 || kept == 0 || fallbacks != 0 {
		t.Fatalf("picker debug text captured=%d kept=%d readbacks=%d; want showing glyphs without readback", captured, kept, fallbacks)
	}
	if a.textOverlay.buf == nil || a.textOverlay.tex == nil {
		t.Fatal("showing debug capture produced no window overlay buffer or texture")
	}
	var calls []text.DrawCall
	for _, ft := range a.TextFates() {
		if ft.Fate == textsmooth.Kept {
			calls = append(calls, ft.Call)
		}
	}
	want := image.NewRGBA(a.textOverlay.buf.Bounds())
	ox, oy := a.place.Origin()
	textsmooth.Composite(want, calls, a.place.Scale(), ox, oy)
	if !bytes.Equal(want.Pix, a.textOverlay.buf.Pix) {
		t.Fatal("App's showing overlay buffer differs from its settled face and shadow calls")
	}
}

func TestDebugFallbackScreensReplayExactlyOnceAndKeepNativeOff(t *testing.T) {
	for _, route := range []string{"town", "chargen", "load", "menu"} {
		t.Run(route, func(t *testing.T) {
			a := newTestApp(t, appRows(3), okLoader(t))
			switch route {
			case "town":
				a.SetTown(&fakeTown{})
				a.flow.showTown("")
			case "chargen":
				a.flow.chargen = NewChargen(chargenLegalSetup())
				a.flow.screen = ScreenChargen
			case "load":
				a.flow.loadList = NewPicker(appRows(3))
				a.flow.screen = ScreenLoad
			case "menu":
				a.flow.openGameMenu(ScreenTown)
			}
			a.Layout(1920, 1080)
			screen := ebiten.NewImage(1920, 1080)
			defer screen.Dispose()
			a.SetTextSmoothing(true)
			a.Draw(screen)
			first := a.TextFates()
			if len(first) == 0 {
				t.Fatal("fallback paints text but captures none")
			}
			showing := 0
			for _, ft := range first {
				if !ft.Call.SourceOver {
					t.Fatal("fallback used an installed glyph")
				}
				if ft.Fate == textsmooth.Kept {
					showing++
				} else if ft.Fate != textsmooth.Blank && ft.Fate != textsmooth.Hidden {
					t.Fatalf("fallback glyph fate %s", ft.Fate)
				}
			}
			if showing == 0 {
				t.Fatal("fallback has no showing overlay")
			}
			a.Draw(screen)
			if !reflect.DeepEqual(first, a.TextFates()) {
				t.Fatal("cached frame changed or duplicated capture")
			}
			a.SetTextSmoothing(false)
			a.Draw(screen)
			if captured, kept, _ := a.TextSettle(); captured != 0 || kept != 0 || len(a.TextFates()) != 0 {
				t.Fatal("native off still captures text")
			}
			a.SetTextSmoothing(true)
			a.Draw(screen)
			if !reflect.DeepEqual(first, a.TextFates()) {
				t.Fatal("on/off/on changed capture")
			}
		})
	}
	t.Cleanup(text.ResetCapture)
}

func TestDebugLayerKeepsItsShadowAndHonoursLaterCoverageAndClip(t *testing.T) {
	dst := ebiten.NewImage(30, 18)
	defer dst.Dispose()
	var log pixelLog
	log.reset(dst.Bounds())
	log.fill(dst.Bounds(), color.RGBA{120, 90, 60, 255})
	text.ResetCapture()
	t.Cleanup(text.ResetCapture)
	text.SetCapture(false)
	drawDebugText(dst, &log, "AB\nC", -2, 1)
	text.StopCapture()
	calls := append([]text.DrawCall(nil), text.Captured()...)
	if len(calls) != 6 {
		t.Fatalf("captured %d masks, want three faces and shadows", len(calls))
	}
	cover := image.Rect(3, 1, 7, 17)
	log.fill(cover, color.RGBA{2, 3, 4, 255})
	log.end()
	outcomes, certain := textsmooth.Explain(calls, dst.Bounds(), log.oracle())
	if !certain {
		t.Fatal("exact debug layer requested a readback")
	}
	shadow, masked := false, false
	for _, out := range outcomes {
		if out.Fate != textsmooth.Kept {
			continue
		}
		c := out.Call
		shadow = shadow || c.Color.A == 128
		masked = masked || c.Mask != nil
		for n, p := range c.Glyph.Pixels {
			at := image.Pt(c.X+n%6, c.Y+n/6)
			if p.Painted && at.In(cover) && (c.Mask == nil || c.Mask[n]) {
				t.Fatal("covered debug cell leaked into overlay", at)
			}
		}
	}
	if !shadow || !masked {
		t.Fatalf("showing shadow=%v partial mask=%v", shadow, masked)
	}
}

func TestMapEditorCapturesItsInstalledFontAtScaleOne(t *testing.T) {
	for _, nilFont := range []bool{false, true} {
		e := NewMapEditor(panelFont(), nil, func(string) (*InspectionDocument, error) {
			return &InspectionDocument{Viewer: litViewer(t)}, nil
		})
		if err := e.Open("map"); err != nil {
			t.Fatal(err)
		}
		if nilFont {
			e.font = nil
		}
		e.Layout(1280, 800)
		if e.doc.Viewer.place.Scale() != 1 {
			t.Fatal("editor text was resized")
		}
		screen := ebiten.NewImage(1280, 800)
		e.Draw(screen)
		captured, kept, _ := e.TextSettle()
		if nilFont {
			if captured != 0 || kept != 0 {
				t.Fatal("nil editor font gained a fallback")
			}
		} else if captured == 0 || kept == 0 {
			t.Fatal("editor panel has no captured installed glyph")
		}
		for _, ft := range e.TextFates() {
			if ft.Fate != textsmooth.Kept && ft.Fate != textsmooth.Blank {
				t.Fatalf("editor glyph fate %s", ft.Fate)
			}
		}
		e.SetTextSmoothing(false)
		e.Draw(screen)
		if captured, kept, _ := e.TextSettle(); captured != 0 || kept != 0 {
			t.Fatal("native editor still captures glyphs")
		}
		screen.Dispose()
	}
	t.Cleanup(text.ResetCapture)
}
