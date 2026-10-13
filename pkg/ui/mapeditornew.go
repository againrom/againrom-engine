package ui

import "image"

// InspectionUnits is implemented by edits whose host adds a unit from
// nothing: a new map holds no unit to clone.
type InspectionUnits interface {
	AddUnit(cell image.Point, class int64) (*InspectionDocument, error)
}

// helpPickUnit is the rail text while a unit is armed.
var helpPickUnit = []string{"Unit: click a cell", "Esc: cancel"}

// newKey handles N: the catalogue opens at its first row, where the new-map
// sizes are listed above the installed maps.
func (e *MapEditor) newKey() {
	e.cancelRecordDrag()
	e.closeReport()
	e.pick, e.fieldMode = "", false
	e.catalog = true
	e.offset, e.listWheel = 0, 0
	e.message = ""
}

// unitKey arms a new unit on the next clicked cell.
func (e *MapEditor) unitKey() {
	if _, ok := e.units(); !ok || e.catalog {
		return
	}
	e.cancelRecordDrag()
	e.pick, e.message = "unit", ""
}

func (e *MapEditor) units() (InspectionUnits, bool) {
	if !e.editable() {
		return nil, false
	}
	u, ok := e.doc.Edits.(InspectionUnits)
	return u, ok
}

// addRecord finishes the prompt of a new sack or unit: v is the item code or
// the class key, and the record is added on the cell the click picked.
func (e *MapEditor) addRecord(cell image.Point, v int64) (*InspectionDocument, string, error) {
	if e.newUnit {
		u, ok := e.units()
		if !ok {
			return nil, "Unit", nil
		}
		next, err := u.AddUnit(cell, v)
		return next, "Unit", err
	}
	next, err := e.doc.Edits.AddSack(cell, v)
	return next, "Sack", err
}
