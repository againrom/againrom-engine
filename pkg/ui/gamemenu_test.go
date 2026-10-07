package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
)

func leaveViaMenu(f *flow) {
	if f.screen != ScreenGameMenu || f.menuList == nil {
		return
	}
	if !selectGameMenuAction(f, gameMenuEndQuest) {
		selectGameMenuAction(f, gameMenuAbortGame)
	}
	f.chooseGameMenu()
	if !selectGameMenuAction(f, gameMenuConfirmEndQuest) {
		selectGameMenuAction(f, gameMenuExitMain)
	}
	f.chooseGameMenu()
}

// selectGameMenuAction moves the focus to the row that does a, and reports
// whether the surface has one.
func selectGameMenuAction(f *flow, a gameMenuAction) bool {
	for i, row := range f.menuRows() {
		if row.Action == a {
			f.menuList.Select(i)
			return true
		}
	}
	return false
}

// escapeOut is escape() as those tests used to call it: it opens the menu and
// takes its EXIT row in one call, and reports escape()'s own answer.
func escapeOut(f *flow) bool {
	exit := f.escape()
	leaveViaMenu(f)
	return exit
}

func TestEscapeRaisesTheMenu(t *testing.T) {
	f := newFlow(NewPicker(nil), nil)
	f.screen = ScreenMap
	if f.escape() {
		t.Fatal("Escape on the map screen exited the program")
	}
	if f.screen != ScreenGameMenu {
		t.Fatalf("Escape on the map screen landed on %v, want the in-game menu", f.screen)
	}
	if len(f.menuRows()) != f.menuList.Len() {
		t.Fatalf("the surface has %d rows and the list shows %d", len(f.menuRows()), f.menuList.Len())
	}
}

// TestTheMiniMenuReturnsToTheScreenItWasOpenedFrom witnesses plan D-5 on both
// doors: RETURN and Escape land back where the player was, and neither is a
// constant.
func TestTheMiniMenuReturnsToTheScreenItWasOpenedFrom(t *testing.T) {
	for _, tc := range []struct {
		name string
		back Screen
	}{{"from the map", ScreenMap}, {"from the town", ScreenTown}} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFlow(NewPicker(nil), nil)
			f.openGameMenu(tc.back)
			selectGameMenuAction(f, gameMenuReturn)
			f.chooseGameMenu()
			if f.screen != tc.back {
				t.Fatalf("RETURN landed on %v, want %v", f.screen, tc.back)
			}
			f.openGameMenu(tc.back)
			if f.escape() {
				t.Fatal("Escape on the mini-menu exited the program")
			}
			if f.screen != tc.back {
				t.Fatalf("Escape landed on %v, want %v", f.screen, tc.back)
			}
		})
	}
}

func TestWithNoStoreSaveAndLoadAreGreyedAndNeverCallNil(t *testing.T) {
	f := newFlow(NewPicker(nil), nil)
	f.openGameMenu(ScreenMap)
	for _, a := range []gameMenuAction{gameMenuSave, gameMenuLoad} {
		if !selectGameMenuAction(f, a) {
			t.Fatalf("the mission surface has no row for action %d", a)
		}
		i := f.menuList.Selection()
		if f.menuList.Rows()[i].Choosable {
			t.Errorf("row %d is choosable with no store behind it", i)
		}
		f.chooseGameMenu()
		if f.screen != ScreenGameMenu {
			t.Fatalf("a greyed row left the menu for %v", f.screen)
		}
	}
}

func TestTheLoadRowOpensTheLoadWindowAndEscapeComesBack(t *testing.T) {
	f := newFlow(NewPicker(nil), nil)
	f.saveList = func() []SaveEntry { return []SaveEntry{{Name: "a.ags", Label: "a save"}} }
	f.openGameMenu(ScreenMap)
	if !selectGameMenuAction(f, gameMenuLoad) {
		t.Fatal("the mission surface has no LOAD row")
	}
	f.chooseGameMenu()
	if f.screen != ScreenLoad {
		t.Fatalf("LOAD landed on %v, want the load window", f.screen)
	}
	if f.escape() {
		t.Fatal("Escape on the load window exited the program")
	}
	if f.screen != ScreenGameMenu {
		t.Errorf("Escape from the load window landed on %v, want the in-game menu", f.screen)
	}
}

