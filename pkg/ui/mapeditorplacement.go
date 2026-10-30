package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

var editorGhost = color.RGBA{128, 128, 128, 128}

type InspectionPlacementPreview interface {
	PreviewPlacement(InspectionNewRecord) (*InspectionDocument, error)
}

// InspectionCellCheck judges a move before it is made: CheckObjectCell the
// Move of one record onto a cell, CheckMove the group move of refs by
// offset (MoveArea's when box is not empty), answering the records it
// would refuse.
type InspectionCellCheck interface {
	CheckObjectCell(string, int, image.Point) error
	CheckMove(refs []RecordRef, box image.Rectangle, offset image.Point) []RecordRef
}

type placementFrame struct {
	Cell    image.Point
	Refused bool
	Sprites []staticScreenRect
	// Cells are the footprints the placement's records will take, each
	// refused or not.
	Cells []placementCells
	// Dragged marks a record drag's silhouettes: the drag marks its cells
	// itself, so no cell box is drawn. RefusedRecords are the dragged
	// records the release would refuse.
	Dragged        bool
	RefusedRecords map[RecordRef]bool
	// Ground is the cells of a paste's terrain patch, drawn translucent
	// from GroundView, the preview of the map after the paste.
	Ground     image.Rectangle
	GroundView *Viewer
}

// placementCells is one footprint of a placement.
type placementCells struct {
	Rect    image.Rectangle
	Refused bool
}

// footprint is the cells r takes: its Footprint, or else its Cell and Size.
func (r InspectionRecord) footprint() image.Rectangle {
	if !r.Footprint.Empty() {
		return r.Footprint
	}
	return image.Rectangle{Min: r.Cell, Max: r.Cell.Add(image.Pt(max(1, r.Size.X), max(1, r.Size.Y)))}
}

// newRecords are the spatial records of d, the preview of the map after a
// placement, past the last index of their kind the open document holds.
func (e *MapEditor) newRecords(d *InspectionDocument) []InspectionRecord {
	next := map[string]int{}
	for _, r := range e.doc.Records {
		next[r.Kind] = max(next[r.Kind], r.Index+1)
	}
	var out []InspectionRecord
	for _, r := range d.Records {
		if r.Spatial && r.Index >= next[r.Kind] {
			out = append(out, r)
		}
	}
	return out
}

// editorGhostAlpha is the opacity of a placement's translucent ground, the
// translucent sprites' own.
const editorGhostAlpha = 0.5

func (e *MapEditor) placement() (placementFrame, bool) {
	if f, ok := e.dragPlacement(); ok {
		return f, true
	}
	if e.pick == pickNone || e.pick == pickArea || e.pick == pickGoal || e.doc == nil || !e.hoverOK {
		return placementFrame{}, false
	}
	f := placementFrame{Cell: e.hoverCell}
	if r, ok := e.paramAreaHover(); ok {
		f.Cells = []placementCells{{Rect: r}}
	}
	if e.pick == pickMove && e.selected >= 0 {
		r := e.doc.Records[e.selected]
		if host, ok := e.doc.Edits.(InspectionCellCheck); ok {
			f.Refused = host.CheckObjectCell(r.Kind, r.Index, f.Cell) != nil
		}
		f.Sprites = e.recordSprites(e.doc.Viewer, r, f.Cell.Sub(r.Cell))
		if f.Refused {
			f.Sprites = nil
		}
		f.Cells = []placementCells{{Rect: r.footprint().Add(f.Cell.Sub(r.Cell)), Refused: f.Refused}}
	}
	if e.pick == pickAddCell || e.pick == pickPaste {
		host, ok := e.doc.Edits.(InspectionPlacementPreview)
		if !ok {
			return f, true
		}
		r := InspectionNewRecord{Cell: f.Cell, HasCell: true, Clip: e.selection.clip}
		if e.pick == pickAddCell {
			r = e.addPlacement(f.Cell)
		}
		key := fmt.Sprint(r)
		if r.Clip != nil {
			key = fmt.Sprint(r, e.doc.Viewer.flat)
		}
		if e.placementKey != key || e.placementDoc != e.doc {
			e.placementKey, e.placementDoc = key, e.doc
			e.placementView, e.placementErr = host.PreviewPlacement(r)
			if d := e.placementView; d != nil && r.Clip != nil {
				d.Viewer.SetFlat(e.doc.Viewer.flat)
			}
		}
		f.Refused = e.placementErr != nil
		if d := e.placementView; d != nil && r.Clip == nil {
			// A refused Add's preview still holds the record.
			for _, rec := range e.newRecords(d) {
				if rec.Kind == r.Kind {
					f.Cells = append(f.Cells, placementCells{Rect: rec.footprint(), Refused: f.Refused})
				}
			}
		}
		if d := e.placementView; d != nil && !f.Refused && r.Clip != nil {
			d.Viewer.cam = e.doc.Viewer.cam
			e.pastePlacement(&f, d, r.Clip)
		} else if d != nil && !f.Refused {
			v := d.Viewer
			v.cam = e.doc.Viewer.cam
			var index int
			for _, rec := range d.Records {
				if rec.Kind == r.Kind {
					index = max(index, rec.Index)
				}
			}
			for _, s := range v.planeSprites() {
				kind := InspectionUnit
				if r.Kind == "Structure" {
					kind = InspectionStructure
				}
				if r.Kind != "Sack" && s.Inspection.Kind == kind && int(s.Inspection.ID) == index {
					f.Sprites = append(f.Sprites, s)
				}
			}
			for _, p := range v.structurePlacements() {
				if r.Kind == "Structure" && p.Class != nil && p.Class.Flat && int(p.StructureID) == index {
					if rect, ok := spriteScreenRect(p.Rect(), v.cam); ok {
						f.Sprites = append(f.Sprites, staticScreenRect{screenRect: rect, Frame: p.Frame, Brightness: 1})
					}
				}
			}
			if r.Kind == "Sack" {
				places := v.sackLayer()
				if len(places) > 0 {
					p := places[len(places)-1]
					if rect, ok := spriteScreenRect(p.Rect(), v.cam); ok {
						f.Sprites = append(f.Sprites, staticScreenRect{screenRect: rect, Frame: p.Frame})
					}
				}
			}
		}
	}
	for i := range f.Sprites {
		f.Sprites[i].Translucent = true
	}
	return f, true
}

