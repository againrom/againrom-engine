package ui

import (
	"againrom/pkg/render/backdrop"
	"image/color"
)

type DialogueButtonState struct {
	Hover, Pressed, Inside, Disabled bool
}

func dialoguePackedWord(c color.RGBA, layout backdrop.Layout) uint16 {
	shift, green := 11, uint16(c.G>>2)
	if layout == backdrop.RGB555 {
		shift, green = 10, uint16(c.G>>3)
	}
	return uint16(c.R>>3)<<shift | green<<5 | uint16(c.B>>3)
}

func dialoguePackedColor(word uint16, layout backdrop.Layout) color.RGBA {
	shift, mask := 11, uint16(63)
	if layout == backdrop.RGB555 {
		shift, mask = 10, 31
	}
	return color.RGBA{uint8((word >> shift & 31) * 255 / 31), uint8((word >> 5 & mask) * 255 / mask), uint8((word & 31) * 255 / 31), 255}
}
