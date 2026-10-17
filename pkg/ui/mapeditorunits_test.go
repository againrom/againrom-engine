package ui

import "image"

// noUnitAuthoring completes InspectionEdits for probes that only move records.
type noUnitAuthoring struct{}

func (noUnitAuthoring) Clone(string, int, image.Point) (*InspectionDocument, error) { return nil, nil }
func (noUnitAuthoring) Delete(string, int) (*InspectionDocument, error)             { return nil, nil }
func (noUnitAuthoring) Fields(string, int) []InspectionField                        { return nil }
func (noUnitAuthoring) SetField(string, int, string, int64) (*InspectionDocument, error) {
	return nil, nil
}
