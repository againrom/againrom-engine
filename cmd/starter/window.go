package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// window shows an app in an Ebitengine window. The app draws itself into an
// RGBA image; the window uploads that image when it changes.
type window struct {
	a   *app
	img *ebiten.Image
}

func runWindow(a *app) error {
	ebiten.SetWindowSize(winW, winH)
	ebiten.SetWindowTitle("Againrom starter " + a.starterVersion)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	ebiten.SetTPS(30)
	return ebiten.RunGame(&window{a: a, img: ebiten.NewImage(winW, winH)})
}

// keyRepeat is true on the frame a key goes down and while it is held.
func keyRepeat(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	return d == 1 || (d > 15 && d%2 == 0)
}

func (w *window) Update() error {
	a := w.a
	a.poll()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		a.click(image.Pt(x, y))
	}
	if _, dy := ebiten.Wheel(); dy != 0 {
		x, y := ebiten.CursorPosition()
		if dy > 0 {
			a.wheel(image.Pt(x, y), -1)
		} else {
			a.wheel(image.Pt(x, y), 1)
		}
	}
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl)
	switch {
	case ctrl && inpututil.IsKeyJustPressed(ebiten.KeyV):
		a.pasteText()
	case !ctrl:
		a.typed(string(ebiten.AppendInputChars(nil)))
	}
	if keyRepeat(ebiten.KeyBackspace) {
		a.backspace()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		a.enter()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.escape()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		a.tab()
	}
	if a.closing {
		return ebiten.Termination
	}
	return nil
}

func (w *window) Draw(screen *ebiten.Image) {
	if w.a.takeDirty() {
		w.img.WritePixels(w.a.render().Pix)
	}
	screen.DrawImage(w.img, nil)
}

func (w *window) Layout(int, int) (int, int) { return winW, winH }
