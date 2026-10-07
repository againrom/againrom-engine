package ui

import (
	"image"
	"image/color"
	"testing"
)

func TestMissionPackBarCornersHaveThreeStates(t *testing.T) {
	capL, capR := color.RGBA{R: 10, A: 0xff}, color.RGBA{R: 11, A: 0xff}
	frame := solidPic(480, 90, color.RGBA{G: 50, A: 0xff})
	for y := 0; y < 90; y++ {
		for x := 0; x < 32; x++ {
			frame.SetRGBA(x, y, capL)
			frame.SetRGBA(432+x, y, capR)
		}
	}
	art := &BottomHUDArt{
		Pack:     frame,
		PackLeft: solidPic(32, 90, color.RGBA{B: 60, A: 0xff}), PackRight: solidPic(48, 90, color.RGBA{B: 61, A: 0xff}),
		PackItem: solidPic(80, 80, color.RGBA{G: 70, A: 0xff}),
	}
	for i := range art.PackArrow {
		art.PackArrow[i] = solidPic(32, 88, color.RGBA{R: uint8(20 + i), A: 0xff})
	}
	bar, cols, ok := packBarRect(image.Pt(1024, 768))
	if !ok {
		t.Fatal("no pack bar")
	}
	cells := packCellRects(bar, cols)
	left := image.Pt(cells[0].Min.X-16, 40)
	right := image.Pt(cells[len(cells)-1].Max.X+16, 40)
	for _, tc := range []struct {
		name         string
		packLen      int
		scroll       int
		wantL, wantR uint8
	}{
		{"nothing to scroll", cols, 0, capL.R, capR.R},
		{"forward only", cols + 3, 0, capL.R, art.PackArrow[1].RGBAAt(0, 0).R},
		{"back only", cols + 3, 3, art.PackArrow[0].RGBAAt(0, 0).R, capR.R},
		{"both ways", cols + 6, 3, art.PackArrow[0].RGBAAt(0, 0).R, art.PackArrow[1].RGBAAt(0, 0).R},
	} {
		s := InventorySubject{Pack: make([]*image.RGBA, tc.packLen)}
		pic := renderPackBarArt(s, tc.scroll, cols, bar, nil, nil, -1, art)
		if got := pic.RGBAAt(left.X, left.Y).R; got != tc.wantL {
			t.Errorf("%s: left corner R=%d, want %d", tc.name, got, tc.wantL)
		}
		if got := pic.RGBAAt(right.X, right.Y).R; got != tc.wantR {
			t.Errorf("%s: right corner R=%d, want %d", tc.name, got, tc.wantR)
		}
	}
}
