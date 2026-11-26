package ui

import (
	"image"
	"time"
)

// stepLoadWindow runs one frame of the open Load page.
func (a *App) stepLoadWindow(in appInput, _ time.Time) { a.stepPage(a.flow.loadView(), in) }

// composeSaveScreen is the open Save page's picture.
func (a *App) composeSaveScreen() (*image.RGBA, error) { return a.composeListPage() }

// savePagePaint is the open Save page painted with its held-back captions.
func (a *App) savePagePaint() (*savePaint, error) {
	v, _ := a.flow.saveView()
	return a.paintPage(v)
}

// saveConfirmationLines are the lines the open Save page's question lists.
func (a *App) saveConfirmationLines() []string {
	v, _ := a.flow.saveView()
	return a.confirmationLines(v)
}

func (s *savePageState) setFocus(c saveControl)  { s.view().setFocus(c) }
func (s *savePageState) controls() []saveControl { return s.view().controls() }
func (s *savePageState) editField() *string      { return s.view().editField() }
func (s *savePageState) edit(in appInput)        { s.view().typeText(in) }
func (s *savePageState) controlAt(p image.Point) (saveControl, bool) {
	return s.view().controlAt(p)
}

// editSaveText applies the frame's text keys to the open Save page.
func (f *flow) editSaveText(in appInput) { f.editText(f.saveScreen.view(), in) }

// savePageOf is a hand-built Save page state with the Save role open.
func savePageOf(page SavePage, edit pageEdit) *savePageState {
	return &savePageState{page: page, ui: loadWindow{role: saveListRole()}, pageEdit: edit}
}
