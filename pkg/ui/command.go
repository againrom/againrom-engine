package ui

import (
	"image"
	"math"
	"slices"
)

const controlSpiritSpellID uint32 = 25

// TapSlop is how far the cursor may travel while the primary button is held,
// in screen pixels, for the press and its release to still be a TAP rather
// than a DRAG.
//
// It is measured as a PATH LENGTH, not as the distance from the press to the
// release: the accumulator behind it is raised on every held tick by |dx| +
// |dy|, the very delta that tick panned the view by, so "the view moved" and
// "this was a drag" are one number that cannot disagree. A press that
// wanders out and back panned that far and is a drag, which is the disclosed
// cost of measuring it this way; four pixels is an eighth of a cell at
// native zoom.
//
// THE TWO SCREENS STILL READ THE CONSTANT AGAINST DIFFERENT FRAMES (round-2
// adversarial review, eleventh pass, judged and left open —
// docs/1005-*/closure.md, TapSlop asymmetry). The mission composes at
// 1024x768 and the shop at 640x480, both 4:3, so a mission frame pixel is
// always 640/1024 = 0.625 of a shop frame pixel on screen: the shop reads a
// tremor as a drag 1.6 times later than the map does, at every window size.
// No claim states a real distance for either screen's own drag tolerance in
// ROM1, so this is not a divergence row. It is an unmatched choice inside
// this implementation's own two drag machines, disclosed here rather than
// fixed, because fixing it changes felt drag behaviour on two screens at
// once and no brief has named which of the two should move.
const TapSlop = 4

// marqueeSlop is the mission map's own marquee threshold, which `AI-INPUT-121`
// gives as `screenW*10/640` -- 10, 12 and 16 pixels at the three shipped screen
// widths -- and states as a STRICT comparison: "a rectangle strictly beyond it
// goes to selection". A release at or under it is a click.
//
// IT IS THE MISSION FRAME'S OWN WIDTH and not the window's. This build composes
// the mission at a fixed frame size and scales that frame to the window
// (`DIV-249`), so the pixels a gesture is measured in are frame pixels; reading
// the window here would make the same hand movement a click at one window size
// and a marquee at another, which is the failure `DIV-249` records for the
// cursor picture and story `1031` for the edge band.
//
// IT REPLACES TapSlop FOR THE BOX/MARQUEE GESTURE ON THIS SCREEN, at every site
// that decides it: the outline (`overlay.go`'s marqueeScreenRects, F1 round 2),
// the cursor (missioncursor.go, which reads the outline) and the release
// (below). On a 1024-wide mission frame the decoded threshold is 16, so a
// tremor this build used to read as a marquee outline -- while the release
// still dispatched it as a click, `sim.Command` and all -- now draws no
// outline either. TapSlop still governs two things on this same screen that
// are not the box gesture: the mission map's own inventory-drag pickup
// (command.go:1163, the doll and pack) and the shop's whole drag, where no
// claim gives a distance. TapSlop's own doc carries the two screens'
// asymmetry, which this narrows to one screen's inventory drag rather than
// closing.
//
// The floor of 1 keeps a degenerate frame from making every release a marquee.
func (v *Viewer) marqueeSlop() int {
	if s := v.frameW * 10 / 640; s > 1 {
		return s
	}
	return 1
}

// selection is the front-end's whole memory of which units are selected:
// entity ids, ascending, REPLACED WHOLE by a tap or a release and by nothing
// else.
//
// ABSENCE IS THE LENGTH, never a sentinel id and no longer a bool beside
// one. No id value is reserved to mean "none", so this package cannot
// CONSTRUCT an id — every one it holds was minted on the far side of the
// seam and arrived in a MapEntity — and the zero value, a nil slice, is
// "nothing selected", which is what a fresh Viewer holds and what a clear
// restores. The element width is uint32, the id's own, so the value crosses
// losslessly both ways where an int would admit values no id can hold; and
// being a builtin it names no simulation type, which is what leaves this
// package able to say WHICH entities without being able to say what an
// entity is.
//
// IT IS ASCENDING BY CONSTRUCTION AND NOTHING SORTS IT. The snapshot arrives in
// ascending id — SetEntities' own contract, and the world's own order behind it
// — so a walk of it keeping the covered units comes out sorted, and a tap
// yields a slice of one. That is what leaves the emission order the set's own
// rather than a step taken to obtain it, which is a step that can be forgotten
// where a walk cannot.
//
// A Go map was rejected for the same property: Go randomises range order per
// process, so the order commands left in would be per-process — which is exactly
// why the shipped commanded-set on the far side of the seam is read by key and
// never ranged.
type selection []uint32

// order is one thing a frame decided to issue: which entity, and —
// depending on its kind — the cell to move it to or the entity to attack.
// No new value type crosses out of this package for it and no simulation
// type is named here either.
//
// IT CARRIES A KIND, AND THE SEAM STILL DOES NOT. Two functions leave this
// package, one per kind, and that division is deliberate; this type is one
// level below them, and it is one type rather than two so that a frame's
// emission is ONE walk. Ascending id is then a property of that walk rather
// than of two lists kept in step, and the two kinds are exclusive per frame
// — a consuming press issues attacks or moves and never both — so a
// second list would be empty on every frame the first was not.
//
// The unused half is the ZERO VALUE and is never read: a move leaves victim at
// zero and an attack leaves x and y at zero, and the reader branches on attack
// before it touches either.
//
// It is only ever meaningful beside the bool decide returns with it. The zero
// value is not "move entity 0 to (0,0)": it is what accompanies a false, and a
// caller reads the bool first.
type order struct {
	kind orderKind

	entity uint32
	x, y   int

	// victim is the entity attacked. A move names no victim and an attack
	// names no cell.
	victim uint32

	spell uint32
	// cell distinguishes an empty-ground cast from an entity-targeted cast.
	cell bool
}

// commandNone, commandPatrol and commandSwarm name the AIMED standing orders
// the player can arm.
//
// ONE BYTE AND NOT TWO BOOLS.
//
// The two CELL-FREE orders are not here, because they are not armed at all:
// they act on the selection the instant their key is pressed, so they have no
// state to hold between frames.
const (
	commandNone uint8 = iota
	commandPatrol
	commandSwarm

	// commandMove is the command panel's own Move cell (docs/1028-command-
	// panel; `AI-PANEL-053`'s armed-mode table entry 2, one of the original's
	// two "always arm" buttons alongside Attack). Unlike commandPatrol and
	// commandSwarm it names no distinct order of its own: `decide` resolves
	// it to a plain move — command 0 — at the order boundary below, so
	// nothing outside this package, and nothing in pkg/game's own seams,
	// ever learns a third command byte exists. It exists only so the panel
	// can hold a Move arm the SAME way it holds Patrol and March, through
	// `armCommand`, with its own cell to show selected while it is up.
	commandMove
	commandDefend
)

// dragNone, dragFromPack and dragFromDoll name the drag machine's own
// origin (1005, "the interactive doll"; Viewer.dragCandKind's own field
// comment): round 1's two sources, and the zero value that means neither.
const (
	dragNone uint8 = iota
	dragFromPack
	dragFromDoll
)

// gesture is one frame's button edges, cursor and press point, as decide
// receives them.
//
// The edges are already classified, and the classification is the impure
// half's: tap is a primary RELEASE the slop accumulator judged a tap rather
// than a drag; boxed is one it judged a drag, on a press the viewer latched
// as a selection rectangle; and right is the secondary button's PRESS EDGE.
// The right button has no level and no anchor anywhere in this package, so
// no drag gesture for it is representable at all.
//
// tap and boxed are EXCLUSIVE by that classification: one release is judged
// once, against one accumulator, so there is no frame on which both are true and
// no state in which a release is neither judged nor discarded.
//
// held is whether a primary press is still in progress as this frame is
// resolved.
//
// x, y is this frame's cursor and px, py the point the current gesture's
// button went down at. One cursor position serves every edge, because they
// are edges of the SAME frame and a frame has one cursor; the press point is
// the viewer's, stored where the gesture began. There is no extent here: it
// arrives inside the camera, so decide takes no second one to disagree with
// it. armed is whether an attack was armed as this frame began. It is STATE
// and not an edge, exactly as held is, and it is read across for held's own
// reason: "what does a secondary press do" is a question about state, and
// asking it inside the one pure function keeps the whole answer there.
//
// The arming KEY is not here. Raising the flag is not a gesture — it reads the
// selection and the local participant and writes nothing else — so it is decided
// beside this function rather than inside it, and what arrives here is only the
// flag's value.
//
// spell is the book's own selection as this frame began, armed's own shape
// and armed's own reason: it is STATE, not an edge, so it is read across
// from the viewer rather than raised by anything in this struct — the
// click that selects a spell is decided in command, beside the swallow the
// inventory window's own click already gets, and never inside this pure
// function.
type gesture struct {
	// Explicit physical attack hit, resolved with the visible sprite identity.
	structure InspectionSubject
	tap       bool
	boxed     bool
	// held is whether the primary button is still down as this gesture is
	// resolved. THERE IS NO `right` FIELD any more: the secondary button orders
	// nothing at all, so no pure decision reads it.
	held      bool
	spell     uint32
	pointCast bool
	x, y      int
	px, py    int

	// cursor IS WHAT DECIDES THE ORDER, and it replaces this struct's own
	// `armed` and `command` fields (`AI-CLICK-050`: a map click is turned into
	// an order by the CURSOR it was made under, not by what it hit). The armed
	// mode selects a cursor through the eight-entry table (`AI-PANEL-053`), the
	// hover cascade selects one when no mode is armed (`AI-CURSOR-226`), and
	// this one string carries either into the dispatch.
	//
	// IT IS STATE AND NOT AN EDGE, exactly as `armed` and `command` were: the
	// name was chosen by the cursor cascade on the tick `step` ran, and what
	// arrives here is that name's value.
	cursor string

	// shift is the Shift latch as this frame's gesture is resolved
	// (`AI-KEYMOD-059`, `AI-SELECT-122`). It is the SELECTION routine's own
	// modifier and reaches nothing else: `AI-CURSOR-226` establishes that
	// Shift changes no cursor on an ordinary hover, so it cannot change which
	// order a click makes.
	shift bool
}

