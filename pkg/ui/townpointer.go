package ui

import (
	"image"
	"time"
)

type townTipOwner struct {
	revision uint64
	room     uint8
	rect     image.Rectangle
	text     string
}

type townTipPress struct {
	owner townTipOwner
	armed bool
}

const chargenTipRoom uint8 = 5

type townRoomPointerState struct {
	shopPress ShopControl
	tipPress  townTipPress
}

func tipPanelOwner(v TipPanelView, room uint8) townTipOwner {
	return townTipOwner{revision: v.Revision, room: room, rect: v.Rect, text: v.Text}
}

func tipPanelWithPointer(v TipPanelView, room uint8, p image.Point, inside bool, press townTipPress) TipPanelView {
	v.Pointer, v.PointerOK = p, inside
	v.CloseHover = inside && p.In(TipPanelCloseRect(v.Rect))
	v.ClosePressed = press.armed && press.owner == tipPanelOwner(v, room)
	return v
}

func (a *App) currentTownTip() (TipPanelView, townTipOwner) {
	var v TipPanelView
	var room uint8
	if a.flow.screen == ScreenChargen && a.flow.chargen != nil {
		v, room = a.flow.chargen.TipPanel(), chargenTipRoom
	} else if a.flow.screen == ScreenTown {
		if _, open := townDialogue(a.flow.town); open {
			return v, townTipOwner{}
		}
		if surface, ok := townSurfaceScreen(a.flow.town); ok {
			v, room = surface.Tip, uint8(surface.Kind)+1
		} else if shop, ok := townShopScreen(a.flow.town); ok {
			v, room = shop.TipPanel, 3
		} else if square, ok := townSquareView(a.flow.town); ok {
			v, room = square.Tip, 4
		}
	}
	return v, tipPanelOwner(v, room)
}

func (a *App) cancelTownTipPointer() {
	if a.townTipPress.armed && a.townTipPress.owner.room != chargenTipRoom {
		a.suppressPrimaryRelease = true
	}
	a.townTipPress = townTipPress{}
}

func (a *App) observeChargenTipPress(in appInput) {
	if in.PrimaryPressed {
		a.townTipPress = townTipPress{}
		v, owner := a.currentTownTip()
		if p, inside := a.windowToNativeFrame(in.CursorX, in.CursorY); inside && v.Showing() && p.In(TipPanelCloseRect(v.Rect)) {
			a.townTipPress = townTipPress{owner: owner, armed: true}
		}
	}
	if in.PrimaryReleased {
		a.townTipPress = townTipPress{}
	}
}

func (a *App) validateTownTipPointer(in appInput) {
	if !a.townTipPress.armed {
		return
	}
	v, owner := a.currentTownTip()
	if in.Unfocused || in.Close || in.Escape || in.Enter || in.SecondaryPressed || a.cutscene != nil || !v.Showing() || owner != a.townTipPress.owner {
		a.cancelTownTipPointer()
	}
}

func (a *App) stepTownTipPointer(in appInput, v TipPanelView, p image.Point, inside bool) bool {
	if !in.PrimaryPressed && !in.PrimaryReleased {
		return false
	}
	_, owner := a.currentTownTip()
	if in.PrimaryPressed {
		a.townTipPress = townTipPress{}
		if inside {
			kind, consumed := TipPanelEventAt(v, p, true)
			switch kind {
			case TipControlToggle:
				a.flow.toggleTips()
			case TipControlClose:
				a.townTipPress = townTipPress{owner: owner, armed: true}
			}
			return consumed
		}
	}
	if in.PrimaryReleased {
		armed := a.townTipPress.armed && a.townTipPress.owner == owner
		a.townTipPress = townTipPress{}
		if inside {
			kind, consumed := TipPanelEventAt(v, p, false)
			if kind == TipControlClose {
				if armed {
					a.flow.closeTip()
				}
				return armed
			}
			return consumed || armed
		}
		return armed
	}
	return false
}

func (a *App) townPaintAllowed() bool {
	if state, ok := a.flow.town.(interface{ TownDialogueShows() uint64 }); ok {
		if state.TownDialogueShows() != 0 {
			return false
		}
	} else if _, shown := townDialogue(a.flow.town); shown {
		return false
	}
	return a.townPaintAdmission == nil || a.townPaintAdmission()
}

// SetTownPaintAdmission supplies the client paint-blocking policy. Nil admits.
func (a *App) SetTownPaintAdmission(admit func() bool) {
	a.townPaintAdmission = admit
}

func (a *App) townEntryState() (bool, uint64) {
	active := a.flow.screen == ScreenTown && atTownSquare(a.flow.town)
	var revision uint64
	if t, ok := a.flow.town.(interface{ TownSquareEntryRevision() uint64 }); ok {
		revision = t.TownSquareEntryRevision()
	}
	return active, revision
}

func (a *App) paintTownEntry(before bool, revision uint64, active bool) {
	after, next := a.townEntryState()
	if _, open := townDialogue(a.flow.town); open {
		return
	}
	if !active || !after || before && revision == next || !a.townPaintAllowed() {
		return
	}
	if t, ok := a.flow.town.(TownSquareAnimator); ok {
		t.TownSquareActive(true)
	}
	_, _ = a.composeTownRoom()
}

func (a *App) resetTownSurfacePair() {
	a.townSurfacePress, a.townSurfaceClick = TownSurfaceControl{}, TownSurfaceControl{}
	a.townSurfaceAt = time.Time{}
	a.townSurfaceKey, a.townSurfaceReleased = "", false
	a.townSurfaceRevision = 0
}
