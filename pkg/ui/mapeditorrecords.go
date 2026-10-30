package ui

import (
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"
)

// dragRank orders the record kinds a drag can move when one press hits
// several: a unit before a sack before a structure, whose footprint is the
// widest. A kind without a rank is not moved by a drag.
var dragRank = map[string]int{"Unit": 1, "Sack": 2, "Structure": 3}

// editableKind reports a record kind the window clones, deletes and sets
// fields of. A unit's stock has no cell and is reached from the list.
func editableKind(kind string) bool { return dragRank[kind] > 0 || kind == "Stock" }

// fieldKind reports a record kind whose fields the window sets: the editable
// kinds, script nodes and nonspatial Players.
func fieldKind(kind string) bool {
	return editableKind(kind) || scriptRecord(kind) || kind == "Player" || kind == "Enchantment"
}

// selectedRecord is the kind and file-order index of the selected editable
// record, or index -1.
func (e *MapEditor) selectedRecord() (string, int) {
	if e.modWindow && e.editable() {
		return "Mod", 0
	}
	if !e.editable() || e.catalog || e.selected < 0 || e.selected >= len(e.doc.Records) {
		return "", -1
	}
	if r := e.doc.Records[e.selected]; fieldKind(r.Kind) {
		return r.Kind, r.Index
	}
	return "", -1
}

// promptFields are the rows the field prompt cycles through: the recipe's
// item, the workspace's fields, else the selected record's fields.
func (e *MapEditor) promptFields() []InspectionField {
	switch e.prompt.kind {
	case promptWorkspace:
		if host, ok := e.workspace(); ok {
			return host.WorkspaceFields(e.prompt.scope)
		}
		return nil
	case promptItemRecipe:
		return e.recipeItemField()
	case promptAdd:
		return nil
	}
	kind, index := e.selectedRecord()
	if index < 0 {
		return nil
	}
	return e.doc.Edits.Fields(kind, index)
}

// recordKey handles the authoring keys: C arms a clone onto the next clicked
// cell, S opens the Add window on a new sack, Delete removes the selected
// record, E opens the field prompt and Tab moves it to the next field. Each
// accepted change is one history entry.
func (e *MapEditor) recordKey(key string) error {
	if key == "tab" && !e.objectTyping() && !e.objectHidden && e.doc != nil {
		e.loadObject()
		if e.nativeUnitWindow() {
			e.objectSpec()
			e.focusObjectField(e.nextObjectField(-1))
			return nil
		}
	}
	if key == "tab" && e.objectTyping() {
		e.objectSpec()
		next := e.nextObjectField(e.obj.focus)
		if e.nativeUnitWindow() && e.fieldRow(next) <= e.fieldRow(e.obj.focus) {
			e.obj.section = (e.obj.section + 1) % 4
			if e.unitWindowWidth() >= 1100 && e.obj.section < 3 {
				e.obj.section = 3
			}
			e.objectTop = 0
			e.objectSpec()
			next = e.nextObjectField(-1)
		}
		e.focusObjectField(next)
		return nil
	}
	if key == "tab" && e.prompt.kind == promptAdd {
		return nil
	}
	if key == "tab" {
		fields := e.promptFields()
		if len(fields) > 0 {
			if e.prompt.open() {
				e.prompt.field = (e.prompt.field + 1) % len(fields)
				e.prompt.path = fieldInput(fields[e.prompt.field])
			}
			e.message = ""
		}
		return nil
	}
	if !e.editable() {
		return nil
	}
	if key == "sack" {
		if !e.catalog {
			e.openAdd("Sack")
		}
		return nil
	}
	if key == "drop" || key == "goals" {
		e.scriptKey(key)
		return nil
	}
	if key == "delete" && e.deleteDropZone() {
		return nil
	}
	if key == "delete" && e.selectionShown() {
		return e.deleteSelection()
	}
	if i := e.selectedScenery(); key == "delete" && i >= 0 {
		e.leave(leaveDrag | leavePick)
		return e.deleteSelected("Scenery", i)
	}
	if key == "field" && e.report.open && len(e.objectBack) == 0 && e.goalRecord(e.report.sel) >= 0 {
		e.openGoal(e.report.sel)
		return nil
	}
	if key == "field" && e.report.open && len(e.report.goals) == 0 && len(e.objectBack) == 0 {
		if f, ok := e.selectedFinding(); ok && f.Kind == "" && f.Rule != keyListRule && f.Text != "" {
			e.message = f.Text
			return nil
		}
	}
	kind, index := e.selectedRecord()
	if index < 0 || (key != "field" && !editableKind(kind) && !(key == "delete" && (kind == "Player" || kind == "Enchantment" || scriptRecord(kind)))) {
		e.message = "Select a unit, structure or sack first."
		return nil
	}
	e.leave(leaveDrag)
	switch key {
	case "clone":
		e.armPick(pickClone)
	case "delete":
		e.leave(leavePick)
		if kind == "Unit" && e.askDelete([]RecordRef{{kind, index}}, true) {
			return nil
		}
		return e.deleteSelected(kind, index)
	case "field":
		e.leave(leavePick | leavePrompt)
		e.loadObject()
		if e.nativeUnitWindow() {
			e.objectHidden = false
			e.objectSpec()
			e.focusObjectField(e.nextObjectField(-1))
			return nil
		}
		fields := e.promptFields()
		if len(fields) == 0 {
			e.message = "This record has no settable fields."
			return nil
		}
		e.openPrompt(editorPrompt{kind: promptField, path: fieldInput(fields[0])})
	}
	return nil
}

