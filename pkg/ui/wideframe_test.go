package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
)

func TestWideFrameCompositionStretchesTheTownPresentationAndAnchorsTheColumn(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	for y := 0; y < frame.H; y++ {
		for x := 0; x < frame.W; x++ {
			src.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: uint8(x >> 8), A: 0xff})
		}
	}

	native := newWideFrameLayout(frame.W, wideFrameTownColumns).compose(src)
	if !bytes.Equal(native.Pix, src.Pix) {
		t.Fatal("the 640x480 town layout changed a native pixel")
	}

	const wideW = 854
	wide := newWideFrameLayout(wideW, wideFrameTownColumns).compose(src)
	if got := wide.Bounds(); got != image.Rect(0, 0, wideW, frame.H) {
		t.Fatalf("wide bounds = %v, want 854x480", got)
	}
	for _, tc := range []struct {
		name string
		got  color.RGBA
		want color.RGBA
	}{
		{"presentation starts at its native left edge", wide.RGBAAt(0, 123), src.RGBAAt(0, 123)},
		{"presentation stretches through the former gap", wide.RGBAAt(600, 123), src.RGBAAt(410, 123)},
		{"presentation ends immediately before the seam", wide.RGBAAt(677, 123), src.RGBAAt(463, 123)},
		{"complete seam starts beside moved column", wide.RGBAAt(678, 123), src.RGBAAt(464, 123)},
		{"right column is flush right", wide.RGBAAt(853, 123), src.RGBAAt(639, 123)},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: pixel = %v, want %v", tc.name, tc.got, tc.want)
		}
	}

	layout := newWideFrameLayout(wideW, wideFrameTownColumns)
	if got, ok := layout.wideToNative(image.Pt(600, 123)); !ok || got != image.Pt(410, 123) {
		t.Fatalf("stretched town presentation maps to %v,%v, want (410,123),true", got, ok)
	}
	if got, ok := layout.wideToNative(image.Pt(700, 123)); !ok || got != image.Pt(486, 123) {
		t.Fatalf("moved right column maps to %v,%v, want (486,123),true", got, ok)
	}
	for x := 0; x < frame.W; x++ {
		widePoint, ok := layout.nativeToWide(image.Pt(x, 123))
		if !ok {
			t.Fatalf("native town x=%d has no output pixel", x)
		}
		if got, ok := layout.wideToNative(widePoint); !ok || got != image.Pt(x, 123) {
			t.Fatalf("native town x=%d round trip = %v,%v", x, got, ok)
		}
	}
}

func TestTownColumnSharedBlitPlanHasNoGapOverlapOrDuplicateSeam(t *testing.T) {
	segments := newWideFrameLayout(854, wideFrameTownColumns).segments()
	if len(segments) != 2 {
		t.Fatalf("town blit segments = %d, want 2", len(segments))
	}
	if got, want := segments[0].Source, image.Rect(0, 0, 464, frame.H); got != want {
		t.Fatalf("presentation source = %v, want %v", got, want)
	}
	if got, want := segments[0].destination(), image.Rect(0, 0, 678, frame.H); got != want {
		t.Fatalf("presentation destination = %v, want %v", got, want)
	}
	if got, want := segments[1].Source, image.Rect(464, 0, frame.W, frame.H); got != want {
		t.Fatalf("right-column source = %v, want %v", got, want)
	}
	if got, want := segments[1].destination(), image.Rect(678, 0, 854, frame.H); got != want {
		t.Fatalf("right-column destination = %v, want %v", got, want)
	}
	if segments[0].destination().Max.X != segments[1].destination().Min.X ||
		segments[0].Source.Max.X != segments[1].Source.Min.X {
		t.Fatal("shared CPU/GPU blit plan leaves a gap, overlap, or duplicate seam")
	}
}

