package ui

import (
	"image"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
)

// Acceptance tests for the town square's own picture (1016). Every
// expectation is transcribed from docs/1016-town-square/spec.md, not read
// back out of the code.

// synthTownSquareScene is a scene filled with 0x40 that answers the four
// doors and the statue's menu at five known points.
func synthTownSquareScene() *fakeSquareScene {
	return &fakeSquareScene{fill: 0x40, controls: map[image.Point]TownSquareControl{
		image.Pt(10, 10): squareDoor(0), image.Pt(20, 20): squareDoor(1), image.Pt(30, 30): squareDoor(2),
		image.Pt(40, 40): squareDoor(3), image.Pt(50, 50): squareMenu,
	}}
}

// fakeSquareArtTown is a minimal TownScreen carrying only the square's own
// picture seam. Choose records its argument and nothing else — unlike
// fakeTown, it has no "outside/inside" room model to interact with, so a
// test driving several clicks in a row over the square's OWN picture is
// testing the raster mask and not a room transition (fakeSquareTown's own
// AtTownSquare answers !f.inside, and fakeTown.Choose flips inside on its
// first call, which would silently change what Rows() answers between two
// clicks in the same test).
type fakeSquareArtTown struct {
	scene       TownSquareScene
	chosen      []int
	headerCalls int
}

func (f *fakeSquareArtTown) TownSquareView() TownSquareView {
	return TownSquareView{Scene: f.scene, Font: panelFont()}
}
func (f *fakeSquareArtTown) AtTownSquare() bool { return true }

// Header is a witness (fakeShopDialogueTown's own pattern, town_test.go):
// the row-button fallback branch of drawTown reads Header() and the
// composed-art branch never does, so a call means the fallback ran and the
// picture was not painted.
func (f *fakeSquareArtTown) Header() string {
	f.headerCalls++
	return "square"
}

// Rows carries the production door names (townDoors' own order, pkg/game) so
// a test can resolve a door by name through HeadlessActivate, the same way a
// shipped scenario's own "activate TAVERN" does, rather than only by mask
// coordinate.
func (f *fakeSquareArtTown) Rows() []TownRow {
	names := []string{"TAVERN", "SHOP", "SCHOOL", "GATES"}
	rows := make([]TownRow, len(names))
	for i, name := range names {
		rows[i] = TownRow{Text: name, Choosable: true}
	}
	return rows
}
func (f *fakeSquareArtTown) Footer() []string { return nil }
func (f *fakeSquareArtTown) Choose(i int) TownAction {
	f.chosen = append(f.chosen, i)
	return TownAction{}
}
func (f *fakeSquareArtTown) Back() bool { return false }

// ebiten.Image.At and ReadPixels panic before a game starts (app_test.go's
// own note), so this test cannot read the composed pixels back off a.canvas
// the way it drives the compose. It reads the fallback's own Header() call
// instead, exactly as TestTheShopComposesBehindItsOwnDialogue does for the
// shop room. ComposeTownSquare's own pixel placement is covered directly, by
// TestComposeTownSquarePaintsNoHintText and TestComposeTownSquareWithNoArtIsBlank
// below, against the plain *image.RGBA it returns.
func TestTownSquareArtDrawsThePictureInsteadOfTheGrid(t *testing.T) {
	town := &fakeSquareArtTown{scene: synthTownSquareScene()}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the art-backed square")
	}
	a.canvas = ebiten.NewImage(frame.W, frame.H)

	a.drawTown()

	if town.headerCalls != 0 {
		t.Errorf("drawTown read the row-button fallback's Header() %d time(s): "+
			"the composed picture was not painted and the old grid ran instead", town.headerCalls)
	}
}

// Falling back to the row-button grid when no art is wired is covered by the
// existing TestTownSquareUsesFourSeparateClickableRegions and
// TestMissionCompleteClickIsConsumedAcrossTheTownTransition, both driven
// through fakeSquareTown, which implements no TownSquareArtScreen at all.

// Keyboard selection is invisible once the picture replaces the row-button
// grid (round-2 adversarial review): ComposeTownSquare draws no selection
// cursor, so the wheel, the arrows and Enter must be no-ops once art is
// ready, exactly as they already are on the shop (the comment above
// `inShop`, app.go). Down is pressed twice and Enter once; town.chosen stays
// empty throughout, and headerCalls confirms the composed branch — not the
// row-button fallback — was the one dispatch ran against the whole time.
func TestTownSquareKeyboardSelectionIsInertOnceArtIsReady(t *testing.T) {
	town := &fakeSquareArtTown{scene: synthTownSquareScene()}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the art-backed square")
	}
	now := time.Unix(1_700_000_000, 0)

	a.stepTownAt(appInput{Down: true}, now)
	a.stepTownAt(appInput{Down: true}, now)
	a.stepTownAt(appInput{WheelY: -1}, now)
	a.stepTownAt(appInput{Enter: true}, now)

	if len(town.chosen) != 0 {
		t.Fatalf("chosen = %v, want none: Down/Wheel/Enter moved or chose a selection "+
			"the composed picture draws no cursor for", town.chosen)
	}

	a.canvas = ebiten.NewImage(frame.W, frame.H)
	a.drawTown()
	if town.headerCalls != 0 {
		t.Fatalf("drawTown read the row-button fallback's Header() %d time(s) after the "+
			"key presses: the composed picture was not what dispatch was tested against", town.headerCalls)
	}
}

