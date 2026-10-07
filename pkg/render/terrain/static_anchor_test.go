package terrain_test

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// staticAnchorCase is one hand-computed row: the ten inputs StaticAnchor takes
// and the four values it must return for them.
type staticAnchorCase struct {
	name             string
	col, row         int
	canvasW, canvasH int
	centerX, centerY int
	frameW, frameH   int
	lift, originY    int

	destX, destY     int
	anchorX, anchorY int

	why string
}

// The rows cover AC-2's three frame-to-canvas relations (equal, smaller,
// larger), odd canvases and centres in both parities against the frame, both
// signs of lift and originY and a mixed pair, and the -1 a class hands over for
// a scalar nothing on its inheritance chain sets. The last row is deliberately
// one that a wrong implementation can PASS, so the bound of what these cases
// discriminate is visible in the file rather than only in a report.
var staticAnchorCases = []staticAnchorCase{
	{
		name: "frame fills the canvas, flat geometry",
		col:  0, row: 0,
		canvasW: 64, canvasH: 64, centerX: 32, centerY: 48,
		frameW: 64, frameH: 64,
		destX: -16, destY: -32, anchorX: 32, anchorY: 48,
		why: "AC-2's reduction: the halvings cancel, so the anchor IS the centre and the top-left is the cell centre minus it — negative here, the sprite reaching left of and above the map's own origin",
	},
	{
		name: "frame smaller than the canvas",
		col:  3, row: 5,
		canvasW: 64, canvasH: 96, centerX: 30, centerY: 80,
		frameW: 20, frameH: 24,
		destX: 104, destY: 132, anchorX: 8, anchorY: 44,
		why: "the frame's half is added back after the canvas's is taken off, so a smaller frame anchors nearer its own top-left",
	},
	{
		name: "frame larger than the canvas",
		col:  2, row: 1,
		canvasW: 32, canvasH: 32, centerX: 16, centerY: 30,
		frameW: 96, frameH: 128,
		destX: 32, destY: -30, anchorX: 48, anchorY: 78,
		why: "the other direction, which an anchor clamped to its canvas could not produce",
	},
	{
		name: "odd canvas, even frame",
		col:  4, row: 7,
		canvasW: 33, canvasH: 45, centerX: 17, centerY: 23,
		frameW: 16, frameH: 8,
		destX: 135, destY: 235, anchorX: 9, anchorY: 5,
		why: "33/2 is 16 and 45/2 is 22: only the canvas's halving is odd here, so a rounded /2 cannot cancel against the frame's",
	},
	{
		name: "even canvas, odd frame",
		col:  1, row: 2,
		canvasW: 32, canvasH: 64, centerX: 15, centerY: 33,
		frameW: 19, frameH: 27,
		destX: 40, destY: 66, anchorX: 8, anchorY: 14,
		why: "the mirror: 19/2 is 9 and 27/2 is 13, so the odd halving is the frame's and rounding moves the anchor the other way",
	},
	{
		name: "lift and originY both negative",
		col:  6, row: 3,
		canvasW: 48, canvasH: 48, centerX: 24, centerY: 40,
		frameW: 48, frameH: 48,
		lift: -7, originY: -19,
		destX: 184, destY: 98,
		anchorX: 24, anchorY: 40,
		why: "both terms are subtracted, so two negatives push the sprite DOWN the canvas; a negative OriginY is the ordinary case for a projected render",
	},
	{
		name: "lift and originY both positive",
		col:  6, row: 3,
		canvasW: 48, canvasH: 48, centerX: 24, centerY: 40,
		frameW: 48, frameH: 48,
		lift: 11, originY: 25,
		destX: 184, destY: 36,
		anchorX: 24, anchorY: 40,
		why: "the same cell and class as the row above, so the 62-pixel gap between the two destY values is the two terms' signs and nothing else",
	},
	{
		name: "lift positive, originY negative",
		col:  0, row: 0,
		canvasW: 10, canvasH: 10, centerX: 5, centerY: 5,
		frameW: 10, frameH: 10,
		lift: 9, originY: -4,
		destX: 11, destY: 6,
		anchorX: 5, anchorY: 5,
		why: "mixed signs, which an implementation folding the two terms into one absolute magnitude would fail",
	},
	{
		name: "absent-scalar canvas and centre of -1",
		col:  2, row: 2,
		canvasW: -1, canvasH: -1, centerX: -1, centerY: -1,
		frameW: 3, frameH: 5,
		destX: 80, destY: 79, anchorX: 0, anchorY: 1,
		why: "-1/2 is 0 by truncation toward zero and -1 by flooring, so this row alone separates the contract's /2 from a right shift; pkg/data yields -1 for an unset object scalar, and all 82 shipped classes happen to set these four",
	},
	{
		name: "a 1x1 frame",
		col:  0, row: 1,
		canvasW: 8, canvasH: 8, centerX: 4, centerY: 6,
		frameW: 1, frameH: 1,
		destX: 16, destY: 46, anchorX: 0, anchorY: 2,
		why: "1/2 is 0, so the frame's own half vanishes and an anchor that halved the canvas ALONE is right here — the one row that mutant survives, kept so the suite's bound is visible",
	},
}

