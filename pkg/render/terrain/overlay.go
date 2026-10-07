package terrain

import (
	"image"
	"image/color"
)

// MarkerColor is the opaque yellow a placed-object marker is drawn in.
//
// The colour and the cross dimensions below are the project's own diagnostic
// design. They assert nothing about how the original engine drew placed
// objects — real object art needs the class registries and the static data
// formats, which are downstream — but once chosen they are the frozen
// contract the acceptance tests pin.
var MarkerColor = color.RGBA{R: 0xff, G: 0xd0, B: 0x00, A: 0xff}

// UnitMarkerColor is the opaque cyan a placed-unit marker is drawn in.
//
// Like MarkerColor it is the project's own diagnostic design and asserts
// nothing about how the original engine drew units — real unit art needs
// the class registries and the static data formats, which are downstream. It
// is deliberately distinct from the object yellow so a composed render tells
// the two placements apart at a glance.
var UnitMarkerColor = color.RGBA{R: 0x00, G: 0xe5, B: 0xff, A: 0xff}

// StaticMarkerColor is the opaque red a static-object marker is drawn in —
// the cross on a type-3 cell whose object byte resolves to a drawable frame.
//
// Like the two above it is the project's own diagnostic design and asserts
// nothing about the original engine. It is deliberately distinct from both the
// object yellow and the unit cyan, so a composed render tells all three
// placement kinds apart at a glance — one question, three glyphs.
//
// This is the one marker whose purpose is to be COMPARED against art rather
// than merely to locate a record: the sprite this layer draws for a cell
// derives its own ground point independently, so a cross standing a tile
// away from the base of its own tree is a placement bug the owner can see
// (AC-9).
var StaticMarkerColor = color.RGBA{R: 0xff, G: 0x20, B: 0x40, A: 0xff}

// EntityMarkerColor is the opaque magenta a simulated entity's marker is
// drawn in — the filled square standing on the cell one entity of the open
// world occupies.
//
// Like the three above it is the project's own diagnostic design and asserts
// nothing about how the original engine draws a unit: this is a placeholder
// square, not a sprite (0020's constraints table). It is deliberately distinct
// from the object yellow, the unit cyan and the static red, so a composed render
// tells a simulated entity apart from all three record markers at a glance.
//
// It is the one glyph in this family that is NOT a cross, and the difference
// is load-bearing rather than decorative. A filled square is a superset of
// every cross it covers, so an entity marker drawn last would erase a
// coincident diagnostic instead of merely dimming it — which is why the
// entity pass runs FIRST, under all three. The order is the caller's; what
// this file owes is a glyph that stays inside its own cell, so that nothing
// outside that cell can be covered at all.
var EntityMarkerColor = color.RGBA{R: 0xff, G: 0x00, B: 0xff, A: 0xff}

// SelectionMarkerColor is the opaque white the selected unit's highlight is
// drawn in — the rim standing on the footprint of the cell that unit
// occupies.
//
// Like the four above it is the project's own diagnostic design and asserts
// nothing about how the original engine marks a selected unit. It is
// deliberately distinct from the object yellow, the unit cyan, the static red
// and the entity magenta, so a composed render tells a highlight apart from
// every placement kind and from the entity layer itself at a glance.
//
// White is the one hue none of the four uses and the one no channel of theirs
// can be mistaken for: this glyph is drawn LAST, over everything, so the pixels
// it owns must be readable as "this cell is selected" and never as a brighter
// version of the marker underneath.
var SelectionMarkerColor = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

// BlockedCellColor is the red wash a cell closed to a ground mover is tinted
// with — the ONE TRANSLUCENT value in this file, and the only one that is
// not a glyph's colour but a whole cell's.
//
// IT IS PREMULTIPLIED, like every color.RGBA, and that is why no channel exceeds
// the alpha. {0xff, 0, 0, 0x60} is not a paler red: it is an invalid colour, and
// what it renders as is undefined rather than wrong in a direction you could
// correct by eye. The five colours above are all opaque, so there is no
// precedent in this file to copy and the constraint is written down here.
//
// 0x60 of 0xff is a wash rather than a fill, which is what lets the pass be
// drawn OVER the terrain and the static art rather than under them: everything
// beneath still reads, and a diagnostic marker standing on a tinted cell is
// tinted rather than hidden. An opaque value would have to be drawn first, and
// first is unreachable — the draw path paints terrain and static art before the
// first overlay pass exists.
var BlockedCellColor = color.RGBA{R: 0x60, G: 0x00, B: 0x00, A: 0x60}

// The marker glyphs at the native CellSize pixels per cell: a cross whose arms
// reach <kind>ArmRadius pixels either side of the anchor cell's centre and are
// <kind>ArmThickness pixels across. All scale with the pixel scale (scaleDim).
//
// The unit cross is deliberately smaller than the object cross, and is in fact a
// strict subset of it at every scale. That is what makes the draw order
// load-bearing: with units drawn after objects a coincident pair reads as a cyan
// cross nested in a yellow one, but with the order reversed the unit marker
// would be covered completely rather than merely partly — indistinguishable from
// no unit at all (0009 R-4).
//
// The static cross continues that nesting downward: radius 3 by thickness 1
// is a strict subset of the unit cross, hence of the object cross too, at
// every scale — scaleDim is monotone in its first argument and the
// thickness is shared with the unit glyph.
const (
	markerArmRadius    = 6
	markerArmThickness = 3

	unitArmRadius    = 4
	unitArmThickness = 1

	staticArmRadius    = 3
	staticArmThickness = 1
)

