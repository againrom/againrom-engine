package ui

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/frame"
)

// TestWindowedSizeFitsTheMonitor pins the decorated window's size: the largest
// whole multiple of the 640x480 frame inside nine tenths of each monitor side.
// The loss control is the next multiple, which must not fit.
func TestWindowedSizeFitsTheMonitor(t *testing.T) {
	for _, tc := range []struct {
		name         string
		monW, monH   int
		wantW, wantH int
	}{
		{"1080p", 1920, 1080, 1280, 960},
		{"1440p", 2560, 1440, 1280, 960},
		{"2160p", 3840, 2160, 2560, 1920},
		{"a 1366x768 laptop", 1366, 768, 640, 480},
		{"a 1440x900 laptop", 1440, 900, 640, 480},
		{"a 1024x600 netbook", 1024, 600, 640, 480},
		{"a monitor smaller than one frame", 600, 400, 540, 360},
		{"no monitor", 0, 0, MenuWindowW, MenuWindowH},
		{"a nonsense report", -1, 1080, MenuWindowW, MenuWindowH},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, h := windowedSize(tc.monW, tc.monH)
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("windowedSize(%d, %d) = %dx%d, want %dx%d", tc.monW, tc.monH, w, h, tc.wantW, tc.wantH)
			}
			if tc.monW <= 0 || tc.monH <= 0 {
				return
			}
			if w*10 > tc.monW*9 || h*10 > tc.monH*9 {
				t.Fatalf("%dx%d exceeds nine tenths of %dx%d", w, h, tc.monW, tc.monH)
			}
			if k := w / frame.W; w == k*frame.W && h == k*frame.H &&
				(k+1)*frame.W*10 <= tc.monW*9 && (k+1)*frame.H*10 <= tc.monH*9 {
				t.Fatalf("%dx%d is not the largest multiple: %d fits", w, h, k+1)
			}
		})
	}
}

