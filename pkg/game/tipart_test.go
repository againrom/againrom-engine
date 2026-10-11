package game

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/ui"
)

const (
	tipBackPath = graphicsPrefix + "interface/t_back.bmp"
	tipGemPath  = graphicsPrefix + "interface/radiob.256"
)

func loadTipGemFrame(src entrySource, addr string, frame int) (*image.RGBA, error) {
	if src == nil {
		return nil, fmt.Errorf("%s: no archive", addr)
	}
	raw, err := src.ReadFile(addr)
	if err != nil {
		return nil, err
	}
	sheet, err := spr256.Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", addr, err)
	}
	table, ok := sheet.Table()
	if !ok {
		return nil, fmt.Errorf("%s: no palette", addr)
	}
	if frame < 0 || frame >= len(sheet.Frames) {
		return nil, fmt.Errorf("%s: frame %d, sheet carries %d", addr, frame, len(sheet.Frames))
	}
	f := sheet.Frames[frame]
	if f.Width <= 0 || f.Height <= 0 || len(f.Pixels) < f.Width*f.Height {
		return nil, fmt.Errorf("%s: frame %d is empty", addr, frame)
	}
	return f.RGBA(table), nil
}

func tipGemSheet() []byte {
	frame := synth.Frame256{Width: 2, Height: 2, Pixels: []synth.Pixel256{{Index: 1, Opaque: true}, {}, {}, {Index: 1, Opaque: true}}}
	return synth.Sheet256(synth.Sheet256Options{Palette: []color.RGBA{{}, {R: 16, G: 32, B: 48}}, Frames: []synth.Frame256{frame, frame, frame, frame, frame, frame}})
}

func tipFrameSheet(sizes []image.Point) []byte {
	frames := make([]synth.Frame256, 9)
	for _, size := range sizes {
		pixels := make([]synth.Pixel256, size.X*size.Y)
		for i := range pixels {
			pixels[i] = synth.Pixel256{Index: uint8(len(frames)), Opaque: true}
		}
		frames = append(frames, synth.Frame256{Width: size.X, Height: size.Y, Pixels: pixels})
	}
	for i := 0; i < 9; i++ {
		frames[i] = frames[9]
	}
	palette := make([]color.RGBA, 18)
	for i := range palette {
		palette[i] = color.RGBA{uint8(i), 80, 70, 255}
	}
	return synth.Sheet256(synth.Sheet256Options{Palette: palette, Frames: frames})
}

func tipFrameSizes() []image.Point {
	return []image.Point{{48, 32}, {32, 32}, {48, 32}, {32, 32}, {32, 32}, {32, 32}, {32, 32}, {48, 32}, {32, 32}}
}

func tipArtSource() chargenSource {
	return chargenSource{
		"graphics/interface/lm.256": tipFrameSheet(tipFrameSizes()),
		tipGemPath:                  tipGemSheet(),
	}
}

func TestLoadTipPanelArtUsesRoomFrameBank(t *testing.T) {
	a, err := LoadTipPanelArt(tipArtSource(), rom1Screens)
	if err != nil {
		t.Fatal(err)
	}
	if a.Frame == nil {
		t.Fatal("room frame bank was not loaded")
	}
	for i, p := range a.Frame.Pieces {
		if p == nil || p.Bounds().Size() != tipFrameSizes()[i] {
			t.Fatalf("piece %d has wrong size", i)
		}
		if got := p.RGBAAt(0, 0).R; got != uint8(9+i) {
			t.Fatalf("piece %d selected index %d, want %d", i, got, 9+i)
		}
	}
	if a.GemOff == nil || a.GemOn == nil {
		t.Fatal("checkbox frames missing")
	}
}

func TestLoadTipPanelArtRejectsMissingOrMalformedRoomFrames(t *testing.T) {
	for _, bad := range []string{"missing", "size", "short"} {
		t.Run(bad, func(t *testing.T) {
			src := tipArtSource()
			switch bad {
			case "missing":
				delete(src, "graphics/interface/lm.256")
			case "size":
				sizes := tipFrameSizes()
				sizes[0] = image.Pt(4, 4)
				src["graphics/interface/lm.256"] = tipFrameSheet(sizes)
			case "short":
				src["graphics/interface/lm.256"] = tipGemSheet()
			}
			if _, err := LoadTipPanelArt(src, rom1Screens); err == nil {
				t.Fatal("malformed room frame accepted")
			}
		})
	}
}

func TestLoadTipPanelArtGemSheetTooShortFails(t *testing.T) {
	src := tipArtSource()
	frame := synth.Frame256{Width: 2, Height: 2, Pixels: []synth.Pixel256{{Index: 1, Opaque: true}, {}, {}, {}}}
	src[tipGemPath] = synth.Sheet256(synth.Sheet256Options{Palette: []color.RGBA{{}, {R: 1}}, Frames: []synth.Frame256{frame, frame, frame, frame, frame}})
	if _, err := LoadTipPanelArt(src, rom1Screens); err == nil {
		t.Fatal("short checkbox sheet accepted")
	}
}

func TestLoadTipGemFrameOutOfRangeFails(t *testing.T) {
	if _, err := loadTipGemFrame(tipArtSource(), tipGemPath, 6); err == nil {
		t.Fatal("out-of-range checkbox frame accepted")
	}
}

func TestFrontEndTipArtCachesAfterFirstLoad(t *testing.T) {
	want := &ui.TipPanelArt{}
	f := &FrontEnd{Presentation: Presentation{tipArtCache: resolved(want, nil)}}
	if got := f.tipArt(); got != want {
		t.Fatal("tip art cache changed")
	}
}

func TestFrontEndTipArtWithNoArchivesReturnsNil(t *testing.T) {
	if got := (&FrontEnd{}).tipArt(); got != nil {
		t.Fatal("tip art without archives")
	}
}

func TestNilFrontEndTipArtDoesNotPanic(t *testing.T) {
	var f *FrontEnd
	if got := f.tipArt(); got != nil {
		t.Fatal("tip art on nil front end")
	}
}
