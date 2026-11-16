package widgetkit

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// WindowForm is one request to the window system: size, title bar and border,
// top-left corner, and whether native fullscreen holds it.
type WindowForm struct {
	W, H       int
	Decorated  bool
	Fullscreen bool
	// Place is whether X, Y is set; a form without it leaves the position to
	// the window system.
	Place bool
	X, Y  int
}

// WindowRow is one program's window form. A MonitorAware row takes its size,
// decoration, place and fullscreen from the monitor; zero limits set none.
type WindowRow struct {
	W, H         int
	MinW, MinH   int
	Resizable    bool
	MonitorAware bool
}

// ApplyWindowRow applies a row that is not monitor-aware.
func ApplyWindowRow(r WindowRow) { ApplyWindow(r, WindowForm{}) }

// WindowCalls is the window system ApplyWindow drives: the engine's calls, or a
// recorder's in a test.
type WindowCalls struct {
	Desktop    func() Desktop
	Maximized  func() bool
	Restore    func()
	SetSize    func(w, h int)
	Decorate   func(bool)
	Place      func(x, y int)
	Limits     func(minW, minH int)
	Resizing   func(resizable bool)
	Fullscreen func(bool)
}

// ApplyWindow is the one place a window size is set. It hands a row to the
// engine, with the form the monitor chose when the row is monitor-aware. A
// desktop on which a call has no effect keeps the window it had.
//
// A decorated window never has an outer size above the desktop's work area:
// its client size is cut to fit, and it is sized before its title bar and
// border are added, so no moment of the switch leaves it taller than the screen.
func ApplyWindow(r WindowRow, f WindowForm) {
	applyWindow(WindowCalls{
		Desktop:    platformDesktop,
		Maximized:  ebiten.IsWindowMaximized,
		Restore:    ebiten.RestoreWindow,
		SetSize:    func(w, h int) { ebiten.SetWindowSize(w, h) },
		Decorate:   ebiten.SetWindowDecorated,
		Place:      ebiten.SetWindowPosition,
		Limits:     func(minW, minH int) { ebiten.SetWindowSizeLimits(minW, minH, -1, -1) },
		Resizing:   func(resizable bool) { ebiten.SetWindowResizingMode(resizingMode(resizable)) },
		Fullscreen: ebiten.SetFullscreen,
	}, r, f)
}

func resizingMode(resizable bool) ebiten.WindowResizingModeType {
	if resizable {
		return ebiten.WindowResizingModeEnabled
	}
	return ebiten.WindowResizingModeDisabled
}

func applyWindow(c WindowCalls, r WindowRow, f WindowForm) {
	w, h := r.W, r.H
	decorated := true
	if r.MonitorAware {
		w, h = f.W, f.H
		decorated = f.Decorated
	}
	// The whole-screen forms are the monitor's size and stay so; a native
	// fullscreen window is the system's to size.
	fit := decorated && !(r.MonitorAware && f.Fullscreen)
	var d Desktop
	if fit {
		d = c.Desktop()
		w, h = fitClient(w, h, d)
	}
	// The engine ignores a size request while the window is maximized.
	if c.Maximized() {
		c.Restore()
	}
	if r.MonitorAware && !decorated {
		c.Decorate(false)
		if f.Place {
			c.Place(f.X, f.Y)
		}
		c.SetSize(w, h)
	} else {
		// Size first: adding the title bar and border to a window still as large
		// as the screen would put its bottom edge below it.
		c.SetSize(w, h)
		if r.MonitorAware {
			c.Decorate(true)
			if f.Place {
				x, y := f.X, f.Y
				if d.Work.W > 0 && d.Work.H > 0 {
					x, y = centreClient(w, h, d)
				}
				c.Place(x, y)
			}
		}
	}
	if r.MinW != 0 || r.MinH != 0 {
		c.Limits(r.MinW, r.MinH)
	}
	c.Resizing(r.Resizable)
	if r.MonitorAware {
		c.Fullscreen(f.Fullscreen)
	}
}

// SetWindowIcon gives the program window its icon. Windows picks the taskbar,
// task-switcher and title-bar pictures from images by size. It is called before
// the window runs and takes effect with the window's first frame; no images
// leave the platform's default icon.
func SetWindowIcon(images []image.Image) { ebiten.SetWindowIcon(images) }
