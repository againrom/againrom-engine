package ui

import (
	"image"
	"time"
)

// ScreenHandler is what a screen contributes to the application's three
// dispatches. A nil member leaves that dispatch to the fallback switch the
// screen was not moved out of.
//
//   - Step runs one tick of input for the screen and reports whether the program
//     should exit.
//   - Compose builds the screen's CPU frame, or reports that it has none.
//   - Draw paints the screen onto the application canvas.
type ScreenHandler struct {
	Step    func(a *App, in appInput, now time.Time) (exit bool)
	Compose func(a *App) (*image.RGBA, error)
	Draw    func(a *App)
}

// screenHandlers is the registry, keyed by Screen. Step and composeScreen look a
// screen up here first; Draw does too. The mission and generation screens keep
// their arms in Step's own switch.
var screenHandlers = map[Screen]ScreenHandler{}

func init() {
	screenHandlers[ScreenCutsceneLibrary] = ScreenHandler{
		Compose: func(a *App) (*image.RGBA, error) { return a.composeCutsceneLibrary(), nil },
		Draw:    drawMediaScreen,
	}
	screenHandlers[ScreenCredits] = ScreenHandler{
		Compose: func(a *App) (*image.RGBA, error) { return a.composeCredits(), nil },
		Draw:    drawMediaScreen,
	}
	screenHandlers[ScreenEnding] = ScreenHandler{
		Step: func(a *App, in appInput, _ time.Time) bool {
			a.stepEnding(in)
			return false
		},
		Compose: (*App).composeEndingScreen,
		Draw:    (*App).drawEnding,
	}
	screenHandlers[ScreenMenu] = ScreenHandler{
		Step: func(a *App, in appInput, _ time.Time) bool { return a.stepMenu(in) },
		Compose: func(a *App) (*image.RGBA, error) {
			pix := a.assets.Compose(a.sel.State())
			a.drawMenuLabel(pix)
			a.drawModMenuEntries(pix)
			return pix, nil
		},
		Draw: (*App).drawMenu,
	}
	screenHandlers[ScreenPicker] = ScreenHandler{
		Step: func(a *App, in appInput, _ time.Time) bool {
			a.stepPicker(in)
			return false
		},
		Draw: (*App).drawPicker,
	}
	screenHandlers[ScreenChargen] = ScreenHandler{
		Compose: (*App).composeChargenScreen,
		Draw:    (*App).drawChargen,
	}
	screenHandlers[ScreenTown] = ScreenHandler{
		Step: func(a *App, in appInput, now time.Time) bool {
			a.stepTownAt(in, now)
			return false
		},
		Compose: (*App).composeTownScreen,
		Draw:    (*App).drawTown,
	}
	screenHandlers[ScreenGameMenu] = ScreenHandler{
		Step: func(a *App, in appInput, now time.Time) bool {
			a.holdMapUnderMenu(in, now)
			return a.stepGameMenu(in)
		},
		Compose: (*App).composeTownScreen,
		Draw:    (*App).drawTown,
	}
	screenHandlers[ScreenLoad] = ScreenHandler{
		Step: func(a *App, in appInput, now time.Time) bool {
			a.stepLoadWindow(in, now)
			return false
		},
		Compose: (*App).composeLoadScreen,
		Draw:    func(a *App) { a.drawList(a.loadHeader(), a.flow.loadList) },
	}
	screenHandlers[ScreenSave] = ScreenHandler{
		Step: func(a *App, in appInput, now time.Time) bool {
			a.holdMapUnderMenu(in, now)
			a.stepSaveDialog(in)
			return false
		},
		Compose: (*App).composeSaveDialogScreen,
		Draw:    (*App).drawSaveDialog,
	}
	screenHandlers[ScreenDocuments] = ScreenHandler{
		Step: func(a *App, in appInput, now time.Time) bool {
			a.holdMapUnderDocuments(in, now)
			a.stepDocuments(in)
			return false
		},
		Compose: func(a *App) (*image.RGBA, error) { return composeDocumentsPanel(a.flow.docPanel), nil },
		Draw: func(a *App) {
			// The panel covers the whole canvas, so nothing under it is drawn and
			// there is no fill: composeDocumentsPanel returns a full 640x480 image
			// whose every pixel it wrote.
			a.hasMenu = false
			a.writeCanvas(composeDocumentsPanel(a.flow.docPanel))
		},
	}
}

// drawMediaScreen paints the cutscene library or the credits from their CPU
// frame.
func drawMediaScreen(a *App) {
	a.hasMenu = false
	pix, _ := a.composeScreen()
	a.writeCanvas(pix)
}
