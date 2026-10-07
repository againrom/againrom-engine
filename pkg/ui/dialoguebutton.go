package ui

import (
	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
	"image"
	"image/color"
	"slices"
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

func drawDialogueButton(dst *image.RGBA, b image.Rectangle, label string, f *text.Font, options ...NoticeLayout) {
	l := NoticeLayout{}
	if len(options) > 0 {
		l = options[0]
	}
	policy, state := l.DialogueBackdrop, l.ButtonState
	left, top, right, bottom := b.Min.X, b.Min.Y, b.Max.X-1, b.Max.Y-1
	shadow := dialogueButtonShadow
	light, dark := dialoguePackedColor(dialoguePackedWord(dialogueBevelLight, policy.Layout), policy.Layout), dialoguePackedColor(dialoguePackedWord(dialogueBevelDark, policy.Layout), policy.Layout)
	if state.Pressed && state.Inside {
		light, dark = dark, light
		shadow = 4
	}
	var calls []text.DrawCall
	if label != "" && f != nil {
		label = gameMenuLabelText(label)
		x := left + (right-left)/2 + 1 - f.Advance(label)/2
		y := top + (bottom-top)/2 - dialogueLabelRaise
		ink := dialogueButtonInk
		if state.Hover {
			ink = color.RGBA{150, 90, 0, 255}
		}
		flat := dialoguePackedColor(dialoguePackedWord(messageShadowColor, policy.Layout), policy.Layout)
		before := slices.Clone(dst.Pix)
		calls = text.Record(func() { f.DrawFlat(dst, label, x+shadow, y+shadow, flat); f.Draw(dst, label, x, y, ink) })
		copy(dst.Pix, before)
		for i := range calls {
			c := &calls[i]
			if c.Glyph == nil || c.Glyph.Width <= 0 {
				continue
			}
			c.Under = make([]color.RGBA, len(c.Glyph.Pixels))
			c.RasterColors = make([]color.RGBA, len(c.Glyph.Pixels))
			for n, p := range c.Glyph.Pixels {
				at := image.Pt(c.X+n%c.Glyph.Width, c.Y+n/c.Glyph.Width)
				shade := c.CellColor(p.Level)
				if !c.Flat {
					shade = dialoguePackedColor(dialoguePackedWord(shade, policy.Layout), policy.Layout)
				}
				c.RasterColors[n] = shade
				if p.Painted && at.In(dst.Bounds()) {
					c.Under[n] = dst.RGBAAt(at.X, at.Y)
					dst.SetRGBA(at.X, at.Y, shade)
				}
			}
		}
	}
	hline := func(y, x0, x1 int, c color.RGBA) {
		for x := x0; x <= x1; x++ {
			dst.SetRGBA(x, y, c)
		}
	}
	vline := func(x, y0, y1 int, c color.RGBA) {
		for y := y0; y <= y1; y++ {
			dst.SetRGBA(x, y, c)
		}
	}
	vline(right, top+2, bottom-2, dark)
	vline(right-1, top+1, bottom-1, dark)
	hline(bottom, left+2, right-2, dark)
	hline(bottom-1, left+1, right-1, dark)
	dst.SetRGBA(right-2, bottom-2, dark)
	hline(top, left+2, right-2, light)
	vline(left, top+2, bottom-2, light)
	dst.SetRGBA(left+1, top+1, light)
	if state.Disabled {
		lookup, err := backdrop.New(policy.Layout, policy.Mode)
		if err == nil {
			lookup.Apply(dst, b, dst.Bounds(), 1)
			remapCaptured(calls, b, lookup, 1)
		}
	}
	text.Append(calls, 0, 0)
}