// entityMarkerSide is the entity glyph's edge length at the native CellSize
// pixels per cell: a filled square 10 pixels across, scaling with the pixel
// scale by the same scaleDim every arm above uses.
//
// 10 is below CellSize, and scaleDim is monotone and linear in its first
// argument, so scaleDim(10, cellpx) <= cellpx at every scale — the square
// never leaves the cell it stands on. That is what makes drawing it under
// the crosses safe: an entity can cover its OWN cell's diagnostics, which is
// the pass order's business, but it can never reach a neighbouring cell's.
const entityMarkerSide = 10

// selectionRimThickness is how thick the selection highlight's border is at
// the native CellSize pixels per cell: 2 pixels, scaling with the pixel
// scale by the same scaleDim every glyph above uses.
//
// The rim is HOLLOW, and that is the whole of why this constant is small
// rather than a side length. A border two pixels thick leaves the cell's
// interior untouched, which is what lets this pass be drawn last.
const selectionRimThickness = 2

// cellGridThickness is how thick the diagnostic cell lattice's outline is at
// the native CellSize pixels per cell: one pixel, scaling by the same
// scaleDim.
//
// It is THINNER THAN THE RIM ON PURPOSE, and the two stand on the same
// rectangle. So a selected unit's rim is the same box drawn heavier, and an
// owner reading one against the other is comparing a line with a line rather
// than judging whether two independent glyphs happen to coincide — which
// is the whole of what the lattice is for.
const cellGridThickness = 1

// CellGridColor is the faint wash the diagnostic cell lattice is drawn in
// — the SECOND translucent value in this file, and it carries
// BlockedCellColor's constraint for BlockedCellColor's reason.
//
// IT IS PREMULTIPLIED, so no channel may exceed the alpha. A brighter-looking
// {0xc0, 0xc8, 0xd8, 0x38} is not a paler grey here, it is an invalid colour
// whose rendering is undefined rather than wrong in a direction the eye could
// correct. There are now two values in this file under that rule and five that
// are opaque, which is why it is written out at both rather than once.
//
// 0x38 of 0xff is weak enough that the lattice reads UNDER everything: the
// terrain and the static art are already painted before the first pass of the
// slice exists, every glyph and sprite is drawn over it, and the selection rim
// — opaque white, twice as thick, on the very same rectangle — is unmistakably
// the brighter line. A cool grey rather than a hue, because all five glyph
// colours are hues and a sixth would read as a sixth kind of marker.
var CellGridColor = color.RGBA{R: 0x2c, G: 0x30, B: 0x38, A: 0x38}

// AnchorCell converts a placed object's or unit's fixed-point anchor to its
// integral map cell: the /256 coordinates are shifted down by 8, so the cell is
// (X>>8, Y>>8).
//
// This is the only place that conversion lives, and it is the same conversion
// for both placement kinds — the game's own loader applies the identical bare
// arithmetic shift to type-4 object and type-6 unit anchors, subtracting no
// origin and adding no border inset (research ALM-PLACE-033). It takes plain
// unsigned integers rather than a map record, so no formats type crosses into
// the render tier — the caller reads alm.Map.Objects or alm.Map.Units and passes
// the two coordinates. The result is never negative: the inputs are unsigned and
// the shift is logical.
func AnchorCell(x, y uint32) (col, row int) {
	return int(x >> 8), int(y >> 8)
}

// ObjectMarkerRects returns the marker glyph for one anchor cell as at most two
// half-open rectangles — the horizontal arm then the vertical arm — already
// clipped to the map's pixel extent.
//
// cellpx is the pixels per cell of the target image (CellSize at native scale).
// For anchor cell (col, row) the centre is (col*cellpx + cellpx/2, row*cellpx +
// cellpx/2); the arms reach scaleDim(markerArmRadius) either side of it and are
// scaleDim(markerArmThickness) across, a centred strip of thickness t occupying
// [c-t/2, c-t/2+t) and an arm of radius r occupying [c-r, c+r+1). At cellpx ==
// CellSize this is exactly [cx-6, cx+7) x [cy-1, cy+2) and [cx-1, cx+2) x
// [cy-6, cy+7).
//
// Every arm is intersected with the map pixel rectangle and dropped when
// that leaves it empty, so the result never reaches outside the map.
// image.Rectangle.Intersect truncates each edge to the overlap rather than
// moving one, so clipping cannot synthesise an edge the glyph did not have.
//
// An off-map anchor returns nil before any geometry is built, and so does an
// invalid scale (cellpx < 1) — no geometry, never a panic. All the
// arithmetic is integer, and the result slice has a fixed capacity of two,
// so nothing allocated here grows with the coordinate magnitude.
func ObjectMarkerRects(col, row, cols, rows, cellpx int) []image.Rectangle {
	return markerRects(col, row, cols, rows, cellpx, 0, 0, markerArmRadius, markerArmThickness)
}

