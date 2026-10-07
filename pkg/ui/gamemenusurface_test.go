package ui

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
)

// The in-game menu's two surfaces, their geometry, their accelerators and the
// freeze behind them (0158).

// openMissionMenu is a campaign flow with save and sound seams installed and
// the menu up over it, so every campaign destination can be exercised.
func openMissionMenu(t *testing.T) *flow {
	t.Helper()
	f := newFlow(NewPicker(nil), nil)
	f.saveGame = func(bool) (string, error) { return "s.ags", nil }
	f.saveList = func() []SaveEntry { return []SaveEntry{{Name: "s.ags", Label: "a save"}} }
	f.menuSound = func() (bool, int, bool) { return true, 100, true }
	f.setMenuSound = func(bool, int) error { return nil }
	f.screen = ScreenMap
	f.openGameMenu(ScreenMap)
	return f
}

func openTownMenu(t *testing.T) *flow {
	t.Helper()
	f := newFlow(NewPicker(nil), nil)
	f.saveGame = func(bool) (string, error) { return "s.ags", nil }
	f.saveList = func() []SaveEntry { return []SaveEntry{{Name: "s.ags", Label: "a save"}} }
	f.town = &stubTown{rows: []TownRow{{Text: "the square", Choosable: true}}}
	f.openGameMenu(ScreenTown)
	return f
}

// TestTheTwoSurfacesAreTheDecodedListsInTheDecodedOrder is AC-6 and AC-7.
//
// THE ORDER IS ASSERTED BY POSITION AND NOT BY MEMBERSHIP, because the order is
// what MENU-ITEM-011 and MENU-ITEM-012 establish and a set test would pass on a
// surface with the rows shuffled.
func TestTheTwoSurfacesAreTheDecodedListsInTheDecodedOrder(t *testing.T) {
	mission := []gameMenuAction{
		gameMenuSave, gameMenuLoad, gameMenuGameOptions, gameMenuSoundOptions,
		gameMenuQuestObjectives, gameMenuEndQuest, gameMenuReturn,
	}
	town := []gameMenuAction{
		gameMenuSave, gameMenuLoad, gameMenuGameOptions, gameMenuSoundOptions, gameMenuAbortGame, gameMenuReturn,
	}
	for _, tc := range []struct {
		name string
		open func(*testing.T) *flow
		want []gameMenuAction
	}{
		{"the mission surface", openMissionMenu, mission},
		{"the town surface", openTownMenu, town},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.open(t)
			rows := f.menuRows()
			if len(rows) != len(tc.want) {
				t.Fatalf("the surface has %d rows, want exactly %d", len(rows), len(tc.want))
			}
			for i, want := range tc.want {
				if rows[i].Action != want {
					t.Errorf("row %d does action %d, want %d", i, rows[i].Action, want)
				}
			}
			if n := f.menuList.Len(); n != len(tc.want) {
				t.Errorf("the drawn list shows %d rows, want %d", n, len(tc.want))
			}
		})
	}
}

// TestTheThreeDestinationsAreReachable keeps a row from regressing to the
// disabled/no-op state DIV-099 recorded.
func TestTheThreeDestinationsAreReachable(t *testing.T) {
	f := openMissionMenu(t)
	for _, a := range []gameMenuAction{gameMenuGameOptions, gameMenuSoundOptions, gameMenuQuestObjectives} {
		f.rebuildGameMenu(gameMenuRoot, 0)
		if !selectGameMenuAction(f, a) {
			t.Fatalf("the mission surface has no row for action %d", a)
		}
		i := f.menuList.Selection()
		if !f.menuList.Rows()[i].Choosable {
			t.Errorf("row %d (action %d) is disabled despite having a destination", i, a)
		}
		f.chooseGameMenu()
		if f.screen != ScreenGameMenu || f.menuPage == gameMenuRoot {
			t.Fatalf("action %d did not reach a nested menu page", a)
		}
	}
}

