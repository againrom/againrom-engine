package ui

import (
	"image"
	"testing"
)

// TestHudColumnSlotsSumToThePanelsOwnTop pins the invariant hud.go's own
// comments assert but do not compute: hudMinimapReserve and hudToggleBarH
// are two independent decoded readings from SHOP-FIGURE-041 (ids 5 and 6),
// and hudPanelTopY is a third, id 7's own top edge. Nothing in this build
// derives hudPanelTopY from the other two; this test is what would fail if
// a future edit moved one constant and not the others out of step with the
// decode.
func TestHudColumnSlotsSumToThePanelsOwnTop(t *testing.T) {
	if got := hudMinimapReserve + hudToggleBarH; got != hudPanelTopY {
		t.Fatalf("hudMinimapReserve(%d) + hudToggleBarH(%d) = %d, want hudPanelTopY(%d)",
			hudMinimapReserve, hudToggleBarH, got, hudPanelTopY)
	}
}

func TestMissionPackGridUsesDecodedRectOriginsAndEndStrips(t *testing.T) {
	tests := []struct {
		name        string
		area        image.Point
		bar         image.Rectangle
		cols        int
		first, last image.Rectangle
		back, fwd   image.Rectangle
	}{
		{
			name: "original 640x480 frame", area: image.Pt(640, 480),
			bar: image.Rect(0, 390, 480, 480), cols: 5,
			first: image.Rect(32, 396, 112, 476), last: image.Rect(352, 396, 432, 476),
			back: image.Rect(0, 390, 32, 480), fwd: image.Rect(432, 390, 480, 480),
		},
		{
			name: "shipped 1024x768 mission frame", area: image.Pt(1024, 768),
			bar: image.Rect(0, 678, 864, 768), cols: 9,
			first: image.Rect(64, 684, 144, 764), last: image.Rect(704, 684, 784, 764),
			back: image.Rect(0, 678, 64, 768), fwd: image.Rect(784, 678, 864, 768),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bar, cols, ok := packBarRect(tc.area)
			if !ok || bar != tc.bar || cols != tc.cols {
				t.Fatalf("packBarRect(%v) = (%v,%d,%v), want (%v,%d,true)", tc.area, bar, cols, ok, tc.bar, tc.cols)
			}
			cells := packCellRects(bar, cols)
			if len(cells) != tc.cols || cells[0] != tc.first || cells[len(cells)-1] != tc.last {
				t.Fatalf("cell run = first %v last %v count %d, want first %v last %v count %d",
					cells[0], cells[len(cells)-1], len(cells), tc.first, tc.last, tc.cols)
			}
			for i := 1; i < len(cells); i++ {
				if cells[i].Min.X-cells[i-1].Min.X != 80 || cells[i].Dx() != 80 || cells[i].Dy() != 80 {
					t.Errorf("cell %d = %v after %v; want contiguous 80x80 cells", i, cells[i], cells[i-1])
				}
			}
			back, fwd := packScrollRects(bar, cols)
			if back != tc.back || fwd != tc.fwd {
				t.Errorf("scroll strips = (%v,%v), want (%v,%v)", back, fwd, tc.back, tc.fwd)
			}
		})
	}
}
