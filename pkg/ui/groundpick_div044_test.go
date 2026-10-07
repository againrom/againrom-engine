package ui

import (
	"image"
	"math"
	"testing"

	"againrom/pkg/render/terrain"
)

// groundPickSlopeGrid is a 2x2 synthetic corner mesh. Its altitude rows are
// identical and rise 64 pixels to the right. With MinV == -64, cell (0,0)'s
// literal bounds are:
//
//	x=0:  top 64, bottom 96
//	x=1:  top 62, bottom 94
//	x=31: top 2,  bottom 34
//
// Cell (0,1) begins on the same bottom edge. The old mean-of-four placement
// rectangles are row 0 at y 32..64 and row 1 at y 64..96. Point (1,80)
// therefore resolves to row 0 by the corner mesh and row 1 by the old picker.
func groundPickSlopeGrid() terrain.Grid {
	return altGrid(2, 2,
		0, 64,
		0, 64,
	)
}

func groundPickSlopeViewer(t *testing.T) *Viewer {
	t.Helper()
	v := identityViewer(t, groundPickSlopeGrid(), 64, 128)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("fixture mode = %v, want displaced", v.Mode())
	}
	v.commandMode = true
	return v
}

func TestGroundCellAtUsesThePerColumnCornerMesh(t *testing.T) {
	v := groundPickSlopeViewer(t)
	tests := []struct {
		name       string
		x, y       float64
		wantCol    int
		wantRow    int
		wantInside bool
	}{
		{"top edge is inclusive", 1, 62, 0, 0, true},
		{"above the top edge is outside", 1, 61, 0, 0, false},
		{"old mean selects the next row", 1, 80, 0, 0, true},
		{"shared horizontal edge belongs to the first row", 1, 94, 0, 0, true},
		{"one row below the seam reaches the second row", 1, 95, 0, 1, true},
		{"bottom map edge is inclusive", 1, 126, 0, 1, true},
		{"below the mesh but inside the viewport has no cell", 1, 127, 0, 0, false},
		{"left corner belongs to the first row", 0, 64, 0, 0, true},
		{"near-right top corner", 31, 2, 0, 0, true},
		{"near-right shared edge", 31, 34, 0, 0, true},
		{"vertical seam belongs to the right column", 32, 16, 1, 0, true},
		{"last-column horizontal seam belongs to the first row", 63, 32, 1, 0, true},
		{"last cell bottom edge is inclusive", 63, 64, 1, 1, true},
		{"left viewport edge is exclusive outside", -1, 16, 0, 0, false},
		{"right viewport edge is exclusive outside", 64, 16, 0, 0, false},
		{"top viewport edge is exclusive outside", 32, -1, 0, 0, false},
		{"NaN x is outside", math.NaN(), 16, 0, 0, false},
		{"NaN y is outside", 32, math.NaN(), 0, 0, false},
		{"positive infinity is outside", math.Inf(1), 16, 0, 0, false},
		{"negative infinity is outside", 32, math.Inf(-1), 0, 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			col, row, inside := v.groundCellAt(tc.x, tc.y)
			if inside != tc.wantInside || col != tc.wantCol || row != tc.wantRow {
				t.Errorf("groundCellAt(%v,%v) = (%d,%d,%v), want (%d,%d,%v)",
					tc.x, tc.y, col, row, inside, tc.wantCol, tc.wantRow, tc.wantInside)
			}
		})
	}
}

func TestGroundCellAtKeepsTheNoAltitudeFlatFallback(t *testing.T) {
	v := identityViewer(t, grid(2, 2), 64, 64)
	if v.Mode() != ModeFlat {
		t.Fatalf("fixture mode = %v, want flat", v.Mode())
	}
	for _, tc := range []struct {
		x, y     float64
		col, row int
		inside   bool
	}{
		{16, 16, 0, 0, true},
		{16, 32, 0, 1, true}, // Camera.ScreenToCell's half-open flat seam
		{63, 63, 1, 1, true},
		{64, 63, 0, 0, false},
	} {
		col, row, inside := v.groundCellAt(tc.x, tc.y)
		if col != tc.col || row != tc.row || inside != tc.inside {
			t.Errorf("groundCellAt(%v,%v) = (%d,%d,%v), want (%d,%d,%v)",
				tc.x, tc.y, col, row, inside, tc.col, tc.row, tc.inside)
		}
	}
}

