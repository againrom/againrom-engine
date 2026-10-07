package terrain_test

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// ---------------------------------------------------------------------------
// The SC-5 fixture.

const (
	staticMarkerCols = 4
	staticMarkerRows = 3

	// A NEGATIVE origin is the ordinary case, not the exotic one: minV is the
	// least row*32 - h(c,row) over the mesh, so any map whose top row carries a
	// positive altitude reports one.
	staticMarkerOriginY = -23
)

// staticMarkerLift is the fixture's height source: asymmetric in col and row,
// distinct at every cell, and BOTH SIGNS over the grid — cell (0,2) lifts by -6
// while cell (2,2) on the same row lifts by +4. A derivation that read one
// coordinate for the other, applied one cell's lift to another, or dropped the
// sign lands on a different number at some placement.
func staticMarkerLift(col, row int) int { return 5*col - 4*row + 2 }

// staticMarkerSet builds the fixture's classes fresh on every call — three
// resolving bytes, one of them artless, and every other byte naming no class.
//
// The two drawable classes are deliberately shaped so that no shortcut agrees
// with the contract by accident: odd canvases and odd frames (so every /2
// truncates), a frame SMALLER than its canvas and a frame LARGER than it (so an
// anchor clamped to the canvas could not place the second), and centres that are
// not the canvas centre.
func staticMarkerSet() *terrain.StaticSet {
	var set terrain.StaticSet

	// Byte 1: canvas 65x97, centre (33,81), frame 21x25 — smaller than canvas.
	//   anchorX = 33 - 65/2 + 21/2 = 33 - 32 + 10 = 11
	//   anchorY = 81 - 97/2 + 25/2 = 81 - 48 + 12 = 45
	set.Classes[1] = &terrain.StaticClass{
		Width: 65, Height: 97, CenterX: 33, CenterY: 81,
		Frame: &terrain.StaticFrame{Width: 21, Height: 25, Pixels: make([]terrain.StaticPixel, 21*25)},
	}

	// Byte 2: canvas 33x33, centre (17,31), frame 97x129 — larger than canvas.
	//   anchorX = 17 - 33/2 +  97/2 = 17 - 16 + 48 = 49
	//   anchorY = 31 - 33/2 + 129/2 = 31 - 16 + 64 = 79
	set.Classes[2] = &terrain.StaticClass{
		Width: 33, Height: 33, CenterX: 17, CenterY: 31,
		Frame: &terrain.StaticFrame{Width: 97, Height: 129, Pixels: make([]terrain.StaticPixel, 97*129)},
	}

	// Byte 3: resolved but artless — a skip, and never a placement.
	set.Classes[3] = &terrain.StaticClass{Width: 40, Height: 40, CenterX: 20, CenterY: 38}

	return &set
}

// staticMarkerGrid is the fixture map: five cells that place, a byte naming no
// loaded class, a class with no drawable frame, and zeros elsewhere.
//
//	row 0:  .  1  .  2
//	row 1:  7  2  .  3
//	row 2:  1  .  1  .
func staticMarkerGrid() terrain.Grid {
	return terrain.Grid{
		Width:  staticMarkerCols,
		Height: staticMarkerRows,
		Tiles:  make([]uint16, staticMarkerCols*staticMarkerRows),
		Overlay: []uint8{
			0, 1, 0, 2,
			7, 2, 0, 3,
			1, 0, 1, 0,
		},
	}
}

// staticMarkerWant is one placement's two hand-computed sides.
type staticMarkerWant struct {
	cell    image.Point
	topLeft image.Point // the SPRITE side's top-left, from "Sprite anchor"
	anchor  image.Point // the frame's own anchor pixel
	sprite  image.Point // topLeft + anchor: what Ground() must sum to
	marker  image.Point //
}

