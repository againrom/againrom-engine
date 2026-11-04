package mapedit

import (
	"image"
	"testing"
)

func TestRectangleCoverDropsUnassignablePatches(t *testing.T) {
	for _, tc := range []struct {
		rows []string
		want image.Rectangle
	}{
		{[]string{"###.", "####", ".###"}, image.Rect(0, 0, 3, 2)},
		{[]string{"######", "######", "...###"}, image.Rect(0, 0, 6, 2)},
	} {
		w, h := len(tc.rows[0]), len(tc.rows)
		cells := make([]bool, w*h)
		for y, row := range tc.rows {
			for x := range row {
				cells[y*w+x] = row[x] == '#'
			}
		}
		rects := rectangleCover(cells, w, h, image.Pt(3, 2), 255)
		if len(rects) != 1 || rects[0] != tc.want {
			t.Fatalf("patches %v stored %v, want only %v", tc.rows, rects, tc.want)
		}
	}
}

func TestRectangleCoverKeepsLongBandsDisjoint(t *testing.T) {
	for _, size := range []image.Point{{256, 2}, {257, 2}, {511, 2}, {3, 256}, {3, 511}} {
		cells := make([]bool, size.X*size.Y)
		for i := range cells {
			cells[i] = true
		}
		for _, r := range rectangleCover(cells, size.X, size.Y, image.Pt(3, 2), 255) {
			if r.Dx() < 3 || r.Dy() < 2 || r.Dx() > 255 || r.Dy() > 255 || !r.In(image.Rectangle{Max: size}) {
				t.Fatalf("band %v stored invalid rectangle %v", size, r)
			}
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					i := y*size.X + x
					if !cells[i] {
						t.Fatalf("band %v overlaps at %d,%d", size, x, y)
					}
					cells[i] = false
				}
			}
		}
		for i, missed := range cells {
			if missed {
				t.Errorf("band %v lost cell %d,%d", size, i%size.X, i/size.X)
			}
		}
	}
}
