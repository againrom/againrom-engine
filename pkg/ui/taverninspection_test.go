package ui

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/render/text"
)

type fakeTavernInspectionTown struct {
	fakeTown
	view         TownSurfaceView
	clicks       []TownSurfaceControl
	surfaceCalls int
}

func (f *fakeTavernInspectionTown) AtTownSurface() bool { return true }
func (f *fakeTavernInspectionTown) TownSurface() TownSurfaceView {
	f.surfaceCalls++
	return f.view
}
func (f *fakeTavernInspectionTown) TownSurfaceClick(c TownSurfaceControl, _ bool) TownAction {
	f.clicks = append(f.clicks, c)
	return TownAction{}
}

func TestTavernAnimationClockDoesNotBuildTheSurfaceModel(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind TownSurfaceKind
	}{{"tavern", TownSurfaceTavern}, {"school", TownSurfaceSchool}} {
		t.Run(tc.name, func(t *testing.T) {
			town := &fakeTavernInspectionTown{view: TownSurfaceView{Kind: tc.kind}}
			a := newTestApp(t, appRows(1), nil)
			a.SetTown(town)
			if !a.flow.showTown("") {
				t.Fatal("showTown refused shared town surface")
			}
			town.surfaceCalls = 0
			now := time.Unix(1_700_000_000, 0)
			a.step(appInput{}, now)
			if town.surfaceCalls != 1 {
				t.Fatalf("neutral Update built TownSurface %d times, want only the input snapshot", town.surfaceCalls)
			}
			if _, err := a.composeTownRoom(); err != nil {
				t.Fatal(err)
			}
			if town.surfaceCalls != 2 {
				t.Fatalf("Update plus Draw built TownSurface %d times, want 2", town.surfaceCalls)
			}
		})
	}
}

func TestComposeTownSurfaceUsesTheCachedCandidatePixels(t *testing.T) {
	font := townShellRosterTestFont()
	figure := image.NewRGBA(image.Rect(0, 0, 4, 4))
	figure.SetRGBA(0, 0, color.RGBA{R: 0x71, A: 0xff})
	art := &TownTavernArt{LeftStats: uniform(160, 238, color.RGBA{G: 0x22, A: 0xff})}
	candidate := TownCharacterView{HasSubject: true, Font: font, CardFont: font, Figure: figure}

	plain := TownSurfaceView{Kind: TownSurfaceTavern, TavernArt: art, Candidate: candidate}
	want := ComposeTownSurface(plain)
	cached := RenderTownCandidateInspection(candidate, art)
	if cached == nil {
		t.Fatal("candidate inspection cache was not rendered")
	}
	plain.CandidatePixels = cached
	got := ComposeTownSurface(plain)
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Fatal("cached candidate inspection differs from the direct production composition")
	}

	marker := image.NewRGBA(image.Rect(0, 0, 160, 480))
	marker.SetRGBA(0, 0, color.RGBA{R: 0xa1, G: 0xb2, B: 0xc3, A: 0xff})
	marker.SetRGBA(159, 479, color.RGBA{R: 0xd4, G: 0xe5, B: 0xf6, A: 0xff})
	plain.CandidatePixels = marker
	got = ComposeTownSurface(plain)
	if got.RGBAAt(0, 0) != marker.RGBAAt(0, 0) || got.RGBAAt(159, 479) != marker.RGBAAt(159, 479) {
		t.Fatal("ComposeTownSurface ignored the cached candidate pixels")
	}
}

// TestCachedCandidateTextIsCapturedEveryFrame is the owner's tavern defect:
// the selected mercenary's statistics showed for one frame and vanished,
// because the cached inspection was captured only on the frame that drew it.
// Its recorded glyphs now re-enter every frame's capture window.
func TestCachedCandidateTextIsCapturedEveryFrame(t *testing.T) {
	font := townShellRosterTestFont()
	art := &TownTavernArt{LeftStats: uniform(160, 238, color.RGBA{G: 0x22, A: 0xff})}
	candidate := TownCharacterView{HasSubject: true, Font: font, CardFont: font, Subject: PanelSubject{Name: "A"}}
	view := TownSurfaceView{Kind: TownSurfaceTavern, TavernArt: art, Candidate: candidate}
	view.CandidateText = text.Record(func() { view.CandidatePixels = RenderTownCandidateInspection(candidate, art) })
	if len(view.CandidateText) == 0 {
		t.Fatal("the inspection drew no glyph to record")
	}
	for frame := 0; frame < 2; frame++ {
		text.ResetCapture()
		text.SetCapture(false)
		ComposeTownSurface(view)
		text.StopCapture()
		if got := text.CapturedLen(); got < len(view.CandidateText) {
			t.Fatalf("frame %d captured %d glyphs, want the cached inspection's %d", frame, got, len(view.CandidateText))
		}
	}
	text.ResetCapture()
}

