package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

// shiftPeriod is the tick length every phase below is stated over: the spec's
// own worked example, so its offsets are the contract's own literals.
const shiftPeriod = 62_000

// The two entities every fixture holds and the cell each stepped into. Both took
// the SAME step, so a build that displaced one family by another entity's vector
// would still agree — which is why the marks and bars are checked against each
// entity's own oracle rather than against each other.
var (
	shiftSpriteCell = image.Pt(3, 3)
	shiftSquareCell = image.Pt(5, 4)
	shiftEast       = image.Pt(1, 0)
)

// cellsOf is the cell of each entity in a list, in the list's own order. The
// square half of the entity split hands back entities now rather than bare
// cells, and this is how the files that assert those cells read them.
func cellsOf(ents []MapEntity) []image.Point {
	cells := make([]image.Point, len(ents))
	for i, e := range ents {
		cells[i] = e.Cell
	}
	return cells
}

// shiftGrid is an 8x8 grid, with a relief rising four native units a column when
// asked for one and none at all when not — flat mode and displaced mode over
// cells whose lifts genuinely differ.
func shiftGrid(relief bool) terrain.Grid {
	g := grid(8, 8)
	if !relief {
		return g
	}
	alts := make([]uint8, 8*8)
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			alts[row*8+col] = uint8(4 * col)
		}
	}
	g.Altitudes = alts
	return g
}

// shiftProjection is the independent oracle's projection: terrain.Project over a
// COPY of that grid's altitude bytes, never the viewer's own.
func shiftProjection() terrain.Projection {
	g := shiftGrid(true)
	return terrain.Project(append([]uint8(nil), g.Altitudes...), g.Width, g.Height)
}

// shiftEntities are the two units, each on its own cell and each carrying the
// given step: one that resolves to a sprite and one that keeps the square, both
// selected and both carrying a bar, so all three glyph families exist for each.
func shiftEntities(step image.Point) []MapEntity {
	a := entityArtA()
	return []MapEntity{
		{ID: 1, Cell: shiftSpriteCell, Step: step, Art: a, Frame: a.Frames[0],
			Life: LifeAlive, HP: 80, MaxHP: 100},
		{ID: 2, Cell: shiftSquareCell, Step: step, Life: LifeAlive, HP: 55, MaxHP: 100},
	}
}

// shiftViewer holds those two over that grid at a camera away from the origin,
// with both selected and the given phase pushed.
func shiftViewer(t *testing.T, relief bool, ents []MapEntity, elapsed, period int) *Viewer {
	t.Helper()
	v, err := NewViewer("shift", shiftGrid(relief), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, 600, 500)
	if displaced := v.Mode() == ModeDisplaced; displaced != relief {
		t.Fatalf("a grid with relief=%v opened in mode %v", relief, v.Mode())
	}
	v.Camera().Pan(17, 11)
	if v.Camera().X == 0 && v.Camera().Y == 0 {
		t.Fatal("the camera sits at the origin, so screen pixels are world pixels and the transform " +
			"below witnesses nothing")
	}
	v.SetEntities(ents)
	// Everything held is selected, so every entity asserted below carries a mark
	// of its own — the family that has no cell list any more.
	v.sel = nil
	for _, e := range ents {
		v.sel = append(v.sel, e.ID)
	}
	v.SetPhase(elapsed, period)
	return v
}

// shiftLift is the vertical shift the shared transform owes one cell:
// -AnchorHeight(col,row)-MinV displaced, zero flat.
func shiftLift(proj terrain.Projection, relief bool, cell image.Point) int {
	if !relief {
		return 0
	}
	return -proj.AnchorHeight(cell.X, cell.Y) - proj.MinV
}

// shiftPlace is the oracle transform: the render tier's own arms for a cell,
// moved by one vector, lifted by that cell's own lift, then through the camera —
// the two steps every placement oracle in this package applies, with the
// displacement inserted ahead of them. It culls nothing.
func shiftPlace(cam *camera.Camera, lift int, shift image.Point, arms ...image.Rectangle) []screenRect {
	var out []screenRect
	for _, arm := range arms {
		if arm.Empty() {
			continue
		}
		arm = arm.Add(shift).Add(image.Pt(0, lift))
		sx, sy := cam.WorldToScreen(float64(arm.Min.X), float64(arm.Min.Y))
		out = append(out, screenRect{X: sx, Y: sy, W: float64(arm.Dx()) * cam.Zoom, H: float64(arm.Dy()) * cam.Zoom})
	}
	return out
}

