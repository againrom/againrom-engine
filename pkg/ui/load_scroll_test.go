package ui

import (
	"fmt"
	"testing"
)

type loadScrollCalls struct {
	loaded   []string
	prepared []string
	removed  int
}

func loadScrollApp(t *testing.T, count int) (*App, *loadScrollCalls) {
	t.Helper()
	a := newTestApp(t, nil, nil)
	a.Layout(640, 480)
	rows := make([]SaveEntry, count)
	for i := range rows {
		rows[i] = SaveEntry{Name: fmt.Sprintf("slot-%02d.sav", i), Label: fmt.Sprintf("Saved game %02d", i)}
	}
	calls := new(loadScrollCalls)
	a.SetSaveSeams(nil, func() []SaveEntry { return rows }, func(name string) (MapOpener, bool, error) {
		calls.loaded = append(calls.loaded, name)
		return nil, false, fmt.Errorf("load token received")
	})
	a.SetSaveDelete(func(string) bool { return true }, func(name string) (func() error, error) {
		calls.prepared = append(calls.prepared, name)
		return func() error { calls.removed++; return nil }, nil
	})
	loadScrollKey(t, a, "load")
	if a.Screen() != ScreenLoad || len(a.HeadlessRows()) != count {
		t.Fatalf("fixture did not open LOAD: screen=%v rows=%d", a.Screen(), len(a.HeadlessRows()))
	}
	return a, calls
}

func loadScrollKey(t *testing.T, a *App, key string) {
	t.Helper()
	if err := a.HeadlessKey(key); err != nil {
		t.Fatal(err)
	}
}

func loadScrollPointer(t *testing.T, a *App, action string, x, y int) {
	t.Helper()
	if err := a.HeadlessPointer(action, x, y); err != nil {
		t.Fatal(err)
	}
}

func loadScrollState(t *testing.T, a *App, calls *loadScrollCalls, selection, top int) {
	t.Helper()
	gotTop, n := a.flow.loadList.Visible()
	if got := a.flow.loadList.Selection(); got != selection || gotTop != top || n != min(10, len(a.HeadlessRows())) {
		t.Fatalf("held input selection=%d viewport=%d,%d; want selection=%d viewport=%d,%d", got, gotTop, n, selection, top, min(10, len(a.HeadlessRows())))
	}
	if len(calls.loaded) != 0 || len(calls.prepared) != 0 || calls.removed != 0 || a.flow.loadUI.confirm {
		t.Fatalf("scroll activated a save action: %+v confirm=%v", calls, a.flow.loadUI.confirm)
	}
}

// The Load bar is the shared bar at (504,152)-(528,344): H 192, W 24. Its
// drag arm maps y to clamp(26*(y-176)/132, 0, 26) over 27 saves, and its
// thumb starts at y 172 (MENU-119).
func loadDragPos(y int) int { return min(max(26*(y-176)/132, 0), 26) }

func TestLoadThumbFollowsHeldPointerAndConsumesRelease(t *testing.T) {
	a, calls := loadScrollApp(t, 27)
	loadScrollPointer(t, a, "press", 516, 188)
	loadScrollState(t, a, calls, 0, 0)
	loadScrollPointer(t, a, "move", 516, 247)
	loadScrollState(t, a, calls, loadDragPos(247), 4)
	loadScrollPointer(t, a, "move", 516, 308)
	loadScrollState(t, a, calls, 26, 17)
	loadScrollPointer(t, a, "move", 516, 221)
	loadScrollState(t, a, calls, loadDragPos(221), loadDragPos(221))
	loadScrollPointer(t, a, "move", 140, 247)
	loadScrollState(t, a, calls, 13, 8)
	loadScrollPointer(t, a, "release", 140, 272)
	loadScrollState(t, a, calls, 13, 8)
	loadScrollPointer(t, a, "move", 516, 306)
	loadScrollState(t, a, calls, 13, 8)
	loadScrollKey(t, a, "enter")
	if len(calls.loaded) != 1 || calls.loaded[0] != "slot-13.sav" {
		t.Fatalf("OK changed exact token: %+v", calls)
	}
}

