package ui

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// popupDim is the one builder of a popup's backdrop dim: the rectangle to
// darken and the colour to darken it with, or false for a frame that draws no
// dim. The rectangle is the whole presented frame, so no part of the screen
// the player sees keeps its undimmed value behind a popup, whichever screen
// and whichever popup. The colour is data the caller reads from the popup
// kind; a fully transparent colour or an empty frame draws nothing.
func popupDim(frame image.Rectangle, c color.RGBA) (popupDimPlan, bool) {
	if c.A == 0 || frame.Empty() {
		return popupDimPlan{}, false
	}
	return popupDimPlan{rect: frame, colour: c}, true
}

// popupDimPlan is one popup dim: the rectangle to darken and the colour to
// darken it with. Both painters run it.
type popupDimPlan struct {
	rect   image.Rectangle
	colour color.RGBA
}

// paintPopupDim draws a popupDim result on an ebiten surface and records it on
// the surface's pixel log.
func paintPopupDim(dst *ebiten.Image, log *pixelLog, p popupDimPlan) {
	r := p.rect
	vector.DrawFilledRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), p.colour, false)
	log.overSolid(r, p.colour)
}

// paintPopupDimRGBA draws a popupDim result on a CPU picture, for the headless
// frames that compose without a graphics context.
func paintPopupDimRGBA(dst *image.RGBA, p popupDimPlan) {
	draw.Draw(dst, p.rect, &image.Uniform{C: p.colour}, image.Point{}, draw.Over)
}
