package ui

import (
	"fmt"
	"image"
	"path/filepath"

	"againrom/pkg/namefold"
	"againrom/pkg/savename"
	"againrom/pkg/ui/widgetkit"
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
	return savename.Hide(name)
}

// A name with two format suffixes must still address the listed disk file.
func saveFieldName(name string) string {
	shown := saveListName(name)
	if ext := filepath.Ext(shown); savename.Is(shown) || namefold.HostEqual(ext, ".ags") {
		return name
	}
	return shown
}

// named reports whether the page has a name field.
func (v pageView) named() bool { return v.role().named }

// nameField is the text the name field edits: the file name on a page that
// names files, else the label.
func (v pageView) nameField() *string {
	if v.page.fileNames {
		return &v.edit.request.Name
	}
	return &v.edit.request.Label
}

// editField is the text the keys edit now, or nil while they edit none.
func (v pageView) editField() *string {
	if !v.named() {
		return nil
	}
	if v.edit.focus == saveDirectoryControl && !v.page.DirectoryBox.Empty() {
		return &v.edit.request.Directory
	}
	if v.edit.focus == saveNameControl || !v.page.keyboardFocus {
		return v.nameField()
	}
	return nil
}

func (v pageView) setFocus(c saveControl) {
	v.edit.focus = c
	v.edit.selectedText = false
	if field := v.editField(); field != nil {
		v.edit.caret = len([]rune(*field))
	}
}

func (v pageView) controls() []saveControl {
	if v.confirming() && !v.page.Confirm.windowed() && v.page.confirmRows > 0 {
		if v.ui.confirm || v.ui.remove != nil {
			return []saveControl{saveCancelControl, saveDeleteControl}
		}
		return []saveControl{saveCancelControl, saveWriteControl}
	}
	if len(v.page.focusOrder) != 0 {
		return v.page.focusOrder
	}
	return []saveControl{saveNameControl, saveListControl, saveWriteControl, saveDeleteControl, saveCancelControl}
}

func (v pageView) moveFocus(delta int) {
	controls := v.controls()
	for i, c := range controls {
		if c == v.edit.focus {
			v.setFocus(controls[(i+delta+len(controls))%len(controls)])
			return
		}
	}
	v.setFocus(controls[0])
}

// pick fills the name field from the selected row.
func (v pageView) pick() {
	value := ""
	if e, _, ok := v.selected(); ok {
		value = e.Label
		if v.page.fileNames {
			value = saveFieldName(e.Name)
		}
	}
	*v.nameField() = value
	v.edit.selectedText = v.page.activateRows
	v.edit.caret = len([]rune(value))
}

// typeText applies the frame's text keys to the edited field.
func (v pageView) typeText(in appInput) {
	field := v.editField()
	if field == nil {
		return
	}
	selection := v.page.selection
	if selection == TextCaretSelection {
		selection = TextStickyAllSelection
	}
	e := v.edit
	state := widgetkit.EditText(TextInputState{Text: *field, Caret: e.caret, SelectedAll: e.selectedText}, textInputKeys(in),
		TextInputRules{Limit: v.page.NameLimit, Selection: selection, SplitLineBreaks: true, LiteralArrows: true})
	*field, e.caret, e.selectedText = state.Text, state.Caret, state.SelectedAll
}

// editText is the frame's text keys on the page: they edit the field and
// clear the refusal.
func (f *flow) editText(v pageView, in appInput) {
	if v.editField() == nil {
		return
	}
	v.typeText(in)
	if (!v.page.keyboardFocus || !in.SelectText) && (in.Backspace || in.Delete || in.Typed != "") {
		f.msg = ""
	}
	if in.Backspace || in.Delete || in.Typed != "" {
		v.edit.listFocus = false
	}
}

func (v pageView) controlAt(p image.Point) (saveControl, bool) {
	layout := v.page.saveLayout()
	for _, c := range v.controls() {
		if layout.holds(int(c), p) {
			return c, true
		}
	}
	return 0, false
}

// rowAt is the list row under p.
func (a *App) rowAt(v pageView, p image.Point) (int, bool) {
	if v.list == nil {
		return 0, false
	}
	i, ok := a.pageListBox(v).RowAt(p)
	top, n := v.list.Visible()
	if ok && i < n {
		return top + i, true
	}
	return 0, false
}

func (v pageView) latchID(c saveControl, row int) int {
	ids := []int{3, 4, 5, 6, 7, 0, 1, 2}
	if len(v.page.latchIDs) != 0 {
		ids = v.page.latchIDs
	}
	if c == saveListControl {
		return ids[c] + (row+1)<<8
	}
	return ids[c]
}

// controlDisabled reports whether control c is disabled: Write while the
// page needs a listed save and none is, or while its name field is empty;
// Delete while no save can be deleted or a question awaits its answer in the
// page's body.
func (f *flow) controlDisabled(v pageView, c saveControl) bool {
	switch c {
	case saveWriteControl:
		if v.confirming() {
			return false
		}
		return v.page.OKNeedsSave && len(v.saves) == 0 || v.named() && !v.page.fileNames && *v.nameField() == ""
	case saveDeleteControl:
		if v.confirmingInPage() && v.page.confirmRows > 0 && v.ui.remove != nil {
			return false
		}
		return !f.canDeletePage(v) || v.confirmingInPage()
	}
	return false
}

func saveDialogError(action string) error {
	return fmt.Errorf("save dialog: %s is unavailable on this screen", action)
}
