package ui

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

func TestSchoolDiamondComposesOpaqueAtClaimedLiteralRectangle(t *testing.T) {
	art := &TownSchoolArt{Background: uniform(480, 480, color.RGBA{R: 61, A: 255})}
	for i := 0; i < 9; i++ {
		// Nonzero source origins catch a blit that assumes every image starts
		// at (0,0). Black corners must overwrite the room, not key through it.
		pic := image.NewRGBA(image.Rect(7, 11, 87, 87))
		for y := 11; y < 87; y++ {
			for x := 7; x < 87; x++ {
				pic.SetRGBA(x, y, color.RGBA{G: uint8(i + 1), B: uint8(x + y), A: 255})
			}
		}
		pic.SetRGBA(7, 11, color.RGBA{A: 255})
		pic.SetRGBA(86, 86, color.RGBA{A: 255})
		art.Diamond[i] = pic
	}
	for _, class := range []int{-1, 0, 1} {
		for frame := 0; frame < 9; frame++ {
			t.Run(fmt.Sprintf("class=%d/frame=%d", class, frame), func(t *testing.T) {
				v := TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: class,
					HoverCell: -1, SchoolDiamondFrame: frame}
				before := ComposeTownSurface(v)
				v.SchoolDiamondActive = true
				got := ComposeTownSurface(v)
				for y := 0; y < 480; y++ {
					for x := 0; x < 640; x++ {
						want := before.RGBAAt(x, y)
						if x >= 200 && x < 280 && y >= 60 && y < 136 {
							want = color.RGBAModel.Convert(art.Diamond[frame].At(x-200+7, y-60+11)).(color.RGBA)
						}
						if c := got.RGBAAt(x, y); c != want {
							t.Fatalf("pixel %d,%d: %v, want %v", x, y, c, want)
						}
					}
				}
			})
		}
	}
}

func TestSchoolDiamondMissingAndInvalidFramesLeaveTheRoomVisible(t *testing.T) {
	art := &TownSchoolArt{Background: uniform(480, 480, color.RGBA{R: 61, A: 255})}
	for _, frame := range []int{-1, 0, 8, 9} {
		v := TownSurfaceView{Kind: TownSurfaceSchool, SchoolArt: art, SchoolClass: -1,
			HoverCell: -1, SchoolDiamondFrame: frame, SchoolDiamondActive: true}
		got := ComposeTownSurface(v)
		if got.RGBAAt(200, 60) != (color.RGBA{R: 61, A: 255}) {
			t.Fatalf("frame %d obscured the background without art", frame)
		}
	}
	// Optional art still permits fixture and damaged-install text fallbacks.
	ComposeTownSurface(TownSurfaceView{Kind: TownSurfaceSchool, SchoolDiamondActive: true})
}
