package ui

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"unicode"
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

type saveDialog struct {
	request      SaveRequest
	directory    SaveDirectory
	list         *Picker
	focus        saveControl
	caret        int
	selectedText bool
	prepared     *PreparedSave
	remove       func() error
	removePath   string
	confirmTop   int
	press        buttonLatch
	bar          scrollBarInput
}

// SetSaveDialogSeams opts this application into the interactive save route.
// SetSaveSeams remains available to callers using the older automatic seam.
func (a *App) SetSaveDialogSeams(seams SaveDialogSeams) {
	if a != nil && a.flow != nil {
		a.flow.saveDialogSeams = seams
	}
}

func (f *flow) atGameSavePoint() bool {
	if f.screen != ScreenGameMenu {
		return false
	}
	switch f.menuBack {
	case ScreenMap:
		return true
	case ScreenTown:
		return townCanSave(f.town)
	default:
		return false
	}
}

func (f *flow) openSaveDialog() {
	if !f.atGameSavePoint() {
		return
	}
	d := &saveDialog{request: SaveRequest{
		OnMap: f.menuBack == ScreenMap, Directory: f.saveDialogSeams.Directory,
		Name: "save", Format: SaveSAV,
	}}
	f.saveDialog = d
	f.msg = ""
	f.setMenuUp(true)
	f.setScreen(ScreenSave)
	f.refreshSaveDirectory()
	d.setFocus(saveNameControl)
	d.selectedText = true
}

func (f *flow) refreshSaveDirectory() {
	d := f.saveDialog
	if d == nil {
		return
	}
	d.prepared = nil
	d.remove = nil
	d.removePath = ""
	d.directory = SaveDirectory{Path: d.request.Directory}
	d.list = NewPicker(nil)
	d.list.SetWindow(saveVisibleRows)
	if f.saveDialogSeams.List == nil {
		f.msg = f.saveWords().Unavailable
		return
	}
	dir, err := f.saveDialogSeams.List(d.request.Directory)
	if err != nil {
		f.msg = err.Error()
		return
	}
	d.directory = dir
	if dir.Path != "" {
		d.request.Directory = dir.Path
	}
	rows := make([]PickerRow, 0, len(dir.Directories)+len(dir.Entries))
	for _, name := range dir.Directories {
		rows = append(rows, PickerRow{Text: "[" + name + "]", Choosable: true})
	}
	for _, entry := range dir.Entries {
		rows = append(rows, PickerRow{Text: saveListName(entry.Name), Choosable: true})
	}
	d.list = NewPicker(rows)
	d.list.SetWindow(saveVisibleRows)
	f.msg = ""
}

// saveListName is a save row's text: the disk name without its one trailing
// format suffix. SAV is the only format, so the suffix tells the player nothing.
func saveListName(name string) string {
	if strings.EqualFold(filepath.Ext(name), ".sav") {
		return name[:len(name)-len(".sav")]
	}
	return name
}

// saveFieldName is what choosing a row puts in the name field. The writer
// removes one trailing format suffix from a typed name and appends ".sav", so a
// row whose remaining name still ends in a format suffix keeps its full disk
// name; stripping it would aim the next SAVE at a different file.
func saveFieldName(name string) string {
	shown := saveListName(name)
	if ext := filepath.Ext(shown); strings.EqualFold(ext, ".sav") || strings.EqualFold(ext, ".ags") {
		return name
	}
	return shown
}

func (d *saveDialog) setFocus(c saveControl) {
	d.focus = c
	d.selectedText = false
	if field := d.editField(); field != nil {
		d.caret = len([]rune(*field))
	}
}

func (d *saveDialog) editField() *string {
	switch d.focus {
	case saveDirectoryControl:
		return &d.request.Directory
	case saveNameControl:
		return &d.request.Name
	}
	return nil
}

func (d *saveDialog) controls() []saveControl {
	if d.remove != nil {
		return []saveControl{saveCancelControl, saveDeleteControl}
	}
	if d.prepared != nil {
		return []saveControl{saveCancelControl, saveWriteControl}
	}
	return []saveControl{saveDirectoryControl, saveOpenControl, saveUpControl,
		saveListControl, saveNameControl, saveWriteControl, saveDeleteControl, saveCancelControl}
}

func (d *saveDialog) moveFocus(delta int) {
	controls := d.controls()
	for i, c := range controls {
		if c == d.focus {
			d.setFocus(controls[(i+delta+len(controls))%len(controls)])
			return
		}
	}
	d.setFocus(controls[0])
}

func (f *flow) cancelSaveDialog() {
	if d := f.saveDialog; d != nil && d.remove != nil {
		d.remove, d.removePath = nil, ""
		d.setFocus(saveListControl)
		f.msg = ""
		return
	}
	if d := f.saveDialog; d != nil && d.prepared != nil {
		d.prepared = nil
		d.setFocus(saveNameControl)
		f.msg = ""
		return
	}
	f.closeSaveDialog("")
}

