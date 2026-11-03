package ui

import (
	"fmt"
	"image"
	"path/filepath"

	"againrom/pkg/namefold"
)

type saveControl int

const (
	saveDirectoryControl saveControl = iota
	saveOpenControl
	saveUpControl
	saveListControl
	saveNameControl
	saveWriteControl
	saveDeleteControl
	saveCancelControl
)

func (a *App) SetSaveDialogSeams(seams SaveDialogSeams) {
	if a != nil && a.flow != nil {
		a.flow.saveDialogSeams = seams
	}
}

func saveListName(name string) string {
	if namefold.HostEqual(filepath.Ext(name), ".sav") {
		return name[:len(name)-len(".sav")]
	}
	return name
}

// A name with two format suffixes must still address the listed disk file.
func saveFieldName(name string) string {
	shown := saveListName(name)
	if ext := filepath.Ext(shown); namefold.HostEqual(ext, ".sav") || namefold.HostEqual(ext, ".ags") {
		return name
	}
	return shown
}

func (s *savePageState) setFocus(c saveControl) {
	s.focus = c
	s.selectedText = false
	if field := s.editField(); field != nil {
		s.caret = len([]rune(*field))
	}
}

func (s *savePageState) editField() *string {
	if s.focus == saveDirectoryControl && !s.page.DirectoryBox.Empty() {
		return &s.request.Directory
	}
	if s.focus == saveNameControl || !s.page.keyboardFocus {
		return s.nameField()
	}
	return nil
}

func (s *savePageState) controls() []saveControl {
	if s.confirming() && !s.page.Confirm.windowed() && s.page.confirmRows > 0 {
		if s.ui.confirm || s.ui.remove != nil {
			return []saveControl{saveCancelControl, saveDeleteControl}
		}
		return []saveControl{saveCancelControl, saveWriteControl}
	}
	if len(s.page.focusOrder) != 0 {
		return s.page.focusOrder
	}
	return []saveControl{saveNameControl, saveListControl, saveWriteControl, saveDeleteControl, saveCancelControl}
}

func (s *savePageState) moveFocus(delta int) {
	controls := s.controls()
	for i, c := range controls {
		if c == s.focus {
			s.setFocus(controls[(i+delta+len(controls))%len(controls)])
			return
		}
	}
	s.setFocus(controls[0])
}

func (f *flow) cancelSavePage() {
	s := f.saveScreen
	if s == nil {
		return
	}
	if s.confirming() {
		deleting := s.ui.confirm || s.ui.remove != nil
		s.prepared, s.ui.remove, s.ui.confirm, s.removePath = nil, nil, false, ""
		if s.page.keyboardFocus {
			focus := saveNameControl
			if deleting {
				focus = saveListControl
			}
			s.setFocus(focus)
		}
		f.msg = ""
		return
	}
	f.closeSavePage("")
}

func (f *flow) activateSaveControl(c saveControl) {
	s := f.saveScreen
	if s == nil {
		return
	}
	if s.confirming() {
		switch c {
		case saveCancelControl:
			f.cancelSavePage()
		case saveWriteControl:
			if s.ui.confirm || s.ui.remove != nil {
				f.commitSaveDelete()
			} else {
				f.submitSavePage()
			}
		case saveDeleteControl:
			if s.ui.confirm || s.ui.remove != nil {
				f.commitSaveDelete()
			}
		}
		return
	}
	switch c {
	case saveDirectoryControl, saveOpenControl:
		f.refreshSavePage()
		s.setFocus(saveListControl)
	case saveUpControl:
		s.request.Directory = filepath.Dir(s.request.Directory)
		f.refreshSavePage()
		s.setFocus(saveListControl)
	case saveListControl:
		if s.list == nil {
			return
		}
		i, ok := s.list.Choose()
		if !ok {
			return
		}
		if s.page.directories && i < len(s.directory.Directories) {
			s.request.Directory = filepath.Join(s.directory.Path, s.directory.Directories[i])
			f.refreshSavePage()
			return
		}
		s.pick()
		if s.page.activateRows {
			s.setFocus(saveNameControl)
			s.selectedText = true
			f.msg = ""
		}
	case saveDeleteControl:
		f.confirmSaveDelete()
	case saveNameControl, saveWriteControl:
		f.submitSavePage()
	case saveCancelControl:
		f.cancelSavePage()
	}
}

