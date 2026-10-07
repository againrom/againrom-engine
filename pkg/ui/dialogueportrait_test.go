package ui

import (
	"againrom/pkg/render/backdrop"
	"image"
	"image/color"
	"testing"
)

func TestDialoguePortraitSelectedSourceWindows(t *testing.T) {
	for _, tc := range []struct{ crop, want image.Rectangle }{
		{AuthoredDialogueLayout().PortraitWindow, image.Rect(36, 140, 108, 232)},
		{NoticeFaceWindow(40, 10), image.Rect(40, 134, 112, 230)},
		{NoticeFaceWindow(26, 23), image.Rect(26, 121, 98, 217)},
		{NoticeFaceWindow(0, 144), image.Rect(0, 0, 72, 96)},
	} {
		if got := dialoguePortraitSourceWindow(tc.crop, 240); got != tc.want {
			t.Errorf("source request %v want %v", got, tc.want)
		}
	}
}

func TestDialoguePortraitPackedCopyRowsKeysClipsAndGuards(t *testing.T) {
	source := []uint16{1, 0, 0x8000, 0x8001, 91, 92, 93, 94, 2, 3, 4, 5, 95, 96, 97, 98, 6, 7, 8, 9, 99, 100, 101, 102, 10, 11, 12, 13, 103, 104, 105, 106}
	for _, keyed := range []bool{false, true} {
		for _, at := range []image.Point{{0, 0}, {-1, -1}, {1, 1}, {20, 20}} {
			for _, window := range []image.Rectangle{image.Rect(0, 0, 4, 4), image.Rect(1, 1, 4, 3), image.Rect(-1, -1, 3, 3)} {
				dst := make([]uint16, 6*5+7)
				for i := range dst {
					dst[i] = 0xdead
				}
				want := append([]uint16(nil), dst...)
				clip := image.Rect(0, 1, 4, 5)
				for y := 0; y < 5; y++ {
					for x := 0; x < 4; x++ {
						row, col := y-at.Y, x-at.X
						if row < 0 || col < 0 || row >= window.Dy() || col >= window.Dx() || !image.Pt(x, y).In(clip) {
							continue
						}
						sy, sx := 3-window.Min.Y-row, window.Min.X+col
						if sx < 0 || sx >= 4 || sy < 0 || sy >= 4 {
							continue
						}
						word := source[sy*8+sx]
						if !keyed || word != 0 {
							want[y*6+x] = word
						}
					}
				}
				if !copyDialoguePortraitWords(dst, 4, 5, 6, source, 4, 4, 8, at, window, clip, keyed) {
					t.Fatal("valid copy refused")
				}
				for i, word := range dst {
					if word != want[i] {
						t.Fatalf("keyed%v at%v window%v word%d=%x want%x", keyed, at, window, i, word, want[i])
					}
				}
			}
		}
	}
}

func TestDialoguePortraitOpaqueBackgroundAndKeyedCanvasReachNotice(t *testing.T) {
	for _, layout := range []backdrop.Layout{backdrop.RGB565, backdrop.RGB555} {
		l := dialogueClaimLayout()
		l.DialogueBackdrop.Layout = layout
		l.Button = image.Rectangle{}
		l.Frame = &DialogFrame{PortraitBack: image.NewRGBA(image.Rect(0, 0, 160, 240)), Portrait: image.NewRGBA(image.Rect(0, 0, 88, 108))}
		face := image.NewRGBA(image.Rect(0, 0, 160, 240))
		// The zero and nonzero words discriminate keyed skip from opacity and blend.
		face.SetRGBA(36, 8, color.RGBA{80, 32, 16, 128})
		face.SetRGBA(37, 8, color.RGBA{1, 1, 1, 255})
		face.SetRGBA(38, 8, color.RGBA{0, 0, 8, 255})
		pic := RenderNotice(l, dialogueClaimFont(), "", face)
		row := l.Portrait.Min.Y + 9
		for _, tc := range []struct {
			x    int
			want color.RGBA
		}{{36, color.RGBA{82, 32, 16, 255}}, {37, color.RGBA{0, 0, 0, 255}}, {38, color.RGBA{0, 0, 8, 255}}} {
			if layout == backdrop.RGB555 && tc.x == 36 {
				tc.want.G = 32
			}
			if got := pic.RGBAAt(l.Portrait.Min.X+8+tc.x-36, row); got != tc.want {
				t.Errorf("layout%v canvas%x got%v want%v", layout, tc.x, got, tc.want)
			}
		}
	}
}