func TestTalkOnlyTavernCandidateRendersItsFigureWithoutInventedStats(t *testing.T) {
	figure := image.NewRGBA(image.Rect(0, 0, 4, 4))
	mark := color.RGBA{R: 0x91, G: 0x42, B: 0x73, A: 0xff}
	figure.SetRGBA(0, 0, mark)
	art := &TownTavernArt{
		LeftStats:   uniform(160, 238, color.RGBA{R: 0x11, A: 0xff}),
		LeftPicture: uniform(160, 242, color.RGBA{G: 0x22, A: 0xff}),
	}
	candidate := TownCharacterView{Figure: figure}
	pixels := RenderTownCandidateInspection(candidate, art)
	if pixels == nil {
		t.Fatal("talk-only candidate figure produced no inspection pixels")
	}
	if got := pixels.RGBAAt(86, 358); got != mark {
		t.Fatalf("talk-only figure pixel = %#v, want %#v", got, mark)
	}
	if got := pixels.RGBAAt(0, 0); got.A != 0 {
		t.Fatalf("talk-only upper pane was painted %#v; no invented statistics are allowed", got)
	}
}

func TestAppTavernClockAdvancesOnlyTheSelectedMiniature(t *testing.T) {
	first := uniform(48, 64, color.RGBA{R: 0x91, A: 0xff})
	second := uniform(48, 64, color.RGBA{G: 0xa2, A: 0xff})
	town := &fakeTavernInspectionTown{view: TownSurfaceView{
		Kind:      TownSurfaceTavern,
		TavernArt: &TownTavernArt{ManBack: uniform(48, 64, color.RGBA{B: 0x13, A: 0xff})},
		Cells: []TownSurfaceCell{
			{Portrait: true, Selected: true, Frames: []image.Image{first, second}},
			{Portrait: true, Frames: []image.Image{first, second}},
		},
	}}
	a := newTestApp(t, appRows(1), nil)
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused tavern surface")
	}
	now := time.Unix(1_700_000_000, 0)
	for i := 0; i < 6; i++ {
		a.step(appInput{}, now.Add(time.Duration(i)*time.Millisecond))
	}
	frame, err := a.composeTownRoom()
	if err != nil {
		t.Fatal(err)
	}
	if got := frame.RGBAAt(176, 416); got.G != 0xa2 {
		t.Fatalf("selected app-clock pixel = %#v, want second frame", got)
	}
	if got := frame.RGBAAt(224, 416); got.R != 0x91 {
		t.Fatalf("unselected app-clock pixel = %#v, want frozen first frame", got)
	}
	a.flow.screen = ScreenMenu
	a.step(appInput{}, now.Add(time.Second))
	if a.townSurfaceAnimationTick != 0 {
		t.Fatalf("leaving the tavern retained presentation tick %d", a.townSurfaceAnimationTick)
	}
}

