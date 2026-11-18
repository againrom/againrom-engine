package ui

import (
	"image"

	"againrom/pkg/render/text"
)

const dropdownRim = 2

func (d dropdown) fitList(bounds image.Rectangle, f *text.Font, width, limit int) dropdown {
	room := bounds.Inset(dropdownRim)
	if !d.Open || d.List == nil || d.List.Len() == 0 || room.Dx() <= widgetSpriteSize {
		d.Open = false
		return d
	}
	measure := editorMeasure(f)
	pitch := listRowPitch(f)
	for _, row := range d.List.Rows() {
		need := measure(row.Text) + 2*listTextX + widgetSpriteSize + listRowLead(row, pitch, listTextX)
		width = max(width, need)
	}
	wrap := width > room.Dx()
	width = min(room.Dx(), max(d.Rect.Dx(), width))
	x := min(max(room.Min.X, d.Rect.Min.X), room.Max.X-width)
	below := max(room.Min.Y, d.Rect.Max.Y+dropdownRim)
	above := min(room.Max.Y, d.Rect.Min.Y-dropdownRim)
	belowRoom := room.Max.Y - below - listBottomExtra
	aboveRoom := above - room.Min.Y - listBottomExtra
	available := max(belowRoom, aboveRoom)
	if d.allRows {
		available = room.Dy() - listBottomExtra
	}
	if wrap && f != nil {
		textWidth := width - widgetSpriteSize - 2*listTextX
		lines := 1
		for _, row := range d.List.Rows() {
			room := textWidth - listRowLead(row, pitch, listTextX)
			lines = max(lines, len(editorTextLines(f, row.Text, room)))
		}
		maxExtra := max(0, (available-pitch)/max(1, f.Height()))
		pitch += min(lines-1, maxExtra) * max(1, f.Height())
	}
	rows := min(d.List.Len(), limit)
	belowRows := max(0, belowRoom/pitch)
	aboveRows := max(0, aboveRoom/pitch)
	top := below
	switch {
	case rows <= belowRows:
	case rows <= aboveRows:
		top = above - rows*pitch - listBottomExtra
	case d.allRows:
		rows = min(rows, max(0, available/pitch))
		top = room.Max.Y - rows*pitch - listBottomExtra
	case aboveRows > belowRows:
		rows = aboveRows
		top = above - rows*pitch - listBottomExtra
	default:
		rows = min(rows, belowRows)
	}
	if rows == 0 {
		d.Open = false
		return d
	}
	l := listAtPitch(image.Rect(x, top, x+width-widgetSpriteSize, top), rows, pitch)
	if wrap {
		l.wrapPitch = listRowPitch(f)
	}
	if d.List.visibleRows() != rows {
		d.List.SetRowLimit(max(d.List.rowLimit(), rows)).SetWindow(rows)
	}
	d.Rows, d.listLayout, d.listFont = rows, &l, f
	return d
}