// pastePlacement fills f with what the paste brush's clip places, read off
// d, the host's preview of the map after the paste: each record the paste
// added with its footprint, the footprint of each record it refuses, and
// for a terrain patch its cells on the map with the objects on them.
func (e *MapEditor) pastePlacement(f *placementFrame, d *InspectionDocument, clip InspectionClip) {
	v := d.Viewer
	for _, r := range e.newRecords(d) {
		if selectable(r.Kind) {
			f.Sprites = append(f.Sprites, e.recordSprites(v, r, image.Point{})...)
			f.Cells = append(f.Cells, placementCells{Rect: r.footprint()})
		}
	}
	for _, r := range d.Refused {
		if !r.Footprint.Empty() {
			f.Cells = append(f.Cells, placementCells{Rect: r.Footprint, Refused: true})
		}
	}
	size := clip.Patch()
	if size.X <= 0 || size.Y <= 0 {
		return
	}
	f.Ground = image.Rectangle{Min: f.Cell, Max: f.Cell.Add(size)}.Intersect(image.Rect(0, 0, d.Width, d.Height))
	f.GroundView = v
	for _, p := range v.staticPlacements() {
		if p.Frame == nil || !p.Cell.In(f.Ground) {
			continue
		}
		if rect, ok := spriteScreenRect(p.Rect(), v.cam); ok {
			f.Sprites = append(f.Sprites, staticScreenRect{screenRect: rect, Frame: p.Frame, Mirror: p.Mirror})
		}
	}
}

// drawPlacement draws f over dst: a paste's ground translucent, then the
// sprites.
func (e *MapEditor) drawPlacement(dst *ebiten.Image, f placementFrame) {
	e.drawPlacementGround(dst, f)
	for _, s := range f.Sprites {
		e.doc.Viewer.drawPlaneSprite(dst, s)
	}
}

// drawPlacementGround paints the cells of f's ground as its preview draws
// them, translucent.
func (e *MapEditor) drawPlacementGround(dst triangleTarget, f placementFrame) {
	v := f.GroundView
	if v == nil || f.Ground.Empty() {
		return
	}
	cells := func(fn func(tx, ty int)) {
		for y := f.Ground.Min.Y; y < f.Ground.Max.Y; y++ {
			for x := f.Ground.Min.X; x < f.Ground.Max.X; x++ {
				fn(x, y)
			}
		}
	}
	v.drawTiles(dst, v.Mode() == ModeDisplaced, cells, editorGhostAlpha)
}