func TestAppCandidateDollHoverClickAndDragAreReadOnly(t *testing.T) {
	mask := &SlotMask{W: 2, H: 2, Slot: []uint8{1, 1, 1, 1}}
	figure := image.NewRGBA(image.Rect(0, 0, 2, 2))
	figure.SetRGBA(0, 0, color.RGBA{R: 0xff, A: 0xff})
	town := &fakeTavernInspectionTown{view: TownSurfaceView{
		Kind:              TownSurfaceTavern,
		Font:              townShellRosterTestFont(),
		TavernArt:         &TownTavernArt{},
		Candidate:         TownCharacterView{HasSubject: true, Figure: figure, Font: townShellRosterTestFont()},
		CandidateSlotMask: mask,
		CandidateSlotInfo: [12][]string{0: {"Sword", "#Damage 3"}},
	}}
	a := newTestApp(t, appRows(1), nil)
	a.SetTooltipDelayPreference(0, nil)
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused tavern surface")
	}
	before := town.view
	now := time.Unix(1_700_000_000, 0)
	occupied := image.Pt(87, 359)
	a.step(appInput{CursorX: occupied.X, CursorY: occupied.Y}, now)
	hover, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	a.step(appInput{CursorX: -1, CursorY: -1}, now)
	bare, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(hover.Pix, bare.Pix) {
		t.Fatal("pointer motion over worn item did not draw its tooltip")
	}

	// Press, move beyond tap slop, release, then repeat as a click. The
	// candidate pane is neither a TownSurfaceControl nor a shop drag family.
	a.step(appInput{CursorX: occupied.X, CursorY: occupied.Y, PrimaryPressed: true}, now.Add(time.Millisecond))
	a.step(appInput{CursorX: 120, CursorY: 420}, now.Add(2*time.Millisecond))
	a.step(appInput{CursorX: 120, CursorY: 420, PrimaryReleased: true}, now.Add(3*time.Millisecond))
	clickAt(a, now.Add(4*time.Millisecond), occupied)
	if len(town.clicks) != 0 {
		t.Fatalf("candidate doll dispatched surface controls: %+v", town.clicks)
	}
	if a.shopDragArmed || a.shopDragMoved != 0 || a.shopDragOrigin != (ShopControl{}) {
		t.Fatalf("candidate doll armed shop drag: armed=%v moved=%d origin=%+v", a.shopDragArmed, a.shopDragMoved, a.shopDragOrigin)
	}
	if !reflect.DeepEqual(town.view, before) {
		t.Fatal("candidate doll input mutated its production view")
	}
}

func TestTavernCandidateCardStartsTwelvePixelsRight(t *testing.T) {
	font := townShellRosterTestFont()
	edge := color.RGBA{R: 0xc8, A: 0xff}
	body := uniform(160, 238, characterCardFill)
	for y := 0; y < 238; y++ {
		for x := 0; x < 12; x++ {
			body.SetRGBA(x, y, edge)
		}
	}
	seam := uniform(16, 238, edge)
	art := &TownTavernArt{LeftStats: body, LeftStatsSeam: seam}
	candidate := TownCharacterView{HasSubject: true, Font: font, CardFont: font, Subject: PanelSubject{Name: "Mercenary"}}

	view := tavernCandidateStats(candidate, art)
	if got, want := view.cardRect(), image.Rect(12, 0, 172, 238); got != want {
		t.Fatalf("card rectangle = %v, want %v", got, want)
	}
	bg := shiftedCardBackground(body, seam, 12)
	if bg.RGBAAt(0, 5) != characterCardFill || bg.RGBAAt(147, 5) != characterCardFill || bg.RGBAAt(148, 5) != edge || bg.RGBAAt(159, 5) != edge {
		t.Fatalf("shifted board columns 0,147,148,159 = %v %v %v %v", bg.RGBAAt(0, 5), bg.RGBAAt(147, 5), bg.RGBAAt(148, 5), bg.RGBAAt(159, 5))
	}

	got := RenderTownCandidateInspection(candidate, art)
	if got == nil || got.Bounds().Dx() != 176 {
		t.Fatalf("inspection = %v, want 176 columns", got)
	}
	for y := 0; y < 238; y++ {
		for x := 0; x < 12; x++ {
			if got.RGBAAt(x, y) != edge {
				t.Fatalf("pixel (%d,%d) = %v left of the card canvas, want the board edge", x, y, got.RGBAAt(x, y))
			}
		}
	}
	lit := false
	for y := 0; y < 238 && !lit; y++ {
		for x := 12; x < 160; x++ {
			if c := got.RGBAAt(x, y); c != characterCardFill && c != edge && c.A != 0 {
				lit = true
				break
			}
		}
	}
	if !lit {
		t.Fatal("the card drew no text right of its canvas origin")
	}
}
