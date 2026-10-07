package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

// The tip panel's own art loader (1018 spec behaviour 1; DIV-162): the three
// shipped nodes, size-checked the way every other install picture is, and
// the front end's own cache.

// tipGemSheet is a six-frame sheet, on radiob.256's own shipped shape
// (contract's own "The shipped texts" table): round off/on, square off/on,
// small-square off/on. Every frame is the same 2x2 shape here; this file's
// tests only need six frames to exist and the loader to reach the right
// index, not a distinct picture per frame.
func tipGemSheet() []byte {
	frame := synth.Frame256{Width: 2, Height: 2, Pixels: []synth.Pixel256{
		{Index: 1, Opaque: true}, {}, {}, {Index: 1, Opaque: true},
	}}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: []color.RGBA{{}, {R: 0x10, G: 0x20, B: 0x30}},
		Frames:  []synth.Frame256{frame, frame, frame, frame, frame, frame},
	})
}

func tipArtSource() chargenSource {
	return chargenSource{
		tipBackPath: synthBMP(160, 240, color.RGBA{R: 0x40, A: 0xff}),
		// t_border.bmp ships 8-bit paletted, not 24-bit like t_back.bmp
		// (tipart.go's own doc); synthBMP8 is chargenassets_test.go's own
		// fixture builder for that shape, reused here rather than duplicated.
		tipBorderPath: synthBMP8(88, 108),
		tipGemPath:    tipGemSheet(),
	}
}

// LoadTipPanelArt reads all three nodes and returns every field non-nil when
// the install ships them at the expected sizes.
//
// TipPanelArt's four fields are the image.Image INTERFACE, not *image.RGBA
// directly (ui/tippanel.go): a typed-nil *image.RGBA assigned into one of
// them produces a non-nil interface value, so a plain "a.Border == nil"
// comparison cannot see a reader that degenerated to returning a nil
// pointer. tipPanelArtField below type-asserts to the concrete pointer
// first and compares THAT to nil, which is the comparison a typed-nil bug
// actually fails.
func TestLoadTipPanelArtReadsTheThreeShippedNodes(t *testing.T) {
	a, err := LoadTipPanelArt(tipArtSource())
	if err != nil {
		t.Fatalf("LoadTipPanelArt: %v", err)
	}
	for name, got := range map[string]image.Image{
		"Fill": a.Fill, "Border": a.Border, "GemOff": a.GemOff, "GemOn": a.GemOn,
	} {
		if tipPanelArtFieldIsNil(got) {
			t.Fatalf("LoadTipPanelArt: %s = %+v, want a non-nil *image.RGBA", name, got)
		}
	}
}

// tipPanelArtFieldIsNil reports whether img is either the nil interface or a
// non-nil interface wrapping a nil *image.RGBA (the typed-nil case this
// story's own four fields can carry — see the test above).
func tipPanelArtFieldIsNil(img image.Image) bool {
	if img == nil {
		return true
	}
	rgba, ok := img.(*image.RGBA)
	return ok && rgba == nil
}

// A missing fill node fails, naming its own address (tipBackPath's own
// "cosmetic, not fatal" contract: LoadTipPanelArt itself still reports the
// error; the room that would have shown the panel is what falls back to
// showing none, at FrontEnd.tipArt).
func TestLoadTipPanelArtMissingFillFails(t *testing.T) {
	src := tipArtSource()
	delete(src, tipBackPath)
	if _, err := LoadTipPanelArt(src); err == nil {
		t.Fatal("LoadTipPanelArt with no shipped t_back.bmp reported success")
	}
}

// A fill node at the wrong size fails — the same chargenSize check every
// other room's own art already goes through.
func TestLoadTipPanelArtWrongSizedFillFails(t *testing.T) {
	src := tipArtSource()
	src[tipBackPath] = synthBMP(4, 4, color.RGBA{A: 0xff})
	if _, err := LoadTipPanelArt(src); err == nil {
		t.Fatal("LoadTipPanelArt with a mis-sized t_back.bmp reported success")
	}
}

// A border node at the wrong size fails the same way.
func TestLoadTipPanelArtWrongSizedBorderFails(t *testing.T) {
	src := tipArtSource()
	src[tipBorderPath] = synthBMP8(4, 4)
	if _, err := LoadTipPanelArt(src); err == nil {
		t.Fatal("LoadTipPanelArt with a mis-sized t_border.bmp reported success")
	}
}

// A gem sheet with fewer frames than tipGemOnFrame needs fails: five frames,
// one short of the frame index (5) the on-gem reads.
func TestLoadTipPanelArtGemSheetTooShortFails(t *testing.T) {
	src := tipArtSource()
	frame := synth.Frame256{Width: 2, Height: 2, Pixels: []synth.Pixel256{{Index: 1, Opaque: true}, {}, {}, {}}}
	src[tipGemPath] = synth.Sheet256(synth.Sheet256Options{
		Palette: []color.RGBA{{}, {R: 1}},
		Frames:  []synth.Frame256{frame, frame, frame, frame, frame},
	})
	if _, err := LoadTipPanelArt(src); err == nil {
		t.Fatal("LoadTipPanelArt with a five-frame radiob.256 reported success (tipGemOnFrame is 5)")
	}
}

// loadTipGemFrame refuses an out-of-range frame directly, naming the address
// and the sheet's own frame count.
func TestLoadTipGemFrameOutOfRangeFails(t *testing.T) {
	src := tipArtSource()
	if _, err := loadTipGemFrame(src, tipGemPath, 6); err == nil {
		t.Fatal("loadTipGemFrame(6) on a six-frame sheet (indices 0..5) reported success")
	}
}

// FrontEnd.tipArt answers the cached value once tipArtLoaded is set, without
// touching Archives at all — proved here by leaving Archives nil (a real
// load would answer nil, not the seeded value) and setting the two cache
// fields directly, on shopArt's own "loaded on first use and cached"
// precedent (this file's own doc). LoadTipPanelArt's own read path is
// covered above; this is tipArt's one additional behaviour, the cache.
func TestFrontEndTipArtCachesAfterFirstLoad(t *testing.T) {
	want := &ui.TipPanelArt{}
	f := &FrontEnd{Presentation: Presentation{tipArtCache: resolved(want, nil)}}
	if got := f.tipArt(); got != want {
		t.Fatalf("tipArt() on an already tried cache = %p, want the cached %p", got, want)
	}
}

// A front end with no Archives answers nil, drawing no panel rather than
// panicking — the same "runs, shows nothing" shape every other room's own
// art loader gives a missing install.
func TestFrontEndTipArtWithNoArchivesReturnsNil(t *testing.T) {
	f := &FrontEnd{}
	if a := f.tipArt(); a != nil {
		t.Fatalf("tipArt() with no Archives = %+v, want nil", a)
	}
}

// A nil *FrontEnd's tipArt does not panic.
func TestNilFrontEndTipArtDoesNotPanic(t *testing.T) {
	var f *FrontEnd
	if a := f.tipArt(); a != nil {
		t.Fatalf("nil FrontEnd tipArt() = %+v, want nil", a)
	}
}
