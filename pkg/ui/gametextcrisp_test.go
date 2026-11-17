package ui

import "testing"

// TestGameTextIsCrispBlocksAtWindowScaleTwo: on the name page at scale 2, each showing cell is a solid 2x2 block.
func TestGameTextIsCrispBlocksAtWindowScaleTwo(t *testing.T) {
	font := rimFont(false, 5)
	frame, calls := rimCapture(rimNamePage(false, font))
	win, _, kept := rimWindow(frame, calls, 2)
	if len(kept) == 0 {
		t.Fatal("no glyph kept on the name page")
	}
	checked := 0
	for _, c := range kept {
		if c.Glyph == nil {
			continue
		}
		for n, p := range c.Glyph.Pixels {
			if !p.Painted || (c.Mask != nil && !c.Mask[n]) {
				continue
			}
			nx, ny := c.X+n%c.Glyph.Width, c.Y+n/c.Glyph.Width
			want := c.ShownColor(p.Level)
			if frame.RGBAAt(nx, ny) != want {
				continue
			}
			checked++
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					if got := win.RGBAAt(nx*2+dx, ny*2+dy); got != want {
						t.Fatalf("cell %d,%d window pixel %d,%d = %v, want the cell colour %v", nx, ny, dx, dy, got, want)
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no showing glyph cell was checked")
	}
}