func TestTheLoadWindowListsSavesAndReportsARefusal(t *testing.T) {
	var asked string
	f := newFlow(NewPicker(nil), nil)
	f.saveList = func() []SaveEntry {
		return []SaveEntry{{Name: "a.ags", Label: "mission 1"}, {Name: "b.ags", Label: "town"}}
	}
	f.loadGame = func(name string) (MapOpener, bool, error) {
		asked = name
		return nil, false, errNotASave
	}
	f.openLoad(ScreenMenu)
	if f.screen != ScreenLoad {
		t.Fatalf("openLoad landed on %v", f.screen)
	}
	if got := f.loadList.RowText(0); got == "" {
		t.Fatal("the first row is empty")
	}
	f.loadList.Select(1)
	f.chooseLoad()
	if asked != "b.ags" {
		t.Errorf("the load seam was asked for %q, want the second row's NAME", asked)
	}
	if f.screen != ScreenLoad {
		t.Errorf("a refused load left the list for %v", f.screen)
	}
	// THE LINE NAMES THE SAVE AND THEN THE REASON (1032 return 1). The reason
	// alone was ambiguous on a list of rows named by date: a player who chose
	// the wrong row read a sentence about a save and could not tell which.
	if want := "b.ags: " + errNotASave.Error(); f.msg != want {
		t.Errorf("message = %q, want %q", f.msg, want)
	}
	if !f.loadList.rows[1].Choosable {
		t.Error("a refused row was marked unusable; it must stay choosable")
	}
}

// errNotASave stands in for whatever the far side refuses with. This package
// cannot name game.ErrNotSave and does not need to: it shows the sentence.
var errNotASave = stubErr("not an againrom save")

type stubErr string

func (e stubErr) Error() string { return string(e) }

// TestTheTownSaveArmIsTheTownScreen witnesses the load window's town arm: a
// seam answering town=true shows the town rather than entering a map.
func TestTheTownSaveArmIsTheTownScreen(t *testing.T) {
	f := newFlow(NewPicker(nil), nil)
	f.town = &stubTown{rows: []TownRow{{Text: "gates", Choosable: true}}}
	f.saveList = func() []SaveEntry { return []SaveEntry{{Name: "t.ags", Label: "town"}} }
	f.loadGame = func(string) (MapOpener, bool, error) { return nil, true, nil }
	f.openLoad(ScreenMenu)
	f.loadList.Select(0)
	f.chooseLoad()
	if f.screen != ScreenTown {
		t.Fatalf("a town save landed on %v, want the town", f.screen)
	}
	if f.townList == nil {
		t.Error("the town screen is showing over a nil list")
	}
}

// stubTown is the smallest TownScreen the load window's town arm needs.
type stubTown struct{ rows []TownRow }

func (s *stubTown) Header() string        { return "town" }
func (s *stubTown) Rows() []TownRow       { return s.rows }
func (s *stubTown) Footer() []string      { return nil }
func (s *stubTown) Choose(int) TownAction { return TownAction{} }
func (s *stubTown) Back() bool            { return false }

