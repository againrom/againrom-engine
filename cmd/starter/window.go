package main

import (
	"image"

	"againrom/internal/appicon"
	"againrom/pkg/pathdialog"
	"againrom/pkg/ui"
	"againrom/pkg/ui/heldkey"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// window shows an app in an Ebitengine window. The app draws itself into an
// RGBA image; the window uploads that image when it changes.
type window struct {
	a                  *app
	img                *ebiten.Image
	dialogInputRelease bool
}

// desktop is the window system the starter ends in: the call that gives the
// window its icon and the call that runs it. A test replaces both.
var desktop = struct {
	setIcon func([]image.Image)
	run     func(ebiten.Game) error
}{ui.SetWindowIcon, ebiten.RunGame}

// starterWindow is the size the starter's layout is drawn for, fixed.
var starterWindow = ui.WindowRow{W: winW, H: winH}

func runWindow(a *app) error {
	if a.deps.pathDialog == nil {
		a.deps.pathDialog = pathdialog.Show
	}
	ebiten.SetWindowTitle(a.t(cWindowTitle, a.starterVersion))
	ui.ApplyWindowRow(starterWindow)
	ebiten.SetTPS(30)
	ebiten.SetRunnableOnUnfocused(true)
	desktop.setIcon(appicon.Images(appicon.Starter))
	return desktop.run(&window{a: a, img: ebiten.NewImage(winW, winH)})
}

// keyRepeat is true on the frame a key goes down and while it is held.
func keyRepeat(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	return d == 1 || (d > 15 && d%2 == 0)
}

func (w *window) Update() error {
	a := w.a
	if w.blockDialogInput(starterInputHeld()) {
		return nil
	}
	a.poll()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		a.click(image.Pt(x, y))
		if a.folderDialogResults != nil {
			return nil
		}
	}
	if _, dy := ebiten.Wheel(); dy != 0 {
		x, y := ebiten.CursorPosition()
		if dy > 0 {
			a.wheel(image.Pt(x, y), -1)
		} else {
			a.wheel(image.Pt(x, y), 1)
		}
	}
	ctrl := heldkey.Pressed(ebiten.KeyControl)
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

func (w *window) blockDialogInput(held bool) bool {
	if w.a.pollFolderDialog() {
		w.dialogInputRelease = true
		return true
	}
	if w.dialogInputRelease {
		w.dialogInputRelease = held
		return true
	}
	return false
}

func starterInputHeld() bool {
	return len(heldkey.AppendPressed(nil)) != 0 ||
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) ||
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) ||
		ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle)
}

func (w *window) Draw(screen *ebiten.Image) {
	if w.a.takeDirty() {
		w.img.WritePixels(w.a.render().Pix)
	}
	screen.DrawImage(w.img, nil)
}

func (w *window) Layout(int, int) (int, int) { return winW, winH }
