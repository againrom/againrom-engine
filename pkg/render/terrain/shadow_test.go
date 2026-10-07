package terrain_test

import (
	"image"
	"image/color"
	"math"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

// TestShadowSlopeSweepAC1 - AC-1: the cycle-off sun's own shear ratio to the
// digit the spec calls out explicitly, and — swept over every minute of
// the cycle through SunAngle (0092/0100's own decoded schedule) — a slope
// that is never 0, never leaves [0.0333, 0.5774] in magnitude, and at
// minutes 360 and 1080 (where SunAngle is exactly 0) is the dead band's own
// edge value, +0.033345685, to within 1e-9. Dropping the clamp from
// ShadowAngle makes minute 360 give slope 0, which is what this sweep is
// for.
func TestShadowSlopeSweepAC1(t *testing.T) {
	const want = 0.57735025728078282
	if got := terrain.ShadowSlope(terrain.DefaultTheta); math.Abs(got-want) > 1e-15 {
		t.Fatalf("ShadowSlope(DefaultTheta) = %.17g, want %.17g within 1e-15 (AC-1)", got, want)
	}

	for m := uint64(0); m < 1440; m++ {
		theta := terrain.SunAngle(m, true)
		slope := terrain.ShadowSlope(theta)
		if slope == 0 {
			t.Fatalf("minute %d: ShadowSlope = 0, want never 0 (AC-1)", m)
		}
		if mag := math.Abs(slope); mag < 0.0333 || mag > 0.5774 {
			t.Errorf("minute %d: |ShadowSlope| = %v, want within [0.0333, 0.5774] (AC-1)", m, mag)
		}
		if m == 360 || m == 1080 {
			const wantEdge = 0.033345685
			if diff := math.Abs(slope - wantEdge); diff > 1e-9 {
				t.Errorf("minute %d (SunAngle exactly 0): ShadowSlope = %.9g, want %v within 1e-9 (AC-1, spec D-1)", m, slope, wantEdge)
			}
		}
	}
}

// TestShadowPivotShiftAC2 - AC-2: the pixel shift differs between two frame
// heights at one theta and between two anchors at one frame height, is 0 at
// 2*(frameH/2) == anchorY, and takes an odd frameH's own frameH-1 value
// rather than frameH's. A shift computed from theta alone collapses the
// first two checks to equal values.
func TestShadowPivotShiftAC2(t *testing.T) {
	const theta = 0.4

	if a, b := terrain.ShadowPivotShift(theta, 40, 10), terrain.ShadowPivotShift(theta, 60, 10); a == b {
		t.Errorf("ShadowPivotShift at frameH 40 and 60 (same anchor) both gave %d, want them to differ (AC-2)", a)
	}
	if a, b := terrain.ShadowPivotShift(theta, 40, 10), terrain.ShadowPivotShift(theta, 40, 30); a == b {
		t.Errorf("ShadowPivotShift at anchorY 10 and 30 (same frameH) both gave %d, want them to differ (AC-2)", a)
	}
	if got := terrain.ShadowPivotShift(theta, 40, 40); got != 0 {
		t.Errorf("ShadowPivotShift(theta, 40, 40) [2*(40/2)==40==anchorY] = %d, want 0 (AC-2)", got)
	}
	if got, want := terrain.ShadowPivotShift(theta, 41, 10), terrain.ShadowPivotShift(theta, 40, 10); got != want {
		t.Errorf("ShadowPivotShift(theta, 41, 10) = %d, want the frameH-1 value ShadowPivotShift(theta, 40, 10) = %d (AC-2)", got, want)
	}
}

// TestUnitShadowPlaceAC3 - AC-3's structural half: the shadow carries the
// body's cell, frame, anchor and mirror bit unchanged, and its top-left
// minus the body's is exactly (-ShadowPivotShift(theta, body.Frame.Height,
// body.Anchor.Y), airLift). A nil body.Frame does not panic.
func TestUnitShadowPlaceAC3(t *testing.T) {
	class := &terrain.UnitClass{Width: 64, Height: 96, CenterX: 30, CenterY: 80}
	frame := &terrain.StaticFrame{Width: 20, Height: 24, Pixels: make([]terrain.StaticPixel, 20*24)}

	body, ok := terrain.UnitPlace(3, 5, class, frame, true, 7, -19)
	if !ok {
		t.Fatal("UnitPlace refused a well-formed body")
	}

	const theta = 0.4
	const airLift = 11
	shadow := terrain.UnitShadowPlace(body, theta, airLift)

	if shadow.Cell != body.Cell {
		t.Errorf("shadow cell = %v, want the body's %v (FR-6)", shadow.Cell, body.Cell)
	}
	if shadow.Frame != body.Frame {
		t.Errorf("shadow frame = %p, want the body's own pointer %p (FR-6)", shadow.Frame, body.Frame)
	}
	if shadow.Anchor != body.Anchor {
		t.Errorf("shadow anchor = %v, want the body's %v (FR-6)", shadow.Anchor, body.Anchor)
	}
	if shadow.Mirror != body.Mirror {
		t.Errorf("shadow mirror = %v, want the body's %v (FR-6)", shadow.Mirror, body.Mirror)
	}

	wantShift := -terrain.ShadowPivotShift(theta, frame.Height, body.Anchor.Y)
	wantDiff := image.Pt(wantShift, airLift)
	if gotDiff := shadow.TopLeft.Sub(body.TopLeft); gotDiff != wantDiff {
		t.Errorf("shadow.TopLeft - body.TopLeft = %v, want %v (AC-3, FR-6)", gotDiff, wantDiff)
	}

	// A nil body.Frame: the shift is read from a zero height rather than
	// dereferenced.
	nilFrameBody := terrain.StaticPlacement{TopLeft: image.Pt(5, 5), Anchor: image.Pt(2, 3)}
	nilShadow := terrain.UnitShadowPlace(nilFrameBody, theta, 0)
	if want := -terrain.ShadowPivotShift(theta, 0, 3); nilShadow.TopLeft.X-nilFrameBody.TopLeft.X != want {
		t.Errorf("UnitShadowPlace with a nil body.Frame: X shift = %d, want %d (P-3)", nilShadow.TopLeft.X-nilFrameBody.TopLeft.X, want)
	}
}

// TestUnitShadowComposedLineAC3 - AC-3's numeric half: over a sweep of frame
// heights, anchors and day minutes, UnitShadowPlace's top-left composed with
// ShadowRowOffset at ShadowPivotRow's pivot is within 2 px of body.TopLeft.X
// + slope*(anchorY-row) at every sampled row, and that bound does not widen
// as the frame grows. Leaving the pivot at the frame's bottom edge (rather
// than ShadowPivotRow's own answer) fails this by tens of pixels, and so
// does a shift built from theta alone.
func TestUnitShadowComposedLineAC3(t *testing.T) {
	frameHeights := []int{10, 24, 41, 96, 200}
	anchors := []int{-5, 0, 7, 50, 199}
	minutes := []uint64{0, 90, 200, 360, 500, 720, 900, 1080, 1200, 1380}

	for _, fh := range frameHeights {
		frame := &terrain.StaticFrame{Width: 10, Height: fh, Pixels: make([]terrain.StaticPixel, 10*fh)}
		pivot := terrain.ShadowPivotRow(frame)
		for _, anchorY := range anchors {
			body := terrain.StaticPlacement{
				TopLeft: image.Pt(1000, 2000),
				Anchor:  image.Pt(3, anchorY),
				Frame:   frame,
			}
			for _, m := range minutes {
				theta := terrain.SunAt(m, true).Theta
				slope := terrain.ShadowSlope(theta)
				shadow := terrain.UnitShadowPlace(body, theta, 0)

				step := fh/8 + 1
				for row := 0; row < fh; row += step {
					gotX := shadow.TopLeft.X + terrain.ShadowRowOffset(slope, pivot, row)
					wantX := float64(body.TopLeft.X) + slope*float64(anchorY-row)
					if diff := math.Abs(float64(gotX) - wantX); diff > 2 {
						t.Errorf("frameH=%d anchorY=%d minute=%d row=%d: composed X=%d, want %v within 2px (AC-3, FR-6, FR-7)",
							fh, anchorY, m, row, gotX, wantX)
					}
				}
			}
		}
	}
}

func TestStructureShadowCompositionAC4(t *testing.T) {
	const (
		tileHeight = 2
		fullHeight = 3
		shadowY    = 40
		col0       = 0
		theta      = terrain.DefaultTheta
	)
	slope := terrain.ShadowSlope(theta)
	frame := &terrain.StaticFrame{Width: terrain.CellSize, Height: terrain.CellSize, Pixels: make([]terrain.StaticPixel, terrain.CellSize*terrain.CellSize)}
	pivot := terrain.ShadowPivotRow(frame)

	pivotWorldY := float64((0+tileHeight)*terrain.CellSize + terrain.CellSize - shadowY)

	type strip struct {
		gridRow int
		dstY    int
	}
	var strips []strip
	for row0 := 0; row0 < tileHeight; row0++ {
		rowTop := row0 - tileHeight + fullHeight
		limit := rowTop
		if row0 == 0 {
			limit = 0
		}
		for k := rowTop; k >= limit; k-- {
			dstY := row0*terrain.CellSize - (rowTop-k)*terrain.CellSize
			strips = append(strips, strip{gridRow: k, dstY: dstY})
		}
	}
	if len(strips) != fullHeight {
		t.Fatalf("hand-built strip list has %d strips, want FullHeight %d", len(strips), fullHeight)
	}

	const baseX = col0 * terrain.CellSize
	for _, s := range strips {
		shift := terrain.StructureShadowShift(theta, fullHeight, s.gridRow, shadowY)
		for row := 0; row < terrain.CellSize; row++ {
			worldY := s.dstY + row
			composedX := baseX + shift + terrain.ShadowRowOffset(slope, pivot, row)

			// The reference line is ONE truncation of the combined term, not
			// the raw continuous float: composedX itself is TWO independent
			// truncations (the strip shift's own and the row shear's own)
			// summed, and trunc(p)+trunc(q) can differ from trunc(p+q) by up
			// to 1 whenever the two fractional parts would otherwise carry —
			// that gap, not float noise, is the "within 1 px" AC-4 states.
			// Comparing against the untruncated float instead would demand a
			// bound near 2 px and this line would need to say so.
			wantX := baseX + int(slope*(pivotWorldY-float64(worldY)))
			if diff := composedX - wantX; diff > 1 || diff < -1 {
				t.Errorf("gridRow=%d row=%d worldY=%d: composed X=%d, want %d within 1px (AC-4, FR-8, FR-9)",
					s.gridRow, row, worldY, composedX, wantX)
			}
		}
	}

	// The per-strip term alone, with no row shear, leaves the join the spec
	// quotes: about 18.4 px between the two strips adjacent in world y
	// (gridRow 0 and gridRow 1 here) — which is exactly the gap the loop
	// above shows composing away.
	shift0 := terrain.StructureShadowShift(theta, fullHeight, 0, shadowY)
	shift1 := terrain.StructureShadowShift(theta, fullHeight, 1, shadowY)
	if diff := math.Abs(float64(shift0 - shift1)); diff < 15 {
		t.Errorf("per-strip term alone between gridRow 0 and 1 = %.2f px, want close to the spec's 18.4 (AC-4)", diff)
	}
}

// TestStructureShadowPlaceGuardsAC5 - AC-5: a VariableSize class, a
// classless placement, a frameless placement and a non-positive TileWidth
// all yield no structure shadow, while a well-formed placement's shadow
// keeps the body's Y, frame and class and displaces X by exactly
// StructureShadowShift's own term.
func TestStructureShadowPlaceGuardsAC5(t *testing.T) {
	const theta = 0.3
	frame := &terrain.StaticFrame{Width: terrain.CellSize, Height: terrain.CellSize, Pixels: make([]terrain.StaticPixel, terrain.CellSize*terrain.CellSize)}

	variable := &terrain.StructureClass{TileWidth: 2, TileHeight: 2, FullHeight: 3, ShadowY: 40, VariableSize: true}
	if _, ok := terrain.StructureShadowPlace(terrain.StructurePlacement{Class: variable, Frame: frame}, theta); ok {
		t.Error("StructureShadowPlace made a shadow for a VariableSize placement (AC-5)")
	}

	ordinary := &terrain.StructureClass{TileWidth: 2, TileHeight: 2, FullHeight: 3, ShadowY: 40}
	if _, ok := terrain.StructureShadowPlace(terrain.StructurePlacement{Frame: frame}, theta); ok {
		t.Error("StructureShadowPlace made a shadow for a placement with no class (AC-5)")
	}
	if _, ok := terrain.StructureShadowPlace(terrain.StructurePlacement{Class: ordinary}, theta); ok {
		t.Error("StructureShadowPlace made a shadow for a placement with no frame (AC-5)")
	}

	zeroWidth := &terrain.StructureClass{TileWidth: 0, TileHeight: 2, FullHeight: 3, ShadowY: 40}
	if _, ok := terrain.StructureShadowPlace(terrain.StructurePlacement{Class: zeroWidth, Frame: frame}, theta); ok {
		t.Error("StructureShadowPlace made a shadow for a class with TileWidth 0 (AC-5)")
	}
	negWidth := &terrain.StructureClass{TileWidth: -3, TileHeight: 2, FullHeight: 3, ShadowY: 40}
	if _, ok := terrain.StructureShadowPlace(terrain.StructurePlacement{Class: negWidth, Frame: frame}, theta); ok {
		t.Error("StructureShadowPlace made a shadow for a class with a negative TileWidth (AC-5)")
	}

	body := terrain.StructurePlacement{
		Class: ordinary, Frame: frame,
		TopLeft:   image.Pt(64, 96),
		GridIndex: 1*ordinary.TileWidth + 1,
	}
	shadow, ok := terrain.StructureShadowPlace(body, theta)
	if !ok {
		t.Fatal("StructureShadowPlace refused a well-formed placement")
	}
	if shadow.TopLeft.Y != body.TopLeft.Y {
		t.Errorf("shadow dstY = %d, want the body's own %d (FR-8)", shadow.TopLeft.Y, body.TopLeft.Y)
	}
	if got, want := shadow.TopLeft.X-body.TopLeft.X, terrain.StructureShadowShift(theta, ordinary.FullHeight, 1, ordinary.ShadowY); got != want {
		t.Errorf("shadow dstX - body dstX = %d, want %d (FR-8)", got, want)
	}
	if shadow.Frame != body.Frame || shadow.Class != body.Class {
		t.Error("shadow does not carry the same strip's frame and class (FR-8)")
	}
}

// TestStructureShadowShiftSuppressionAC6 - AC-6's numeric half: at every
// minute of the cycle and every strip of a swept range of FullHeight values,
// a class with ShadowY 20000 displaces by more than 650 world px and one
// with ShadowY 10000 by more than 300. "Every shipped FullHeight" is
// modelled as a plausible small range here — the suppression is arithmetic
// and holds on the footprint term alone, not on which FullHeight a real
// class happens to ship.
func TestStructureShadowShiftSuppressionAC6(t *testing.T) {
	for fullHeight := 1; fullHeight <= 12; fullHeight++ {
		for gridRow := 0; gridRow < fullHeight; gridRow++ {
			for m := uint64(0); m < 1440; m++ {
				theta := terrain.SunAt(m, true).Theta

				if got := terrain.StructureShadowShift(theta, fullHeight, gridRow, 20000); math.Abs(float64(got)) <= 650 {
					t.Fatalf("fullHeight=%d gridRow=%d minute=%d: |StructureShadowShift| with ShadowY 20000 = %d, want > 650 (AC-6, FR-11)",
						fullHeight, gridRow, m, got)
				}
				if got := terrain.StructureShadowShift(theta, fullHeight, gridRow, 10000); math.Abs(float64(got)) <= 300 {
					t.Fatalf("fullHeight=%d gridRow=%d minute=%d: |StructureShadowShift| with ShadowY 10000 = %d, want > 300 (AC-6, FR-11)",
						fullHeight, gridRow, m, got)
				}
			}
		}
	}
}

// TestStructureShadowNeverFloatsFreeOfItsCaster is the hotfix witness
// (docs/hotfix/LEDGER.md, DIV-068) for the owner's report of a ground
// symbol's shadow appearing "somewhere unrelated to it" at certain times of
// day. It sweeps the whole day-night cycle for both a normal-ShadowY class
// (28, 40, 55 — the shipped roster's own range, TERR-SHDW-136(c)) and the
// two suppression sentinels (10000, 20000): at every minute, a placement
// either draws no shadow at all or its shadow's TopLeft.X stays within
// bound of the body's own — 200 world px, six cells, comfortably above any
// real single-strip lean (TestStructureShadowShiftSuppressionAC6's own
// fixture never separates a normal class by more than a few tens of pixels)
// and comfortably below the smallest displacement a sentinel ShadowY can
// produce at ANY minute (300+/650+, AC-6).
//
// Before this hotfix StructureShadowPlace had no ShadowY guard at all: a
// sentinel class's shadow displaced by StructureShadowShift's own
// arithmetic and drew wherever that landed, which — measured with the real
// Viewer/Camera pair and no game data (docs/hotfix/LEDGER.md, DIV-068) —
// a resizable window the size of the original's own largest shipped
// resolution (1024x768, SESS-VIEW-028) or larger can bring back on screen
// near the sun's daily and nightly zero-crossings, floating apart from any
// caster.
func TestStructureShadowNeverFloatsFreeOfItsCaster(t *testing.T) {
	const bound = 200 // world px, six cells
	frame := &terrain.StaticFrame{Width: terrain.CellSize, Height: terrain.CellSize, Pixels: make([]terrain.StaticPixel, terrain.CellSize*terrain.CellSize)}
	body := image.Pt(64, 96)

	for _, shadowY := range []int{28, 40, 55, 10000, 20000} {
		class := &terrain.StructureClass{TileWidth: 1, TileHeight: 1, FullHeight: 1, ShadowY: shadowY}
		placement := terrain.StructurePlacement{Class: class, Frame: frame, TopLeft: body}

		for m := uint64(0); m < 1440; m++ {
			theta := terrain.SunAt(m, true).Theta
			shadow, ok := terrain.StructureShadowPlace(placement, theta)
			if !ok {
				continue // no shadow at all is within any bound of its caster
			}
			gap := shadow.TopLeft.X - body.X
			if gap < 0 {
				gap = -gap
			}
			if gap > bound {
				t.Fatalf("ShadowY=%d minute=%d theta=%.6f: shadow separates %d px (%.1f cells) from its own caster, want <= %d px (%.1f cells)",
					shadowY, m, theta, gap, float64(gap)/terrain.CellSize, bound, float64(bound)/terrain.CellSize)
			}
		}
	}
}

// TestObjectShadowPlaceAC7 - AC-7: an object whose drawn frame differs in
// size from frame 0 anchors at frame 0 and takes frame 0's own pixel-shift
// displacement, while its silhouette (Frame, Class) and its shear pivot
// (ShadowPivotRow of the returned Frame) stay the drawn frame's; a placement
// with no class, no drawn frame or no frame 0 yields none.
func TestObjectShadowPlaceAC7(t *testing.T) {
	frame0 := &terrain.StaticFrame{Width: 20, Height: 24, Pixels: make([]terrain.StaticPixel, 20*24)}
	drawn := &terrain.StaticFrame{Width: 33, Height: 41, Pixels: make([]terrain.StaticPixel, 33*41)}
	class := &terrain.StaticClass{Width: 64, Height: 96, CenterX: 30, CenterY: 80, Frames: []*terrain.StaticFrame{frame0, drawn}}

	const col, row = 3, 5
	const lift, originY = 7, -19
	const theta = 0.4

	bodyDestX, bodyDestY, bodyAnchorX, bodyAnchorY := terrain.StaticAnchor(
		col, row, class.Width, class.Height, class.CenterX, class.CenterY, drawn.Width, drawn.Height, lift, originY)
	body := terrain.StaticPlacement{
		Cell:    image.Pt(col, row),
		TopLeft: image.Pt(bodyDestX, bodyDestY),
		Anchor:  image.Pt(bodyAnchorX, bodyAnchorY),
		Frame:   drawn,
		Class:   class,
	}

	shadow, ok := terrain.ObjectShadowPlace(body, theta, originY)
	if !ok {
		t.Fatal("ObjectShadowPlace refused a well-formed placement")
	}

	wantDestX, wantDestY, wantAnchorX, wantAnchorY := terrain.StaticAnchor(
		col, row, class.Width, class.Height, class.CenterX, class.CenterY, frame0.Width, frame0.Height, lift, originY)
	wantX := wantDestX - terrain.ShadowPivotShift(theta, frame0.Height, wantAnchorY)

	if want := (image.Point{X: wantX, Y: wantDestY}); shadow.TopLeft != want {
		t.Errorf("shadow.TopLeft = %v, want %v (FR-12, FR-13)", shadow.TopLeft, want)
	}
	if want := (image.Point{X: wantAnchorX, Y: wantAnchorY}); shadow.Anchor != want {
		t.Errorf("shadow.Anchor = %v, want the frame-0 anchor %v (FR-13)", shadow.Anchor, want)
	}
	if shadow.Cell != body.Cell || shadow.Frame != body.Frame || shadow.Class != body.Class {
		t.Error("ObjectShadowPlace moved the cell or changed the frame/class — the silhouette must stay the drawn frame's (FR-12)")
	}
	if got, want := terrain.ShadowPivotRow(shadow.Frame), drawn.Height; got != want {
		t.Errorf("ShadowPivotRow(shadow.Frame) = %d, want the drawn frame's own height %d — the shear pivot is the drawn frame's (FR-12)", got, want)
	}

	// No class, no drawn frame, or no frame 0 all refuse rather than panic.
	if _, ok := terrain.ObjectShadowPlace(terrain.StaticPlacement{Frame: drawn}, theta, originY); ok {
		t.Error("ObjectShadowPlace made a shadow for a placement with no class")
	}
	if _, ok := terrain.ObjectShadowPlace(terrain.StaticPlacement{Class: class}, theta, originY); ok {
		t.Error("ObjectShadowPlace made a shadow for a placement with no frame")
	}
	if _, ok := terrain.ObjectShadowPlace(terrain.StaticPlacement{Class: &terrain.StaticClass{}, Frame: drawn}, theta, originY); ok {
		t.Error("ObjectShadowPlace made a shadow for a class with no frame 0")
	}
}

// TestGeometryTotalityAC9 - AC-9's first half: every geometry function named
// in the spec returns rather than panics for a nil frame, a nil class, a
// negative frame size, a negative grid row, and theta of 1e9 and -1e9.
func TestGeometryTotalityAC9(t *testing.T) {
	thetas := []float64{1e9, -1e9}

	// ShadowAngle, ShadowSlope, ShadowShear16: total over any theta.
	for _, theta := range thetas {
		_ = terrain.ShadowAngle(theta)
		_ = terrain.ShadowSlope(theta)
		_ = terrain.ShadowShear16(theta)
	}

	// ShadowRowOffset: any slope and any pivot/row, negative included.
	_ = terrain.ShadowRowOffset(0.5, -100, -50)

	// ShadowPivotRow: a nil frame, and a frame with a negative height.
	if got := terrain.ShadowPivotRow(nil); got != 0 {
		t.Errorf("ShadowPivotRow(nil) = %d, want 0 (P-3)", got)
	}
	if got := terrain.ShadowPivotRow(&terrain.StaticFrame{Height: -10}); got != -10 {
		t.Errorf("ShadowPivotRow of a negative-height frame = %d, want -10 (P-3)", got)
	}

	// ShadowPivotShift: a negative frame size and a negative anchor, at the
	// extreme thetas.
	for _, theta := range thetas {
		_ = terrain.ShadowPivotShift(theta, -50, -20)
	}

	// UnitShadowPlace: a body with a nil Frame, at the extreme thetas.
	nilBody := terrain.StaticPlacement{}
	for _, theta := range thetas {
		_ = terrain.UnitShadowPlace(nilBody, theta, 0)
	}

	// StructureShadowShift: a negative fullHeight and a negative gridRow, at
	// the extreme thetas.
	for _, theta := range thetas {
		_ = terrain.StructureShadowShift(theta, -5, -3, 40)
	}

	// StructureShadowPlace: the zero placement (nil class), a negative
	// TileWidth (excluded), and a negative GridIndex on an otherwise
	// well-formed class (accepted, exercising a negative gridRow), at the
	// extreme thetas.
	negWidth := &terrain.StructureClass{TileWidth: -1, TileHeight: 2, FullHeight: 3}
	validStructure := &terrain.StructureClass{TileWidth: 2, TileHeight: 2, FullHeight: 3, ShadowY: 40}
	frame1x1 := &terrain.StaticFrame{Width: 1, Height: 1, Pixels: make([]terrain.StaticPixel, 1)}
	for _, theta := range thetas {
		if _, ok := terrain.StructureShadowPlace(terrain.StructurePlacement{}, theta); ok {
			t.Error("StructureShadowPlace made a shadow for the zero placement")
		}
		if _, ok := terrain.StructureShadowPlace(terrain.StructurePlacement{Class: negWidth, Frame: frame1x1}, theta); ok {
			t.Error("StructureShadowPlace made a shadow for a negative-TileWidth class")
		}
		if _, ok := terrain.StructureShadowPlace(terrain.StructurePlacement{Class: validStructure, Frame: frame1x1, GridIndex: -7}, theta); !ok {
			t.Error("StructureShadowPlace refused a well-formed placement with a negative GridIndex")
		}
	}

	// ObjectShadowPlace: the zero placement (nil class), and a well-formed
	// one, at the extreme thetas.
	objFrame := &terrain.StaticFrame{Width: 5, Height: 5, Pixels: make([]terrain.StaticPixel, 25)}
	objClass := &terrain.StaticClass{Width: 10, Height: 10, Frames: []*terrain.StaticFrame{objFrame}}
	objBody := terrain.StaticPlacement{Class: objClass, Frame: objFrame}
	for _, theta := range thetas {
		if _, ok := terrain.ObjectShadowPlace(terrain.StaticPlacement{}, theta, 0); ok {
			t.Error("ObjectShadowPlace made a shadow for the zero placement")
		}
		if _, ok := terrain.ObjectShadowPlace(objBody, theta, 0); !ok {
			t.Errorf("ObjectShadowPlace refused a well-formed placement at theta=%v", theta)
		}
	}
}

// TestBlitShadowAC4 covers the recolour law's own AC-4 (docs/0101-shadows),
// unchanged by this story.
func TestBlitShadowAC4(t *testing.T) {
	frameA := &terrain.StaticFrame{Width: 2, Height: 2, Pixels: []terrain.StaticPixel{
		{Index: 3, Opaque: true}, {Opaque: false},
		{Index: 3, Opaque: true}, {Index: 3, Opaque: true},
	}}
	frameB := &terrain.StaticFrame{Width: 2, Height: 2, Pixels: []terrain.StaticPixel{
		{Index: 200, Opaque: true}, {Opaque: false},
		{Index: 200, Opaque: true}, {Index: 200, Opaque: true},
	}}
	frameA.Palette[3] = color.RGBA{R: 10, G: 20, B: 30, A: 255}
	frameB.Palette[200] = color.RGBA{R: 250, G: 240, B: 230, A: 255}

	solid := func(topLeft, rest color.RGBA) *image.RGBA {
		dst := image.NewRGBA(image.Rect(0, 0, 2, 2))
		for y := 0; y < 2; y++ {
			for x := 0; x < 2; x++ {
				c := rest
				if x == 0 && y == 0 {
					c = topLeft
				}
				dst.SetRGBA(x, y, c)
			}
		}
		return dst
	}

	// Part 1: two destinations differing in exactly the (0,0) pixel — the one
	// covered by an opaque frame pixel — give two results still differing
	// there, and agreeing everywhere the sources already agreed.
	dst1 := solid(color.RGBA{R: 0, G: 0, B: 0, A: 255}, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	dst2 := solid(color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	terrain.BlitShadow(dst1, frameA, 0, 0, 8, 0, 0, false)
	terrain.BlitShadow(dst2, frameA, 0, 0, 8, 0, 0, false)
	if dst1.RGBAAt(0, 0) == dst2.RGBAAt(0, 0) {
		t.Errorf("the differing pixel came out equal after BlitShadow: %v vs %v (AC-4)", dst1.RGBAAt(0, 0), dst2.RGBAAt(0, 0))
	}
	if got, want := dst1.RGBAAt(0, 0), (color.RGBA{R: 0, G: 0, B: 0, A: 255}); got != want {
		t.Errorf("black under a level-8 shadow = %v, want %v (AC-4, FR-13)", got, want)
	}
	if got, want := dst2.RGBAAt(0, 0), (color.RGBA{R: 127, G: 127, B: 127, A: 255}); got != want {
		t.Errorf("white under a level-8 shadow = %v, want %v (AC-4, FR-13)", got, want)
	}
	if got, want := dst1.RGBAAt(1, 1), (color.RGBA{R: 64, G: 64, B: 64, A: 255}); got != want {
		t.Errorf("grey 128 under a level-8 shadow = %v, want %v (AC-4, FR-13)", got, want)
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			if x == 0 && y == 0 {
				continue
			}
			if dst1.RGBAAt(x, y) != dst2.RGBAAt(x, y) {
				t.Errorf("pixel (%d,%d) diverged though the sources agreed there: %v vs %v", x, y, dst1.RGBAAt(x, y), dst2.RGBAAt(x, y))
			}
		}
	}

	// Part 2: one destination, two frames sharing an opaque mask but not a
	// palette, give the identical result.
	dstA := solid(color.RGBA{R: 40, G: 60, B: 80, A: 255}, color.RGBA{R: 10, G: 200, B: 90, A: 255})
	dstB := solid(color.RGBA{R: 40, G: 60, B: 80, A: 255}, color.RGBA{R: 10, G: 200, B: 90, A: 255})
	terrain.BlitShadow(dstA, frameA, 0, 0, 8, 0, 0, false)
	terrain.BlitShadow(dstB, frameB, 0, 0, 8, 0, 0, false)
	if !reflect.DeepEqual(dstA.Pix, dstB.Pix) {
		t.Errorf("two frames sharing an opaque mask but not a palette gave different results over one background (AC-4)")
	}
}

func TestShadowChannelAC5(t *testing.T) {
	for level := 0; level <= 16; level++ {
		for in := 0; in <= 255; in++ {
			want := uint8((in * (16 - level)) >> 4)
			if got := terrain.ShadowChannel(uint8(in), level); got != want {
				t.Fatalf("ShadowChannel(%d, %d) = %d, want %d (FR-13, AC-5)", in, level, got, want)
			}
		}
	}

	for level := 0; level <= 16; level++ {
		alpha := terrain.ShadowAlpha(level)
		for in := 0; in <= 255; in++ {
			exact := float64(terrain.ShadowChannel(uint8(in), level))
			approx := math.Round(float64(in) * (1 - float64(alpha)/255))
			if diff := math.Abs(approx - exact); diff > 1 {
				t.Fatalf("level %d, in %d: blend approx %v vs exact %v differ by more than 1 (AC-5, D-4)", level, in, approx, exact)
			}
		}
	}
}

func TestShadowMask(t *testing.T) {
	pixels := []terrain.StaticPixel{
		{Opaque: true}, {Opaque: false}, {Opaque: true},
		{Opaque: false}, {Opaque: true}, {Opaque: false},
	}
	f := &terrain.StaticFrame{Width: 3, Height: 2, Pixels: pixels}
	mask := terrain.ShadowMask(f)

	if want := image.Rect(0, 0, 3, 2); mask.Bounds() != want {
		t.Fatalf("mask bounds = %v, want %v", mask.Bounds(), want)
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			wantOpaque := pixels[y*3+x].Opaque
			c := mask.RGBAAt(x, y)
			if gotOpaque := c.A == 0xff; gotOpaque != wantOpaque {
				t.Errorf("mask(%d,%d).A = %d, want opaque=%v", x, y, c.A, wantOpaque)
			}
			switch {
			case wantOpaque && c != (color.RGBA{A: 0xff}):
				t.Errorf("mask(%d,%d) opaque pixel = %v, want opaque black", x, y, c)
			case !wantOpaque && c != (color.RGBA{}):
				t.Errorf("mask(%d,%d) transparent pixel = %v, want the zero colour", x, y, c)
			}
		}
	}

	// Totality: a nil frame and one with no area yield an empty image rather
	// than a panic, mirroring StaticFrame.RGBA's own total shape.
	if got := terrain.ShadowMask(nil); got.Bounds() != (image.Rectangle{}) {
		t.Errorf("ShadowMask(nil) bounds = %v, want the zero rectangle", got.Bounds())
	}
	if got := terrain.ShadowMask(&terrain.StaticFrame{}); got.Bounds() != (image.Rectangle{}) {
		t.Errorf("ShadowMask of a frame with no area, bounds = %v, want the zero rectangle", got.Bounds())
	}
}

func TestShadowLevelPairing(t *testing.T) {
	lt := terrain.Light{ShroudUnit: 3, ShroudObject: 6}
	if got := terrain.ShadowLevel(lt, terrain.ShadowUnit); got != 3 {
		t.Errorf("ShadowLevel(unit) = %d, want ShroudUnit 3 (FR-14)", got)
	}
	if got := terrain.ShadowLevel(lt, terrain.ShadowObject); got != 6 {
		t.Errorf("ShadowLevel(object) = %d, want ShroudObject 6 (FR-14)", got)
	}
}
