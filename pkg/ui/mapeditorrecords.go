package ui

import (
	"fmt"
	"image"
	"strconv"
)

// dragRank orders the record kinds a drag can move when one press hits
// several: a unit before a sack before a structure, whose footprint is the
// widest. A kind without a rank is not moved by a drag.
var dragRank = map[string]int{"Unit": 1, "Sack": 2, "Structure": 3}

// editableKind reports a record kind the window clones, deletes and sets
// fields of. A unit's stock has no cell and is reached from the list.
func editableKind(kind string) bool { return dragRank[kind] > 0 || kind == "Stock" }

// fieldKind reports a record kind whose fields the window sets: the editable
// kinds and the script nodes, which have no cell to clone or delete.
func fieldKind(kind string) bool {
	return editableKind(kind) || kind == "Action" || kind == "Condition"
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
	case promptNewUnit:
		return []InspectionField{{Key: "class", Label: "Class key for the new unit"}}
	case promptNewSack:
		return []InspectionField{{Key: "item", Label: "Item code for the new sack"}}
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
	if key == "tab" {
		fields := e.promptFields()
		if len(fields) > 0 {
			if e.prompt.open() {
				e.prompt.field = (e.prompt.field + 1) % len(fields)
				e.prompt.path = strconv.FormatInt(fields[e.prompt.field].Value, 10)
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
	if index < 0 || (key != "field" && !editableKind(kind)) {
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
	case "field":
		e.leave(leavePick | leavePrompt)
		fields := e.promptFields()
		if len(fields) == 0 {
			e.message = "This record has no settable fields."
			return nil
		}
		e.openPrompt(editorPrompt{kind: promptField, path: strconv.FormatInt(fields[0].Value, 10)})
	}
	return nil
}

func (e *MapEditor) fieldTitle() string {
	fields := e.promptFields()
	if e.prompt.field >= len(fields) {
		return "Field / Enter: set"
	}
	if e.prompt.adds() {
		return fields[e.prompt.field].Label + " / Enter: add"
	}
	return fmt.Sprintf("%s / Tab: next / Enter: set", fields[e.prompt.field].Label)
}

func (e *MapEditor) setField(text string) error {
	fields := e.promptFields()
	if e.prompt.field >= len(fields) {
		e.leave(leavePrompt)
		return nil
	}
	v, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		e.message = "Enter a whole number."
		return err
	}
	var next *InspectionDocument
	newKind := ""
	adds := e.prompt.adds()
	if adds {
		next, newKind, err = e.addRecord(e.prompt.cell, v)
	} else {
		kind, index := e.selectedRecord()
		next, err = e.doc.Edits.SetField(kind, index, fields[e.prompt.field].Key, v)
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
	if pick == pickDrop {
		e.dropAt(image.Pt(x, y))
		return
	}
	kind, index := e.selectedRecord()
	if index < 0 {
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
}
