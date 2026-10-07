package ui

import "image"

// THE MISSION MAP'S OWN CURSOR SELECTION.
//
// THE THREE ARE TRIED IN THIS ORDER because the decoded routines are:
// AI-CURSOR-190's own routine tests the four screen edges before falling to
// its "normal hover path", and AI-CURSOR-207's block inside the minimap's own
// selector does the same, jumping past its own widget test on an edge match.
// A pointer at both an edge and a unit's own picked rectangle is drawn as the
// edge, never the unit, for that reason.

// missionEdgeBands is the window's own edge-scroll band membership for a
// position: the four flags panIntent adds a pan term for and the four
// missionEdgeArrow names an arrow from. It is ONE function so that the two
// cannot come to describe different bands, which is the shape step already
// uses for the camera itself.
//
// A position outside the window, and a window with no area, are in no band.
// panIntent has always refused both — a pointer that has left the window
// reports a position, and treating it as an edge would pan the map for as long
// as it sat on another window — and the arrow refuses them for the same
// reason.
func missionEdgeBands(x, y, w, h, margin int) (left, right, top, bottom bool) {
	if w <= 0 || h <= 0 || x < 0 || y < 0 || x >= w || y >= h {
		return false, false, false, false
	}
	return x < margin, x >= w-margin, y < margin, y >= h-margin
}

// edgeScrollBands is missionEdgeBands over THIS TICK'S pointer position,
// plus the two conditions that decide whether the edge-scroll term runs at
// all: a held primary button suppresses it (0014 SC-15 — a tick that pans
// by drag must not also edge-scroll), and a viewer that has never observed a
// cursor has no position to test.
//
// BOTH READERS ASK THIS ONE FUNCTION, and that is the whole of what adversarial
// pass 1's second finding cost: missionEdgeArrow read the band and no drag
// state, so a pointer resting in the band during a drag drew a directional
// arrow on every tick while the camera did not move. The claim the doc below
// makes about this build — an arrow shows exactly on the ticks the edge-scroll
// term runs — is now a property of there being one function rather than of two
// call sites agreeing.
//
// primaryDown is a parameter rather than a field read because panIntent is
// handed the tick's own Input and mapCursorName is not: step stores
// v.primaryDown in the same block it stores the cursor position, and the
// cursor path passes that.
//
// IT DOES NOT ASK WHETHER THE CAMERA CAN STILL MOVE. A camera already clamped
// at the world's edge pans nothing while the term runs, and the arrow still
// shows there. That is the original's own shape too: AI-CURSOR-190's routine
// tests the screen edge and nothing about the camera's limits.
func (v *Viewer) edgeScrollBands(primaryDown bool) (left, right, top, bottom bool) {
	if primaryDown || !v.hasWinCursor {
		return false, false, false, false
	}
	win := v.place.WindowSize()
	return missionEdgeBands(v.winCursorX, v.winCursorY, win.X, win.Y, EdgeMargin)
}

func missionEdgeArrow(left, right, top, bottom bool) (string, bool) {
	switch {
	case left && top:
		return "arrow7", true
	case left && bottom:
		return "arrow5", true
	case left:
		return "arrow6", true
	case right && top:
		return "arrow1", true
	case right && bottom:
		return "arrow3", true
	case right:
		return "arrow2", true
	case top:
		return "arrow0", true
	case bottom:
		return "arrow4", true
	}
	return "", false
}

// The owner-facing minimap is a camera control, independent of armed orders.
func (v *Viewer) minimapModeCursor() (string, bool) {
	if !v.hasCursor {
		return "", false
	}
	return v.minimapActionCursor(v.cursorX, v.cursorY)
}

func (v *Viewer) minimapActionCursor(x, y int) (string, bool) {
	if !v.minimapCaptures(x, y) {
		return "", false
	}
	return "default", true
}

