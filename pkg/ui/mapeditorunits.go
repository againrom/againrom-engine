package ui

import (
	"fmt"
	"image"
	"strconv"
)

// selectedUnit is the file-order index of the selected unit record, or -1.
func (e *MapEditor) selectedUnit() int {
	if !e.editable() || e.catalog || e.selected < 0 || e.selected >= len(e.doc.Records) {
		return -1
	}
	if r := e.doc.Records[e.selected]; r.Kind == "Unit" {
		return r.Index
	}
	return -1
}

// unitKey handles the unit authoring keys: C arms a clone onto the next
// clicked cell, Delete removes the unit, E opens the field prompt and Tab
// moves it to the next field. Each accepted change is one history entry.
func (e *MapEditor) unitKey(key string) error {
	if key == "tab" {
		fields := e.doc.Edits.UnitFields(e.selectedUnit())
		if len(fields) > 0 {
			e.field = (e.field + 1) % len(fields)
			e.path, e.message = strconv.FormatInt(fields[e.field].Value, 10), ""
		}
		return nil
	}
	index := e.selectedUnit()
	if !e.editable() {
		return nil
	}
	if index < 0 {
		e.message = "Select a unit first."
		return nil
	}
	e.cancelUnitDrag()
	switch key {
	case "clone":
		e.cloning, e.message = true, ""
	case "delete":
		e.cloning = false
		next, err := e.doc.Edits.DeleteUnit(index)
		if err := e.adoptEdit(next, err); err != nil {
			return err
		}
		e.selected = -1
	case "field":
		fields := e.doc.Edits.UnitFields(index)
		if len(fields) == 0 {
			e.message = "This unit has no settable fields."
			return nil
		}
		e.cloning = false
		e.pathMode, e.pathSave, e.fieldMode, e.field = true, false, true, 0
		e.path, e.message = strconv.FormatInt(fields[0].Value, 10), ""
	}
	return nil
}

func (e *MapEditor) fieldTitle() string {
	fields := e.doc.Edits.UnitFields(e.selectedUnit())
	if e.field >= len(fields) {
		return "Field / Enter: set"
	}
	return fmt.Sprintf("%s / Tab: next / Enter: set", fields[e.field].Label)
}

func (e *MapEditor) setField(text string) error {
	index := e.selectedUnit()
	fields := e.doc.Edits.UnitFields(index)
	if e.field >= len(fields) {
		e.pathMode, e.fieldMode = false, false
		return nil
	}
	v, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		e.message = "Enter a whole number."
		return err
	}
	next, err := e.doc.Edits.SetUnitField(index, fields[e.field].Key, v)
	if err := e.adoptEdit(next, err); err != nil {
		return err
	}
	e.pathMode, e.fieldMode = false, false
	return nil
}

// cloneAt clones the selected unit onto the ground cell under p and selects
// the clone, which is the highest unit index after the edit.
func (e *MapEditor) cloneAt(p image.Point) {
	e.cloning = false
	index := e.selectedUnit()
	x, y, ok := e.doc.Viewer.groundCellAt(float64(p.X), float64(p.Y))
	if index < 0 || !ok {
		e.message = "Clone needs a selected unit and a map cell."
		return
	}
	next, err := e.doc.Edits.CloneUnit(index, image.Pt(x, y))
	if e.adoptEdit(next, err) != nil || next == nil {
		return
	}
	best := -1
	for i, r := range e.doc.Records {
		if r.Kind == "Unit" && (best < 0 || r.Index > e.doc.Records[best].Index) {
			best = i
		}
	}
	e.selected = best
}