// The FLAT geometry: lift is nil (0 everywhere) and originY is 0, so both sides
// reduce to the cell's centre plus their own anchor arithmetic.
//
// Cell centres are (col*32+16, row*32+16).
//
//	(1,0) b1  centre ( 48, 16)  anchor (11,45)  topLeft = ( 48-11,  16-45) = ( 37,-29)
//	(3,0) b2  centre (112, 16)  anchor (49,79)  topLeft = (112-49,  16-79) = ( 63,-63)
//	(1,1) b2  centre ( 48, 48)  anchor (49,79)  topLeft = ( 48-49,  48-79) = ( -1,-31)
//	(0,2) b1  centre ( 16, 80)  anchor (11,45)  topLeft = ( 16-11,  80-45) = (  5, 35)
//	(2,2) b1  centre ( 80, 80)  anchor (11,45)  topLeft = ( 80-11,  80-45) = ( 69, 35)
//
// The marker column is computed the other way round, from the cell alone:
// MarkerAnchor(col, row, 32, 0, 0) = (col*32+16, row*32+16).
var staticMarkerFlat = []staticMarkerWant{
	{cell: image.Pt(1, 0), topLeft: image.Pt(37, -29), anchor: image.Pt(11, 45), sprite: image.Pt(48, 16), marker: image.Pt(48, 16)},
	{cell: image.Pt(3, 0), topLeft: image.Pt(63, -63), anchor: image.Pt(49, 79), sprite: image.Pt(112, 16), marker: image.Pt(112, 16)},
	{cell: image.Pt(1, 1), topLeft: image.Pt(-1, -31), anchor: image.Pt(49, 79), sprite: image.Pt(48, 48), marker: image.Pt(48, 48)},
	{cell: image.Pt(0, 2), topLeft: image.Pt(5, 35), anchor: image.Pt(11, 45), sprite: image.Pt(16, 80), marker: image.Pt(16, 80)},
	{cell: image.Pt(2, 2), topLeft: image.Pt(69, 35), anchor: image.Pt(11, 45), sprite: image.Pt(80, 80), marker: image.Pt(80, 80)},
}

// The DISPLACED geometry: lift = staticMarkerLift(col,row) and originY = -23.
//
// Sprite side, destY = row*32 + 16 - anchorY - lift - originY, i.e. + 23 here:
//
//	(1,0) lift  5*1-4*0+2 =  7   destY =  16 - 45 -  7 + 23 = -13   ground.Y = -13+45 =  32
//	(3,0) lift  5*3-4*0+2 = 17   destY =  16 - 79 - 17 + 23 = -57   ground.Y = -57+79 =  22
//	(1,1) lift  5*1-4*1+2 =  3   destY =  48 - 79 -  3 + 23 = -11   ground.Y = -11+79 =  68
//	(0,2) lift  5*0-4*2+2 = -6   destY =  80 - 45 +  6 + 23 =  64   ground.Y =  64+45 = 109
//	(2,2) lift  5*2-4*2+2 =  4   destY =  80 - 45 -  4 + 23 =  54   ground.Y =  54+45 =  99
//
// destX carries neither term, so every top-left X is the flat one.
//
// Marker side, MarkerAnchor(col, row, 32, -originY = +23, -lift):
//
//	(1,0) y =  0*32+16 + 23 -  7 =  32      (3,0) y =  0*32+16 + 23 - 17 =  22
//	(1,1) y =  1*32+16 + 23 -  3 =  68      (0,2) y =  2*32+16 + 23 + 6  = 109
//	(2,2) y =  2*32+16 + 23 -  4 =  99
var staticMarkerDisplaced = []staticMarkerWant{
	{cell: image.Pt(1, 0), topLeft: image.Pt(37, -13), anchor: image.Pt(11, 45), sprite: image.Pt(48, 32), marker: image.Pt(48, 32)},
	{cell: image.Pt(3, 0), topLeft: image.Pt(63, -57), anchor: image.Pt(49, 79), sprite: image.Pt(112, 22), marker: image.Pt(112, 22)},
	{cell: image.Pt(1, 1), topLeft: image.Pt(-1, -11), anchor: image.Pt(49, 79), sprite: image.Pt(48, 68), marker: image.Pt(48, 68)},
	{cell: image.Pt(0, 2), topLeft: image.Pt(5, 64), anchor: image.Pt(11, 45), sprite: image.Pt(16, 109), marker: image.Pt(16, 109)},
	{cell: image.Pt(2, 2), topLeft: image.Pt(69, 54), anchor: image.Pt(11, 45), sprite: image.Pt(80, 99), marker: image.Pt(80, 99)},
}

// staticMarkerLists builds the fixture's two placement lists, flat then
// displaced, and checks the census both share.
func staticMarkerLists(t *testing.T) (flat, displaced []terrain.StaticPlacement) {
	t.Helper()

	g, set := staticMarkerGrid(), staticMarkerSet()
	flat, flatCounts, _ := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateTiles)
	displaced, dispCounts, _ := terrain.StaticPlacements(g, set, staticMarkerLift, staticMarkerOriginY, terrain.AnimGateTiles)

	want := terrain.StaticCounts{Placed: 5, NoClass: 1, NoFrame: 1}
	if flatCounts != want || dispCounts != want {
		t.Fatalf("fixture: census flat %+v, displaced %+v, want %+v both — five placing cells, byte 7 naming no class, byte 3 artless",
			flatCounts, dispCounts, want)
	}
	return flat, displaced
}