// TestTheApplicationDrivesTheMiniMenuAndTheLoadWindow drives App.step and
// App.Draw rather than the flow, because the dispatch arms are what a player's
// press actually reaches: a screen wired into the flow and not into the switch
// would pass every test above and show nothing.
//
// IT IS THE WHOLE LOOP IN ONE TEST — Escape on the map, Down to SAVE, Enter,
// Escape back, L on the main menu, Enter on the one row — because that is the
// sequence a player performs and the one the automated suite can hold when a
// live window cannot be driven.
func TestTheApplicationDrivesTheMiniMenuAndTheLoadWindow(t *testing.T) {
	a := newTestApp(t, appRows(3), okLoader(t))
	now := time.Unix(1_700_000_000, 0)

	var saved, loaded string
	a.SetSaveSeams(
		func(onMap bool) (string, error) {
			if !onMap {
				return "", errNotASave
			}
			saved = "save-1.ags"
			return saved, nil
		},
		func() []SaveEntry {
			if saved == "" {
				return nil
			}
			return []SaveEntry{{Name: saved, Label: "mission 10 — tick 300 — gold 0"}}
		},
		func(name string) (MapOpener, bool, error) {
			loaded = name
			return nil, true, nil
		},
	)
	a.SetTown(&stubTown{rows: []TownRow{{Text: "the square", Choosable: true}}})

	// Onto the map screen the way a player gets there.
	a.step(appInput{}, now)
	a.flow.activateNewGame()
	a.step(appInput{Enter: true}, now)
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}

	// Escape opens the mini-menu, Down selects SAVE, Enter saves.
	if exit := a.step(appInput{Escape: true}, now); exit {
		t.Fatal("Escape on the map screen exited the program")
	}
	if a.Screen() != ScreenGameMenu {
		t.Fatalf("Escape landed on %v, want the mini-menu", a.Screen())
	}
	if !selectGameMenuAction(a.flow, gameMenuSave) {
		t.Fatal("the mission surface has no SAVE row")
	}
	a.step(appInput{Enter: true}, now)
	if saved == "" {
		t.Fatal("Enter on SAVE wrote nothing")
	}
	if a.Screen() != ScreenGameMenu {
		t.Errorf("SAVE left the menu for %v; it reports and stays", a.Screen())
	}
	if a.flow.msg == "" {
		t.Error("SAVE said nothing about what it wrote")
	}

	// It paints, without a window.
	a.canvas = ebiten.NewImage(frame.W, frame.H)
	a.Draw(ebiten.NewImage(a.winW, a.winH))

	// Out of the mini-menu, out of the map, back to the main menu, and L opens
	// the LOAD GAME window over it.
	if !selectGameMenuAction(a.flow, gameMenuEndQuest) {
		t.Fatal("the mission surface has no END QUEST row")
	}
	a.step(appInput{Enter: true}, now)
	if a.Screen() != ScreenGameMenu || a.flow.menuPage != gameMenuEndQuestConfirmation {
		t.Fatalf("END QUEST did not open its confirmation page")
	}
	a.step(appInput{Enter: true}, now) // CHANGE MAP: the map list
	if a.Screen() != ScreenPicker {
		t.Fatalf("END QUEST landed on %v, want the map list", a.Screen())
	}
	if a.flow.viewer != nil || a.flow.tick != nil {
		t.Error("END QUEST left the map screen's seams held")
	}
	if exit := a.step(appInput{Escape: true}, now); exit {
		t.Fatal("Escape on the map list exited the program")
	}
	if a.Screen() != ScreenMenu {
		t.Fatalf("Escape from the map list landed on %v, want the main menu", a.Screen())
	}
	a.step(appInput{Load: true}, now)
	if a.Screen() != ScreenLoad {
		t.Fatalf("L on the main menu landed on %v, want the load window", a.Screen())
	}
	if n := a.flow.loadList.Len(); n != 1 {
		t.Fatalf("the load window lists %d rows, want the one save", n)
	}
	if got := a.loadHeader(); !strings.Contains(got, "1 saved") {
		t.Errorf("header = %q, want it to count the saves", got)
	}
	a.Draw(ebiten.NewImage(a.winW, a.winH))
	a.step(appInput{Enter: true}, now)
	if loaded != saved {
		t.Fatalf("the load window asked for %q, want %q", loaded, saved)
	}
	if a.Screen() != ScreenTown {
		t.Fatalf("a town save landed on %v, want the town", a.Screen())
	}
}

func TestSaveTellsTheFarSideWhichScreenItWasTakenOn(t *testing.T) {
	for _, tc := range []struct {
		name      string
		back      Screen
		wantOnMap bool
		wantExit  Screen
	}{
		{"in a mission", ScreenMap, true, ScreenPicker},
		{"in the town", ScreenTown, false, ScreenMenu},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []bool
			f := newFlow(NewPicker(nil), nil)
			f.town = &stubTown{rows: []TownRow{{Text: "the square", Choosable: true}}}
			f.saveGame = func(onMap bool) (string, error) {
				got = append(got, onMap)
				return "s.ags", nil
			}
			f.saveList = func() []SaveEntry { return []SaveEntry{{Name: "s.ags", Label: "a save"}} }
			f.loadGame = func(string) (MapOpener, bool, error) { return nil, true, nil }

			// SAVE
			f.openGameMenu(tc.back)
			selectGameMenuAction(f, gameMenuSave)
			f.chooseGameMenu()
			if len(got) != 1 || got[0] != tc.wantOnMap {
				t.Fatalf("SAVE reported onMap=%v, want %v", got, tc.wantOnMap)
			}
			if f.screen != ScreenGameMenu {
				t.Errorf("SAVE left the menu for %v", f.screen)
			}

			// LOAD, and Escape back to this same menu.
			selectGameMenuAction(f, gameMenuLoad)
			f.chooseGameMenu()
			if f.screen != ScreenLoad {
				t.Fatalf("LOAD landed on %v", f.screen)
			}
			f.escape()
			if f.screen != ScreenGameMenu {
				t.Fatalf("Escape from the load window landed on %v, want the mini-menu", f.screen)
			}

			// RETURN
			selectGameMenuAction(f, gameMenuReturn)
			f.chooseGameMenu()
			if f.screen != tc.back {
				t.Fatalf("RETURN landed on %v, want %v", f.screen, tc.back)
			}

			// END QUEST in a mission, ABORT GAME in the town.
			f.openGameMenu(tc.back)
			leaveViaMenu(f)
			if f.screen != tc.wantExit {
				t.Fatalf("the leaving row landed on %v, want %v", f.screen, tc.wantExit)
			}
		})
	}
}

