package terrain_test

// Tests for the terrain-tier per-marker height lift added by
// docs/0015-height-projected-placements (tasks.md T2):
// DrawObjectMarkersAtHeights / DrawUnitMarkersAtHeights, and the
// liftY/liftAt seam they thread through the package-private markerRects /
// drawMarkers.
//
// SEPARATE CONTEXT, as every other test file in this package states of
// itself.
//
// Every fixture is synthetic; nothing here reads a game install.

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

func paintLiftMask(imgW, imgH int, cells []image.Point, cols, rows, cellpx, offsetY int, lift func(col, row int) int, radius, thickness int) []bool {
	want := make([]bool, imgW*imgH)
	bounds := image.Rect(0, 0, imgW, imgH)
	stage1 := specMapRectAt(cols, rows, cellpx, offsetY)

	for _, cell := range cells {
		if cell.X < 0 || cell.Y < 0 || cell.X >= cols || cell.Y >= rows {
			continue
		}
		ly := 0
		if lift != nil {
			ly = lift(cell.X, cell.Y)
		}
		for _, arm := range specArmsRawAt(cell.X, cell.Y, cellpx, offsetY+ly, radius, thickness) {
			c := arm.Intersect(stage1)
			if c.Empty() {
				continue
			}
			if c = c.Intersect(bounds); c.Empty() {
				continue
			}
			for y := c.Min.Y; y < c.Max.Y; y++ {
				for x := c.Min.X; x < c.Max.X; x++ {
					want[(y-bounds.Min.Y)*imgW+(x-bounds.Min.X)] = true
				}
			}
		}
	}
	return want
}

func paintFoldedLiftMask(imgW, imgH int, cells []image.Point, cols, rows, cellpx, offsetY int, lift func(col, row int) int, radius, thickness int) []bool {
	want := make([]bool, imgW*imgH)
	bounds := image.Rect(0, 0, imgW, imgH)

	for _, cell := range cells {
		if cell.X < 0 || cell.Y < 0 || cell.X >= cols || cell.Y >= rows {
			continue
		}
		ly := 0
		if lift != nil {
			ly = lift(cell.X, cell.Y)
		}
		stage1 := specMapRectAt(cols, rows, cellpx, offsetY+ly) // the trap
		for _, arm := range specArmsRawAt(cell.X, cell.Y, cellpx, offsetY+ly, radius, thickness) {
			c := arm.Intersect(stage1)
			if c.Empty() {
				continue
			}
			if c = c.Intersect(bounds); c.Empty() {
				continue
			}
			for y := c.Min.Y; y < c.Max.Y; y++ {
				for x := c.Min.X; x < c.Max.X; x++ {
					want[(y-bounds.Min.Y)*imgW+(x-bounds.Min.X)] = true
				}
			}
		}
	}
	return want
}

// masksEqual is a plain element-wise comparison of two boolean pixel masks of
// the same length.
func masksEqual(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// maskCount is the number of true entries in a pixel mask.
func maskCount(m []bool) int {
	n := 0
	for _, v := range m {
		if v {
			n++
		}
	}
	return n
}

// checkAgainstMask paints markerBG over an imgW x imgH RGBA anchored at the
// origin, runs draw, and asserts that EXACTLY the pixels the mask marks hold
// markColor and every other pixel still holds the background — the same
// complement-checking discipline overlay_test.go's drawAndCheck/drawBothAt use.
// It returns the marked-pixel count.
func checkAgainstMask(t *testing.T, what string, imgW, imgH int, want []bool, markColor color.RGBA, draw func(img *image.RGBA)) int {
	t.Helper()

	img := backgroundImage(imgW, imgH)
	noPanicOverlay(t, what, func() { draw(img) })

	marked := 0
	for y := 0; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			got := img.RGBAAt(x, y)
			if want[y*imgW+x] {
				marked++
				if got != markColor {
					t.Fatalf("%s: pixel (%d,%d) = %v, want %v (inside the DD-2 lift oracle mask)", what, x, y, got, markColor)
				}
				continue
			}
			if got != markerBG {
				t.Fatalf("%s: pixel (%d,%d) = %v, want the untouched background %v (outside every clipped, lifted arm)", what, x, y, got, markerBG)
			}
		}
	}
	return marked
}