func staticMarkerPoint(col, row, originY int, lift func(col, row int) int) image.Point {
	ly := 0
	if lift != nil {
		ly = lift(col, row)
	}
	x, y := terrain.MarkerAnchor(col, row, terrain.CellSize, -originY, -ly)
	return image.Pt(x, y)
}

// staticMarkerAgree compares each placement's Ground() with a marker-derived
// point for the same cell and returns how many agreed and how many did not.
//
// marker is a FUNCTION OF THE CELL ALONE — it is handed the cell and nothing
// from the placement, so no perturbation below can leak one side's answer into
// the other's expectation.
func staticMarkerAgree(ps []terrain.StaticPlacement, marker func(col, row int) image.Point) (agree, disagree int) {
	for _, p := range ps {
		if p.Ground() == marker(p.Cell.X, p.Cell.Y) {
			agree++
			continue
		}
		disagree++
	}
	return agree, disagree
}

// ---------------------------------------------------------------------------
// SC-5, the measurement itself.

// TestStaticMarkerGroundPointAgreement covers SC-5's first half (AC-8): in
// both geometries every placement's sprite ground point equals the marker
// geometry's own point for its cell.
//
// Each side is asserted against its OWN hand-computed literal before the two are
// compared with each other, so the agreement cannot be an artefact of one
// derivation having been used to predict the other.
func TestStaticMarkerGroundPointAgreement(t *testing.T) {
	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; every literal in this file is stated over 32-pixel cells", terrain.CellSize)
	}

	flat, displaced := staticMarkerLists(t)

	cases := []struct {
		geometry string
		ps       []terrain.StaticPlacement
		want     []staticMarkerWant
		originY  int
		lift     func(col, row int) int
	}{
		{"flat", flat, staticMarkerFlat, 0, nil},
		{"displaced", displaced, staticMarkerDisplaced, staticMarkerOriginY, staticMarkerLift},
	}

	for _, tc := range cases {
		if len(tc.ps) != len(tc.want) {
			t.Fatalf("%s: %d placements, want %d", tc.geometry, len(tc.ps), len(tc.want))
		}
		for i, w := range tc.want {
			p := tc.ps[i]
			if p.Cell != w.cell {
				t.Fatalf("%s: placement %d is for cell %v, want %v — the list is row-major", tc.geometry, i, p.Cell, w.cell)
			}

			// (a) The SPRITE side against its own literals. TopLeft and Anchor are
			//     what was DRAWN; Ground() must be their sum and nothing else.
			if p.TopLeft != w.topLeft || p.Anchor != w.anchor {
				t.Errorf("%s: cell %v drew at top-left %v with anchor %v, want %v and %v — hand-computed from the Sprite anchor formula",
					tc.geometry, w.cell, p.TopLeft, p.Anchor, w.topLeft, w.anchor)
			}
			if g := p.Ground(); g != w.sprite {
				t.Errorf("%s: cell %v Ground() = %v, want %v = %v + %v — the sum of the two values the placement carries",
					tc.geometry, w.cell, g, w.sprite, w.topLeft, w.anchor)
			}

			// (b) The MARKER side against its own literals, from the cell alone.
			m := staticMarkerPoint(w.cell.X, w.cell.Y, tc.originY, tc.lift)
			if m != w.marker {
				t.Errorf("%s: cell %v marker anchor = %v, want %v — hand-computed from col*32+16, row*32+16+offsetY+liftY",
					tc.geometry, w.cell, m, w.marker)
			}

			// (c) Only now the two against each other. Both have already been
			//     pinned to independent literals, so this line measures the
			//     contract rather than restating the two above.
			if got := p.Ground(); got != m {
				t.Errorf("%s: cell %v — the sprite stands its ground point at %v, the marker geometry puts the cell's cross at %v (P-4: the two derivations must agree)",
					tc.geometry, w.cell, got, m)
			}
		}

		agree, disagree := staticMarkerAgree(tc.ps, func(col, row int) image.Point {
			return staticMarkerPoint(col, row, tc.originY, tc.lift)
		})
		t.Logf("SC-5 %s geometry: %d placements agree, %d disagree", tc.geometry, agree, disagree)
		if disagree != 0 || agree != len(tc.want) {
			t.Errorf("%s: %d of %d placements disagree with the marker geometry (AC-8, P-4)", tc.geometry, disagree, len(tc.ps))
		}
	}
}