func TestLoadThumbBoundsAndCancellation(t *testing.T) {
	for _, scenario := range []string{"outside bar", "outside frame", "focus", "idle", "reopen", "confirmation"} {
		t.Run(scenario, func(t *testing.T) {
			a, calls := loadScrollApp(t, 27)
			loadScrollPointer(t, a, "press", 516, 190)
			loadScrollPointer(t, a, "move", 516, 247)
			loadScrollState(t, a, calls, 13, 4)
			switch scenario {
			case "outside bar":
				loadScrollPointer(t, a, "move", 10, 400)
				loadScrollState(t, a, calls, 26, 17)
				loadScrollPointer(t, a, "move", 10, 80)
				loadScrollState(t, a, calls, 0, 0)
				loadScrollPointer(t, a, "release", 324, 392)
				loadScrollState(t, a, calls, 0, 0)
				return
			case "outside frame":
				// Off the frame the drag keeps its capture and asks nothing.
				loadScrollPointer(t, a, "move", -1, 258)
				loadScrollState(t, a, calls, 13, 4)
				loadScrollPointer(t, a, "move", 516, 300)
				loadScrollState(t, a, calls, loadDragPos(300), loadDragPos(300)-9)
				loadScrollPointer(t, a, "release", 200, 392)
				return
			case "focus":
				if err := a.HeadlessFocus(false); err != nil {
					t.Fatal(err)
				}
				loadScrollPointer(t, a, "move", 516, 317)
				if err := a.HeadlessFocus(true); err != nil {
					t.Fatal(err)
				}
			case "idle":
				loadScrollPointer(t, a, "hover", 516, 258)
			case "reopen":
				loadScrollKey(t, a, "escape")
				loadScrollKey(t, a, "load")
				loadScrollPointer(t, a, "move", 516, 317)
				loadScrollPointer(t, a, "release", 200, 392)
				loadScrollState(t, a, calls, 0, 0)
				return
			case "confirmation":
				loadScrollPointer(t, a, "release", 516, 247)
				loadScrollKey(t, a, "delete")
				loadScrollPointer(t, a, "press", 516, 247)
				loadScrollPointer(t, a, "move", 516, 317)
				loadScrollPointer(t, a, "release", 200, 392)
				if a.flow.loadList.Selection() != 13 || len(calls.prepared) != 1 || calls.prepared[0] != "slot-13.sav" || calls.removed != 0 || !a.flow.loadUI.confirm {
					t.Fatalf("thumb changed Delete ownership: %+v", calls)
				}
				return
			}
			loadScrollPointer(t, a, "move", 516, 317)
			loadScrollPointer(t, a, "release", 200, 392)
			loadScrollState(t, a, calls, 13, 4)
		})
	}
}

func TestLoadThumbReleaseResetsDuplicateClick(t *testing.T) {
	a, calls := loadScrollApp(t, 27)
	loadScrollPointer(t, a, "press", 140, 158)
	loadScrollPointer(t, a, "release", 140, 158)
	loadScrollPointer(t, a, "press", 516, 188)
	loadScrollPointer(t, a, "move", 516, 188)
	loadScrollPointer(t, a, "release", 140, 158)
	loadScrollPointer(t, a, "press", 140, 158)
	loadScrollPointer(t, a, "release", 140, 158)
	loadScrollState(t, a, calls, 0, 0)
	loadScrollPointer(t, a, "press", 140, 158)
	loadScrollPointer(t, a, "release", 140, 158)
	if len(calls.loaded) != 1 || calls.loaded[0] != "slot-00.sav" {
		t.Fatalf("same-save double click was lost: %+v", calls)
	}
}

func TestLoadThumbSmallListsRemainBounded(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			a, calls := loadScrollApp(t, count)
			loadScrollPointer(t, a, "press", 516, 188)
			loadScrollPointer(t, a, "move", 516, 340)
			loadScrollPointer(t, a, "release", 324, 392)
			loadScrollState(t, a, calls, count-1, 0)
		})
	}
}

// TestLoadScrollKeepsTrackArrowAndWheelActions: the bar acts on the press
// (MENU-119): the track below the thumb pages as Page Down does (MENU-120),
// the endcaps step a line, and a release elsewhere changes nothing. The
// wheel moves three rows.
func TestLoadScrollKeepsTrackArrowAndWheelActions(t *testing.T) {
	a, calls := loadScrollApp(t, 27)
	loadScrollPointer(t, a, "press", 516, 260)
	loadScrollState(t, a, calls, 9, 0)
	loadScrollPointer(t, a, "move", 516, 280)
	loadScrollPointer(t, a, "release", 516, 260)
	loadScrollState(t, a, calls, 9, 0)
	loadScrollPointer(t, a, "press", 516, 300)
	loadScrollPointer(t, a, "release", 516, 300)
	loadScrollState(t, a, calls, 19, 10)
	loadScrollPointer(t, a, "press", 516, 164)
	loadScrollPointer(t, a, "release", 516, 164)
	loadScrollState(t, a, calls, 18, 10)
	loadScrollPointer(t, a, "press", 516, 336)
	loadScrollPointer(t, a, "release", 516, 164)
	loadScrollState(t, a, calls, 19, 10)
	loadScrollPointer(t, a, "wheel-down", 140, 158)
	loadScrollState(t, a, calls, 22, 13)
	loadScrollPointer(t, a, "wheel-up", 140, 158)
	loadScrollState(t, a, calls, 19, 13)
}

func TestLoadThumbUsesWindowPlacement(t *testing.T) {
	for _, scale := range []int{1, 2, 3} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			a, calls := loadScrollApp(t, 27)
			a.Layout(640*scale, 480*scale)
			loadScrollPointer(t, a, "press", 516*scale, 188*scale)
			loadScrollPointer(t, a, "move", 516*scale, 247*scale)
			loadScrollState(t, a, calls, 13, 4)
			loadScrollPointer(t, a, "release", 516*scale, 247*scale)
			loadScrollState(t, a, calls, 13, 4)
		})
	}
}
