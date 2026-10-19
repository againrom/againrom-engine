package ui

import (
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
)

// windowForm is one complete request to the window system: the window's size,
// whether it has a title bar and border, where its top-left corner goes, and
// whether the platform's native fullscreen holds it.
type windowForm struct {
	W, H       int
	Decorated  bool
	Fullscreen bool
	// Place is whether X, Y is set; a form without it leaves the position to
	// the window system.
	Place bool
	X, Y  int
}

// windowedScreenShare is the fraction of each monitor side, in tenths, that
// the decorated window may cover. The rest leaves room for its title bar and
// for a taskbar or dock.
const windowedScreenShare = 9

// windowedSize is the decorated window's size on a monW x monH monitor: the
// largest integer multiple of the menu frame that fits nine tenths of each
// side. An integer multiple draws the menu art pixel for pixel, as
// MenuWindowScale explains; a 1920x1080 monitor gets 1280x960. A monitor too
// small for one frame gets nine tenths of itself, and a monitor that reports
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
// keeps the Dock and menu bar off the game and keeps decorations for its
// windowed return, and a borderless window at the monitor's origin elsewhere.
// A monitor that reports no size gets the decorated menu window either way.
//
// The window is windowedSize, decorated, centred on the monitor. Both forms
// are resizable.
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

// applyWindowForm hands a form to the engine. A desktop on which a call has no
// effect, such as a window manager that places windows itself, keeps the
// window it had.
func applyWindowForm(f windowForm) {
	ebiten.SetWindowDecorated(f.Decorated)
	if f.Place {
		ebiten.SetWindowPosition(f.X, f.Y)
	}
	ebiten.SetWindowSize(f.W, f.H)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(f.Fullscreen)
}

// windowSystem is what a switch of the window form needs from the desktop.
type windowSystem interface {
	Maximized() bool
	Restore()
	Apply(windowForm)
}

// engineWindow is the windowSystem of the running game: Ebitengine's calls.
type engineWindow struct{}

func (engineWindow) Maximized() bool    { return ebiten.IsWindowMaximized() }
func (engineWindow) Restore()           { ebiten.RestoreWindow() }
func (engineWindow) Apply(f windowForm) { applyWindowForm(f) }

// changeWindow gives the window form f. A maximized window is restored first:
// outside macOS the engine ignores a size request while the window is
// maximized, so the whole screen would keep the maximized work area. A window
// in macOS native fullscreen does not report itself maximized.
func changeWindow(ws windowSystem, f windowForm) {
	if ws.Maximized() {
		ws.Restore()
	}
	ws.Apply(f)
}

// splitAltEnter divides this tick's Enter press: with either Alt key held it
// switches the window form and is not Enter; without Alt it is Enter. In the
// original, Enter with Alt held arrives as a system key that reaches no Enter
// arm (MENU-082), so no screen loses an Enter it had there.
func splitAltEnter(alt, enter bool) (plainEnter, toggle bool) {
	return enter && !alt, enter && alt
}

// SetWindowed chooses the form Run opens the window in: a decorated window
// when true, the whole screen when false.
func (a *App) SetWindowed(windowed bool) { a.windowed = windowed }

// Windowed reports the window form the game has now.
func (a *App) Windowed() bool { return a.windowed }

// toggleWindow switches between the decorated window and the whole screen of
// the monitor the window is on now.
func (a *App) toggleWindow() {
	a.windowed = !a.windowed
	w, h := monitorSize()
	ws := a.window
	if ws == nil {
		ws = engineWindow{}
	}
	changeWindow(ws, windowPlan(w, h, runtime.GOOS, a.windowed))
}