// UnitMarkerRects returns the marker glyph for one placed unit's anchor cell,
// under exactly the rules ObjectMarkerRects documents above — at most two
// half-open rectangles, horizontal arm then vertical, already clipped to the
// map's pixel extent, nil for an off-map anchor or an invalid scale.
//
// The only difference is the glyph's size: the arms reach scaleDim(4) either
// side of the centre and are scaleDim(1) across, so at cellpx == CellSize
// this is exactly [cx-4, cx+5) x [cy, cy+1) and [cx, cx+1) x [cy-4, cy+5).
//
// Note where the centred-strip rule actually bites: scaleDim(1, cellpx) reaches
// 2 only at cellpx >= 48, so at the native 32 and below the strip is a single
// pixel and the centring offset is zero. At cellpx 64 (an integer scale of 2,
// which the PNG tool renders routinely) the thickness is even and the strip sits
// one pixel toward the low side of the centre pixel. That is the exact intended
// contract, not a rounding accident.
func UnitMarkerRects(col, row, cols, rows, cellpx int) []image.Rectangle {
	return markerRects(col, row, cols, rows, cellpx, 0, 0, unitArmRadius, unitArmThickness)
}

// StaticMarkerRects returns the marker glyph for one static-object cell, under
// exactly the rules ObjectMarkerRects documents above — at most two half-open
// rectangles, horizontal arm then vertical, already clipped to the map's pixel
// extent, nil for an off-map cell or an invalid scale.
//
// The only difference is the glyph's size: the arms reach scaleDim(3) either
// side of the centre and are scaleDim(1) across, so at cellpx == CellSize
// this is exactly [cx-3, cx+4) x [cy, cy+1) and [cx, cx+1) x [cy-3, cy+4).
//
// It takes a CELL and the canvas alone — no class field, no frame size.
// WHICH cells are marked comes from the built placement list; WHERE a mark
// goes comes from the cell, through MarkerAnchor, and from nothing the art
// knows. That is the second derivation the sprite's own anchor is measured
// against, and it is why this is a sibling of the two glyphs above rather
// than a wrapper over the sprite anchor.
//
// Being radius 3 by thickness 1 it is a strict subset of the unit cross, and
// so of the object cross, at every scale — which is what lets the static
// pass run LAST of the three without covering either.
func StaticMarkerRects(col, row, cols, rows, cellpx int) []image.Rectangle {
	return markerRects(col, row, cols, rows, cellpx, 0, 0, staticArmRadius, staticArmThickness)
}

// EntityMarkerRects returns the marker glyph for one simulated entity's cell as
// at most ONE half-open rectangle — a filled square, not a cross — already
// clipped to the map's pixel extent, and nil for an off-map cell or an invalid
// scale (cellpx < 1).
//
// It has the signature the three cross builders have, because the caller
// that places a glyph in a view takes exactly that signature and this glyph
// goes through the same placement, the same height lift, the same camera
// transform and the same view cull as the other three. What differs is only
// the shape.
//
// The square is scaleDim(entityMarkerSide) across on both axes and is centred on
// MarkerAnchor's point by the same centred-strip rule a cross's thickness uses —
// a strip of width s about c occupying [c-s/2, c-s/2+s) — applied to BOTH axes
// rather than to one. At cellpx == CellSize that is exactly [cx-5, cx+5) x
// [cy-5, cy+5): 10 pixels on a side.
//
// Its side never exceeds cellpx (see entityMarkerSide), so the square lies
// wholly inside its own cell and the map-extent clip below cannot trim an in-map
// one. The clip is applied anyway rather than assumed away, so that this builder
// and the cross builder are clipped by one rule instead of two, and so that the
// inertness is a property that can be TESTED rather than a step that was skipped
// (0020 SC-6).
//
// There is deliberately no DrawEntityMarkers twin on the raster path: the
// offline PNG tool draws record markers, not simulated state, and an entity
// layer there is out of scope (0020's out-of-scope list).
func EntityMarkerRects(col, row, cols, rows, cellpx int) []image.Rectangle {
	if markerCellRejected(col, row, cols, rows, cellpx) {
		return nil
	}

	cx, cy := MarkerAnchor(col, row, cellpx, 0, 0)
	s := scaleDim(entityMarkerSide, cellpx)
	square := image.Rect(cx-s/2, cy-s/2, cx-s/2+s, cy-s/2+s)

	clipped, ok := clipToMapRect(square, cols, rows, cellpx, 0)
	if !ok {
		return nil
	}
	return []image.Rectangle{clipped}
}