func TestGroundCellAtRejectsAnInvertedColumn(t *testing.T) {
	v := identityViewer(t, pushedGrid(), pushedW*terrain.CellSize, pushedCanvasH)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("fixture mode = %v, want displaced", v.Mode())
	}

	// Row 0 has top 96 and bottom 0 throughout. Its old mean placement rectangle
	// was y 32..64, so (1,63) is also a direct old-picker counterexample.
	col, row, inside := v.groundCellAt(1, 63)
	if inside || col != 0 || row != 0 {
		t.Errorf("inverted column pick = (%d,%d,%v), want (0,0,false)", col, row, inside)
	}
	if col, row, inside := v.groundCellAt(1, 16); !inside || col != 0 || row != 1 {
		t.Errorf("non-inverted row below it = (%d,%d,%v), want (0,1,true)", col, row, inside)
	}
}

func TestGroundCellAtUsesCameraPanZoomAndFractionalOffsets(t *testing.T) {
	v := newViewer(t, groundPickSlopeGrid())
	layoutViewport(v, 32, 64)
	cam := v.Camera()
	cam.SetZoom(2)
	cam.X = 0.25
	cam.Y = 63.75
	cam.Clamp()
	if cam.X != 0.25 || cam.Y != 63.75 || cam.Zoom != 2 {
		t.Fatalf("camera = (%v,%v) zoom %v, want (0.25,63.75) zoom 2", cam.X, cam.Y, cam.Zoom)
	}

	// Screen (2,33) maps to world (1.25,80.25). Native column floor(1.25)
	// is 1, whose literal row-0 bounds are 62..94. The old mean picker names
	// row 1 at this point.
	col, row, inside := v.groundCellAt(2, 33)
	if !inside || col != 0 || row != 0 {
		t.Errorf("panned and zoomed pick = (%d,%d,%v), want (0,0,true)", col, row, inside)
	}
}

func TestGroundPickerFeedsHoverCommandAndDropWithOneCell(t *testing.T) {
	const x, y = 1, 80

	t.Run("hover and cursor", func(t *testing.T) {
		v := groundPickSlopeViewer(t)
		// The mesh-selected row 0 is visible and the old mean-selected row 1 is
		// unseen. This makes fogGateSack itself load-bearing rather than letting
		// an all-visible plane witness only the sack lookup behind the gate.
		v.SetFog([]byte{FogVisible, FogVisible, FogUnseen, FogVisible}, 2, 2)
		v.SetSacks([]MapSack{{Cell: image.Pt(0, 0)}})
		v.SetEntities([]MapEntity{{ID: 7, Cell: image.Pt(1, 1), PlayerCharacter: true}})
		v.sel = selection{7}

		mask, _, hit := v.hoverMask(x, y)
		if mask != hoverMaskDrawable || hit {
			t.Errorf("hover at (%d,%d) = (mask %#x, hit %v), want sack-only mask %#x",
				x, y, mask, hit, hoverMaskDrawable)
		}
		if got := v.gestureCursorAt(x, y); got != "pickup" {
			t.Errorf("cursor at (%d,%d) = %q, want pickup", x, y, got)
		}
	})

	t.Run("click command", func(t *testing.T) {
		v := groundPickSlopeViewer(t)
		v.SetFog([]byte{FogVisible, FogVisible, FogVisible, FogVisible}, 2, 2)
		v.SetEntities([]MapEntity{{ID: 7, Cell: image.Pt(1, 1), PlayerCharacter: true}})
		v.sel = selection{7}

		orders, ok := tapAt(v, x, y)
		want := order{kind: orderKindMove, entity: 7, x: 0, y: 0}
		if !ok || len(orders) != 1 || orders[0] != want {
			t.Errorf("tap at (%d,%d) = (%+v,%v), want ([%+v],true)", x, y, orders, ok, want)
		}
	})

	t.Run("ground drop", func(t *testing.T) {
		v := groundPickSlopeViewer(t)
		col, row := v.dropCellAt(x, y)
		if col != 0 || row != 0 {
			t.Errorf("dropCellAt(%d,%d) = (%d,%d), want (0,0)", x, y, col, row)
		}
	})
}
