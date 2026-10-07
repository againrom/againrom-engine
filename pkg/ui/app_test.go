package ui

// Tests for the front-end shell's dispatch.
//
// The shell is the one layer that touches the engine, so what can be asserted
// here is bounded by what Ebitengine allows without a graphics context. Draw
// calls all run (NewImage, WritePixels, DrawImage, Fill, DebugPrintAt); reading
// pixels back does not — ReadPixels and At panic before the game starts. So the
// dispatch is tested exactly, the drawing only for not panicking, and what a
// human would see is left to the developer-run criterion rather than claimed
// here.

import (
	"errors"
	"image"
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/internal/synth"
	"againrom/pkg/render/frame"
	"againrom/pkg/render/menu"
	"againrom/pkg/render/terrain"
)

// hotIndex is the mask index the decoded contract assigns to each button.
var hotIndex = [menu.ButtonCount]byte{0x80, 0x90, 0xa0, 0xb0, 0xc0, 0xd0, 0xe0, 0xf0}

// maskBlock is the frame rectangle this fixture gives button b's mask region.
//
// The regions are DISJOINT, and deliberately not the placement rectangles: those
// are bounding boxes of irregular brooch shapes and they overlap one another, so
// a mask painted at them would have later buttons cover earlier ones' centres and
// no probe point would identify a button uniquely. The real mask's regions are
// disjoint; the placement table's overlap is exactly why the contract says only
// the mask decides what the cursor is over.
func maskBlock(button int) image.Rectangle {
	x := 16 + (button-1)*72
	return image.Rect(x, 300, x+48, 348)
}

// appAssets builds a synthetic menu asset set whose mask carries each button's
// hot index across that button's own disjoint block.
func appAssets(t *testing.T) *menu.Assets {
	t.Helper()

	mask := image.NewPaletted(image.Rect(0, 0, menu.FrameW, menu.FrameH), synth.GrayRamp())
	for b := 1; b <= menu.ButtonCount; b++ {
		r := maskBlock(b)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				mask.Pix[y*mask.Stride+x] = hotIndex[b-1]
			}
		}
	}

	files := synth.MenuFiles(synth.MenuOptions{
		Prefix:  menu.EntryPrefix,
		Hover:   menu.HoverRects,
		Pressed: menu.PressedRects,
		Edits: map[string][]byte{
			menu.EntryPrefix + menu.MaskEntry: synth.BMP8(mask),
		},
	})
	a, err := menu.Load(appSource(files))
	if err != nil {
		t.Fatalf("menu.Load: %v", err)
	}
	return a
}

type appSource map[string][]byte

func (s appSource) ReadFile(name string) ([]byte, error) {
	b, ok := s[name]
	if !ok {
		return nil, errors.New("absent")
	}
	return b, nil
}

func appRows(n int) []PickerRow {
	rows := make([]PickerRow, n)
	for i := range rows {
		rows[i] = PickerRow{Text: "map", Choosable: true}
	}
	return rows
}

func newTestApp(t *testing.T, rows []PickerRow, load MapLoader) *App {
	t.Helper()
	a := NewApp("t", appAssets(t), rows, load)
	a.Layout(frame.W, frame.H) // scale 1: window coordinates are frame coordinates
	return a
}

// okLoader returns a viewer over a synthetic grid and neither a tick nor an
// order seam, as the shipped loader does: a map screen with nothing running
// under it.
func okLoader(t *testing.T) MapLoader {
	t.Helper()
	return func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, err := NewViewer("m", grid(60, 60), &terrain.Tileset{})
		return v, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
}

