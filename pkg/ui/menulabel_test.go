package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/menu"
)

func TestMenuLabelStaysInItsCornerAndOffEveryButton(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	plain, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	label := "Againrom 12.34.56 (abcdef1+dirty)"
	a.SetMenuLabel(label)
	labelled, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	box := menuLabelBounds(label)
	if !box.In(image.Rect(0, 0, menu.FrameW, menu.FrameH)) {
		t.Fatalf("label box %v leaves the %dx%d frame", box, menu.FrameW, menu.FrameH)
	}
	for i, r := range menu.HoverRects {
		if box.Overlaps(r) {
			t.Errorf("label box %v overlaps button %d at %v", box, i, r)
		}
	}
	changed := 0
	for y := 0; y < menu.FrameH; y++ {
		for x := 0; x < menu.FrameW; x++ {
			if plain.RGBAAt(x, y) != labelled.RGBAAt(x, y) {
				changed++
				if !image.Pt(x, y).In(box) {
					t.Fatalf("pixel %d,%d changed outside the label box %v", x, y, box)
				}
			}
		}
	}
	if changed == 0 {
		t.Fatal("the label painted nothing")
	}
	a.SetMenuLabel("")
	again, _, _ := a.HeadlessFrame()
	if string(again.Pix) != string(plain.Pix) {
		t.Fatal("clearing the label did not restore the plain menu")
	}
}
