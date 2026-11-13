package ui

import (
	"image"
	"image/color"
	"time"
)

type LoadWindowWords struct{ Title, Subtitle, OK, Delete, Cancel, Confirm string }

func defaultLoadWindowWords() LoadWindowWords {
	return LoadWindowWords{"Load Saved Game", "Load the Game", "OK", "Delete", "Cancel", "Delete selected saved game?"}
}

// loadWindow is one open page's pointer and confirmation state.
type loadWindow struct {
	quick         QuickSaveControls
	words         LoadWindowWords
	press         buttonLatch
	bar           scrollBarInput
	confirm       bool
	canDelete     func(string) bool
	prepareDelete func(string) (func() error, error)
	remove        func() error
	// rowPressed is set by a single press on a save row. A double click
	// loads only after one; focus loss, a key or wheel step, a scroll bar
	// gesture, a press off the rows, a load and a closed list clear it
	// (DIV-2870).
	rowPressed bool
	// doubled: a double click's second press; its release loads.
	doubled bool
	// completed counts the loads that replaced the session, so presentation can
	// tell a fresh load from an unchanged screen. It is never reset.
	completed uint32
	// mismatch is the open window listing a refusal over the mods; it owns
	// the page's input until it closes.
	mismatch *dialogRun
	// mismatchTown: the window lists what a completed town load lost, and
	// closing it enters that town (closeModMismatch).
	mismatchTown bool
	// question is the delete question of a page that asks in a window; it
	// owns the page's input while the deletion awaits its answer.
	question dialogRun
}

func (w *loadWindow) resetPointer() {
	w.press.Clear()
	w.bar.reset()
	w.rowPressed = false
	w.doubled = false
}

func (a *App) SetLoadWindowWords(w LoadWindowWords) { a.flow.loadUI.words = w }
func (a *App) SetSaveDelete(can func(string) bool, prepare func(string) (func() error, error)) {
	a.flow.loadUI.canDelete, a.flow.loadUI.prepareDelete = can, prepare
}

const loadVisibleRows = 10

var (
	loadPanel        = image.Rect(76, 60, 556, 412)
	loadListArg      = image.Rect(116, 152, 497, 342)
	loadMessageBox   = image.Rect(116, 346, 516, 366)
	loadSelectedText = color.RGBA{218, 183, 71, 255}
	loadDisabledText = color.RGBA{112, 122, 115, 255}
)

// The three page buttons, as button latch ids.
const (
	loadOKButton = iota
	loadDeleteButton
	loadCancelButton
)

func loadButtonRect(i int) image.Rectangle { return firstLoadPage(LoadWindowWords{}).layout().rect(i) }

// pageListBox is a page's shared list.
func (a *App) pageListBox(v pageView) listBox {
	box := newListBox(v.page.List, v.page.Rows, a.flow.menuFont)
	box.Ruled, box.Framed, box.BarGap = v.page.Ruled, v.page.Framed, v.page.BarGap
	box.Dark = v.page.Ruled || v.page.Framed
	return box
}

// loadListBox is the Load window's shared list.
func (a *App) loadListBox() listBox { return a.pageListBox(a.flow.loadView()) }

// buttonAt is the page button under p.
func (v pageView) buttonAt(p image.Point) (int, bool) {
	layout := v.page.layout()
	for i := range v.page.Buttons {
		if layout.holds(i, p) {
			return i, true
		}
	}
	return 0, false
}

// pageButtonDisabled reports whether page button i is disabled: Ok on a page
// that needs a listed save while none is, or on a save page with an empty
// name; Delete while no save can be deleted or a deletion awaits
// confirmation in the page's body.
func (f *flow) pageButtonDisabled(v pageView, i int) bool {
	if v.save != nil {
		controls := [...]saveControl{saveWriteControl, saveDeleteControl, saveCancelControl}
		return i >= 0 && i < len(controls) && f.saveControlDisabled(v.save, controls[i])
	}
	switch i {
	case loadOKButton:
		if v.confirmingInPage() {
			return false
		}
		return v.page.OKNeedsSave && len(v.saves) == 0
	case loadDeleteButton:
		return !f.canDeletePage(v) || v.confirmingInPage()
	}
	return false
}

// canDeletePage reports whether the selected row is a save the page may
// delete.
func (f *flow) canDeletePage(v pageView) bool {
	entry, _, ok := v.selected()
	return ok && f.pageCanDelete(v, entry)
}

func (f *flow) pageCanDelete(v pageView, e SaveEntry) bool {
	if v.save != nil {
		s := f.saveDialogSeams
		return s.CanDelete != nil && s.PrepareDelete != nil && s.CanDelete(v.save.directory.Path, e.Name)
	}
	return v.ui.canDelete != nil && v.ui.prepareDelete != nil && v.ui.canDelete(e.Name)
}

func (f *flow) pagePrepareDelete(v pageView, e SaveEntry) (func() error, error) {
	if v.save != nil {
		return f.saveDialogSeams.PrepareDelete(v.save.directory.Path, e.Name)
	}
	return v.ui.prepareDelete(e.Name)
}

// acceptPage is Ok: load or save the selection, or delete it once confirmed.
func (a *App) acceptPage(v pageView) {
	f := a.flow
	if v.save != nil {
		v.ui.resetPointer()
		f.activateSaveControl(saveWriteControl)
		return
	}
	v.ui.resetPointer()
	if !v.ui.confirm {
		a.activatePicker(v.list, f.chooseLoad)
		a.syncViewerLayout()
		return
	}
	if !f.canDeletePage(v) {
		v.ui.confirm = false
		return
	}
	i := v.list.Selection()
	if v.ui.remove == nil {
		return
	}
	if err := v.ui.remove(); err != nil {
		f.msg = err.Error()
		return
	}
	back := f.loadBack
	f.openLoad(back)
	if f.loadList != nil {
		f.loadList.Move(i)
	}
}

