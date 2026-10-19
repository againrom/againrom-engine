package game

import (
	"fmt"
	"image"
	"image/draw"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// BrushTerrains lists the terrains the window can paint, from the model's
// palette, each with a swatch of its base word from the install's tiles.
func (s *mapInspectionEdits) BrushTerrains() []ui.BrushTerrain {
	swatches := s.inspector.terrainSwatches()
	out := make([]ui.BrushTerrain, len(mapedit.Palette))
	for i, p := range mapedit.Palette {
		out[i] = ui.BrushTerrain{Class: int(p.Class), Name: p.Name, Swatch: swatches[i]}
	}
	return out
}

// terrainSwatches is one picture per Palette row: the cell its base word
// draws from the installed tileset, nil where the install lacks it. They are
// drawn once and kept.
func (x *MapInspector) terrainSwatches() []*image.RGBA {
	if x.swatches != nil {
		return x.swatches
	}
	x.swatches = make([]*image.RGBA, len(mapedit.Palette))
	for i, p := range mapedit.Palette {
		ref := terrain.Resolve(p.Word)
		cell := x.tiles.Slot(ref.Slot).SubCell(ref.Sub)
		if cell == nil {
			continue
		}
		pic := image.NewRGBA(cell.Bounds())
		draw.Draw(pic, pic.Bounds(), cell, cell.Bounds().Min, draw.Src)
		x.swatches[i] = pic
	}
	return x.swatches
}

// passArms is the window's form of the world's ground-block reasons.
func passArms(arms []mapload.GroundArm) []ui.PassArm {
	pairs := [...]struct {
		from mapload.GroundArm
		to   ui.PassArm
	}{{mapload.GroundBorder, ui.PassBorder}, {mapload.GroundMountain, ui.PassMountain}, {mapload.GroundWater, ui.PassWater},
		{mapload.GroundFlag, ui.PassFlag}, {mapload.GroundObject, ui.PassObject}, {mapload.GroundStructure, ui.PassStructure}}
	out := make([]ui.PassArm, len(arms))
	for i, a := range arms {
		for _, p := range pairs {
			if a&p.from != 0 {
				out[i] |= p.to
			}
		}
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
	ui.BrushPlaneTerrain:     {Plane: mapedit.PlaneTile},
	ui.BrushPlaneHeight:      {Plane: mapedit.PlaneHeight, Falloff: mapedit.FalloffCosine},
	ui.BrushPlaneObjects:     {Plane: mapedit.PlaneObject},
	ui.BrushPlanePassability: {Plane: mapedit.PlaneImpassable},
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
// byte order, named by its DescText. The editor words the clear row and a
// class without a DescText in the install's language.
func objectBrushRows(src terrain.EntrySource, set *terrain.StaticSet) []ui.BrushObject {
	rows := []ui.BrushObject{{Code: 0}}
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
		rows = append(rows, ui.BrushObject{Code: b, Name: names[b]})
	}
	return rows
}