// centreOf returns a window position inside a button's mask region. At scale 1
// window coordinates are frame coordinates.
func centreOf(button int) (int, int) {
	r := maskBlock(button)
	return (r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2
}

func TestAppDispatch(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	t.Run("the app opens on the menu", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		if a.Screen() != ScreenMenu {
			t.Fatalf("Screen() = %v, want ScreenMenu", a.Screen())
		}
	})

	t.Run("the cursor selects the button under it and nothing else", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		for b := 1; b <= menu.ButtonCount; b++ {
			x, y := centreOf(b)
			a.step(appInput{CursorX: x, CursorY: y}, now)
			if got := a.sel.State().Selected; got != b {
				t.Errorf("cursor at button %d's centre selected %d", b, got)
			}
			if a.sel.State().Pressed {
				t.Errorf("button %d reported pressed with no press", b)
			}
		}
	})

	t.Run("a position in the letterbox selects nothing", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		// A window taller than 4:3 letterboxes top and bottom.
		a.Layout(frame.W, frame.H+200)
		if _, ok := a.place.WindowToFrame(10, 10); ok {
			t.Fatalf("fixture: (10,10) is not in the letterbox at this window size")
		}
		a.step(appInput{CursorX: 10, CursorY: 10}, now)
		if got := a.sel.State().Selected; got != 0 {
			t.Errorf("a letterbox position selected button %d, want none", got)
		}
	})

	t.Run("press and release on NEW GAME opens the picker", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		x, y := centreOf(menu.NewGameButton)

		a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now)
		if !a.sel.State().Pressed {
			t.Fatalf("the press did not latch NEW GAME")
		}
		if a.Screen() != ScreenMenu {
			t.Fatalf("a press alone changed the screen to %v", a.Screen())
		}

		a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now)
		if a.Screen() != ScreenPicker {
			t.Fatalf("Screen() = %v after activating NEW GAME, want ScreenPicker", a.Screen())
		}
	})

	t.Run("a release away from the latched button activates nothing", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		x, y := centreOf(menu.NewGameButton)
		other := 0
		for b := 1; b <= menu.ButtonCount; b++ {
			if b != menu.NewGameButton {
				other = b
				break
			}
		}
		ox, oy := centreOf(other)

		a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now)
		a.step(appInput{CursorX: ox, CursorY: oy, PrimaryReleased: true}, now)
		if a.Screen() != ScreenMenu {
			t.Errorf("releasing off the latched button moved to %v", a.Screen())
		}
	})

	t.Run("press and release on EXIT quits", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		x, y := centreOf(menu.ExitButton)

		if exit := a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now); exit {
			t.Fatalf("a press alone on EXIT exited the program")
		}
		if exit := a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now); !exit {
			t.Errorf("press and release on EXIT did not exit")
		}
	})

	t.Run("EXIT released away from itself does not quit", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		x, y := centreOf(menu.ExitButton)
		ox, oy := centreOf(menu.NewGameButton)

		a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now)
		if exit := a.step(appInput{CursorX: ox, CursorY: oy, PrimaryReleased: true}, now); exit {
			t.Errorf("a press on EXIT released elsewhere exited the program")
		}
		if a.Screen() != ScreenMenu {
			t.Errorf("that release moved to %v, want the menu unchanged", a.Screen())
		}
	})

	t.Run("press and release on LOAD GAME opens the load window", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		x, y := centreOf(menu.LoadGameButton)

		if exit := a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now); exit {
			t.Fatalf("a press alone on LOAD GAME exited the program")
		}
		if a.Screen() != ScreenMenu {
			t.Fatalf("a press alone changed the screen to %v", a.Screen())
		}

		if exit := a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now); exit {
			t.Fatalf("activating LOAD GAME exited the program")
		}
		if a.Screen() != ScreenLoad {
			t.Errorf("Screen() = %v after activating LOAD GAME, want ScreenLoad", a.Screen())
		}
	})

	t.Run("LOAD GAME released away from itself does nothing", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		x, y := centreOf(menu.LoadGameButton)
		ox, oy := centreOf(menu.NewGameButton)

		a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now)
		a.step(appInput{CursorX: ox, CursorY: oy, PrimaryReleased: true}, now)
		if a.Screen() != ScreenMenu {
			t.Errorf("that release moved to %v, want the menu unchanged", a.Screen())
		}
	})

	t.Run("the remaining unbound buttons are consumed and do nothing", func(t *testing.T) {
		n := 0
		for b := 1; b <= menu.ButtonCount; b++ {
			if b == menu.NewGameButton || b == menu.ExitButton || b == menu.LoadGameButton || b == menu.CutscenesButton || b == menu.CreditsButton {
				continue
			}
			n++
			a := newTestApp(t, appRows(3), okLoader(t))
			x, y := centreOf(b)
			a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now)
			if exit := a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now); exit {
				t.Errorf("activating button %d exited the program", b)
			}
			if a.Screen() != ScreenMenu {
				t.Errorf("activating button %d moved to %v, want the menu unchanged", b, a.Screen())
			}
		}
		if n != menu.ButtonCount-5 {
			t.Errorf("checked %d unbound buttons, want %d — five are bound in this fixture",
				n, menu.ButtonCount-5)
		}
	})

	t.Run("the menu reads no key but Esc", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		for _, in := range []appInput{{Up: true}, {Down: true}, {Enter: true}} {
			if exit := a.step(in, now); exit {
				t.Fatalf("input %+v exited the program", in)
			}
			if a.Screen() != ScreenMenu {
				t.Fatalf("input %+v moved to %v", in, a.Screen())
			}
		}
		if exit := a.step(appInput{Escape: true}, now); !exit {
			t.Errorf("Esc at the menu did not exit")
		}
	})

	t.Run("the picker takes arrows, Enter and a click", func(t *testing.T) {
		a := newTestApp(t, appRows(10), okLoader(t))
		a.flow.screen = ScreenPicker

		a.step(appInput{Down: true}, now)
		a.step(appInput{Down: true}, now)
		if got := a.flow.picker.Selection(); got != 2 {
			t.Errorf("selection after two Downs = %d, want 2", got)
		}
		a.step(appInput{Up: true}, now)
		if got := a.flow.picker.Selection(); got != 1 {
			t.Errorf("selection after an Up = %d, want 1", got)
		}

		a.step(appInput{Enter: true}, now)
		if a.Screen() != ScreenMap {
			t.Fatalf("Enter did not open the map screen (screen = %v)", a.Screen())
		}
	})

	t.Run("the wheel scrolls the picker and chooses nothing", func(t *testing.T) {
		a := newTestApp(t, appRows(pickerTestRows), okLoader(t))
		a.flow.screen = ScreenPicker
		p := a.flow.picker

		// Towards the end of the list, one notch at a time.
		a.step(appInput{WheelY: -1}, now)
		if got := p.Selection(); got != pickerWheelRows {
			t.Fatalf("one notch towards the user: selection = %d, want %d", got, pickerWheelRows)
		}
		if a.Screen() != ScreenPicker {
			t.Fatalf("the wheel chose a row (screen = %v)", a.Screen())
		}

		// ...and back towards the start.
		a.step(appInput{WheelY: +1}, now)
		if got := p.Selection(); got != 0 {
			t.Fatalf("one notch away from the user: selection = %d, want 0", got)
		}

		// The window follows once the selection leaves it, and clamps at the end.
		for i := 0; i < pickerTestRows; i++ {
			a.step(appInput{WheelY: -1}, now)
		}
		if got := p.Selection(); got != pickerTestRows-1 {
			t.Fatalf("rolling past the end: selection = %d, want %d", got, pickerTestRows-1)
		}
		if top, n := p.Visible(); top+n != pickerTestRows {
			t.Fatalf("rolling past the end: Visible() = (%d, %d), want the window at the bottom",
				top, n)
		}
		if a.Screen() != ScreenPicker {
			t.Fatalf("rolling past the end left the picker (screen = %v)", a.Screen())
		}

		// ...and clamps at the start.
		for i := 0; i < pickerTestRows; i++ {
			a.step(appInput{WheelY: +1}, now)
		}
		if got := p.Selection(); got != 0 {
			t.Fatalf("rolling past the start: selection = %d, want 0", got)
		}
		if top, _ := p.Visible(); top != 0 {
			t.Fatalf("rolling past the start: Visible() top = %d, want 0", top)
		}
	})

	t.Run("the wheel on the menu and the map screen leaves the picker alone", func(t *testing.T) {
		for _, screen := range []Screen{ScreenMenu, ScreenMap} {
			a := newTestApp(t, appRows(pickerTestRows), okLoader(t))
			a.flow.screen = screen

			before := a.flow.picker.Selection()
			beforeTop, beforeN := a.flow.picker.Visible()
			for i := 0; i < 5; i++ {
				a.step(appInput{WheelY: -1}, now)
			}
			if got := a.flow.picker.Selection(); got != before {
				t.Errorf("%v: the wheel moved the picker's selection from %d to %d",
					screen, before, got)
			}
			if top, n := a.flow.picker.Visible(); top != beforeTop || n != beforeN {
				t.Errorf("%v: the wheel moved the picker's window from (%d, %d) to (%d, %d)",
					screen, beforeTop, beforeN, top, n)
			}
			if a.Screen() != screen {
				t.Errorf("%v: the wheel changed the screen to %v", screen, a.Screen())
			}
		}
	})

	t.Run("a click chooses the row under the cursor", func(t *testing.T) {
		a := newTestApp(t, appRows(10), okLoader(t))
		a.flow.screen = ScreenPicker

		// Row 4's line, in frame pixels; at scale 1 these are window pixels.
		const wantRow = 4
		y := pickerTop + wantRow*pickerLine
		if _, ok := a.flow.picker.RowAt(image.Pt(pickerLeft, y)); !ok {
			t.Fatalf("fixture: no row at y=%d", y)
		}
		a.step(appInput{CursorX: pickerLeft, CursorY: y, PrimaryReleased: true}, now)

		if got := a.flow.picker.Selection(); got != wantRow {
			t.Errorf("clicked row %d, selection = %d", wantRow, got)
		}
		if a.Screen() != ScreenMap {
			t.Errorf("the click did not open the map screen (screen = %v)", a.Screen())
		}
	})

	t.Run("Esc unwinds map to picker to menu, then exits", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		x, y := centreOf(menu.NewGameButton)
		a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now)
		a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now)
		a.step(appInput{Enter: true}, now)
		if a.Screen() != ScreenMap {
			t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
		}

		for _, want := range []Screen{ScreenPicker, ScreenMenu} {
			if exit := a.step(appInput{Escape: true}, now); exit {
				t.Fatalf("Esc exited before reaching the menu")
			}
			leaveViaMenu(a.flow) //
			if a.Screen() != want {
				t.Fatalf("Esc landed on %v, want %v", a.Screen(), want)
			}
		}
		if exit := a.step(appInput{Escape: true}, now); !exit {
			t.Errorf("Esc at the menu did not exit")
		}
	})

	t.Run("a failing load reports and stays in the picker", func(t *testing.T) {
		boom := errors.New("decode blew up")
		a := newTestApp(t, appRows(3), func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, boom
		})
		a.flow.screen = ScreenPicker

		if exit := a.step(appInput{Enter: true}, now); exit {
			t.Fatalf("a failing load exited the program")
		}
		if a.Screen() != ScreenPicker {
			t.Errorf("screen = %v after a failing load, want ScreenPicker", a.Screen())
		}
		if a.flow.msg == "" {
			t.Errorf("a failing load left no message to report")
		}
	})
}