func TestCenteredWideFrameKeepsTheNativeSurfaceIntact(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	src.SetRGBA(0, 0, color.RGBA{R: 0x41, A: 0xff})
	src.SetRGBA(frame.W-1, 0, color.RGBA{B: 0x72, A: 0xff})

	wide := newWideFrameLayout(854, wideFrameCentered).compose(src)
	if got := wide.RGBAAt(107, 0); got != src.RGBAAt(0, 0) {
		t.Fatalf("centred native left pixel = %v, want %v", got, src.RGBAAt(0, 0))
	}
	if got := wide.RGBAAt(746, 0); got != src.RGBAAt(639, 0) {
		t.Fatalf("centred native right pixel = %v, want %v", got, src.RGBAAt(639, 0))
	}
	if got := wide.RGBAAt(0, 0); got != wideFrameFill {
		t.Fatalf("left extension = %v, want interface backing %v", got, wideFrameFill)
	}
	if got := wide.RGBAAt(853, 0); got != wideFrameFill {
		t.Fatalf("right extension = %v, want interface backing %v", got, wideFrameFill)
	}
}

func TestStretchedWideFrameFillsBothEdgesAndKeepsInputOnTheSamePixels(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	for x := 0; x < frame.W; x++ {
		src.SetRGBA(x, 73, color.RGBA{R: uint8(x), G: uint8(x >> 8), A: 0xff})
	}
	layout := newWideFrameLayout(854, wideFrameStretched)
	wide := layout.compose(src)
	for _, tc := range []struct {
		wideX   int
		nativeX int
	}{
		{0, 0},
		{427, 320},
		{853, 639},
	} {
		if got, want := wide.RGBAAt(tc.wideX, 73), src.RGBAAt(tc.nativeX, 73); got != want {
			t.Errorf("stretched x=%d = %v, want native x=%d %v", tc.wideX, got, tc.nativeX, want)
		}
	}
	for x := 0; x < frame.W; x++ {
		widePoint, ok := layout.nativeToWide(image.Pt(x, 73))
		if !ok {
			t.Fatalf("native x=%d has no stretched output pixel", x)
		}
		if got, ok := layout.wideToNative(widePoint); !ok || got != image.Pt(x, 73) {
			t.Fatalf("native x=%d round trip = %v,%v", x, got, ok)
		}
	}
}

func TestTownFamilyLayoutKeepsNative640AndLetterboxesWideWindow(t *testing.T) {
	a := newTestApp(t, appRows(3), okLoader(t))
	a.Layout(1920, 1080)
	if got := a.place.FrameSize(); got != image.Pt(640, 480) {
		t.Fatalf("1920x1080 town logical frame = %v, want (640,480)", got)
	}
	for _, p := range []image.Point{{0, 540}, {1919, 540}} {
		if _, ok := a.place.WindowToFrame(p.X, p.Y); ok {
			t.Errorf("window edge %v entered the native 4:3 frame", p)
		}
	}
	if got, ok := a.place.WindowToFrame(960, 540); !ok || got != image.Pt(320, 240) {
		t.Fatalf("wide-window centre maps to %v,%v, want (320,240),true", got, ok)
	}

	// A 4:3 resize returns to the native width without changing the height.
	a.Layout(1600, 1200)
	if got := a.place.FrameSize(); got != image.Pt(640, 480) {
		t.Fatalf("1600x1200 town logical frame = %v, want (640,480)", got)
	}
}

func TestNativeTownRightControlClicksAndInputRoundTripsInWideWindow(t *testing.T) {
	town := &fakeStatsSurfaceTown{}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the surface fixture")
	}
	a.Layout(1920, 1080)

	button := townSurfaceButtonRect(TownSurfaceTavern, 0)
	native := image.Pt((button.Min.X+button.Max.X)/2, (button.Min.Y+button.Max.Y)/2)
	wx, wy, ok := a.nativeFrameToWindow(native)
	if !ok {
		t.Fatal("moved right-column button has no window pixel")
	}
	now := time.Unix(1_700_000_000, 0)
	a.stepTownAt(appInput{CursorX: wx, CursorY: wy, PrimaryPressed: true}, now)
	a.stepTownAt(appInput{CursorX: wx, CursorY: wy, PrimaryReleased: true}, now)
	if want := (TownSurfaceControl{Kind: TownSurfaceControlButton, Index: 0}); len(town.surfaceClicks) != 1 || town.surfaceClicks[0] != want {
		t.Fatalf("letterboxed right-column click = %v, want [%v]", town.surfaceClicks, want)
	}

	presentationWindowX, presentationWindowY, ok := a.place.FrameToWindow(image.Pt(600, 100))
	if !ok {
		t.Fatal("native presentation point has no window pixel")
	}
	if got, ok := a.windowToNativeFrame(presentationWindowX, presentationWindowY); !ok || got != image.Pt(600, 100) {
		t.Fatalf("native presentation inverse = %v,%v, want (600,100),true", got, ok)
	}
}

