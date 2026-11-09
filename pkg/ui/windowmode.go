package ui

import (
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
	"againrom/pkg/ui/widgetkit"
)

// windowedScreenShare is the fraction of each monitor side, in tenths, the
// decorated window may cover; the rest leaves room for a title bar and taskbar.
const windowedScreenShare = 9

// windowedSize is the decorated window's size on a monW x monH monitor: the
// largest integer multiple of the menu frame that fits nine tenths of each
// side, which draws the menu art pixel for pixel (1920x1080 gets 1280x960). A
// monitor too small for one frame gets nine tenths of itself; one that reports
// no size gets the menu window.
func windowedSize(monW, monH int) (w, h int) {
	if monW <= 0 || monH <= 0 {
		return MenuWindowW, MenuWindowH
	}
	fitW, fitH := max(monW*windowedScreenShare/10, 1), max(monH*windowedScreenShare/10, 1)
	scale := min(fitW/frame.W, fitH/frame.H)
	if scale < 1 {
		return fitW, fitH
	}
	return frame.W * scale, frame.H * scale
}

// windowPlan is the form the window takes on goos for a monW x monH monitor.
//
// The whole screen is the monitor's size: native fullscreen on macOS, which
// keeps the Dock and menu bar off the game, and a borderless window at the
// monitor's origin elsewhere. A monitor that reports no size gets the decorated
// menu window either way. The window is windowedSize, decorated, centred.
func windowPlan(monW, monH int, goos string, windowed bool) windowForm {
	if windowed {
		w, h := windowedSize(monW, monH)
		f := windowForm{W: w, H: h, Decorated: true}
		if monW > 0 && monH > 0 {
			f.Place, f.X, f.Y = true, max((monW-w)/2, 0), max((monH-h)/2, 0)
		}
		return f
	}
	w, h, whole := startupWindow(monW, monH)
	native := whole && goos == "darwin"
	return windowForm{W: w, H: h, Decorated: !whole || native, Fullscreen: native, Place: whole && !native}
}

// The game's, the editor's and the viewer's rows; the starter's is its own.
var (
	GameWindow   = WindowRow{Resizable: true, MonitorAware: true}
	EditorWindow = WindowRow{W: 1280, H: 800, MinW: 640, MinH: 480, Resizable: true}
	ViewerWindow = WindowRow{W: DefaultWindowW, H: DefaultWindowH, Resizable: true}
)

// windowSystem is what a window form switch needs from the desktop.
type windowSystem interface {
	Maximized() bool
	Restore()
	Apply(windowForm)
}

// engineWindow is the windowSystem of the running game: Ebitengine's calls.
type engineWindow struct{}

func (engineWindow) Maximized() bool    { return ebiten.IsWindowMaximized() }
func (engineWindow) Restore()           { ebiten.RestoreWindow() }
func (engineWindow) Apply(f windowForm) { widgetkit.ApplyWindow(GameWindow, f) }

// changeWindow gives the window form f. A maximized window is restored first:
// outside macOS the engine ignores a size request while it is maximized. A
// window in macOS native fullscreen does not report itself maximized.
func changeWindow(ws windowSystem, f windowForm) {
	if ws.Maximized() {
		ws.Restore()
	}
	ws.Apply(f)
}

// splitAltEnter divides this tick's Enter press: with Alt held it switches the
// window form and is not Enter; without Alt it is Enter. In the original, Enter
// with Alt held reaches no Enter arm (MENU-082).
func splitAltEnter(alt, enter bool) (plainEnter, toggle bool) {
	return enter && !alt, enter && alt
}

// SetWindowed chooses the form Run opens in: decorated when true, whole screen
// when false.
func (a *App) SetWindowed(windowed bool) { a.windowed = windowed }

// Windowed reports the window form the game has now.
func (a *App) Windowed() bool { return a.windowed }

// toggleWindow switches between the decorated window and the whole screen.
func (a *App) toggleWindow() {
	a.windowed = !a.windowed
	w, h := monitorSize()
	ws := a.window
	if ws == nil {
		ws = engineWindow{}
	}
	changeWindow(ws, windowPlan(w, h, runtime.GOOS, a.windowed))
}