func (f *flow) closeSaveDialog(message string) {
	f.saveDialog = nil
	f.setScreen(ScreenGameMenu)
	f.rebuildGameMenu(gameMenuRoot, 0)
	f.setMenuUp(true)
	f.msg = message
}

func (f *flow) submitSaveDialog() {
	d := f.saveDialog
	if d == nil {
		return
	}
	f.msg = ""
	if d.prepared != nil {
		f.commitSaveDialog(true)
		return
	}
	if f.saveDialogSeams.Prepare == nil {
		f.msg = f.saveWords().Unavailable
		return
	}
	prepared, err := f.saveDialogSeams.Prepare(d.request)
	if err != nil {
		f.msg = err.Error()
		return
	}
	if prepared.Commit == nil || len(prepared.Paths) == 0 {
		f.msg = f.saveWords().Unavailable
		return
	}
	d.prepared = &prepared
	if len(prepared.Existing) != 0 {
		d.confirmTop = 0
		d.setFocus(saveCancelControl)
		return
	}
	f.commitSaveDialog(false)
}

func (f *flow) commitSaveDialog(overwrite bool) {
	d := f.saveDialog
	if d == nil || d.prepared == nil || d.prepared.Commit == nil {
		return
	}
	notice := d.prepared.Notice
	_, err := d.prepared.Commit(overwrite)
	// A refused publication needs a fresh target check before another attempt.
	d.prepared = nil
	if err != nil {
		f.msg = err.Error()
		d.setFocus(saveNameControl)
		return
	}
	f.saveDialogSeams.Directory = d.request.Directory
	message := f.words.SaveAcknowledgement
	if message == "" {
		message = AuthoredWords().SaveAcknowledgement
	}
	if notice != "" {
		message = message + " " + notice
	}
	f.closeSaveDialog(message)
}

func (f *flow) activateSaveControl(c saveControl) {
	d := f.saveDialog
	if d == nil {
		return
	}
	if d.remove != nil {
		switch c {
		case saveDeleteControl:
			f.commitSaveDelete()
		case saveCancelControl:
			f.cancelSaveDialog()
		}
		return
	}
	if d.prepared != nil {
		switch c {
		case saveWriteControl:
			f.submitSaveDialog()
		case saveCancelControl:
			f.cancelSaveDialog()
		}
		return
	}
	switch c {
	case saveDirectoryControl, saveOpenControl:
		f.refreshSaveDirectory()
		d.setFocus(saveListControl)
	case saveUpControl:
		d.request.Directory = filepath.Dir(d.request.Directory)
		f.refreshSaveDirectory()
		d.setFocus(saveListControl)
	case saveListControl:
		if d.list == nil {
			return
		}
		i, ok := d.list.Choose()
		if !ok {
			return
		}
		if i < len(d.directory.Directories) {
			d.request.Directory = filepath.Join(d.directory.Path, d.directory.Directories[i])
			f.refreshSaveDirectory()
			return
		}
		i -= len(d.directory.Directories)
		if i < len(d.directory.Entries) {
			d.request.Name = saveFieldName(d.directory.Entries[i].Name)
			d.setFocus(saveNameControl)
			d.selectedText = true
			f.msg = ""
		}
	case saveDeleteControl:
		f.confirmSaveDelete()
	case saveNameControl, saveWriteControl:
		f.submitSaveDialog()
	case saveCancelControl:
		f.cancelSaveDialog()
	}
}

func (d *saveDialog) selectedSave() (SaveEntry, bool) {
	if d.list == nil {
		return SaveEntry{}, false
	}
	i := d.list.Selection() - len(d.directory.Directories)
	if i < 0 || i >= len(d.directory.Entries) {
		return SaveEntry{}, false
	}
	return d.directory.Entries[i], true
}

func (f *flow) canDeleteSave() bool {
	d, seams := f.saveDialog, f.saveDialogSeams
	if d == nil || seams.CanDelete == nil || seams.PrepareDelete == nil {
		return false
	}
	entry, ok := d.selectedSave()
	return ok && seams.CanDelete(d.directory.Path, entry.Name)
}

func (f *flow) confirmSaveDelete() {
	if !f.canDeleteSave() {
		return
	}
	d := f.saveDialog
	entry, _ := d.selectedSave()
	remove, err := f.saveDialogSeams.PrepareDelete(d.directory.Path, entry.Name)
	if err != nil {
		f.msg = err.Error()
		return
	}
	if remove == nil {
		return
	}
	d.remove = remove
	d.removePath = filepath.Join(d.directory.Path, entry.Name)
	d.confirmTop = 0
	d.setFocus(saveCancelControl)
	f.msg = ""
}

func (f *flow) commitSaveDelete() {
	d := f.saveDialog
	if d == nil || d.remove == nil {
		return
	}
	err := d.remove()
	d.remove, d.removePath = nil, ""
	d.setFocus(saveListControl)
	if err != nil {
		f.msg = err.Error()
		return
	}
	i := d.list.Selection()
	d.request.Directory = d.directory.Path
	f.refreshSaveDirectory()
	d.list.Move(i)
}

