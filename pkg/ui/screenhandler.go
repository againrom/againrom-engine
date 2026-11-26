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
//   - Page answers the list page the screen shows; nil for a screen that
//     shows none.
type ScreenHandler struct {
	Step    func(a *App, in appInput, now time.Time) (exit bool)
	Compose func(a *App) (*image.RGBA, error)
	Draw    func(a *App)
	Page    func(f *flow) (pageView, bool)
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
	screenHandlers[ScreenLoad] = listPageHandler(
		func(a *App, in appInput, now time.Time) { a.flow.menuPlacement().underlay.hold(a, in, now) },
		func(f *flow) (pageView, bool) { return f.loadView(), true })
	screenHandlers[ScreenSave] = listPageHandler((*App).holdMapUnderMenu, (*flow).saveView)
	screenHandlers[ScreenDocuments] = ScreenHandler{
		Step: func(a *App, in appInput, now time.Time) bool {
			a.holdMapUnderCover(in, now)
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

// listPageHandler is a list page screen: what the screen holds under the
// page, and the page it shows. Every list page screen steps, composes and
// draws through the page's one builder.
func listPageHandler(hold func(a *App, in appInput, now time.Time), page func(f *flow) (pageView, bool)) ScreenHandler {
	return ScreenHandler{
		Step: func(a *App, in appInput, now time.Time) bool {
			hold(a, in, now)
			if v, ok := page(a.flow); ok {
				a.stepPage(v, in)
			}
			return false
		},
		Compose: (*App).composeListPage,
		Draw:    (*App).drawList,
		Page:    page,
	}
}

// drawMediaScreen paints the cutscene library or the credits from their CPU
// frame.
func drawMediaScreen(a *App) {
	a.hasMenu = false
	pix, _ := a.composeScreen()
	a.writeCanvas(pix)
}
