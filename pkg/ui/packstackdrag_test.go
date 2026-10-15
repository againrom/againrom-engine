package ui

import (
	"image"
	"image/color"
	"testing"
)

func TestPackStackDragLatchesQuantityAndPreviewsRemainder(t *testing.T) {
	for _, whole := range []bool{false, true} {
		a := inventoryTestApp(t)
		v := a.flow.viewer
		layoutViewport(v, MenuWindowW, MenuWindowH)
		v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}})
		v.sel = selection{5}
		pic := solidPic(8, 8, color.RGBA{R: 255, A: 255})
		s := packOf(5, pic, pic)
		s.PackCount = []uint32{3, 2}
		v.SetInventorySubject(s)
		sx, sy := packCellCenter(t, v, 0)
		dx, dy := packCellCenter(t, v, 1)
		prefix := ""
		if whole {
			prefix = "shift-"
		}
		for _, edge := range []string{"press", "move"} {
			x, y := sx, sy
			if edge == "move" {
				x, y = dx, dy
			}
			if err := a.HeadlessPointer(prefix+edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
		want := uint32(2)
		if whole {
			want = 0
		}
		if got := v.previewSubject().PackCount[0]; got != want || v.invSubject.PackCount[0] != 3 {
			t.Fatalf("whole=%v preview=%d stored=%d", whole, got, v.invSubject.PackCount[0])
		}
		if err := a.HeadlessPointer("release", dx, dy); err != nil {
			t.Fatal(err)
		}
		source, target, all, ok := v.TakeInventoryPackMove()
		if !ok || source != 0 || target != 1 || all != whole {
			t.Fatalf("request=%d,%d,%v,%v", source, target, all, ok)
		}
		if _, _, _, ok := v.TakeInventoryPackMove(); ok {
			t.Fatal("request was not drained")
		}
	}
}