// ---------------------------------------------------------------------------
// SC-4: the four pre-existing exported functions are unaffected, and the two
// new *Heights entry points reduce to them exactly at a zero lift — both a nil
// callback and one that always answers 0.

func TestMarkerLiftZeroMatchesExistingEntryPoints(t *testing.T) {
	const cols, rows = 5, 4
	cells := []image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 4, Y: 3}, {X: cols, Y: 0}, {X: -1, Y: -1}}
	zero := func(col, row int) int { return 0 }

	canvases := []struct {
		cellpx, offsetY, imgH int
	}{
		{32, 0, rows * 32},
		{32, 40, rows*32 + 40},
		{32, -24, rows*32 + 12},
		{64, 40, rows*64 + 40},
	}

	for _, cv := range canvases {
		w := cols * cv.cellpx

		object := backgroundImage(w, cv.imgH)
		terrain.DrawObjectMarkersAt(object, cells, cols, rows, cv.cellpx, cv.offsetY)

		objectNil := backgroundImage(w, cv.imgH)
		terrain.DrawObjectMarkersAtHeights(objectNil, cells, cols, rows, cv.cellpx, cv.offsetY, nil)

		objectZero := backgroundImage(w, cv.imgH)
		terrain.DrawObjectMarkersAtHeights(objectZero, cells, cols, rows, cv.cellpx, cv.offsetY, zero)

		for i := range object.Pix {
			if object.Pix[i] != objectNil.Pix[i] {
				t.Fatalf("cellpx=%d offsetY=%d: DrawObjectMarkersAtHeights with a nil liftY differs from DrawObjectMarkersAt at Pix[%d] (%d vs %d); a nil lift must be byte-identical to the pre-0015 entry point",
					cv.cellpx, cv.offsetY, i, object.Pix[i], objectNil.Pix[i])
			}
			if object.Pix[i] != objectZero.Pix[i] {
				t.Fatalf("cellpx=%d offsetY=%d: DrawObjectMarkersAtHeights with an always-0 liftY differs from DrawObjectMarkersAt at Pix[%d] (%d vs %d); a zero lift must be byte-identical to the pre-0015 entry point",
					cv.cellpx, cv.offsetY, i, object.Pix[i], objectZero.Pix[i])
			}
		}

		unit := backgroundImage(w, cv.imgH)
		terrain.DrawUnitMarkersAt(unit, cells, cols, rows, cv.cellpx, cv.offsetY)

		unitNil := backgroundImage(w, cv.imgH)
		terrain.DrawUnitMarkersAtHeights(unitNil, cells, cols, rows, cv.cellpx, cv.offsetY, nil)

		unitZero := backgroundImage(w, cv.imgH)
		terrain.DrawUnitMarkersAtHeights(unitZero, cells, cols, rows, cv.cellpx, cv.offsetY, zero)

		for i := range unit.Pix {
			if unit.Pix[i] != unitNil.Pix[i] {
				t.Fatalf("cellpx=%d offsetY=%d: DrawUnitMarkersAtHeights with a nil liftY differs from DrawUnitMarkersAt at Pix[%d] (%d vs %d)",
					cv.cellpx, cv.offsetY, i, unit.Pix[i], unitNil.Pix[i])
			}
			if unit.Pix[i] != unitZero.Pix[i] {
				t.Fatalf("cellpx=%d offsetY=%d: DrawUnitMarkersAtHeights with an always-0 liftY differs from DrawUnitMarkersAt at Pix[%d] (%d vs %d)",
					cv.cellpx, cv.offsetY, i, unit.Pix[i], unitZero.Pix[i])
			}
		}
	}
}

// ---------------------------------------------------------------------------
// SC-7: the stage-1 clip is built from the UNLIFTED map extent.