// TestStaticMarkerPerturbations covers SC-5's second half: the BOUND on what the
// agreement discriminates, measured rather than asserted.
//
//	PT-1  the marker's cell-to-world mapping loses its half-cell centring
//	PT-2  the marker's lift takes the wrong sign
//	PT-3  every class's CenterX moves by +7
//	PT-4  every class's canvas and frame dimensions are exchanged
//
// PT-1 and PT-2 must FAIL the comparison; PT-3 and PT-4 must leave it passing —
// and the last two carry a guard proving the perturbation genuinely moved the
// drawn sprite, or "it still passes" would mean nothing.
func TestStaticMarkerPerturbations(t *testing.T) {
	flat, displaced := staticMarkerLists(t)

	geometries := []struct {
		name    string
		ps      []terrain.StaticPlacement
		originY int
		lift    func(col, row int) int
	}{
		{"flat", flat, 0, nil},
		{"displaced", displaced, staticMarkerOriginY, staticMarkerLift},
	}

	// ---- PT-1: the cell-to-world mapping. The marker side places a cell at the
	//      TOP-LEFT CORNER of its lattice square instead of its centre — the
	//      +cellpx/2 dropped on both axes, which is the canonical cell-to-world
	//      mistake. Written out here rather than obtained by perturbing an
	//      argument, so the wrong mapping is visibly a mapping and not a lookup.
	for _, g := range geometries {
		wrong := func(col, row int) image.Point {
			ly := 0
			if g.lift != nil {
				ly = g.lift(col, row)
			}
			return image.Pt(col*terrain.CellSize, row*terrain.CellSize-g.originY-ly)
		}
		agree, disagree := staticMarkerAgree(g.ps, wrong)
		t.Logf("PT-1 (cell-to-world mapping, half-cell centring dropped), %s geometry: %d agree, %d disagree — MUST disagree",
			g.name, agree, disagree)
		if agree != 0 {
			t.Errorf("PT-1 %s: %d placement(s) still agreed with a marker geometry that maps a cell to its corner instead of its centre; SC-5 requires this perturbation to fail the comparison",
				g.name, agree)
		}
	}

	// ---- PT-2: the lift's sign. The marker side is handed +lift where the
	//      convention is -lift, the origin term left correct so the sign is the
	//      only difference.
	//
	//      In the FLAT geometry this perturbation is provably a no-op and cannot
	//      be otherwise: lift is 0 at every cell, so there is no sign to get
	//      wrong. That is reported rather than hidden — a flat render simply
	//      cannot witness this half of the bound.
	for _, g := range geometries {
		wrong := func(col, row int) image.Point {
			ly := 0
			if g.lift != nil {
				ly = g.lift(col, row)
			}
			x, y := terrain.MarkerAnchor(col, row, terrain.CellSize, -g.originY, +ly) // the trap: +ly
			return image.Pt(x, y)
		}
		agree, disagree := staticMarkerAgree(g.ps, wrong)
		if g.lift == nil {
			t.Logf("PT-2 (lift sign flipped), %s geometry: %d agree, %d disagree — VACUOUS here, the lift is 0 at every cell so the sign is unobservable",
				g.name, agree, disagree)
			if disagree != 0 {
				t.Errorf("PT-2 %s: flipping the sign of a lift that is 0 everywhere changed %d placement(s); the fixture is not the flat geometry it claims to be",
					g.name, disagree)
			}
			continue
		}
		t.Logf("PT-2 (lift sign flipped), %s geometry: %d agree, %d disagree — MUST disagree", g.name, agree, disagree)
		if agree != 0 {
			t.Errorf("PT-2 %s: %d placement(s) still agreed after the marker lift's sign was flipped; SC-5 requires this perturbation to fail the comparison",
				g.name, agree)
		}
	}

	// ---- PT-3 and PT-4 perturb the SPRITE side's class data, rebuild both
	//      lists, and re-run the same comparison against the UNCHANGED marker
	//      geometry. Both must still pass — and each carries the guard that says
	//      the perturbation was not a no-op.
	perturbSet := []struct {
		name  string
		claim string
		apply func(*terrain.StaticSet)
	}{
		{
			"PT-3 (CenterX + 7 on every class)",
			"a wrong CenterX convention",
			func(s *terrain.StaticSet) {
				for _, c := range s.Classes {
					if c != nil {
						c.CenterX += 7
					}
				}
			},
		},
		{
			"PT-4 (canvas and frame dimensions exchanged)",
			"a canvas-for-frame mix-up",
			func(s *terrain.StaticSet) {
				for _, c := range s.Classes {
					if c == nil || c.Frame == nil {
						continue
					}
					f := c.Frame
					c.Width, f.Width = f.Width, c.Width
					c.Height, f.Height = f.Height, c.Height
					f.Pixels = make([]terrain.StaticPixel, f.Width*f.Height)
				}
			},
		},
	}

	type perturbedGeometry struct {
		name    string
		ps      []terrain.StaticPlacement // built from the perturbed classes
		base    []terrain.StaticPlacement // the unperturbed list, for the moved guard
		originY int
		lift    func(col, row int) int
	}

	for _, pt := range perturbSet {
		set := staticMarkerSet()
		pt.apply(set)
		g := staticMarkerGrid()

		flatP, _, _ := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateTiles)
		dispP, _, _ := terrain.StaticPlacements(g, set, staticMarkerLift, staticMarkerOriginY, terrain.AnimGateTiles)
		perturbed := []perturbedGeometry{
			{"flat", flatP, flat, 0, nil},
			{"displaced", dispP, displaced, staticMarkerOriginY, staticMarkerLift},
		}

		for _, pg := range perturbed {
			if len(pg.ps) != len(pg.base) {
				t.Fatalf("%s %s: the perturbation changed the placement COUNT (%d, was %d); it is meant to move where art is drawn, not which cells draw",
					pt.name, pg.name, len(pg.ps), len(pg.base))
			}

			// The guard: the perturbation must genuinely have moved the drawn sprite
			// at every placement, or "the comparison still passes" is a statement
			// about nothing.
			moved := 0
			for i := range pg.ps {
				if pg.ps[i].TopLeft != pg.base[i].TopLeft {
					moved++
				}
			}
			if moved != len(pg.ps) {
				t.Fatalf("%s %s: only %d of %d placements moved; a perturbation the sprite side ignores cannot show that the comparison is blind to it",
					pt.name, pg.name, moved, len(pg.ps))
			}

			agree, disagree := staticMarkerAgree(pg.ps, func(col, row int) image.Point {
				return staticMarkerPoint(col, row, pg.originY, pg.lift)
			})
			t.Logf("%s, %s geometry: %d agree, %d disagree, %d/%d sprites MOVED — MUST still agree (P-4 cannot catch %s)",
				pt.name, pg.name, agree, disagree, moved, len(pg.ps), pt.claim)
			if disagree != 0 {
				t.Errorf("%s %s: %d placement(s) disagreed. SC-5 requires this perturbation to leave the comparison PASSING — the anchor terms cancel, so %s is invisible here and is left to AC-9, where a human sees the art standing away from its cross",
					pt.name, pg.name, disagree, pt.claim)
			}
		}
	}
}