// TestWindowPlanPerPlatform pins both forms on each platform family. The
// whole-screen rows are the start a launch without -window has always had.
func TestWindowPlanPerPlatform(t *testing.T) {
	for _, tc := range []struct {
		name       string
		goos       string
		monW, monH int
		windowed   bool
		want       windowForm
	}{
		{"windows whole screen", "windows", 1920, 1080, false, windowForm{W: 1920, H: 1080, Place: true}},
		{"linux whole screen", "linux", 2560, 1440, false, windowForm{W: 2560, H: 1440, Place: true}},
		{"freebsd whole screen", "freebsd", 1920, 1080, false, windowForm{W: 1920, H: 1080, Place: true}},
		{"mac whole screen", "darwin", 1440, 900, false, windowForm{W: 1440, H: 900, Decorated: true, Fullscreen: true}},
		{"windows window", "windows", 1920, 1080, true, windowForm{W: 1280, H: 960, Decorated: true, Place: true, X: 320, Y: 60}},
		{"linux window", "linux", 2560, 1440, true, windowForm{W: 1280, H: 960, Decorated: true, Place: true, X: 640, Y: 240}},
		{"mac window", "darwin", 1440, 900, true, windowForm{W: 640, H: 480, Decorated: true, Place: true, X: 400, Y: 210}},
		{"no monitor, whole screen", "windows", 0, 0, false, windowForm{W: MenuWindowW, H: MenuWindowH, Decorated: true}},
		{"no monitor, window", "darwin", 0, 0, true, windowForm{W: MenuWindowW, H: MenuWindowH, Decorated: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := windowPlan(tc.monW, tc.monH, tc.goos, tc.windowed); got != tc.want {
				t.Fatalf("windowPlan = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestAltEnterIsTheToggleAndNotEnter: Enter with Alt held switches the window
// and is not Enter; Enter alone is Enter; Alt alone is neither.
func TestAltEnterIsTheToggleAndNotEnter(t *testing.T) {
	for _, tc := range []struct {
		alt, enter        bool
		wantEnter, toggle bool
	}{
		{false, false, false, false},
		{true, false, false, false},
		{false, true, true, false},
		{true, true, false, true},
	} {
		enter, toggle := splitAltEnter(tc.alt, tc.enter)
		if enter != tc.wantEnter || toggle != tc.toggle {
			t.Errorf("splitAltEnter(alt=%t, enter=%t) = enter %t toggle %t, want %t %t",
				tc.alt, tc.enter, enter, toggle, tc.wantEnter, tc.toggle)
		}
	}
}

// TestAltEnterTogglesTheWindowOnEveryScreen drives the toggle through App.step
// on the main menu, the map picker, the mission map and the town, and checks
// that each press flips the form, hands the window system the matching form,
// and leaves the screen where it was. A tick without the toggle is the loss
// control: it flips nothing.
func TestAltEnterTogglesTheWindowOnEveryScreen(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a := newTestApp(t, appRows(3), okLoader(t))
	rec := &windowRecorder{}
	a.window = rec
	a.SetWindowed(true)
	toggle := func(screen Screen) {
		t.Helper()
		before := a.Windowed()
		a.step(appInput{CursorX: -1, CursorY: -1}, now)
		if a.Windowed() != before || len(rec.forms) != 0 {
			t.Fatalf("%v: a tick without Alt+Enter changed the window", screen)
		}
		a.step(appInput{WindowToggle: true, CursorX: -1, CursorY: -1}, now)
		if a.Windowed() == before {
			t.Fatalf("%v: Alt+Enter left the window form at windowed=%t", screen, before)
		}
		monW, monH := monitorSize()
		if want := windowPlan(monW, monH, runtime.GOOS, a.Windowed()); len(rec.forms) != 1 || rec.forms[0] != want {
			t.Fatalf("%v: the window system was given %+v, want one %+v", screen, rec.forms, want)
		}
		if a.Screen() != screen {
			t.Fatalf("Alt+Enter moved the screen from %v to %v", screen, a.Screen())
		}
		rec.forms = rec.forms[:0]
	}
	toggle(ScreenMenu)
	a.flow.screen = ScreenPicker
	toggle(ScreenPicker)
	a.step(appInput{Enter: true}, now)
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	cam := a.flow.viewer.Camera()
	x, y, z := cam.X, cam.Y, cam.Zoom
	toggle(ScreenMap)
	if cam.X != x || cam.Y != y || cam.Zoom != z {
		t.Fatalf("Alt+Enter moved the mission camera from (%v,%v)x%v to (%v,%v)x%v", x, y, z, cam.X, cam.Y, cam.Zoom)
	}
	a.flow.screen = ScreenTown
	toggle(ScreenTown)
}

// windowRecorder is a windowSystem that records each call in order.
type windowRecorder struct {
	maximized bool
	calls     []string
	forms     []windowForm
}

func (r *windowRecorder) Maximized() bool { return r.maximized }

func (r *windowRecorder) Restore() {
	r.calls = append(r.calls, "restore")
	r.maximized = false
}

func (r *windowRecorder) Apply(f windowForm) {
	r.calls = append(r.calls, "apply")
	r.forms = append(r.forms, f)
}

// TestAltEnterRestoresAMaximizedWindowBeforeTheWholeScreen: outside macOS the
// engine ignores a size request on a maximized window, so Alt+Enter from a
// maximized window must restore it before the whole-screen form is applied.
// The loss control is a window that is not maximized: it gets the form alone.
func TestAltEnterRestoresAMaximizedWindowBeforeTheWholeScreen(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, tc := range []struct {
		name      string
		maximized bool
		want      []string
	}{
		{"maximized", true, []string{"restore", "apply"}},
		{"not maximized", false, []string{"apply"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestApp(t, appRows(3), okLoader(t))
			rec := &windowRecorder{maximized: tc.maximized}
			a.window = rec
			a.SetWindowed(true)
			a.step(appInput{WindowToggle: true, CursorX: -1, CursorY: -1}, now)
			if a.Windowed() {
				t.Fatal("Alt+Enter left the window form")
			}
			if strings.Join(rec.calls, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("window calls %q, want %q", rec.calls, tc.want)
			}
			monW, monH := monitorSize()
			if want := windowPlan(monW, monH, runtime.GOOS, false); len(rec.forms) != 1 || rec.forms[0] != want {
				t.Fatalf("the window system was given %+v, want one %+v", rec.forms, want)
			}
		})
	}
}
