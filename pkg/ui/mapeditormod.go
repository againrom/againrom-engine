package ui

import (
	"errors"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"strconv"
	"strings"
)

func (e *MapEditor) FocusModField(key string) bool {
	if !e.modWindow {
		return false
	}
	e.loadObject()
	for n, action := range []string{"add", "remove", "save", "publish", "close"} {
		if key == action {
			e.obj.focus = len(e.obj.fields) + n
			return true
		}
	}
	for i, f := range e.obj.fields {
		if f.Key == key && !f.ReadOnly {
			e.obj.detail = f.Parent
			e.objectSpec()
			e.focusObjectField(i)
			return true
		}
	}
	return false
}

func (e *MapEditor) ModWindowOpen() bool { return e.modWindow }

func (e *MapEditor) applyModField() error {
	o := &e.obj
	if o.focus < 0 || o.focus >= len(o.fields) {
		return nil
	}
	f := o.fields[o.focus]
	if f.Action {
		return nil
	}
	if o.buf[o.focus] == fieldInput(f) {
		return nil
	}
	return e.modAction(f.Key, o.buf[o.focus])
}

func (e *MapEditor) modAction(key, value string) error {
	host, ok := e.workspace()
	if !ok {
		return nil
	}
	focus, detail := e.obj.focus, e.obj.detail
	button := -1
	fieldKey := ""
	if focus >= len(e.obj.fields) {
		button = focus - len(e.obj.fields)
	} else if focus >= 0 {
		fieldKey = e.obj.fields[focus].Key
	}
	next, err := host.SetWorkspaceField("package", key, value)
	if err != nil {
		var leave *WorkspaceLeave
		if errors.As(err, &leave) {
			e.askLeave(leaveStep{open: true, source: leave.Folder, workspace: true})
			return nil
		}
		var ask *WorkspaceConfirmation
		if errors.As(err, &ask) {
			e.modConfirm = ask.Action
			e.modConfirmLabel = e.editorHostText(ask.Confirm)
			e.modStatus = ask.Message
			e.obj.focus = len(e.obj.fields)
			return nil
		}
		e.obj.note = err.Error()
		e.modStatus = err.Error()
		return err
	}
	e.modConfirm = ""
	e.modStatus = ""
	_ = e.adoptEdit(next, nil)
	e.message = ""
	e.modWindow = true
	e.objectHidden = false
	e.obj.doc = nil
	e.loadObject()
	e.obj.detail = detail
	e.obj.focus = 0
	if button >= 0 {
		e.obj.focus = len(e.obj.fields) + min(button, e.modButtonCount()-1)
	} else {
		for i, f := range e.obj.fields {
			if f.Key == fieldKey && !f.ReadOnly {
				e.obj.focus = i
				break
			}
		}
	}
	if status, ok := e.doc.Edits.(interface{ SaveNote() string }); ok {
		e.modStatus = status.SaveNote()
	}
	e.objectSpec()
	return nil
}

func (e *MapEditor) modButton(n int) error {
	if e.modConfirm != "" {
		if n == 0 {
			return e.modAction(e.modConfirm, "confirmed")
		} else {
			e.modConfirm = ""
			e.modStatus = ""
			e.obj.focus = 0
		}
		return nil
	}
	if n == 4 {
		e.closeObject()
		return nil
	}
	if err := e.applyModField(); err != nil {
		return err
	}
	key := []string{"add", "remove", "save", "publish"}[n]
	return e.modAction(key, "")
}

