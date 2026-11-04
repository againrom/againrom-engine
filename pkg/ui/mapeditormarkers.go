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
	var occupied []image.Rectangle
	for _, l := range e.areaLabels() {
		occupied = append(occupied, l.rect)
	}
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
		for j := 0; j < len(occupied); {
			if rect.Overlaps(occupied[j]) {
				rect = rect.Add(image.Pt(0, fontHeight(f, 15)+2))
				j = 0
			} else {
				j++
			}
		}
		if !rect.Overlaps(area) {
			continue
		}
		out = append(out, recordMarkerLabel{rect: rect, text: label})
		occupied = append(occupied, rect)
	}
	return out
}

func (e *MapEditor) drawMarkerLabels(pic *image.RGBA) {
	for _, l := range e.markerLabels() {
		clip := l.rect.Intersect(e.mapArea())
		draw.Draw(pic, clip, image.NewUniform(editorInk), image.Point{}, draw.Src)
		if f := e.bodyFont(); f != nil {
			x, y := l.rect.Min.X+3, l.rect.Min.Y+1
			if y < clip.Min.Y || y+f.Height() > clip.Max.Y {
				continue
			}
			label := l.text
			for len(label) > 0 && x < clip.Min.X {
				x += f.Advance(label[:1])
				label = label[1:]
			}
			label = editorClip(f, label, clip.Max.X-x)
			f.Draw(pic.SubImage(clip).(*image.RGBA), label, x, y, editorPaper)
		}
	}
}
