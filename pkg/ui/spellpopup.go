package ui

import (
	"image"
	"image/color"
)

// The spellbook's two additions: the popup that states what a spell does
// before the player spends the mana on it, and the rotating dashed border
// that says a spell is on autocast.
//
// THE POPUP IS THE ITEM POPUP'S OWN BOX (itempopup.go), handed different lines.
// It is a HOVER and issues nothing: it reads v.cursorX/v.cursorY/v.hasCursor,
// the same per-frame cursor the item popup and the cell readout already read,
// and touches no click state at all — the spellbook's own press handling
// (command.go) is untouched by this file.
//
// THIS PACKAGE STILL HOLDS NO SPELL KNOWLEDGE. The lines arrive already
// composed on SpellEntry.Info, beside the id, the name and the icon; nothing
// here computes a cost, a range, a band or a school, and nothing here could
// — the world's own table is on the far side of the seam.

// spellPopupPresent is the popup's picture for this frame and where its
// top-left corner goes, or false for a frame that draws none: no cursor
// observed yet, a cursor over no book cell, a cell past the end of the book, an
// entry carrying no lines, or a viewer holding no font.
//
// IT IS itemPopupPresent's OWN BODY over a different lookup, including the
// clamp that keeps the box off the view's right and bottom edges — the one
// placement rule a box pinned to a moving cursor needs.
func (v *Viewer) spellPopupPresent() (*image.RGBA, image.Point, bool) {
	if !v.hasCursor {
		return nil, image.Point{}, false
	}
	idx, ok := v.spellbookEntryAt(v.cursorX, v.cursorY)
	if !ok || idx < 0 || idx >= len(v.spellbook) {
		return nil, image.Point{}, false
	}
	lines := v.spellbook[idx].Info
	if len(lines) == 0 {
		return nil, image.Point{}, false
	}
	return v.tooltipPicture(tooltipTarget{tooltipSpell, "spell", lines, v.font})
}

// The rotating dashed border's own numbers, all AUTHORED. The owner's design
// is "a dashed border that rotates"; its dash length, its gap, its colour
// and how fast it turns are ours.
const (
	// autocastDashOn and autocastDashOff are the dash and the gap, in pixels
	// along the cell's own perimeter. Three on and three off reads as a dash
	// rather than as a dotted line at the 36-pixel cell this bar draws.
	autocastDashOn  = 3
	autocastDashOff = 3

	// autocastDashStep is how many pixels the pattern travels per ambient
	// animation step. One pixel a step turns the border at the speed the rest
	// of the ambient picture animates at, which is what makes the mark stop
	// when the player pauses and slow when he slows the game down.
	autocastDashStep = 1
)

// autocastDashColor is the ink the dashes are drawn in — the same yellow the
// bar already picks a SELECTED cell out with, so a player reads one palette on
// this box and not two. A cell that is both selected and autocasting therefore
// carries a solid border and a travelling dashed one in the same colour, which
// is the intended reading: they are two statements about the same cell.
var autocastDashColor = spellbookSelected

// drawAutocastBorder paints the rotating dashed outline around one cell.
// phase is the ambient animation count; the pattern's origin is
// phase*autocastDashStep pixels along the perimeter, so two frames one
// ambient step apart draw the dashes in different places and a frame with
// the animation stopped draws them in the same place twice.
//
// IT WALKS THE PERIMETER AS ONE CLOSED PATH — top edge left to right, right
// edge top to bottom, bottom edge right to left, left edge bottom to top — so
// the dashes travel CLOCKWISE around the cell rather than each edge running its
// own independent pattern, which would read as four blinking sides.
//
// A box with no interior draws nothing, which is what keeps a bar squeezed to
// nothing by a small window from panicking rather than simply showing no mark.
func drawAutocastBorder(dst *image.RGBA, box image.Rectangle, phase int, c color.RGBA) {
	if dst == nil || box.Dx() < 2 || box.Dy() < 2 {
		return
	}
	period := autocastDashOn + autocastDashOff
	w, h := box.Dx(), box.Dy()
	perimeter := 2*w + 2*h - 4
	// The offset is reduced into one period so a long-running counter cannot
	// overflow the walk below, and it is subtracted so the pattern travels
	// FORWARD along the path as the counter rises.
	off := (phase * autocastDashStep) % period
	for i := 0; i < perimeter; i++ {
		if ((i+period-off)%period)>>0 >= autocastDashOn {
			continue
		}
		var x, y int
		switch {
		case i < w:
			x, y = box.Min.X+i, box.Min.Y
		case i < w+h-1:
			x, y = box.Max.X-1, box.Min.Y+(i-w+1)
		case i < 2*w+h-2:
			x, y = box.Max.X-1-(i-(w+h-1)), box.Max.Y-1
		default:
			x, y = box.Min.X, box.Max.Y-1-(i-(2*w+h-2))
		}
		if image.Pt(x, y).In(dst.Bounds()) {
			dst.SetRGBA(x, y, c)
		}
	}
}