func (f *flow) editSaveText(in appInput) {
	d := f.saveDialog
	field := d.editField()
	if field == nil {
		return
	}
	runes := []rune(*field)
	if d.caret > len(runes) {
		d.caret = len(runes)
	}
	if in.SelectText {
		d.selectedText = true
		return
	}
	if in.Home || in.End || in.Left || in.Right {
		switch {
		case in.Home:
			d.caret = 0
		case in.End:
			d.caret = len(runes)
		case in.Left && d.caret > 0:
			d.caret--
		case in.Right && d.caret < len(runes):
			d.caret++
		}
		d.selectedText = false
	}
	if !in.Backspace && !in.Delete && in.Typed == "" {
		return
	}
	if d.selectedText {
		runes = nil
		d.caret = 0
		d.selectedText = false
	} else if in.Backspace && d.caret > 0 {
		runes = append(runes[:d.caret-1], runes[d.caret:]...)
		d.caret--
	} else if in.Delete && d.caret < len(runes) {
		runes = append(runes[:d.caret], runes[d.caret+1:]...)
	}
	for _, r := range in.Typed {
		if unicode.IsControl(r) || len(runes) >= 4096 {
			continue
		}
		runes = append(runes, 0)
		copy(runes[d.caret+1:], runes[d.caret:])
		runes[d.caret] = r
		d.caret++
	}
	*field = string(runes)
	f.msg = ""
}

func (a *App) stepSaveDialog(in appInput) {
	d := a.flow.saveDialog
	if d == nil || in.Unfocused {
		if d != nil {
			d.press.Clear()
		}
		return
	}
	if in.PaneMode {
		delta := 1
		if in.ShiftHeld {
			delta = -1
		}
		d.moveFocus(delta)
		return
	}
	if d.prepared != nil || d.remove != nil {
		if in.Up || in.WheelY > 0 {
			d.confirmTop--
		}
		if in.Down || in.WheelY < 0 {
			d.confirmTop++
		}
		a.clampSaveConfirmationScroll()
	} else {
		if in.Delete && d.editField() == nil {
			a.flow.confirmSaveDelete()
			return
		}
		a.flow.editSaveText(in)
		if d.focus == saveListControl && d.list != nil {
			listKey(d.list, in.Up, in.Down, in.PageUp, in.PageDown)
			if in.WheelY > 0 {
				d.list.Move(-1)
			}
			if in.WheelY < 0 {
				d.list.Move(1)
			}
		}
		if in.WheelY != 0 && d.focus != saveListControl {
			if p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY); ok && p.In(saveListRect) {
				d.list.Move(-int(in.WheelY))
			}
		}
	}
	if in.Enter {
		a.flow.activateSaveControl(d.focus)
		return
	}
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	if d.prepared == nil && d.remove == nil && d.list != nil {
		if req, pos := d.bar.step(listBar(a.saveListBox(), d.list), p, ok, in); req != barNone {
			d.setFocus(saveListControl)
			listBarRequest(d.list, req, pos)
			return
		}
		if d.bar.active() {
			return
		}
	}
	if !in.PrimaryPressed && !in.PrimaryReleased {
		return
	}
	if !ok {
		d.press.Clear()
		return
	}
	c, hit := d.controlAt(p)
	row, rowOK := 0, hit && c != saveListControl
	if hit && c == saveListControl {
		row, rowOK = a.saveRowAt(p)
	}
	if in.PrimaryPressed {
		if hit {
			d.setFocus(c)
			if c == saveListControl && rowOK {
				d.list.Select(row)
			}
		}
		d.press.Press(saveLatchID(c, row), rowOK)
	}
	if in.PrimaryReleased {
		_, activate := d.press.Release(saveLatchID(c, row), rowOK)
		if activate && c != saveDirectoryControl && c != saveNameControl {
			a.playUISound(UISoundCommonControl)
			a.flow.activateSaveControl(c)
		}
	}
}

// saveLatchID is the latch's button: a control, or one row of the list.
func saveLatchID(c saveControl, row int) int {
	if c == saveListControl {
		return int(c) + (row+1)<<8
	}
	return int(c)
}

func (d *saveDialog) controlAt(p image.Point) (saveControl, bool) {
	for _, c := range d.controls() {
		if p.In(saveControlRect(c)) {
			return c, true
		}
	}
	return 0, false
}

func (a *App) saveRowAt(p image.Point) (int, bool) {
	d := a.flow.saveDialog
	if d == nil || d.list == nil {
		return 0, false
	}
	i, ok := a.saveListBox().RowAt(p)
	top, n := d.list.Visible()
	if ok && i < n {
		return top + i, true
	}
	return 0, false
}

func saveDialogError(action string) error {
	return fmt.Errorf("save dialog: %s is unavailable on this screen", action)
}
