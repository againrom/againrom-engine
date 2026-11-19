package game

import (
	"errors"
	"image"

	"againrom/pkg/ui"
)

// secondEdits is the edit host of a second-game map. It offers the terrain
// brush, undo, redo and the save: the tools whose written fields the format
// notes read. Every other call is refused with one line, and a tool whose
// host seam is left out here is off.
type secondEdits struct{ s *mapInspectionEdits }

func (c secondEdits) refusal() error {
	return errors.New(c.s.inspector.word("editor.map.secondtool"))
}

func (c secondEdits) Dirty() bool   { return c.s.Dirty() }
func (c secondEdits) CanUndo() bool { return c.s.CanUndo() }
func (c secondEdits) CanRedo() bool { return c.s.CanRedo() }

func (c secondEdits) Move(ui.RecordKind, int, image.Point) (*ui.InspectionDocument, error) {
	return nil, c.refusal()
}

func (c secondEdits) Clone(ui.RecordKind, int, image.Point) (*ui.InspectionDocument, error) {
	return nil, c.refusal()
}

func (c secondEdits) Delete(ui.RecordKind, int) (*ui.InspectionDocument, error) {
	return nil, c.refusal()
}

func (c secondEdits) Fields(ui.RecordKind, int) []ui.InspectionField { return nil }

func (c secondEdits) SetField(ui.RecordKind, int, string, int64) (*ui.InspectionDocument, error) {
	return nil, c.refusal()
}

func (c secondEdits) Undo() (*ui.InspectionDocument, error) { return c.s.Undo() }
func (c secondEdits) Redo() (*ui.InspectionDocument, error) { return c.s.Redo() }

func (c secondEdits) SaveAs(path string) (string, error) { return c.s.SaveAs(path) }
func (c secondEdits) SaveFolder() string                 { return c.s.SaveFolder() }
func (c secondEdits) OwnFile() (string, bool)            { return c.s.OwnFile() }

func (c secondEdits) SaveInPlace() (string, error) { return c.s.SaveInPlace() }

func (c secondEdits) BrushTerrains() []ui.BrushTerrain { return c.s.BrushTerrains() }
func (c secondEdits) BrushObjects() []ui.BrushObject   { return c.s.BrushObjects() }
func (c secondEdits) BrushCells(op ui.BrushStroke) []image.Point {
	return c.s.BrushCells(op)
}
func (c secondEdits) BrushMaxRadius() int                  { return c.s.BrushMaxRadius() }
func (c secondEdits) BrushVariants(class int) int          { return c.s.BrushVariants(class) }
func (c secondEdits) BrushObjectIcon(code int) *image.RGBA { return c.s.BrushObjectIcon(code) }

// Paint takes a terrain stroke; the height, object, passability and bridge
// planes are refused.
func (c secondEdits) Paint(op ui.BrushStroke) (*ui.InspectionDocument, error) {
	if op.Plane != ui.BrushPlaneTerrain {
		return nil, c.refusal()
	}
	return c.s.Paint(op)
}

// The palettes list the catalog and every add is refused.
func (c secondEdits) NewRecordStep(r ui.InspectionNewRecord, key string) ui.InspectionNewStep {
	return viewPalettes{c.s}.NewRecordStep(r, key)
}

func (c secondEdits) CheckNewRecord(r ui.InspectionNewRecord, key string) error {
	return viewPalettes{c.s}.CheckNewRecord(r, key)
}

func (c secondEdits) NewRecordPreview(kind ui.RecordKind, key string, value int64) *image.RGBA {
	return viewPalettes{c.s}.NewRecordPreview(kind, key, value)
}

func (c secondEdits) AddRecord(r ui.InspectionNewRecord) (*ui.InspectionDocument, error) {
	return viewPalettes{c.s}.AddRecord(r)
}

func (c secondEdits) AddKinds() []ui.RecordKind { return viewPalettes{c.s}.AddKinds() }
