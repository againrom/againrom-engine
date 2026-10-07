package ui

import (
	"fmt"
	"image"
	"image/draw"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func (a *App) updateTooltip(in appInput, now time.Time, changed bool) {
	a.tooltipPoint = image.Pt(in.CursorX, in.CursorY)
	if a.flow.viewer != nil {
		a.flow.viewer.tooltip, a.flow.viewer.tooltipManaged = &a.tooltip, true
	}
	blocked := changed || in.Unfocused || a.cutscene != nil || a.cutsceneDrain ||
		in.AnyKey || in.AnyHeld || in.PrimaryPressed || in.PrimaryReleased ||
		in.SecondaryPressed || in.SecondaryReleased || in.Viewer.PrimaryDown ||
		in.Viewer.SecondaryDown || in.WheelY != 0
	if blocked {
		a.tooltip.observe(now, a.tooltipPoint, "", true)
		return
	}
	target := a.tooltipTargetWithSurface(a.tooltipSurface)
	a.tooltip.observe(now, a.tooltipPoint, target.key(), false)
}

func (v *Viewer) updateTooltip(in Input, now time.Time) {
	if v.tooltipManaged {
		return
	}
	if v.tooltip == nil {
		v.tooltip = &tooltipController{delay: DefaultTooltipDelay}
	}
	blocked := in.Unfocused || in.PrimaryDown || in.SecondaryDown || in.WheelY != 0 ||
		in.PanLeft || in.PanRight || in.PanUp || in.PanDown || v.popupOpen() || v.dragActive
	v.tooltip.observe(now, image.Pt(in.CursorX, in.CursorY), v.tooltipTarget().key(), blocked)
}

func (v *Viewer) SetTooltipDelay(ms int) {
	if v.tooltip == nil {
		v.tooltip = &tooltipController{}
	}
	v.tooltip.setDelay(ms)
}

func (v *Viewer) tooltipTarget() tooltipTarget {
	if !v.hasCursor || v.popupOpen() || v.dragActive ||
		!image.Pt(v.cursorX, v.cursorY).In(image.Rect(0, 0, v.frameW, v.frameH)) {
		return tooltipTarget{}
	}
	if lines, ok := v.hoveredItemInfoAt(v.cursorX, v.cursorY); ok {
		return tooltipTarget{tooltipItem, fmt.Sprintf("item/%d", v.invSubject.ID), lines, v.font}
	}
	// Unavailable answers no target rather than falling through to whatever
	// sits under the same rectangle (TEXT-HOVERTEXT-052's required
	// availability bit): a cell the selected units cannot cast states
	// nothing, exactly as spellbookEntryAt still lets it be CLICKED so a
	// player can see which catalog cells exist at all.
	if i, ok := v.spellbookEntryAt(v.cursorX, v.cursorY); ok && i >= 0 && i < len(v.spellbook) {
		if spell := v.spellbook[i]; !spell.Unavailable {
			return tooltipTarget{tooltipSpell, fmt.Sprintf("spell/%d", spell.ID), spell.Info, v.font}
		}
	}
	if cell, ok := v.commandCellAt(v.cursorX, v.cursorY); ok && !commandPanelCellSkipped(cell) &&
		commandPanelActive(v.sel, v.entities, v.localOwner) {
		return tooltipTarget{tooltipCommand, fmt.Sprintf("command/%d/%d", cell, v.selectedSpell),
			tooltipLines(commandTooltipLabel(v, cell)), v.font}
	}
	return v.additionalTooltipAt(image.Pt(v.cursorX, v.cursorY))
}

func (v *Viewer) tooltipPresent() (*image.RGBA, image.Point, bool) {
	target := v.tooltipTarget()
	if !v.tooltip.visible(target.key()) {
		return nil, image.Point{}, false
	}
	return v.tooltipPicture(target)
}

func (v *Viewer) tooltipPicture(target tooltipTarget) (*image.RGBA, image.Point, bool) {
	if v.tooltip == nil {
		v.tooltip = &tooltipController{delay: DefaultTooltipDelay}
	}
	if !v.tooltipManaged {
		v.tooltip.baseFont = v.cardFont()
		if v.tooltip.baseFont == nil {
			v.tooltip.baseFont = v.font
		}
	}
	return v.tooltip.picture(target, image.Pt(v.cursorX, v.cursorY), image.Rect(0, 0, v.frameW, v.frameH))
}

func (a *App) tooltipPresent(bounds image.Rectangle) (*image.RGBA, image.Point, bool) {
	target := a.tooltipTarget()
	if !a.tooltip.visible(target.key()) {
		return nil, image.Point{}, false
	}
	point, ok := a.place.WindowToFrame(a.tooltipPoint.X, a.tooltipPoint.Y)
	if !ok {
		return nil, image.Point{}, false
	}
	return a.tooltip.picture(target, point, bounds)
}

func (a *App) drawTooltip(dst *ebiten.Image) {
	// The tooltip is its own composed picture, pasted at at (DIV-1385, owner
	// decision method C) — the same coordinate-space shift overlayTownDialogue
	// applies, for the same reason: markCapture/shiftCapture are no-ops
	// outside an open capture window, so this costs nothing when smoothing
	// is off.
	start := markCapture()
	pic, at, ok := a.tooltipPresent(dst.Bounds())
	if !ok {
		return
	}
	shiftCapture(start, at)
	if a.tooltipTexture == nil || a.tooltipTexture.Bounds().Size() != pic.Bounds().Size() {
		if a.tooltipTexture != nil {
			a.tooltipTexture.Dispose()
		}
		a.tooltipTexture = ebiten.NewImage(pic.Bounds().Dx(), pic.Bounds().Dy())
	}
	a.tooltipTexture.WritePixels(pic.Pix)
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	dst.DrawImage(a.tooltipTexture, &op)
	a.presentLog.over(pic, at)
}

func (a *App) paintTooltip(dst *image.RGBA) {
	if pic, at, ok := a.tooltipPresent(dst.Bounds()); ok {
		draw.Draw(dst, pic.Bounds().Add(at), pic, pic.Bounds().Min, draw.Over)
	}
}