// shiftAssert is where every comparison in this file is made: the three glyph
// families a viewer draws for ONE entity, each against the same one vector.
//
// EVERY FAMILY IS READ THROUGH THE PRODUCTION READER the frame itself uses — the
// square out of the pass slice, the mark out of selectionScreenRects, the bar
// out of the status bar walk — because a reader that reached past one of them
// into the shared walk beneath would agree with a build whose glyph paths had
// come apart above it. All three walk the whole snapshot, so the snapshot is
// narrowed to this one entity for the length of the read and put straight back:
// what is then compared is THIS entity's own vector, in all three families, at
// this phase.
func shiftAssert(t *testing.T, v *Viewer, proj terrain.Projection, relief bool, e MapEntity, label string, shift image.Point) {
	t.Helper()

	cam := v.Camera()
	w, h := v.grid.Width, v.grid.Height
	lift := shiftLift(proj, relief, e.Cell)

	all := v.entities
	v.entities = []MapEntity{e}
	sprites, squares, _ := v.entityLayer()
	squareRects := entitySquares(v)
	marks := v.selectionScreenRects()
	grounds, fills := v.healthBarScreenRects()
	v.entities = all

	if e.Art != nil {
		if len(sprites) != 1 || len(squares) != 0 {
			t.Fatalf("%s: entity %d split into %d sprite(s) and %d square(s), want 1 and 0",
				label, e.ID, len(sprites), len(squares))
		}
		base, ok := terrain.UnitPlace(e.Cell.X, e.Cell.Y, e.Art, e.Frame, e.Mirror,
			projLift(proj, relief, e.Cell), projOrigin(proj, relief))
		if !ok {
			t.Fatalf("%s: UnitPlace refused entity %d's own art", label, e.ID)
		}
		if want := base.TopLeft.Add(shift); sprites[0].TopLeft != want {
			t.Errorf("%s: entity %d's sprite is placed at %v, want %v — its top-left carries the "+
				"frame's own displacement (FR-3)", label, e.ID, sprites[0].TopLeft, want)
		}
		if want := base.Ground().Add(shift); sprites[0].Ground() != want {
			t.Errorf("%s: entity %d's ground point is %v, want %v", label, e.ID, sprites[0].Ground(), want)
		}
	} else {
		if len(sprites) != 0 || len(squares) != 1 {
			t.Fatalf("%s: entity %d split into %d sprite(s) and %d square(s), want 0 and 1",
				label, e.ID, len(sprites), len(squares))
		}
		assertScreenRects(t, squareRects,
			shiftPlace(cam, lift, shift, terrain.EntityMarkerRects(e.Cell.X, e.Cell.Y, w, h, terrain.CellSize)...))
	}

	assertScreenRects(t, marks,
		shiftPlace(cam, lift, shift, terrain.SelectionMarkerRects(e.Cell.X, e.Cell.Y, w, h, terrain.CellSize)...))

	bar, ok := terrain.StatusBarRect(terrain.HealthBar, e.Cell.X, e.Cell.Y, w, h, e.TokenSize, e.Art)
	fill, pool := terrain.StatusBarFill(terrain.HealthBar, bar.Dx(), e.HP, e.MaxHP)
	if !ok || !pool {
		t.Fatalf("%s: the render tier builds no bar for entity %d", label, e.ID)
	}
	assertScreenRects(t, grounds, shiftPlace(cam, lift, shift, bar))
	assertScreenRects(t, fills, shiftPlace(cam, lift, shift,
		image.Rect(bar.Min.X+4, bar.Min.Y, bar.Min.X+4+fill, bar.Max.Y)))
}

// projLift and projOrigin are the two terms the entity layer hands UnitPlace,
// unnegated, and 0 and 0 in flat mode.
func projLift(proj terrain.Projection, relief bool, cell image.Point) int {
	if !relief {
		return 0
	}
	return proj.AnchorHeight(cell.X, cell.Y)
}

func projOrigin(proj terrain.Projection, relief bool) int {
	if !relief {
		return 0
	}
	return proj.MinV
}

