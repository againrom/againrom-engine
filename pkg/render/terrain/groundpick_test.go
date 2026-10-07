package terrain

import "testing"

// These expected values are literal evaluations of TERR-GEOM-036(d)'s picker
// expression. They do not call an interpolation helper to build an oracle.
// The synthetic corners for cell (0,0), after projection, are:
//
//	TL (0,0)   TR (32,31)
//	BL (0,32)  BR (32,1)
//
// The two 31-row deltas have opposite signs. At native column 1 each quotient
// has magnitude 31/32 and must truncate to zero, not floor to -1 on the bottom
// edge. At column 17 the two bounds collapse to row 16. Past it they invert.
func TestCellColumnBoundsMasksTheColumnAndTruncatesTowardZero(t *testing.T) {
	p := Project([]uint8{
		0, 0xe1, // int8(-31)
		0, 31,
	}, 2, 2)
	if p.MinV != 0 {
		t.Fatalf("fixture MinV = %d, want 0; the literal bounds below would be wrong", p.MinV)
	}

	tests := []struct {
		nativeX    int
		wantTop    int
		wantBottom int
	}{
		{-1, 30, 2}, // -1 & 31 == 31
		{0, 0, 32},
		{1, 0, 32}, // -31/32 truncates to 0
		{16, 15, 17},
		{17, 16, 16}, // one-coordinate collapsed span
		{31, 30, 2},
		{32, 0, 32}, // 32 & 31 == 0
		{33, 0, 32},
	}
	for _, tc := range tests {
		top, bottom := p.CellColumnBounds(0, 0, tc.nativeX)
		if top != tc.wantTop || bottom != tc.wantBottom {
			t.Errorf("native x %d: bounds = (%d,%d), want (%d,%d)",
				tc.nativeX, top, bottom, tc.wantTop, tc.wantBottom)
		}
	}
}

// The four cases cover flat, rising, falling, and crossed corner pairs. Every
// expected bound is worked out from the stated corner rows and native column;
// none is derived from CellColumnBounds or from another production function.
func TestCellColumnBoundsCoversBothEdgeOrientations(t *testing.T) {
	tests := []struct {
		name       string
		altitudes  []uint8
		nativeX    int
		wantTop    int
		wantBottom int
	}{
		{
			name:       "flat",
			altitudes:  []uint8{7, 7, 7, 7},
			nativeX:    16,
			wantTop:    0,
			wantBottom: 32,
		},
		{
			name:       "both edges rise toward the right",
			altitudes:  []uint8{0, 0xe1, 0, 0xe1},
			nativeX:    16,
			wantTop:    15,
			wantBottom: 47,
		},
		{
			name:       "both edges fall toward the right",
			altitudes:  []uint8{0xe1, 0, 0xe1, 0},
			nativeX:    16,
			wantTop:    16,
			wantBottom: 48,
		},
		{
			name:       "edges cross",
			altitudes:  []uint8{0, 0xe1, 0, 31},
			nativeX:    31,
			wantTop:    30,
			wantBottom: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Project(tc.altitudes, 2, 2)
			top, bottom := p.CellColumnBounds(0, 0, tc.nativeX)
			if top != tc.wantTop || bottom != tc.wantBottom {
				t.Errorf("bounds = (%d,%d), want (%d,%d)",
					top, bottom, tc.wantTop, tc.wantBottom)
			}
		})
	}
}

// The last cell's right corners use Projection's defined far-edge clamp. The
// grid below gives cell (1,0) the literal top row 31 and bottom row 1 at both
// ends, so its whole width is inverted. This pins the map-edge producer used by
// the ground picker without constructing an out-of-grid corner in the test.
func TestCellColumnBoundsUsesTheProjectionFarEdge(t *testing.T) {
	p := Project([]uint8{
		0, 0xe1, // int8(-31)
		0, 31,
	}, 2, 2)
	for _, nativeX := range []int{32, 33, 48, 63} {
		top, bottom := p.CellColumnBounds(1, 0, nativeX)
		if top != 31 || bottom != 1 {
			t.Errorf("native x %d: last-cell bounds = (%d,%d), want (31,1)",
				nativeX, top, bottom)
		}
	}
}
