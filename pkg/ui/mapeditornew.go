package ui

import (
	"image"
	"strconv"
)

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

// InspectionUnitTiers is implemented by a host whose new unit may take one of
// its class's tiers: the choices and the default for class, none when the
// class has one tier, the prompt's label, and the add with a chosen tier.
type InspectionUnitTiers interface {
	UnitTiers(class int64) ([]InspectionChoice, int64)
	UnitTierLabel() string
	AddUnitTier(cell image.Point, class, tier int64) (*InspectionDocument, error)
}

func (e *MapEditor) unitTiers() (InspectionUnitTiers, bool) {
	if !e.editable() {
		return nil, false
	}
	u, ok := e.doc.Edits.(InspectionUnitTiers)
	return u, ok
}

// askUnitTier turns the new-unit prompt into the tier prompt when class has
// several tiers, its text the default tier, and reports that it did.
func (e *MapEditor) askUnitTier(class int64) bool {
	host, ok := e.unitTiers()
	if !ok {
		return false
	}
	choices, def := host.UnitTiers(class)
	if len(choices) < 2 {
		return false
	}
	e.openPrompt(editorPrompt{kind: promptNewUnitTier, cell: e.prompt.cell, class: class, path: strconv.FormatInt(def, 10)})
	return true
}

// newUnitTierField is the tier prompt's one field.
func (e *MapEditor) newUnitTierField() []InspectionField {
	host, ok := e.unitTiers()
	if !ok {
		return nil
	}
	choices, _ := host.UnitTiers(e.prompt.class)
	return []InspectionField{{Key: "tier", Label: host.UnitTierLabel(), Choices: choices}}
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
	if e.prompt.kind == promptNewUnitTier {
		host, ok := e.unitTiers()
		if !ok {
			return nil, "Unit", nil
		}
		next, err := host.AddUnitTier(cell, e.prompt.class, v)
		return next, "Unit", err
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
