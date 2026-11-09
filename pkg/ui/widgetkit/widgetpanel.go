package widgetkit

import (
	"image"
	"image/color"
	"image/draw"
)

// FrameSides is a set of a border's edges.
type FrameSides uint8

const (
	EdgeTop FrameSides = 1 << iota
	EdgeBottom
	EdgeLeft
	EdgeRight
)

// FlatFrame is one flat frame: a fill, a border of Width pixels (zero is one)
// on Sides (zero is all four), or both. Blend draws the fill over the picture
// instead of replacing it.
type FlatFrame struct {
	Fill, Border           color.RGBA
	Width                  int
	Sides                  FrameSides
	PaintFill, PaintBorder bool
	Blend                  bool
}

// DrawFlatFrame is the one painter of the flat frame kinds (panel, outline,
// tint and fill) over body; pkg/ui's frame builder calls it for those kinds.
func DrawFlatFrame(dst *image.RGBA, body image.Rectangle, f FlatFrame) {
	r := body.Intersect(dst.Bounds())
	if r.Empty() {
		return
	}
	if f.PaintFill {
		op := draw.Src
		if f.Blend {
			op = draw.Over
		}
		draw.Draw(dst, r, &image.Uniform{C: f.Fill}, image.Point{}, op)
	}
	if f.PaintBorder {
		w, sides := max(f.Width, 1), f.Sides
		if sides == 0 {
			sides = EdgeTop | EdgeBottom | EdgeLeft | EdgeRight
		}
		ink := &image.Uniform{C: f.Border}
		for _, e := range [...]struct {
			side FrameSides
			at   image.Rectangle
		}{
			{EdgeTop, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+w)},
			{EdgeBottom, image.Rect(r.Min.X, r.Max.Y-w, r.Max.X, r.Max.Y)},
			{EdgeLeft, image.Rect(r.Min.X, r.Min.Y, r.Min.X+w, r.Max.Y)},
			{EdgeRight, image.Rect(r.Max.X-w, r.Min.Y, r.Max.X, r.Max.Y)},
		} {
			if sides&e.side != 0 {
				draw.Draw(dst, e.at, ink, image.Point{}, draw.Src)
			}
		}
	}
}

// DrawPanel draws the flat panel over r for programs outside pkg/ui.
func DrawPanel(dst *image.RGBA, r image.Rectangle, fill, border color.RGBA) {
	if dst == nil {
		return
	}
	DrawFlatFrame(dst, r, FlatFrame{Fill: fill, Border: border, PaintFill: true, PaintBorder: true})
}