func staticMarkerArmsAt(centre image.Point, cellpx, radius, thickness int) [2]image.Rectangle {
	r := max(1, specRound(radius*cellpx))
	t := max(1, specRound(thickness*cellpx))
	lo := t / 2
	return [2]image.Rectangle{
		image.Rect(centre.X-r, centre.Y-lo, centre.X+r+1, centre.Y-lo+t), // horizontal
		image.Rect(centre.X-lo, centre.Y-r, centre.X-lo+t, centre.Y+r+1), // vertical
	}
}

// staticMarkerClip is the stage-1 map-extent clip at offsetY 0, applied to arms
// built by staticMarkerArmsAt.
func staticMarkerClip(arms [2]image.Rectangle, cols, rows, cellpx int) []image.Rectangle {
	clip := specMapRect(cols, rows, cellpx)
	out := make([]image.Rectangle, 0, 2)
	for _, arm := range arms {
		if c := arm.Intersect(clip); !c.Empty() {
			out = append(out, c)
		}
	}
	return out
}

func TestMarkerAnchorIsTheCentreEveryGlyphUses(t *testing.T) {
	// (a) The formula, hand-written, over both signs of both translation terms
	//     and at scales where cellpx/2 truncates and where it does not.
	anchorCases := []struct {
		col, row, cellpx, offsetY, liftY int
		x, y                             int
	}{
		{0, 0, 32, 0, 0, 16, 16},        // 0*32+16, 0*32+16
		{1, 2, 32, 0, 0, 48, 80},        // 1*32+16, 2*32+16
		{3, 1, 32, 40, 0, 112, 88},      // 48+40 = 88
		{3, 1, 32, -24, 0, 112, 24},     // 48-24 = 24
		{2, 2, 32, 23, -6, 80, 97},      // 80+23-6 = 97
		{2, 2, 32, 23, 6, 80, 109},      // 80+23+6 = 109
		{2, 1, 64, 0, 0, 160, 96},       // 2*64+32, 1*64+32
		{2, 1, 17, 0, 0, 42, 25},        // 2*17+8 = 42, 17+8 = 25
		{2, 1, 15, 0, 0, 37, 22},        // 2*15+7 = 37, 15+7 = 22
		{5, 4, 1, 0, 0, 5, 4},           // cellpx/2 = 0
		{1, 1, 32, -1000, 900, 48, -52}, // 48-1000+900
	}
	for _, tc := range anchorCases {
		x, y := terrain.MarkerAnchor(tc.col, tc.row, tc.cellpx, tc.offsetY, tc.liftY)
		if x != tc.x || y != tc.y {
			t.Errorf("MarkerAnchor(%d,%d,%d,%d,%d) = (%d,%d), want (%d,%d) = (col*cellpx+cellpx/2, row*cellpx+cellpx/2+offsetY+liftY)",
				tc.col, tc.row, tc.cellpx, tc.offsetY, tc.liftY, x, y, tc.x, tc.y)
		}
	}

	glyphs := []struct {
		name              string
		radius, thickness int
		build             func(col, row, cols, rows, cellpx int) []image.Rectangle
	}{
		{"object", 6, 3, terrain.ObjectMarkerRects},
		{"unit", 4, 1, terrain.UnitMarkerRects},
		{"static", 3, 1, terrain.StaticMarkerRects},
	}
	const cols, rows = 5, 4
	for _, gl := range glyphs {
		for _, cellpx := range []int{32, 64, 17, 15, 48, 96} {
			for row := 0; row < rows; row++ {
				for col := 0; col < cols; col++ {
					cx, cy := terrain.MarkerAnchor(col, row, cellpx, 0, 0)
					want := staticMarkerClip(staticMarkerArmsAt(image.Pt(cx, cy), cellpx, gl.radius, gl.thickness), cols, rows, cellpx)
					got := gl.build(col, row, cols, rows, cellpx)
					if !sameRects(got, want) {
						t.Fatalf("%s glyph at cell (%d,%d), cellpx=%d: got %v, want the FR-6 arms built around MarkerAnchor's point (%d,%d): %v — the extraction must move no arithmetic (DD-4)",
							gl.name, col, row, cellpx, got, cx, cy, want)
					}
				}
			}
		}
	}
}

