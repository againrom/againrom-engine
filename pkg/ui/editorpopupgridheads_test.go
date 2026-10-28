package ui

import (
	"crypto/sha256"
	"fmt"
	"image"
	"slices"
	"testing"
)

// A grid whose columns carry no heads, and one whose columns keep one-row
// heads, lay out and paint exactly as they did before a grid could head its
// columns upright: the sums below were taken from the painter and layout
// that had no upright heads.
func TestPopupGridWithoutUprightHeadsLaysOutAsBefore(t *testing.T) {
	f := panelFont()
	plain := &popupGrid{Rows: 2}
	for i := range 7 {
		cell := popupGridCell{Label: fmt.Sprint(i), Pressed: i == 2, Ticked: i == 3}
		if i == 1 {
			cell.Label, cell.Icon = "", relationIcon(1)
		}
		plain.Cells = append(plain.Cells, cell)
	}
	headed := &popupGrid{Side: relationSide, Rows: 2, Cols: 2, HeadW: 60, RowHeads: []string{"1: Self", "2: Enemy"}, ColHeads: []string{"1", "2"},
		Cells: []popupGridCell{{Inert: true}, {Icon: relationIcon(2)}, {Label: "5"}, {Inert: true}}}
	for _, tc := range []struct {
		name         string
		grid         *popupGrid
		layout, draw string
	}{
		{"no heads", plain, "c56458bd723bd3226695ddf2bcfe0e7b8c6368b77adf16d9680b9f9eec4af508", "8bd960cd2e0d0635cf3ac614ea458f3bb6d5ba3e69bf0927e19ef4a36e323aa9"},
		{"one-row heads", headed, "18127024a897041d0db8287e14e74248457b0e56074a3f8d2cfb8d130c474d30", "9fffab8f7b513527385a673ab544932616a32eeed91634e7f95b83876baa3b74"},
	} {
		s := popupSpec{Title: "Grid", At: image.Pt(20, 40), Width: 300, Bounds: image.Rect(0, 0, 1024, 768),
			Rows:    []popupRow{{Kind: popupHeading, Label: "Head"}, {Kind: popupText, Label: "Name", Value: "Short"}},
			Buttons: []string{"OK"}, Grid: tc.grid}
		l := layoutPopup(s, f, f)
		geometry := fmt.Sprint(tc.grid.height(), l.Rect, l.Grid, l.GridBar, l.GridCells, l.GridColHeads, l.GridRowHeads, l.GridInert, l.Buttons, l.Notes)
		pic := image.NewRGBA(image.Rect(0, 0, 400, 400))
		drawPopup(pic, f, f, nil, s, l, editorPopupInk, image.Pt(-1, -1), false)
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(geometry))); got != tc.layout {
			t.Errorf("%s: layout %s, sum %s, want %s", tc.name, geometry, got, tc.layout)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(pic.Pix)); got != tc.draw {
			t.Errorf("%s: picture sum %s, want %s", tc.name, got, tc.draw)
		}
		// Neither grid had a hover line over a cell or a head, and has none.
		for _, r := range slices.Concat(l.GridCells, l.GridColHeads, l.GridRowHeads) {
			if tip, ok := popupTipAt(image.Rect(0, 0, 1024, 768), f, f, s, l, center(r), true); ok {
				t.Errorf("%s: a hover line %q over %v", tc.name, tip.Lines, r)
			}
		}
	}
}
