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
	if e.sackCell != nil {
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
			e.field = (e.field + 1) % len(fields)
			e.path, e.message = strconv.FormatInt(fields[e.field].Value, 10), ""
		}
		return nil
	}
	if !e.editable() {
		return nil
	}
	if key == "sack" {
		e.cancelRecordDrag()
		e.pick, e.message = "sack", ""
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
	e.cancelRecordDrag()
	switch key {
	case "clone":
		e.pick, e.message = "clone", ""
	case "delete":
		e.pick = ""
		next, err := e.doc.Edits.Delete(kind, index)
		if err := e.adoptEdit(next, err); err != nil {
			return err
		}
		e.selected = -1
	case "field":
		e.pick, e.sackCell = "", nil
		fields := e.promptFields()
		if len(fields) == 0 {
			e.message = "This record has no settable fields."
			return nil
		}
		e.openPrompt(strconv.FormatInt(fields[0].Value, 10))
	}
	return nil
}

func (e *MapEditor) openPrompt(text string) {
	e.pathMode, e.pathSave, e.fieldMode, e.field = true, false, true, 0
	e.path, e.message = text, ""
}

func (e *MapEditor) fieldTitle() string {
	fields := e.promptFields()
	if e.field >= len(fields) {
		return "Field / Enter: set"
	}
	if e.sackCell != nil {
		return fields[e.field].Label + " / Enter: add"
	}
	return fmt.Sprintf("%s / Tab: next / Enter: set", fields[e.field].Label)
}

func (e *MapEditor) setField(text string) error {
	fields := e.promptFields()
	if e.field >= len(fields) {
		e.pathMode, e.fieldMode, e.sackCell = false, false, nil
		return nil
	}
	v, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		e.message = "Enter a whole number."
		return err
	}
	var next *InspectionDocument
	if e.sackCell != nil {
		next, err = e.doc.Edits.AddSack(*e.sackCell, v)
	} else {
		kind, index := e.selectedRecord()
		next, err = e.doc.Edits.SetField(kind, index, fields[e.field].Key, v)
	}
	if err := e.adoptEdit(next, err); err != nil {
		return err
	}
	if e.sackCell != nil && next != nil {
		e.selectNewest("Sack")
	}
	e.pathMode, e.fieldMode, e.sackCell = false, false, nil
	return nil
}

// pickAt finishes an armed clone or sack on the ground cell under p.
func (e *MapEditor) pickAt(p image.Point) {
	pick := e.pick
	e.pick = ""
	if pick == "goal" {
		e.goalAt(p)
		return
	}
	x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	if !ok {
		e.message = "Click a map cell."
		return
	}
	if pick == "sack" {
		cell := image.Pt(x, y)
		e.sackCell = &cell
		e.openPrompt("")
		return
	}
	if pick == "drop" {
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
