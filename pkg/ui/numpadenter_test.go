package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// enterEdge is the tick a press of key gives: the key edge read through the
// bindings a windowed tick asks, with no window.
func enterEdge(key ebiten.Key) appInput {
	return appInput{AnyKey: true, Enter: enterPressed(onlyKey(key))}
}

// TestEnterPressedIsExactlyTheTwoEnterKeys asks the Enter binding about every key
// the engine knows, one at a time: only the main Enter key and the numpad's
// Enter key are Enter.
func TestEnterPressedIsExactlyTheTwoEnterKeys(t *testing.T) {
	for key := ebiten.Key(0); key <= ebiten.KeyMax; key++ {
		want := key == ebiten.KeyEnter || key == ebiten.KeyNumpadEnter
		if got := enterPressed(onlyKey(key)); got != want {
			t.Errorf("enterPressed with only %v down = %v, want %v", key, got, want)
		}
	}
	if enterPressed(func(ebiten.Key) bool { return false }) {
		t.Error("enterPressed with no key down is true")
	}
	if !enterPressed(func(ebiten.Key) bool { return true }) {
		t.Error("enterPressed with every key down is false")
	}
}

// enterRoutes are screens that act on Enter, each built fresh and given one
// tick. The answer names what a player could tell apart afterwards.
var enterRoutes = []struct {
	name string
	run  func(t *testing.T, in appInput) string
}{
	{"the map list chooses its row", func(t *testing.T, in appInput) string {
		a := newTestApp(t, appRows(10), okLoader(t))
		a.flow.screen = ScreenPicker
		a.step(appInput{Down: true}, numpadEnterAt)
		a.step(in, numpadEnterAt)
		return fmt.Sprintf("screen %v, row %d", a.Screen(), a.flow.picker.Selection())
	}},
	{"an open notice is acknowledged", func(t *testing.T, in appInput) string {
		a, seam := noticeApp(t)
		seam.dest = NoticeToMenu
		seam.v.SetNotice("you lost", NoticeOutcome)
		a.step(in, numpadEnterAt)
		return fmt.Sprintf("screen %v, map held %v", a.Screen(), a.flow.viewer != nil)
	}},
	{"the town's menu chooses its row", func(t *testing.T, in appInput) string {
		a := newTestApp(t, appRows(3), okLoader(t))
		a.SetTown(&stubTown{rows: []TownRow{{Text: "the square", Choosable: true}}})
		a.flow.openGameMenu(ScreenTown)
		if !selectGameMenuAction(a.flow, gameMenuAbortGame) {
			t.Fatal("the town's menu has no row that aborts the game")
		}
		a.step(in, numpadEnterAt)
		return fmt.Sprintf("screen %v, page %v, row %d", a.Screen(), a.flow.menuPage, a.flow.menuList.Selection())
	}},
	{"the load window loads the selected save", func(t *testing.T, in appInput) string {
		a := newTestApp(t, nil, nil)
		asked := ""
		a.SetSaveSeams(nil,
			func() []SaveEntry { return []SaveEntry{{Name: "file-0.ags", Label: "Saved game"}} },
			func(name string) (MapOpener, bool, error) {
				asked = name
				return nil, false, fmt.Errorf("read refused")
			})
		a.flow.openLoad(ScreenMenu)
		a.step(in, numpadEnterAt)
		return fmt.Sprintf("screen %v, asked for %q", a.Screen(), asked)
	}},
}

var numpadEnterAt = time.Unix(1_700_000_000, 0)

// TestNumpadEnterActsAsEnterOnTheScreensThatReadEnter gives each screen the
// tick of the main Enter key, the tick of the numpad's Enter key and a tick with
// neither. The two Enter keys leave the same state, and it is not the state the
// idle tick leaves, so the screen does act on Enter.
func TestNumpadEnterActsAsEnterOnTheScreensThatReadEnter(t *testing.T) {
	for _, r := range enterRoutes {
		t.Run(r.name, func(t *testing.T) {
			idle := r.run(t, enterEdge(ebiten.KeyA))
			enter := r.run(t, enterEdge(ebiten.KeyEnter))
			numpad := r.run(t, enterEdge(ebiten.KeyNumpadEnter))
			if enter == idle {
				t.Fatalf("Enter left %q, the same as a key that is not Enter, so the route proves nothing", enter)
			}
			if numpad != enter {
				t.Errorf("numpad Enter left %q, Enter left %q", numpad, enter)
			}
		})
	}
}

// TestNumpadEnterOnTheMainMenuDoesNothing keeps the main menu's own rule: it
// reads no key but Esc, and that holds for the numpad's Enter as for the main
// one.
func TestNumpadEnterOnTheMainMenuDoesNothing(t *testing.T) {
	a := newTestApp(t, appRows(3), okLoader(t))
	if exit := a.step(enterEdge(ebiten.KeyNumpadEnter), numpadEnterAt); exit {
		t.Fatal("numpad Enter at the main menu exited the program")
	}
	if a.Screen() != ScreenMenu {
		t.Fatalf("numpad Enter at the main menu moved to %v", a.Screen())
	}
}
