package ui

import (
	"image"
	"image/color"
	"testing"
)

func markedArrow(c color.RGBA) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, 32, 88))
	for x := 0; x < 32; x++ {
		pic.SetRGBA(x, 0, c)
	}
	return pic
}

func firstLitRow(pic *image.RGBA, x int, c color.RGBA) int {
	for y := 0; y < pic.Bounds().Dy(); y++ {
		if pic.RGBAAt(x, y) == c {
			return y
		}
	}
	return -1
}

func TestPackArrowsSitBelowTheCapByTheFrameOffset(t *testing.T) {
	mark := color.RGBA{R: 200, G: 1, B: 2, A: 0xff}
	art := &BottomHUDArt{
		Pack:     solidPic(480, 90, color.RGBA{G: 50, A: 0xff}),
		PackLeft: solidPic(32, 90, color.RGBA{B: 60, A: 0xff}), PackRight: solidPic(48, 90, color.RGBA{B: 61, A: 0xff}),
		PackItem: solidPic(80, 80, color.RGBA{G: 70, A: 0xff}),
	}
	for i := range art.PackArrow {
		art.PackArrow[i] = markedArrow(mark)
	}
	bar, cols, ok := packBarRect(image.Pt(1024, 768))
	if !ok {
		t.Fatal("no pack bar")
	}
	cells := packCellRects(bar, cols)
	s := InventorySubject{Pack: make([]*image.RGBA, cols+6)}
	pic := renderPackBarArt(s, 3, cols, bar, nil, nil, -1, art)
	for name, x := range map[string]int{"left": cells[0].Min.X - 16, "right": cells[len(cells)-1].Max.X + 16} {
		if got := firstLitRow(pic, x, mark); got != 2 {
			t.Errorf("mission bar %s arrow top row at %d, want 2", name, got)
		}
	}

	shop := testArt()
	for i := range shop.PackArrow {
		shop.PackArrow[i] = markedArrow(mark)
		shop.PackCap[i%2] = flat(32, 88, color.RGBA{R: 20, A: 0xff})
	}
	for _, hover := range []bool{false, true} {
		v := ShopScreenView{Chosen: -1, Art: shop, PackBack: true, PackForward: true}
		p := ComposeShopScreen(v, shopPackLeftRect.Min.Add(image.Pt(5, 5)), hover, nil, false)
		for name, r := range map[string]image.Rectangle{"left": shopPackLeftRect, "right": shopPackRightRect} {
			got := firstLitRow(p, r.Min.X+16, mark)
			if got != r.Min.Y+2 {
				t.Errorf("shop %s arrow (hover %v) top row at %d, want %d", name, hover, got, r.Min.Y+2)
			}
		}
	}
}
