package game

import (
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// BrushTerrains lists the terrains the window can paint, from the model's
// palette.
func (s *mapInspectionEdits) BrushTerrains() []ui.BrushTerrain {
	out := make([]ui.BrushTerrain, len(mapedit.Palette))
	for i, p := range mapedit.Palette {
		out[i] = ui.BrushTerrain{Class: int(p.Class), Name: p.Name}
	}
	return out
}

// Paint applies one stroke to the model as one edit.
func (s *mapInspectionEdits) Paint(op ui.BrushStroke) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) { return paintStroke(m, op) })
}

// paintStroke runs a window stroke on the model: a terrain stroke is one brush
// of the tile plane, a height stroke one brush of the height plane with the
// cosine fall-off.
func paintStroke(m *mapedit.Editor, op ui.BrushStroke) (bool, error) {
	b := mapedit.Brush{Plane: mapedit.PlaneTile, Value: op.Value, Radius: op.Radius}
	if op.Height {
		b.Plane, b.Falloff = mapedit.PlaneHeight, mapedit.FalloffCosine
	}
	cells := make([]mapedit.Cell, len(op.Cells))
	for i, c := range op.Cells {
		cells[i] = mapedit.Cell{X: c.X, Y: c.Y}
	}
	return m.Paint(b, cells, mapload.TileClass)
}
