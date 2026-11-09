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

// ApplyWindow is the one place a window size is set. It hands a row to the
// engine, with the form the monitor chose when the row is monitor-aware. A
// desktop on which a call has no effect keeps the window it had.
func ApplyWindow(r WindowRow, f WindowForm) {
	w, h := r.W, r.H
	if r.MonitorAware {
		ebiten.SetWindowDecorated(f.Decorated)
		if f.Place {
			ebiten.SetWindowPosition(f.X, f.Y)
		}
		w, h = f.W, f.H
	}
	ebiten.SetWindowSize(w, h)
	if r.MinW != 0 || r.MinH != 0 {
		ebiten.SetWindowSizeLimits(r.MinW, r.MinH, -1, -1)
	}
	mode := ebiten.WindowResizingModeDisabled
	if r.Resizable {
		mode = ebiten.WindowResizingModeEnabled
	}
	ebiten.SetWindowResizingMode(mode)
	if r.MonitorAware {
		ebiten.SetFullscreen(f.Fullscreen)
	}
}

// SetWindowIcon gives the program window its icon. Windows picks the taskbar,
// task-switcher and title-bar pictures from images by size. It is called before
// the window runs and takes effect with the window's first frame; no images
// leave the platform's default icon.
func SetWindowIcon(images []image.Image) { ebiten.SetWindowIcon(images) }
