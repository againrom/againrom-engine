package ui

import (
	"image"
	"image/color"

	"againrom/pkg/render/terrain"
)

// The path overlay: the line a selected unit is about to walk, drawn from the
// cell it stands on along the route the simulation already holds for it.
//
// NOTHING HERE COMPUTES A PATH AND NOTHING HERE REMEMBERS ONE. The route
// arrives on the seam per tick, as the world's own answer, so this file is a
// scoping rule and a transform and no more. A cache would need an
// invalidation rule, and an invalidation rule on this side of the seam is a
// second opinion about when a route went stale — a question the simulation
// already answers, with the state to answer it from.

// THERE IS NO LIMIT ON HOW MANY UNITS ARE PREVIEWED, and the constant that
// used to impose one is gone (0037 FR-8a). It capped the DRAWABLE UNIT COUNT at
// thirty-two and, over that, drew nothing at all — so a large enough selection
// silently lost the whole overlay, which is the defect the owner reported.
//
// The count was never the cost either. What a frame pays for is the LEGS it
// issues, and a unit's leg count is its route's length, which runs from 1 to
// 251 on a 256x256 map: the cap was on the wrong quantity by up to that factor
// in either direction. Measured on that map at the default zoom, a cap's worth
// of movers on full-diagonal routes is 8032 legs and two hundred movers on
// two-cell routes is 400 — and the rule drew the first and refused the second.
//
// What bounds the cost instead is below, in pathScreenSegments, and it is
// not a limit at all: a leg that cannot reach the window is not issued. That
// is exact rather than a judgement — it removes no pixel — and it is
// what makes drawing every line affordable.

// PathColor is the overlay's line colour. It is exported beside MarqueeColor and
// for that constant's own reason: it is this package's own choice about an
// instrument it draws itself, where every colour a CELL is drawn in belongs to
// the render tier's palette.
var PathColor = color.RGBA{R: 0xff, G: 0xa0, B: 0x30, A: 0xff}

// PathWidth is the stroke width in SCREEN pixels, and it deliberately does not
// scale with the zoom. The line is an instrument reporting an order, not a thing
// standing on the map, so it stays legible zoomed out — where the route is
// longest and the reading matters most — instead of thinning to nothing.
const PathWidth = 2

// pathsToDraw is the entities whose route the overlay strokes this frame:
// those SELECTED, still in the snapshot, still alive or downed, and holding
// a route — ALL of them, however many there are (FR-8a).
//
// IT IS PURE and it is the whole decision. The frame's drawing reads its result
// and asks nothing else, so "which lines are on screen" is a value a test can
// hold rather than an effect it would have to observe through a window.
//
// The presence and life filter is presentSelected — CALLED, not copied. The
// units that show a path are exactly the units that would take an order and
// exactly the units wearing a selection rim, and that is one predicate with
// three readers rather than three rules kept in step. A corpse is excluded by
// it, which is right twice over: it holds no route, and it could not be given
// one.
//
// It takes no count and returns no short answer, so there is no number here for
// a selection to be on the wrong side of.
func pathsToDraw(cur selection, ents []MapEntity) []MapEntity {
	var out []MapEntity
	for _, e := range presentSelected(cur, ents) {
		if len(e.Route) == 0 {
			continue
		}
		out = append(out, e)
	}
	return out
}

// segmentMisses answers whether a leg spanning a..b on one axis lies wholly
// outside lo..hi on that axis. Two of these, one per axis, are a bounding-box
// reject: a leg failing EITHER cannot meet the rectangle at all.
func segmentMisses(a, b, lo, hi float64) bool {
	if a > b {
		a, b = b, a
	}
	return b < lo || a > hi
}

// pathSegment is one drawn leg of a path, in screen pixels: from one cell centre
// to the next.
type pathSegment struct {
	X0, Y0, X1, Y1 float64
}

// pathCellCentre is where a cell's centre lands on the screen, through the same
// two transforms every glyph on that cell goes through: the displaced mode's own
// height lift, then the camera.
//
// It is the point form of placeArm and it is deliberately separate rather than a
// widening of it, because it differs in the one thing that would be wrong to
// share — placeArm CULLS, answering false for an arm the view does not reach,
// and a culled endpoint is not a culled segment. A leg whose two ends are both
// off screen may still cross the middle of it, so a line drawn from culled
// endpoints would blink out exactly when a unit walks past the edge of the view.
// Clipping the drawn line is the drawer's job and it already does it.
func (v *Viewer) pathCellCentre(cell image.Point) (float64, float64) {
	wx, wy := v.cellWorldCentre(cell)
	return v.cam.WorldToScreen(wx, wy)
}