// TestStaticAnchorHandComputed covers SC-2 (AC-2): the four returned values
// against every row's literals.
func TestStaticAnchorHandComputed(t *testing.T) {
	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; every destX/destY literal below is stated over 32-pixel cells", terrain.CellSize)
	}

	for _, c := range staticAnchorCases {
		destX, destY, anchorX, anchorY := terrain.StaticAnchor(
			c.col, c.row, c.canvasW, c.canvasH, c.centerX, c.centerY, c.frameW, c.frameH, c.lift, c.originY)

		if anchorX != c.anchorX || anchorY != c.anchorY {
			t.Errorf("%s: anchor = (%d,%d), want (%d,%d) — %s",
				c.name, anchorX, anchorY, c.anchorX, c.anchorY, c.why)
		}
		if destX != c.destX || destY != c.destY {
			t.Errorf("%s: top-left = (%d,%d), want (%d,%d) — %s",
				c.name, destX, destY, c.destX, c.destY, c.why)
		}
	}
}

// TestStaticAnchorEqualFrameReducesToTheCellCentre covers AC-2's stated
// reduction on its own terms: wherever the drawn frame fills its class's
// canvas and the geometry is flat, the anchor is the centre pixel itself and
// the top-left is (col*32+16-centerX, row*32+16-centerY).
//
// This is a SIMPLER statement than the formula, not a transcription of it — the
// two halvings are the same value here, whatever the parity, so the reduction
// holds at odd canvases and at the negative default too. What it pins is that
// the canvas and frame terms carry opposite signs: any implementation adding
// them, or halving only one of the two, breaks it at every size.
func TestStaticAnchorEqualFrameReducesToTheCellCentre(t *testing.T) {
	cells := []struct{ col, row int }{{0, 0}, {1, 0}, {0, 1}, {7, 13}, {255, 255}}
	squares := []struct {
		size             int
		centerX, centerY int
	}{
		{64, 32, 48},
		{33, 17, 23}, // odd: the halvings still cancel exactly
		{1, 0, 0},
		{-1, -1, -1}, // the absent-scalar default, canvas and centre alike
	}

	for _, cell := range cells {
		for _, s := range squares {
			destX, destY, anchorX, anchorY := terrain.StaticAnchor(
				cell.col, cell.row, s.size, s.size, s.centerX, s.centerY, s.size, s.size, 0, 0)

			if anchorX != s.centerX || anchorY != s.centerY {
				t.Errorf("cell (%d,%d), %dx%d canvas and frame: anchor = (%d,%d), want the centre (%d,%d)",
					cell.col, cell.row, s.size, s.size, anchorX, anchorY, s.centerX, s.centerY)
			}
			wantX := cell.col*terrain.CellSize + terrain.CellSize/2 - s.centerX
			wantY := cell.row*terrain.CellSize + terrain.CellSize/2 - s.centerY
			if destX != wantX || destY != wantY {
				t.Errorf("cell (%d,%d), %dx%d canvas and frame: top-left = (%d,%d), want (%d,%d)",
					cell.col, cell.row, s.size, s.size, destX, destY, wantX, wantY)
			}
		}
	}
}