// TestDefaultWindowIsIntegerScaled pins the one property MenuWindowW/H has to
// have: the brooch is reproduced pixel for pixel.
//
// The number itself (2) is a judgement about desktops and is allowed to change;
// what must not change is that it is an integer, because the frame is blitted
// with nearest-neighbour filtering and a fractional factor would resample the one
// screen this project renders exactly.
func TestDefaultWindowIsIntegerScaled(t *testing.T) {
	if MenuWindowW != frame.W*MenuWindowScale || MenuWindowH != frame.H*MenuWindowScale {
		t.Fatalf("default window is %dx%d, want an exact %dx multiple of the %dx%d frame",
			MenuWindowW, MenuWindowH, MenuWindowScale, frame.W, frame.H)
	}
	if MenuWindowScale < 1 {
		t.Fatalf("MenuWindowScale = %d, want at least 1", MenuWindowScale)
	}

	// A freshly built App has adopted that size before Layout is ever called, so
	// the first frame drawn is already at the exact scale.
	a := NewApp("t", appAssets(t), appRows(3), nil)
	if !a.place.Valid() {
		t.Fatalf("the default placement is not valid")
	}
	if got := a.place.Scale(); got != float64(MenuWindowScale) {
		t.Errorf("scale at the default window = %v, want exactly %d — a fractional factor "+
			"resamples the brooch under nearest-neighbour filtering", got, MenuWindowScale)
	}
	if ox, oy := a.place.Origin(); ox != 0 || oy != 0 {
		t.Errorf("origin at the default window = (%v, %v), want (0, 0) — an exact multiple "+
			"leaves no letterbox", ox, oy)
	}

	// Every frame pixel still maps, at both corners, which is what "nothing is
	// cropped" means at this size.
	for _, p := range []image.Point{{X: 0, Y: 0}, {X: frame.W - 1, Y: frame.H - 1}} {
		wx, wy, ok := a.place.FrameToWindow(p)
		if !ok {
			t.Fatalf("frame pixel %v maps to no window pixel at the default size", p)
		}
		if got, ok := a.place.WindowToFrame(wx, wy); !ok || got != p {
			t.Fatalf("round trip of frame pixel %v gave (%v, %v)", p, got, ok)
		}
	}
}

// TestStartupWindowCoversTheScreen pins both of startupWindow's answers: the
// monitor's own size whenever the monitor reports one at all, and the old
// decorated window when it does not.
//
// The fallback must not replace a valid small monitor: a 1024x600 display is
// a display, and a 1280x960 window on it hangs off two edges at once.
func TestStartupWindowCoversTheScreen(t *testing.T) {
	for _, tc := range []struct {
		name         string
		monW, monH   int
		wantW, wantH int
		wantWhole    bool
	}{
		{"a 1080p desktop", 1920, 1080, 1920, 1080, true},
		{"a 1440p desktop", 2560, 1440, 2560, 1440, true},
		{"a display smaller than the menu window", 1024, 600, 1024, 600, true},
		{"exactly the menu window", MenuWindowW, MenuWindowH, MenuWindowW, MenuWindowH, true},
		{"no monitor to ask", 0, 0, MenuWindowW, MenuWindowH, false},
		{"a nonsense report", -1, 1080, MenuWindowW, MenuWindowH, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, h, whole := startupWindow(tc.monW, tc.monH)
			if w != tc.wantW || h != tc.wantH || whole != tc.wantWhole {
				t.Errorf("startupWindow(%d, %d) = %dx%d whole=%v, want %dx%d whole=%v",
					tc.monW, tc.monH, w, h, whole, tc.wantW, tc.wantH, tc.wantWhole)
			}
			// Whatever it answers is a window Ebitengine will accept:
			// SetWindowSize panics on a non-positive extent, and the fallback
			// exists precisely because the reported size can be one.
			if w <= 0 || h <= 0 {
				t.Errorf("startupWindow(%d, %d) = %dx%d, which SetWindowSize panics on",
					tc.monW, tc.monH, w, h)
			}
		})
	}
}

