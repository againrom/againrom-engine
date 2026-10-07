package main

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/hajimehoshi/bitmapfont/v4"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var (
	colBackground = color.RGBA{24, 26, 33, 255}
	colPanel      = color.RGBA{38, 42, 54, 255}
	colButton     = color.RGBA{58, 68, 96, 255}
	colBorder     = color.RGBA{86, 94, 118, 255}
	colFocus      = color.RGBA{120, 170, 255, 255}
	colText       = color.RGBA{222, 224, 230, 255}
	colDim        = color.RGBA{138, 144, 158, 255}
	colGood       = color.RGBA{128, 204, 138, 255}
	colBad        = color.RGBA{236, 118, 106, 255}
	colAccent     = color.RGBA{240, 200, 110, 255}
)

func (t tone) color() color.RGBA {
	switch t {
	case toneDim:
		return colDim
	case toneGood:
		return colGood
	case toneBad:
		return colBad
	}
	return colText
}

func textWidth(s string) int { return font.MeasureString(bitmapfont.Face, s).Ceil() }

// fitHead cuts s to maxW pixels, ending with "..." when it was cut.
func fitHead(s string, maxW int) string {
	if textWidth(s) <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && textWidth(string(r)+"...") > maxW {
		r = r[:len(r)-1]
	}
	return string(r) + "..."
}

// fitTail keeps the end of s within maxW pixels, starting with "..." when it
// was cut: the end of a path is the part that tells folders apart.
func fitTail(s string, maxW int) string {
	if textWidth(s) <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && textWidth("..."+string(r)) > maxW {
		r = r[1:]
	}
	return "..." + string(r)
}

func drawText(dst *image.RGBA, s string, x, y int, c color.RGBA) {
	d := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(c),
		Face: bitmapfont.Face,
		Dot:  fixed.P(x, y+bitmapfont.Face.Metrics().Ascent.Ceil()),
	}
	d.DrawString(s)
}

func strokeRect(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	for _, e := range []image.Rectangle{
		image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1),
		image.Rect(r.Min.X, r.Max.Y-1, r.Max.X, r.Max.Y),
		image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y),
		image.Rect(r.Max.X-1, r.Min.Y, r.Max.X, r.Max.Y),
	} {
		draw.Draw(dst, e, image.NewUniform(c), image.Point{}, draw.Src)
	}
}

func fill(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	draw.Draw(dst, r, image.NewUniform(c), image.Point{}, draw.Src)
}

// render draws the whole window.
func (a *app) render() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, winW, winH))
	fill(img, img.Bounds(), colBackground)
	fill(img, image.Rect(0, 440+panelH, winW, winH), colPanel)
	fill(img, image.Rect(10, 488+panelH, winW-10, winH-6), colBackground)
	strokeRect(img, image.Rect(10, 488+panelH, winW-10, winH-6), colBorder)
	for _, it := range a.layout() {
		r := it.rect
		switch it.kind {
		case iText:
			drawText(img, fitHead(it.text, r.Dx()), r.Min.X, r.Min.Y, it.tone.color())
		case iHeading:
			drawText(img, it.text, r.Min.X, r.Min.Y, colAccent)
		case iButton:
			fill(img, r, colButton)
			strokeRect(img, r, colBorder)
			s := fitHead(it.text, r.Dx()-8)
			drawText(img, s, r.Min.X+(r.Dx()-textWidth(s))/2, r.Min.Y+(r.Dy()-16)/2, colText)
		case iToggle:
			fill(img, r, colButton)
			strokeRect(img, r, colBorder)
			mark := "[ ] "
			if it.checked {
				mark = "[x] "
			}
			drawText(img, mark+it.text, r.Min.X+6, r.Min.Y+3, colText)
		case iField:
			fill(img, r, colPanel)
			if it.focused {
				strokeRect(img, r, colFocus)
			} else {
				strokeRect(img, r, colBorder)
			}
			s := it.text
			if it.focused {
				s += "|"
			}
			drawText(img, fitTail(s, r.Dx()-10), r.Min.X+5, r.Min.Y+3, colText)
		case iBaseRow, iModRow:
			mark := "( ) "
			if it.kind == iModRow {
				mark = "[ ] "
			}
			if it.checked {
				mark = map[itemKind]string{iBaseRow: "(*) ", iModRow: "[x] "}[it.kind]
			}
			drawText(img, fitHead(mark+it.text, r.Dx()-4), r.Min.X+2, r.Min.Y+1, it.tone.color())
		}
	}
	return img
}