func TestStaticAnchorDisplacesVerticallyOnly(t *testing.T) {
	displacements := []struct{ lift, originY int }{
		{0, 0}, {7, 0}, {0, 7}, {-7, 0}, {0, -7}, {13, -29}, {-13, 29}, {1, -1}, {-1, 1},
	}

	for _, c := range staticAnchorCases {
		flatX, flatY, flatAX, flatAY := terrain.StaticAnchor(
			c.col, c.row, c.canvasW, c.canvasH, c.centerX, c.centerY, c.frameW, c.frameH, 0, 0)

		for _, d := range displacements {
			destX, destY, anchorX, anchorY := terrain.StaticAnchor(
				c.col, c.row, c.canvasW, c.canvasH, c.centerX, c.centerY, c.frameW, c.frameH, d.lift, d.originY)

			if destX != flatX {
				t.Errorf("%s at lift=%d originY=%d: destX = %d, want the flat %d — the displacement is vertical only and destX takes no origin term (P-2)",
					c.name, d.lift, d.originY, destX, flatX)
			}
			if want := flatY - (d.lift + d.originY); destY != want {
				t.Errorf("%s at lift=%d originY=%d: destY = %d, want %d (the flat %d shifted by exactly -(lift+originY))",
					c.name, d.lift, d.originY, destY, want, flatY)
			}
			if anchorX != flatAX || anchorY != flatAY {
				t.Errorf("%s at lift=%d originY=%d: anchor = (%d,%d), want the flat (%d,%d) — the frame's own anchor pixel is geometry-independent",
					c.name, d.lift, d.originY, anchorX, anchorY, flatAX, flatAY)
			}
		}
	}
}

// staticAnchorPlace is the cell-to-world tail every variant below shares: the
// ground point is the cell's centre, from which the anchor pixel and the two
// vertical terms are subtracted. Only the ANCHOR rule differs between the
// variants, so each one perturbs exactly the thing it is named for. It is used
// by the variants alone — the assertions above never go through it.
func staticAnchorPlace(c staticAnchorCase, anchorX, anchorY int) (destX, destY int) {
	return c.col*terrain.CellSize + terrain.CellSize/2 - anchorX,
		c.row*terrain.CellSize + terrain.CellSize/2 - anchorY - c.lift - c.originY
}

// staticAnchorRoundHalf halves rounding half AWAY FROM ZERO rather than
// truncating toward it: SC-2's second required break.
func staticAnchorRoundHalf(v int) int {
	if v < 0 {
		return -((-v + 1) / 2)
	}
	return (v + 1) / 2
}

// TestStaticAnchorWrongHalvingsFailNamedRows covers SC-2's breaks. Each variant
// is a wrong anchor a reader could plausibly write; the assertion is the EXACT
// set of rows it disagrees with the literals on, so the suite states both what
// it catches and what it does not.
//
// A variant that fails no row would mean the suite cannot see that mistake at
// all; a variant failing every row would mean no row is telling us anything
// specific. Both are worth knowing, which is why the sets are written out.
func TestStaticAnchorWrongHalvingsFailNamedRows(t *testing.T) {
	variants := []struct {
		name   string
		anchor func(c staticAnchorCase) (anchorX, anchorY int)
		kills  []string
		why    string
	}{
		{
			name: "the canvas halved alone, the frame's own half dropped",
			anchor: func(c staticAnchorCase) (int, int) {
				return c.centerX - c.canvasW/2, c.centerY - c.canvasH/2
			},
			kills: []string{
				"frame fills the canvas, flat geometry",
				"frame smaller than the canvas",
				"frame larger than the canvas",
				"odd canvas, even frame",
				"even canvas, odd frame",
				"lift and originY both negative",
				"lift and originY both positive",
				"lift positive, originY negative",
				"absent-scalar canvas and centre of -1",
			},
			why: "every row but the 1x1 frame, whose own half is 0",
		},
		{
			name: "every /2 rounded away from zero instead of truncated",
			anchor: func(c staticAnchorCase) (int, int) {
				return c.centerX - staticAnchorRoundHalf(c.canvasW) + staticAnchorRoundHalf(c.frameW),
					c.centerY - staticAnchorRoundHalf(c.canvasH) + staticAnchorRoundHalf(c.frameH)
			},
			kills: []string{
				"odd canvas, even frame",
				"even canvas, odd frame",
				"absent-scalar canvas and centre of -1",
				"a 1x1 frame",
			},
			why: "only where the canvas's and the frame's halvings do not both round the same way; two odd dimensions would cancel, which is why no row has that shape",
		},
		{
			name: "every /2 written as a right shift, so a negative floors",
			anchor: func(c staticAnchorCase) (int, int) {
				return c.centerX - (c.canvasW >> 1) + (c.frameW >> 1),
					c.centerY - (c.canvasH >> 1) + (c.frameH >> 1)
			},
			kills: []string{"absent-scalar canvas and centre of -1"},
			why:   "the shift agrees with the contract at every non-negative dimension, so the -1 default is the whole of what separates them",
		},
		{
			name: "the centre alone, the baseline's own shape",
			anchor: func(c staticAnchorCase) (int, int) {
				return c.centerX, c.centerY
			},
			kills: []string{
				"frame smaller than the canvas",
				"frame larger than the canvas",
				"odd canvas, even frame",
				"even canvas, odd frame",
				"absent-scalar canvas and centre of -1",
				"a 1x1 frame",
			},
			why: "right exactly where a frame fills its canvas, which is DD-2's reason for a signature that cannot omit either",
		},
	}

	for _, v := range variants {
		var failed []string
		for _, c := range staticAnchorCases {
			anchorX, anchorY := v.anchor(c)
			destX, destY := staticAnchorPlace(c, anchorX, anchorY)
			if anchorX != c.anchorX || anchorY != c.anchorY || destX != c.destX || destY != c.destY {
				failed = append(failed, c.name)
			}
		}
		t.Logf("variant %q fails %d of %d rows: %v (%s)", v.name, len(failed), len(staticAnchorCases), failed, v.why)

		if len(failed) == 0 {
			t.Errorf("variant %q passes every row: the suite cannot tell it from the contract", v.name)
			continue
		}
		if len(failed) != len(v.kills) {
			t.Errorf("variant %q fails %v, want exactly %v", v.name, failed, v.kills)
			continue
		}
		for i, name := range failed {
			if name != v.kills[i] {
				t.Errorf("variant %q fails %v, want exactly %v (rows are compared in table order)", v.name, failed, v.kills)
				break
			}
		}
	}
}

