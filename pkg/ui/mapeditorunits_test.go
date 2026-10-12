package ui

import "image"

// noUnitAuthoring completes InspectionEdits for probes that only move units.
type noUnitAuthoring struct{}

func (noUnitAuthoring) CloneUnit(int, image.Point) (*InspectionDocument, error) { return nil, nil }
func (noUnitAuthoring) DeleteUnit(int) (*InspectionDocument, error)             { return nil, nil }
func (noUnitAuthoring) UnitFields(int) []InspectionField                        { return nil }
func (noUnitAuthoring) SetUnitField(int, string, int64) (*InspectionDocument, error) {
	return nil, nil
}