// presentSelected is every selected id the CURRENT SNAPSHOT still holds AND
// still holds selectable, with the entry it holds it as, in the SELECTION'S
// OWN ORDER — which is ascending id, since nothing ever puts an id into
// the set out of order.
//
// ONE PREDICATE SERVES BOTH READERS.
//
// A TERMINAL DEAD ENTRY IS SKIPPED, NOT DROPPED. A downed one and a finishable
// -1..-9 body are kept: they can still be clicked and marked even though the
// simulation refuses their move order. The explicit Selectable bit widens only
// that corpse band and does not turn terminal bodies into candidates.
//
// It walks the selection and looks each id up, rather than walking the
// snapshot and keeping what the selection holds.
func presentSelected(cur selection, ents []MapEntity) []MapEntity {
	var out []MapEntity
	for _, id := range cur {
		for _, e := range ents {
			if e.ID != id {
				continue
			}
			if selectableEntity(e) {
				out = append(out, e)
			}
			break
		}
	}
	return out
}

// selectableEntity is the one selection-population predicate shared by taps,
// boxes, marks and emitted orders. Life keeps its rendering meaning; Selectable
// is only the owner-authored exception for a finishable fallen body (DIV-488).
func selectableEntity(e MapEntity) bool { return e.Life != LifeDead || e.Selectable }

// marked is the ids the two debug keys act on this frame: exactly the units
// presentSelected keeps, in the selection's own ascending order.
//
// IT IS THE SAME FILTER THE MARKS AND THE ORDERS READ, called rather than
// copied, so "one command per marked unit" is the same set the frame drew a rim
// around — one test and three readers. It returns IDS and not entries, because
// what a blow names is an entity and nothing else about it.
//
// It reads and writes nothing: the stored selection is untouched, which is what
// keeps a key press from being a fifth way to replace it.
func (v *Viewer) marked() []uint32 {
	present := presentSelected(v.sel, v.entities)
	if len(present) == 0 {
		return nil
	}
	ids := make([]uint32, 0, len(present))
	for _, e := range present {
		ids = append(ids, e.ID)
	}
	return ids
}

// topAt is the entity a single point names: the lowest id whose DRAWN
// rectangle holds it, and whether any does.
//
// It is a minimum over the whole selectable set and not the first match, so the
// tie is settled over ids and never quietly becomes slice position. Attack and
// cast targeting deliberately use targetAt below: selection and combat have
// different populations at both the finishable and terminal corpse boundaries.
func topAt(ents []MapEntity, at func(MapEntity) (screenRect, bool), x, y float64) (uint32, bool) {
	var best uint32
	hit := false
	for _, e := range picked(ents, at, x, y, x, y) {
		if !hit || e.ID < best {
			best, hit = e.ID, true
		}
	}
	return best, hit
}

// targetAt is the attack/cast hit-test. It deliberately differs from topAt:
// selection reads Selectable while combat reads Untargetable. Control Spirit
// asks with allowCorpse and owns the one exception after the -10 boundary.
func targetAt(ents []MapEntity, at func(MapEntity) (screenRect, bool), x, y float64, allowCorpse bool) (uint32, bool) {
	var best uint32
	hit := false
	for _, e := range ents {
		if e.Untargetable && !allowCorpse {
			continue
		}
		r, ok := at(e)
		if !ok || !r.meets(x, y, x, y) {
			continue
		}
		if !hit || e.ID < best {
			best, hit = e.ID, true
		}
	}
	return best, hit
}

// canArmAttack is the whole of what decides whether an attack may be armed
// over this selection and this local participant.
//
// TWO CLAUSES, AND NEITHER IS ABOUT A UNIT'S CLASS. There must be a unit to
// order — the same filter the orders, the marks and the blow keys read, so "a
// selection that can be ordered to attack" and "a selection an order is emitted
// for" cannot come apart — and the PRIMARY of those, the lowest present id, must
// be owned by the local participant.
//
// The class half is absent on purpose. The clause that once said a capability
// mask over the selected units gates this was retracted when the routine behind
// it was read: it tests no unit class anywhere and compares two player values.
// A control greyed out by class would be reproducing a rule the game does not
// have, which is the harder failure to notice of the two.
//
// A LOCAL PARTICIPANT OF ZERO IS NONE ESTABLISHED and compares against nothing,
// so the gate reduces to its first clause. That is this build's state rather
// than a permanent shape: nothing here performs a session join, and which roster
// slot a participant holds is that path's answer and not the map's. The day one
// exists it pushes a nonzero value and this comparison becomes live with no
// other change anywhere.
func canArmAttack(cur selection, ents []MapEntity, localOwner uint32) bool {
	present := presentSelected(cur, ents)
	if len(present) == 0 {
		return false
	}
	return localOwner == 0 || present[0].Owner == localOwner
}

// meets reports whether this half-open screen rectangle shares a point with
// the CLOSED rectangle [x0,x1] x [y0,y1] a gesture spans.
//
// THE TWO SHAPES ARE DELIBERATELY DIFFERENT. A placed rectangle is
// half-open, because it tiles: adjacent cells share an edge and a pixel on
// it belongs to exactly one of them. A gesture is closed, because it is a
// path the cursor travelled and its far corner is a pixel the user pointed
// at. Put together, a gesture of NO extent — a press and a release at one
// point — is the single point, so the tap below is the box above with both
// corners equal and NOT a second rule that has to be kept in step with it.
//
// Every comparison is stated POSITIVELY, for ScreenToCell's own reason: every
// comparison against a NaN is false, so a non-finite gesture or a non-finite
// placement meets nothing rather than slipping past a negated test.
//
// THE EXTENT IS TESTED FIRST, and that guard is not decoration. The half-open
// reading is only correct for a rectangle with area: at W == 0 the interval
// [X, X) is empty and yet x0 < X && x1 >= X can still hold, so an empty target
// would be hittable; at W < 0 the rectangle is inverted and would be hit by a
// gesture strictly to its LEFT. Both are reachable, because placeArm scales its
// extent by the camera's RAW Zoom field while taking its position through the
// camera's clamped one — a viewer whose Zoom was assigned directly, or left at
// the zero value, produces a finite position with a zero extent. That was
// harmless while such a rectangle was only drawn; it is not harmless now that it
// is a hit target.
func (r screenRect) meets(x0, y0, x1, y1 float64) bool {
	if !(r.W > 0) || !(r.H > 0) {
		return false
	}
	return x0 < r.X+r.W && x1 >= r.X && y0 < r.Y+r.H && y1 >= r.Y
}

// coveredMoreThanHalf reports whether the closed gesture rectangle [x0,x1] x
// [y0,y1] covers STRICTLY MORE THAN HALF of this placed rectangle's own
// area. It is `AI-SELECT-122`'s rectangle test.
//
// EQUALITY FAILS. The decoded test is a signed greater-than against half the
// object's own extent, so a rectangle covering exactly half of a unit does NOT
// take it. That is why the comparison below is doubled rather than divided:
// `2*inter > area` is the same test with no rounding and no halving of an odd
// extent, and it keeps the boundary case decidable in float arithmetic.
//
// IT IS A SECOND TEST AND NOT A REPLACEMENT FOR meets. picked answers which
// entries the gesture touches at all, which is the population; this answers
// which of those the plain rectangle form takes. A one-pixel gesture meets a
// unit and covers far less than half of it, so a marquee of nearly no extent
// takes nothing -- which is correct, because a gesture that small is a TAP and
// never reaches the rectangle form at all.
//
// A rectangle with no extent covers nothing, on meets' own reason: the
// half-open reading is only correct for a rectangle with area, and an inverted
// one would otherwise be "covered" by a gesture beside it.
func (r screenRect) coveredMoreThanHalf(x0, y0, x1, y1 float64) bool {
	if !(r.W > 0) || !(r.H > 0) {
		return false
	}
	ix0, ix1 := math.Max(r.X, x0), math.Min(r.X+r.W, x1)
	iy0, iy1 := math.Max(r.Y, y0), math.Min(r.Y+r.H, y1)
	iw, ih := ix1-ix0, iy1-iy0
	if !(iw > 0) || !(ih > 0) {
		return false
	}
	return 2*iw*ih > r.W*r.H
}

// spanned normalises two opposite corners into a closed rectangle, low end
// first, so orientation-independence is ONE EXPRESSION'S PROPERTY and no
// caller has to order the points it hands over: a drag right-to-left or
// bottom-to-top yields exactly what its corner-swapped twin yields.
//
// A non-finite corner is left where it was — the comparison that would swap it
// is false — and meets then answers false for it, so nothing downstream has to
// look at it twice.
func spanned(ax, ay, bx, by float64) (x0, y0, x1, y1 float64) {
	if bx < ax {
		ax, bx = bx, ax
	}
	if by < ay {
		ay, by = by, ay
	}
	return ax, ay, bx, by
}

