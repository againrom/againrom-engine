package sim

import "testing"

// TestProjectileDirectionSplitsAtTheClaimedSlopes pins ANIM-138: in each
// quadrant the four boundaries 1/4, 3/4, 4/3 and 4 and the vector just past
// each, the axes, the zero vector and a vector whose products wrap.
func TestProjectileDirectionSplitsAtTheClaimedSlopes(t *testing.T) {
	t.Parallel()

	// (a, b) magnitudes; q is the quadrant step the claim's comparisons give.
	steps := []struct{ a, b, q int32 }{
		{4, 1, 0}, {3, 1, 1}, // slope 1/4 takes the lower q
		{4, 3, 1}, {7, 6, 2}, // slope 3/4 takes the lower q
		{3, 4, 3}, {6, 7, 2}, // slope 4/3 takes q 3
		{1, 4, 4}, {2, 7, 3}, // slope 4 takes q 4
	}
	quadrants := []struct {
		name   string
		sx, sy int32
		want   [5]int32
	}{
		{"south-east", 1, 1, [5]int32{4, 5, 6, 7, 8}},
		{"south-west", -1, 1, [5]int32{12, 11, 10, 9, 8}},
		{"north-west", -1, -1, [5]int32{12, 13, 14, 15, 0}},
		{"north-east", 1, -1, [5]int32{4, 3, 2, 1, 0}},
	}
	for _, quad := range quadrants {
		for _, st := range steps {
			dx, dy := quad.sx*st.a, quad.sy*st.b
			if got := ProjectileDirection(dx, dy); got != quad.want[st.q] {
				t.Errorf("%s (%d,%d) = %d, want %d", quad.name, dx, dy, got, quad.want[st.q])
			}
		}
	}
	for _, tc := range []struct{ dx, dy, want int32 }{
		{0, -1, 0}, {1, 0, 4}, {0, 1, 8}, {-1, 0, 12},
		{0, 0, 4},
		// 4*b wraps negative at 2^31, so a >= 4*b holds: q 0.
		{1 << 30, 1 << 29, 4},
	} {
		if got := ProjectileDirection(tc.dx, tc.dy); got != tc.want {
			t.Errorf("ProjectileDirection(%d, %d) = %d, want %d", tc.dx, tc.dy, got, tc.want)
		}
	}
}
