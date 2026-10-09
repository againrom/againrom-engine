package ui

import (
	"image"
	"image/color"
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
// IT DRAWS THROUGH THE SHARED HOVER BOX (composeHoverBox, MENU-128).

// popupTextColor is the hover box's label ink: the label ramp's runtime
// colour is Unknown (MENU-128), so the engine keeps the notice box's cream.
var popupTextColor = color.RGBA{R: 0xf2, G: 0xe6, B: 0xc4, A: 0xff}

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