func TestStaticMarkerGlyph(t *testing.T) {
	if want := (color.RGBA{R: 0xff, G: 0x20, B: 0x40, A: 0xff}); terrain.StaticMarkerColor != want {
		t.Fatalf("FR-6: StaticMarkerColor = %v, want the opaque red %v", terrain.StaticMarkerColor, want)
	}
	if terrain.StaticMarkerColor == terrain.MarkerColor || terrain.StaticMarkerColor == terrain.UnitMarkerColor {
		t.Fatal("FR-6: the static marker's colour must be distinct from the -objects and -units markers")
	}
	if terrain.StaticMarkerColor == markerBG {
		t.Fatal("fixture: the background must differ from StaticMarkerColor")
	}

	// The native pair, hand-computed: r = max(1, floor((3*32+16)/32)) = 3,
	// t = max(1, floor((1*32+16)/32)) = 1, lo = 0, half = 16.
	// Cell (2,1) at cellpx=32: cx = 2*32+16 = 80, cy = 1*32+16 = 48.
	//   H = [80-3, 80+4) x [48, 49) = [77,84) x [48,49)
	//   V = [80, 81)     x [48-3, 48+4) = [80,81) x [45,52)
	spot := []image.Rectangle{image.Rect(77, 48, 84, 49), image.Rect(80, 45, 81, 52)}
	if got := terrain.StaticMarkerRects(2, 1, 5, 4, 32); !sameRects(got, spot) {
		t.Errorf("StaticMarkerRects(2,1,5,4,32) = %v, want the hand-computed FR-6 pair %v", got, spot)
	}

	// Off-map cells and an invalid scale contribute nothing, exactly as the two
	// shipped builders do — the glyph is a sibling, not a special case.
	for _, tc := range []struct {
		name             string
		col, row, cellpx int
	}{
		{"col past the map", 5, 1, 32},
		{"row past the map", 1, 4, 32},
		{"negative col", -1, 1, 32},
		{"negative row", 1, -1, 32},
		{"invalid scale", 1, 1, 0},
		{"negative scale", 1, 1, -32},
	} {
		var got []image.Rectangle
		noPanicOverlay(t, "StaticMarkerRects "+tc.name, func() {
			got = terrain.StaticMarkerRects(tc.col, tc.row, 5, 4, tc.cellpx)
		})
		if got != nil {
			t.Errorf("%s: StaticMarkerRects(%d,%d,5,4,%d) = %v, want no geometry", tc.name, tc.col, tc.row, tc.cellpx, got)
		}
	}

	const cols, rows, col, row = 5, 4, 2, 1
	mask := func(arms []image.Rectangle) map[image.Point]bool {
		m := map[image.Point]bool{}
		for _, a := range arms {
			for y := a.Min.Y; y < a.Max.Y; y++ {
				for x := a.Min.X; x < a.Max.X; x++ {
					m[image.Pt(x, y)] = true
				}
			}
		}
		return m
	}
	for _, cellpx := range []int{1, 2, 3, 8, 15, 16, 17, 32, 48, 64, 96, 128} {
		static := mask(terrain.StaticMarkerRects(col, row, cols, rows, cellpx))
		unit := mask(terrain.UnitMarkerRects(col, row, cols, rows, cellpx))
		object := mask(terrain.ObjectMarkerRects(col, row, cols, rows, cellpx))
		if len(static) == 0 {
			t.Fatalf("cellpx=%d: the static glyph covered no pixel at cell (%d,%d)", cellpx, col, row)
		}
		for p := range static {
			if !unit[p] {
				t.Errorf("cellpx=%d: static pixel %v is outside the unit cross; the static glyph must be a subset of both shipped glyphs at every scale (DD-4)", cellpx, p)
			}
			if !object[p] {
				t.Errorf("cellpx=%d: static pixel %v is outside the object cross (DD-4)", cellpx, p)
			}
		}
		if cellpx >= terrain.CellSize && len(static) >= len(unit) {
			t.Errorf("cellpx=%d: the static cross covers %d pixels and the unit cross %d; at the native scale and above the containment must be STRICT",
				cellpx, len(static), len(unit))
		}
	}
}

