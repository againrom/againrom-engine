package spr16

import (
	"image"
	"image/color"

	"againrom/pkg/formats/pal"
)

// The level field's 16 steps and the byte that full coverage maps to.
const (
	levels   = 16
	fullByte = 0xff
)

// Resolve is one painted cell as a premultiplied colour: the palette entry at
// the cell's index, at coverage (level+1)/16 carried into alpha (SPR16A-031).
// An index past the palette resolves to black at the cell's own coverage, so a
// hand-built palette shorter than 256 entries cannot panic a frame walk.
func Resolve(palette []Color, p PixelA) color.RGBA {
	a := uint32((int(p.Level) + 1) * fullByte / levels)
	var e Color
	if int(p.Index) < len(palette) {
		e = palette[p.Index]
	}
	return color.RGBA{
		R: uint8(uint32(e.R) * a / fullByte),
		G: uint8(uint32(e.G) * a / fullByte),
		B: uint8(uint32(e.B) * a / fullByte),
		A: uint8(a),
	}
}

// Colors resolves every cell of f through palette; an unpainted cell stays the
// zero colour, which is distinct from a painted cell at level 0.
func (f FrameA) Colors(palette []Color) []color.RGBA {
	out := make([]color.RGBA, len(f.Pixels))
	for i, p := range f.Pixels {
		if p.Painted {
			out[i] = Resolve(palette, p)
		}
	}
	return out
}

// RGBA is Colors on an image of the frame's size.
func (f FrameA) RGBA(palette []Color) *image.RGBA {
	return pal.Picture(f.Width, f.Height, f.Colors(palette))
}
