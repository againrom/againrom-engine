package ui

import (
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/text"
)

// The hover box's requested colours and line pitch (MENU-128).
var (
	hoverFill  = color.RGBA{36, 44, 39, 255}
	hoverLight = color.RGBA{160, 120, 50, 255}
	hoverDark  = color.RGBA{80, 60, 24, 255}
)

const (
	hoverPitch    = 14
	hoverExtraW   = 11
	hoverExtraH   = 5
	hoverTextX    = 5
	hoverTextY    = 4
	hoverBallSize = 4
)

// hoverBoxSize is the box for lines: the widest line plus 11 by 14 per line
// plus 5.
func hoverBoxSize(lines []string, f *text.Font) image.Point {
	width := 0
	for _, ln := range lines {
		w, _ := f.Measure(ln)
		width = max(width, w, f.Advance(ln))
	}
	return image.Pt(width+hoverExtraW, hoverPitch*len(lines)+hoverExtraH)
}

// composeHoverBox is the one hover box painter (MENU-128): the green fill,
// the two-colour gold bevel, Ball.bmp at the four corners and each line at
// (L+5,T+4+14i). The picture's right edge R is its last column and its
// bottom B one past its last row, so every side keeps a one-pixel margin
// outside the bevel. Without a ball picture the corners stay unpainted.
func composeHoverBox(lines []string, f *text.Font, ball image.Image) *image.RGBA {
	if f == nil || f.Height() <= 0 || len(lines) == 0 {
		return nil
	}
	size := hoverBoxSize(lines, f)
	if size.X <= hoverExtraW {
		return nil
	}
	img := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	recordWidget(widgetHoverBox, img.Bounds(), lines)
	r, b := size.X-1, size.Y
	draw.Draw(img, image.Rect(1, 1, r-1, b-2), &image.Uniform{C: hoverFill}, image.Point{}, draw.Src)
	hline := func(y int, c color.RGBA) {
		for x := 3; x <= r-3; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	vline := func(x int, c color.RGBA) {
		for y := 3; y <= b-4; y++ {
			img.SetRGBA(x, y, c)
		}
	}
	hline(1, hoverLight)
	hline(2, hoverDark)
	hline(b-3, hoverLight)
	hline(b-2, hoverDark)
	vline(1, hoverLight)
	vline(2, hoverDark)
	vline(r-2, hoverLight)
	vline(r-1, hoverDark)
	if ball != nil {
		for _, at := range []image.Point{{0, 0}, {r - 3, 0}, {0, b - hoverBallSize}, {r - 3, b - hoverBallSize}} {
			src := ball.Bounds()
			dst := image.Rect(at.X, at.Y, at.X+hoverBallSize, at.Y+hoverBallSize).Intersect(img.Bounds())
			draw.Draw(img, dst, ball, src.Min, draw.Over)
		}
	}
	for i, ln := range lines {
		f.Draw(img, ln, hoverTextX, hoverTextY+i*hoverPitch, popupTextColor)
	}
	return img
}
