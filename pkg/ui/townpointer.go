package ui

import "image"

// townTipOwner is the popup a held Close belongs to: its tip table row, its
// construction and what it shows.
type townTipOwner struct {
	revision uint64
	row      TipOwner
	rect     image.Rectangle
	text     string
}

// townRoomPointerState is the press state a room composes with.
type townRoomPointerState struct {
	shopPress buttonLatch
	tipPress  buttonLatch
	tipOwner  townTipOwner
}

// townSurfaceLatchID is a town surface control's id in the surface latch:
// its kind above its index.
func townSurfaceLatchID(c TownSurfaceControl) int {
	return int(c.Kind)<<16 | c.Index&0xffff
}

// townSurfacePressed is the control the surface latch holds.
func (a *App) townSurfacePressed() TownSurfaceControl {
	id, held := a.townSurfacePress.Latched()
	if !held {
		return TownSurfaceControl{}
	}
	return TownSurfaceControl{Kind: TownSurfaceControlKind(id >> 16), Index: id & 0xffff}
}

// latchTownTip latches owner's tip close button; clearTownTip drops it.
func (a *App) latchTownTip(owner townTipOwner) {
	a.townTipPress.Clear()
	a.townTipPress.Press(0, true)
	a.townTipPressOwner = owner
}

func (a *App) clearTownTip() {
	a.townTipPress.Clear()
	a.townTipPressOwner = townTipOwner{}
}

func tipPanelOwner(v TipPanelView) townTipOwner {
	return townTipOwner{revision: v.Revision, row: v.Owner, rect: v.Rect, text: v.Text}
}

func tipPanelWithPointer(v TipPanelView, p image.Point, inside bool, press buttonLatch, owner townTipOwner) TipPanelView {
	v.Pointer, v.PointerOK = p, inside
	v.CloseHover = inside && p.In(TipPanelCloseRect(v.Rect))
	v.ClosePressed = press.Holds() && owner == tipPanelOwner(v)
	return v
}

func (a *App) currentTownTip() (TipPanelView, townTipOwner) {
	var v TipPanelView
	if a.flow.screen == ScreenChargen && a.flow.chargen != nil {
		v = a.flow.chargen.TipPanel()
	} else if a.flow.screen == ScreenTown {
		if _, open := townDialogue(a.flow.town); open {
			return v, townTipOwner{}
		}
		if surface, ok := townSurfaceScreen(a.flow.town); ok {
			v = surface.Tip
		} else if shop, ok := townShopScreen(a.flow.town); ok {
			v = shop.TipPanel
		} else if square, ok := townSquareView(a.flow.town); ok {
			v = square.Tip
		}
	}
	return v, tipPanelOwner(v)
}

// cancelTownTipPointer drops a held Close; a town popup's cancelled press
// also swallows its release, which the generator page takes itself.
func (a *App) cancelTownTipPointer() {
	if a.townTipPress.Holds() && a.townTipPressOwner.row.Pointer == "town" {
		a.suppressPrimaryRelease = true
	}
	a.clearTownTip()
}

func (a *App) observeChargenTipPress(in appInput) {
	if in.PrimaryPressed {
		a.clearTownTip()
		v, owner := a.currentTownTip()
		if p, inside := a.windowToNativeFrame(in.CursorX, in.CursorY); inside && v.Showing() && p.In(TipPanelCloseRect(v.Rect)) {
			a.latchTownTip(owner)
		}
	}
	if in.PrimaryReleased {
		a.clearTownTip()
	}
}

func (a *App) validateTownTipPointer(in appInput) {
	if !a.townTipPress.Holds() {
		return
	}
	v, owner := a.currentTownTip()
	if in.Unfocused || in.Close || in.Escape || in.Enter || in.SecondaryPressed || a.cutscene != nil || !v.Showing() || owner != a.townTipPressOwner {
		a.cancelTownTipPointer()
	}
}

func (a *App) stepTownTipPointer(in appInput, v TipPanelView, p image.Point, inside bool) bool {
	if !in.PrimaryPressed && !in.PrimaryReleased {
		return false
	}
	_, owner := a.currentTownTip()
	if in.PrimaryPressed {
		a.clearTownTip()
		if inside {
			kind, consumed := TipPanelEventAt(v, p, true)
			switch kind {
			case TipControlToggle:
				a.flow.toggleTips()
			case TipControlClose:
				a.latchTownTip(owner)
			}
			return consumed
		}
	}
	if in.PrimaryReleased {
		_, armed := a.townTipPress.Release(0, a.townTipPressOwner == owner)
		a.townTipPressOwner = townTipOwner{}
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
	a.townSurfacePress.Clear()
}
