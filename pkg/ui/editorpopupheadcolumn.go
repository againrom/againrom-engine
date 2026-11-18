package ui

import (
	"image"

	"againrom/pkg/render/text"
)

// headColumnGap is the gap between two controls of a matrix row's head
// column.
const headColumnGap = 4

// placeHeadColumn lays out a matrix's head column: the controls each matrix
// row holds (the rows with HeadOf set, row i in column cols[i]) stand side
// by side in the row's head box. A column is as wide as its widest
// control's natural width, so every caption and value shows whole and the
// columns line up from row to row; the grid's head column takes their sum.
// It writes each such row's Place and the grid's HeadW.
func placeHeadColumn(g *popupGrid, rows []popupRow, cols []int, body *text.Font) {
	widths := map[int]int{}
	n := 0
	for i, r := range rows {
		if r.HeadOf > 0 {
			widths[cols[i]] = max(widths[cols[i]], headControlWidth(r, body, g.side()))
			n = max(n, cols[i]+1)
		}
	}
	at := make([]int, n+1)
	for k := range n {
		at[k+1] = at[k]
		if widths[k] > 0 {
			at[k+1] += widths[k] + headColumnGap
		}
	}
	for i, r := range rows {
		if r.HeadOf > 0 {
			k := cols[i]
			rows[i].Place = image.Rect(at[k], 2, at[k]+widths[k], g.side()-2)
		}
	}
	g.HeadW = at[n]
}

// headControlWidth is a head column control's natural width: what its
// caption or value needs inside the room the painter gives it.
func headControlWidth(r popupRow, body *text.Font, side int) int {
	advance := func(s string) int {
		if body == nil {
			return 8 * len(s)
		}
		return body.Advance(s)
	}
	switch {
	case len(r.Buttons) > 0:
		return max(side-4, advance(r.Buttons[0])+12)
	case r.Kind == popupSwitch:
		return advance(r.Value) + 16
	case r.Kind == popupEdit:
		return max(popupMinField, advance(r.Value)+2*editTextX+8)
	}
	return advance(r.Value) + 8
}
