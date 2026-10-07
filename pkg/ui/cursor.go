package ui

import (
	"image"
	"image/color"
	"math"
	"time"
)

// THE ATTACK POINTER AND THE TARGET MARKER — the whole of what a player
// sees of the attack mode, and the whole of what decides it.
//
// The file holds the DECISIONS and none of the drawing. Every function here is
// pure: it reads the mode, the cursor and the snapshot and answers with a
// picture, a point or a rectangle. Draw makes the engine calls and decides
// nothing. That is the unit panel's, the readout's and the notice's own shape
// and it is here for their reason — a decision taken inside a draw call is a
// decision no test without a window can see, and this one is the story's whole
// deliverable.
//
// WHY THE POINTER AND NOT A HINT SOMEWHERE. What is being reconstructed turns a
// click into an order BY THE CURSOR IT WAS MADE UNDER: the pointer is not a
// caption about the mode, it is the mode. So the affordance this front-end owed
// was never a label — it was the thing the player is already looking at.

// AttackMarkerColor is the outline drawn around the unit a press would name, and
// AttackMarkerWidth its stroke in SCREEN pixels.
//
// Both are OURS. Nothing decoded describes a target marker at all — the
// original's affordance is the pointer alone — so this asserts nothing about it
// and exists to answer the second question the pointer cannot: not "is a mode
// up" but "what will this press hit".
//
// The width does not scale with the zoom, for PathWidth's own reason: it is an
// instrument reporting a decision, not a thing standing on the map, so it stays
// legible zoomed out instead of thinning to nothing.
var AttackMarkerColor = color.RGBA{R: 0xff, G: 0x40, B: 0x40, A: 0xff}

// AttackMarkerWidth is the marker's stroke width in screen pixels.
const AttackMarkerWidth = 2

// The authored pointer — what is drawn at the cursor when no art was
// supplied.
//
// IT IS DELIBERATELY NOT A DRAWN SWORD. A hand-made imitation of art this build
// could not load would be the worst of both: it would assert a shape while
// claiming to assert none. Two crossed strokes say that a mode is up and say
// nothing else, and the colour is the marker's so the two instruments of this
// story read as one.
//
// Its arm is measured from the cursor point, so the cross is centred on the very
// pixel a press would be made at — where the art, having a point of its own,
// hangs from that pixel instead.
const (
	AttackAuthoredArm   = 7
	AttackAuthoredWidth = 2
)

// SetAttackPointer adopts the picture drawn at the cursor while the attack
// mode is up, or nil for none.
//
// ONE PICTURE CROSSES AND NOTHING ELSE — no archive, no container, no frame set,
// no animation. The tier that owns the install resolves the art whole and hands
// over the result, exactly as it hands over a font, so this package gains no
// format knowledge and no import for it. It cannot tell where a picture came
// from, which is what leaves a synthetic one in a test and the game's own art in
// a mission on one path.
//
// NIL IS "NONE SUPPLIED" and is the ordinary state: it is what the standalone
// developer viewer holds, what a front-end whose install would not yield the art
// holds, and what every viewer built before this story holds. The authored
// pointer is what makes that state visible rather than silent.
//
// THE UPLOAD IS NOT MARKED HERE any more. It used to set a "fresh" flag that
// the draw path cleared at the first upload, which was correct while one
// picture per viewer was all there ever was: the manager then began handing
// the draw path frames 1..9 of the ten-frame attack sheet and none of them was
// ever written to the texture. cursorTexture compares the source picture
// instead, so every frame that reaches the draw path reaches the screen.
func (v *Viewer) SetAttackPointer(pic *image.RGBA) {
	v.attackPointer = pic
}

