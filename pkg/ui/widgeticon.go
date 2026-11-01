package ui

import (
	"image"
	"image/color"
	"image/draw"
)

// The one icon painter. An icon is data: a small bitmap of our own, one
// string per row and one palette letter per pixel, '.' clear. drawIcon
// paints any icon and drawIconButton paints any icon button; no icon has a
// draw function of its own.

// editorIcon is one icon's bitmap: its rows, each as wide as the first.
type editorIcon struct {
	rows []string
}

// iconInk is the icon palette: a letter's colour.
var iconInk = map[byte]color.RGBA{
	'#': editorLight,
	'o': editorPaper,
	'g': editorGold,
	'k': editorInk,
	'G': {0x6a, 0xa8, 0x4f, 255},
	'B': {0x9a, 0x6b, 0x3a, 255},
	'R': {0xd0, 0x50, 0x3c, 255},
}

// iconDisabledInk is every pixel of an icon on a disabled button.
var iconDisabledInk = color.RGBA{0x6e, 0x64, 0x55, 255}

// size is the icon's width and height in pixels.
func (ic *editorIcon) size() image.Point {
	if ic == nil || len(ic.rows) == 0 {
		return image.Point{}
	}
	return image.Pt(len(ic.rows[0]), len(ic.rows))
}

// drawIcon paints icon ic centred in box, each bitmap pixel a square of
// scale by scale screen pixels of its colour; a scale of zero or one is one
// screen pixel per bitmap pixel. With disabled set every pixel takes the
// disabled ink.
func drawIcon(dst *image.RGBA, box image.Rectangle, ic *editorIcon, disabled bool, scale int) {
	size := ic.size()
	if dst == nil || size == (image.Point{}) {
		return
	}
	scale = max(1, scale)
	at := box.Min.Add(box.Size().Sub(size.Mul(scale)).Div(2))
	for y, row := range ic.rows {
		for x := 0; x < len(row); x++ {
			c, ok := iconInk[row[x]]
			if !ok {
				continue
			}
			if disabled {
				c = iconDisabledInk
			}
			corner := at.Add(image.Pt(x, y).Mul(scale))
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					if p := corner.Add(image.Pt(dx, dy)); p.In(box) && p.In(dst.Bounds()) {
						dst.SetRGBA(p.X, p.Y, c)
					}
				}
			}
		}
	}
}

// iconButton is one icon push button: its rectangle and icon, whether it is
// on (the current tool, a switch that is on, an open drop-down), under the
// pointer, or disabled, and the scale its icon draws at (zero is one).
type iconButton struct {
	Rect                image.Rectangle
	Icon                *editorIcon
	On, Hover, Disabled bool
	Scale               int
}

// iconOnFill is the well an icon button that is on shows behind its icon.
var iconOnFill = editorInk

// drawIconButton paints button b: the one push-button painter's bevel, sunk
// over a dark well while the button is on, and the icon over it.
func drawIconButton(dst *image.RGBA, b iconButton) {
	if dst == nil {
		return
	}
	if b.On {
		draw.Draw(dst, b.Rect.Inset(1), image.NewUniform(iconOnFill), image.Point{}, draw.Src)
	}
	drawPushButton(dst, nil, pushButton{Rect: b.Rect, Pressed: b.On, Inside: b.On, Hover: b.Hover, Disabled: b.Disabled})
	drawIcon(dst, b.Rect.Inset(2), b.Icon, b.Disabled, b.Scale)
}
