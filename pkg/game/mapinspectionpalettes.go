package game

import (
	"errors"
	"image"

	"againrom/pkg/ui"
)

// viewPalettes is the host of a map the editor only views: the unit,
// structure and sack palettes list the catalog through the map's own host,
// and every add is refused.
type viewPalettes struct{ s *mapInspectionEdits }

func (v viewPalettes) NewRecordStep(r ui.InspectionNewRecord, key string) ui.InspectionNewStep {
	switch key {
	case "identity", "kind", "item":
		return v.s.NewRecordStep(r, key)
	}
	return ui.InspectionNewStep{Absent: true}
}

func (v viewPalettes) CheckNewRecord(ui.InspectionNewRecord, string) error { return v.refusal() }

func (v viewPalettes) NewRecordPreview(kind ui.RecordKind, key string, value int64) *image.RGBA {
	return v.s.NewRecordPreview(kind, key, value)
}

func (v viewPalettes) AddRecord(ui.InspectionNewRecord) (*ui.InspectionDocument, error) {
	return nil, v.refusal()
}

func (v viewPalettes) refusal() error {
	return errors.New(v.s.inspector.word("editor.map.viewonly"))
}