// AttackPointerHotspot is the pixel of the attack sheet's own frames that names
// the cursor point: (3,3), the `attack` registration's hotspot argument
// (SPR16A-CURSOR-046, transcribed in pkg/game/cursorregistry.go).
//
// IT IS NOT AUTHORED any more. This build placed the picture's top-left at
// the cursor point and said so — "that hotspot is OURS: nothing read gives
// one" — which was true when it was written and stopped being true when
// the cursor registry was decoded. Where an authored value stands in for a
// decoded fact and no owner directive drove the difference, the decoded
// value wins (owner). It is used only for the story-0080 fallback picture,
// which arrives through SetAttackPointer with no registration behind it; a
// viewer drawing through the shared manager reads that slot's own hotspot
// instead.
var AttackPointerHotspot = image.Pt(3, 3)

func (v *Viewer) SetCursorManager(mgr *CursorManager) { v.cursorMgr = mgr }

// advanceCursorManager is step's own per-tick mutation of the shared cursor
// manager (1030 B3; 1031 B1-B4): while the attack mode is up, the manager's
// current cursor is "attack" (idempotent past the first tick, AI-CURSOR-193's
// guard); the manager's clock always advances, so a slot with more than one
// frame keeps animating for as long as it stays current.
//
// OUTSIDE ATTACK MODE, A HELD ITEM OVERRIDES EVERYTHING ELSE (1031 B4,
// AI-CURSOR-192): dragItemPresent draws its own picture directly, so the
// manager's current cursor is left exactly as it stood rather than set to
// anything — B2's persistence rule (AI-CURSOR-193) is what makes that correct,
// since the tick a drag ends picks a fresh name from the pointer's own current
// position anyway.
//
// OTHERWISE THE MISSION MAP'S OWN SELECTION RUNS (1031 mapCursorName,
// missioncursor.go): the edge arrows, the minimap's mode cursor and the
// hostility test at hover, falling back to `default`, which is also the exit
// cursor the map screen's surface transition already sets on entry
// (flow.screenExitCursor) — so a viewer whose camera has never stepped, or
// whose pointer names none of the three, still shows a cursor rather than
// none. Leaving the manager on "attack" once the mode ends was reachable and
// player-visible before this: a mission that ends sends the flow to the map
// list, which runs no transition of its own, and the list was then drawn
// under the animated attack sword.
//
// IT IS ONE FRAME BEHIND THE MODE, and that is a property of where it has to
// stand rather than a choice. Neither edge is visible: on the frame the mode
// goes up the manager still names whatever mapCursorName last picked, so
// attackPointerPresent falls back to the static story-0080 picture, which is
// frame 0 of the same sheet; on the frame it comes down nothing is drawn
// either way. What matters is that the lowering runs at all under a popup
// — the mission-end notice is one — so a mission that ends does not
// carry the sword to the map list.
func (v *Viewer) advanceCursorManager(now time.Time) {
	if v.cursorMgr == nil {
		return
	}
	switch {
	case v.attackShown():
		v.cursorMgr.SetCursor("attack")
		v.cursorAttackFromMode = true
	case v.dragActive:
		// See the doc above: the held item's own picture is drawn elsewhere,
		// and leaving the manager's current cursor (and cursorAttackFromMode)
		// untouched is correct.
	default:
		v.cursorMgr.SetCursor(v.mapCursorName())
		v.cursorAttackFromMode = false
	}
	v.cursorMgr.Advance(now.UnixMilli())
}

// attackShown is the ONE gate both instruments read: the mode is up, no popup
// stands over the map and the pointer is over the game area.
//
// THE POPUP TEST HERE IS NOT REDUNDANT WITH THE ONE THAT LOWERS THE MODE.
// That one lowers the STATE, inside the viewer's step, and it is what makes
// "not restored when the box is dismissed" true. This one is what makes the
// rule true on a frame composed without a step having run before it. They
// close different halves and each is one condition.
func (v *Viewer) attackShown() bool {
	return v.attackMode() && !v.popupOpen() && !(v.hasCursor && !v.gameAreaAt(v.cursorX, v.cursorY))
}