func TestStartupWindowRequestsNativeFullscreenOnMac(t *testing.T) {
	w, h := ebiten.WindowSize()
	fullscreen, decorated, resizing := ebiten.IsFullscreen(), ebiten.IsWindowDecorated(), ebiten.WindowResizingMode()
	t.Cleanup(func() {
		ebiten.SetFullscreen(fullscreen)
		ebiten.SetWindowDecorated(decorated)
		ebiten.SetWindowSize(w, h)
		ebiten.SetWindowResizingMode(resizing)
	})
	for _, tc := range []struct {
		name, goos            string
		monW, monH            int
		fullscreen, decorated bool
	}{
		{"mac fullscreen", "darwin", 1280, 800, true, true},
		{"windows borderless", "windows", 1920, 1080, false, false},
		{"linux borderless", "linux", 2560, 1440, false, false},
		{"mac fallback", "darwin", 0, 0, false, true},
		{"windows fallback", "windows", 0, 0, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configureStartupWindow(tc.monW, tc.monH, tc.goos)
			if got := ebiten.IsFullscreen(); got != tc.fullscreen {
				t.Errorf("fullscreen = %v, want %v", got, tc.fullscreen)
			}
			if got := ebiten.IsWindowDecorated(); got != tc.decorated {
				t.Errorf("decorated = %v, want %v", got, tc.decorated)
			}
			wantW, wantH, _ := startupWindow(tc.monW, tc.monH)
			if gotW, gotH := ebiten.WindowSize(); gotW != wantW || gotH != wantH {
				t.Errorf("window = %dx%d, want %dx%d", gotW, gotH, wantW, wantH)
			}
		})
	}
}

// TestMapScreenRoutesThroughStep is the front-end's half of the non-divergence
// guarantee.
//
// It is a STRUCTURAL witness, and worth being plain about what it does and does
// not show. It does not independently prove the camera's behaviour — that is
// TestViewerStep's job, against the camera contract. What it shows is that the
// front-end's map screen advances the camera through the same method the
// standalone viewer's Update calls, so the two cannot offer different camera
// behaviour. It would fail the moment the front-end grew camera handling of its
// own.
func TestMapScreenRoutesThroughStep(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)

	const cols, rows, winW, winH = 60, 60, 800, 600
	mk := func() *Viewer {
		v, err := NewViewer("m", grid(cols, rows), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		layoutViewport(v, winW, winH)
		v.Camera().X, v.Camera().Y = 400, 400
		v.Camera().Clamp()
		return v
	}

	// The front-end, sitting on the map screen over one viewer...
	a := NewApp("t", appAssets(t), appRows(1), nil)
	a.Layout(winW, winH)
	front := mk()
	a.flow.viewer = front
	a.flow.screen = ScreenMap

	// ...and a standalone viewer built and sized identically.
	alone := mk()

	inputs := []Input{
		{PanRight: true, CursorX: winW / 2, CursorY: winH / 2},
		{PanDown: true, CursorX: winW / 2, CursorY: winH / 2},
		{CursorX: 0, CursorY: 0},
		{CursorX: winW / 2, CursorY: winH / 2, WheelY: +1},
		{CursorX: winW - 1, CursorY: winH - 1},
		{CursorX: winW / 2, CursorY: winH / 2, WheelY: -1},
	}
	for i, in := range inputs {
		a.step(appInput{Viewer: in, CursorX: in.CursorX, CursorY: in.CursorY}, now)
		alone.step(in, now)

		fc, ac := front.Camera(), alone.Camera()
		if fc.X != ac.X || fc.Y != ac.Y || fc.Zoom != ac.Zoom {
			t.Fatalf("step %d: front-end camera (%v,%v,z=%v) != standalone (%v,%v,z=%v)",
				i, fc.X, fc.Y, fc.Zoom, ac.X, ac.Y, ac.Zoom)
		}
	}

	// Non-vacuity: the sequence must actually have moved the camera, or the
	// comparison above would hold for a front-end that did nothing at all.
	if front.Camera().X == 400 && front.Camera().Y == 400 && front.Camera().Zoom == 1 {
		t.Fatalf("the input sequence moved nothing; the comparison proves nothing")
	}
}

// TestAppDrawDoesNotPanic exercises the draw path on every screen.
//
// Only that it runs: an *ebiten.Image's pixels cannot be read back before the
// game starts, so what the frame actually contains is not observable here. The
// composition itself is asserted at the pixel level one tier down, where it
// returns a plain image.
func TestAppDrawDoesNotPanic(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a := newTestApp(t, appRows(38), okLoader(t))
	screen := ebiten.NewImage(frame.W, frame.H)

	a.Draw(screen) // menu

	x, y := centreOf(menu.NewGameButton)
	a.step(appInput{CursorX: x, CursorY: y}, now)
	a.Draw(screen) // menu, hovered
	a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now)
	a.Draw(screen) // menu, pressed

	a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now)
	a.Draw(screen) // picker

	a.step(appInput{Down: true}, now)
	a.Draw(screen) // picker, scrolled selection

	a.step(appInput{Enter: true}, now)
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	a.Draw(screen) // map

	// A window that letterboxes, and a degenerate one.
	a.step(appInput{Escape: true}, now)
	a.Layout(1000, 480)
	a.Draw(screen)
	a.Layout(0, 0)
	a.Draw(screen)
}