// picked is every unit the closed screen rectangle [x0,x1] x [y0,y1]
// catches: one walk of the snapshot IN ITS OWN ORDER, keeping the entries
// whose placed rectangle the gesture meets.
//
// IT IS THE WHOLE HIT TEST, AND BOTH BRANCHES BELOW CALL IT. That is what makes
// 0030 C-1's claim — a click and a one-cell box cannot answer differently about
// the very same pixels — a property of there being one function rather than a
// sentence asserted beside two. It was asserted and untested until 0058, and the
// two had in fact come apart: the box resolved a cell RANGE and the tap a single
// cell, both through the flat lattice, while every glyph on the map took the
// terrain's own height.
//
// at is the placement — an entity to the rectangle it is drawn on. Taking it as
// a parameter is what keeps this pure and reachable with no window and no
// camera: a synthetic placement can be handed to it directly.
//
// A TERMINAL DEAD ENTRY IS NOT A CANDIDATE, and the test is at the hit rather
// than after it: a tap on one is a miss, like empty ground. Downed entries and
// finishable dead entries carrying Selectable are taken. Their orders go
// nowhere because the simulation refuses movement for every not-alive actor;
// that remains the world's business, not the picker's. An entity with no
// placement — off the map, or outside the view — is caught by nothing.
//
// The walk is the snapshot's own order, which is ascending id, so a caller
// keeping the ids gets them ascending by construction and nothing sorts
// them.
func picked(ents []MapEntity, at func(MapEntity) (screenRect, bool), x0, y0, x1, y1 float64) []MapEntity {
	var out []MapEntity
	for _, e := range ents {
		if !selectableEntity(e) {
			continue
		}
		r, ok := at(e)
		if !ok {
			continue
		}
		if !r.meets(x0, y0, x1, y1) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// decide is the whole of what a frame's input does to the selection and to the
// world: the next selection, and the order to issue if there is one.
//
// IT IS PURE — no receiver, no input, no output, no drawing, no world. It
// reads the current selection, the snapshot the viewer holds, the two
// placements and one frame's edges. That is what makes every case below
// reachable in a test with no window.
//
// at IS THE PLACEMENT AND IT IS WHERE THE UNIT PICK LIVES: an entity to the
// rectangle it is DRAWN on, which carries its cell's own height lift, its
// own displacement, the camera and the view cull. groundAt answers the
// GROUND PICK ALONE — which cell of the map a left click names. A raw
// *camera.Camera answered this through the flat lattice before the first
// DIV-044 hotfix; the interim picker searched the entity placement surface.
// groundAt now searches ROM1's per-column corner-mesh bounds. Entity and
// ground picking therefore share the camera and map extent, but
// intentionally use the two different altitude models their subjects
// require.
//
// THE BOOL FROM groundAt IS READ FIRST, always. Outside what it resolves it
// returns 0, 0 and those zeros are not a cell — the contract declines to
// promise a value there and no clamp exists anywhere on the path — so col and
// row are untouched unless inside is true, which is what keeps a left click
// past the edge from walking a unit off the map (0028 AC-1, AC-10, C-2).
//
// THE BODY IS A BRANCH CHAIN — box, else tap, else order, else nothing —
// so a release and a press arriving in one frame yield the RELEASE ALONE and
// the four outcomes (select, clear, order, nothing) are total by
// construction, with no fifth and none undefined. Honouring both was
// rejected: an order's subject would then depend on a 16 ms coincidence.
// Keeping it ONE function is what leaves that totality observable: a second
// pure function for the release would split the four outcomes across two
// bodies and no test could see they are exhaustive.
//
// A BOX SELECTS EVERY UNIT ITS RECTANGLE MEETS and a TAP the one whose
// rectangle holds its point — the SAME call, with the tap's two corners
// equal.
//
// Where two placed rectangles hold one point the tap takes the LOWER id —
// a minimum over the whole set rather than the first match, so the tie is
// settled over ids and never quietly becomes slice position. Relief is what
// makes that reachable between DIFFERENT cells now, where before it needed
// two units on one cell: a cliff can put one cell's footprint over
// another's.
//
// A RIGHT PRESS ORDERS EVERY SELECTED UNIT THE SNAPSHOT STILL HOLDS, one
// order each, all naming the one resolved cell, in ascending id. An id the
// snapshot no longer holds is skipped and stays selected; a press while the
// left button is still down orders nothing at all, and neither does one
// resolving outside the extent or one every selected id is absent from.
//
// The orders are returned as a slice and the bool beside it says the frame
// ordered — the shape the seam's caller already has, so k orders are k calls
// into the queue that exists rather than a collection crossing the seam.
//
// The second result is the units a selection form put into the selection, for
// the selection reply; an order, a miss or a removal names none.
func decide(cur selection, ents []MapEntity, groundAt func(x, y float64) (col, row int, inside bool), at func(MapEntity) (screenRect, bool), localOwner uint32, g gesture) (selection, []uint32, []order, bool) {
	present := presentSelected(cur, ents)

	if g.boxed {
		x0, y0, x1, y1 := spanned(float64(g.px), float64(g.py), float64(g.x), float64(g.y))
		next, added := rectangleSelection(cur, present, ents, at, localOwner, x0, y0, x1, y1, g.shift)
		return next, added, nil, false
	}

	if !g.tap {
		return cur, nil, nil, false
	}

	if len(present) == 0 {
		next, added := clickSelection(cur, present, ents, at, localOwner, float64(g.x), float64(g.y), g.shift)
		return next, added, nil, false
	}

	kind := cursorOrderKind(g.cursor)
	if kind == orderKindCast {
		present = bookCasters(present, g.spell)
		if len(present) == 0 {
			return cur, nil, nil, false
		}
	}
	switch kind {
	case orderKindNone:
		// A cursor with no arm falls out of the dispatch chain and does
		// nothing at all, which is the original's own shape: `default`, the
		// eight edge arrows and the five minimap cursors have no arm.
		return cur, nil, nil, false
	case orderKindSelect:
		next, added := clickSelection(cur, present, ents, at, localOwner, float64(g.x), float64(g.y), g.shift)
		return next, added, nil, false
	case orderKindPickup:
		// PICKUP IS ONE ORDER AND NOT ONE PER SELECTED UNIT. Its own cursor
		// gate requires exactly one selected object (`AI-CURSOR-242`), so the
		// walk below would emit exactly one in any case, and `AI-CLICK-050`
		// gives the arm one opcode, `0x21`.
		//
		// THE EXTENT TEST IS THE SAME ONE EVERY CELL-NAMING ARM BELOW USES, and
		// a press outside what groundAt resolves orders nothing, exactly as
		// there.
		col, row, inside := groundAt(float64(g.x), float64(g.y))
		if !inside {
			return cur, nil, nil, false
		}
		return cur, nil, []order{{kind: orderKindPickup, entity: present[0].ID, x: col, y: row}}, true
	case orderKindTown:
		if g.structure.Kind != InspectionStructure || !townGate(present, hoverMaskStructure|hoverMaskStructureCell) {
			return cur, nil, nil, false
		}
		return cur, nil, []order{{kind: orderKindTown, entity: present[0].ID, victim: g.structure.ID}}, true
	}

	// THE ATTACK ARM IS THE ONLY ONE THAT PRODUCES TWO DIFFERENT ORDERS
	// (`AI-CLICK-050`): `0x19` at a qualifying target and `0x16`, a plain move
	// to the cell, otherwise. NO OWNERSHIP AND NO DIPLOMACY IS READ HERE, in
	// either direction, because the click-time test in what is being
	// reconstructed is a runtime-class test and no diplomacy runs at click
	// time at all.
	if kind == orderKindAttack || kind == orderKindCast && !g.pointCast || kind == orderKindDefend {
		if kind == orderKindAttack && g.structure.Kind == InspectionStructure {
			orders := make([]order, 0, len(present))
			for _, e := range present {
				orders = append(orders, order{kind: kind, entity: e.ID, victim: g.structure.ID, cell: true})
			}
			return cur, nil, orders, true
		}
		allowCorpse := kind == orderKindCast && g.spell == controlSpiritSpellID
		if victim, hit := targetAt(ents, at, float64(g.x), float64(g.y), allowCorpse); hit {
			orders := make([]order, 0, len(present))
			for _, e := range present {
				orders = append(orders, order{kind: kind, entity: e.ID, victim: victim, spell: g.spell})
			}
			return cur, nil, orders, true
		}
		// A `cast` press that named no unit is the EMPTY-GROUND cast and
		// falls through to the cell walk below, which sets `cell`. An
		// `attack` press that named no unit becomes a plain move, which is
		// `AI-CLICK-050`'s own second arm for that cursor.
		if kind == orderKindAttack {
			kind = orderKindMove
		}
		if kind == orderKindDefend {
			return cur, nil, nil, false
		}
	}

	// EVERY REMAINING ARM NAMES A CELL, so one extent test serves all of them.
	// Outside what groundAt resolves the press orders nothing at all, which is
	// what keeps a click past the edge from walking a unit off the map.
	col, row, inside := groundAt(float64(g.x), float64(g.y))
	if !inside {
		return cur, nil, nil, false
	}
	orders := make([]order, 0, len(present))
	for _, e := range present {
		o := order{kind: kind, entity: e.ID, x: col, y: row}
		if kind == orderKindCast {
			// `cast` is `0x25`/`0x1e` at a unit and `0x1f`/`0x26` at a
			// cell (`AI-CLICK-050`), and this build carries the distinction
			// on ONE seam through `MapAttack`'s own `cell` flag. The unit
			// half is the arm above; reaching here means that arm found no
			// unit, so this is the cell half.
			o.spell, o.cell = g.spell, true
		}
		orders = append(orders, o)
	}
	return cur, nil, orders, true
}

// clickSelection returns the next selection and the unit the click added or
// selected again. A Shift click that toggles its unit off adds none.
func clickSelection(cur selection, present []MapEntity, ents []MapEntity, at func(MapEntity) (screenRect, bool), localOwner uint32, x, y float64, shift bool) (selection, []uint32) {
	best, hit := topAt(ents, at, x, y)
	if !shift {
		if !hit {
			return nil, nil
		}
		return selection{best}, []uint32{best}
	}
	if !hit {
		return cur, nil
	}
	if !ownedCandidate(ents, best, localOwner) {
		return cur, nil
	}
	if oldSummaryForeign(present, localOwner) {
		return cur, nil
	}
	if slices.Contains(cur, best) {
		return toggled(cur, best), nil
	}
	return toggled(cur, best), []uint32{best}
}

// rectangleSelection returns the next selection and the qualified units it
// holds: every one for a plain rectangle, those toggled on for Shift.
func rectangleSelection(cur selection, present []MapEntity, ents []MapEntity, at func(MapEntity) (screenRect, bool), localOwner uint32, x0, y0, x1, y1 float64, shift bool) (selection, []uint32) {
	var qualified []uint32
	for _, e := range picked(ents, at, x0, y0, x1, y1) {
		if !ownedCandidate(ents, e.ID, localOwner) {
			continue
		}
		r, ok := at(e)
		if !ok || !r.coveredMoreThanHalf(x0, y0, x1, y1) {
			continue
		}
		qualified = append(qualified, e.ID)
	}
	if len(qualified) == 0 {
		return cur, nil
	}
	if !shift {
		return selection(qualified), qualified
	}
	if oldSummaryForeign(present, localOwner) {
		return cur, nil
	}
	next := cur
	var added []uint32
	for _, id := range qualified {
		if !slices.Contains(next, id) {
			added = append(added, id)
		}
		next = toggled(next, id)
	}
	return next, added
}

// A minimap press or drag centres the camera and never issues a unit order.
func (v *Viewer) minimapAction(x, y int) []order {
	if _, ok := v.minimapActionCursor(x, y); ok {
		v.CenterOnMinimapPixel(x, y)
	}
	return nil
}

// topEntityAtCell is the last entity of the snapshot standing on this cell, or
// false when none does. LAST rather than first, which is entityDraws' own
// paint order read as a hit order: the snapshot is drawn in list order, so the
// last one on a cell is the one on top of it.
//
// IT IS FOG-GATED, on hoverMask's own terms: an enemy standing in the dark is
// not a target the minimap can name, because the widget does not draw him
// either (minimapMarks is gated). Without the gate, `AI-MINIMAP-124`'s own
// "attack emits 0x19 on any non-zero cell object id and otherwise 0x1a" would
// let a player pick an unseen enemy off a blank corner of the overview and
// learn he is there from which order came out.
func (v *Viewer) topEntityAtCell(cell image.Point) (uint32, bool) {
	var id uint32
	var found bool
	for _, e := range v.entities {
		if e.Cell != cell || e.Untargetable || !v.fogGateEntity(e.Owner, cell.X, cell.Y) {
			continue
		}
		id, found = e.ID, true
	}
	return id, found
}

// ownedCandidate reports whether the entity with this id is owned by the local
// participant. A local participant of zero is none established and every
// candidate is owned, which is canArmAttack's own reading of the same field.
func ownedCandidate(ents []MapEntity, id uint32, localOwner uint32) bool {
	if localOwner == 0 {
		return true
	}
	for _, e := range ents {
		if e.ID == id {
			return e.Owner == localOwner
		}
	}
	return false
}

// oldSummaryForeign is `view+0x144` bit `0x4` over the selection a gesture
// BEGAN with (`AI-PANEL-061`, `AI-SELECT-122`): the primary selected object's
// own player is not the local participant's.
func oldSummaryForeign(present []MapEntity, localOwner uint32) bool {
	if len(present) == 0 || localOwner == 0 {
		return false
	}
	return present[0].Owner != localOwner
}

// toggled adds an id to the selection or removes it, keeping the set ASCENDING
// -- the invariant every reader of `selection` depends on and nothing sorts.
func toggled(cur selection, id uint32) selection {
	next := make(selection, 0, len(cur)+1)
	placed := false
	for _, have := range cur {
		if have == id {
			// Present: this is a removal, so it is not copied and nothing is
			// inserted in its place.
			placed = true
			continue
		}
		if have > id && !placed {
			next = append(next, id)
			placed = true
		}
		next = append(next, have)
	}
	if !placed {
		next = append(next, id)
	}
	if len(next) == 0 {
		return nil
	}
	return next
}

// dropOffMapCell is the ground drop's own off-map sentinel (1005 round 2,
// `ITEM-DROP-008`): a release point groundCellAt cannot place on a drawn
// tile still names a cell, one far outside any real map on both axes, so
// `pkg/sim`'s own Chebyshev window (drop.go) falls back to the dropper's
// own position exactly as it does for any other out-of-window request —
// ITEM-DROP-008's own "never refused", carried through a release this
// package cannot place on the ground at all.
const dropOffMapCell int32 = 1 << 20

// dropCellAt turns a release point into the (x, y) pair raiseGroundDrop
// carries into a sim.Command: groundCellAt's own cell when the point falls
// on a drawn tile, dropOffMapCell on both axes otherwise.
func (v *Viewer) dropCellAt(x, y int) (int32, int32) {
	col, row, inside := v.groundCellAt(float64(x), float64(y))
	if !inside {
		return dropOffMapCell, dropOffMapCell
	}
	return int32(col), int32(row)
}

// command is the impure shell around decide: it turns this frame's raw
// button edges into a gesture, stores the selection decide returns, and
// hands the order back to its caller.
//
// IT IS UNEXPORTED, and that is the mechanism rather than a convention:
// cmd/mapview is another package and cannot call it, so the standalone
// viewer gaining no selection, no order and no highlight is Go's export
// rule.
//
// The press edge sets held and zeroes the accumulator. Both are needed beside
// dragIntent's own anchor branch and neither alone suffices: the anchor never
// runs for a press and a release inside ONE frame, and the press edge never
// fires for a button already held when the map opened. held is what makes a
// release with no press nothing at all rather than a tap at wherever the cursor
// happens to be (0028 AC-2).
//
// The slop is read at the release, so this must run after the frame's camera
// step: that tick's delta has to be in the accumulator before the release is
// judged.
//
// ONE RELEASE IS JUDGED ONCE. Under the slop it is a tap; over it, it is a
// box exactly when the viewer latched this press as one, and otherwise it is
// the pan that has already happened and nothing more. The latch and the
// press point are both read from the viewer here rather than recomputed:
// dragIntent wrote them where the gesture began, which is the only tick on
// which the modifier was consulted.
func (v *Viewer) command(in appInput) ([]order, bool) {
	defer v.syncMapViewport()
	// THE SECOND OF THE THREE DOORS (1026 B4). in.CursorX/CursorY arrive in
	// WINDOW pixels and every hit test below this line reads them as FRAME
	// pixels: the doll, the worn box, the pack bar and its arrows, the
	// spellbook, the minimap, the unit panel, the control panel, the ground
	// surface and the drop cell. None of them changed for this story, and that
	// is the point of mapping here rather than in each of them.
	in.CursorX, in.CursorY = v.windowToFrame(in.CursorX, in.CursorY)

	// THE INVENTORY'S OWN BOXES SWALLOW THEIR OWN BUTTONS, and it is the FIRST
	// statement here so that nothing below can have already run for a press
	// one of them took (hotfix: the window is drawn over the world and the
	// click fell through it — a press on the figure selected whatever stood
	// under the window and a left click walked the party there). Since 0140
	// there are two of them, the doll box and the pack bar, and
	// inventoryCaptures answers for both as one surface.
	//
	// TWO TESTS, ONE SWALLOW. The latch beside it is for the DRAG: a gesture
	// that began on the window stays the window's until the button comes up, so
	// pulling the cursor off the frame mid-press cannot resume selecting or
	// panning halfway through.
	//
	// v.held IS STILL CLEARED ON A SWALLOWED RELEASE, and it must be: leaving
	// it raised would make the NEXT release anywhere on the map read as the tap
	// of a press that was never delivered.
	// THE COMMAND PANEL TAKES ITS PRESS FIRST OF ALL (0140, owner: "under the
	// map add a UI control panel"; docs/1028-command-panel, which put the
	// original's own decoded panel in this same slot). It is asked before the
	// inventory's own boxes for the same reason it is PAINTED after them
	// (viewer.go's Draw): it is the one surface down here that can bring any
	// of the others back, so wherever two of them ever came to overlap, this
	// one must keep the pixel.
	//
	// LEFT DOWN ACTS, A DOUBLE-CLICK ALIASES IT AND A LEFT-DRAG RE-ENTERS IT
	// ON EVERY DELIVERED MOVE (contract B2, `MENU-COMBAT-019`'s own mouse
	// edges). Every press edge already fires independently — a double-click's
	// second press is just another PrimaryPressed — so no separate double-
	// click bookkeeping is needed the way the inventory's own boxes need one.
	// cmdDragCell is the drag's own "last cell resolved", reset whenever the
	// button is not held, so a held drag re-fires exactly once per NEW cell
	// entered rather than once per frame it sits still.
	//
	// RIGHT UP CANCELS; THE OTHER RIGHT EDGES ARE NO-OPS (contract B2).
	// SecondaryReleased entered appInput with this panel (readAppInput,
	// app.go). The general map gesture in `decide` still reads no secondary
	// edge at all; command handles cancellation after decide, so no
	// right-button ordering path is added.
	//
	// THE MISSION CHARACTER PANE OWNS EVERY SECONDARY EDGE (`TOWN-344`,
	// High). Its right-down handler does nothing, while right-up posts the map
	// view's ordinary 0x405 cancel without consulting a coordinate inside the
	// pane. This stands before every other HUD surface because ownership from a
	// pane down edge survives the cursor leaving: no later surface may consume
	// that gesture's SECONDARY release first. Primary input remains independent:
	// holding right must not disable a corner's left click or strand an inventory
	// drag whose left button comes up on the same frame.
	paneHere := v.panelCaptures(in.CursorX, in.CursorY)
	if !in.Viewer.SecondaryDown && !in.SecondaryReleased {
		// A lost release edge must not strand ownership forever. The ordinary
		// release frame is retained by SecondaryReleased and clears below.
		v.paneSecondaryGrab = false
	}
	if in.SecondaryPressed && paneHere {
		v.paneSecondaryGrab = true
	}
	paneOwnsSecondary := v.paneSecondaryGrab || (paneHere &&
		(in.Viewer.SecondaryDown || in.SecondaryPressed || in.SecondaryReleased))
	if paneOwnsSecondary {
		if in.SecondaryReleased {
			// A release delivered over the pane posts 0x405 even when the
			// gesture first moved on the map. Unlike the map widget's own
			// right-up path, this handler has no marked-drag gate.
			if paneHere {
				v.cancelMapCommand()
			}
			v.paneSecondaryGrab = false
		}
		// Ownership suppresses only the secondary route. appInput is a value,
		// so clearing its three secondary facts here leaves the physical input
		// snapshot intact for step while preventing the command panel,
		// spellbook, minimap and map handlers below from seeing a pane-owned
		// edge or level. The primary facts deliberately continue through those
		// same existing routes, including every release cleanup.
		in.Viewer.SecondaryDown = false
		in.SecondaryPressed = false
		in.SecondaryReleased = false
	}
	// A marquee belongs to the map where its press was accepted. HUD hover
	// must neither consume its release nor press a command while crossing it.
	// Include the final pointer segment: a fast drag may have no held frame
	// between its press and release.
	marqueeMoved := v.dragMoved
	if in.PrimaryReleased {
		marqueeMoved += absInt(in.CursorX-v.dragX) + absInt(in.CursorY-v.dragY)
	}
	if v.held && v.boxing && marqueeMoved > v.marqueeSlop() &&
		(in.Viewer.PrimaryDown || in.PrimaryReleased) {
		if in.PrimaryReleased {
			var added []uint32
			v.sel, added, _, _ = decide(v.sel, v.entities, v.groundCellAt, v.entityPickRect, v.localOwner, gesture{
				boxed: true, shift: v.shiftLatch,
				x: in.CursorX, y: in.CursorY, px: v.pressX, py: v.pressY,
			})
			if !v.shiftLatch {
				v.noteSelected(added)
			}
			v.held = false
		}
		if in.SecondaryReleased && !v.rightPanned {
			v.cancelMapCommand()
		}
		return nil, false
	}
	// The mission tip popup takes the presses on its own controls and the
	// release over its list (MENU-137); the rest of its body passes through.
	if v.missionTipGesture(image.Pt(in.CursorX, in.CursorY), in.PrimaryPressed, in.PrimaryReleased) {
		if in.PrimaryReleased {
			v.held = false
			v.dragActive = false
			v.dragCandKind = dragNone
			v.dragIcon = nil
			v.invEquipTap = false
			v.invGrab = false
		}
		return nil, false
	}
	if v.commandPanelCaptures(in.CursorX, in.CursorY) {
		cell, cellOK := v.commandCellAt(in.CursorX, in.CursorY)
		if in.PrimaryPressed {
			v.cmdDragCell = -1
			v.pressCommandPanelCell(cell, cellOK)
			if cellOK {
				v.cmdDragCell = int(cell)
			}
		} else if in.Viewer.PrimaryDown && cellOK && int(cell) != v.cmdDragCell {
			v.pressCommandPanelCell(cell, cellOK)
			v.cmdDragCell = int(cell)
		}
		if !in.Viewer.PrimaryDown {
			v.cmdDragCell = -1
		}
		if in.SecondaryReleased {
			// A MARGIN OR DISABLED LEFT-DOWN'S OWN EFFECT (pressCommandPanelCell's
			// doc), reached here instead because this is a RIGHT release: it
			// clears only the drawn overlay and leaves any armed mode intact.
			v.cmdOverlayHidden = true
		}
		if in.PrimaryReleased {
			v.held = false
			// AN ARMED INVENTORY DRAG IS CANCELLED HERE TOO, not only v.held
			// (counterexample 5, round-2 adversarial review, carried over unchanged
			// by this story): this branch RETURNS before the inventory's own
			// swallowed block below, which is the ONE place dragActive, dragCandKind,
			// dragIcon and invGrab are otherwise reset on release. Left standing, a
			// release inside this slot left the state exactly as armed as it was
			// mid-press — invGrab true, dragActive true — so swallowed :=
			// v.invGrab || ... stayed true on every later frame regardless of where
			// the cursor went, and the NEXT primary release anywhere off the
			// inventory reached the drag-release switch with a stale origin and
			// resolved it as a ground drop at THAT unrelated release's own cell — a
			// cell the player never aimed this drag at. Cancelling here,
			// unconditionally, matches "released anywhere else... raises neither
			// request... the item returns to its origin": this slot is exactly such
			// an "anywhere else", a HUD surface the inventory's own boxes do not
			// claim.
			v.dragActive = false
			v.dragCandKind = dragNone
			v.dragIcon = nil
			v.invEquipTap = false
			v.invGrab = false
		}
		return nil, false
	}

	// THE CHARACTER PANE'S SIX CORNERS, BEFORE THE INVENTORY'S OWN BOXES. Two
	// of them are the controls that bring the pack bar and the spellbook back,
	// and all six sit on a surface the map must never see a press through, so
	// this arm swallows the gesture whether or not a corner acts on it.
	//
	// A CORNER AND THE FIGURE ARE SEPARATED BY GESTURE, NOT BY GEOMETRY
	// (`DIV-308`, characterpane.go's own note). Rect B and rect C stand inside
	// the painted figure crop; in the original a single left click there is
	// the corner and the slot map answers other messages entirely. So a press
	// and release runs the corner, and a press over a pixel the slot map also
	// marks FALLS THROUGH to the inventory below, where the same press arms
	// the figure's own drag. Once that drag is in flight dragActive skips this
	// arm outright, which is the held-item rule below.
	//
	// TOWN-348, DIV-315
	if in.PrimaryReleased && v.dragActive {
		// A drag that armed from an overlapping pixel ends in the inventory's
		// own drop resolution below, and this latch must not survive it.
		v.paneCornerGrab = false
	}
	if !v.dragActive {
		corners := v.characterPaneCornerAt(in.CursorX, in.CursorY)
		if len(corners) > 0 {
			if in.PrimaryPressed {
				v.paneCornerGrab = true
			}
			if in.PrimaryReleased && v.paneCornerGrab {
				v.pressCharacterPaneCorner(in.CursorX, in.CursorY)
				v.paneCornerGrab = false
				v.held = false
				v.invGrab = false
				v.dragActive = false
				v.dragCandKind = dragNone
				v.dragIcon = nil
				v.invEquipTap = false
				return nil, false
			}
			if in.PrimaryReleased {
				v.paneCornerGrab = false
			}
			if _, onSlot := v.dollFigureSlotAt(in.CursorX, in.CursorY); !onSlot {
				return nil, false
			}
			// The pixel is both a corner and a marked equipment slot. Only a
			// drag off it can be the slot's own gesture, so the press goes on
			// to the inventory; the release above has already been taken.
		} else if in.PrimaryReleased {
			v.paneCornerGrab = false
		}
	}

	if in.PrimaryPressed && v.inventoryCaptures(in.CursorX, in.CursorY) {
		v.invGrab = true
	}
	swallowed := v.invGrab || v.inventoryCaptures(in.CursorX, in.CursorY)
	if in.PrimaryReleased {
		v.invGrab = false
	}
	if swallowed {
		// THE DOUBLE-CLICK COUNT AND THE PRESS TEST LIVE HERE, and that is
		// deliberate rather than a matter of convenience: a double-click is only
		// ever a double-click on a window that is already taking the press, so
		// nothing below can be reached by a frame the map is handling instead —
		// a press outside the window never runs a single line of it.
		//
		// THE COUNT IS DECREMENTED FIRST, unconditionally, once per call —
		// which is once per map-screen frame, since command is called
		// exactly once per Update on the map arm (app.go) — before this
		// frame's own press is even looked at. That ordering is what makes
		// invClickFrames a frame count and nothing subtler: the frame a
		// first click lands on spends none of its own window, and every
		// frame after it, press or not, spends exactly one.
		if v.invClickFrames > 0 {
			v.invClickFrames--
		}
		// THE WORN BOX RUNS THE SAME COUNTDOWN, ON ITS OWN FIELD (0151,
		// defect 4): a separate window over a separate box, decremented
		// unconditionally beside the pack's own count so a press on one box
		// can never extend or spend the other's.
		if v.invWornClickFrames > 0 {
			v.invWornClickFrames--
		}
		if in.PrimaryPressed {
			// THE SCROLL BUTTONS COME FIRST AND THEY ARE NOT CELLS (0140,
			// owner: "the inventory scrolls with arrows on the right,
			// potentially endlessly"). A press on one moves the bar by a
			// cell and touches NEITHER double-click field, which is what
			// keeps scrolling past an item and then clicking it twice from
			// reading as a double-click on whatever the first press
			// happened to be over.
			// A FRESH PRESS DISCARDS WHATEVER CANDIDATE THE LAST GESTURE LEFT
			// (1005, "the interactive doll"): dragCandKind is reset to dragNone
			// before either branch below has a chance to set it, so a press
			// landing on a scroll button, a worn cell or the doll's own
			// background — none of which is a drag source in round 1 — leaves no
			// stale origin for the arm check further down to read.
			v.dragCandKind = dragNone
			v.invEquipTap = false
			if delta, ok := v.packArrowAt(in.CursorX, in.CursorY); ok {
				v.ScrollPack(delta)
			} else if cell, ok := v.inventoryPackCellAt(in.CursorX, in.CursorY); ok {
				if v.invClickFrames > 0 && cell == v.invClickCell {
					// THE SAME CELL, INSIDE THE WINDOW STILL OPEN ON IT: the
					// count is spent so a third press in a row starts a fresh
					// count rather than raising a second request under it, and
					// the press is REMEMBERED rather than acted on. The request
					// itself is raised by this press's own release, in the
					// release arm below, once the gesture is known not to have
					// become a drag. Raising it here acted before that was
					// known: App advances the world before command() every
					// frame, so the drain removed and reindexed the pack while
					// the still-held press went on to cross TapSlop, and the
					// drag then carried an index naming the neighbour that had
					// shifted into it.
					v.invEquipTap = true
					v.invClickFrames = 0
				} else {
					// EVERY OTHER PRESS ON A PACK CELL RESTARTS THE COUNT ON
					// THE CELL IT LANDED ON (plan D-4) — a first click on a
					// cell nothing was counting down on, a click on a
					// DIFFERENT cell than the one being counted, and a click
					// that arrives after the count has already run out, are
					// the same case: this is a fresh first click and nothing
					// about it is a double-click yet.
					v.invClickCell = cell
					v.invClickFrames = InventoryDoubleClickFrames
				}
				// THE DRAG MACHINE'S FIRST SOURCE (1005 spec, "sources in round
				// 1: a pack-bar cell, and a slot on the doll"): every press on a
				// pack cell is a candidate drag origin, ALONGSIDE the double-
				// click bookkeeping above and not instead of it — a press that
				// never crosses TapSlop stays that bookkeeping's own tap, and
				// one that does is armed below into a drag, which a released
				// double-click has already finished being.
				v.dragCandKind, v.dragCandIdx = dragFromPack, cell
			} else if slot, ok := v.inventoryWornSlotAt(in.CursorX, in.CursorY); ok {
				// THE WORN BOX'S OWN MATCH-OR-RESTART, over its own two
				// fields (0151, defect 4): the pack cell branch's shape,
				// restated for the box the gesture now also runs on.
				if v.invWornClickFrames > 0 && slot == v.invWornClickCell {
					v.invUnequipRequest = slot + 1
					v.invWornClickFrames = 0
				} else {
					v.invWornClickCell = slot
					v.invWornClickFrames = InventoryDoubleClickFrames
				}
				// THE WORN BOX IS NOT A DRAG SOURCE IN ROUND 1 (contract "out of
				// scope": a worn-box drag would need the compositor's own icons
				// to move, which is a different mechanism from the mask this
				// story builds) — dragCandKind stays dragNone from the reset
				// above.
			} else if slot, ok := v.dollFigureSlotAt(in.CursorX, in.CursorY); ok {
				// THE DRAG MACHINE'S SECOND SOURCE, and round 1's "press to take
				// off" both start here: a press on the doll figure's own slot
				// pixels — dollFigureSlotAt's own mask lookup, the same one the
				// hover popup reads (itempopup.go) — is a candidate this gesture
				// may still resolve as a tap (the release handling below) or arm
				// into a drag (the check right after this block).
				v.dragCandKind, v.dragCandIdx = dragFromDoll, slot
			}
			// A press that names none of the three — a scroll button, an empty
			// pack cell past the end of the pack, or the doll's own transparent
			// pixels and background — touches none of the click fields and
			// leaves dragCandKind at dragNone: it is not a click on a cell of
			// any box and not a drag source, so a live count on some other cell
			// keeps counting down underneath it exactly as if this press had
			// landed on the map instead.
		}

		// THE DRAG ARM, and it runs on EVERY SWALLOWED FRAME rather than only on
		// the press (1005 spec, "a press that moves past TapSlop picks the item
		// up"): v.dragMoved is the SAME accumulator the map's own box-select
		// judges a gesture by, raised in dragIntent (viewer.go) before command
		// ever runs this frame, so "past TapSlop" means the identical distance for
		// an inventory drag that it means for a map box, and there is no second
		// measurement to disagree with it. Arming reads the candidate's own icon
		// — Pack[idx] for a pack origin, Slots[idx] for a doll origin, the SAME
		// icon the worn box already draws for that slot — so dragItemPresent has
		// a picture to carry with no archive read of its own.
		if v.dragCandKind != dragNone && !v.dragActive && v.dragMoved >= TapSlop {
			v.dragActive = true
			switch v.dragCandKind {
			case dragFromPack:
				if v.dragCandIdx >= 0 && v.dragCandIdx < len(v.invSubject.Pack) {
					v.dragIcon = v.invSubject.Pack[v.dragCandIdx]
				}
				// A drag of the purse reserves its share in the preview.
				v.armGoldDrag(v.dragCandIdx, in.ShiftHeld || in.Viewer.Shift)
			case dragFromDoll:
				if v.dragCandIdx >= 0 && v.dragCandIdx < len(v.invSubject.Slots) {
					v.dragIcon = v.invSubject.Slots[v.dragCandIdx]
				}
			}
		}

		if in.PrimaryReleased {
			// A RELEASE BACK ON THE DRAG CANDIDATE'S OWN ORIGIN CELL IS NEVER A DRAG
			// (round-2 adversarial review, fifth pass, counterexamples E and F —
			// the shop's own dest == origin tremor guard, app.go:1677, restated for
			// the mission map). TapSlop is a 4-pixel Manhattan threshold accumulated
			// over the whole press, held against a 48-pixel pack cell (invCellSize)
			// or a doll slot of any size: a hand's own tremor crosses it while the
			// gesture never leaves the cell or slot it began on. Judging such a
			// release by v.dragActive alone — which the tremor already set true —
			// ran it through the doll-box/ground-drop resolution below, and neither
			// of that resolution's two arms matches a release still inside its own
			// origin, losing the gesture outright: a double-click's matching second
			// press lost the equip (counterexample E), and a plain tap-to-unequip
			// lost the unequip (counterexample F, pre-existing on master, undisclosed
			// until this pass). originSame asks the question once, ahead of the
			// switch below, for both origins, so the ordinary tap resolution runs
			// whenever the release names the same cell or slot the press did, whether
			// or not TapSlop was crossed getting there — exactly the cases
			// v.dragCandKind's own two non-drag arms below already handled for a
			// press that never crossed TapSlop at all.
			originSame := false
			switch v.dragCandKind {
			case dragFromPack:
				if cell, ok := v.inventoryPackCellAt(in.CursorX, in.CursorY); ok && cell == v.dragCandIdx {
					originSame = true
				}
			case dragFromDoll:
				// dollFigureSlotAtUnsuppressed, NOT dollFigureSlotAt (round-2
				// adversarial review, tenth pass, counterexample 1): while this drag has
				// armed, refreshDollDrag (pkg/game) pushes a suppressed mask with the
				// origin slot's own code cleared, and dollFigureSlotAt reads that mask
				// whenever the suppression is in force — so it can never answer the
				// origin's own index at a release, and this arm of originSame was
				// unreachable for any release reached with an armed drag. The origin's
				// identity does not change because the drawn picture did; the
				// unsuppressed variant reads v.invSubject.SlotMask directly, in force
				// before the drag suppressed anything, which is the question this
				// comparison asks.
				if slot, ok := v.dollFigureSlotAtUnsuppressed(in.CursorX, in.CursorY); ok && slot == v.dragCandIdx {
					originSame = true
				}
			}
			switch {
			case originSame && v.dragCandKind == dragFromPack && v.invEquipTap:
				v.packCellTap(v.dragCandIdx)
			case originSame && v.dragCandKind == dragFromDoll:
				v.invDollUnequipRequest = v.dragCandIdx + 1
			case v.dragActive:
				// THE DRAG MACHINE'S RELEASE (1005 spec): a pack origin released
				// over the doll box raises the SAME one-shot request a double-
				// click on that pack cell already would (invEquipRequest), so
				// equipFromPack (pkg/game) processes it through the identical
				// path, wear rule included, with no second copy of enqueueEquip's
				// own rule. A doll origin released over the pack bar raises
				// invDollUnequipRequest, drained by the one new function this
				// story adds (pkg/game's unequipFromDoll) beside equipFromPack
				// and unequipFromWorn. A RELEASE ANYWHERE ELSE — back onto the
				// same box, or off both boxes entirely, including over the
				// running map — RAISES NEITHER: nothing was ever removed from
				// the pack array or the world's own equipment to begin with,
				// since a drag is view state alone (spec: "the world is never
				// mid-drag"), so returning the item to where it came from costs
				// no statement here at all — the next frame's ordinary
				// presentation already shows it there.
				p := image.Pt(in.CursorX, in.CursorY)
				switch v.dragCandKind {
				case dragFromPack:
					if v.isPurseCell(v.dragCandIdx) {
						// THE PURSE IS NOT AN ITEM: a release on the ground
						// spends the held share as a gold request, and a
						// release anywhere else returns it to the preview.
						onGround := v.mapSurfaceCaptures(in.CursorX, in.CursorY) &&
							!v.groundSurfaceCaptures(in.CursorX, in.CursorY)
						var gx, gy int32
						if onGround {
							gx, gy = v.dropCellAt(in.CursorX, in.CursorY)
						}
						v.releaseGoldDrag(onGround, gx, gy)
						break
					}
					box, onPane := v.heldItemBox()
					switch {
					case onPane && p.In(box):
						v.invEquipRequest = v.dragCandIdx + 1
					case v.mapSurfaceCaptures(in.CursorX, in.CursorY) &&
						!v.groundSurfaceCaptures(in.CursorX, in.CursorY):
						// THE GROUND DROP (1005 round 2, `ITEM-DROP-008`,
						// `DIV-088`): a release outside every inventory box AND
						// every other HUD surface — groundSurfaceCaptures' own
						// single question — during a mission (this whole
						// function is the mission map's own gesture pipeline;
						// no other screen runs it). worn is false: the origin
						// is a pack ELEMENT, dragCandIdx's own CarriedStacks
						// ordering.
						//
						// `!v.inventoryCaptures(...)` ALONE, THE CHECK THIS REPLACED (round-2
						// adversarial review, tenth pass, unproven observation measured true):
						// a release over the spellbook strip, the minimap or the unit panel
						// during an armed drag reached this arm with inventoryCaptures
						// answering false for it — none of those three is one of
						// inventoryCaptures' own three boxes — and planted a sack at
						// whatever world cell the drawn map showed underneath that HUD chrome.
						dx, dy := v.dropCellAt(in.CursorX, in.CursorY)
						v.raiseGroundDrop(false, v.dragCandIdx, dx, dy)
					}
					// A release back inside the doll box itself but not on the
					// figure — the box's own frame or background — or anywhere
					// else groundSurfaceCaptures still claims, raises neither
					// request: the item returns to its origin exactly as before
					// this story's ground drop existed.
				case dragFromDoll:
					// packBarArea, NOT packBar (round-2 adversarial review, fifth pass,
					// counterexample D): under a subject-led multi-selection packBar
					// answers false (inventoryEligible's own narrow gate), but
					// inventoryCaptures — asked one line below, and now armed exactly
					// because this is a live dragFromDoll release — still reserves that
					// same screen rectangle as the inventory's own ground. Reading the
					// narrow packBar here left the gesture recognised (inventoryCaptures
					// true, so the ground-drop arm below never fired) but unresolved (inBar
					// false, so neither request was raised either) — the drag simply
					// vanished, the item snapping back to its origin with no unequip and no
					// drop. packBarArea is the same rectangle packBar draws with the
					// eligibility gate dropped, so an ordinary single-selection release is
					// unaffected: packBar and packBarArea agree whenever packBar itself
					// would answer true.
					bar, inBar := v.packBarArea()
					switch {
					case inBar && p.In(bar):
						v.invDollUnequipRequest = v.dragCandIdx + 1
					case v.mapSurfaceCaptures(in.CursorX, in.CursorY) &&
						!v.groundSurfaceCaptures(in.CursorX, in.CursorY):
						// worn is true: the origin is equipment slot
						// dragCandIdx (zero-based, dollFigureSlotAt's own
						// numbering). groundSurfaceCaptures, not
						// inventoryCaptures alone — same fix, same reason, as
						// the pack-origin arm above.
						dx, dy := v.dropCellAt(in.CursorX, in.CursorY)
						v.raiseGroundDrop(true, v.dragCandIdx, dx, dy)
					}
				}
			case v.dragCandKind == dragFromPack && v.invEquipTap:
				// A MATCHING SECOND PRESS BECOMES AN EQUIP ONLY HERE, on
				// release, once it is known not to be a drag (1005 round-2
				// fourth adversarial review). App's map arm advances the world
				// before command every frame. Raising invEquipRequest on the
				// press let that advance remove and reindex the pack before a
				// held second press crossed TapSlop; the active drag then kept
				// the old integer and acted again on its new occupant. The
				// release is still required to land on the same pack element,
				// on the doll tap's own match-at-release rule.
				if cell, ok := v.inventoryPackCellAt(in.CursorX, in.CursorY); ok && cell == v.dragCandIdx {
					v.packCellTap(cell)
				}
				// THE PLAIN TAP FOR A DOLL ORIGIN NEEDS NO CASE HERE (removed, round-2
				// adversarial review, tenth pass, counterexample 5): "originSame &&
				// v.dragCandKind == dragFromDoll" above already resolves every release
				// that names the drag candidate's own doll slot, whether or not TapSlop
				// was crossed getting there. A case testing the same condition again
				// here — v.dragCandKind == dragFromDoll with dollFigureSlotAt(...) ==
				// v.dragCandIdx — is reached only once originSame has already answered
				// false for that exact comparison and v.dragActive is false, and a drag
				// that never armed never pushes a suppression, so dollFigureSlotAt and
				// dollFigureSlotAtUnsuppressed read the identical mask in that state:
				// the case's own condition could never differ from originSame's, and it
				// is unreachable. It carried "press to take off" — this story's
				// primary gesture — on code that never ran.
			}
			v.dragActive = false
			v.dragCandKind = dragNone
			v.dragIcon = nil
			v.invEquipTap = false
			v.gold.held = 0
			v.held = false
		}
		return nil, false
	}

	// THE SPELLBOOK SWALLOWS ITS OWN CLICKS, beside the inventory window's own
	// guard immediately above and on the same reasoning (hotfix precedent,
	// inventoryCaptures' own doc): a press landing on the strip selects or
	// clears whichever row it named rather than falling through to the map
	// underneath it, and hovering it with no press swallows the frame exactly
	// as a hover over the open inventory window already does — this is not a
	// full-screen modal either.
	//
	// NO LATCH IS NEEDED, unlike the inventory window's invGrab: a row click
	// is a discrete toggle and never a drag, so nothing here can be dragged
	// off the strip mid-gesture the way a pack cell's icon can be — v.held is
	// still cleared on a swallowed release, for invGrab's own reason: leaving
	// it raised would make the NEXT release anywhere on the map read as the
	// tap of a press that was never delivered.
	if v.spellbookCaptures(in.CursorX, in.CursorY) {
		if in.SecondaryPressed {
			if idx, ok := v.spellbookEntryAt(in.CursorX, in.CursorY); ok {
				v.toggleAutocastAt(idx)
			}
		}
		if in.PrimaryPressed {
			if idx, ok := v.spellbookEntryAt(in.CursorX, in.CursorY); ok {
				if v.spellbook[idx].Unavailable {
					return nil, false
				}
				// A SECOND CLICK ON THE SAME ONE CLEARS IT; any other press on the strip
				// — a different row, or the first press since a unit change already
				// cleared it — selects the row it named.
				v.itemCast = nil
				if id := v.spellbook[idx].ID; v.selectedSpell == id {
					v.selectedSpell = 0
					v.spellArmed = false
				} else {
					v.selectedSpell = id
					v.armSpell()
					v.spellNeedsBook = true
					// The panel's own Cast cell reflects this selection
					// (commandpanel.go's pressCommandPanelCell); armAttack's
					// own reason for clearing the hidden overlay here too.
					v.cmdOverlayHidden = false
				}
			}
		}
		if in.PrimaryReleased {
			v.held = false
		}
		return nil, false
	}

	// THE MINIMAP TAKES THE PRESS AND CENTRES THE VIEW (0140, owner: "clicking
	// any point of the map moves it — that is the centring"). It is the third
	// box in this function to swallow a press, after the inventory window and
	// the spellbook strip, and it is placed LAST of the three on purpose: those
	// two are drawn over the world and may sit over this corner, so whichever
	// is on top keeps its own click.
	//
	// IT REVERSES 0118 AC-12, which held this box inert and had a test reading
	// THIS FILE'S SOURCE to prove no minimap symbol appeared in it. That test
	// is gone and its negation stands in its place; the reversal is the owner's
	// and is recorded in minimap.go's own head comment.
	//
	// THE LATCH IS THE DRAG. minimapGrab is raised by a press that lands on the
	// box and lowered by the release, and while it is up every frame re-centres
	// on the cell under the cursor — so holding the button drags the view, which
	// is what "move it" asks for. It is also what keeps a gesture that began
	// here from becoming a box-select the moment the cursor leaves the box.
	//
	// A CURSOR THAT LEAVES THE PICTURE MID-DRAG CENTRES NOTHING rather than
	// centring on the nearest edge: CenterOnMinimapPixel answers false for a
	// pixel outside the terrain and the view simply stops following, which is
	// the one behaviour here that cannot put the camera somewhere the player
	// did not point at.
	//
	// v.held IS CLEARED ON A SWALLOWED RELEASE, the swallow above's own reason:
	// leaving it raised would make the next release anywhere on the map read as
	// the tap of a press that was never delivered.
	//
	// THE ACTION RUNS ON THE DOWN EDGE AND REPEATS PER DELIVERED MOVE, which is
	// the claim's own shape and not this build's usual release-judged gesture:
	// the map decides at the release because a release is where a marquee ends,
	// and the minimap has no marquee to end.
	if in.PrimaryPressed && v.minimapCaptures(in.CursorX, in.CursorY) {
		v.minimapGrab = true
		v.minimapActX, v.minimapActY = in.CursorX, in.CursorY
		ords := v.minimapAction(in.CursorX, in.CursorY)
		return ords, len(ords) > 0
	}
	if v.minimapGrab || v.minimapCaptures(in.CursorX, in.CursorY) {
		var ords []order
		// Only held moves repeat. A new position on left-up is not a drag
		// and must not replace the last protected actor (AI-MINIMAP-124).
		if v.minimapGrab && in.Viewer.PrimaryDown && (in.CursorX != v.minimapActX || in.CursorY != v.minimapActY) {
			v.minimapActX, v.minimapActY = in.CursorX, in.CursorY
			ords = v.minimapAction(in.CursorX, in.CursorY)
		}
		// THE RIGHT BUTTON CENTRES THE CAMERA HERE and does not pan by delta
		// (`AI-MINIMAP-124`); `step`'s own right-drag branch drops its delta
		// over this widget for the same reason. It runs BELOW the left arm
		// above, which is the claim's own both-buttons rule: with
		// `MK_LBUTTON|MK_RBUTTON` the handler tests the left bit first,
		// executes the left action and returns, so no right-camera pan occurs.
		if in.Viewer.SecondaryDown && len(ords) == 0 && !v.minimapGrab {
			v.CenterOnMinimapPixel(in.CursorX, in.CursorY)
		}
		if in.PrimaryReleased {
			// Left up only clears a special cursor (`AI-MINIMAP-124`); it
			// issues nothing and it is not a click here.
			v.minimapGrab = false
			v.held = false
		}
		return ords, len(ords) > 0
	}

	// THE UNIT PANEL SWALLOWS ITS OWN PRESSES TOO (0140). It is the last of the
	// four boxes to be asked, because it is the only one drawn UNDER the other
	// three in the paint order and so must lose the pixel wherever two overlap.
	// panelCaptures' own doc carries why it exists and what it deliberately
	// does not do.
	if v.panelCaptures(in.CursorX, in.CursorY) {
		if in.PrimaryReleased {
			v.held = false
		}
		return nil, false
	}

	if in.PrimaryPressed {
		v.held = true
		v.dragMoved = 0
	}

	// THE LEFT BUTTON IS THE ONE THAT ACTS, AND IT ACTS ON THE RELEASE
	// (`AI-INPUT-121`). Until this story the map's two buttons were the other
	// way round here: the SECONDARY press issued every order and the primary
	// only ever selected. The decoded physical contract is
	// inactive-down-to-start, active-up-to-act -- left down starts the marquee
	// and stores the origin, left up ends it and is what the handler runs on --
	// so a press that never comes up orders nothing, and the release is the
	// only edge below that can produce an order.
	//
	// THE CURSOR IS TAKEN AT THE RELEASE POINT, not from the viewer's stored
	// hover name (`AI-CLICK-050`: a map click becomes an order by the CURSOR it
	// was made under). Resolving it here from this frame's own coordinates is
	// what keeps a headless drive -- which delivers a press and a release with
	// no hover tick between them -- deciding the same arm a played frame does.
	g := gesture{
		cursor:    v.gestureCursorAt(in.CursorX, in.CursorY),
		spell:     v.selectedSpell,
		pointCast: v.bookTargetsPoint(v.selectedSpell),
		shift:     v.shiftLatch,
		x:         in.CursorX, y: in.CursorY, px: v.pressX, py: v.pressY,
	}
	if in.PrimaryReleased {
		if v.dragMoved <= v.marqueeSlop() {
			g.tap = v.held
		} else {
			g.boxed = v.held && v.boxing
		}
		v.held = false
	}
	// Read AFTER the release has been accounted for, so the frame a press ends
	// on is not itself "a press in progress".
	g.held = v.held
	if g.tap && v.itemCast != nil {
		v.releaseItemCast(g.x, g.y)
		return nil, false
	}

	// Hover inspection is read-only. Only an explicit attack cursor can consume
	// this identity, on the acting release. Do not fall back to selected subject.
	if g.tap && g.cursor == "attack" {
		if ref, hit := v.inspectionAt(g.x, g.y); hit && ref.Kind == InspectionStructure {
			if panel, exists := v.inspectionPanel(ref); exists && panel.HP > 0 {
				g.structure = ref
			}
		}
	}
	if g.tap && g.cursor == "town" {
		if ref, hit := v.inspectionAt(g.x, g.y); hit && v.usableStructure(ref) {
			g.structure = ref
		}
	}
	hadUnits := len(presentSelected(v.sel, v.entities)) > 0
	sel, added, ords, ok := decide(v.sel, v.entities, v.groundCellAt, v.entityPickRect, v.localOwner, g)
	v.sel = sel
	if !g.shift {
		v.noteSelected(added)
	}
	// A plain click that selected no unit selects the structure under it when
	// the class allows (MISSION-065); one that missed everything clears it.
	if g.tap && !g.shift && !ok && (!hadUnits || g.cursor == "select") {
		v.selStructure = InspectionSubject{}
		if ref, hit := v.inspectionAt(g.x, g.y); hit && len(sel) == 0 && v.selectableStructure(ref) {
			v.selStructure = ref
		}
	}

	// ONE ACTING RELEASE, ONE ARM, SPENT (moved onto the left button by this
	// story). It is lowered after decide has read it and by the TAP alone --
	// whatever that tap produced, including a tap that produced nothing at all
	// and one that fell outside the map. Spending it only on a tap that issued
	// something would leave the mode up after a misfire, and the next tap would
	// attack a unit the player was reaching for with a move.
	//
	// A MARQUEE DOES NOT SPEND IT. A rectangle goes to the selection routine
	// and never reaches the cursor-to-order dispatch at all (`AI-SELECT-122`,
	// `AI-CLICK-050`), so it is not the press the mode was armed for.
	if g.tap {
		v.armed = false
		v.aimed = commandNone
	}
	// Keep a book spell ready for repeated casts, including Teleport. This
	// owner-directed behaviour replaces 0127's one-cast selection. Explicit
	// cancellation, another command or a changed book owner still ends the mode.
	// Item spells take their separate releaseItemCast path above.
	if ok && len(ords) > 0 && ords[0].spell != 0 {
		if v.castOnce && v.spellArmed {
			v.endCastOnce()
		} else {
			v.armSpell()
		}
	} else if g.tap && g.cursor == "cast" && v.castOnce && v.spellArmed {
		// A click under the cast cursor that issued no cast is the C hook's
		// miscast: the hook is spent like a cast.
		v.endCastOnce()
	}

	// THE RIGHT BUTTON CANCELS AND NEVER ORDERS (`AI-INPUT-127`): right up
	// releases capture, a MARKED DRAG performs no cancel, and a click cancels
	// an armed mode or deselects all when no mode is armed. rightPanned is the
	// mark; rightDragIntent in viewer.go raises it and step clears it.
	//
	// THE ORDER OF THE TWO CLAUSES IS THE CLAIM'S OWN and it is not a
	// preference: an armed mode absorbs the cancel, so a player who armed by
	// mistake gets the mode back down without losing the selection he armed it
	// for, and only a right click with nothing armed clears the selection.
	if in.SecondaryReleased && !v.rightPanned {
		v.cancelMapCommand()
	}
	return ords, ok
}

// cancelMapCommand is message 0x405's whole front-end effect
// (`AI-CURSOR-177`, `AI-INPUT-127`). An armed attack, aimed order or selected
// spell absorbs the cancel and leaves the selected units in place; with no
// such mode, the same cancel clears the selection. Both the ordinary map
// right-click and the mission character pane's right-up call this helper, so
// the pane cannot grow a second interpretation of the map message it posts.
func (v *Viewer) cancelMapCommand() {
	switch {
	case v.armed || v.aimed != commandNone || v.spellArmed || v.selectedSpell != 0 || v.itemCast != nil:
		// An armed Cast closes the spell popup whoever opened it
		// (`AI-CURSOR-177`).
		if v.spellArmed && v.hudShown(hudPanelBook) {
			v.toggleHudPanel(hudPanelBook)
		}
		v.castOnce = false
		v.armed = false
		v.aimed = commandNone
		v.selectedSpell = 0
		v.spellArmed = false
		v.itemCast = nil
		v.cmdOverlayHidden = false
	default:
		v.sel = nil
		v.selStructure = InspectionSubject{}
	}
}

// armAttack is the arming key's whole effect: it TOGGLES the attack mode,
// and raising it is gated.
//
// Lowering is ungated on purpose — a mode raised by accident must come down by
// the same key that put it up, and a gate on the way down would be a mode the
// player could not cancel once his selection changed under him.
//
// It is UNEXPORTED, which is the mechanism rather than a convention:
// cmd/mapview is another package and cannot call it, so the standalone
// viewer gaining no attack mode is Go's export rule.
func (v *Viewer) armAttack() {
	v.itemCast = nil
	v.spellArmed = false
	// THE PANEL'S OWN OVERLAY IS CLEARED BEFORE EITHER BRANCH BELOW (docs/
	// 1028-command-panel): a fresh raise or a lower both leave no stale
	// "hidden by a margin press" state behind them, so a key press and a
	// panel press reach the same visible result whichever raised the mode.
	v.cmdOverlayHidden = false
	if v.armed {
		v.armed = false
		return
	}
	v.armed = canArmAttack(v.sel, v.entities, v.localOwner)
	v.aimed = commandNone
}

// armCommand is the aimed-order keys' whole effect: it TOGGLES the named
// order, and raising it lowers the attack mode and drops any selected spell.
//
// TOGGLING, and toggling per ORDER: pressing the armed order's own key again
// disarms, and pressing the OTHER one replaces it. That is armAttack's shape,
// and it is what keeps a mode raised by accident cancellable by the key that
// raised it.
//
// RAISING IS UNGATED, where armAttack's is. canArmAttack already asks the
// ownership question and no class question, which is the right shape; what
// is authored here is that the standing orders skip the test. Nothing
// decoded says a standing order is refused over an owned selection, and an
// order issued over a selection that turns out to hold nothing orderable is
// a no-op on the far side rather than a state this tier must refuse. What it
// does need — that a press with nothing selected issues nothing — decide
// already answers for every press there is.
//
// It is UNEXPORTED for armAttack's own reason: cmd/mapview is another package
// and cannot call it, so the standalone viewer gaining no command mode is Go's
// export rule.
func (v *Viewer) armCommand(command uint8) {
	v.itemCast = nil
	v.spellArmed = false
	// See armAttack's own doc, one function up, for why this is cleared on
	// both branches.
	v.cmdOverlayHidden = false
	if v.aimed == command {
		v.aimed = commandNone
		return
	}
	v.aimed = command
	v.armed, v.attackHeld, v.selectedSpell = false, false, 0
}

// aimedOrder is which aimed order is armed, and it is the one question every
// reader asks — the readout's row and the gesture alike.
func (v *Viewer) aimedOrder() uint8 { return v.aimed }

// setAttackHeld adopts this tick's modifier level.
//
// IT ASSIGNS RATHER THAN LATCHES, which is the whole of "it is a level": a tick
// with the modifier up lowers the mode with no press and no second key, and the
// next tick with it down raises it again. Nothing is remembered between ticks
// except by the modifier still being held.
//
// It is UNGATED — no selection, no owner, no class — because the surface it
// reconstructs consults none of them. The gate lives on armAttack, which is the
// other surface, and this function deliberately does not call it.
//
// It is unexported for armAttack's own reason: cmd/mapview is another package
// and cannot reach it, so the standalone viewer gaining no attack mode is Go's
// export rule rather than a rule someone keeps.
func (v *Viewer) setAttackHeld(held bool) { v.attackHeld = held }

// clearAttackMode lowers BOTH writers at once.
//
// It exists because the two events that take the mode away — the window losing
// focus and a popup standing over the map — take it away whole, and a caller
// that lowered one of the two would leave a mode up that the player has no way
// to see the reason for. One function, two assignments, and no caller has to
// know there are two fields.
func (v *Viewer) clearAttackMode() { v.armed, v.attackHeld = false, false }

// attackMode is whether an attack is armed, by EITHER surface, and it is the
// one question every reader asks.
//
// Two writers and one predicate is what keeps the press, the pointer, the marker
// and the readout from being able to disagree about which mode is up. The
// difference between the surfaces is entirely in how each field comes to be set
// — a gated toggle spent by a press, and an ungated level held by a key — and
// nothing downstream of this line can tell which one raised it.
func (v *Viewer) attackMode() bool { return v.armed || v.attackHeld }

// SetLocalOwner adopts which roster slot the LOCAL PARTICIPANT holds — the
// other half of the arming gate's comparison.
//
// It is pushed by the tier that owns the world, in SetEntities' own shape and for
// its own reason: the answer is that tier's and re-deriving it here would be a
// second copy of it. ZERO IS NONE ESTABLISHED, so a caller that pushes nothing
// leaves the gate open, which is what every caller written before this story
// does.
func (v *Viewer) SetLocalOwner(owner uint32) { v.localOwner = owner }

// LocalOwner is the slot last pushed, and it exists for AttackArmed's own reason
// one function down: what the tier above pushed is otherwise observable only
// through the two rules that consume it — the arming gate and the numeral's
// drift — so a caller that pushed the wrong slot, or pushed none at all, would be
// caught by whichever of those two somebody happened to exercise. This is the
// value itself.
func (v *Viewer) LocalOwner() uint32 { return v.localOwner }

// AttackArmed reports whether an attack is armed, mirroring EntityMarkers'
// shape and reason: the state is observable without opening a window, and a
// front-end that silently armed nothing is otherwise indistinguishable from one
// that armed.
//
// It answers the PREDICATE and not the key's field, so an observer outside
// this package sees the one mode the press, the pointer and the marker all
// read, and cannot come to believe there are two.
func (v *Viewer) AttackArmed() bool { return v.attackMode() }

// absInt is |v| for a screen-pixel delta. The accumulator sums it over both
// axes, so a diagonal tick counts the two legs rather than the hypotenuse —
// the same quantity dragIntent hands the camera, one metric for both.
func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
