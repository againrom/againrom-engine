package ui

import (
	"fmt"
	"image"

	"againrom/pkg/render/text"
)

// DollHover describes one paper doll for every question a pointer asks of it:
// which worn slot stands under a pixel, whether a pixel is on the doll at all,
// and how the tip that slot shows is placed. Each screen builds the
// description where it draws its doll, from the data it draws with, and
// dollSlotAt answers for all of them.
//
// A doll is a slot mask laid at At and answered inside Clip, except where a
// Hole of another control covers a pixel; or twelve slot Cells answered before
// the mask (the worn box). The builder applies any centring in At, so a
// description holds window pixels only.
type DollHover struct {
	// At is the window pixel of the mask's (0,0).
	At image.Point
	// Clip is the window area the doll answers in.
	Clip image.Rectangle
	// Holes are window areas inside Clip that belong to another control.
	Holes []image.Rectangle
	// Mask is the slot map: 1..12 per pixel, 0 where no slot is painted.
	Mask *SlotMask
	// Cells are the twelve slot rectangles in window pixels, in slot order.
	Cells [12]image.Rectangle
	// Info is each slot's item text, zero-based like Cells.
	Info [12][]string
	// Tip says how the text of a hovered slot is placed.
	Tip DollTip
	// NoTip withholds the text although the slots still answer a press.
	NoTip bool
}

// DollTip is the placement of the tip a hovered slot shows. Kind and Key name
// the tip to the hover controller, which anchors it at the cursor; Beside
// asks the screen's own composition to draw it beside the pointer too.
type DollTip struct {
	// Kind is tooltipText or tooltipItem.
	Kind uint8
	// Key identifies the tip; with WithSlot the one-based slot follows it.
	Key      string
	WithSlot bool
	// Beside draws the tip in the screen's composition at the pointer.
	Beside bool
}

// TownSquareDoll is a worn figure a room shows: its slot map with the map's
// origin at At, answered inside Clip, and each slot's worn item text.
type TownSquareDoll = DollHover

// dollSlotAt is the one hit function of every paper doll: the zero-based slot
// under window pixel p, and whether one is there. The cells answer first; the
// mask answers inside the doll's area, and a pixel the mask leaves blank, or
// outside the mask, answers no slot.
func dollSlotAt(d DollHover, p image.Point) (int, bool) {
	for i, r := range d.Cells {
		if p.In(r) {
			return i, true
		}
	}
	if !d.Covers(p) {
		return 0, false
	}
	n, ok := d.Mask.At(p.X-d.At.X, p.Y-d.At.Y)
	if !ok {
		return 0, false
	}
	return n - 1, true
}

// Covers reports whether p is on the doll's area: inside Clip and outside
// every Hole, whether or not the mask marks a slot there.
func (d DollHover) Covers(p image.Point) bool {
	if !p.In(d.Clip) {
		return false
	}
	for _, h := range d.Holes {
		if p.In(h) {
			return false
		}
	}
	return true
}

// Lines is the item text of the zero-based slot, and whether the slot has any.
func (d DollHover) Lines(slot int) ([]string, bool) {
	if slot < 0 || slot >= len(d.Info) || len(d.Info[slot]) == 0 {
		return nil, false
	}
	return d.Info[slot], true
}

// TipAt is the item text of the slot under p with its zero-based number, or
// false over no slot, over a slot with no text, or while NoTip holds.
func (d DollHover) TipAt(p image.Point) ([]string, int, bool) {
	if d.NoTip {
		return nil, 0, false
	}
	slot, ok := dollSlotAt(d, p)
	if !ok {
		return nil, 0, false
	}
	lines, ok := d.Lines(slot)
	if !ok {
		return nil, 0, false
	}
	return lines, slot, true
}

// LinesAt is the item text of the slot under p.
func (d DollHover) LinesAt(p image.Point) ([]string, bool) {
	lines, _, ok := d.TipAt(p)
	return lines, ok
}

// Target is the hover controller's target for lines shown at the zero-based
// slot.
func (t DollTip) Target(lines []string, font *text.Font, slot int) tooltipTarget {
	id := t.Key
	if t.WithSlot {
		id = fmt.Sprintf("%s/%d", id, slot+1)
	}
	return tooltipTarget{t.Kind, id, lines, font}
}

// DrawBeside paints lines beside the pointer at, for a tip that asks it.
func (t DollTip) DrawBeside(dst *image.RGBA, font *text.Font, lines []string, at image.Point) {
	if t.Beside {
		drawShopHover(dst, font, lines, at)
	}
}
