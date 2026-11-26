package ui

import (
	"image"
	"image/color"
	"path/filepath"
)

type LoadWindowWords struct{ Title, Subtitle, OK, Delete, Cancel, Confirm string }

func defaultLoadWindowWords() LoadWindowWords {
	return LoadWindowWords{"Load Saved Game", "Load the Game", "OK", "Delete", "Cancel", "Delete selected saved game?"}
}

// loadWindow is one open page's pointer and confirmation state.
type loadWindow struct {
	// session shows the character list again with the running multiplayer
	// session's character row selected and a message on its line; nil while
	// the map screen runs no session's map.
	session       func(message string)
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
	// role is the list page's open role; nil is the Load page (openRole).
	role *listRole
	// edit is the edit state of a page the flow holds; the save page keeps
	// its own beside its list.
	edit pageEdit
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

// pageButtonDisabled reports whether page button i is disabled: the controls
// the buttons stand for, in order Ok, Delete and Cancel.
func (f *flow) pageButtonDisabled(v pageView, i int) bool {
	controls := [...]saveControl{saveWriteControl, saveDeleteControl, saveCancelControl}
	return i >= 0 && i < len(controls) && f.controlDisabled(v, controls[i])
}

// canDeletePage reports whether the selected row is a save the page may
// delete.
func (f *flow) canDeletePage(v pageView) bool {
	entry, _, ok := v.selected()
	return ok && f.pageCanDelete(v, entry)
}

func (f *flow) pageCanDelete(v pageView, e SaveEntry) bool {
	can, prepare := f.deleteSeams(v)
	return can != nil && prepare != nil && can(e.Name)
}

func (f *flow) pagePrepareDelete(v pageView, e SaveEntry) (func() error, error) {
	_, prepare := f.deleteSeams(v)
	return prepare(e.Name)
}

// deleteSeams are the open role's delete seams; a role without them deletes
// no row.
func (f *flow) deleteSeams(v pageView) (func(string) bool, func(string) (func() error, error)) {
	if deletes := v.role().deletes; deletes != nil {
		return deletes(f)
	}
	return nil, nil
}

// acceptPage is Ok: the role's accept on the selection, or the deletion or
// write once confirmed.
func (a *App) acceptPage(v pageView) {
	v.ui.resetPointer()
	a.activateControl(v, saveWriteControl)
}

// submitPage is Ok with nothing awaiting an answer: the role's accept.
func (a *App) submitPage(v pageView) {
	role := v.role()
	if role.chooses {
		a.activatePicker(v.list, a.flow.chooseLoad)
		a.syncViewerLayout()
		return
	}
	if role.accept != nil {
		role.accept(a.flow)
	}
}

// activateControl runs control c of the open page: with a question awaiting
// its answer, Cancel leaves it and Ok or Delete answers it; otherwise each
// control does its own part.
func (a *App) activateControl(v pageView, c saveControl) {
	f := a.flow
	e := v.edit
	if v.confirming() {
		deleting := v.ui.confirm || v.ui.remove != nil
		switch c {
		case saveCancelControl:
			f.cancelPage(v)
		case saveWriteControl:
			if deleting {
				f.commitDelete(v)
			} else {
				a.submitPage(v)
			}
		case saveDeleteControl:
			if deleting {
				f.commitDelete(v)
			}
		}
		return
	}
	switch c {
	case saveDirectoryControl, saveOpenControl, saveUpControl:
		if c == saveUpControl {
			e.request.Directory = filepath.Dir(e.request.Directory)
		}
		f.refreshPage(v)
		if nv, ok := f.openPage(); ok {
			nv.setFocus(saveListControl)
		}
	case saveListControl:
		if v.list == nil {
			return
		}
		i, ok := v.list.Choose()
		if !ok {
			return
		}
		if v.page.directories && i < len(e.directory.Directories) {
			e.request.Directory = filepath.Join(e.directory.Path, e.directory.Directories[i])
			f.refreshPage(v)
			return
		}
		v.pick()
		if v.page.activateRows {
			v.setFocus(saveNameControl)
			e.selectedText = true
			f.msg = ""
		}
	case saveDeleteControl:
		f.confirmDeletePage(v)
	case saveNameControl, saveWriteControl:
		a.submitPage(v)
	case saveCancelControl:
		f.cancelPage(v)
	}
}

// refreshPage lists the open page's rows again.
func (f *flow) refreshPage(v pageView) { f.openListPage(f.loadBack, v.role()) }

// confirmDeletePage is Delete on the selected row: its deletion is prepared
// and asked for.
func (f *flow) confirmDeletePage(v pageView) {
	entry, _, ok := v.selected()
	if !ok || !f.pageCanDelete(v, entry) {
		return
	}
	remove, err := f.pagePrepareDelete(v, entry)
	if err != nil {
		f.msg = err.Error()
		return
	}
	if remove == nil && v.page.clearDeleteAttempt {
		return
	}
	v.ui.remove, v.ui.confirm = remove, true
	v.edit.removePath = filepath.Join(v.edit.directory.Path, entry.Name)
	v.edit.confirmTop = 0
	if v.page.keyboardFocus {
		v.setFocus(saveCancelControl)
	}
	if c := v.page.Confirm; c.windowed() {
		v.ui.question = c.question(f.menuDisplayText(entry.Label))
	}
	f.msg = ""
}

// commitDelete removes the confirmed row and lists the page again with the
// row's place kept, or the first row picked on a page that names its file.
func (f *flow) commitDelete(v pageView) {
	if !v.page.clearDeleteAttempt && !f.canDeletePage(v) {
		v.ui.confirm, v.ui.remove = false, nil
		return
	}
	if v.ui.remove == nil {
		return
	}
	err := v.ui.remove()
	if v.page.clearDeleteAttempt {
		v.ui.remove, v.ui.confirm, v.edit.removePath = nil, false, ""
		v.setFocus(saveListControl)
	}
	if err != nil {
		f.msg = err.Error()
		return
	}
	i := v.list.Selection()
	if v.page.retainDeletedRow {
		v.edit.request.Directory = v.edit.directory.Path
	}
	f.openListPage(f.loadBack, v.role())
	nv, _ := f.openPage()
	if v.page.retainDeletedRow || !v.named() {
		nv.list.Move(i)
	} else {
		nv.list.Select(0)
		nv.pick()
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

// cancelPage is Cancel: it leaves a question, else closes the page.
func (f *flow) cancelPage(v pageView) {
	if v.confirming() {
		deleting := v.ui.confirm || v.ui.remove != nil
		v.edit.prepared, v.ui.remove, v.ui.confirm, v.edit.removePath = nil, nil, false, ""
		if v.page.keyboardFocus {
			focus := saveNameControl
			if deleting {
				focus = saveListControl
			}
			v.setFocus(focus)
		}
		f.msg = ""
		return
	}
	f.closeLoad()
}

// stepPage drives one frame of the open page: the question or window that
// owns the input, the keys, the scroll bar, the pointer on the list and the
// controls, and Ok, Delete and Cancel.
func (a *App) stepPage(v pageView, in appInput) {
	f := a.flow
	l := v.list
	e := v.edit
	if v.ui.mismatch != nil {
		a.stepModMismatch(v, in)
		return
	}
	if v.asking() {
		a.stepPageQuestion(v, in)
		return
	}
	if in.Unfocused || l == nil {
		if v.page.preserveListPointer {
			v.ui.press.Clear()
		} else {
			v.ui.resetPointer()
		}
		return
	}
	if in.PaneMode && v.page.keyboardFocus {
		delta := 1
		if in.ShiftHeld {
			delta = -1
		}
		v.moveFocus(delta)
		return
	}
	before := l.Selection()
	role := v.role()
	defer func() {
		now, ok := f.openPage()
		if ok && role.named && !v.page.activateRows && f.screen == role.shownOn && now.list.Selection() != before {
			now.pick()
		}
	}()
	if in.Enter && !v.page.keyboardFocus {
		a.acceptPage(v)
		return
	}
	if v.confirming() {
		wheel := confirmList.wheelStep(in)
		if in.Up || wheel < 0 {
			e.confirmTop -= confirmList.key
		}
		if in.Down || wheel > 0 {
			e.confirmTop += confirmList.key
		}
		a.clampConfirmationScroll(v)
	} else {
		if in.Delete && v.editField() == nil && f.canDeletePage(v) {
			f.confirmDeletePage(v)
			return
		}
		f.editText(v, in)
		if !v.page.keyboardFocus && pageListKeyed(in) {
			v.ui.rowPressed = false
			e.listFocus = true
		}
		if !v.page.keyboardFocus || e.focus == saveListControl {
			rule := v.page.wheel
			if rule.wheel == 0 {
				rule = pickerList
			}
			stepPageListKeys(l, in, rule, !role.named)
		}
		if v.page.keyboardFocus && wheeled(in) && e.focus != saveListControl {
			if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok && v.page.saveLayout().holds(int(saveListControl), p) {
				l.Move(v.page.hoverWheel.wheelStep(in))
			}
		}
	}
	if in.Enter {
		a.activateControl(v, e.focus)
		return
	}
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	if !v.confirming() {
		if a.stepPageBar(v, l, p, ok, in, func() {
			if v.page.keyboardFocus {
				v.setFocus(saveListControl)
			}
		}) {
			return
		}
	}
	if !in.PrimaryPressed && !in.PrimaryReleased {
		return
	}
	if in.PrimaryReleased && v.ui.doubled {
		a.acceptPage(v)
		return
	}
	if !ok {
		v.ui.press.Clear()
		v.ui.rowPressed = false
		return
	}
	c, hit := v.controlAt(p)
	box := a.pageListBox(v)
	if !v.confirming() && !hit && (p.In(box.Rect) || p.In(box.Bar())) {
		c, hit = saveListControl, true
	}
	row, rowOK := 0, hit && c != saveListControl
	if hit && c == saveListControl {
		row, rowOK = a.rowAt(v, p)
	}
	if in.PrimaryPressed {
		if !v.page.keyboardFocus && !v.confirming() {
			if c == saveNameControl && hit {
				e.listFocus = false
			}
			if p.In(box.Rect) || p.In(box.Bar()) {
				e.listFocus = true
			}
		}
		if hit && v.page.keyboardFocus {
			v.setFocus(c)
		}
		_, onRow := box.RowAt(p)
		if !v.confirming() && hit && c == saveListControl {
			if v.ui.pressRow(l, onRow, rowOK, row, in.PrimaryDouble, !v.page.activateRows) {
				return
			}
		} else {
			v.ui.rowPressed = false
		}
		enabled := rowOK && (v.page.keyboardFocus || !f.controlDisabled(v, c))
		v.ui.press.Press(v.latchID(c, row), enabled && (c != saveListControl || v.page.activateRows))
	}
	if in.PrimaryReleased {
		_, activate := v.ui.press.Release(v.latchID(c, row), rowOK)
		if activate && c != saveDirectoryControl && c != saveNameControl {
			if v.page.controlSound {
				a.playUISound(UISoundCommonControl)
			}
			a.activateControl(v, c)
		}
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
	owner := listOwner(a.pageListBox(v), l)
	owner.Before = func() {
		if onRequest != nil {
			onRequest()
		}
		v.ui.rowPressed = false
	}
	if v.ui.bar.drive(&owner, p, ok, in) != barNone {
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