// TestTheDecodedRectangles is AC-8 and plan SC-3. The numbers are written out
// here rather than taken from the layout helpers, so this test compares the
// code against the decode rather than against itself.
func TestTheDecodedRectangles(t *testing.T) {
	if got, want := gameMenuPanelRect(gameMenuMission), image.Rect(100, 60, 440, 400); got != want {
		t.Errorf("the mission panel is %v, want %v", got, want)
	}
	if got, want := gameMenuPanelRect(gameMenuTown), image.Rect(100, 100, 440, 340); got != want {
		t.Errorf("the town panel is %v, want %v", got, want)
	}
	// Row 0 of the mission panel: 40 in from the left, 40 below the panel top,
	// 252 wide, 30 high.
	if got, want := gameMenuRowRect(gameMenuMission, 0), image.Rect(140, 100, 392, 130); got != want {
		t.Errorf("mission row 0 is %v, want %v", got, want)
	}
	if got, want := gameMenuRowRect(gameMenuTown, 0), image.Rect(140, 140, 392, 170); got != want {
		t.Errorf("town row 0 is %v, want %v", got, want)
	}
	for _, s := range []gameMenuSurface{gameMenuMission, gameMenuTown} {
		panel := gameMenuPanelRect(s)
		for n := 0; n < 7; n++ {
			r := gameMenuRowRect(s, n)
			if r.Dx() != 252 || r.Dy() != 30 {
				t.Errorf("surface %d row %d is %dx%d, want 252x30", s, n, r.Dx(), r.Dy())
			}
			if n > 0 && r.Min.Y-gameMenuRowRect(s, n-1).Min.Y != 30 {
				t.Errorf("surface %d row %d does not sit 30 below row %d", s, n, n-1)
			}
			if !r.In(panel) && n < 5 {
				t.Errorf("surface %d row %d at %v is outside its panel %v", s, n, r, panel)
			}
		}
	}
	// Every row a surface actually lists stands inside its own panel.
	for _, tc := range []struct {
		s    gameMenuSurface
		rows int
	}{{gameMenuMission, 7}, {gameMenuTown, 5}} {
		panel := gameMenuPanelRect(tc.s)
		for n := 0; n < tc.rows; n++ {
			if r := gameMenuRowRect(tc.s, n); !r.In(panel) {
				t.Errorf("surface %d row %d at %v leaves its panel %v", tc.s, n, r, panel)
			}
		}
	}
}

// TestTheAcceleratorComesFromTheLabel is AC-9 and AC-10, and it is
// MENU-KEY-013's own content: the label wins over the row's immediate.
func TestTheAcceleratorComesFromTheLabel(t *testing.T) {
	for _, tc := range []struct {
		label    string
		fallback byte
		want     byte
	}{
		{"~SAVE GAME", 'S', 's'},
		{"SOU~ND OPTIONS", 'N', 'n'},
		{"GAME ~OPTIONS", 'O', 'o'},
		// The decoded residue: the immediate is M, the label marks Q, and Q is
		// what the release ships.
		{"~QUEST OBJECTIVES", 'M', 'q'},
		// The two English labels with no mark keep their immediate.
		{"ABORT GAME", 'E', 'e'},
		{"DIPLOMACY", 'D', 'd'},
		// `~~` is an escape and marks nothing.
		{"A~~B", 'Z', 'z'},
		// A lone trailing `~` marks nothing either.
		{"TRAIL~", 'Z', 'z'},
	} {
		if got := gameMenuAccelerator(tc.label, tc.fallback, 0); got != tc.want {
			t.Errorf("%q with fallback %q resolves %q, want %q",
				tc.label, tc.fallback, got, tc.want)
		}
	}
}

// TestTheDrawnLabelCarriesNoMark is AC-15, and it also fixes the column the
// paint underlines.
func TestTheDrawnLabelCarriesNoMark(t *testing.T) {
	for _, tc := range []struct {
		label string
		text  string
		col   int
	}{
		{"~SAVE GAME", "SAVE GAME", 0},
		{"SOU~ND OPTIONS", "SOUND OPTIONS", 3},
		{"GAME ~OPTIONS", "GAME OPTIONS", 5},
		{"ABORT GAME", "ABORT GAME", -1},
		{"A~~B", "A~B", -1},
		{"TRAIL~", "TRAIL", -1},
	} {
		if got := gameMenuLabelText(tc.label); got != tc.text {
			t.Errorf("%q draws as %q, want %q", tc.label, got, tc.text)
		}
		if got := gameMenuAcceleratorColumn(tc.label); got != tc.col {
			t.Errorf("%q underlines column %d, want %d", tc.label, got, tc.col)
		}
		if tc.col >= 0 {
			text := gameMenuLabelText(tc.label)
			if tc.col >= len(text) {
				t.Fatalf("%q underlines column %d, past the end of %q", tc.label, tc.col, text)
			}
			if lowerASCII(text[tc.col]) != gameMenuAccelerator(tc.label, 0, 0) {
				t.Errorf("%q underlines %q, which is not its accelerator %q",
					tc.label, text[tc.col], gameMenuAccelerator(tc.label, 0, 0))
			}
		}
	}
}