func (s *savePageState) selectedSave() (SaveEntry, bool) {
	if s.list == nil {
		return SaveEntry{}, false
	}
	i := s.list.Selection() - s.first()
	if i < 0 || i >= len(s.saves) {
		return SaveEntry{}, false
	}
	return s.saves[i], true
}

func (f *flow) canDeleteSave() bool {
	s := f.saveScreen
	if s == nil {
		return false
	}
	entry, ok := s.selectedSave()
	return ok && f.pageCanDelete(s.view(), entry)
}

func (f *flow) confirmSaveDelete() {
	if !f.canDeleteSave() {
		return
	}
	s := f.saveScreen
	entry, _ := s.selectedSave()
	remove, err := f.pagePrepareDelete(s.view(), entry)
	if err != nil {
		f.msg = err.Error()
		return
	}
	if remove == nil && s.page.clearDeleteAttempt {
		return
	}
	s.ui.remove, s.ui.confirm = remove, true
	s.removePath = filepath.Join(s.directory.Path, entry.Name)
	s.confirmTop = 0
	if s.page.keyboardFocus {
		s.setFocus(saveCancelControl)
	}
	if c := s.page.Confirm; c.windowed() {
		s.ui.question = c.question(f.menuDisplayText(entry.Label))
	}
	f.msg = ""
}

func (f *flow) commitSaveDelete() {
	s := f.saveScreen
	if s == nil {
		return
	}
	if !s.page.clearDeleteAttempt && !f.canDeleteSave() {
		s.ui.confirm = false
		s.ui.remove = nil
		return
	}
	if s.ui.remove == nil {
		return
	}
	err := s.ui.remove()
	if s.page.clearDeleteAttempt {
		s.ui.remove, s.ui.confirm, s.removePath = nil, false, ""
		s.setFocus(saveListControl)
	}
	if err != nil {
		f.msg = err.Error()
		return
	}
	i := s.list.Selection()
	if s.page.retainDeletedRow {
		s.request.Directory = s.directory.Path
	}
	f.refreshSavePage()
	if s.page.retainDeletedRow {
		s.list.Move(i)
	} else {
		s.list.Select(0)
		s.pick()
	}
}

func (f *flow) editSaveText(in appInput) {
	s := f.saveScreen
	if s.editField() == nil {
		return
	}
	s.edit(in)
	if (!s.page.keyboardFocus || !in.SelectText) && (in.Backspace || in.Delete || in.Typed != "") {
		f.msg = ""
	}
	if in.Backspace || in.Delete || in.Typed != "" {
		s.listFocus = false
	}
}

func (s *savePageState) controlAt(p image.Point) (saveControl, bool) {
	layout := s.page.saveLayout()
	for _, c := range s.controls() {
		if layout.holds(int(c), p) {
			return c, true
		}
	}
	return 0, false
}

func (a *App) saveRowAt(p image.Point) (int, bool) {
	s := a.flow.saveScreen
	if s == nil || s.list == nil {
		return 0, false
	}
	i, ok := a.pageListBox(s.view()).RowAt(p)
	top, n := s.list.Visible()
	if ok && i < n {
		return top + i, true
	}
	return 0, false
}

func (s *savePageState) latchID(c saveControl, row int) int {
	ids := []int{3, 4, 5, 6, 7, 0, 1, 2}
	if len(s.page.latchIDs) != 0 {
		ids = s.page.latchIDs
	}
	if c == saveListControl {
		return ids[c] + (row+1)<<8
	}
	return ids[c]
}