// AI-CURSOR-052, DIV-262, DIV-270
func (v *Viewer) missionHoverCursor() (string, bool) {
	if !v.hasCursor {
		return "", false
	}
	// REPLACEMENT (1), THE MARQUEE (`AI-CURSOR-230`): a live marquee gives
	// `default`. It is tested first here rather than last because every arm
	// below it would be overwritten by it anyway, and a replacement that
	// overwrites unconditionally and a gate that returns early are the same
	// function.
	if v.marqueeCursorLive() {
		return "default", true
	}
	// REPLACEMENTS (2) AND (3), the map view's children 2 and 3.
	// groundSurfaceCaptures is that set, already written for the ground drop
	// and already the five boxes this build draws over the map.
	if !v.gameAreaAt(v.cursorX, v.cursorY) {
		return "default", true
	}
	// REPLACEMENTS (4) AND (5), the held-item cursor and `backpack`, are NOT
	// applied here and are a named gap (`DIV-290`, the same row that carries
	// (2) and (3): one row per claim, `AI-CURSOR-230`'s replacement list). This build draws the
	// dragged item's own picture at the pointer while a drag is live
	// (dragItemPresent, inventory.go) and mapCursorPresent already refuses to
	// draw a manager cursor over it, so replacement (4)'s effect -- the hover
	// cursor does not answer while an item is held -- is reproduced; the slot
	// the original names and `backpack`'s own six conditions are not.
	if name, ok := v.armedMissionCursorAt(v.cursorX, v.cursorY); ok {
		return name, true
	}
	mask, hit, hasHit := v.hoverMask(v.cursorX, v.cursorY)
	present := presentSelected(v.sel, v.entities)
	return hoverCursorName(present, v.selectionSummary(), mask, hit, hasHit,
		v.ctrlLatch, v.altLatch), true
}

// gestureCursorAt is the cursor a MAP CLICK at this frame position was made
// under, and it is what turns that click into an order (`AI-CLICK-050`, story
// 1034). It is the same cascade missionHoverCursor runs, at a caller-supplied
// point and without the drawn cursor's own three concerns.
//
// IT DOES NOT TEST hasCursor. The hover cursor answers "what picture is on
// screen", and there is no picture with no pointer; a click is delivered with
// its own coordinates whether or not a hover tick preceded it, which is exactly
// the headless case -- a scenario delivers a press and a release with no move
// between them, and the arm it takes must be the one a played frame takes.
//
// IT DOES NOT APPLY THE MARQUEE OR CHILD-WIDGET REPLACEMENTS. Both give
// `default`, which has no arm, and both are already handled before any cursor
// is read: a marquee release goes to the rectangle form inside `decide` without
// consulting the cursor at all, and every child widget this build draws over
// the map swallows its own press earlier in `command`.
//
// The armed-mode table comes first, as it does in missionHoverCursor and for
// the same reason: an armed mode replaces the whole hover cascade
// (`AI-PANEL-053`'s eight-entry table, indexed by the mode).
func (v *Viewer) gestureCursorAt(x, y int) string {
	// A CLICK THAT IS NOT ON THE MAP SURFACE HAS NO ARM AT ALL, and that
	// includes no selection: in what is being reconstructed the map view's own
	// window handler is the only route to this dispatch, so a click outside it
	// never reaches the routine. Here the four HUD boxes swallow their own
	// presses earlier in command, and this is what answers for the letterbox
	// and for the right strip's own margins. groundCellAt already refuses a
	// cell off the surface (overlay.go, DIV-212); this is the same refusal
	// applied to the SELECTION half, which topAt does not gate.
	if !v.mapSurfaceCaptures(x, y) {
		return ""
	}
	if name, ok := v.armedMissionCursorAt(x, y); ok {
		return name
	}
	mask, hit, hasHit := v.hoverMask(x, y)
	present := presentSelected(v.sel, v.entities)
	return hoverCursorName(present, v.selectionSummary(), mask, hit, hasHit,
		v.ctrlLatch, v.altLatch)
}