func TestMapTickAdvancesOncePerMapScreenTick(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)

	// Ticks are 100 ms apart — past the 62 ms the water phase takes at the speed
	// index map load selects — so the water counter is observably moving on the
	// ticks the advance count rises.
	at := func(i int) time.Time { return base.Add(time.Duration(i) * 100 * time.Millisecond) }

	// The cursor sits away from every window edge, so the camera moves by what a
	// subtest asks for and not by an edge-scroll it did not ask for.
	const cx, cy = frame.W / 2, frame.H / 2

	// appOnMap parks the front-end on the map screen over a counting loader,
	// REACHED through the picker rather than assigned, and returns it with the
	// counters the loads handed out.
	appOnMap := func(t *testing.T) (*App, *[]*int) {
		t.Helper()
		counts := &[]*int{}
		a := newTestApp(t, appRows(3), tickLoader(t, counts))
		a.flow.screen = ScreenPicker
		a.step(appInput{Enter: true}, at(0))
		if a.Screen() != ScreenMap {
			t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
		}
		if len(*counts) != 1 {
			t.Fatalf("setup: the loader handed out %d ticks, want 1", len(*counts))
		}
		return a, counts
	}

	t.Run("opening a map advances nothing", func(t *testing.T) {
		_, counts := appOnMap(t)
		if got := *(*counts)[0]; got != 0 {
			t.Errorf("the tick that opened the map advanced the world %d times, want 0", got)
		}
	})

	// SC-1: "exactly one call per map-screen tick — under neutral input and under
	// pan, drag and wheel".
	t.Run("exactly one advance per map-screen tick, whatever the input", func(t *testing.T) {
		a, counts := appOnMap(t)
		a.flow.viewer.Camera().X, a.flow.viewer.Camera().Y = 400, 400
		a.flow.viewer.Camera().Clamp()

		inputs := []Input{
			{CursorX: cx, CursorY: cy},                                 // neutral
			{PanRight: true, CursorX: cx, CursorY: cy},                 // keyboard pan
			{PanDown: true, PanLeft: true, CursorX: cx, CursorY: cy},   // two keys at once
			{CursorX: 0, CursorY: 0},                                   // edge-scroll
			{PrimaryDown: true, CursorX: cx, CursorY: cy},              // a drag anchors
			{PrimaryDown: true, CursorX: cx - 40, CursorY: cy - 25},    // ...and moves
			{CursorX: cx, CursorY: cy, WheelY: +1},                     // zoom in
			{CursorX: cx, CursorY: cy, WheelY: -1},                     // zoom out
			{PanUp: true, PrimaryDown: true, CursorX: cx, CursorY: cy}, // pan and drag together
		}
		for i, in := range inputs {
			a.step(appInput{Viewer: in, CursorX: in.CursorX, CursorY: in.CursorY}, at(i+1))
			if got, want := *(*counts)[0], i+1; got != want {
				t.Fatalf("after %d map-screen ticks the world advanced %d times, want %d (input %+v)",
					want, got, want, in)
			}
		}

		// ...and it stays one per tick over a long neutral run, where a drift of
		// one call in several would show.
		before := *(*counts)[0]
		const runs = 50
		for i := 0; i < runs; i++ {
			a.step(appInput{Viewer: Input{CursorX: cx, CursorY: cy}}, at(len(inputs)+1+i))
		}
		if got, want := *(*counts)[0], before+runs; got != want {
			t.Errorf("after %d further ticks the world stands at %d advances, want %d", runs, got, want)
		}
	})

	t.Run("the camera and the water counter move on the tick the count rises", func(t *testing.T) {
		a, counts := appOnMap(t)
		v := a.flow.viewer
		v.Camera().X, v.Camera().Y = 400, 400
		v.Camera().Clamp()

		pan := appInput{Viewer: Input{PanRight: true, CursorX: cx, CursorY: cy}, CursorX: cx, CursorY: cy}

		// The first map-screen tick only takes the animation baseline — the
		// standalone viewer's own pinned behaviour — so the measured tick is the
		// second one.
		a.step(pan, at(1))
		camX, water, advances := v.Camera().X, v.AnimationCounter(), *(*counts)[0]

		a.step(pan, at(2))
		if got := *(*counts)[0]; got != advances+1 {
			t.Fatalf("one map-screen tick advanced the world %d times, want 1", got-advances)
		}
		if v.Camera().X <= camX {
			t.Errorf("the camera did not respond on the tick the world advanced: X %v -> %v",
				camX, v.Camera().X)
		}
		if v.AnimationCounter() <= water {
			t.Errorf("the water phase did not move on the tick the world advanced: %d -> %d",
				water, v.AnimationCounter())
		}
	})

	t.Run("the tick that leaves the map screen advances nothing and discards both", func(t *testing.T) {
		a, counts := appOnMap(t)
		a.step(appInput{Viewer: Input{CursorX: cx, CursorY: cy}}, at(1))
		if got := *(*counts)[0]; got != 1 {
			t.Fatalf("precondition: one map-screen tick gave %d advances, want 1", got)
		}

		if exit := a.step(appInput{Escape: true}, at(2)); exit {
			t.Fatalf("Esc on the map screen exited the program")
		}
		leaveViaMenu(a.flow) //
		if got := *(*counts)[0]; got != 1 {
			t.Errorf("the Esc tick advanced the world: %d advances, want the 1 it stood at", got)
		}
		if a.Screen() != ScreenPicker {
			t.Errorf("Esc landed on %v, want ScreenPicker", a.Screen())
		}
		if a.flow.viewer != nil {
			t.Errorf("the Esc tick left a viewer behind")
		}
		if a.flow.tick != nil {
			t.Errorf("the Esc tick left the advance behind — the world outlived the map screen")
		}

		// ...and the ticks that follow it, on the picker, advance nothing either.
		for i := 3; i <= 6; i++ {
			a.step(appInput{Down: true}, at(i))
		}
		if got := *(*counts)[0]; got != 1 {
			t.Errorf("picker ticks after Esc advanced the world to %d, want the 1 it stood at", got)
		}
	})

	t.Run("no advance while the menu or the picker shows", func(t *testing.T) {
		for _, screen := range []Screen{ScreenMenu, ScreenPicker} {
			a, counts := appOnMap(t)
			a.flow.screen = screen

			for i := 1; i <= 6; i++ {
				a.step(appInput{
					Viewer:         Input{PanRight: true, CursorX: cx, CursorY: cy},
					CursorX:        cx,
					CursorY:        cy,
					WheelY:         -1,
					Up:             i%2 == 0,
					Down:           i%2 == 1,
					PrimaryPressed: i == 3,
				}, at(i))
			}
			if got := *(*counts)[0]; got != 0 {
				t.Errorf("%v: six ticks on this screen advanced the world %d times, want 0", screen, got)
			}
			if a.Screen() != screen {
				t.Errorf("%v: the run changed the screen to %v", screen, a.Screen())
			}
		}
	})

	t.Run("reopening advances the second world and never the first", func(t *testing.T) {
		a, counts := appOnMap(t)
		a.step(appInput{Viewer: Input{CursorX: cx, CursorY: cy}}, at(1)) // advance the first
		a.step(appInput{Escape: true}, at(2))                            //
		leaveViaMenu(a.flow)                                             // ...and EXIT goes back to the picker
		a.step(appInput{Enter: true}, at(3))                             // and open again

		if a.Screen() != ScreenMap {
			t.Fatalf("the second choice landed on %v, want ScreenMap", a.Screen())
		}
		if len(*counts) != 2 {
			t.Fatalf("two openings called the loader for %d ticks, want 2", len(*counts))
		}

		a.step(appInput{Viewer: Input{CursorX: cx, CursorY: cy}}, at(4))
		if got := *(*counts)[0]; got != 1 {
			t.Errorf("the discarded world advanced %d times, want the 1 it was left at", got)
		}
		if got := *(*counts)[1]; got != 1 {
			t.Errorf("the reopened world advanced %d times, want 1 — the flow kept the first tick", got)
		}
	})

	t.Run("a failing load leaves neither a viewer nor a tick", func(t *testing.T) {
		boom := errors.New("decode grid: payload 512, want 2048")
		a := newTestApp(t, appRows(3), func(int) (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, boom
		})
		a.flow.screen = ScreenPicker

		if exit := a.step(appInput{Enter: true}, at(0)); exit {
			t.Fatalf("a failing load exited the program")
		}
		if a.Screen() != ScreenPicker {
			t.Errorf("screen = %v after a failing load, want ScreenPicker", a.Screen())
		}
		if a.flow.viewer != nil {
			t.Errorf("a failing load left a viewer behind")
		}
		if a.flow.tick != nil {
			t.Errorf("a failing load left an advance behind")
		}
		// The ticks that follow must not reach for what was never stored.
		for i := 1; i <= 3; i++ {
			a.step(appInput{Down: true}, at(i))
		}
	})

	t.Run("a loader that hands back no tick still opens a working map screen", func(t *testing.T) {
		a := newTestApp(t, appRows(3), okLoader(t))
		a.flow.screen = ScreenPicker
		a.step(appInput{Enter: true}, at(0))
		if a.Screen() != ScreenMap {
			t.Fatalf("screen = %v, want ScreenMap", a.Screen())
		}
		if a.flow.tick != nil {
			t.Fatalf("fixture: this loader must hand back no tick")
		}

		v := a.flow.viewer
		v.Camera().X, v.Camera().Y = 400, 400
		v.Camera().Clamp()
		startX := v.Camera().X
		for i := 1; i <= 3; i++ {
			a.step(appInput{Viewer: Input{PanRight: true, CursorX: cx, CursorY: cy}, CursorX: cx, CursorY: cy}, at(i))
		}
		if v.Camera().X <= startX {
			t.Errorf("with no tick behind it the map screen stopped moving the camera: X %v -> %v",
				startX, v.Camera().X)
		}
	})
}

