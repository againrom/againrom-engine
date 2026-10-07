package backdrop

import (
	"bytes"
	"encoding/binary"
	"image"
	"testing"
)

func independentWord(p int, greenBits int, reduced bool) uint16 {
	blue := p % 32
	if reduced {
		blue = blue/8*8 + 4
	}
	green := p / 32 % (1 << greenBits)
	red := p / (32 * (1 << greenBits)) % 32
	return uint16((red*13/16)*(32*(1<<greenBits)) + (green*13/16)*32 + blue*13/16)
}

func TestPackedBackdropWholePopulationsAndLossControls(t *testing.T) {
	for _, layout := range []Layout{RGB565, RGB555} {
		for _, mode := range []Mode{Full, Reduced} {
			l, err := New(layout, mode)
			if err != nil {
				t.Fatal(err)
			}
			bits, n := 6, 65536
			if layout == RGB555 {
				bits, n = 5, 32768
			}
			texture := l.Texture()
			losses := [4]int{}
			for p := 0; p < n; p++ {
				want := independentWord(p, bits, mode == Reduced)
				if got := l.Word(uint16(p)); got != want {
					t.Fatalf("%d/%d word %d: %d want %d", layout, mode, p, got, want)
				}
				packed := int(want)
				c := texture.RGBAAt(p%256, p/256)
				if int(c.R)/8*(32*(1<<bits))+int(c.G)/(256/(1<<bits))*32+int(c.B)/8 != packed {
					t.Fatalf("submission texture %d/%d at %d does not round-trip %d", layout, mode, p, want)
				}
				if uint16(p*13/16) != want {
					losses[0]++
				}
				if independentWord(p, bits, false) != want {
					losses[1]++
				}
				if want != uint16(p) {
					losses[2]++
				}
				if l.Word(want) != want {
					losses[3]++
				}
			}
			if losses[0] == 0 || losses[2] == 0 || losses[3] == 0 || mode == Reduced && losses[1] == 0 {
				t.Fatal("insensitive controls", losses)
			}
			t.Logf("layout=%d mode=%d inputs=%d packed-multiply/full-only/omitted/second-show differences=%v", layout, mode, n, losses)
		}
	}
}

func TestPackedBackdropHalfOpenPitchAndGuards(t *testing.T) {
	requests := []image.Rectangle{image.Rect(0, 0, 9, 6), image.Rect(3, 2, 6, 4), image.Rect(-2, -2, 4, 3), image.Rect(4, 0, 4, 6), {Min: image.Pt(0, 5), Max: image.Pt(9, 2)}, image.Rect(20, 20, 22, 22)}
	clip := image.Rect(2, 1, 7, 5)
	for _, layout := range []Layout{RGB565, RGB555} {
		for _, mode := range []Mode{Full, Reduced} {
			l, _ := New(layout, mode)
			for _, r := range requests {
				buf := bytes.Repeat([]byte{0xad}, 26*6+64)
				for y := 0; y < 6; y++ {
					for x := 0; x < 9; x++ {
						binary.LittleEndian.PutUint16(buf[y*26+x*2:], uint16((x+9*y)*491%32768))
					}
				}
				want := bytes.Clone(buf)
				for y := 0; y < 6; y++ {
					for x := 0; x < 9; x++ {
						if image.Pt(x, y).In(r) && image.Pt(x, y).In(clip) {
							at := y*26 + x*2
							p := int(binary.LittleEndian.Uint16(want[at:]))
							bits := 6
							if layout == RGB555 {
								bits = 5
							}
							binary.LittleEndian.PutUint16(want[at:], independentWord(p, bits, mode == Reduced))
						}
					}
				}
				if err := l.Remap(buf, 9, 6, 26, r, clip); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(buf, want) {
					t.Fatalf("%d/%d request %v changed clip/pitch/guard", layout, mode, r)
				}
			}
		}
	}
	l, _ := New(RGB565, Full)
	for _, geom := range [][3]int{{9, 6, 16}, {9, 7, 26}, {-1, 6, 26}, {9, 6, -1}} {
		if l.Remap(make([]byte, 156), geom[0], geom[1], geom[2], clip, clip) == nil {
			t.Fatal("unbounded geometry accepted", geom)
		}
	}
}
