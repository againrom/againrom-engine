package ui

import "image"

// noUnitAuthoring completes InspectionEdits for probes that only move records.
type noUnitAuthoring struct{}

func (noUnitAuthoring) Clone(RecordKind, int, image.Point) (*InspectionDocument, error) {
	return nil, nil
}
func (noUnitAuthoring) Delete(RecordKind, int) (*InspectionDocument, error) { return nil, nil }
func (noUnitAuthoring) Fields(RecordKind, int) []InspectionField            { return nil }
func (noUnitAuthoring) SetField(RecordKind, int, string, int64) (*InspectionDocument, error) {
	return nil, nil
}
