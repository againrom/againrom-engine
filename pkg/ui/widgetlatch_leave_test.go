package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/menu"
)

// A press held on a main-menu button while the Load key leaves the menu
// does not survive the screen: after Escape returns, the first click on
// another button activates it (MENU-116).
func TestMainMenuPressDoesNotOutliveTheMenu(t *testing.T) {
	a := newTestApp(t, appRows(3), okLoader(t))
	nx, ny := centreOf(menu.NewGameButton)
	lx, ly := centreOf(menu.LoadGameButton)
	a.step(appInput{CursorX: nx, CursorY: ny, PrimaryPressed: true}, atAt)
	a.step(appInput{CursorX: nx, CursorY: ny, Load: true}, atAt)
	if a.Screen() != ScreenLoad {
		t.Fatalf("the Load key opened %v", a.Screen())
	}
	a.step(appInput{CursorX: 600, CursorY: 460, PrimaryReleased: true}, atAt)
	a.step(appInput{CursorX: 600, CursorY: 460, Escape: true}, atAt)
	if a.Screen() != ScreenMenu {
		t.Fatalf("Escape returned to %v", a.Screen())
	}
	click(a, image.Pt(lx, ly))
	if a.Screen() != ScreenLoad {
		t.Fatalf("the first click on Load Game after a held press left the menu activated nothing: screen %v", a.Screen())
	}
}

// gameMenuRowsFor returns a latchable in-game menu row and the Quest
// Objectives row.
func gameMenuRowsFor(t *testing.T, a *App) (held, quest image.Point) {
	t.Helper()
	s := a.flow.menuPanelSurface()
	h, q := -1, -1
	for i, r := range a.flow.menuRows() {
		if r.Action == gameMenuQuestObjectives {
			q = i
		} else if h < 0 && r.Enabled && !r.Status {
			h = i
		}
	}
	if h < 0 || q < 0 {
		t.Fatalf("fixture: rows held %d quest %d", h, q)
	}
	return midOf(gameMenuRowRect(s, h)), midOf(gameMenuRowRect(s, q))
}

// A press held while Escape closes the in-game menu does not survive it:
// after the menu reopens, the first click on Quest Objectives opens it.
func TestGameMenuPressDoesNotOutliveTheMenu(t *testing.T) {
	a := latchMenuApp(t)
	held, quest := gameMenuRowsFor(t, a)
	a.step(appInput{CursorX: held.X, CursorY: held.Y, PrimaryPressed: true}, atAt)
	if !a.flow.menuPress.Holds() {
		t.Fatal("fixture: the press latched no row")
	}
	a.step(appInput{CursorX: held.X, CursorY: held.Y, Escape: true}, atAt)
	a.step(appInput{CursorX: 600, CursorY: 470, PrimaryReleased: true}, atAt)
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	click(a, quest)
	if a.flow.menuPage != gameMenuQuestObjectivesPage {
		t.Fatalf("the first click on Quest Objectives after a held press left the menu did nothing: page %v", a.flow.menuPage)
	}
}

// A release that falls in the same tick as a key activates nothing and
// drops the latch, so the next click on another row activates it.
func TestGameMenuReleaseWithAKeyDropsTheLatch(t *testing.T) {
	a := latchMenuApp(t)
	held, quest := gameMenuRowsFor(t, a)
	a.step(appInput{CursorX: held.X, CursorY: held.Y, PrimaryPressed: true}, atAt)
	if !a.flow.menuPress.Holds() {
		t.Fatal("fixture: the press latched no row")
	}
	a.step(appInput{CursorX: held.X, CursorY: held.Y, PrimaryReleased: true, Down: true}, atAt)
	if a.flow.menuPress.Holds() {
		t.Fatal("a release in a key's tick kept the latch")
	}
	click(a, quest)
	if a.flow.menuPage != gameMenuQuestObjectivesPage {
		t.Fatalf("the first click on Quest Objectives after a release with a key did nothing: page %v", a.flow.menuPage)
	}
}
