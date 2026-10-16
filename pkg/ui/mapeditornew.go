package ui

import "image"

// InspectionUnits is implemented by edits whose host adds a unit from
// nothing: a new map holds no unit to clone.
type InspectionUnits interface {
	AddUnit(cell image.Point, class int64) (*InspectionDocument, error)
}

// unitKey arms a new unit on the next clicked cell.
func (e *MapEditor) unitKey() {
	if _, ok := e.units(); !ok || e.catalog {
		return
	}
	e.leave(leaveDrag)
	e.armPick(pickUnit)
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
	if e.prompt.kind == promptNewStructure {
		host, ok := e.doc.Edits.(InspectionRecords)
		if !ok {
			return nil, "Structure", nil
		}
		next, err := host.AddStructure(cell, v)
		return next, "Structure", err
	}
	if e.prompt.kind == promptNewUnit {
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
