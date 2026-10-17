package ui

import (
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
	if !e.editable() || e.catalog || e.selected < 0 || e.selected >= len(e.doc.Records) {
		return "", -1
	}
	if r := e.doc.Records[e.selected]; fieldKind(r.Kind) {
		return r.Kind, r.Index
	}
	return "", -1
}

// promptFields are the rows the field prompt cycles through: the new sack's
// item code while a sack waits for one, else the selected record's fields.
func (e *MapEditor) promptFields() []InspectionField {
	switch e.prompt.kind {
	case promptNewScript:
		return []InspectionField{{Key: "kind", Label: "New script record", Choices: []InspectionChoice{{0, "Action"}, {1, "Condition"}, {2, "Trigger"}}}}
	case promptWorkspace:
		if host, ok := e.workspace(); ok {
			return host.WorkspaceFields(e.prompt.scope)
		}
		return nil
	case promptNewUnit:
		return e.newRecordField("Unit", "class", "Class key for the new unit")
	case promptNewUnitTier:
		return e.newUnitTierField()
	case promptNewSack:
		return e.newRecordField("Sack", "item", "Item code for the new sack")
	case promptNewStructure:
		return e.newRecordField("Structure", "kind", "Kind for the new structure")
	case promptItemRecipe:
		return e.recipeItemField()
	}
	kind, index := e.selectedRecord()
	if index < 0 {
		return nil
	}
	return e.doc.Edits.Fields(kind, index)
}

// recordKey handles the authoring keys: C arms a clone onto the next clicked
// cell, S arms a new sack on the next clicked cell, Delete removes the
// selected record, E opens the field prompt and Tab moves it to the next
// field. Each accepted change is one history entry.
func (e *MapEditor) recordKey(key string) error {
	if key == "tab" && e.objectTyping() {
		e.obj.focus, e.obj.open = e.nextObjectField(e.obj.focus), -1
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
		e.leave(leaveDrag)
		e.armPick(pickSack)
		return nil
	}
	if key == "drop" || key == "goals" {
		e.scriptKey(key)
		return nil
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
		next, err := e.doc.Edits.Delete(kind, index)
		if err := e.adoptEdit(next, err); err != nil {
			return err
		}
		e.selected = -1
		if kind == "Player" && e.filter == editorPlayersFilter {
			if rows := e.rows(); len(rows) > 0 {
				e.Select(rows[min(index, len(rows)-1)], false)
			}
		}
	case "field":
		e.leave(leavePick | leavePrompt)
		fields := e.promptFields()
		if len(fields) == 0 {
			e.message = "This record has no settable fields."
			return nil
		}
		e.openPrompt(editorPrompt{kind: promptField, path: fieldInput(fields[0])})
	}
	return nil
}

func fieldInput(f InspectionField) string {
	if f.Text {
		return f.TextValue
	}
	return strconv.FormatInt(f.Value, 10)
}

func (e *MapEditor) fieldTitle() string {
	fields := e.promptFields()
	if e.prompt.field >= len(fields) {
		return "Field / Enter: set"
	}
	if e.prompt.adds() {
		return choiceTitle(fields[e.prompt.field], e.prompt.path) + " / Enter: add"
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
	var next *InspectionDocument
	var err error
	newKind := ""
	adds := e.prompt.adds()
	if field.Text {
		host, ok := e.doc.Edits.(InspectionTextFields)
		if !ok {
			err = fmt.Errorf("This host cannot edit text fields.")
			return e.adoptEdit(nil, err)
		}
		kind, index := e.selectedRecord()
		next, err = host.SetTextField(kind, index, field.Key, text)
	} else {
		v, parseErr := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if parseErr != nil {
			e.message = "Enter a whole number."
			return parseErr
		}
		if e.prompt.kind == promptNewScript {
			kinds := []string{"Action", "Condition", "Trigger"}
			if v < 0 || v >= int64(len(kinds)) {
				return fmt.Errorf("Choose Action, Condition or Trigger")
			}
			newKind = kinds[v]
			next, err = e.doc.Edits.(InspectionScriptAuthor).AddScript(newKind)
		} else if e.prompt.kind == promptNewUnit && e.askUnitTier(v) {
			return nil
		} else if adds {
			next, newKind, err = e.addRecord(e.prompt.cell, v)
		} else {
			kind, index := e.selectedRecord()
			next, err = e.doc.Edits.SetField(kind, index, field.Key, v)
		}
	}
	if err := e.adoptEdit(next, err); err != nil {
		return err
	}
	if adds && next != nil {
		e.selectNewest(newKind)
	}
	e.leave(leavePrompt)
	return nil
}

// pickAt finishes an armed clone or sack on the ground cell under p.
func (e *MapEditor) pickAt(p image.Point) {
	pick := e.pick
	e.leave(leavePick)
	if pick == pickGoal {
		e.goalAt(p)
		return
	}
	x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	if !ok {
		e.message = "Click a map cell."
		return
	}
	if pick == pickSack {
		e.openPrompt(editorPrompt{kind: promptNewSack, cell: image.Pt(x, y)})
		return
	}
	if pick == pickUnit {
		e.openPrompt(editorPrompt{kind: promptNewUnit, cell: image.Pt(x, y)})
		return
	}
	if pick == pickStructure {
		e.openPrompt(editorPrompt{kind: promptNewStructure, cell: image.Pt(x, y)})
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
			next, err := e.doc.Edits.Move(kind, index, image.Pt(x, y))
			_ = e.adoptEdit(next, err)
		}
		return
	}
	kind, index := e.selectedRecord()
	if index < 0 || !editableKind(kind) {
		e.message = "Clone needs a selected unit, structure or sack."
		return
	}
	next, err := e.doc.Edits.Clone(kind, index, image.Pt(x, y))
	if e.adoptEdit(next, err) != nil || next == nil {
		return
	}
	e.selectNewest(kind)
}

// selectNewest selects the record of kind with the highest index, which is
// where a clone or a new sack is appended.
func (e *MapEditor) selectNewest(kind string) {
	best := -1
	for i, r := range e.doc.Records {
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