func TestHeadlessFrameKeepsNativeTownAndShopGeometryInWideWindow(t *testing.T) {
	for _, tc := range []struct {
		name string
		town TownScreen
	}{
		{"tavern", &fakeStatsSurfaceTown{}},
		{"shop", &fakeShopTown{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestApp(t, appRows(3), okLoader(t))
			a.SetTown(tc.town)
			if !a.flow.showTown("") {
				t.Fatal("showTown refused fixture")
			}
			native, err := a.composeScreen()
			if err != nil {
				t.Fatal(err)
			}
			a.Layout(1920, 1080)
			letterboxed, _, err := a.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			if got := letterboxed.Bounds(); got != image.Rect(0, 0, 640, 480) {
				t.Fatalf("HeadlessFrame bounds = %v, want 640x480", got)
			}
			if !bytes.Equal(letterboxed.Pix, native.Pix) {
				t.Fatal("wide-window Layout changed the native town-family frame")
			}
		})
	}
}

func TestNativeShopDialogueStaysIntactInWideWindow(t *testing.T) {
	dialogue := image.NewRGBA(image.Rect(0, 0, 580, 240))
	dialogueColor := color.RGBA{R: 0xe7, G: 0x31, B: 0x92, A: 0xff}
	draw.Draw(dialogue, dialogue.Bounds(), &image.Uniform{C: dialogueColor}, image.Point{}, draw.Src)
	town := &fakeShopDialogueTown{pic: dialogue}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the dialogue-over-shop fixture")
	}
	a.Layout(1920, 1080)

	letterboxed, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	origin := image.Pt(30, 120) // (640-580)/2, (480-240)/2.
	for y := 0; y < dialogue.Bounds().Dy(); y++ {
		for x := 0; x < dialogue.Bounds().Dx(); x++ {
			if got := letterboxed.RGBAAt(origin.X+x, origin.Y+y); got != dialogueColor {
				t.Fatalf("dialogue split at local (%d,%d): got %v, want %v", x, y, got, dialogueColor)
			}
		}
	}

	// The fixture's local button centre is (140,75), hence native-frame centre
	// (170,195). The production inverse must keep that point unchanged.
	wx, wy, ok := a.place.FrameToWindow(image.Pt(170, 195))
	if !ok {
		t.Fatal("native dialogue button has no window pixel")
	}
	if got, ok := a.windowToNativeFrame(wx, wy); !ok || got != image.Pt(170, 195) {
		t.Fatalf("native dialogue button inverse = %v,%v, want (170,195),true", got, ok)
	}
}

func TestNativeTownCanvasIsReusedAcrossWindowResizes(t *testing.T) {
	a := newTestApp(t, appRows(3), okLoader(t))
	a.canvas = ebiten.NewImage(frame.W, frame.H)
	a.Layout(1920, 1080)
	first := a.prepareWideCanvas(a.baseWideFrameLayout())
	second := a.prepareWideCanvas(a.baseWideFrameLayout())
	if first != second {
		t.Fatal("equal-size frames replaced the wide canvas")
	}

	a.Layout(1600, 1200)
	if a.wideCanvas != first {
		t.Fatal("window resize replaced the fixed native town canvas")
	}
	third := a.prepareWideCanvas(a.baseWideFrameLayout())
	if third != first || third.Bounds().Size() != image.Pt(640, 480) {
		t.Fatalf("resized canvas = %p %v, want reused 640x480 image", third, third.Bounds().Size())
	}
}