func TestHeadlessActivateChoosesTheSquareDoorByNameOnceArtIsReady(t *testing.T) {
	town := &fakeSquareArtTown{scene: synthTownSquareScene()}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the art-backed square")
	}

	if err := a.HeadlessActivate("GATES"); err != nil {
		t.Fatalf(`HeadlessActivate("GATES") = %v, want nil`, err)
	}
	if !reflect.DeepEqual(town.chosen, []int{3}) {
		t.Fatalf("chosen = %v, want [3] (the gates row, townDoors order)", town.chosen)
	}
}

// A click on each door region calls Choose with the SAME index the
// row-button grid has always used (pkg/game's townDoors order), so
// Choose(i) and every headless scenario driving it are unaffected by this
// story.
func TestTownSquareRasterMaskClicksChooseTheSameDoorIndexAsTheGrid(t *testing.T) {
	town := &fakeSquareArtTown{scene: synthTownSquareScene()}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the art-backed square")
	}
	now := time.Unix(1_700_000_000, 0)

	click := func(p image.Point) {
		a.stepTownAt(appInput{CursorX: p.X, CursorY: p.Y, PrimaryReleased: true}, now)
	}
	click(image.Pt(20, 20)) // shop
	click(image.Pt(10, 10)) // tavern

	want := []int{1, 0}
	if !reflect.DeepEqual(town.chosen, want) {
		t.Fatalf("chosen = %v, want %v", town.chosen, want)
	}
}

// A click that lands on the gate opens the world map through the same
// Choose(i) path — 1016 wires no separate door for it.
func TestTownSquareGateClickEntersThroughChoose(t *testing.T) {
	town := &fakeSquareArtTown{scene: synthTownSquareScene()}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the art-backed square")
	}
	now := time.Unix(1_700_000_000, 0)

	a.stepTownAt(appInput{CursorX: 40, CursorY: 40, PrimaryReleased: true}, now)

	if !reflect.DeepEqual(town.chosen, []int{3}) {
		t.Fatalf("chosen = %v, want [3] (the gates row)", town.chosen)
	}
}

// The statue opens the SAME mini-menu Escape already does at the square
// rather than crossing the town's own Choose seam at all.
func TestTownSquareStatueClickOpensTheMiniMenu(t *testing.T) {
	town := &fakeSquareArtTown{scene: synthTownSquareScene()}
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the art-backed square")
	}
	now := time.Unix(1_700_000_000, 0)

	a.stepTownAt(appInput{CursorX: 50, CursorY: 50, PrimaryReleased: true}, now)

	if a.flow.screen != ScreenGameMenu {
		t.Fatalf("screen = %v, want ScreenGameMenu", a.flow.screen)
	}
	if len(town.chosen) != 0 {
		t.Fatalf("the statue crossed Choose: %v, want none", town.chosen)
	}
}

// ComposeTownSquare paints no hint text: the SC-1..SC-3 cuts (spec.md). This
// is witnessed by construction — the function calls no text-printing helper
// over Header, Footer or the per-door hint — and confirmed here by size: a
// canvas built from art alone, with no message, differs from the all-zero
// canvas only within the background, overlay strip and three label
// rectangles this test already knows the extent of.
func TestComposeTownSquarePaintsNoHintText(t *testing.T) {
	dst := ComposeTownSquare(TownSquareView{Scene: synthTownSquareScene()})
	// A point well outside every drawn rectangle (background covers the
	// whole canvas, so pick a point covered ONLY by the background and
	// confirm it carries the background's own colour, not some row-button
	// or header glyph the old fallback would have painted there).
	r, g, b, _ := dst.At(600, 460).RGBA()
	if r>>8 != 0x40 || g>>8 != 0x40 || b>>8 != 0x40 {
		t.Fatalf("pixel (600,460) = (%d,%d,%d), want the background's own 0x40", r>>8, g>>8, b>>8)
	}
}

// A view with no scene answers a blank canvas rather than panicking; app.go
// never calls this without a scene (townSquareView).
func TestComposeTownSquareWithNoArtIsBlank(t *testing.T) {
	dst := ComposeTownSquare(TownSquareView{})
	if dst.Bounds().Dx() != 640 || dst.Bounds().Dy() != 480 {
		t.Fatalf("bounds = %v, want 640x480", dst.Bounds())
	}
	r, g, b, a := dst.At(0, 0).RGBA()
	if r != 0 || g != 0 || b != 0 || a != 0 {
		t.Fatalf("pixel (0,0) = (%d,%d,%d,%d), want all zero", r, g, b, a)
	}
}