func (a *App) stepSaveScreen(in appInput) {
	s := a.flow.saveScreen
	if s == nil {
		return
	}
	v := s.view()
	if v.asking() {
		a.stepPageQuestion(v, in)
		return
	}
	if in.Unfocused || s.list == nil {
		if s.page.preserveListPointer {
			s.ui.press.Clear()
		} else {
			s.ui.resetPointer()
		}
		return
	}
	if in.PaneMode && s.page.keyboardFocus {
		delta := 1
		if in.ShiftHeld {
			delta = -1
		}
		s.moveFocus(delta)
		return
	}
	before := s.list.Selection()
	defer func() {
		if !s.page.activateRows && a.flow.saveScreen == s && s.list.Selection() != before {
			s.pick()
		}
	}()
	if in.Enter && !s.page.keyboardFocus {
		a.acceptPage(v)
		return
	}
	if s.confirming() {
		wheel := confirmList.wheelStep(in)
		if in.Up || wheel < 0 {
			s.confirmTop -= confirmList.key
		}
		if in.Down || wheel > 0 {
			s.confirmTop += confirmList.key
		}
		a.clampSaveConfirmationScroll()
	} else {
		if in.Delete && s.editField() == nil {
			a.flow.confirmSaveDelete()
			return
		}
		a.flow.editSaveText(in)
		if !s.page.keyboardFocus && pageListKeyed(in) {
			s.ui.rowPressed = false
			s.listFocus = true
		}
		if !s.page.keyboardFocus || s.focus == saveListControl {
			rule := s.page.wheel
			if rule.wheel == 0 {
				rule = pickerList
			}
			stepPageListKeys(s.list, in, rule, false)
		}
		if s.page.keyboardFocus && wheeled(in) && s.focus != saveListControl {
			if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok && s.page.saveLayout().holds(int(saveListControl), p) {
				s.list.Move(s.page.hoverWheel.wheelStep(in))
			}
		}
	}
	if in.Enter {
		a.flow.activateSaveControl(s.focus)
		return
	}
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	if !s.confirming() {
		if a.stepPageBar(v, s.list, p, ok, in, func() {
			if s.page.keyboardFocus {
				s.setFocus(saveListControl)
			}
		}) {
			return
		}
	}
	if !in.PrimaryPressed && !in.PrimaryReleased {
		return
	}
	if in.PrimaryReleased && s.ui.doubled {
		a.acceptPage(v)
		return
	}
	if !ok {
		s.ui.press.Clear()
		s.ui.rowPressed = false
		return
	}
	c, hit := s.controlAt(p)
	box := a.pageListBox(v)
	if !s.confirming() && !hit && (p.In(box.Rect) || p.In(box.Bar())) {
		c, hit = saveListControl, true
	}
	row, rowOK := 0, hit && c != saveListControl
	if hit && c == saveListControl {
		row, rowOK = a.saveRowAt(p)
	}
	if in.PrimaryPressed {
		if !s.page.keyboardFocus && !s.confirming() {
			if c == saveNameControl && hit {
				s.listFocus = false
			}
			box := a.pageListBox(v)
			if p.In(box.Rect) || p.In(box.Bar()) {
				s.listFocus = true
			}
		}
		if hit && s.page.keyboardFocus {
			s.setFocus(c)
		}
		_, onRow := box.RowAt(p)
		if !s.confirming() && hit && c == saveListControl {
			if s.ui.pressRow(s.list, onRow, rowOK, row, in.PrimaryDouble, !s.page.activateRows) {
				return
			}
		} else {
			s.ui.rowPressed = false
		}
		enabled := rowOK && (s.page.keyboardFocus || !a.flow.saveControlDisabled(s, c))
		s.ui.press.Press(s.latchID(c, row), enabled && (c != saveListControl || s.page.activateRows))
	}
	if in.PrimaryReleased {
		_, activate := s.ui.press.Release(s.latchID(c, row), rowOK)
		if activate && c != saveDirectoryControl && c != saveNameControl {
			if s.page.controlSound {
				a.playUISound(UISoundCommonControl)
			}
			a.flow.activateSaveControl(c)
		}
	}
}

func (f *flow) saveControlDisabled(s *savePageState, c saveControl) bool {
	switch c {
	case saveWriteControl:
		return !s.confirming() && !s.page.fileNames && *s.nameField() == ""
	case saveDeleteControl:
		if s.view().confirmingInPage() && s.page.confirmRows > 0 && s.ui.remove != nil {
			return false
		}
		return !f.canDeleteSave() || s.view().confirmingInPage()
	}
	return false
}

func saveDialogError(action string) error {
	return fmt.Errorf("save dialog: %s is unavailable on this screen", action)
}