// shiftPhases are the points AC-4 and AC-5 read: the tick's start, a quarter
// through, half way, and a whole period — plus the two arms an unclamped
// remainder would throw an entity out of the segment on.
//
// wantX is the horizontal offset of a ONE-CELL EAST step at that phase, and
// every value is the contract's own: (-32,0) at the start, (-16,0) half way and
// (0,0) at a whole period, with the quarter point -32*46500/62000 = -24.
var shiftPhases = []struct {
	name    string
	elapsed int
	left    int
	wantX   int
}{
	{"the tick's start", 0, shiftPeriod, -32},
	{"a quarter through", 15_500, 46_500, -24},
	{"half way", 31_000, 31_000, -16},
	{"a whole period", shiftPeriod, 0, 0},
	{"past a whole period, clamped", shiftPeriod * 3, 0, 0},
	{"before the tick began, clamped", -9_000, shiftPeriod, -32},
}

// TestOneStepIsDrawnBetweenTheTwoCellsItJoins — 0047 SC-3 (AC-4, R-1): a
// unit that stepped one cell east is drawn between the cell it left and the
// cell it entered, and its sprite, its selection mark and its health bar all
// carry the same displacement.
func TestOneStepIsDrawnBetweenTheTwoCellsItJoins(t *testing.T) {
	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; every offset here is the contract's own literal", terrain.CellSize)
	}

	ents := shiftEntities(shiftEast)
	for _, ph := range shiftPhases {
		t.Run(ph.name, func(t *testing.T) {
			v := shiftViewer(t, false, ents, ph.elapsed, shiftPeriod)
			for _, e := range ents {
				shiftAssert(t, v, terrain.Projection{}, false, e, ph.name, image.Pt(ph.wantX, 0))
			}
		})
	}

	for _, end := range []struct {
		name    string
		elapsed int
		back    image.Point
	}{
		{"the tick's start stands on the cell it left", 0, shiftEast},
		{"a whole period stands on the cell it entered", shiftPeriod, image.Point{}},
	} {
		t.Run(end.name, func(t *testing.T) {
			moving := shiftViewer(t, false, shiftEntities(shiftEast), end.elapsed, shiftPeriod)
			at := shiftEntities(image.Point{})
			for i := range at {
				at[i].Cell = at[i].Cell.Sub(end.back)
			}
			resting := shiftViewer(t, false, at, 0, 0)

			assertScreenRects(t, entitySquares(moving), entitySquares(resting))
			assertScreenRects(t, moving.selectionScreenRects(), resting.selectionScreenRects())
			mg, mf := moving.healthBarScreenRects()
			rg, rf := resting.healthBarScreenRects()
			assertScreenRects(t, mg, rg)
			assertScreenRects(t, mf, rf)

			ms, _, _ := moving.entityLayer()
			rs, _, _ := resting.entityLayer()
			if len(ms) != 1 || len(rs) != 1 {
				t.Fatalf("%d moving and %d resting sprite(s), want one each", len(ms), len(rs))
			}
			if ms[0].TopLeft != rs[0].TopLeft || ms[0].Ground() != rs[0].Ground() {
				t.Errorf("the moving sprite is at %v/%v and the resting one at %v/%v",
					ms[0].TopLeft, ms[0].Ground(), rs[0].TopLeft, rs[0].Ground())
			}
		})
	}
}

// TestTheReliefRunsAcrossTheTickWithTheStep — 0047 SC-4 (AC-5): over a map
// with altitudes, a unit stepping between two cells of different height
// rises or falls ACROSS the tick rather than at its boundary, between
// exactly the two per-cell heights the glyphs on those two cells are lifted
// by — and the sprite, the mark and the bar share that vertical term as
// they share the horizontal one.
func TestTheReliefRunsAcrossTheTickWithTheStep(t *testing.T) {
	proj := shiftProjection()
	ents := shiftEntities(shiftEast)

	// The two cells must genuinely differ in height, or this test is the flat
	// one over again.
	for _, e := range ents {
		from := e.Cell.Sub(shiftEast)
		if proj.AnchorHeight(from.X, from.Y) == proj.AnchorHeight(e.Cell.X, e.Cell.Y) {
			t.Fatalf("entity %d steps between two cells of equal height; pick another relief", e.ID)
		}
	}

	for _, ph := range shiftPhases {
		t.Run(ph.name, func(t *testing.T) {
			v := shiftViewer(t, true, ents, ph.elapsed, shiftPeriod)
			for _, e := range ents {
				from := e.Cell.Sub(shiftEast)
				rise := proj.AnchorHeight(from.X, from.Y) - proj.AnchorHeight(e.Cell.X, e.Cell.Y)
				want := image.Pt(ph.wantX, -rise*ph.left/shiftPeriod)
				shiftAssert(t, v, proj, true, e, ph.name, want)
			}
		})
	}

	// The endpoints again, in relief: at a remainder of zero the drawn height is
	// the LEFT cell's own lift and not the entered cell's, which is what says the
	// rise happens across the tick and not at its boundary.
	moving := shiftViewer(t, true, shiftEntities(shiftEast), 0, shiftPeriod)
	at := shiftEntities(image.Point{})
	for i := range at {
		at[i].Cell = at[i].Cell.Sub(shiftEast)
	}
	resting := shiftViewer(t, true, at, 0, 0)
	assertScreenRects(t, moving.selectionScreenRects(), resting.selectionScreenRects())
	mg, _ := moving.healthBarScreenRects()
	rg, _ := resting.healthBarScreenRects()
	assertScreenRects(t, mg, rg)
}