func TestTheLoadWindowShowsTheHIGHLIGHTEDRowsCaveatAndARefusalWinsTheLine(t *testing.T) {
	f := &flow{
		screen: ScreenLoad,
		saves: []SaveEntry{
			{Name: "ours.ags", Label: "mission 10 — tick 300"},
			{Name: "game0000.sav", Label: "1 — mission 10", Note: "carries almost nothing"},
		},
	}
	f.loadList = NewPicker([]PickerRow{
		{Text: f.saves[0].Label, Choosable: true},
		{Text: f.saves[1].Label, Choosable: true},
	})

	if got := f.loadNote(); got != "" {
		t.Errorf("our own save showed the caveat %q; it has none", got)
	}
	f.loadList.Move(1)
	if got := f.loadNote(); got != f.saves[1].Note {
		t.Errorf("on the original row the line reads %q, want %q", got, f.saves[1].Note)
	}

	// A refusal takes the line back, because there is only one and what just
	// failed is the more urgent of the two.
	f.msg = "this file will not read"
	if got := f.loadNote(); got != "" {
		t.Errorf("a refusal was on the line and the caveat still claimed it: %q", got)
	}
	f.msg = ""

	// Off the load window it answers nothing at all, so no other screen can
	// pick the line up by accident.
	f.screen = ScreenMenu
	if got := f.loadNote(); got != "" {
		t.Errorf("the caveat %q reached the %v screen", got, f.screen)
	}

	// And an empty list is not an index error.
	f.screen, f.saves = ScreenLoad, nil
	if got := f.loadNote(); got != "" {
		t.Errorf("an empty list produced the caveat %q", got)
	}
}

// TestTheCaveatReachesTheWindowAndTheDrawPathTakesIt is as far as this package
// can witness a drawn string, and the header of app_test.go says why: Draw runs
// headless but ReadPixels panics before a game starts, so what a human would see
// is a developer-run criterion and is not claimed here.
//
// What IS witnessed: the note survives the seam into the row the window holds,
// loadNote answers it while that row is highlighted, and the draw path runs over
// exactly that state without panicking. The statement that puts it on the canvas
// is the same shape as the four message-line draws beside it.
func TestTheCaveatReachesTheWindowAndTheDrawPathTakesIt(t *testing.T) {
	const note = "ORIGINAL SAVE: map unit positions only"
	a := newTestApp(t, appRows(3), okLoader(t))
	now := time.Unix(1_700_000_000, 0)
	a.SetSaveSeams(nil,
		func() []SaveEntry {
			return []SaveEntry{{Name: "game0000.sav", Label: "1 — mission 10", Note: note}}
		},
		func(string) (MapOpener, bool, error) { return nil, true, nil },
	)
	a.step(appInput{}, now)
	a.step(appInput{Load: true}, now)
	if a.Screen() != ScreenLoad {
		t.Fatalf("L landed on %v, want the load window", a.Screen())
	}
	if a.flow.msg != "" {
		t.Fatalf("the line already holds %q, so the caveat could not show anyway", a.flow.msg)
	}
	if got := a.flow.loadNote(); got != note {
		t.Fatalf("the window would draw %q on its message line, want the row's note %q", got, note)
	}
	a.canvas = ebiten.NewImage(frame.W, frame.H)
	a.Draw(ebiten.NewImage(a.winW, a.winH))
}
