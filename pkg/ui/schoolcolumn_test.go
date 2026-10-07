package ui

import (
	"image"
	"image/color"
	"testing"
)

func TestSchoolColumnIntermediateFrameSuppressesIconsAndMaskHits(t *testing.T) {
	art := &TownSchoolArt{Background: image.NewUniform(color.RGBA{R: 91, A: 255})}
	for i := range art.Column {
		pic := image.NewRGBA(image.Rect(0, 0, 148, 208))
		for y := 0; y < 208; y++ {
			for x := 0; x < 148; x++ {
				pic.SetRGBA(x, y, color.RGBA{R: uint8(i), A: 255})
			}
		}
		art.Column[i] = pic
	}
	mask := image.NewPaletted(image.Rect(0, 0, 92, 120), color.Palette{color.Black})
	for i := range mask.Pix {
		mask.Pix[i] = 0x37
	}
	art.Masks[0] = mask
	art.Skills[0][0][0] = image.NewUniform(color.RGBA{G: 255, A: 255})
	v := TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: 0,
		SchoolColumnSet: true, HoverCell: -1, Cells: []TownSurfaceCell{{Enabled: true, Selected: true}}}
	for _, frame := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14} {
		v.SchoolColumnFrame = frame
		pix := ComposeTownSurface(v)
		if got := pix.RGBAAt(240, 212); got != (color.RGBA{R: uint8(frame), A: 255}) {
			t.Fatalf("frame %d overpainted by an endpoint/skill: %v", frame, got)
		}
		if _, ok := TownSurfaceControlAt(v, image.Pt(240, 212)); ok {
			t.Fatalf("frame %d retained an endpoint mask", frame)
		}
	}
	for _, frame := range []int{-1, 16, 100} {
		v.SchoolColumnFrame = frame
		ComposeTownSurface(v)
		if _, ok := TownSurfaceControlAt(v, image.Pt(240, 212)); ok {
			t.Fatal("invalid column frame retained a mask")
		}
	}
}