func TestMarkerLiftClipsAtUnliftedTopExtent(t *testing.T) {
	// A canvas already translated by offsetY = 50 (an OriginY of -50 in
	// terraintool terms), so the map's own top edge sits at output row 50, away
	// from the image's own row 0 - otherwise img.Bounds() alone would clip the
	// same pixels a correct mapRect does, and the fold could not be told apart
	// from the fix (image.RGBA can never hold a row < 0, so a fixture at
	// offsetY = 0 cannot isolate this).
	const cols, rows, cellpx, offsetY = 4, 3, 32, 50
	const imgW, imgH = cols * cellpx, 200 // generous: stage-2 must not be what clips here

	cells := []image.Point{{X: 0, Y: 0}}

	t.Run("lift exceeds the reach entirely: the anchor's whole cross clips away", func(t *testing.T) {
		// cy at row 0 is 0*32+16+50 = 66; liftY=-40 carries it to 26, whose
		// r=6 cross (native y in [20,33)) sits entirely above the map's own top
		// edge at output row 50. The correct answer is nothing at all; a folded
		// implementation moves the clip's own top up by 40 as well (to row 10)
		// and would still draw the whole cross.
		const liftY = -40
		lift := func(col, row int) int { return liftY }

		correct := paintLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, lift, specObjectRadius, specObjectThickness)
		folded := paintFoldedLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, lift, specObjectRadius, specObjectThickness)
		if masksEqual(correct, folded) {
			t.Fatalf("fixture guard: the correct and folded oracles agree on this case (both mark %d px), so it cannot distinguish DD-2's fix from FR-4's named trap",
				maskCount(correct))
		}
		if n := maskCount(correct); n != 0 {
			t.Fatalf("fixture guard: expected the correct oracle to clip this anchor's cross away entirely (0 px), got %d", n)
		}
		if n := maskCount(folded); n == 0 {
			t.Fatalf("fixture guard: expected the FOLDED oracle to still draw this anchor's cross (its own clip moved with the lift), got 0 px")
		}

		marked := checkAgainstMask(t, "top-edge full clip", imgW, imgH, correct, terrain.MarkerColor, func(img *image.RGBA) {
			terrain.DrawObjectMarkersAtHeights(img, cells, cols, rows, cellpx, offsetY, lift)
		})
		if marked != 0 {
			t.Errorf("marked %d pixels, want 0: a marker lifted entirely past the map's own (unlifted) top edge must clip away completely (SC-7, FR-4)", marked)
		}
	})

	t.Run("lift exceeds the reach partially: the cross truncates at the unlifted edge", func(t *testing.T) {
		// cy at row 0 is 66; liftY=-13 carries it to 53, whose r=6 vertical arm
		// (native y in [47,60)) straddles the map's own top edge at 50. The
		// correct answer truncates that arm to [50,60); a folded implementation,
		// whose own clip has moved up by 13 (to row 37), leaves it whole at
		// [47,60).
		const liftY = -13
		lift := func(col, row int) int { return liftY }

		correct := paintLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, lift, specObjectRadius, specObjectThickness)
		folded := paintFoldedLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, lift, specObjectRadius, specObjectThickness)
		if masksEqual(correct, folded) {
			t.Fatalf("fixture guard: the correct and folded oracles agree on this case, so it cannot distinguish DD-2's fix from FR-4's named trap")
		}
		if maskCount(correct) == 0 {
			t.Fatalf("fixture guard: expected the correct oracle to still mark a truncated remainder, got 0 px")
		}
		if maskCount(folded) <= maskCount(correct) {
			t.Fatalf("fixture guard: expected the folded oracle to mark MORE pixels than the correctly-clipped one (%d), got %d",
				maskCount(correct), maskCount(folded))
		}

		marked := checkAgainstMask(t, "top-edge partial clip", imgW, imgH, correct, terrain.MarkerColor, func(img *image.RGBA) {
			terrain.DrawObjectMarkersAtHeights(img, cells, cols, rows, cellpx, offsetY, lift)
		})
		if marked == 0 {
			t.Errorf("marked 0 pixels, want the truncated remainder (SC-7): the anchor's cross only partly crosses the map's own top edge")
		}
	})
}

