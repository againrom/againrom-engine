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

type loadWindow struct {
	quick         QuickSaveControls
	words         LoadWindowWords
	press         buttonLatch
	bar           scrollBarInput
	confirm       bool
	canDelete     func(string) bool
	prepareDelete func(string) (func() error, error)
	remove        func() error
	lastName      string
	lastClick     time.Time
	// doubled is the row a double click's second press landed on, plus one;
	// the release over it loads, so the release reaches no later screen.
	doubled int
	// completed counts the loads that replaced the session, so presentation can
	// tell a fresh load from an unchanged screen. It is never reset.
	completed uint32
}

func (w *loadWindow) resetClick() { w.lastName, w.lastClick, w.doubled = "", time.Time{}, 0 }

func (w *loadWindow) resetPointer() {
	w.press.clear()
	w.bar.reset()
}

func (a *App) SetLoadWindowWords(w LoadWindowWords) { a.flow.loadUI.words = w }
func (a *App) SetSaveDelete(can func(string) bool, prepare func(string) (func() error, error)) {
	a.flow.loadUI.canDelete, a.flow.loadUI.prepareDelete = can, prepare
}

const loadVisibleRows = 10

var (
	loadPanel        = image.Rect(80, 56, 560, 420)
	loadListArg      = image.Rect(122, 152, 504, 342)
	loadMessageBox   = image.Rect(122, 344, 522, 375)
	loadSelectedText = color.RGBA{218, 183, 71, 255}
	loadDisabledText = color.RGBA{112, 122, 115, 255}
)

// The three Load buttons, as button latch ids.
const (
	loadOKButton = iota
	loadDeleteButton
	loadCancelButton
)

func loadButtonRect(i int) image.Rectangle { return image.Rect(156+i*124, 380, 252+i*124, 404) }

// loadListBox is the Load window's shared list.
func (a *App) loadListBox() listBox { return newListBox(loadListArg, loadVisibleRows, a.flow.menuFont) }

// loadButtonAt is the Load button under p.
func loadButtonAt(p image.Point) (int, bool) {
	for i := 0; i < 3; i++ {
		if p.In(loadButtonRect(i)) {
			return i, true
		}
	}
	return 0, false
}

// loadButtonDisabled reports whether Load button i is disabled: Delete
// while no save can be deleted or a deletion awaits confirmation.
func (f *flow) loadButtonDisabled(i int) bool {
	return i == loadDeleteButton && (!f.canDeleteLoad() || f.loadUI.confirm)
}

func (f *flow) canDeleteLoad() bool {
	if f.loadList == nil || f.loadUI.canDelete == nil || f.loadUI.prepareDelete == nil {
		return false
	}
	i := f.loadList.Selection()
	return i >= 0 && i < len(f.saves) && f.loadUI.canDelete(f.saves[i].Name)
}

func (a *App) acceptLoad() {
	f := a.flow
	f.loadUI.resetPointer()
	f.loadUI.resetClick()
	if !f.loadUI.confirm {
		a.activatePicker(f.loadList, f.chooseLoad)
		a.syncViewerLayout()
		return
	}
	if !f.canDeleteLoad() {
		f.loadUI.confirm = false
		return
	}
	i := f.loadList.Selection()
	if f.loadUI.remove == nil {
		return
	}
	if err := f.loadUI.remove(); err != nil {
		f.msg = err.Error()
		return
	}
	back := f.loadBack
	f.openLoad(back)
	if f.loadList != nil {
		f.loadList.Move(i)
	}
}

func (f *flow) confirmDeleteLoad() {
	if !f.canDeleteLoad() {
		return
	}
	remove, err := f.loadUI.prepareDelete(f.saves[f.loadList.Selection()].Name)
	if err != nil {
		f.msg = err.Error()
		return
	}
	f.loadUI.remove, f.loadUI.confirm, f.msg = remove, true, ""
}

func (a *App) stepLoadWindow(in appInput, now time.Time) {
	f := a.flow
	l := f.loadList
	if in.Unfocused || l == nil {
		f.loadUI.resetPointer()
		f.loadUI.resetClick()
		return
	}
	if in.Enter {
		a.acceptLoad()
		return
	}
	if in.Delete && f.canDeleteLoad() {
		f.loadUI.resetPointer()
		f.loadUI.resetClick()
		f.confirmDeleteLoad()
		return
	}
	if !f.loadUI.confirm {
		if in.Up || in.Down || in.PageUp || in.PageDown || in.Home || in.End || in.WheelY != 0 {
			f.loadUI.resetClick()
		}
		listKey(l, in.Up, in.Down, in.PageUp, in.PageDown)
		switch {
		case in.Home:
			l.Select(0)
		case in.End:
			l.Select(len(l.Rows()) - 1)
		case in.WheelY > 0:
			l.Move(-3)
		case in.WheelY < 0:
			l.Move(3)
		}
	}
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	box := a.loadListBox()
	if !f.loadUI.confirm {
		if req, pos := f.loadUI.bar.step(listBar(box, l), p, ok, in); req != barNone {
			f.loadUI.resetClick()
			listBarRequest(l, req, pos)
			return
		}
		if f.loadUI.bar.active() {
			return
		}
	}
	if in.PrimaryPressed {
		button, onButton := loadButtonAt(p)
		f.loadUI.press.press(button, ok && onButton && !f.loadButtonDisabled(button))
		if !ok || onButton || f.loadUI.confirm {
			f.loadUI.resetClick()
			return
		}
		row, onRow := box.RowAt(p)
		top, count := l.Visible()
		if !onRow || row >= count {
			f.loadUI.resetClick()
			return
		}
		index := top + row
		l.Select(index)
		if index >= len(f.saves) {
			f.loadUI.resetClick()
			return
		}
		name := f.saves[index].Name
		elapsed := now.Sub(f.loadUI.lastClick)
		if name == f.loadUI.lastName && !f.loadUI.lastClick.IsZero() && elapsed >= 0 && elapsed <= 500*time.Millisecond {
			f.loadUI.resetClick()
			f.loadUI.doubled = index + 1
			return
		}
		f.loadUI.lastName, f.loadUI.lastClick = name, now
		return
	}
	if !in.PrimaryReleased {
		return
	}
	if doubled := f.loadUI.doubled; doubled != 0 {
		f.loadUI.doubled = 0
		row, onRow := box.RowAt(p)
		top, _ := l.Visible()
		if ok && onRow && top+row == doubled-1 {
			a.acceptLoad()
		}
		return
	}
	at, inside := loadButtonAt(p)
	button, activated := f.loadUI.press.release(at, ok && inside)
	if !activated {
		return
	}
	switch button {
	case loadOKButton:
		a.acceptLoad()
	case loadDeleteButton:
		if !f.loadUI.confirm && f.canDeleteLoad() {
			f.confirmDeleteLoad()
		}
	case loadCancelButton:
		if f.loadUI.confirm {
			f.loadUI.confirm = false
			f.msg = ""
		} else {
			f.closeLoad()
		}
	}
}