// SelectionMarkerRects returns the selected unit's highlight for one cell as at
// most FOUR half-open rectangles — the border of that cell's own footprint, top
// strip, bottom strip, then the two sides between them — already clipped to the
// map's pixel extent, and nil for an off-map cell or an invalid scale
// (cellpx < 1).
//
// It has the signature the four glyph builders above have, and answers the
// family's two shared rules on their terms: markerCellRejected before any
// geometry is built, clipToMapRect on every piece. The caller that places a
// glyph in a view takes exactly that signature, so the highlight goes
// through the same placement, the same height lift, the same camera
// transform and the same view cull as the other four. What differs is only
// the shape — and, unlike every glyph above, it is not centred on
// MarkerAnchor at all.
//
// THE GEOMETRY IS THE CELL'S OWN FOOTPRINT, NOT AN ANCHOR AND A RADIUS. For
// cell (col,row) the footprint is [col*cellpx, (col+1)*cellpx) x
// [row*cellpx, (row+1)*cellpx), and the rim is its border t =
// scaleDim(selectionRimThickness, cellpx) pixels thick:
//
//	top    = [x0, x1) x [y0,   y0+t)      full width
//	bottom = [x0, x1) x [y1-t, y1)        full width
//	left   = [x0,   x0+t) x [y0+t, y1-t)  between the two
//	right  = [x1-t, x1)   x [y0+t, y1-t)  between the two
//
// At cellpx == CellSize, t is 2 and cell (0,0) yields exactly [0,32) x [0,2),
// [0,32) x [30,32), [0,2) x [2,30) and [30,32) x [2,30).
//
// THE FOUR STRIPS ARE DISJOINT AND THE INTERIOR IS UNTOUCHED. A filled
// footprint would have to run first and would then be hidden by the very art
// it marks.
//
// THE STRIPS ARE CLAMPED, not assumed to fit. The vertical extents are built so
// that no strip can cross another however coarse the scale: the top strip never
// passes the footprint's bottom edge, the bottom strip never starts above where
// the top one ended, and the sides occupy exactly the band left between them.
// At cellpx 3 and above 2*t < cellpx and all four strips are present; at cellpx
// 2 the two full-width strips tile the footprint and the sides are empty; at
// cellpx 1 the single remaining strip IS the footprint. Those two degenerate
// scales are unreachable from either renderer — the window builds every glyph at
// the native CellSize and only then scales by the camera's zoom, and the raster
// tool's -scale is at least 1 — but an empty strip is dropped rather than
// returned inverted, so no caller can be handed a rectangle whose corners have
// been silently swapped by image.Rect.
//
// All the arithmetic is integer, and the result slice has a fixed capacity of
// four, so nothing allocated here grows with the coordinate magnitude. A
// rejected cell returns before the slice exists and so allocates nothing.
func SelectionMarkerRects(col, row, cols, rows, cellpx int) []image.Rectangle {
	return footprintRimRects(col, row, cols, rows, cellpx, selectionRimThickness)
}

// CellGridRects returns the diagnostic cell lattice's outline for one cell,
// under exactly the rules SelectionMarkerRects documents above — at most
// four half-open strips bordering that cell's own footprint, already clipped
// to the map's pixel extent, nil for an off-map cell or an invalid scale.
//
// The only difference is the thickness: scaleDim(1) where the rim takes
// scaleDim(2). It is the SAME RECTANGLE bordered more finely, which is what
// makes "a selected unit's rim lands exactly on one lattice cell" a property
// of there being one footprint rather than of two glyphs agreeing.
//
// It is a diagnostic and asserts nothing about the original game: nothing
// published describes a cell lattice drawn over the original's map, and the
// ledgers were swept for one.
func CellGridRects(col, row, cols, rows, cellpx int) []image.Rectangle {
	return footprintRimRects(col, row, cols, rows, cellpx, cellGridThickness)
}