func TestMarkerLiftClipsAtUnliftedBottomExtent(t *testing.T) {
	const cols, rows, cellpx, offsetY = 4, 3, 32, 0
	const imgW, imgH = cols * cellpx, 200 // generous: stage-2 must not be what clips here

	cells := []image.Point{{X: 0, Y: rows - 1}} // the bottom row

	t.Run("downward lift exceeds the reach entirely", func(t *testing.T) {
		// cy at row 2 is 2*32+16 = 80; liftY=+40 carries it to 120, whose r=6
		// cross (native y in [114,127)) sits entirely below the map's own
		// bottom edge at output row 96. The correct answer is nothing; a folded
		// implementation moves its own clip's bottom down by 40 as well (to row
		// 136) and would still draw the whole cross.
		const liftY = 40
		lift := func(col, row int) int { return liftY }

		correct := paintLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, lift, specObjectRadius, specObjectThickness)
		folded := paintFoldedLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, lift, specObjectRadius, specObjectThickness)
		if masksEqual(correct, folded) {
			t.Fatalf("fixture guard: the correct and folded oracles agree on this case, so it cannot distinguish DD-2's fix from FR-4's named trap")
		}
		if n := maskCount(correct); n != 0 {
			t.Fatalf("fixture guard: expected the correct oracle to clip this anchor's cross away entirely, got %d px", n)
		}
		if n := maskCount(folded); n == 0 {
			t.Fatalf("fixture guard: expected the FOLDED oracle to still draw this anchor's cross, got 0 px")
		}

		marked := checkAgainstMask(t, "bottom-edge full clip", imgW, imgH, correct, terrain.MarkerColor, func(img *image.RGBA) {
			terrain.DrawObjectMarkersAtHeights(img, cells, cols, rows, cellpx, offsetY, lift)
		})
		if marked != 0 {
			t.Errorf("marked %d pixels, want 0: a marker lifted downward entirely past the map's own (unlifted) bottom edge must clip away completely (SC-7 mirror, AC-7)", marked)
		}
	})

	t.Run("downward lift exceeds the reach partially", func(t *testing.T) {
		// cy at row 2 is 80; liftY=+13 carries it to 93, whose r=6 vertical arm
		// (native y in [87,100)) straddles the map's own bottom edge at 96. The
		// correct answer truncates that arm to [87,96); a folded implementation,
		// whose own clip has moved down by 13 (to row 109), leaves it whole.
		const liftY = 13
		lift := func(col, row int) int { return liftY }

		correct := paintLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, lift, specObjectRadius, specObjectThickness)
		folded := paintFoldedLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, lift, specObjectRadius, specObjectThickness)
		if masksEqual(correct, folded) {
			t.Fatalf("fixture guard: the correct and folded oracles agree on this case, so it cannot distinguish DD-2's fix from FR-4's named trap")
		}
		if maskCount(correct) == 0 {
			t.Fatalf("fixture guard: expected the correct oracle to still mark a truncated remainder, got 0 px")
		}
		if maskCount(folded) <= maskCount(correct) {
			t.Fatalf("fixture guard: expected the folded oracle to mark MORE pixels than the correctly-clipped one (%d), got %d",
				maskCount(correct), maskCount(folded))
		}

		marked := checkAgainstMask(t, "bottom-edge partial clip", imgW, imgH, correct, terrain.MarkerColor, func(img *image.RGBA) {
			terrain.DrawObjectMarkersAtHeights(img, cells, cols, rows, cellpx, offsetY, lift)
		})
		if marked == 0 {
			t.Errorf("marked 0 pixels, want the truncated remainder (SC-7 mirror): the anchor's cross only partly crosses the map's own bottom edge")
		}
	})
}

// ---------------------------------------------------------------------------
// Direction: an interior anchor, far from every edge, translates by EXACTLY
// liftY and by nothing else. This is what a sign flip in the lift (adding
// -liftY instead of +liftY) or a dropped lift (liftY read but never applied)
// would fail: the oracle's mask, at each of these liftY values, sits at a
// vertical band the wrong sign or a zero lift would never reach.

