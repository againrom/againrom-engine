package ui

import (
	"image"
	"image/color"

	"againrom/pkg/render/text"
)

// The item popup (0151, defect 5): a small text box, over whichever cell of
// the worn box or the pack grid the cursor stands on, stating that cell's
// InventorySubject.SlotInfo or PackInfo lines — the name and, where this
// tree decodes them, the characteristics pkg/game's itemInfoLines already
// composed (spec, docs/0151-layers-and-names).
//
// IT IS A HOVER, NOT A PRESS: it reads v.cursorX/v.cursorY/v.hasCursor, the
// SAME per-frame cursor readout.go's own cell readout already uses, and
// touches no click state — inventoryCaptures, invClickFrames and every
// other field command.go owns are untouched by this file (fence: "the
// popup gates nothing and issues nothing").
//
// IT DRAWS THROUGH composeItemPopup, BELOW: a framed box of shadowed lines, the
// frame painted there so the popup's text stays legible over the map.

// itemPopupBorderPad is the fill kept between the frame's one-pixel border
// (fillPanelFrame, panel.go) and the first glyph, on every side — so the
// text stays legible against invFill rather than crowding invBorder. It is
// added on top of the shadow's clearance; this box needs a second margin for
// the border it alone draws.
const itemPopupBorderPad = 3

const (
	// popupShadow is the offset a line's shadow stands at behind its face,
	// and popupLinePad is added to the font's height for the pitch between
	// two lines.
	popupShadow  = 1
	popupLinePad = 2
)

var (
	// popupTextColor is the cream ink of the notice box, and popupShadowColor
	// the opaque black behind it, for legibility over the map's own art.
	popupTextColor   = color.RGBA{R: 0xf2, G: 0xe6, B: 0xc4, A: 0xff}
	popupShadowColor = color.RGBA{A: 0xff}
)

// composeItemPopup composes lines over the project's usual framed
// background rather than as bare text: fillPanelFrame's fill and
// one-pixel border (panel.go), in invFill and invBorder — inventory.go's
// own colours, reused rather than a third palette invented for one more
// box standing over the same worn box and pack bar RenderDoll, RenderWorn
// and renderPackBar already fill with them.
//
// The width and line pitch come from the font, and each line takes two draws
// (shadow then face, at popupShadow's offset, in popupShadowColor and
// popupTextColor), moved in from the canvas edge by the border and
// itemPopupBorderPad.
//
// nil for nothing to draw: no font, a font holding no record, no lines, or
// lines that measure to no width at all.
func composeItemPopup(lines []string, f *text.Font) *image.RGBA {
	if f == nil || f.Height() <= 0 || len(lines) == 0 {
		return nil
	}
	width := 0
	for _, ln := range lines {
		w, _ := f.Measure(ln)
		width = max(width, w, f.Advance(ln))
	}
	if width <= 0 {
		return nil
	}
	pitch := f.Height() + popupLinePad
	pad := 1 + itemPopupBorderPad // the border pixel itself, plus the clearance kept off it
	size := image.Pt(width+popupShadow+2*pad, len(lines)*pitch+popupShadow+2*pad)
	img := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	fillPanelFrame(img, size, invFill, invBorder)
	for i, ln := range lines {
		x, y := pad, pad+i*pitch
		f.Draw(img, ln, x+popupShadow, y+popupShadow, popupShadowColor)
		f.Draw(img, ln, x, y, popupTextColor)
	}
	return img
}

// hoveredItemInfoAt is the popup text for whichever cell of the worn box,
// the doll figure or the pack bar stands under the window pixel (x, y), or
// false for a pixel that names no cell carrying one.
//
// THE WORN BOX IS ASKED FIRST, THE DOLL FIGURE SECOND AND THE PACK BAR
// THIRD. The doll box used to carry no per-cell text at all, on the ground
// that it draws one whole figure and not twelve cells — 1005 ("the
// interactive doll") is what changes that: dollFigureSlotAt answers a slot
// number over the SAME mask a press reads, so a cursor over an occupied
// layer's own pixels shows that slot's line, and one over the figure's
// transparent pixels or its background still shows nothing, exactly as a
// point outside every box already does.
func (v *Viewer) hoveredItemInfoAt(x, y int) ([]string, bool) {
	if i, ok := v.inventoryWornSlotAt(x, y); ok {
		if len(v.invSubject.SlotInfo[i]) == 0 {
			return nil, false
		}
		return v.invSubject.SlotInfo[i], true
	}
	if i, ok := v.dollFigureSlotAt(x, y); ok {
		if len(v.invSubject.SlotInfo[i]) == 0 {
			return nil, false
		}
		return v.invSubject.SlotInfo[i], true
	}
	if idx, ok := v.inventoryPackCellAt(x, y); ok {
		if idx < 0 || idx >= len(v.invSubject.PackInfo) || len(v.invSubject.PackInfo[idx]) == 0 {
			return nil, false
		}
		return v.invSubject.PackInfo[idx], true
	}
	return nil, false
}

// itemPopupPresent is the popup's picture for this frame and where its
// top-left corner goes, or false for a frame that draws none: no cursor
// observed yet, a cursor over no cell carrying a line, or a font this
// viewer holds no record for.
//
// IT RECOMPOSES EVERY FRAME IT DRAWS, packBarPresent's own choice
// (inventory.go): a handful of short lines is cheap, and the alternative —
// a key that had to carry the hovered cell's own identity — costs more to
// keep honest than the composition costs to redo.
//
// The common tooltip layout anchors the lower-left corner at the cursor,
// shifts left at the frame's right edge, and down at its top edge.
//
// THE CLAMP IS THE FRAME, NOT THE CAMERA VIEW: hoveredItemInfoAt is only
// ever true over the doll, the worn box or the pack bar, all three in the
// right-hand column and the left-hand bars — ground the camera's own view
// does not cover since 1026 narrowed it to the map viewport alone. Clamping
// against v.cam.ViewW/ViewH here reads a cursor legitimately near the
// frame's own right edge as already past it, pushing the popup well clear of
// the cursor it is meant to sit beside.
func (v *Viewer) itemPopupPresent() (*image.RGBA, image.Point, bool) {
	if !v.hasCursor {
		return nil, image.Point{}, false
	}
	lines, ok := v.hoveredItemInfoAt(v.cursorX, v.cursorY)
	if !ok {
		return nil, image.Point{}, false
	}
	return v.tooltipPicture(tooltipTarget{tooltipItem, "item", lines, v.font})
}
