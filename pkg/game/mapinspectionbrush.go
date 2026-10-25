package game

import (
	"fmt"
	"image"
	"image/draw"
	"slices"
	"sync/atomic"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
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
	if op.Plane == ui.BrushPlaneObjects {
		for i, code := range append([]int{op.Value}, op.Forest.Kinds...) {
			if !slices.ContainsFunc(s.inspector.objectRows, func(r ui.BrushObject) bool { return r.Code == code }) || i > 0 && code == 0 {
				return nil, fmt.Errorf("%d is not a class the object list offers", code)
			}
		}
	}
	return s.change(func(m *mapedit.Editor) (bool, error) { return paintStroke(m, op, s.inspector.table) })
}

// BrushObjectIcon is the picture of an object class the forest grid shows:
// its frame from the installed sprites, nil where the install has none.
func (s *mapInspectionEdits) BrushObjectIcon(code int) *image.RGBA {
	if set := s.inspector.statics; set != nil && code > 0 && code < len(set.Classes) && set.Classes[code] != nil && set.Classes[code].Frame != nil {
		return set.Classes[code].Frame.RGBA()
	}
	return nil
}

// BrushVariants is how many look-alike tiles the terrain class has.
func (s *mapInspectionEdits) BrushVariants(class int) int {
	return len(mapedit.Variants(uint8(class)))
}

// strokes counts the random fills of this process, so two strokes in one clock
// tick still take different seeds.
var strokes atomic.Uint64

// strokeSeed is the seed of one random fill: the session clock's reading and
// the stroke's number, so no two strokes of a process share an interior.
// Tests replace it.
var strokeSeed = func() uint64 { return clockSessionSeed() + strokes.Add(1) }

// brushFills maps the window's fill modes to the model's.
var brushFills = map[ui.BrushFillMode]mapedit.FillMode{
	ui.BrushFillPattern: mapedit.FillPattern,
	ui.BrushFillRandom:  mapedit.FillRandom,
	ui.BrushFillVariant: mapedit.FillVariant,
}

// brushPlanes maps the window's plane to the model's, with the fall-off the
// plane's brush takes.
var brushPlanes = map[ui.BrushPlane]mapedit.Brush{
	ui.BrushPlaneTerrain:     {Plane: mapedit.PlaneTile},
	ui.BrushPlaneHeight:      {Plane: mapedit.PlaneHeight, Falloff: mapedit.FalloffCosine},
	ui.BrushPlaneObjects:     {Plane: mapedit.PlaneObject},
	ui.BrushPlanePassability: {Plane: mapedit.PlaneImpassable},
}

// paintStroke runs a window stroke on the model as one brush of its plane. A
// forest stroke draws from a fresh seeded source and is fenced off the cells
// the table's structure footprints attach; a nil table fences none.
func paintStroke(m *mapedit.Editor, op ui.BrushStroke, table *mapload.Table) (bool, error) {
	b, ok := brushPlanes[op.Plane]
	if !ok {
		return false, fmt.Errorf("no brush plane %d", op.Plane)
	}
	b.Value, b.Radius, b.Shape, b.Set = op.Value, op.Radius, brushShapes[op.Shape], op.Set
	if op.Plane == ui.BrushPlaneObjects && len(op.Forest.Kinds) > 0 {
		var fenced []bool
		if !op.Forest.Thin {
			doc, err := m.Map()
			if err != nil {
				return false, err
			}
			fenced = forestFence(doc, table)
		}
		b.Scatter = mapedit.Scatter{Classes: op.Forest.Kinds, Share: op.Forest.Share, Thin: op.Forest.Thin,
			Draw: random.NewGo(int64(strokeSeed())).Intn, Fenced: fenced}
	}
	if op.Plane == ui.BrushPlaneTerrain {
		b.Fill = mapedit.Fill{Mode: brushFills[op.Fill.Mode], Variant: op.Fill.Variant}
		if b.Fill.Mode == mapedit.FillRandom {
			b.Fill.Draw = random.NewGo(int64(strokeSeed())).Intn
		}
	}
	return m.Paint(b, strokeCells(op), mapload.TileClass)
}

// forestFence is the per-cell plane a planting stroke keeps clear: the cells
// the table's structure footprints attach (none for a nil table), the
// party's drop cells and the cells units stand on, the cells the map report
// checks for a blocked drop or unit.
func forestFence(doc *alm.Map, table *mapload.Table) []bool {
	var fenced []bool
	if table != nil {
		fenced = mapload.StructureCells(doc, table)
	}
	if len(fenced) != int(doc.Width)*int(doc.Height) {
		fenced = make([]bool, int(doc.Width)*int(doc.Height))
	}
	fence := func(cell image.Point) {
		if cell.In(image.Rect(0, 0, int(doc.Width), int(doc.Height))) {
			fenced[cell.Y*int(doc.Width)+cell.X] = true
		}
	}
	for _, d := range mapload.DropCells(doc) {
		fence(image.Pt(int(d.X), int(d.Y)))
	}
	for _, u := range doc.Units {
		fence(unitCell(u))
	}
	return fenced
}

// brushShapes is the model's shape of each window shape.
var brushShapes = map[ui.BrushShape]mapedit.BrushShape{ui.BrushCircle: mapedit.BrushCircle, ui.BrushSquare: mapedit.BrushSquare}

func strokeCells(op ui.BrushStroke) []mapedit.Cell {
	cells := make([]mapedit.Cell, len(op.Cells))
	for i, c := range op.Cells {
		cells[i] = mapedit.Cell{X: c.X, Y: c.Y}
	}
	return cells
}

// BrushCells is every cell of the map a stroke touches, each once: the
// model's own brush walk, so the marker shows what Paint writes.
func (s *mapInspectionEdits) BrushCells(op ui.BrushStroke) []image.Point {
	w, h := s.model.GridSize()
	b := mapedit.Brush{Plane: brushPlanes[op.Plane].Plane, Radius: op.Radius, Shape: brushShapes[op.Shape], Set: op.Set}
	var out []image.Point
	for _, c := range b.Footprint(strokeCells(op), w, h) {
		out = append(out, image.Pt(c.X, c.Y))
	}
	return out
}

// BrushMaxRadius is the largest radius the model takes.
func (s *mapInspectionEdits) BrushMaxRadius() int { return mapedit.MaxRadius }

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