// cellWorldCentre is the WORLD point at the middle of a cell as this viewer
// draws it: the flat lattice's centre, plus displaced mode's own per-cell lift.
//
// It is the half of pathCellCentre that stops short of the camera, split out
// because a second caller needs the point rather than the pixel — 0088 aims the
// camera AT a cell, and a camera transform is exactly what it must not have
// already applied.
//
// THE LIFT IS THE SAME EXPRESSION placeArm APPLIES, sign and origin alike:
// -AnchorHeight(cell) - MinV, the shift WorldCorner already carries on terrain.
// Keeping one expression is the point — a marker, a health bar, a path leg and
// now a start view all sit on one cell, and a copy is what would let one of them
// drift off the mark on relief while every flat test stayed green.
//
// The guard is v.Mode() == ModeDisplaced and not the weaker v.proj != nil, for
// the reason placeArm's is: a projection can exist while SetFlat has
// deliberately selected flat, and nothing may carry a height offset there.
func (v *Viewer) cellWorldCentre(cell image.Point) (float64, float64) {
	wx := cell.X*terrain.CellSize + terrain.CellSize/2
	wy := cell.Y*terrain.CellSize + terrain.CellSize/2
	if v.Mode() == ModeDisplaced {
		wy += -v.proj.AnchorHeight(cell.X, cell.Y) - v.proj.MinV
	}
	return float64(wx), float64(wy)
}

// pathScreenSegments is every leg of every drawn path, in the order the units
// come off pathsToDraw and, within one unit, in walking order.
//
// THE FIRST LEG STARTS AT THE UNIT and not at the route's first cell, because
// the route is what is LEFT to walk: its head is the next cell, not the current
// one. Anchoring the line to the entity's own Cell is what makes it shrink as
// the unit advances instead of trailing a stub behind it — and Cell is the same
// field every other glyph on that unit is placed from, so the line begins where
// the unit is drawn rather than near it.
//
// It carries NO per-frame interpolation. entityShift moves a unit smoothly
// between its two cells across a tick; the path is anchored to the cell the unit
// is ON, so the line's tail meets the unit exactly at each tick boundary and
// lags it by at most the one cell it is mid-step across. That is deliberate: a
// route is a statement about cells, and a preview that slid with the sprite
// would be redrawing the same statement thirty times a tick to no end.
//
// A LEG THAT CANNOT REACH THE WINDOW IS NOT ISSUED (FR-9a), and the test is
// the leg's own bounding box against the view — never its endpoints. A
// bounding box that misses the view cannot contain a point inside it, so
// nothing dropped here could have painted a pixel. The window is grown by
// the full stroke width, which is twice what the stroke reaches past its own
// line.
func (v *Viewer) pathScreenSegments() []pathSegment {
	drawn := pathsToDraw(v.sel, v.entities)
	if len(drawn) == 0 {
		return nil
	}
	loX, loY := float64(-PathWidth), float64(-PathWidth)
	hiX := float64(v.cam.ViewW) + PathWidth
	hiY := float64(v.cam.ViewH) + PathWidth
	var out []pathSegment
	for _, e := range drawn {
		// A PATH IS AN INDICATOR TOO (hotfix), and this one says more than a
		// health bar does: it shows where an enemy is AND where he is going.
		// Nothing scopes the selection to the local participant's own units, so a
		// selected enemy that walked into the dark drew his whole route across it.
		// Gated on the same predicate as the rim over the same unit, at the draw
		// and never inside pathsToDraw — that filter is shared with
		// presentSelected's own order path and this hotfix makes no input rule.
		if !v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
			continue
		}
		x0, y0 := v.pathCellCentre(e.Cell)
		for _, c := range e.Route {
			x1, y1 := v.pathCellCentre(c)
			if !segmentMisses(x0, x1, loX, hiX) && !segmentMisses(y0, y1, loY, hiY) {
				out = append(out, pathSegment{X0: x0, Y0: y0, X1: x1, Y1: y1})
			}
			x0, y0 = x1, y1
		}
	}
	return out
}