func TestEverySurfacesAcceleratorsAreDistinct(t *testing.T) {
	for _, tc := range []struct {
		name string
		open func(*testing.T) *flow
		want string
	}{
		{"the mission surface", openMissionMenu, "slonqer"},
		{"the town surface", openTownMenu, "sloner"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []byte
			seen := map[byte]bool{}
			for _, row := range tc.open(t).menuRows() {
				c := gameMenuAccelerator(row.Label, row.Fallback, 0)
				if seen[c] {
					t.Errorf("two rows answer to %q", c)
				}
				seen[c] = true
				got = append(got, c)
			}
			if string(got) != tc.want {
				t.Errorf("accelerators are %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTheAcceleratorChoosesItsRow(t *testing.T) {
	f := openMissionMenu(t)
	if !f.chooseGameMenuAccelerator('R') {
		t.Fatal("upper-case R found no row")
	}
	if f.screen != ScreenMap {
		t.Fatalf("R landed on %v, want the map screen behind the menu", f.screen)
	}

	f = openMissionMenu(t)
	if !f.chooseGameMenuAccelerator('o') {
		t.Fatal("o found no row")
	}
	if f.screen != ScreenGameMenu || f.menuPage != gameMenuGameOptionsPage {
		t.Fatalf("GAME OPTIONS accelerator reached screen %v page %v", f.screen, f.menuPage)
	}
	if f.chooseGameMenuAccelerator('z') {
		t.Error("z claimed to find a row")
	}
}

// DIV-006
func TestRUKeyboardAcceleratorReachesItsLabel(t *testing.T) {
	f := openMissionMenu(t)
	f.words.MenuSave = string([]byte{0x7e, 0x91, 0xae, 0xe5, 0xe0, 0xa0, 0xad,
		0xa8, 0xe2, 0xec, 0x20, 0xa8, 0xa3, 0xe0, 0xe3})
	f.menuFont = &text.Font{Selector: text.SelectorConverting}
	f.encodeMenuKey = func(r rune) (byte, bool) { return textinput.EncodeRune(r, text.SelectorConverting) }

	// Lowercase 'с' (U+0441) is already the CP866 byte the label folds to and
	// needs no further fold; uppercase 'С' (U+0421) needs gameMenuLower's own
	// +0x50 to meet it. Both reach the same row.
	if !f.chooseGameMenuAccelerator('с') {
		t.Fatal("lowercase с found no row")
	}
	if f.msg != "Your character is saved" {
		t.Fatalf("lowercase с did not choose Save: msg = %q", f.msg)
	}
	f.msg = ""
	if !f.chooseGameMenuAccelerator('С') {
		t.Fatal("uppercase С found no row")
	}
	if f.msg != "Your character is saved" {
		t.Fatalf("uppercase С did not choose Save: msg = %q", f.msg)
	}
}

// TestRUTownAcceleratorCollisionFirstMatchWins is DIV-141: the town surface's
// own RU labels for Load and Abort both mark CP866 byte 0x82 (uppercase В,
// MENU-KEY-013 evidence), so a key that resolves to it can only ever reach
// the first row in scan order. This pins that resolution rather than leaving
// it implicit, so a later change to row order or to the scan itself shows up
// here.
func TestRUTownAcceleratorCollisionFirstMatchWins(t *testing.T) {
	f := openTownMenu(t)
	// "~Восстановить игру" (Load, row 1) and "~Выход из игры" (Abort, row 3),
	// both marking byte 0x82.
	f.words.MenuLoad = string([]byte{0x7e, 0x82, 0xae, 0xe1, 0xe1, 0xe2, 0xa0,
		0xad, 0xae, 0xa2, 0xa8, 0xe2, 0xec, 0x20, 0xa8, 0xa3, 0xe0, 0xe3})
	f.words.MenuAbort = string([]byte{0x7e, 0x82, 0xeb, 0xe5, 0xae, 0xa4, 0x20,
		0xa8, 0xa7, 0x20, 0xa8, 0xa3, 0xe0, 0xeb})
	f.menuFont = &text.Font{Selector: text.SelectorConverting}
	f.encodeMenuKey = func(r rune) (byte, bool) { return textinput.EncodeRune(r, text.SelectorConverting) }

	if !f.chooseGameMenuAccelerator('в') { // 'в' lowercase
		t.Fatal("в found no row")
	}
	if f.screen != ScreenLoad {
		t.Fatalf("в landed on %v, want ScreenLoad (Load is the first match in row order)", f.screen)
	}
}

func TestTheMenuStopsTheWorldAndDarkensTheMapBehindIt(t *testing.T) {
	v, err := NewViewer("m", grid(60, 60), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetNoticeBackdrop(AuthoredNoticeBackdrop())
	layoutViewport(v, frame.W, frame.H)

	f := newFlow(NewPicker(nil), nil)
	f.viewer = v
	f.screen = ScreenMap

	if f.popupOpen() {
		t.Fatal("a map screen with no menu already reports a popup")
	}
	if _, _, ok := f.viewer.noticeBackdropOf(); ok {
		t.Fatal("the map is darkened with no menu up")
	}

	f.openGameMenu(ScreenMap)
	if !f.popupOpen() {
		t.Fatal("the menu is up over the map and no popup is reported")
	}
	if _, c, ok := f.viewer.noticeBackdropOf(); !ok {
		t.Error("the menu is up and the map behind it is not darkened")
	} else if c != AuthoredNoticeBackdrop() {
		t.Errorf("the dim is %v, want the authored value %v", c, AuthoredNoticeBackdrop())
	}

	f.closeGameMenu()
	if f.popupOpen() {
		t.Fatal("the menu closed and the popup answer stayed true")
	}
	if _, _, ok := f.viewer.noticeBackdropOf(); ok {
		t.Error("the menu closed and the map is still darkened")
	}
}

// TestTheTownMenuRaisesNoMapPopup is the other half of the same answer: the
// town has no viewer, so nothing about a map may become true when the panel
// goes up over it.
func TestTheTownMenuRaisesNoMapPopup(t *testing.T) {
	f := openTownMenu(t)
	if f.mapShowing() {
		t.Error("the town menu reports the map showing behind it")
	}
	if f.popupOpen() {
		t.Error("the town menu raised the map's popup answer")
	}
}

// TestTheWorldIsToldStoppedAndNoSpanIsBanked is AC-4 and AC-5, and it is the
// defect this story fixes in the screen it replaces: a menu that stopped being
// asked rather than being told stopped would pay the held span in one jump.
func TestTheWorldIsToldStoppedAndNoSpanIsBanked(t *testing.T) {
	a := newTestApp(t, appRows(3), okLoader(t))
	now := time.Unix(1_700_000_000, 0)
	a.step(appInput{}, now)
	a.flow.activateNewGame()
	a.step(appInput{Enter: true}, now)
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}

	var stops []bool
	ticks := 0
	a.flow.cadence = func(_ int, stopped, _, _ bool) { stops = append(stops, stopped) }
	a.flow.tick = func() { ticks++ }
	a.flow.farRung, a.flow.farStopped = -1, false // force the next sync to cross

	a.step(appInput{Escape: true}, now)
	if a.Screen() != ScreenGameMenu {
		t.Fatalf("Escape landed on %v, want the menu", a.Screen())
	}
	before := ticks
	a.step(appInput{}, now)
	if ticks != before+1 {
		t.Errorf("the advance ran %d times on a menu frame, want exactly 1", ticks-before)
	}
	if len(stops) == 0 || !stops[len(stops)-1] {
		t.Fatalf("the far side was told %v, want a stop while the menu is up", stops)
	}

	a.step(appInput{Escape: true}, now)
	if a.Screen() != ScreenMap {
		t.Fatalf("Escape landed on %v, want the map screen", a.Screen())
	}
	a.step(appInput{}, now)
	if stops[len(stops)-1] {
		t.Errorf("the menu closed and the far side is still told stopped: %v", stops)
	}
}

func TestAPointerReleaseHitsTheRowItIsDrawnOn(t *testing.T) {
	for n := 0; n < 7; n++ {
		r := gameMenuRowRect(gameMenuMission, n)
		mid := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
		got, ok := gameMenuRowAt(gameMenuMission, 7, mid)
		if !ok || got != n {
			t.Errorf("the middle of row %d resolves to (%d,%v)", n, got, ok)
		}
	}
	panel := gameMenuPanelRect(gameMenuMission)
	for _, p := range []image.Point{
		{X: 0, Y: 0},                              // outside the panel entirely
		{X: panel.Min.X + 5, Y: panel.Min.Y + 5},  // inside the panel, above row 0
		{X: panel.Min.X + 5, Y: panel.Min.Y + 45}, // inside the panel, left of row 0
		{X: panel.Min.X + 45, Y: panel.Max.Y - 5}, // inside the panel, below the last row
	} {
		if _, ok := gameMenuRowAt(gameMenuMission, 7, p); ok {
			t.Errorf("%v resolved to a row; it is on none", p)
		}
	}
	// The town surface lists five rows, so the sixth rectangle is not a row.
	r := gameMenuRowRect(gameMenuTown, 5)
	if _, ok := gameMenuRowAt(gameMenuTown, 5, r.Min.Add(image.Pt(5, 5))); ok {
		t.Error("a point below the town surface's last row resolved to a row")
	}
}

// TestTheDrawPathComposesBothSurfaces is as far as this package can witness a
// drawn panel, on app_test.go's own reasoning: Draw runs headless but
// ReadPixels panics before a game starts, so what a human would see is a
// developer-run criterion and is not claimed here.
//
// What IS witnessed: both paths run over the real state without panicking, and
// the map path allocates its own overlay frame rather than writing into the
// canvas the town path composes into.
func TestTheDrawPathComposesBothSurfaces(t *testing.T) {
	a := newTestApp(t, appRows(3), okLoader(t))
	now := time.Unix(1_700_000_000, 0)
	a.canvas = ebiten.NewImage(frame.W, frame.H)

	a.step(appInput{}, now)
	a.flow.activateNewGame()
	a.step(appInput{Enter: true}, now)
	a.step(appInput{Escape: true}, now)
	if a.Screen() != ScreenGameMenu || a.flow.menuSurface != gameMenuMission {
		t.Fatalf("setup: screen %v surface %v", a.Screen(), a.flow.menuSurface)
	}
	a.Draw(ebiten.NewImage(a.winW, a.winH))
	if a.menuCanvas == nil {
		t.Error("the map path drew the panel without its own overlay frame")
	}
	if b := a.menuCanvas.Bounds(); b.Dx() != frame.W || b.Dy() != frame.H {
		t.Errorf("the overlay frame is %v, want the %dx%d virtual frame", b, frame.W, frame.H)
	}

	a.SetTown(&stubTown{rows: []TownRow{{Text: "the square", Choosable: true}}})
	a.flow.closeGameMenu()
	a.flow.showTown("")
	a.flow.openGameMenu(ScreenTown)
	if a.flow.menuSurface != gameMenuTown {
		t.Fatalf("the town raised surface %v", a.flow.menuSurface)
	}
	a.Draw(ebiten.NewImage(a.winW, a.winH))
}

func TestThePanelPaintsInsideItsOwnRect(t *testing.T) {
	f := openMissionMenu(t)
	dst := ebiten.NewImage(frame.W, frame.H)
	paintGameMenu(dst, nil, f.menuSurface, f.menuRows(), f.menuList)
	// ReadPixels is not available before a game starts, so this asserts the
	// contract the paint is written to rather than the pixels: every rectangle
	// the paint composes is derived from the panel rect, and the row rects are
	// checked against it in TestTheDecodedRectangles. What is witnessed here is
	// that the paint accepts a nil list, an empty surface and a real one
	// without reaching outside any of them.
	paintGameMenu(dst, nil, f.menuSurface, nil, f.menuList)
	paintGameMenu(dst, nil, f.menuSurface, f.menuRows(), nil)
	paintGameMenu(nil, nil, f.menuSurface, f.menuRows(), f.menuList)

	// BOTH PAINT PATHS RUN. The nil font is the debug-font path this package
	// had before 0168; a font is the install path, and a row label whose bytes
	// are CP866 is what it exists for. Neither may reach outside the panel and
	// neither may fail on an empty or a marked label.
	font := gameMenuTestFont()
	paintGameMenu(dst, font, f.menuSurface, f.menuRows(), f.menuList)
	cyrillic := []gameMenuRow{
		{Label: string([]byte{0x87, '~', 0xa0, 0xa4}), Enabled: true},
		{Label: "", Enabled: false},
		{Label: "~~escaped", Enabled: true},
	}
	paintGameMenu(dst, font, f.menuSurface, cyrillic, f.menuList)
}

// TestGameMenuPaintUsesDecodedRectangles observes the rectangles passed by
// paintGameMenu to the production vector boundary. The expected rectangles are
// written as literals. The test therefore fails when a draw-call use site moves
// even if gameMenuPanelRect and gameMenuRowRect remain unchanged.
func TestGameMenuPaintUsesDecodedRectangles(t *testing.T) {
	type draw struct {
		kind       string
		x, y, w, h float32
		colour     color.Color
	}
	var got []draw
	oldFill, oldStroke := fillGameMenuRect, strokeGameMenuRect
	fillGameMenuRect = func(_ *ebiten.Image, x, y, w, h float32, c color.Color, _ bool) {
		got = append(got, draw{kind: "fill", x: x, y: y, w: w, h: h, colour: c})
	}
	strokeGameMenuRect = func(_ *ebiten.Image, x, y, w, h, _ float32, c color.Color, _ bool) {
		got = append(got, draw{kind: "stroke", x: x, y: y, w: w, h: h, colour: c})
	}
	defer func() { fillGameMenuRect, strokeGameMenuRect = oldFill, oldStroke }()

	f := openMissionMenu(t)
	f.menuSound = nil
	f.setMenuSound = nil
	f.rebuildGameMenu(gameMenuRoot, 0)
	paintGameMenu(ebiten.NewImage(frame.W, frame.H), nil, f.menuSurface, f.menuRows(), f.menuList)

	want := []draw{
		{kind: "fill", x: 100, y: 60, w: 340, h: 340, colour: gameMenuFill},
		{kind: "stroke", x: 100.5, y: 60.5, w: 339, h: 339, colour: gameMenuBorder},
		{kind: "fill", x: 140, y: 100, w: 252, h: 30, colour: gameMenuFocusFill},
		{kind: "fill", x: 140, y: 190, w: 252, h: 30, colour: gameMenuDisabled},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("vector draws = %#v, want %#v", got, want)
	}
}

func TestCompleteGameMenuPaintFrames(t *testing.T) {
	type frameCase struct {
		name       string
		surface    gameMenuSurface
		page       gameMenuPage
		campaign   bool
		save, load bool
		sound      bool
		want       string
	}
	cases := []frameCase{
		{name: "campaign root", surface: gameMenuMission, page: gameMenuRoot, campaign: true, save: true, load: true, sound: true, want: "b6ae5dfe17fc05a9dc471f8526d5c7fc4b5d4f6a478fe8962eef94c8a6b4b9ac"},
		{name: "standalone root", surface: gameMenuMission, page: gameMenuRoot, save: true, load: true, sound: true, want: "67a8a7f60b8792d92461ca001234b9643895a92e49fa09487add169041692580"},
		{name: "town root", surface: gameMenuTown, page: gameMenuRoot, save: true, load: true, sound: true, want: "fd6b1a4d830bb4ac5b9f7c1d50e2855758e87a15a3280a829946fe18b0b12ff8"},
		{name: "game options", surface: gameMenuMission, page: gameMenuGameOptionsPage, campaign: true, want: "fee60c3d4bac042c1dea4dc4cc8f15f4be61e1aa1e2d64b83e6598cd187eaf77"},
		{name: "sound options", surface: gameMenuMission, page: gameMenuSoundOptionsPage, campaign: true, sound: true, want: "52dfc3d5a762dea4d8d2f0e850904edb369eedbefe5359233b1df81c3bb96a16"},
		{name: "diplomacy", surface: gameMenuMission, page: gameMenuDiplomacyPage, campaign: true, want: "8288d7b6b9e4795904760a7b1242e5328279a071ebc50c591cd7100fbf0006c5"},
		{name: "end confirmation", surface: gameMenuMission, page: gameMenuEndQuestConfirmation, campaign: true, want: "f971676a751fe61e0f9e431cbeeece60e40c9069d1926e0c699805d43a1a4fa6"},
		{name: "abort confirmation", surface: gameMenuTown, page: gameMenuAbortGameConfirmation, want: "c46a20e94a692a8c1c2381faecd9d8dd7846270872fbcd97febeac521a29af27"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFlow(NewPicker(nil), nil)
			f.menuSurface = tc.surface
			f.menuContext = GameMenuContext{
				Campaign:        tc.campaign,
				CampaignVictory: tc.campaign,
				Objective:       "HOLD THE BRIDGE.",
				Relations:       []GameMenuRelation{{Slot: 2, State: "ALLY"}, {Slot: 3, State: "ENEMY"}},
			}
			if tc.save {
				f.saveGame = func(bool) (string, error) { return "frame.ags", nil }
			}
			if tc.load {
				f.saveList = func() []SaveEntry { return []SaveEntry{{Name: "frame.ags", Label: "frame"}} }
			}
			f.menuTips = func() bool { return true }
			f.setMenuTips = func(bool) {}
			if tc.sound {
				f.menuSound = func() (bool, int, bool) { return true, 75, true }
				f.setMenuSound = func(bool, int) error { return nil }
			}
			f.rebuildGameMenu(tc.page, 0)

			var trace strings.Builder
			colour := func(c color.Color) string {
				r, g, b, a := c.RGBA()
				return fmt.Sprintf("%04x%04x%04x%04x", r, g, b, a)
			}
			oldFill, oldRect, oldLine, oldLabel := fillGameMenuRect, strokeGameMenuRect, strokeGameMenuLine, drawGameMenuLabel
			fillGameMenuRect = func(_ *ebiten.Image, x, y, w, h float32, c color.Color, _ bool) {
				fmt.Fprintf(&trace, "fill %.1f %.1f %.1f %.1f %s\n", x, y, w, h, colour(c))
			}
			strokeGameMenuRect = func(_ *ebiten.Image, x, y, w, h, width float32, c color.Color, _ bool) {
				fmt.Fprintf(&trace, "rect %.1f %.1f %.1f %.1f %.1f %s\n", x, y, w, h, width, colour(c))
			}
			strokeGameMenuLine = func(_ *ebiten.Image, x1, y1, x2, y2, width float32, c color.Color, _ bool) {
				fmt.Fprintf(&trace, "line %.1f %.1f %.1f %.1f %.1f %s\n", x1, y1, x2, y2, width, colour(c))
			}
			drawGameMenuLabel = func(_ *ebiten.Image, _ *text.Font, label string, x, y int, c color.RGBA) {
				fmt.Fprintf(&trace, "text %d %d %s %q\n", x, y, colour(c), label)
			}
			paintGameMenu(ebiten.NewImage(frame.W, frame.H), gameMenuTestFont(), f.menuPanelSurface(), f.menuRows(), f.menuList)
			fillGameMenuRect, strokeGameMenuRect, strokeGameMenuLine, drawGameMenuLabel = oldFill, oldRect, oldLine, oldLabel

			got := fmt.Sprintf("%x", sha256.Sum256([]byte(trace.String())))
			if got != tc.want {
				t.Fatalf("complete paint digest = %s, want %s\ntrace:\n%s", got, tc.want, trace.String())
			}
		})
	}
}

// gameMenuTestFont is a byte-indexed font with one painted pixel per record, so
// every one of the 224 records draws something and none is confusable with a
// blank. It stands in for the install font the front end hands over.
func gameMenuTestFont() *text.Font {
	font := &text.Font{Spacing: 1, Glyphs: make([]text.Glyph, 224)}
	for k := range font.Glyphs {
		g := text.Glyph{Width: 5, Height: 8, Pixels: make([]text.Pixel, 5*8), Advance: 5}
		g.Pixels[k%(5*8)] = text.Pixel{Level: text.MaxLevel, Painted: true}
		font.Glyphs[k] = g
	}
	return font
}