// TestNothingThatDidNotMoveIsEverDisplaced — 0047 SC-5 (AC-6): an entity
// that did not move on the most recent advance is drawn with NO
// displacement, on its own cell, at every remainder — standing, blocked,
// downed and dead alike, which are four states and one rule.
func TestNothingThatDidNotMoveIsEverDisplaced(t *testing.T) {
	// A blocked unit is one that still holds an order and took no step, which is
	// the same seam value as a standing one: the seam carries what the world DID.
	resting := []MapEntity{
		{ID: 1, Cell: image.Pt(2, 2), Life: LifeAlive, HP: 100, MaxHP: 100},
		{ID: 2, Cell: image.Pt(3, 2), Life: LifeAlive, HP: 40, MaxHP: 100},
		{ID: 3, Cell: image.Pt(4, 2), Life: LifeDowned, HP: 0, MaxHP: 100},
		{ID: 4, Cell: image.Pt(5, 2), Life: LifeDead, HP: -7, MaxHP: 100},
	}
	for _, relief := range []bool{false, true} {
		proj := shiftProjection()
		for _, ph := range shiftPhases {
			v := shiftViewer(t, relief, resting, ph.elapsed, shiftPeriod)
			for _, e := range resting {
				if e.Life == LifeDead {
					continue // no mark and no bar: filtered before any geometry
				}
				shiftAssert(t, v, proj, relief, e, ph.name, image.Point{})
			}
			// The dead unit is drawn as a square standing on its own cell.
			_, squares, _ := v.entityLayer()
			if got, want := cellsOf(squares), cellsOf(resting); len(got) != len(want) {
				t.Fatalf("%v: %d squares over %d artless entities", ph.name, len(got), len(want))
			}
		}
	}
}

// TestAViewerToldNothingDrawsEveryEntityOnItsOwnCell — 0047 SC-6's
// never-told and clamp arms as geometry (AC-7): a viewer never given a phase
// draws every entity on its own cell however far it stepped, and a period
// shrunk below the remainder already accumulated draws it on the cell it
// ENTERED and never past it.
//
// The never-told arm is what leaves the standalone front-end, and every caller
// written before this story, drawing exactly what it drew before.
func TestAViewerToldNothingDrawsEveryEntityOnItsOwnCell(t *testing.T) {
	ents := shiftEntities(shiftEast)
	still := shiftEntities(image.Point{})

	for _, tc := range []struct {
		name            string
		elapsed, period int
	}{
		{"never told", 0, 0},
		{"a period of zero with a remainder held", 40_000, 0},
		{"a negative period", 40_000, -5},
		{"a period shrunk below the remainder", 90_000, 3_906},
	} {
		t.Run(tc.name, func(t *testing.T) {
			told := shiftViewer(t, true, ents, tc.elapsed, tc.period)
			none := shiftViewer(t, true, still, 0, 0)

			assertScreenRects(t, entitySquares(told), entitySquares(none))
			assertScreenRects(t, told.selectionScreenRects(), none.selectionScreenRects())
			tg, tf := told.healthBarScreenRects()
			ng, nf := none.healthBarScreenRects()
			assertScreenRects(t, tg, ng)
			assertScreenRects(t, tf, nf)

			ts, _, _ := told.entityLayer()
			ns, _, _ := none.entityLayer()
			if len(ts) != 1 || len(ns) != 1 || ts[0].TopLeft != ns[0].TopLeft {
				t.Errorf("the sprite is placed at %v, want the undisplaced %v", ts, ns)
			}
		})
	}
}