func TestStaticBundleIsKeyedByTheWholeByteDomain(t *testing.T) {
	var set terrain.StaticSet
	if len(set.Classes) != 256 {
		t.Errorf("StaticSet holds %d class slots, want 256 — one per placement byte, so no lookup needs a bounds test", len(set.Classes))
	}
	for b := range set.Classes {
		if set.Classes[b] != nil {
			t.Fatalf("a zero StaticSet answers non-nil at byte %d; it must be a legal empty bundle", b)
		}
	}

	var empty terrain.StaticFrame
	if len(empty.Palette) != 256 {
		t.Errorf("StaticFrame holds %d palette entries, want 256 — one per uint8 index a pixel can carry", len(empty.Palette))
	}

	drawable := &terrain.StaticClass{Width: 8, Height: 8, CenterX: 4, CenterY: 6, Frame: &terrain.StaticFrame{
		Width: 1, Height: 1, Pixels: []terrain.StaticPixel{{Index: 0, Opaque: true}},
	}}
	set.Classes[1] = drawable
	set.Classes[2] = &terrain.StaticClass{Width: 8, Height: 8, CenterX: 4, CenterY: 6} // resolved, artless

	if set.Classes[0] != nil {
		t.Error("byte 0 must stay nil: 0 is *no object*, not the class whose ID is 0")
	}
	if set.Classes[2] == nil || set.Classes[2].Frame != nil {
		t.Error("an artless class must be a non-nil class with a nil Frame — the skip kinds are counted apart")
	}
	if set.Classes[3] != nil {
		t.Error("a byte naming no loaded class must answer nil")
	}
	if p := drawable.Frame.Pixels[0]; !p.Opaque || p.Index != 0 {
		t.Error("an opaque pixel at index 0 must be representable: transparency is structural, never a reserved index")
	}

	// The fields compose into the anchor without a conversion or a second
	// halving anywhere: this is the whole of what the per-cell path does with a
	// class (T3 adds the walk over the grid).
	f := drawable.Frame
	destX, destY, anchorX, anchorY := terrain.StaticAnchor(
		0, 1, drawable.Width, drawable.Height, drawable.CenterX, drawable.CenterY, f.Width, f.Height, 0, 0)
	if destX != 16 || destY != 46 || anchorX != 0 || anchorY != 2 {
		t.Errorf("the bundle's own class through StaticAnchor = (%d,%d) anchor (%d,%d), want (16,46) anchor (0,2) — the \"a 1x1 frame\" row",
			destX, destY, anchorX, anchorY)
	}
}