func (e *MapEditor) placementPicture() *image.RGBA {
	f, ok := e.placement()
	if !ok || len(f.Sprites) == 0 {
		return nil
	}
	v := e.doc.Viewer
	pic := image.NewRGBA(image.Rect(0, 0, v.frameW, v.frameH))
	for _, s := range f.Sprites {
		if s.Frame == nil {
			continue
		}
		src := s.Frame.RGBA()
		ghost := src
		if s.Translucent {
			ghost = image.NewRGBA(src.Bounds())
			draw.DrawMask(ghost, ghost.Bounds(), src, src.Bounds().Min, image.NewUniform(editorGhost), image.Point{}, draw.Src)
		}
		editorBlit(pic, ghost, s.screenRect, s.Mirror)
	}
	return pic
}

// dragPlacement is a record drag that has moved: a translucent silhouette of
// each dragged record where the release would put it, the whole selection
// when the drag holds it, and the dragged records the release would refuse.
// A drag of the selection box moves objects too and draws no silhouettes.
// The records themselves stay until the release.
func (e *MapEditor) dragPlacement() (placementFrame, bool) {
	d := e.recordDrag
	if d == nil || !d.moved || d.area != nil || d.drop > 0 || d.record < 0 && !d.box || e.doc == nil || d.record >= len(e.doc.Records) {
		return placementFrame{}, false
	}
	f := placementFrame{Cell: d.cell, Dragged: true, RefusedRecords: e.dragRefused(d)}
	offset := d.cell.Sub(d.origin)
	for i, r := range e.doc.Records {
		if !d.box && (i == d.record || d.group && e.inSelection(i)) {
			f.Sprites = append(f.Sprites, e.recordSprites(e.doc.Viewer, r, offset)...)
		}
	}
	for i := range f.Sprites {
		f.Sprites[i].Translucent = true
	}
	return f, true
}

// dragRefused is the records d's release would refuse, as the host judges
// the move the release makes: CheckMove for the selection, CheckObjectCell
// for one record. The answer is kept while the document and the move stay.
func (e *MapEditor) dragRefused(d *editorRecordDrag) map[RecordRef]bool {
	host, ok := e.doc.Edits.(InspectionCellCheck)
	offset := d.cell.Sub(d.origin)
	if !ok || offset == (image.Point{}) {
		return nil
	}
	var box image.Rectangle
	if d.box {
		box = e.selection.box
	}
	key := fmt.Sprint(d.group, d.kind, d.index, e.selection.refs, box, offset)
	if e.dragCheckKey == key && e.dragCheckDoc == e.doc {
		return e.dragCheck
	}
	out := map[RecordRef]bool{}
	if d.group {
		for _, r := range host.CheckMove(e.selection.refs, box, offset) {
			out[r] = true
		}
	} else if host.CheckObjectCell(d.kind, d.index, d.cell) != nil {
		out[RecordRef{Kind: d.kind, Index: d.index}] = true
	}
	e.dragCheckKey, e.dragCheckDoc, e.dragCheck = key, e.doc, out
	return out
}

// recordSprites are the map sprites of record r, shifted by offset cells: the
// unit or structure body, a flat structure's floor and a sack, as v draws
// them.
func (e *MapEditor) recordSprites(v *Viewer, r InspectionRecord, offset image.Point) []staticScreenRect {
	delta := e.CellPoint(r.Cell.Add(offset)).Sub(e.CellPoint(r.Cell))
	var out []staticScreenRect
	add := func(s staticScreenRect) {
		s.X += float64(delta.X) / v.place.Scale()
		s.Y += float64(delta.Y) / v.place.Scale()
		out = append(out, s)
	}
	for _, s := range v.planeSprites() {
		if (r.Kind == "Unit" && s.Inspection.Kind == InspectionUnit || r.Kind == "Structure" && s.Inspection.Kind == InspectionStructure) && int(s.Inspection.ID) == r.Index {
			add(s)
		}
	}
	if r.Kind == "Sack" {
		for _, s := range v.sackLayer() {
			if s.Cell == r.Cell {
				if rect, ok := spriteScreenRect(s.Rect(), v.cam); ok {
					add(staticScreenRect{screenRect: rect, Frame: s.Frame})
				}
				break
			}
		}
	}
	for _, s := range v.structurePlacements() {
		if r.Kind == "Structure" && int(s.StructureID) == r.Index && s.Class != nil && s.Class.Flat {
			if rect, ok := spriteScreenRect(s.Rect(), v.cam); ok {
				add(staticScreenRect{screenRect: rect, Frame: s.Frame})
			}
		}
	}
	return out
}