// gameAreaAt reports whether the frame position is on the game surface and not
// under a panel, the inventory, the book, the command panel or the minimap. The
// ordinary cursor and the order-mode cursors show only there; everywhere else
// the arrow stands.
func (v *Viewer) gameAreaAt(x, y int) bool {
	return v.mapSurfaceCaptures(x, y) && !v.groundSurfaceCaptures(x, y)
}

// attackPointerPresent is the picture to draw at the cursor, where it goes,
// and whether a pointer is drawn at all.
//
// THE THREE ANSWERS OF AC-5 ARE THIS FUNCTION'S THREE RETURNS. A false bool is
// "no pointer, and the system one is visible"; a true bool with a picture is the
// game's own art; a true bool with NIL is the authored cross. Splitting the last
// two across a field test inside Draw would put one third of the contract where
// no test without a window could reach it.
//
// The cursor must have been OBSERVED. Without that test a viewer that has never
// been stepped would draw its pointer at (0, 0) — a real pixel, in the corner,
// under nobody's hand — which is the readout's own reason for the same test.
//
// THE POINT IS WHERE THE THING GOES, and it is not the cursor position. For a
// picture it is the cursor position less the registration's own hotspot, so
// that the hotspot pixel — the blade tip, (3,3) — lands on the pixel a press
// names. For the authored cross, which has no art and no registration, it is
// the cursor position itself, which is the cross's centre. What a press
// actually names is the system cursor's position either way; the offset is
// what makes the drawn pointer agree with it.
func (v *Viewer) attackPointerPresent() (*image.RGBA, image.Point, bool) {
	if !v.attackShown() || !v.hasCursor {
		return nil, image.Point{}, false
	}
	// THE MANAGER'S ANIMATED FRAME WINS WHEN ONE IS AVAILABLE: ten frames at
	// the registered period, over the single still frame SetAttackPointer alone
	// ever supplied. The manager's own current name is checked rather than
	// assumed, so a manager set to some other cursor by a caller this story
	// does not anticipate never has its picture drawn here under the attack
	// marker's own colour and position.
	if v.cursorMgr != nil && v.cursorMgr.CurrentName() == "attack" {
		if pic, hot, ok := v.cursorMgr.Current(); ok {
			return pic, image.Pt(v.cursorX-hot.X, v.cursorY-hot.Y), true
		}
	}
	if v.attackPointer == nil {
		return nil, image.Pt(v.cursorX, v.cursorY), true
	}
	return v.attackPointer, image.Pt(v.cursorX-AttackPointerHotspot.X, v.cursorY-AttackPointerHotspot.Y), true
}

// attackTargetRect is the rectangle drawn around the unit a consuming press
// made at the current cursor would name, and whether there is one.
//
// IT CALLS THE PRESS'S OWN ATTACK HIT TEST rather than the selection hit.
// targetAt is what decide asks for the victim, over the same per-entity pick
// rectangle, so "what is outlined is what would be attacked" includes the
// -1 through -9 finishing band, the -10 refusal and the lowest-id tie rule.
//
// It looks the named id back up in the snapshot because the pick answers with an
// id and a rectangle is a property of the entry. The walk is the snapshot's own
// and stops at the first match, since ids are unique in it.
//
// THE RETURNED RECT IS CLIPPED TO THE WORLD VIEWPORT (adversarial pass 2, F2).
// entityPickRect is shared with topAt's own hit test above and must not be
// clamped — a unit whose art crosses into the right strip must still be
// pickable exactly as it is today. The marker is a world annotation drawn
// over the map only (viewer.go's own drawFrame comment), so the clamp is
// applied here, to this function's own return, and nowhere upstream of it.
func (v *Viewer) attackTargetRect() (screenRect, bool) {
	if !v.attackShown() || !v.hasCursor || !v.mapSurfaceCaptures(v.cursorX, v.cursorY) {
		return screenRect{}, false
	}
	id, hit := targetAt(v.entities, v.entityPickRect, float64(v.cursorX), float64(v.cursorY), false)
	if !hit {
		return screenRect{}, false
	}
	for _, e := range v.entities {
		if e.ID == id {
			// AND IT IS NOT DRAWN OVER A UNIT THAT MAY NOT BE SEEN (hotfix). An
			// outline appearing when the cursor crosses a dark cell is an enemy
			// indicator like any other -- it says something is standing there -- and
			// the owner's instruction is that none of them shows.
			//
			// THE PRESS ITSELF IS DELIBERATELY NOT GATED, and the asymmetry
			// is stated rather than accidental. Gating topAt's INPUT would
			// change what the player can target, not merely what he can see,
			// and whether the original refuses an attack on a unit outside
			// the party's sight is not decoded -- it is a question for a
			// story, not a thing to decide in a hotfix. So the outline's own
			// contract weakens in ONE SAFE DIRECTION: everything outlined is
			// still exactly what would be attacked, and there is now a case
			// where something would be attacked and is not outlined.
			if !v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
				return screenRect{}, false
			}
			r, ok := v.entityPickRect(e)
			if !ok {
				return screenRect{}, false
			}
			return clipScreenRectToViewport(r, float64(v.cam.ViewW), float64(v.cam.ViewH))
		}
	}
	return screenRect{}, false
}