// TestDrawStaticMarkers covers the two draw entry points: DrawStaticMarkersAt on
// a translated canvas, DrawStaticMarkersAtHeights with a per-cell lift, the
// nil-lift reduction between them, and the draw order that puts the static glyph
// last of the three without hiding either.
//
// The oracle is paintLiftMask — this package's established transcription of the
// two-stage clip, where the per-cell lift reaches the glyph's centre and never
// the map-extent rectangle — called with the static glyph's own radius and
// thickness. It calls nothing from overlay.go.
func TestDrawStaticMarkers(t *testing.T) {
	const cols, rows = 5, 4
	const staticRadius, staticThickness = 3, 1

	// Off-map cells are in the list throughout: the drop is a cell-index test,
	// so no translation or lift may admit one.
	cells := []image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 4, Y: 3}, {X: cols, Y: 0}, {X: 1, Y: -1}, {X: 1, Y: rows}}
	lift := func(col, row int) int { return 9*col - 5*row + 3 }

	canvases := []struct {
		name                  string
		cellpx, offsetY, imgH int
	}{
		{"native, flat canvas", 32, 0, rows * 32},
		{"native, offsetY +40 (the ordinary negative origin)", 32, 40, rows*32 + 40},
		{"native, offsetY -24 (a positive origin)", 32, -24, rows*32 + 12},
		{"scale 2, offsetY +40", 64, 40, rows*64 + 40},
	}

	for _, cv := range canvases {
		imgW := cols * cv.cellpx

		// (a) DrawStaticMarkersAt: no lift.
		want := paintLiftMask(imgW, cv.imgH, cells, cols, rows, cv.cellpx, cv.offsetY, nil, staticRadius, staticThickness)
		if maskCount(want) == 0 {
			t.Fatalf("%s: the oracle marked no pixel, so the case can say nothing", cv.name)
		}
		checkAgainstMask(t, cv.name+": DrawStaticMarkersAt", imgW, cv.imgH, want, terrain.StaticMarkerColor, func(img *image.RGBA) {
			terrain.DrawStaticMarkersAt(img, cells, cols, rows, cv.cellpx, cv.offsetY)
		})

		// (b) DrawStaticMarkersAtHeights with a nil lift and with a zero one must
		//     both reduce to (a) exactly.
		for _, z := range []struct {
			name string
			fn   func(col, row int) int
		}{{"nil lift", nil}, {"zero lift", func(col, row int) int { return 0 }}} {
			checkAgainstMask(t, cv.name+": DrawStaticMarkersAtHeights, "+z.name, imgW, cv.imgH, want, terrain.StaticMarkerColor, func(img *image.RGBA) {
				terrain.DrawStaticMarkersAtHeights(img, cells, cols, rows, cv.cellpx, cv.offsetY, z.fn)
			})
		}

		// (c) DrawStaticMarkersAtHeights with a real per-cell lift. The oracle
		//     builds the stage-1 clip from offsetY ALONE, so a lift that carries a
		//     glyph past the map extent is cut at the unlifted extent — the 0015
		//     seam this glyph inherits unchanged.
		lifted := paintLiftMask(imgW, cv.imgH, cells, cols, rows, cv.cellpx, cv.offsetY, lift, staticRadius, staticThickness)
		if masksEqual(lifted, want) {
			t.Fatalf("%s: the lifted and unlifted masks are identical, so the lift case would pass vacuously", cv.name)
		}
		checkAgainstMask(t, cv.name+": DrawStaticMarkersAtHeights, per-cell lift", imgW, cv.imgH, lifted, terrain.StaticMarkerColor, func(img *image.RGBA) {
			terrain.DrawStaticMarkersAtHeights(img, cells, cols, rows, cv.cellpx, cv.offsetY, lift)
		})
	}

	t.Run("the static glyph draws last", func(t *testing.T) {
		const cellpx, offsetY = 32, 0
		one := []image.Point{{X: 2, Y: 1}}
		imgW, imgH := cols*cellpx, rows*cellpx

		staticMask := paintLiftMask(imgW, imgH, one, cols, rows, cellpx, offsetY, nil, staticRadius, staticThickness)
		unitMask := paintLiftMask(imgW, imgH, one, cols, rows, cellpx, offsetY, nil, specUnitRadius, specUnitThickness)
		objectMask := paintLiftMask(imgW, imgH, one, cols, rows, cellpx, offsetY, nil, specObjectRadius, specObjectThickness)
		if maskCount(staticMask) >= maskCount(unitMask) || maskCount(unitMask) >= maskCount(objectMask) {
			t.Fatalf("fixture: the three crosses cover %d, %d and %d pixels; the nesting this test rests on needs them strictly increasing",
				maskCount(staticMask), maskCount(unitMask), maskCount(objectMask))
		}

		want := make([]color.RGBA, imgW*imgH)
		for i := range want {
			switch {
			case staticMask[i]:
				want[i] = terrain.StaticMarkerColor
			case unitMask[i]:
				want[i] = terrain.UnitMarkerColor
			case objectMask[i]:
				want[i] = terrain.MarkerColor
			default:
				want[i] = markerBG
			}
		}

		img := backgroundImage(imgW, imgH)
		noPanicOverlay(t, "objects, units, then statics", func() {
			terrain.DrawObjectMarkersAt(img, one, cols, rows, cellpx, offsetY)
			terrain.DrawUnitMarkersAt(img, one, cols, rows, cellpx, offsetY)
			terrain.DrawStaticMarkersAt(img, one, cols, rows, cellpx, offsetY)
		})
		for y := 0; y < imgH; y++ {
			for x := 0; x < imgW; x++ {
				if got := img.RGBAAt(x, y); got != want[y*imgW+x] {
					t.Fatalf("pixel (%d,%d) = %v, want %v — with the order objects -> units -> statics the three crosses must nest, the static one on top (FR-6)",
						x, y, got, want[y*imgW+x])
				}
			}
		}

		reversed := backgroundImage(imgW, imgH)
		noPanicOverlay(t, "statics first", func() {
			terrain.DrawStaticMarkersAt(reversed, one, cols, rows, cellpx, offsetY)
			terrain.DrawObjectMarkersAt(reversed, one, cols, rows, cellpx, offsetY)
			terrain.DrawUnitMarkersAt(reversed, one, cols, rows, cellpx, offsetY)
		})
		red := 0
		for y := 0; y < imgH; y++ {
			for x := 0; x < imgW; x++ {
				if reversed.RGBAAt(x, y) == terrain.StaticMarkerColor {
					red++
				}
			}
		}
		if red != 0 {
			t.Errorf("drawing the static glyph FIRST left %d StaticMarkerColor pixel(s), want 0: it is a strict subset of both shipped crosses, so a wrong order must erase it completely — the control that makes the order assertion above meaningful (FR-6, DD-4)",
				red)
		}
	})

	// (e) A nil image and an empty cell list are no-ops rather than panics, the
	//     same as the two shipped kinds.
	noPanicOverlay(t, "DrawStaticMarkersAt on a nil image", func() {
		terrain.DrawStaticMarkersAt(nil, cells, cols, rows, 32, 0)
	})
	noPanicOverlay(t, "DrawStaticMarkersAtHeights on a nil image", func() {
		terrain.DrawStaticMarkersAtHeights(nil, cells, cols, rows, 32, 0, lift)
	})
	empty := paintLiftMask(cols*32, rows*32, nil, cols, rows, 32, 0, nil, staticRadius, staticThickness)
	if n := checkAgainstMask(t, "empty cell list", cols*32, rows*32, empty, terrain.StaticMarkerColor, func(img *image.RGBA) {
		terrain.DrawStaticMarkersAtHeights(img, nil, cols, rows, 32, 0, lift)
	}); n != 0 {
		t.Errorf("an empty cell list marked %d pixel(s), want 0", n)
	}
}
