package game

import (
	"fmt"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
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

// BrushObjects lists the rows of the objects list: the clear row, then each
// class the installed registry names and the game can draw, by placement byte.
func (s *mapInspectionEdits) BrushObjects() []ui.BrushObject {
	return s.inspector.objectRows
}

// Paint applies one stroke to the model as one edit.
func (s *mapInspectionEdits) Paint(op ui.BrushStroke) (*ui.InspectionDocument, error) {
	if op.Plane == ui.BrushPlaneObjects && !slices.ContainsFunc(s.inspector.objectRows, func(r ui.BrushObject) bool { return r.Code == op.Value }) {
		return nil, fmt.Errorf("%d is not a class the object list offers", op.Value)
	}
	return s.change(func(m *mapedit.Editor) (bool, error) { return paintStroke(m, op) })
}

// brushPlanes maps the window's plane to the model's, with the fall-off the
// plane's brush takes.
var brushPlanes = map[ui.BrushPlane]mapedit.Brush{
	ui.BrushPlaneTerrain: {Plane: mapedit.PlaneTile},
	ui.BrushPlaneHeight:  {Plane: mapedit.PlaneHeight, Falloff: mapedit.FalloffCosine},
	ui.BrushPlaneObjects: {Plane: mapedit.PlaneObject},
}

// paintStroke runs a window stroke on the model as one brush of its plane.
func paintStroke(m *mapedit.Editor, op ui.BrushStroke) (bool, error) {
	b, ok := brushPlanes[op.Plane]
	if !ok {
		return false, fmt.Errorf("no brush plane %d", op.Plane)
	}
	b.Value, b.Radius = op.Value, op.Radius
	cells := make([]mapedit.Cell, len(op.Cells))
	for i, c := range op.Cells {
		cells[i] = mapedit.Cell{X: c.X, Y: c.Y}
	}
	return m.Paint(b, cells, mapload.TileClass)
}

// objectBrushRows builds the objects list: the clear row, then every class of
// the object registry the game can draw (a class with a frame), in placement
// byte order, named by its DescText or, where it has none, by its code.
func objectBrushRows(src terrain.EntrySource, set *terrain.StaticSet) []ui.BrushObject {
	rows := []ui.BrushObject{{Code: 0, Name: "No object"}}
	if set == nil {
		return rows
	}
	names := map[int]string{}
	if stream, err := src.ReadFile(ObjectRegistry); err == nil {
		if r, err := reg.Parse(stream); err == nil {
			if classes, err := data.LoadObjectClasses(r); err == nil {
				for b := 1; b <= 0xff; b++ {
					if c, ok := classes.ByCode(byte(b)); ok {
						names[b] = c.DescText
					}
				}
			}
		}
	}
	for b := 1; b <= 0xff; b++ {
		if c := set.Classes[b]; c == nil || c.Frame == nil {
			continue
		}
		name := names[b]
		if name == "" {
			name = fmt.Sprintf("Object %d", b)
		}
		rows = append(rows, ui.BrushObject{Code: b, Name: name})
	}
	return rows
}