func (v *Viewer) armedMissionCursorAt(x, y int) (string, bool) {
	mode := v.missionMode()
	if mode == modeCast {
		return v.castCursorAt(x, y)
	}
	return missionModeCursor(mode)
}

// mapCursorName is the mission map's own cursor for this tick, outside
// attack mode and outside a held item — advanceCursorManager's other two
// branches, which this function is never reached from. B1 first, since
// AI-CURSOR-190's and AI-CURSOR-207's own routines test the screen edge
// before falling to anything else; then B2; then B3; `default` otherwise,
// the map's own surface-transition cursor (1030 B4) standing until one of
// the three applies.
//
// IT READS THIS TICK'S POSITION. step stores the cursor and calls
// advanceCursorManager below those two statements, so the name chosen here and
// the place mapCursorPresent draws the picture at come from one tick. Above
// them — where the call stood until adversarial pass 1's first finding — every
// branch here chose a name for where the pointer WAS while the picture went
// where it IS.
//
// A POPUP ANSWERS `default` AND NONE OF THE THREE. step returns above
// panIntent, above the drag and above the wheel while a popup stands, so an
// arrow, a minimap mode cursor or a hover cursor named on such a tick
// describes an interaction the frame will not perform. `default` rather than
// that routine's no-change exit, because the manager must not be left
// holding `attack` when a mission-end notice sends the flow to the map list
// — advanceCursorManager's own reason for running above the popup return
// at all.
func (v *Viewer) mapCursorName() string {
	if v.popupOpen() {
		return "default"
	}
	if name, ok := v.minimapModeCursor(); ok {
		return name
	}
	if name, ok := missionEdgeArrow(v.edgeScrollBands(v.primaryDown)); ok {
		return name
	}
	if name, ok := v.missionHoverCursor(); ok {
		return name
	}
	return "default"
}

// mapCursorPresent is B5's other half of the picture: what to draw for the
// mission map's own cursor, and where its hotspot lands, outside attack mode
// (attackPointerPresent owns that picture) and outside a held item
// (dragItemPresent owns that one). It is the shared manager's current
// picture — the name mapCursorName picked on the last tick step ran, through
// advanceCursorManager — at the cursor position, 1:1 in mission-frame pixels:
// App.Draw scales the frame as a whole afterwards, exactly as it does for
// every other screen (DIV-249), and the hotspot is subtracted before that
// scale, in the same frame space attackPointerPresent uses.
//
// THE FOURTH EXCLUSION IS BY PROVENANCE, NOT BY NAME. A guard here used to
// refuse to draw whenever CurrentName() == "attack", on the theory that only
// the attackShown branch of advanceCursorManager could have put the manager
// in that state. The theory was false: hoverHostilityCursor (B3) legitimately
// names "attack" on an unmodified hostile hover, and that name reaches the
// manager through this same function's default branch. The guard then
// refused to draw a legitimate hostile-hover selection, leaving the system
// arrow over the enemy unit (adversarial pass 2, F1).
//
// cursorAttackFromMode is what the guard reads instead, and it is exactly the
// state this exclusion exists for: the ONE tick advanceCursorManager runs
// behind the mode (its own header) leaves the manager holding the PREVIOUS
// tick's attackShown() write on the tick the mode comes down, and nothing
// must be drawn on that one tick either (TestTheSystemPointerIsHiddenExactly
// -WhenThisBuildDrawsOne). hoverHostilityCursor's own "attack" never sets the
// flag, so it is never excluded here.
func (v *Viewer) mapCursorPresent() (*image.RGBA, image.Point, bool) {
	if v.attackShown() || v.dragActive || !v.hasCursor || v.cursorMgr == nil {
		return nil, image.Point{}, false
	}
	if v.cursorAttackFromMode {
		return nil, image.Point{}, false
	}
	pic, hot, ok := v.cursorMgr.Current()
	if !ok {
		return nil, image.Point{}, false
	}
	return pic, image.Pt(v.cursorX-hot.X, v.cursorY-hot.Y), true
}