// fieldInput is field f's value as its edit shows it; an inheriting value
// shows empty.
func fieldInput(f InspectionField) string {
	if f.Text {
		return f.TextValue
	}
	if f.Inherit && f.Value == f.Absent {
		return ""
	}
	return strconv.FormatInt(f.Value, 10)
}

// errStoredAsInherited refuses a typed 0 in a field whose record stores 0 as
// "inherit".
var errStoredAsInherited = errors.New("0 is stored as inherited")

// fieldNumber is the number text writes into field f: an empty inheriting
// field writes its Absent value.
func fieldNumber(f InspectionField, text string) (int64, error) {
	text = strings.TrimSpace(text)
	if f.Inherit && text == "" {
		return f.Absent, nil
	}
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, err
	}
	if f.Inherit && f.Absent == 0 && n == 0 {
		return 0, errStoredAsInherited
	}
	return n, nil
}

// fieldNumberNote is the popup's reason a typed value of field f is refused.
func (e *MapEditor) fieldNumberNote(f InspectionField, err error) string {
	if errors.Is(err, errStoredAsInherited) {
		return fmt.Sprintf(e.word("editor.unit.zero.error"), f.Label)
	}
	return fmt.Sprintf("%s: %s", f.Label, e.word("editor.object.number"))
}

func (e *MapEditor) fieldTitle() string {
	fields := e.promptFields()
	if e.prompt.field >= len(fields) {
		return "Field / Enter: set"
	}
	if e.prompt.kind == promptWorkspace && e.prompt.scope == "pages" {
		return fmt.Sprintf(e.word("editor.pages.prompt"), e.editorHostText(choiceTitle(fields[e.prompt.field], e.prompt.path)))
	}
	return fmt.Sprintf("%s / Tab: next / Enter: set", choiceTitle(fields[e.prompt.field], e.prompt.path))
}