func (e *MapEditor) modKey(key string) (bool, error) {
	o := &e.obj
	if e.modConfirm != "" {
		switch key {
		case "tab", "backtab":
			o.focus = len(o.fields) + (o.focus-len(o.fields)+1)%2
		case "enter":
			return true, e.modButton(max(0, min(1, o.focus-len(o.fields))))
		case "escape":
			e.modConfirm, e.modStatus = "", ""
			o.focus = 0
		}
		return true, nil
	}
	switch key {
	case "close":
		e.closeObject()
		return true, e.keyClose(key)
	case "package":
		return true, e.keyWorkspace(key)
	case "saveas":
		return true, e.modButton(2)
	case "testplay":
		if err := e.applyModField(); err != nil {
			return true, err
		}
		e.closeObject()
		return true, e.keyTestPlay(key)
	case "undo", "redo":
		err := e.keyHistory(key)
		e.modWindow = true
		e.objectHidden = false
		e.obj.doc = nil
		e.loadObject()
		e.objectSpec()
		return true, err
	case "tab":
		if err := e.applyModField(); err != nil {
			return true, err
		}
		e.objectSpec()
		if o.focus >= len(o.fields) {
			o.focus++
			if o.focus >= len(o.fields)+e.modButtonCount() {
				o.focus = e.nextObjectField(-1)
			}
		} else {
			next := e.nextObjectField(o.focus)
			if next <= o.focus {
				o.focus = len(o.fields)
			} else {
				e.focusObjectField(next)
			}
		}
		return true, nil
	case "backtab":
		if err := e.applyModField(); err != nil {
			return true, err
		}
		e.objectSpec()
		var visible []int
		for i := range o.fields {
			if e.fieldRow(i) >= 0 {
				visible = append(visible, i)
			}
		}
		for i := 0; i < e.modButtonCount(); i++ {
			visible = append(visible, len(o.fields)+i)
		}
		at := 0
		for i, n := range visible {
			if n == o.focus {
				at = i
				break
			}
		}
		o.focus = visible[(at+len(visible)-1)%len(visible)]
		if o.focus < len(o.fields) {
			e.focusObjectField(o.focus)
		}
		return true, nil
	case "enter":
		if o.focus >= len(o.fields) {
			return true, e.modButton(o.focus - len(o.fields))
		}
		if e.dropdownOpen() && o.choice != nil {
			e.chooseObjectChoice(o.choice.Selection())
			return true, nil
		}
		if o.focus >= 0 && o.focus < len(o.fields) && o.fields[o.focus].Action {
			return true, e.modAction(o.fields[o.focus].Key, "")
		}
		if o.focus >= 0 && o.focus < len(o.fields) && o.fields[o.focus].Key == "folder" {
			return true, e.modAction("folder", o.buf[o.focus])
		}
		if o.focus >= 0 && o.focus < len(o.fields) && len(o.fields[o.focus].Choices) > 0 && o.buf[o.focus] == fieldInput(o.fields[o.focus]) {
			e.objectRowPress(popupRowControl, e.fieldRow(o.focus))
			return true, nil
		}
		return true, e.applyModField()
	case "escape":
		if e.modConfirm != "" {
			e.modConfirm = ""
			e.modStatus = ""
			o.focus = 0
			return true, nil
		}
		if e.dropdownOpen() {
			o.open = -1
			return true, nil
		}
		if o.focus >= 0 && o.focus < len(o.fields) && o.buf[o.focus] != fieldInput(o.fields[o.focus]) {
			o.buf[o.focus] = fieldInput(o.fields[o.focus])
			return true, nil
		}
		e.closeObject()
		return true, nil
	case "cardup", "carddown":
		action := "up"
		if key == "carddown" {
			action = "down"
		}
		if o.focus >= 0 && o.focus < len(o.fields) {
			f := o.fields[o.focus]
			if strings.HasPrefix(f.Key, "party.") {
				parts := strings.Split(f.Key, ".")
				action = parts[0] + "." + parts[1] + "." + action
			}
		}
		return true, e.modAction(action, "")
	}
	if o.focus < 0 || o.focus >= len(o.fields) {
		return true, nil
	}
	return false, nil
}

func (e *MapEditor) keyModBacktab(string) error { return nil }

func (e *MapEditor) modNativeKeys(in editorNativeInput) (bool, error) {
	if !e.modWindow || e.prompt.open() {
		return false, nil
	}
	if in.justPressed(ebiten.KeyTab) {
		key := "tab"
		if in.pressed(ebiten.KeyShift) {
			key = "backtab"
		}
		return true, e.Key(key)
	}
	if in.pressed(ebiten.KeyAlt) {
		if in.justPressed(ebiten.KeyArrowUp) {
			return true, e.Key("cardup")
		}
		if in.justPressed(ebiten.KeyArrowDown) {
			return true, e.Key("carddown")
		}
	}
	if !in.pressed(ebiten.KeyControl) {
		return false, nil
	}
	if in.justPressed(ebiten.KeyS) {
		return true, e.Key("saveas")
	}
	pending := false
	o := &e.obj
	if o.focus >= 0 && o.focus < len(o.fields) {
		pending = o.buf[o.focus] != fieldInput(o.fields[o.focus])
	}
	if !pending {
		if in.justPressed(ebiten.KeyZ) {
			key := "undo"
			if in.pressed(ebiten.KeyShift) {
				key = "redo"
			}
			return true, e.Key(key)
		}
		if in.justPressed(ebiten.KeyY) {
			return true, e.Key("redo")
		}
	}
	return false, nil
}

