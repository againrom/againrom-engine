package ui

import (
	"image"
	"image/draw"
)

type recordMarkerLabel struct {
	rect image.Rectangle
	text string
}

func (e *MapEditor) markerLabels() []recordMarkerLabel {
	if e.doc == nil {
		return nil
	}
	f := e.bodyFont()
	ox, oy := e.mapOrigin()
	area := e.mapArea()
	var out []recordMarkerLabel
	for i, r := range e.doc.Records {
		if !r.Spatial || r.Marker == "" {
			continue
		}
		if d := e.recordDrag; d != nil && d.moved && i == d.record {
			r.Cell = d.cell
		}
		wx, wy := e.cellCenter(r.Cell)
		x, y := e.doc.Viewer.cam.WorldToScreen(wx, wy)
		label := r.Marker
		width := 6
		if f != nil {
			width += f.Advance(label)
		}
		at := image.Pt(int(x+ox)+7, int(y+oy)-fontHeight(f, 15)-2)
		rect := image.Rectangle{Min: at, Max: at.Add(image.Pt(width, fontHeight(f, 15)+2))}
		for j := 0; j < len(out); {
			if rect.Overlaps(out[j].rect) {
				rect = rect.Add(image.Pt(0, fontHeight(f, 15)+2))
				j = 0
			} else {
				j++
			}
		}
		if !rect.Overlaps(area) {
			continue
		}
		out = append(out, recordMarkerLabel{rect: rect.Intersect(area), text: label})
	}
	return out
}

func (e *MapEditor) drawMarkerLabels(pic *image.RGBA) {
	for _, l := range e.markerLabels() {
		draw.Draw(pic, l.rect, image.NewUniform(editorInk), image.Point{}, draw.Src)
		if f := e.bodyFont(); f != nil {
			f.Draw(pic.SubImage(l.rect).(*image.RGBA), l.text, l.rect.Min.X+3, l.rect.Min.Y+1, editorPaper)
		}
	}
}
