package camera

import (
	"math"
	"testing"
)

func TestZoomStepAboutMovesToTheNextGridValueNeverPast(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start float64
		dir   int
		want  float64
	}{
		{"in from native", 1, +1, 1.05},
		{"out from native", 1, -1, 0.95},
		{"in from 93 percent", 0.93, +1, 0.95},
		{"out from 93 percent", 0.93, -1, 0.90},
		{"in from 95 percent lands on 100", 0.95, +1, 1.00},
		{"in from 195 lands on 200", 1.95, +1, 2.00},
		{"in from 200 goes by 25", 2.00, +1, 2.25},
		{"out from 225 lands on 200", 2.25, -1, 2.00},
		{"out from 200 goes by 5", 2.00, -1, 1.95},
		{"in from 210 lands on 225", 2.10, +1, 2.25},
		{"out from 210 lands on 200", 2.10, -1, 2.00},
		{"in from 775 lands on 800", 7.75, +1, 8.00},
		{"in at ZoomMax stays", ZoomMax, +1, ZoomMax},
		{"out at the floor stays", ZoomMin, -1, ZoomMin},
		{"out from 12.5 percent floor stops at floor", 0.125, -1, 0.125},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := New(256, 256, 320, 480)
			c.SetZoom(tc.start)
			c.ZoomStepAbout(160, 120, 5, 25, 200, tc.dir)
			if math.Abs(c.Zoom-tc.want) > 1e-12 {
				t.Errorf("Zoom = %v, want %v", c.Zoom, tc.want)
			}
		})
	}
}

func TestZoomStepAboutAtALimitMovesNothing(t *testing.T) {
	c := New(256, 256, 320, 480)
	c.SetZoom(ZoomMax)
	c.Pan(40, 25)
	x, y := c.X, c.Y
	c.ZoomStepAbout(10, 10, 5, 25, 200, +1)
	if c.X != x || c.Y != y {
		t.Errorf("origin moved at ZoomMax: (%v,%v) -> (%v,%v)", x, y, c.X, c.Y)
	}
}
