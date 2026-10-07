package game

import (
	"image"
	"testing"
)

func TestTopRowHiddenOnlyWhenImpassableAndEmpty(t *testing.T) {
	const w, h = 4, 3
	plane := func(row []byte) []byte {
		p := make([]byte, w*h)
		for i := range p {
			p[i] = 2
		}
		copy(p[w:], row)
		return p
	}
	blocked := []byte{1, 1, 1, 1}
	for _, tc := range []struct {
		name  string
		row   []byte
		cells []image.Point
		want  bool
	}{
		{"all blocked and empty", blocked, nil, true},
		{"one walkable cell", []byte{1, 0, 1, 1}, nil, false},
		{"a unit stands in the row", blocked, []image.Point{{2, 1}}, false},
		{"a sack lies in the row", blocked, []image.Point{{0, 1}}, false},
		{"an occupant in another row", blocked, []image.Point{{2, 2}}, true},
	} {
		if got := topRowHidden(plane(tc.row), w, h, tc.cells); got != tc.want {
			t.Errorf("%s: hidden=%v, want %v", tc.name, got, tc.want)
		}
	}
}
