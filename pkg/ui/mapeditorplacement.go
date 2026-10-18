package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"
)

var editorGhost = color.RGBA{128, 128, 128, 128}

type InspectionPlacementPreview interface {
	PreviewPlacement(InspectionNewRecord) (*InspectionDocument, error)
}

type InspectionCellCheck interface {
	CheckObjectCell(string, int, image.Point) error
}

type placementFrame struct {
	Cell    image.Point
	Refused bool
	Sprites []staticScreenRect
}

func (e *MapEditor) addPlacement(cell image.Point) InspectionNewRecord {
	r, ok := e.addRunning()
	if !ok {
		return InspectionNewRecord{}
	}
	host, _ := e.addHost()
	out := InspectionNewRecord{Kind: r.kind, Cell: cell, HasCell: true}
	for i, step := range r.steps {
		if step.Control == addCell || step.Control == addKind {
			continue
		}
		text := e.addText(r, i)
		offer := host.NewRecordStep(out, step.Key)
		if step.Control == addMany {
			values := r.many[i]
			if v, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64); err == nil {
				values = append(append([]int64(nil), values...), v)
			}
			if len(values) == 0 && len(offer.Choices) > 0 {
				values = []int64{offer.Choices[0].Value}
			}
			for _, v := range values {
				out.Values = append(out.Values, InspectionFieldValue{Key: step.Key, Value: v})
			}
			continue
		}
		if step.Control == addName {
			out.Values = append(out.Values, InspectionFieldValue{Key: step.Key, Text: true, TextValue: text})
			continue
		}
		v, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil {
			v = offer.Value
		}
		out.Values = append(out.Values, InspectionFieldValue{Key: step.Key, Value: v})
	}
	return out
}

func (e *MapEditor) placement() (placementFrame, bool) {
	if e.pick == pickNone || e.pick == pickArea || e.pick == pickGoal || e.doc == nil || !e.hoverOK {
		return placementFrame{}, false
	}
	f := placementFrame{Cell: e.hoverCell}
	if e.pick == pickMove && e.selected >= 0 {
		r := e.doc.Records[e.selected]
		if host, ok := e.doc.Edits.(InspectionCellCheck); ok {
			f.Refused = host.CheckObjectCell(r.Kind, r.Index, f.Cell) != nil
		}
		v := e.doc.Viewer
		delta := e.CellPoint(f.Cell).Sub(e.CellPoint(r.Cell))
		add := func(s staticScreenRect) {
			s.X += float64(delta.X) / v.place.Scale()
			s.Y += float64(delta.Y) / v.place.Scale()
			f.Sprites = append(f.Sprites, s)
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
		if f.Refused {
			f.Sprites = nil
		}
	}
	if e.pick == pickAddCell {
		host, ok := e.doc.Edits.(InspectionPlacementPreview)
		if !ok {
			return f, true
		}
		r := e.addPlacement(f.Cell)
		key := fmt.Sprint(r)
		if e.placementKey != key || e.placementDoc != e.doc {
			e.placementKey, e.placementDoc = key, e.doc
			e.placementView, e.placementErr = host.PreviewPlacement(r)
		}
		f.Refused = e.placementErr != nil
		if d := e.placementView; d != nil && !f.Refused {
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
