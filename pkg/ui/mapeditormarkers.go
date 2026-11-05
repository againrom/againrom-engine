package ui

import "image"

func (e *MapEditor) markerLabelRows() []mapLabel {
	if e.doc == nil {
		return nil
	}
	ox, oy := e.mapOrigin()
	var out []mapLabel
	for i, r := range e.doc.Records {
		if !r.Spatial || r.Marker == "" {
			continue
		}
		if d := e.recordDrag; d != nil && d.moved && i == d.record {
			r.Cell = d.cell
		}
		wx, wy := e.cellCenter(r.Cell)
		x, y := e.doc.Viewer.cam.WorldToScreen(wx, wy)
		out = append(out, mapLabel{
			source: labelRecord, index: i, anchor: image.Pt(int(x+ox), int(y+oy)),
			offset: image.Pt(7, 0), above: true, text: r.Marker,
			padding: image.Pt(3, 1), ink: editorPaper, edge: labelClip,
			collides: labelTrigger | labelRecord, moveDown: true, layer: 1,
		})
	}
	return out
}
