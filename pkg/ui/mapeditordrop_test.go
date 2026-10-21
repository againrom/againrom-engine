package ui

import (
	"image"
	"math"
	"testing"
)

// The drop tool's square follows MISSION-SCATTER-053: radius
// trunc(max(5, sqrt(n)+4)) as its low byte, each axis offset -r/2 plus a
// draw of 0..r.
func TestDropScatterFollowsTheClaim(t *testing.T) {
	for n := 0; n <= 70000; n += 1 + n/7 {
		want := int(uint8(int(math.Max(5, math.Sqrt(float64(n))+4))))
		if got := dropScatterRadius(n); got != want {
			t.Fatalf("n %d: radius %d, want %d", n, got, want)
		}
	}
	for _, c := range []struct{ r, lo, hi int }{{5, -2, 3}, {6, -3, 3}, {8, -4, 4}} {
		got := dropScatter(image.Pt(20, 20), c.r)
		if want := image.Rect(20+c.lo, 20+c.lo, 20+c.hi+1, 20+c.hi+1); got != want {
			t.Errorf("radius %d: square %v, want %v", c.r, got, want)
		}
	}
}