func (e *MapEditor) modSpec() (popupSpec, []int, bool) {
	e.loadObject()
	o := &e.obj
	o.rowActs = map[int][]string{}
	s := popupSpec{Title: e.word("editor.mod.window"), Width: min(620, e.window.X-16), At: image.Pt(8, e.barHeight()+6), Bounds: e.popupBounds(), Columns: 2, NoteInk: editorPaper, LabelWidth: 80}
	var links []int
	e.objectFieldRows(&s, &links)
	for _, f := range o.fields {
		if f.Key != "source" {
			continue
		}
		start := max(0, int(f.Value)-2)
		end := min(len(f.Choices), start+3)
		start = max(0, end-3)
		for n := start; n < end; n++ {
			row := popupRow{Kind: popupChoice, Label: editorClip(e.bodyFont(), e.editorHostText(f.Choices[n].Label), s.Width/2-62), Buttons: []string{e.word("editor.mod.open")}, Column: 0, Focus: int64(n+1) == f.Value}
			index := len(s.Rows)
			s.Rows = append(s.Rows, row)
			links = append(links, 0)
			if o.rowActs == nil {
				o.rowActs = map[int][]string{}
			}
			o.rowActs[index] = []string{strconv.FormatInt(f.Choices[n].Value, 10)}
		}
	}
	s.Buttons = []string{e.word("editor.mod.addshort"), e.word("editor.mod.remove"), e.word("editor.mod.save"), e.word("editor.mod.publish"), e.word("editor.mod.close")}
	s.ButtonW = min(104, (s.Width-24)/5)
	s.Disabled = nil
	if e.modConfirm != "" {
		s.Buttons = []string{e.word("editor.mod.confirm"), e.word("editor.button.cancel")}
		if e.modConfirmLabel != "" {
			s.Buttons[0] = e.modConfirmLabel
		}
		s.ButtonW = min(210, (s.Width-24)/2)
	}
	if o.focus >= len(o.fields) {
		n := o.focus - len(o.fields)
		s.ButtonFocus = &n
	}
	status := e.modStatus
	if status == "" {
		status = e.word("editor.mod.keys")
	}
	s.Notes = editorWrap(e.bodyFont(), e.editorHostText(status), s.Width-20)
	if host, ok := e.doc.Edits.(interface{ ModSummary() string }); ok {
		s.Notes = append([]string{editorClip(e.bodyFont(), e.editorHostText(host.ModSummary()), s.Width-20)}, s.Notes...)
	}
	if host, ok := e.doc.Edits.(interface{ ModOutput() string }); ok {
		if output := host.ModOutput(); output != "" {
			s.Notes = append(s.Notes, editorWrap(e.bodyFont(), e.editorHostText(output), s.Width-20)...)
		}
	}
	s.MaxRows = max(3, (s.Bounds.Dy()-fontHeight(e.font, 15)-len(s.Notes)*(fontHeight(e.bodyFont(), 15)+popupRowExtra)-60)/(fontHeight(e.bodyFont(), 15)+popupRowExtra))
	e.objectTop = max(0, min(e.objectTop, max(0, popupRowCount(s)-s.MaxRows)))
	s.Top = e.objectTop
	s.ColumnTops = []int{0, e.objectTop}
	return s, links, true
}

func (e *MapEditor) modPress(p image.Point) bool {
	if !e.modWindow {
		return false
	}
	s, _, _ := e.modSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	part, n := l.hit(p)
	if part == popupClose {
		e.closeObject()
		return true
	}
	if part == popupButton {
		e.modButton(n)
		return true
	}
	if e.modConfirm != "" {
		return true
	}
	if part == popupRowButton || part == popupRowControl || part == popupRowLine {
		if err := e.applyModField(); err != nil {
			return true
		}
	}
	if part == popupRowButton {
		if row, k, ok := l.rowButtonAt(p); ok {
			if i, found := e.obj.rowField[row]; found && e.obj.fields[i].Key == "source" {
				action := "up"
				if k == 1 {
					action = "down"
				}
				_ = e.modAction(action, "")
				return true
			}
			if actions := e.obj.rowActs[row]; k < len(actions) {
				_ = e.modAction("source", actions[k])
				return true
			}
		}
	}
	if part == popupRowButton || part == popupRowControl || part == popupRowLine {
		e.objectRowPress(part, n)
		return true
	}
	return false
}

func (e *MapEditor) modButtonCount() int {
	if e.modConfirm != "" {
		return 2
	}
	return 5
}
