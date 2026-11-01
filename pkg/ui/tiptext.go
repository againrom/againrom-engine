package ui

import (
	"image"

	"againrom/pkg/render/text"
)

// tipTextLines breaks a tip's text to width: the dialogue paragraphs, with
// discretionary hyphens where the text has them.
func tipTextLines(f *text.Font, s string, width int) []dialogueLine {
	return dialogueLinesOf(tipTextRule, f, s, width)
}

func drawTipText(dst *image.RGBA, f *text.Font, lines []dialogueLine, r image.Rectangle) {
	if f == nil {
		return
	}
	pitch := f.Height() + 2
	indent := 0
	if glyph := f.GlyphFor('@'); glyph != nil {
		indent = glyph.Advance
	}
	for i, ln := range lines {
		y := r.Min.Y + i*pitch
		if pitch <= 0 || y+f.Height() > r.Max.Y {
			break
		}
		x, width := r.Min.X, r.Dx()
		if ln.indent {
			x, width = x+indent, width-indent
		}
		drawDialogueLine(dst, f, ln, 0, x, y, width, DialogueArithmetic{}, dialogueButtonInk)
	}
}