func (f *flow) confirmDeletePage(v pageView) {
	if v.save != nil {
		f.confirmSaveDelete()
		return
	}
	entry, _, ok := v.selected()
	if !ok || !f.pageCanDelete(v, entry) {
		return
	}
	remove, err := f.pagePrepareDelete(v, entry)
	if err != nil {
		f.msg = err.Error()
		return
	}
	v.ui.remove, v.ui.confirm, f.msg = remove, true, ""
	if c := v.page.Confirm; c.windowed() {
		v.ui.question = c.question(f.menuDisplayText(entry.Label))
	}
}

// stepPageQuestion drives the open delete question: its first answer deletes
// as Ok does, any other leaves the question with the row still chosen.
func (a *App) stepPageQuestion(v pageView, in appInput) {
	font := a.flow.menuFont
	if font == nil {
		a.flow.cancelPage(v)
		return
	}
	w := &v.ui.question
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	answer, done := w.step(in, p, ok, w.layout(font, v.page.Panel), 0)
	switch {
	case !done:
	case answer == 0:
		a.acceptPage(v)
	default:
		a.flow.cancelPage(v)
	}
}

// cancelPage is Cancel: it leaves a confirmation, else closes the page.
func (f *flow) cancelPage(v pageView) {
	if v.save != nil {
		f.cancelSavePage()
		return
	}
	if v.ui.confirm {
		v.ui.confirm = false
		f.msg = ""
		return
	}
	f.closeLoad()
}

func (a *App) stepLoadWindow(in appInput, now time.Time) { a.stepPage(a.flow.loadView(), in) }

func (a *App) stepPage(v pageView, in appInput) {
	if v.save != nil {
		a.stepSaveScreen(in)
		return
	}
	f := a.flow
	l := v.list
	if v.ui.mismatch != nil {
		a.stepModMismatch(v, in)
		return
	}
	if v.asking() {
		a.stepPageQuestion(v, in)
		return
	}
	if in.Unfocused || l == nil {
		v.ui.resetPointer()
		return
	}
	if in.Enter {
		a.acceptPage(v)
		return
	}
	if in.Delete && f.canDeletePage(v) {
		v.ui.resetPointer()
		f.confirmDeletePage(v)
		return
	}
	if !v.ui.confirm {
		if pageListKeyed(in) {
			v.ui.rowPressed = false
		}
		stepPageListKeys(l, in, pickerList, true)
	}
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	box := a.pageListBox(v)
	if !v.ui.confirm && a.stepPageBar(v, l, p, ok, in, nil) {
		return
	}
	if in.PrimaryPressed {
		button, onButton := v.buttonAt(p)
		v.ui.press.Press(button, ok && onButton && !f.pageButtonDisabled(v, button))
		if !ok || onButton || v.ui.confirm {
			v.ui.rowPressed = false
			return
		}
		row, onRow := box.RowAt(p)
		top, count := l.Visible()
		v.ui.pressRow(l, onRow, onRow && row < count, top+row, in.PrimaryDouble, true)
		return
	}
	if !in.PrimaryReleased {
		return
	}
	if v.ui.doubled {
		a.acceptPage(v)
		return
	}
	at, inside := v.buttonAt(p)
	button, activated := v.ui.press.Release(at, ok && inside)
	if !activated {
		return
	}
	switch button {
	case loadOKButton:
		a.acceptPage(v)
	case loadDeleteButton:
		if !v.ui.confirm && f.canDeletePage(v) {
			f.confirmDeletePage(v)
		}
	case loadCancelButton:
		f.cancelPage(v)
	}
}

// pageListKeyed reports a key or wheel step on the list.
func pageListKeyed(in appInput) bool {
	return in.Up || in.Down || in.PageUp || in.PageDown || in.Home || in.End || wheeled(in)
}

// stepPageListKeys moves the list; home lets Home and End pick its ends.
func stepPageListKeys(l *Picker, in appInput, rule scrollSpec, home bool) {
	listKey(l, in.Up, in.Down, in.PageUp, in.PageDown)
	switch {
	case home && in.Home:
		l.Select(0)
	case home && in.End:
		l.Select(len(l.Rows()) - 1)
	case rule.wheelStep(in) != 0:
		l.Move(rule.wheelStep(in))
	}
}

// stepPageBar runs the scroll bar and reports whether it took the input.
func (a *App) stepPageBar(v pageView, l *Picker, p image.Point, ok bool, in appInput, onRequest func()) bool {
	if req, pos := v.ui.bar.step(listBar(a.pageListBox(v), l), p, ok, in); req != barNone {
		if onRequest != nil {
			onRequest()
		}
		v.ui.rowPressed = false
		listBarRequest(l, req, pos)
		return true
	}
	if v.ui.bar.active() {
		v.ui.rowPressed = false
		return true
	}
	return false
}

// pressRow handles a press on the rows; true is a double click's second press (DIV-2870).
func (w *loadWindow) pressRow(l *Picker, onRow, rowOK bool, index int, double, doubleAllowed bool) bool {
	if doubleAllowed && onRow && double && w.rowPressed {
		w.rowPressed, w.doubled = false, true
		return true
	}
	w.rowPressed = rowOK
	if rowOK {
		l.Select(index)
	}
	return false
}