// footprintRimRects is the shared builder behind both rims: the border of
// one cell's footprint at a native thickness, clipped by the family's own
// rule.
//
// It exists so that "the rim and the lattice stand on one rectangle" is
// structural. Written twice, the two would agree until the next change to
// either — which is exactly how the pick and the draw came apart in the first
// place, one story earlier and one tier up.
//
// The strips are CLAMPED, not assumed to fit, and the clamp is the rim's own:
// the top strip never passes the footprint's bottom edge, the bottom strip
// never starts above where the top one ended, and the sides occupy exactly the
// band left between them. An empty strip is dropped rather than returned
// inverted, so no caller is handed a rectangle whose corners image.Rect has
// silently swapped.
func footprintRimRects(col, row, cols, rows, cellpx, thickness int) []image.Rectangle {
	foot, ok := CellFootprint(col, row, cols, rows, cellpx)
	if !ok {
		return nil
	}
	t := scaleDim(thickness, cellpx)

	x0, y0 := foot.Min.X, foot.Min.Y
	x1, y1 := foot.Max.X, foot.Max.Y
	topHi := min(y0+t, y1)
	botLo := max(y1-t, topHi)

	strips := [4]image.Rectangle{
		image.Rect(x0, y0, x1, topHi),      // top, full width
		image.Rect(x0, botLo, x1, y1),      // bottom, full width
		image.Rect(x0, topHi, x0+t, botLo), // left, between the two
		image.Rect(x1-t, topHi, x1, botLo), // right, between the two
	}

	out := make([]image.Rectangle, 0, len(strips))
	for _, s := range strips {
		if s.Empty() {
			continue
		}
		if clipped, ok := clipToMapRect(s, cols, rows, cellpx, 0); ok {
			out = append(out, clipped)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// CellFootprint returns the whole rectangle one cell occupies, already clipped
// to the map's pixel extent:
//
//	[col*cellpx, (col+1)*cellpx) x [row*cellpx, (row+1)*cellpx)
//
// ok is false for an off-map cell or an invalid scale (cellpx < 1), by the
// family's own rejection, and the rectangle is intersected with the map
// extent rather than assumed to lie inside it — for an in-map cell that
// intersection is an identity, and it is applied anyway so that one rule
// clips every piece of geometry in this file.
//
// IT IS THE FILE'S ONE ANSWER TO "WHERE IS THIS CELL", and four things read
// it. The blocked tint FILLS it. The selection rim BORDERS it. The
// diagnostic lattice borders it more finely. And the window's hit test
// PLACES it and asks whether a gesture met it — so the rectangle a unit is
// picked by and the rectangle its highlight is drawn on are one expression,
// not two that agree.
//
// It was named for the first of those four until 0058, which is the bug this
// rename fixes: a value named after one consumer is a value the next three
// re-derive. The window's pick did exactly that, on the flat lattice, and drifted
// from the drawing by the whole of the terrain's relief.
//
// IT IS SINGULAR WHERE EVERY GLYPH BUILDER BESIDE IT RETURNS A SLICE. A glyph's
// cell list is a RECORD list — the placed objects, the placed units, the cells a
// static resolves — hundreds of entries on a large map. This one's callers walk
// a PLANE or a view: the impassable ring alone is 3840 cells on a 128x128 map
// before any water, mountain or scenery. A builder returning a one-element slice
// would heap-allocate once per cell on every frame drawn; a caller wanting one
// rectangle holds it in a local.
func CellFootprint(col, row, cols, rows, cellpx int) (image.Rectangle, bool) {
	if markerCellRejected(col, row, cols, rows, cellpx) {
		return image.Rectangle{}, false
	}
	x0, y0 := col*cellpx, row*cellpx
	return clipToMapRect(image.Rect(x0, y0, x0+cellpx, y0+cellpx), cols, rows, cellpx, 0)
}

// MarkerAnchor returns the point one cell's marker glyph is centred on: the
// centre of the cell's lattice square, translated by the canvas origin and by
// the cell's own height lift.
//
//	x = col*cellpx + cellpx/2
//	y = row*cellpx + cellpx/2 + offsetY + liftY
//
// This is the cross centre markerRects has computed since 0008, extracted
// verbatim and given a name. markerRects calls it, so the three glyphs and
// this function are one derivation and not two that happen to agree: every
// marker in the tree stands where this says, at every scale and on every
// canvas.
//
// WHAT IT DOES NOT TAKE IS THE POINT OF THE SIGNATURE. There is no class field
// here — no canvas, no centre pixel — and no frame size. A marker's position is
// a function of its CELL and the canvas it is drawn on, and of nothing the art
// knows, which is what makes it an INDEPENDENT second derivation of the same
// ground point the sprite anchor computes from the class and the frame
// (StaticAnchor, statics.go). The two never call each other and neither reads
// the other's inputs.
//
// That separation is the whole discriminating power of the static-object
// layer (spec's constraints table). Routed through one shared helper, a
// wrong anchor would move the cross and the art TOGETHER: the screen would
// stay perfectly self-consistent and perfectly wrong, and the disagreement
// would stop being representable at all. Kept apart, it shows as a tree
// standing a tile away from its own cross — something a human sees at once
// (AC-9) and something a tool can refuse to write.
//
// What that comparison discriminates is BOUNDED, and the bound is part of the
// contract rather than a shortcoming to be fixed here: the two anchor terms
// cancel algebraically, so a disagreement catches a wrong cell-to-world mapping
// or a wrong lift/origin lookup or sign, and cannot catch a wrong
// CenterX/CenterY convention or a canvas-for-frame mix-up. Those are AC-9's,
// where a human sees the art itself standing away.
//
// offsetY and liftY are in OUTPUT pixels, the same units as cellpx, and both are
// already carrying whatever sign and scale the caller's own sources use — this
// function performs no negation and no scaling. A caller holding a Render and a
// Projection therefore passes -r.OriginY*scale and -p.AnchorHeight(col, row),
// where the sprite side is handed the un-negated height and origin and performs
// the one negation the formula that owns them contains. The two stay separate
// expressions over the same two sources, which is why a wrong sign on either
// side shows in the comparison instead of cancelling out of it.
//
// It is a pure integer function of its arguments and is total: no bound is
// checked and none is needed, because there is no image, no slice and no
// allocation here — a cell outside the map, or a cellpx below 1, yields a point
// like any other, and it is markerCellRejected, asked by every glyph builder in
// this file, that rejects those before any geometry is built.
func MarkerAnchor(col, row, cellpx, offsetY, liftY int) (x, y int) {
	return col*cellpx + cellpx/2, row*cellpx + cellpx/2 + offsetY + liftY
}

// markerRects is the shared cross generator behind all three CROSS kinds: the
// arm geometry is identical for objects, units and statics and differs only in
// the native radius and thickness, so it is written once here rather than three
// times.
//
// The two rules it does not own are the ones the square shares with it: the
// invalid-scale and off-map rejections (markerCellRejected) and the
// map-extent clip (clipToMapRect) belong to every glyph in the family, cross
// or not, so they sit beside this function rather than inside it. What
// stayed here is exactly what is a cross's own.
//
// radius and thickness are the glyph's dimensions at the native CellSize; they
// are supplied by this package's own entry points and are never caller-chosen,
// so no caller can produce a glyph the specs do not define.
//
// offsetY translates the whole cell lattice down by that many output pixels,
// for a canvas whose row 0 stands for a native row other than 0. The glyph's
// centre and the map-extent rectangle take the SAME translation, so which
// arms survive the clip cannot depend on the offset: an arm kept at offset 0
// is kept at every offset, and one the map rect dropped is dropped at every
// offset. Only the anchor's row moves, never its cell — the off-map test
// is on the cell indices and runs before any of this, so a translation can
// neither admit an off-map anchor nor drop an in-map one.
//
// liftY is 0015's own per-marker height offset: it reaches ONLY cy, the
// glyph's own centre, and never the map rectangle, which stays built from
// offsetY alone. That asymmetry is the whole point of the parameter, not an
// oversight — offsetY also positions mapRect, the map-extent rectangle
// every arm is clipped against (the stage-1 clip), so a value that moved the
// clip along with the marker would let an over-lifted glyph escape a
// boundary it should have been cut against, and "is this glyph on the map"
// would silently stop meaning what it says. A liftY of 0 makes this call
// identical to the pre-0015 geometry, which is what every existing entry
// point below still passes.
func markerRects(col, row, cols, rows, cellpx, offsetY, liftY, radius, thickness int) []image.Rectangle {
	if markerCellRejected(col, row, cols, rows, cellpx) {
		return nil
	}

	cx, cy := MarkerAnchor(col, row, cellpx, offsetY, liftY)
	r := scaleDim(radius, cellpx)
	t := scaleDim(thickness, cellpx)

	arms := [2]image.Rectangle{
		image.Rect(cx-r, cy-t/2, cx+r+1, cy-t/2+t), // horizontal
		image.Rect(cx-t/2, cy-r, cx-t/2+t, cy+r+1), // vertical
	}

	out := make([]image.Rectangle, 0, len(arms))
	for _, arm := range arms {
		if clipped, ok := clipToMapRect(arm, cols, rows, cellpx, offsetY); ok {
			out = append(out, clipped)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// markerCellRejected reports whether a marker on (col, row) contributes no
// geometry at all: an invalid pixel scale, or a cell outside the cols x rows
// map. It is the first shared rule of the glyph family — the cross builder
// above and EntityMarkerRects both ask it, so "which cells get a marker" has
// one answer for every glyph kind rather than one per kind.
//
// The bounds are half-open on the high side — col == cols is already off the map
// — because cols and rows are counts, and cell indices run 0..cols-1. The low
// side is guarded too: a negative cell is not merely out of range, it would index
// backwards into the pixel lattice.
func markerCellRejected(col, row, cols, rows, cellpx int) bool {
	return cellpx < 1 || col < 0 || row < 0 || col >= cols || row >= rows
}

// clipToMapRect intersects one already-built glyph rectangle with the map's
// pixel extent and reports whether anything survives. It is the second
// shared rule of the glyph family: every glyph in this package is cut
// against the same rectangle by the same test, so no glyph can reach outside
// the map.
//
// The map rectangle takes offsetY — the lattice translation of a canvas whose
// row 0 stands for a native row other than 0 — because the glyph's own centre
// took it too: the two move together, so which parts survive this clip cannot
// depend on the offset. It does NOT take the per-cell height lift, and that
// asymmetry is the point rather than an oversight; markerRects' own doc comment
// above states why.
//
// ok is false exactly when the intersection is empty. Empty() is the semantic
// test — a piece that survives the clip must have positive area — and is not
// there to distinguish an empty-but-non-zero result: Intersect normalises every
// empty overlap, edge-only included, to the zero rectangle. Relying on that
// normalisation instead would still be wrong, because a glyph piece is only
// meaningful when it has area.
//
// image.Rectangle.Intersect truncates each edge to the overlap rather than moving
// one, so clipping can shrink a glyph but cannot synthesise an edge it did not
// have.
func clipToMapRect(r image.Rectangle, cols, rows, cellpx, offsetY int) (image.Rectangle, bool) {
	mapRect := image.Rect(0, offsetY, cols*cellpx, rows*cellpx+offsetY)
	clipped := r.Intersect(mapRect)
	return clipped, !clipped.Empty()
}

// scaleDim scales a native marker dimension to cellpx pixels per cell, rounding
// to nearest and never yielding less than one pixel: max(1, round(n*cellpx/32)),
// where round(v/32) is (v+16)/32 for a non-negative v (Go truncates toward zero,
// which is the floor here).
func scaleDim(n, cellpx int) int {
	v := (n*cellpx + CellSize/2) / CellSize
	if v < 1 {
		return 1
	}
	return v
}

// DrawObjectMarkers stamps a marker over an already-composited image at every
// in-map anchor cell, leaving every other pixel untouched.
//
// It runs after Composite/CompositeLit rather than inside them, so the
// overlay composes with the unshaded and the shaded terrain path alike and
// none of the compositors change. A caller that never invokes it gets a
// byte-identical terrain image.
//
// Clipping is two-stage: ObjectMarkerRects has already intersected each arm
// with the map extent, and each surviving arm is then intersected with the
// image bounds. When img is the full map extent the two coincide; when it is
// smaller the second stage truncates.
//
// img must be anchored at the origin — img.Bounds().Min == image.Point{}. The
// arms are in map-pixel space, so intersecting them with a rectangle that starts
// elsewhere would compare two different coordinate frames. Every compositor in
// this package allocates image.Rect(0, 0, w, h), so the condition holds for the
// production callers. A nil image and an empty cell list are both no-ops.
//
// The image is assumed to begin at native row 0, which is the flat raster's
// canvas: this is DrawObjectMarkersAt's offsetY == 0 case, and a height-displaced
// render needs that entry point instead.
func DrawObjectMarkers(img *image.RGBA, cells []image.Point, cols, rows, cellpx int) {
	DrawObjectMarkersAt(img, cells, cols, rows, cellpx, 0)
}

// DrawObjectMarkersAt stamps the object markers over an already-composited image
// whose row 0 stands for a native row other than 0 — every height-displaced
// render — translating the whole cell lattice down by offsetY output pixels and
// changing nothing else about the glyph.
//
// offsetY is in OUTPUT pixels, the same units as cellpx, and a caller
// holding a Render passes -r.OriginY * scale: the projected canvas begins at
// native row minV, so native row row*32 lands at row*32*scale - minV*scale
// and the lattice takes the terrain's own translation. Either sign occurs,
// and a NEGATIVE origin — hence a positive offsetY — is the ordinary
// case rather than the exotic one, minV being the least row*32 - h(c,row)
// over the mesh: any map whose top row carries a positive altitude reports
// one.
//
// What moves and what does not:
//
//   - The glyph and the map-extent clip move together, so the arms that survive
//     that clip are the offset-0 arms translated. Moving one without the other
//     would either shave the lattice's first or last row away or let a marker
//     escape the map extent.
//   - The clip against img.Bounds() does NOT move: the image IS the translated
//     canvas, so its bounds are already in the translated frame. That second clip
//     is what truncates an in-map anchor whose glyph leaves the canvas once the
//     translation has carried the lattice past an edge of it.
//   - The off-map drop stays a cell-index test, so an anchor outside the map
//     contributes nothing however the translation happens to place its pixels
//     over the canvas.
//
// The markers stay on the UN-DISPLACED cell lattice: every cell takes the one
// offsetY, none is displaced by its own altitudes onto the quad the terrain was
// drawn as, so on a projected image a marker does not register with the terrain
// beneath it — this story's stated diagnostic limitation (0012 R-4). Placing a
// sprite on the displaced surface is work item 0015's contract, and needs
// geometry this story does not have.
func DrawObjectMarkersAt(img *image.RGBA, cells []image.Point, cols, rows, cellpx, offsetY int) {
	drawMarkers(img, cells, cols, rows, cellpx, offsetY, nil, markerArmRadius, markerArmThickness, MarkerColor)
}

// DrawObjectMarkersAtHeights stamps the object markers exactly as
// DrawObjectMarkersAt does — the same two-stage clip, the same
// origin-anchored precondition on img, the same nil-image and empty-list
// no-ops — except each marker's centre additionally shifts by liftY(col,
// row) output pixels before either clip stage runs.
//
// liftY is read at most once per cell (nil-safe: a nil liftY is the same as
// one that always returns 0, which makes this call identical to
// DrawObjectMarkersAt). It is the caller's job to have already folded in
// whatever sign and scale convention its own height source uses — this
// function performs no further scaling.
//
// The lift reaches only the glyph's own centre, never the map-extent
// rectangle the stage-1 clip in markerRects builds from offsetY alone: an
// anchor lifted enough to reach past the map's edge is clipped at the
// UNLIFTED extent (0015 SC-7), and separately at img.Bounds() once lifted,
// exactly as markerRects' own doc comment states. This is the seam
// DrawObjectMarkersAt itself cannot serve without widening every one of its
// existing 0008/0009/0012 call sites and tests with a parameter they never
// use — which is why this is a new function rather than a wider one.
func DrawObjectMarkersAtHeights(img *image.RGBA, cells []image.Point, cols, rows, cellpx, offsetY int, liftY func(col, row int) int) {
	drawMarkers(img, cells, cols, rows, cellpx, offsetY, liftY, markerArmRadius, markerArmThickness, MarkerColor)
}

// DrawUnitMarkers stamps a unit marker over an already-composited image at
// every in-map anchor cell, under exactly the rules DrawObjectMarkers
// documents above: the same two-stage clip, the same origin-anchored
// precondition on img, the same nil-image and empty-list no-ops, and the
// same untouched pixels everywhere else (so a caller that never invokes it
// gets a byte-identical image).
//
// It is meant to run AFTER DrawObjectMarkers, which is what gives the
// composed render its terrain -> objects -> units order. The store is an
// unconditional opaque SetRGBA, so where a unit arm overlaps an object arm
// the unit colour wins. Because the unit cross is a strict subset of the
// object cross, running the two in the wrong order does not merely dim the
// unit marker, it hides it entirely (0009 R-4) — the order is
// load-bearing, not cosmetic.
func DrawUnitMarkers(img *image.RGBA, cells []image.Point, cols, rows, cellpx int) {
	DrawUnitMarkersAt(img, cells, cols, rows, cellpx, 0)
}

// DrawUnitMarkersAt stamps the unit markers on a canvas translated by offsetY
// output pixels, under exactly the rules DrawObjectMarkersAt documents above: the
// glyph and the map-extent clip move together, the img.Bounds() clip does not
// move, the off-map drop stays a cell-index test, and the lattice stays
// un-displaced.
//
// It is meant to run AFTER DrawObjectMarkersAt at the SAME offsetY, which
// keeps the terrain -> objects -> units order on a height-displaced canvas.
// Two different offsets would not merely mis-place one overlay, they would
// break the nesting the order relies on: the unit cross is a strict subset
// of the object cross only where the two are concentric.
func DrawUnitMarkersAt(img *image.RGBA, cells []image.Point, cols, rows, cellpx, offsetY int) {
	drawMarkers(img, cells, cols, rows, cellpx, offsetY, nil, unitArmRadius, unitArmThickness, UnitMarkerColor)
}

// DrawUnitMarkersAtHeights stamps the unit markers under exactly the rules
// DrawObjectMarkersAtHeights documents above: the same per-cell liftY shift
// applied before either clip stage, the same nil-safe callback, the same
// two-stage clip and origin-anchored precondition on img.
//
// It is meant to run AFTER DrawObjectMarkersAtHeights at the SAME offsetY
// and the SAME liftY function, which keeps the terrain -> objects -> units
// order on a height-lifted canvas exactly as DrawUnitMarkersAt keeps it on a
// height-displaced one.
func DrawUnitMarkersAtHeights(img *image.RGBA, cells []image.Point, cols, rows, cellpx, offsetY int, liftY func(col, row int) int) {
	drawMarkers(img, cells, cols, rows, cellpx, offsetY, liftY, unitArmRadius, unitArmThickness, UnitMarkerColor)
}

// DrawStaticMarkersAt stamps the static-object markers on a canvas translated by
// offsetY output pixels, under exactly the rules DrawObjectMarkersAt documents
// above: the glyph and the map-extent clip move together, the img.Bounds() clip
// does not move, the off-map drop stays a cell-index test, and the lattice stays
// un-displaced.
//
// The cells are the ones a built placement list resolved to a drawable frame
// — a census the marker geometry itself knows nothing about, and takes no
// position from. offsetY = 0 is the flat geometry, which is why this layer
// needs no third, un-translated entry point of its own: the two callers are
// a flat raster at 0 and a projected one at -OriginY*scale.
func DrawStaticMarkersAt(img *image.RGBA, cells []image.Point, cols, rows, cellpx, offsetY int) {
	drawMarkers(img, cells, cols, rows, cellpx, offsetY, nil, staticArmRadius, staticArmThickness, StaticMarkerColor)
}

// DrawStaticMarkersAtHeights stamps the static-object markers under exactly the
// rules DrawObjectMarkersAtHeights documents above: the same per-cell liftY
// shift applied before either clip stage, the same nil-safe callback read at
// most once per cell, the same two-stage clip and origin-anchored precondition
// on img.
//
// It is meant to run AFTER the object and unit height-lifted passes at the SAME
// offsetY and the SAME liftY function, which keeps the objects -> units ->
// statics order on a height-lifted canvas exactly as DrawStaticMarkersAt keeps
// it on a flat one.
//
// liftY is the caller's already-negated, already-scaled height offset —
// the marker path's convention since 0015, and NOT the un-negated cell
// height the sprite side's builder takes. Written out at the two call sites
// over the same projection, the two remain separate expressions of one
// geometry: a wrong sign on either side moves that side's ground point
// alone, which is precisely what the comparison this glyph exists to make
// visible would catch.
func DrawStaticMarkersAtHeights(img *image.RGBA, cells []image.Point, cols, rows, cellpx, offsetY int, liftY func(col, row int) int) {
	drawMarkers(img, cells, cols, rows, cellpx, offsetY, liftY, staticArmRadius, staticArmThickness, StaticMarkerColor)
}

// drawMarkers is the shared rasterizer behind both marker kinds: the two-stage
// clip and the opaque per-pixel store are identical for objects and units and
// differ only in the glyph dimensions and the colour.
//
// offsetY reaches only markerRects, which is where the lattice translation lives;
// the bounds clip and the store below are in the image's own frame and need no
// knowledge of it.
//
// liftAt is 0015's per-cell height source: nil-safe (a nil liftAt behaves as
// a lift of 0 everywhere, which is what every pre-0015 caller below still
// passes) and read AT MOST ONCE per cell, before markerRects is called for
// that cell — so a caller's liftAt can be an arbitrarily expensive lookup
// without paying for it twice per cell (once per arm). The resulting
// per-cell lift feeds markerRects' own liftY parameter and inherits its
// contract: reaches the glyph's centre only, never the map-extent clip.
func drawMarkers(img *image.RGBA, cells []image.Point, cols, rows, cellpx, offsetY int, liftAt func(col, row int) int, radius, thickness int, c color.RGBA) {
	if img == nil {
		return
	}
	bounds := img.Bounds()
	for _, cell := range cells {
		lift := 0
		if liftAt != nil {
			lift = liftAt(cell.X, cell.Y)
		}
		for _, arm := range markerRects(cell.X, cell.Y, cols, rows, cellpx, offsetY, lift, radius, thickness) {
			clipped := arm.Intersect(bounds)
			if clipped.Empty() {
				continue
			}
			for y := clipped.Min.Y; y < clipped.Max.Y; y++ {
				for x := clipped.Min.X; x < clipped.Max.X; x++ {
					img.SetRGBA(x, y, c)
				}
			}
		}
	}
}