func TestMarkerLiftTranslatesInteriorAnchorsBySignedAmount(t *testing.T) {
	const cols, rows, cellpx, offsetY = 6, 6, 32, 0
	const imgW, imgH = cols * cellpx, rows * cellpx // the whole map; no clip anywhere below

	cell := image.Point{X: 2, Y: 2} // interior: cy = 2*32+16 = 80, far from every edge
	cells := []image.Point{cell}

	for _, liftY := range []int{-50, -20, -1, 0, 1, 20, 50} {
		liftY := liftY
		lift := func(col, row int) int { return liftY }

		want := paintLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, lift, specObjectRadius, specObjectThickness)
		if n := maskCount(want); n == 0 {
			t.Fatalf("fixture guard: liftY=%d marked 0 oracle pixels; the case can say nothing", liftY)
		}

		marked := checkAgainstMask(t, "object, liftY", imgW, imgH, want, terrain.MarkerColor, func(img *image.RGBA) {
			terrain.DrawObjectMarkersAtHeights(img, cells, cols, rows, cellpx, offsetY, lift)
		})
		if marked != maskCount(want) {
			t.Errorf("liftY=%d: marked %d pixels, want %d (the interior cross translated by exactly liftY and by nothing else)",
				liftY, marked, maskCount(want))
		}

		unitWant := paintLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, lift, specUnitRadius, specUnitThickness)
		unitMarked := checkAgainstMask(t, "unit, liftY", imgW, imgH, unitWant, terrain.UnitMarkerColor, func(img *image.RGBA) {
			terrain.DrawUnitMarkersAtHeights(img, cells, cols, rows, cellpx, offsetY, lift)
		})
		if unitMarked != maskCount(unitWant) {
			t.Errorf("unit liftY=%d: marked %d pixels, want %d", liftY, unitMarked, maskCount(unitWant))
		}
	}

	// A same-magnitude, opposite-sign pair must land on DISJOINT pixel sets: if
	// the two masks below shared any pixel, either the sign is not threading
	// through at all (both landing on the zero-lift band) or the fixture is too
	// small to tell +30 from -30 apart.
	pos := paintLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, func(int, int) int { return 30 }, specObjectRadius, specObjectThickness)
	neg := paintLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, func(int, int) int { return -30 }, specObjectRadius, specObjectThickness)
	for i := range pos {
		if pos[i] && neg[i] {
			t.Fatalf("fixture guard: liftY=+30 and liftY=-30 oracles overlap at pixel index %d; they must land on disjoint bands for the sign to be checkable", i)
		}
	}
}

func TestMarkerLiftCullsAfterTheHeightOffset(t *testing.T) {
	const cols, rows, cellpx, offsetY = 4, 6, 32, 0
	const imgW, imgH = cols * cellpx, 50 // rows 0..49 only; the map itself is 192 rows tall

	intoView := image.Point{X: 0, Y: 4}  // cy = 4*32+16 = 144: off-canvas unlifted
	outOfView := image.Point{X: 0, Y: 0} // cy = 0*32+16 = 16: on-canvas unlifted
	cells := []image.Point{intoView, outOfView}

	lift := func(col, row int) int {
		switch {
		case col == intoView.X && row == intoView.Y:
			return -110 // 144-110=34: now on-canvas (arm y in roughly [28,41))
		case col == outOfView.X && row == outOfView.Y:
			return 40 // 16+40=56: now off-canvas (arm y in roughly [50,63), >= imgH)
		default:
			return 0
		}
	}

	// Fixture guards: each anchor's own zero-lift visibility is the opposite of
	// its lifted visibility, or the case proves nothing about culling ORDER.
	unliftedIntoView := paintLiftMask(imgW, imgH, []image.Point{intoView}, cols, rows, cellpx, offsetY, nil, specObjectRadius, specObjectThickness)
	liftedIntoView := paintLiftMask(imgW, imgH, []image.Point{intoView}, cols, rows, cellpx, offsetY, lift, specObjectRadius, specObjectThickness)
	if maskCount(unliftedIntoView) != 0 {
		t.Fatalf("fixture guard: %v is already visible before any lift (%d px); it cannot show a marker that becomes visible ONLY after the offset",
			intoView, maskCount(unliftedIntoView))
	}
	if maskCount(liftedIntoView) == 0 {
		t.Fatalf("fixture guard: %v is still invisible after its lift; the fixture does not reach the canvas", intoView)
	}

	unliftedOutOfView := paintLiftMask(imgW, imgH, []image.Point{outOfView}, cols, rows, cellpx, offsetY, nil, specObjectRadius, specObjectThickness)
	liftedOutOfView := paintLiftMask(imgW, imgH, []image.Point{outOfView}, cols, rows, cellpx, offsetY, lift, specObjectRadius, specObjectThickness)
	if maskCount(unliftedOutOfView) == 0 {
		t.Fatalf("fixture guard: %v is already invisible before any lift; it cannot show a marker that is dropped ONLY after the offset", outOfView)
	}
	if maskCount(liftedOutOfView) != 0 {
		t.Fatalf("fixture guard: %v is still visible after its lift (%d px); the fixture does not leave the canvas", outOfView, maskCount(liftedOutOfView))
	}

	want := paintLiftMask(imgW, imgH, cells, cols, rows, cellpx, offsetY, lift, specObjectRadius, specObjectThickness)
	if !masksEqual(want, liftedIntoView) {
		t.Fatalf("fixture: the combined oracle should equal the lifted intoView mask alone (outOfView contributes nothing after its own lift)")
	}

	marked := checkAgainstMask(t, "cull after offset", imgW, imgH, want, terrain.MarkerColor, func(img *image.RGBA) {
		terrain.DrawObjectMarkersAtHeights(img, cells, cols, rows, cellpx, offsetY, lift)
	})
	if marked == 0 {
		t.Errorf("marked 0 pixels, want %v's lifted cross to be drawn (SC-11: a marker visible only after its height offset must still be drawn)", intoView)
	}
	if marked != maskCount(want) {
		t.Errorf("marked %d pixels, want %d: %v must contribute nothing (it is visible only BEFORE its offset, and culling must run after it), and %v's whole lifted cross must survive",
			marked, maskCount(want), outOfView, intoView)
	}
}

