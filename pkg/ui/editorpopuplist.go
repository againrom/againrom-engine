package ui

import (
	"image"
	"slices"
	"strings"

	"againrom/pkg/render/text"
	"againrom/pkg/render/textfit"
)

// Cell is the text box of a visible list cell, excluding the column gap.
func (l listBox) Cell(row, column int) image.Rectangle {
	if row < 0 || row >= l.Rows || column < 0 || column >= len(l.Columns) {
		return image.Rectangle{}
	}
	r := l.Row(row)
	x := r.Min.X + listTextX
	stops := columnStops(l.Columns, r.Dx()-2*listTextX)
	return image.Rect(x+stops[column], r.Min.Y, x+stops[column+1]-listColumnGap, r.Max.Y)
}

// CellAt is the visible row under p and the column whose cell holds p, -1
// in a gap between cells or in a list without columns.
func (l listBox) CellAt(p image.Point) (row, column int, ok bool) {
	if !p.In(l.Rect) {
		return 0, -1, false
	}
	row, column = (p.Y-l.Rect.Min.Y)/l.Pitch, -1
	for k := range l.Columns {
		if p.In(l.Cell(row, k)) {
			column = k
		}
	}
	return row, column, row < l.Rows
}

// fitCells is row text full with each cell fitted by fit to its column of
// a row width wide.
func (l listBox) fitCells(fit textfit.Spec, full string, width int) string {
	stops := columnStops(l.Columns, width)
	cells := strings.SplitN(full, "\t", len(l.Columns))
	for k := range cells {
		cells[k] = fit.Fit(cells[k], max(0, stops[k+1]-stops[k]-listColumnGap))
	}
	return strings.Join(cells, "\t")
}

// listHeadFont is the heading face for column measurement, wrapping and paint.
func listHeadFont(title, body *text.Font) *text.Font {
	if title != nil {
		return title
	}
	return body
}

// listColumnNeeds are the widths list's columns ask: each the wider of its
// head in the heading face and its widest cell in the body face, every row
// counted, plus the column gap.
func listColumnNeeds(list uiList, heading, body *text.Font) []int {
	measure := func(face *text.Font, s string) int {
		if face != nil {
			return face.Advance(s)
		}
		return len([]rune(s)) * 8
	}
	need := make([]int, len(list.Columns))
	for k, col := range list.Columns {
		need[k] = measure(heading, col.Head)
		if heading != nil {
			// A head is drawn whole, so its last glyph's overhang counts.
			ink, _ := heading.Measure(col.Head)
			need[k] = max(need[k], ink)
		}
	}
	if list.Picker != nil {
		for _, row := range list.Picker.Rows() {
			for k, cell := range strings.SplitN(row.Text, "\t", len(list.Columns)) {
				need[k] = max(need[k], measure(body, cell))
			}
		}
	}
	for k := range need {
		need[k] += listColumnGap
	}
	return need
}

// listColumnWidths parts a row width among list's columns. When every need
// fits, each column takes its need and the room left is parted by the
// shares. Otherwise the columns whose need fits keep it and the columns that
// are cut share the room those leave equally.
func listColumnWidths(list uiList, heading, body *text.Font, width int) []int {
	need := listColumnNeeds(list, heading, body)
	out := make([]int, len(need))
	total := 0
	for _, n := range need {
		total += n
	}
	if total <= width {
		spare := columnStops(listShares(list.Columns), width-total)
		for k := range need {
			out[k] = need[k] + spare[k+1] - spare[k]
		}
		return out
	}
	order := make([]int, len(need))
	for k := range order {
		order[k] = k
	}
	slices.SortStableFunc(order, func(a, b int) int { return need[a] - need[b] })
	left := max(0, width)
	for i, k := range order {
		rest := len(order) - i
		if need[k]*rest <= left {
			out[k], left = need[k], left-need[k]
			continue
		}
		for j, c := range order[i:] {
			out[c] = left / rest
			if j < left%rest {
				out[c]++
			}
		}
		break
	}
	return out
}

// listCellTip reads a clipped cell's full model text through the one tip.
func listCellTip(bounds image.Rectangle, body *text.Font, list uiList, box listBox, p image.Point) (popupTip, bool) {
	row, k, ok := box.CellAt(p)
	if !ok || k < 0 || list.Picker == nil {
		return popupTip{}, false
	}
	top, shown := list.Picker.Visible()
	if row >= shown {
		return popupTip{}, false
	}
	cells := strings.SplitN(list.Picker.Rows()[top+row].Text, "\t", len(box.Columns))
	if k >= len(cells) {
		return popupTip{}, false
	}
	r := box.Cell(row, k)
	if body.Advance(cells[k]) <= r.Dx() {
		return popupTip{}, false
	}
	return layoutTip(bounds, body, r, cells[k], nil, tipShape{Width: min(popupTipW, bounds.Dx()), Pull: true, Clip: true, Scroll: true}), true
}
