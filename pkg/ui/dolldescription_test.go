package ui

import (
	"image"
	"testing"
)

func descriptionMask() *SlotMask {
	m := &SlotMask{W: 4, H: 2, Slot: []uint8{1, 1, 2, 0, 3, 0, 0, 12}}
	return m
}

// A doll answers its mask laid at At inside Clip, minus the Holes, with the
// zero-based slot; the cells answer first and need no mask.
func TestDollSlotAtAnswersTheDescription(t *testing.T) {
	d := DollHover{At: image.Pt(10, 20), Clip: image.Rect(10, 20, 14, 22),
		Holes: []image.Rectangle{image.Rect(12, 20, 13, 21)}, Mask: descriptionMask()}
	for _, tc := range []struct {
		p    image.Point
		slot int
		ok   bool
	}{
		{image.Pt(10, 20), 0, true},
		{image.Pt(11, 20), 0, true},
		{image.Pt(12, 20), 0, false}, // hole over slot 2
		{image.Pt(13, 20), 0, false}, // blank pixel
		{image.Pt(10, 21), 2, true},
		{image.Pt(11, 21), 0, false},
		{image.Pt(13, 21), 11, true},
		{image.Pt(14, 21), 0, false}, // outside the clip
		{image.Pt(9, 20), 0, false},
	} {
		if n, ok := dollSlotAt(d, tc.p); ok != tc.ok || (ok && n != tc.slot) {
			t.Errorf("dollSlotAt(%v) = (%d,%v), want (%d,%v)", tc.p, n, ok, tc.slot, tc.ok)
		}
	}
	if !d.Covers(image.Pt(13, 20)) || d.Covers(image.Pt(12, 20)) || d.Covers(image.Pt(14, 20)) {
		t.Error("Covers must hold the blank pixel, and refuse a hole and the outside")
	}
	if _, ok := dollSlotAt(DollHover{Clip: d.Clip, At: d.At}, image.Pt(10, 20)); ok {
		t.Error("a description with no mask answered a slot")
	}
	if _, ok := dollSlotAt(DollHover{}, image.Pt(0, 0)); ok {
		t.Error("the empty description answered a slot")
	}

	var cells DollHover
	cells.Cells[4] = image.Rect(0, 0, 5, 5)
	cells.Cells[5] = image.Rect(5, 0, 10, 5)
	cells.Mask, cells.Clip = descriptionMask(), image.Rect(0, 0, 10, 5)
	if n, ok := dollSlotAt(cells, image.Pt(6, 1)); !ok || n != 5 {
		t.Errorf("dollSlotAt over the sixth cell = (%d,%v), want (5,true)", n, ok)
	}
	if n, ok := dollSlotAt(cells, image.Pt(0, 0)); !ok || n != 4 {
		t.Errorf("the cell must answer before the mask: got (%d,%v), want (4,true)", n, ok)
	}
}

// The tip is the slot's lines, given only over a slot that has some while the
// doll may show a tip, and the target carries the placement's identity.
func TestDollTipPlacement(t *testing.T) {
	d := DollHover{At: image.Pt(0, 0), Clip: image.Rect(0, 0, 4, 2), Mask: descriptionMask(),
		Tip: DollTip{Kind: tooltipItem, Key: "k", WithSlot: true}}
	d.Info[0] = []string{"a", "b"}
	d.Info[2] = []string{"c"}

	lines, slot, ok := d.TipAt(image.Pt(1, 0))
	if !ok || slot != 0 || len(lines) != 2 {
		t.Fatalf("TipAt over slot 1 = (%q,%d,%v)", lines, slot, ok)
	}
	if got := d.Tip.Target(lines, nil, slot); got.kind != tooltipItem || got.id != "k/1" || len(got.lines) != 2 {
		t.Errorf("Target = %+v, want the item tip k/1", got)
	}
	if _, _, ok := d.TipAt(image.Pt(2, 0)); ok {
		t.Error("slot 2 has no lines yet gave a tip")
	}
	if _, _, ok := d.TipAt(image.Pt(3, 0)); ok {
		t.Error("a blank pixel gave a tip")
	}
	if got, ok := d.LinesAt(image.Pt(0, 1)); !ok || len(got) != 1 || got[0] != "c" {
		t.Errorf("LinesAt over slot 3 = (%q,%v)", got, ok)
	}
	d.NoTip = true
	if _, _, ok := d.TipAt(image.Pt(1, 0)); ok {
		t.Error("NoTip must withhold the tip")
	}
	if _, ok := dollSlotAt(d, image.Pt(1, 0)); !ok {
		t.Error("NoTip must leave the press")
	}
	plain := DollTip{Kind: tooltipText, Key: "shop-item/1"}
	if got := plain.Target([]string{"x"}, nil, 7); got.kind != tooltipText || got.id != "shop-item/1" {
		t.Errorf("a tip without the slot must keep its key, got %+v", got)
	}
}

// Only a tip that asks to be drawn beside the pointer is drawn there.
func TestDollTipDrawsBesideOnlyWhenAsked(t *testing.T) {
	font := messageFont()
	draw := func(tip DollTip) bool {
		dst := image.NewRGBA(image.Rect(0, 0, 200, 100))
		tip.DrawBeside(dst, font, []string{"Sword"}, image.Pt(20, 20))
		for _, b := range dst.Pix {
			if b != 0 {
				return true
			}
		}
		return false
	}
	if !draw(DollTip{Beside: true}) {
		t.Error("a tip that asks to be drawn beside the pointer drew nothing")
	}
	if draw(DollTip{}) {
		t.Error("a tip that did not ask drew beside the pointer")
	}
}
