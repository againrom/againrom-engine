package ui

import (
	"image"
	"image/color"
	"time"
)

// AttackMarkerColor is the authored attack pointer cross color.
var AttackMarkerColor = color.RGBA{R: 0xff, G: 0x40, B: 0x40, A: 0xff}

// The authored attack pointer is a centered cross when pointer art is absent.
const (
	AttackAuthoredReach = 7
	AttackAuthoredWidth = 2
)

// SetAttackPointer supplies a still picture. A resolved attack slot owns its hotspot;
// without a slot the pointer uses the authored cross.
func (v *Viewer) SetAttackPointer(pic *image.RGBA) {
	v.attackPointer = pic
}

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
// The manager advances after the camera and pointer settle in step.
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

// attackShown gates the attack pointer on mode, popup and game area.
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

// attackPointerPresent places the animated attack slot or its still picture
// at the pointer minus the registered hotspot. Without a slot it draws the cross.
func (v *Viewer) attackPointerPresent() (*image.RGBA, image.Point, bool) {
	if !v.attackShown() || !v.hasCursor {
		return nil, image.Point{}, false
	}
	// THE MANAGER'S ANIMATED FRAME WINS WHEN ONE IS AVAILABLE: ten frames at
	// the registered period, over the single still frame SetAttackPointer alone
	// ever supplied. The manager's own current name is checked rather than
	// assumed, so a manager set to some other cursor by a caller this story
	// does not anticipate never has its picture drawn here under the attack
	// pointer's own position.
	if v.cursorMgr != nil && v.cursorMgr.CurrentName() == "attack" {
		if pic, hot, ok := v.cursorMgr.Current(); ok {
			return pic, image.Pt(v.cursorX-hot.X, v.cursorY-hot.Y), true
		}
	}
	if v.cursorMgr != nil {
		if slot, ok := v.cursorMgr.registry.Slot("attack"); ok && len(slot.Frames) > 0 {
			pic := v.attackPointer
			if pic == nil {
				pic = slot.Frames[0]
			}
			return pic, image.Pt(v.cursorX-slot.Hotspot.X, v.cursorY-slot.Hotspot.Y), true
		}
	}
	return nil, image.Pt(v.cursorX, v.cursorY), true
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
