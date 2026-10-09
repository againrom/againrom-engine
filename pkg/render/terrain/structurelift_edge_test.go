package terrain_test

import (
	"testing"

	"againrom/pkg/render/terrain"
)

func TestStructureLiftUsesSignedMidpointInterpolation(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height int
		corners       [4]int
		want          int
	}{
		{"odd both descending positive", 3, 3, [4]int{1, 0, 0, 0}, 1},
		{"odd both ascending positive", 3, 3, [4]int{0, 1, 0, 0}, 0},
		{"odd both ascending negative", 3, 3, [4]int{-1, 0, 0, 0}, -1},
		{"odd both descending negative", 3, 3, [4]int{0, -1, 0, 0}, 0},
		{"odd width descending", 3, 2, [4]int{1, 0, 0, 0}, 1},
		{"odd height descending", 2, 3, [4]int{1, 0, 0, 0}, 1},
		{"even both exact vertex", 2, 2, [4]int{1, 0, 0, 0}, 1},
		{"castle centre", 3, 3, [4]int{28, 28, 28, 44}, 32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			class := &terrain.StructureClass{TileWidth: tc.width, TileHeight: tc.height, FullHeight: tc.height + 2}
			class.Frames = make([]*terrain.StaticFrame, tc.width*class.FullHeight)
			for i := range class.Frames {
				class.Frames[i] = &terrain.StaticFrame{Width: 32, Height: 32}
			}
			set := new(terrain.StructureSet)
			set.Classes[1] = class
			g := terrain.Grid{Width: 16, Height: 16, Structures: []terrain.StructureRecord{{X: 2 << 8, Y: 3 << 8, Key: 1}}}
			centreX, centreY := 2+tc.width/2, 3+tc.height/2
			corner := func(x, y int) int {
				if x < centreX || x > centreX+1 || y < centreY || y > centreY+1 {
					t.Fatalf("unexpected corner sample (%d,%d)", x, y)
				}
				return tc.corners[(y-centreY)*2+x-centreX]
			}
			flat, _, _ := terrain.StructurePlacements(g, set, nil, 0)
			lifted, _, _ := terrain.StructurePlacements(g, set, corner, 0)
			if len(flat) == 0 || len(flat) != len(lifted) {
				t.Fatalf("flat/lifted entries = %d/%d", len(flat), len(lifted))
			}
			for i := range flat {
				if flat[i].Cell != lifted[i].Cell || flat[i].GridIndex != lifted[i].GridIndex || flat[i].TopLeft.X != lifted[i].TopLeft.X {
					t.Fatalf("lift changed entry %d identity or X", i)
				}
				if got := flat[i].TopLeft.Y - lifted[i].TopLeft.Y; got != tc.want {
					t.Fatalf("entry %d lift = %d, want %d", i, got, tc.want)
				}
			}
		})
	}
}