// ---------------------------------------------------------------------------
// liftAt/liftY is read at most once per cell, not once per arm — so a caller's
// lift source (an AnchorHeight lookup, in T3) is not charged twice for a
// two-armed cross.

func TestMarkerLiftYReadOncePerCell(t *testing.T) {
	const cols, rows, cellpx, offsetY = 4, 3, 32, 0
	cells := []image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 3, Y: 2}}

	calls := map[image.Point]int{}
	lift := func(col, row int) int {
		calls[image.Point{X: col, Y: row}]++
		return 0
	}

	img := backgroundImage(cols*cellpx, rows*cellpx)
	terrain.DrawObjectMarkersAtHeights(img, cells, cols, rows, cellpx, offsetY, lift)

	if len(calls) != len(cells) {
		t.Fatalf("liftY was called for %d distinct cells, want exactly %d", len(calls), len(cells))
	}
	for _, c := range cells {
		if n := calls[c]; n != 1 {
			t.Errorf("cell %v: liftY called %d times, want exactly 1 (read once per cell, not once per arm)", c, n)
		}
	}
}

// TestMarkerLiftNeverMutatesTheBorrowedAltitudeSlice drives both *AtHeights
// entry points with a liftY closing over a real Projection, over in-map,
// off-map and out-of-range anchors alike, then byte-compares the exact slice
// the Projection was built over (never a copy this test kept back and never
// handed to it).
func TestMarkerLiftNeverMutatesTheBorrowedAltitudeSlice(t *testing.T) {
	const cols, rows, cellpx, offsetY = 5, 4, 32, 40
	alt := projRampAlt(cols, rows) // varies in both axes: no cell's lift is a trivial zero
	original := append([]uint8(nil), alt...)

	proj := terrain.Project(alt, cols, rows) // borrows alt directly, never a copy
	liftY := func(col, row int) int { return -proj.AnchorHeight(col, row) }

	cells := []image.Point{
		{X: 0, Y: 0}, {X: cols - 1, Y: 0}, {X: 0, Y: rows - 1},
		{X: cols - 1, Y: rows - 1}, {X: cols / 2, Y: rows / 2},
		{X: -1, Y: -1}, {X: cols, Y: rows}, // off-map anchors, probed too
	}

	imgW, imgH := cols*cellpx, rows*cellpx+offsetY

	object := backgroundImage(imgW, imgH)
	noPanicOverlay(t, "object draw", func() {
		terrain.DrawObjectMarkersAtHeights(object, cells, cols, rows, cellpx, offsetY, liftY)
	})

	unit := backgroundImage(imgW, imgH)
	noPanicOverlay(t, "unit draw", func() {
		terrain.DrawUnitMarkersAtHeights(unit, cells, cols, rows, cellpx, offsetY, liftY)
	})

	if !bytes.Equal(alt, original) {
		t.Fatalf("a real displaced draw through DrawObjectMarkersAtHeights/DrawUnitMarkersAtHeights "+
			"mutated the borrowed altitude slice: got %v, want %v (P-6)", alt, original)
	}
}