// clipScreenRectToViewport intersects r with the world viewport, (0,0) to
// (w,h) in frame pixels, and reports whether anything of it remains. w and h
// are the viewer's own v.cam.ViewW/ViewH — the same rect mapSurfaceCaptures
// tests and MissionViewportSize/viewportSize compute (1026 B3; DIV-249): the
// frame minus the 160-pixel right strip.
//
// THE INSET IS THE STROKE'S, NOT THE RECT'S (adversarial pass 3). The only
// caller draws this return with vector.StrokeRect, which centres a
// strokeWidth-wide line on the rect's own edge — Ebiten's own arithmetic
// translates the right edge to x+width-strokeWidth/2, so a rect clipped
// flush to w paints strokeWidth/2 past it. A rect clipped to exactly x=864
// (this build's own MissionPanelRect().Min.X) painted x in [863,865): one
// pixel column inside the panel, on every camera position, not only a
// straddling one. Insetting the bound the rect is clipped TO, by half the
// stroke's own width, is what makes the PAINTED span stop at the true edge
// rather than the rect clipped to the edge with a stroke still centred on
// it.
func clipScreenRectToViewport(r screenRect, w, h float64) (screenRect, bool) {
	const strokeInset = AttackMarkerWidth / 2
	x0, y0 := math.Max(r.X, strokeInset), math.Max(r.Y, strokeInset)
	x1, y1 := math.Min(r.X+r.W, w-strokeInset), math.Min(r.Y+r.H, h-strokeInset)
	if x1 <= x0 || y1 <= y0 {
		return screenRect{}, false
	}
	return screenRect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}, true
}

// pointerModeChange adopts whether the system pointer should be hidden and
// reports whether the engine has to be told.
//
// ONE BOOL DRIVES THE HIDE AND THE DRAW, and that is the whole mitigation for
// the risk this story's own change creates: a frame that hides the system
// pointer and then draws nothing is a map with NO pointer at all, which is worse
// than the defect being fixed. Taking both from attackPointerPresent's single
// answer makes that state unreachable rather than merely unlikely.
//
// The engine call is made only on a CHANGE. Not for cost — it is a cheap call —
// but so that a program which never shows a map never asks the engine for
// anything, and so that the standalone developer viewer's frame is unchanged
// down to the calls it makes. The zero value is false, which is the engine's own
// default, so nothing has to establish a baseline.
//
// It is separated from the call itself so that the tracking is reachable in a
// test with no window: what AC-6 asserts is that the wanted state equals the
// pointer's own bool on every frame, and that is this field.
func (v *Viewer) pointerModeChange(hidden bool) bool {
	if hidden == v.pointerHidden {
		return false
	}
	v.pointerHidden = hidden
	return true
}