func (e *MapEditor) setField(text string) error {
	if e.prompt.kind == promptWorkspace {
		return e.setWorkspaceField(text)
	}
	fields := e.promptFields()
	if e.prompt.field >= len(fields) {
		e.leave(leavePrompt)
		return nil
	}
	field := fields[e.prompt.field]
	kind, index := e.selectedRecord()
	edit := fieldEdit{Kind: kind, Index: index, Value: InspectionFieldValue{Key: field.Key, Text: field.Text, TextValue: text}}
	if !field.Text {
		v, parseErr := fieldNumber(field, text)
		if errors.Is(parseErr, errStoredAsInherited) {
			e.message = e.fieldNumberNote(field, parseErr)
			return parseErr
		}
		if parseErr != nil {
			e.message = "Enter a whole number."
			return parseErr
		}
		if v == NewMessageChoice {
			if i := e.messageFieldOf(field.Key); i >= 0 {
				e.leave(leavePrompt)
				e.beginMessage(InspectionMessageOwner{Kind: kind, Index: index, Key: field.Key}, true)
				return nil
			}
		}
		edit.Value.TextValue, edit.Value.Value = "", v
	}
	if _, err := e.commandSetField(edit); err != nil {
		e.message = err.Error()
		return err
	}
	e.leave(leavePrompt)
	return nil
}

// messageFieldOf is the object popup's message field of key, -1 for none.
func (e *MapEditor) messageFieldOf(key string) int {
	e.loadObject()
	for i, f := range e.obj.fields {
		if f.Key == key && e.messageField(i) {
			return i
		}
	}
	return -1
}

// pickAt finishes an armed pick on the ground cell under p.
func (e *MapEditor) pickAt(p image.Point) {
	pick := e.pick
	e.leave(leavePick)
	if pick == pickGoal {
		e.goalAt(p)
		return
	}
	if pick == pickParamUnit || pick == pickParamArea {
		e.paramAt(pick, p)
		return
	}
	if pick == pickPaste {
		_ = e.pasteAt(p)
		return
	}
	x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	if !ok {
		e.message = "Click a map cell."
		return
	}
	if pick == pickAddCell {
		e.addPicked(image.Pt(x, y))
		return
	}
	if pick == pickDrop {
		e.dropAt(image.Pt(x, y))
		return
	}
	if pick == pickScriptCell {
		e.scriptCellPicked(image.Pt(x, y))
		return
	}
	if pick == pickMove {
		if kind, index := e.selectedRecord(); index >= 0 && editableKind(kind) {
			cell := image.Pt(x, y)
			e.obj.move = &cell
		}
		return
	}
	kind, index := e.selectedRecord()
	if index < 0 || !editableKind(kind) {
		e.message = "Clone needs a selected unit, structure or sack."
		return
	}
	ch, err := e.commandClone(recordAt{Ref: RecordRef{Kind: kind, Index: index}, Cell: image.Pt(x, y)})
	if err != nil {
		e.message = err.Error()
		return
	}
	if ch.doc == nil {
		return
	}
	e.selectNewest(kind)
}

// selectNewest selects the Add target, or the highest index of kind.
func (e *MapEditor) selectNewest(kind string, target ...RecordRef) {
	e.messagePage = 0
	best := -1
	for i, r := range e.doc.Records {
		if len(target) > 0 && target[0].Kind == kind && r.Index != target[0].Index {
			continue
		}
		if r.Kind == kind && (best < 0 || r.Index > e.doc.Records[best].Index) {
			best = i
		}
	}
	e.selected = best
	if best >= 0 {
		visible := false
		for _, row := range e.rows() {
			visible = visible || row == best
		}
		if !visible {
			e.filter = 0
		}
		e.Select(best, false)
	}
}

// deleteSelected removes the selected record.
func (e *MapEditor) deleteSelected(kind string, index int) error {
	if _, err := e.commandDelete(RecordRef{Kind: kind, Index: index}); err != nil {
		e.message = err.Error()
		return err
	}
	e.selected = -1
	if kind == "Player" && e.filter == editorPlayersFilter {
		if rows := e.rows(); len(rows) > 0 {
			e.Select(rows[min(index, len(rows)-1)], false)
		}
	}
	return nil
}