func TestTheOrderLeavesTheMapArmThroughTheSeam(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	at := func(i int) time.Time { return base.Add(time.Duration(i) * 100 * time.Millisecond) }

	// One unit, on a cell the camera at the origin places well inside the
	// window, and a target cell elsewhere in the extent. The id is sparse and is
	// not the slice position, so an arm addressing its order at an index rather
	// than at the id it was handed answers with something else.
	const (
		unitID             = 9
		unitCol, unitRow   = 5, 5
		orderCol, orderRow = 10, 8
	)

	// appOnMapSeam parks the front-end on the map screen over a recording
	// loader, REACHED through the picker rather than assigned, with the camera
	// at the origin and one entity in the snapshot.
	appOnMapSeam := func(t *testing.T) (*App, *Viewer, *mapSeam) {
		t.Helper()
		seams := &[]*mapSeam{}
		a := newTestApp(t, appRows(3), seamLoader(t, seams))
		a.flow.screen = ScreenPicker
		a.step(appInput{Enter: true}, at(0))
		if a.Screen() != ScreenMap {
			t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
		}
		if len(*seams) != 1 {
			t.Fatalf("setup: the loader handed out %d seams, want 1", len(*seams))
		}
		mapAtScaleOne(a)
		v := a.flow.viewer
		v.Camera().X, v.Camera().Y = 0, 0
		v.Camera().Clamp()
		if got := v.Camera().Zoom; got != 1 {
			t.Fatalf("setup: zoom = %v, want 1", got)
		}
		v.SetEntities([]MapEntity{{ID: unitID, Cell: image.Pt(unitCol, unitRow)}})
		return a, v, (*seams)[0]
	}

	// frameAt is one tick of front-end input at a window position, with the
	// VIEWER's cursor set to the same place — which is what readAppInput does,
	// and what keeps a probe position from being read as two different cursors
	// by the camera and the pick. Every position used below sits further than
	// EdgeMargin from all four window edges, so no frame edge-scrolls and the
	// camera stands still under the whole gesture.
	frameAt := func(x, y int) appInput {
		return appInput{Viewer: Input{CursorX: x, CursorY: y}, CursorX: x, CursorY: y}
	}
	tapAt := func(x, y int) appInput {
		in := frameAt(x, y)
		in.PrimaryPressed, in.PrimaryReleased = true, true
		return in
	}
	// orderAt is the map's ACTING gesture over a point: the left button's press
	// and release on one frame, which `command` reads as a tap because nothing
	// moved between them.
	orderAt := func(x, y int) appInput { return tapAt(x, y) }

	t.Run("a tap on empty ground with a selection issues exactly one order", func(t *testing.T) {
		a, v, s := appOnMapSeam(t)

		ux, uy := cellPoint(v, unitCol, unitRow)
		a.step(tapAt(ux, uy), at(1))
		if want := (selection{unitID}); !slices.Equal(v.sel, want) {
			t.Fatalf("the tap on the unit's cell left selection %+v, want %+v", v.sel, want)
		}
		if len(s.orders) != 0 {
			t.Fatalf("the tap issued %+v, want nothing — a tap selects and orders nothing", s.orders)
		}

		ox, oy := cellPoint(v, orderCol, orderRow)
		a.step(orderAt(ox, oy), at(2))

		want := issued{entity: unitID, x: orderCol, y: orderRow}
		if len(s.orders) != 1 || s.orders[0] != want {
			t.Fatalf("the ordering tap issued %+v, want exactly [%+v]", s.orders, want)
		}
		if want := (selection{unitID}); !slices.Equal(v.sel, want) {
			t.Errorf("the order left selection %+v, want %+v", v.sel, want)
		}
	})

	t.Run("with nothing selected the seam is not reached", func(t *testing.T) {
		a, v, s := appOnMapSeam(t)
		if len(v.sel) != 0 {
			t.Fatalf("setup: a fresh map screen already holds selection %+v", v.sel)
		}

		ox, oy := cellPoint(v, orderCol, orderRow)
		for i := 1; i <= 3; i++ {
			a.step(orderAt(ox, oy), at(i))
		}
		if len(s.orders) != 0 {
			t.Errorf("three taps with nothing selected issued %+v, want nothing", s.orders)
		}
	})

	t.Run("one advance per map-screen tick, order or no order", func(t *testing.T) {
		a, v, s := appOnMapSeam(t)
		ux, uy := cellPoint(v, unitCol, unitRow)
		ox, oy := cellPoint(v, orderCol, orderRow)

		frames := []appInput{
			frameAt(ux, uy),   // neutral
			tapAt(ux, uy),     // selects
			orderAt(ox, oy),   // issues an order
			orderAt(ox, oy),   // issues a second
			frameAt(300, 300), // neutral again
		}
		for i, in := range frames {
			a.step(in, at(i+1))
			if got, want := s.ticks, i+1; got != want {
				t.Fatalf("after %d map-screen ticks the advance ran %d times, want %d (frame %d)",
					want, got, want, i)
			}
		}
		if len(s.orders) != 2 {
			t.Fatalf("the run issued %d orders, want 2 — with none this says nothing about a frame "+
				"that produced one", len(s.orders))
		}
	})

	t.Run("the release is judged after the camera step", func(t *testing.T) {
		a, v, s := appOnMapSeam(t)
		if len(v.sel) != 0 {
			t.Fatalf("setup: a fresh map screen already holds selection %+v", v.sel)
		}

		held := func(x, y int) appInput {
			in := frameAt(x, y)
			in.Viewer.PrimaryDown = true
			return in
		}
		// The press is on the top-left corner of the unit's own cell and the
		// release two cells down and right of it, so the rectangle covers the
		// whole of that unit's drawable rectangle — which a plain marquee needs,
		// since it takes a unit only where it covers strictly more than half of
		// it (`AI-SELECT-122`). The travel is two cells in each axis at this
		// fixture's native zoom, well past the slop, and every pixel of it lands
		// on the release frame.
		//
		// IT USED TO BE A HORIZONTAL DRAG of 40 px. A line has no area, so under
		// the decoded rule it would qualify nobody and the case could not tell
		// the two readings apart. The far corner is two cells away rather than
		// one so that the release POSITION lands on a cell the unit is not on,
		// which is what makes the tap below a real discriminator.
		px, py, qx, qy := cellSpan(v, unitCol, unitRow, unitCol+1, unitRow+1)
		press := held(px, py)
		press.PrimaryPressed = true
		a.step(press, at(1)) // anchors, travels nothing

		release := held(qx, qy)
		release.PrimaryReleased = true
		a.step(release, at(2)) // travels two whole cells on this very frame

		if want := (selection{unitID}); !slices.Equal(v.sel, want) {
			t.Errorf("a box whose travel landed on the release frame left selection %+v, want %+v "+
				"— the release was judged before that frame's movement reached the accumulator", v.sel, want)
		}
		if len(s.orders) != 0 {
			t.Errorf("the drag issued %+v, want nothing", s.orders)
		}

		// Non-vacuity: the WRONG reading, performed on a fresh fixture. A tap at
		// the release position with nothing selected is exactly what "judged
		// before the movement reached the accumulator" would have made of that
		// one frame, and it leaves the run where it began — no selection and no
		// order. Without this the assertion above would also pass against an
		// implementation that selected the unit for some other reason.
		//
		// It runs on a SECOND fixture rather than on this one because the
		// selection standing here changes what a tap at that position does: with
		// a unit selected, empty ground puts up `move` and the tap is an order
		// (`AI-CLICK-050`), which is the correct behaviour and not the reading
		// under test.
		a2, v2, s2 := appOnMapSeam(t)
		a2.step(tapAt(qx, qy), at(1))
		if len(v2.sel) != 0 {
			t.Fatalf("a tap on the release position left selection %+v, want none — at this position the "+
				"two readings of the drag are indistinguishable and the subtest above says nothing", v2.sel)
		}
		if len(s2.orders) != 0 {
			t.Fatalf("a tap on the release position with nothing selected issued %+v, want nothing", s2.orders)
		}
	})

	t.Run("a loader that hands back no order seam still opens a working map screen", func(t *testing.T) {
		counts := &[]*int{}
		a := newTestApp(t, appRows(3), tickLoader(t, counts))
		a.flow.screen = ScreenPicker
		a.step(appInput{Enter: true}, at(0))
		if a.Screen() != ScreenMap {
			t.Fatalf("screen = %v, want ScreenMap", a.Screen())
		}
		if a.flow.order != nil {
			t.Fatalf("fixture: this loader must hand back no order seam")
		}
		mapAtScaleOne(a)

		v := a.flow.viewer
		v.Camera().X, v.Camera().Y = 0, 0
		v.Camera().Clamp()
		v.SetEntities([]MapEntity{{ID: unitID, Cell: image.Pt(unitCol, unitRow)}})

		ux, uy := cellPoint(v, unitCol, unitRow)
		a.step(tapAt(ux, uy), at(1))
		if want := (selection{unitID}); !slices.Equal(v.sel, want) {
			t.Fatalf("the tap left selection %+v, want %+v", v.sel, want)
		}

		ox, oy := cellPoint(v, orderCol, orderRow)
		a.step(orderAt(ox, oy), at(2))
		if want := (selection{unitID}); !slices.Equal(v.sel, want) {
			t.Errorf("the ordering tap over a nil seam left selection %+v, want %+v", v.sel, want)
		}
		if got := *(*counts)[0]; got != 2 {
			t.Errorf("the two map-screen ticks advanced the world %d times, want 2", got)
		}
	})

	// 0030 T4 (AC-4): k selected members leave this arm as k CALLS into the
	// seam it already holds, in ascending entity id, each naming the one
	// resolved cell — and the arm's single advance is unmoved whether the
	// frame produced none, one or many.
	//
	// The seam is the shipped three-scalar one, so what is asserted here is the
	// far side's whole view: how many calls arrived, in what order, with what
	// payload.
	t.Run("k selected members reach the seam as k calls in ascending id", func(t *testing.T) {
		a, v, s := appOnMapSeam(t)
		// Three units, ids SPARSE and not their slice positions, on cells the box
		// below covers; the box is dragged corner to corner through the app's own
		// dispatch, so the latch, the press point and the accumulator are the
		// ones the map arm's own camera step wrote.
		v.SetEntities([]MapEntity{
			{ID: 4, Cell: image.Pt(3, 3)},
			{ID: 9, Cell: image.Pt(5, 5)},
			{ID: 12, Cell: image.Pt(4, 4)},
		})

		held := func(x, y int) appInput {
			in := frameAt(x, y)
			in.Viewer.PrimaryDown = true
			return in
		}
		// CORNER TO CORNER, not centre to centre: a centre-to-centre drag covers
		// a quarter of each end unit's rectangle and a plain marquee needs
		// strictly more than half (`AI-SELECT-122`), so it would take only the
		// middle one.
		ax, ay, bx, by := cellSpan(v, 3, 3, 5, 5)
		press := held(ax, ay)
		press.PrimaryPressed = true
		a.step(press, at(1))
		a.step(held(bx, by), at(2))
		release := frameAt(bx, by)
		release.PrimaryReleased = true
		a.step(release, at(3))

		if want := (selection{4, 9, 12}); !slices.Equal(v.sel, want) {
			t.Fatalf("the box left selection %+v, want %+v", v.sel, want)
		}
		if len(s.orders) != 0 {
			t.Fatalf("the box issued %+v, want nothing", s.orders)
		}

		ox, oy := cellPoint(v, orderCol, orderRow)
		a.step(orderAt(ox, oy), at(4))

		want := []issued{
			{entity: 4, x: orderCol, y: orderRow},
			{entity: 9, x: orderCol, y: orderRow},
			{entity: 12, x: orderCol, y: orderRow},
		}
		if len(s.orders) != len(want) {
			t.Fatalf("the ordering tap issued %+v, want %+v", s.orders, want)
		}
		for i := range want {
			if s.orders[i] != want[i] {
				t.Fatalf("order %d is %+v, want %+v — k calls, ascending, one cell", i, s.orders[i], want[i])
			}
		}
	})

	t.Run("one advance per map-screen tick at zero, one and many orders", func(t *testing.T) {
		a, v, s := appOnMapSeam(t)
		v.SetEntities([]MapEntity{
			{ID: 4, Cell: image.Pt(3, 3)},
			{ID: 9, Cell: image.Pt(5, 5)},
			{ID: 12, Cell: image.Pt(4, 4)},
		})

		held := func(x, y int) appInput {
			in := frameAt(x, y)
			in.Viewer.PrimaryDown = true
			return in
		}
		// Corner to corner: a plain marquee takes a unit only where it covers
		// strictly more than half of its rectangle (`AI-SELECT-122`), and a
		// centre-to-centre drag covers a quarter of each end unit.
		ax, ay, bx, by := cellSpan(v, 3, 3, 5, 5)
		ux, uy := cellPoint(v, 4, 4)
		ox, oy := cellPoint(v, orderCol, orderRow)

		press := held(ax, ay)
		press.PrimaryPressed = true
		release := frameAt(bx, by)
		release.PrimaryReleased = true

		// Each row is one frame and how many orders it is expected to produce.
		frames := []struct {
			in    appInput
			want  int
			label string
		}{
			{frameAt(ux, uy), 0, "a neutral frame"},
			{tapAt(ux, uy), 0, "a tap, which selects one"},
			{orderAt(ox, oy), 1, "an ordering tap with one selected"},
			{press, 0, "the box's press"},
			{held(bx, by), 0, "the box's move"},
			{release, 0, "the box's release, which selects three"},
			{orderAt(ox, oy), 3, "an ordering tap with three selected"},
		}
		issuedSoFar := 0
		for i, f := range frames {
			before := s.ticks
			a.step(f.in, at(i+1))
			if got := s.ticks - before; got != 1 {
				t.Fatalf("%s: the frame advanced the world %d times, want exactly 1", f.label, got)
			}
			if got := len(s.orders) - issuedSoFar; got != f.want {
				t.Fatalf("%s: the frame issued %d orders, want %d", f.label, got, f.want)
			}
			issuedSoFar = len(s.orders)
		}
		if issuedSoFar != 4 {
			t.Fatalf("the run issued %d orders in total, want 4 — with fewer, the zero/one/many "+
				"readings above are not all present", issuedSoFar)
		}
	})
}
