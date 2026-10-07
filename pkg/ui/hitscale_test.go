package ui

// The mission screen's hit tests at scales other than 1:1 (contract B4).
//
// WHY THIS FILE EXISTS. A hit test left in window space is then correct in
// every test and wrong on the owner's screen at any other scale. This file
// runs the shipped hit tests at 2:1, 1:2 and at two windows whose aspect is
// not the frame's, so the letterbox is non-zero on one axis and then on the
// other.
//
// THE WINDOW POSITIONS ARE COMPUTED HERE, not read back from frame.Placement.
// windowAt writes out origin + scale * (frame pixel + 1/2) from the scale and
// origin the case names, and those two numbers are written out by hand from the
// window size and the frame's own 1024x768. Nothing below asks the mapping under
// test where a frame pixel went.
//
// Every fixture is synthetic, no window opens and nothing reads a game install.

import (
	"image"
	"testing"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

// The fixture's world is 64x64 cells, 2048 pixels square, so it is larger than
// the map view on both axes and the camera sits where it is put rather than
// where a clamp centres it. With the camera at the origin a frame pixel is a
// world pixel, which is what makes every cell below readable by inspection.
const (
	hitWorldCells = 64

	hitUnitID              = 5
	hitUnitCol, hitUnitRow = 5, 5 // frame (176,176) at the cell's centre

	hitTargetCol, hitTargetRow = 10, 12 // frame (336,400)

	hitEdgeRow = 12

	// The base frame's strip probe. Wide cases below move the probe with the
	// strip so this position never accidentally remains over newly revealed map.
	hitStripFrameY = 400

	// The pan the letterbox cases use, and the cell that sits at frame
	// (176,176) from there: 21*32+16 = 688 world, less the pan's 512.
	hitPanX, hitPanY             = 512, 512
	hitPanUnitCol, hitPanUnitRow = 21, 21
)

// hitScaleViewer is the fixture at one window size: the mission frame placed in that
// window, camera at the world origin, one unit on hitUnitCol/hitUnitRow.
func hitScaleViewer(t *testing.T, winW, winH int) *Viewer {
	t.Helper()
	v, err := NewViewer("hitscale", grid(hitWorldCells, hitWorldCells), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	hideBottomPanels(v)
	v.Layout(winW, winH)
	v.Camera().X, v.Camera().Y = 0, 0
	v.Camera().Clamp()
	if v.Camera().X != 0 || v.Camera().Y != 0 {
		t.Fatalf("fixture camera at (%v,%v), want the origin", v.Camera().X, v.Camera().Y)
	}
	if v.Camera().Zoom != 1 {
		t.Fatalf("fixture zoom = %v, want 1", v.Camera().Zoom)
	}
	if got, frameSize := v.ViewportSize(), v.FrameSize(); got != image.Pt(frameSize.X-MissionPanelW, MissionFrameH) {
		t.Fatalf("fixture camera view = %v, want frame %v less the %d-pixel strip", got, frameSize, MissionPanelW)
	}
	v.SetEntities([]MapEntity{{ID: hitUnitID, Cell: image.Pt(hitUnitCol, hitUnitRow)}})
	return v
}

// hitScales are the placements every hit test below is run at. The scale and
// origin are written out from the window size and the frame's 1024x768 exactly
// as missiongeometry_test.go's own table does, and for the same reason.
var hitScales = []struct {
	name                 string
	winW, winH           int
	scale, ox, oy        float64
	edgeCol, stripFrameX int
}{
	{"1:1", 1024, 768, 1, 0, 0, 26, 944},
	{"2:1", 2048, 1536, 2, 0, 0, 26, 944},
	{"1:2", 512, 384, 0.5, 0, 0, 26, 944},
	// ceil(1920*768/1080)=1366, scale 1920/1366, viewport 1206;
	// column 36 is the last full cell and the strip probe is at 1286.
	{"16:9, expanded to the side edges", 1920, 1080,
		1920.0 / 1366.0, 0, (1080.0 - 768.0*1920.0/1366.0) / 2, 36, 1286},
	{"4:5, letterboxed top and bottom", 1024, 1200, 1, 0, 216, 26, 944},
}

// windowAt is the window pixel at the centre of a frame pixel, from the
// placement's own arithmetic written out here.
func windowAt(fx, fy int, scale, ox, oy float64) (int, int) {
	return int(ox + scale*(float64(fx)+0.5)), int(oy + scale*(float64(fy)+0.5))
}

// cellWindowAt is windowAt applied to a cell's centre. The frame position of a
// cell centre is the camera's own forward transform, which is what every other
// test in this package derives an expected screen position from.
func cellWindowAt(v *Viewer, col, row int, scale, ox, oy float64) (int, int) {
	fx, fy := v.Camera().WorldToScreen(
		float64(col*camera.CellSize+camera.CellSize/2),
		float64(row*camera.CellSize+camera.CellSize/2))
	return windowAt(int(fx), int(fy), scale, ox, oy)
}

func TestMissionHitTestsHoldAtEveryScale(t *testing.T) {
	for _, tc := range hitScales {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("a tap selects the unit it is aimed at", func(t *testing.T) {
				v := hitScaleViewer(t, tc.winW, tc.winH)
				x, y := cellWindowAt(v, hitUnitCol, hitUnitRow, tc.scale, tc.ox, tc.oy)
				if ords, ok := tapAt(v, x, y); ok {
					t.Fatalf("the tap issued orders %+v", ords)
				}
				if len(v.sel) != 1 || v.sel[0] != hitUnitID {
					t.Fatalf("a tap at window (%d,%d), the unit's own cell (%d,%d), left selection %+v, want [%d]",
						x, y, hitUnitCol, hitUnitRow, v.sel, hitUnitID)
				}
			})

			t.Run("a left click orders the cell it is aimed at", func(t *testing.T) {
				v := hitScaleViewer(t, tc.winW, tc.winH)
				sx, sy := cellWindowAt(v, hitUnitCol, hitUnitRow, tc.scale, tc.ox, tc.oy)
				if _, ok := tapAt(v, sx, sy); ok {
					t.Fatal("the setup tap issued an order")
				}
				x, y := cellWindowAt(v, hitTargetCol, hitTargetRow, tc.scale, tc.ox, tc.oy)
				ords, ok := tapAt(v, x, y)
				if !ok || len(ords) != 1 {
					t.Fatalf("a left click at window (%d,%d) issued %+v (ok=%v), want one order", x, y, ords, ok)
				}
				if ords[0].x != hitTargetCol || ords[0].y != hitTargetRow {
					t.Fatalf("a left click aimed at cell (%d,%d) ordered cell (%d,%d)",
						hitTargetCol, hitTargetRow, ords[0].x, ords[0].y)
				}
			})

			t.Run("the last fully drawn column still orders", func(t *testing.T) {
				v := hitScaleViewer(t, tc.winW, tc.winH)
				sx, sy := cellWindowAt(v, hitUnitCol, hitUnitRow, tc.scale, tc.ox, tc.oy)
				tapAt(v, sx, sy)
				x, y := cellWindowAt(v, tc.edgeCol, hitEdgeRow, tc.scale, tc.ox, tc.oy)
				ords, ok := tapAt(v, x, y)
				if !ok || len(ords) != 1 || ords[0].x != tc.edgeCol || ords[0].y != hitEdgeRow {
					t.Fatalf("a left click on the last drawn column (%d,%d) issued %+v (ok=%v), want one order to that cell",
						tc.edgeCol, hitEdgeRow, ords, ok)
				}
			})

			t.Run("a press in the right strip orders nothing", func(t *testing.T) {
				v := hitScaleViewer(t, tc.winW, tc.winH)
				sx, sy := cellWindowAt(v, hitUnitCol, hitUnitRow, tc.scale, tc.ox, tc.oy)
				tapAt(v, sx, sy)
				x, y := windowAt(tc.stripFrameX, hitStripFrameY, tc.scale, tc.ox, tc.oy)
				if ords, ok := tapAt(v, x, y); ok {
					t.Fatalf("a left click in the strip, frame (%d,%d), ordered %+v; last fully drawn column is %d",
						tc.stripFrameX, hitStripFrameY, ords, tc.edgeCol)
				}
			})
		})
	}

	t.Run("a press in the letterbox orders nothing", func(t *testing.T) {
		for _, tc := range []struct {
			name          string
			winW, winH    int
			scale, ox, oy float64
			x, y          int
			cell          string
		}{
			// oy = 216 at scale 1, so window y = 100 is frame y = -116, world
			// 396, row 12.
			{"above a 1024x1200 window's frame", 1024, 1200, 1, 0, 216, 512, 100, "row 12"},
			// The frame ends at 216 + 768 = 984; window y = 1100 is frame
			// y = 884, world 1396, row 43.
			{"below a 1024x1200 window's frame", 1024, 1200, 1, 0, 216, 512, 1100, "row 43"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				v := hitScaleViewer(t, tc.winW, tc.winH)
				v.Camera().X, v.Camera().Y = hitPanX, hitPanY
				v.Camera().Clamp()
				if v.Camera().X != hitPanX || v.Camera().Y != hitPanY {
					t.Fatalf("the pan clamped to (%v,%v), want (%d,%d)", v.Camera().X, v.Camera().Y, hitPanX, hitPanY)
				}
				v.SetEntities([]MapEntity{{ID: hitUnitID, Cell: image.Pt(hitPanUnitCol, hitPanUnitRow)}})

				sx, sy := cellWindowAt(v, hitPanUnitCol, hitPanUnitRow, tc.scale, tc.ox, tc.oy)
				if _, ok := tapAt(v, sx, sy); ok {
					t.Fatal("the setup tap issued an order")
				}
				if len(v.sel) != 1 {
					t.Fatalf("the setup tap left selection %+v, want one unit: a press below cannot order with nothing selected", v.sel)
				}
				if ords, ok := tapAt(v, tc.x, tc.y); ok {
					t.Fatalf("a left click at window (%d,%d), outside the frame, ordered %+v: that position is on the grid at %s",
						tc.x, tc.y, ords, tc.cell)
				}
			})
		}
	})
}

// TestUnmappedWindowPixelsWouldMissTheirSurface is this file's own control. It
// asserts that the mapping does work rather than agreeing with the identity: at
// 2:1 the same numbers taken as window pixels reach a different cell, so a hit
// test that read window pixels directly would answer wrongly rather than
// coincidentally right.
func TestUnmappedWindowPixelsWouldMissTheirSurface(t *testing.T) {
	v := hitScaleViewer(t, 2048, 1536)

	// The unit's own frame position, sent unmapped. At 2:1 the door halves it,
	// so it resolves to cell (2,2) and not the unit's (5,5).
	fx, fy := v.Camera().WorldToScreen(
		float64(hitUnitCol*camera.CellSize+camera.CellSize/2),
		float64(hitUnitRow*camera.CellSize+camera.CellSize/2))
	if _, ok := tapAt(v, int(fx), int(fy)); ok {
		t.Fatal("the tap issued an order")
	}
	if len(v.sel) != 0 {
		t.Fatalf("an unmapped tap at (%v,%v) selected %+v: at 2:1 those numbers are a different cell, so this test could not tell a mapped hit from an unmapped one",
			fx, fy, v.sel)
	}
}
