package ui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

// sackOutlineColor is the contour colour of a highlighted sack: opaque orange,
// the owner's choice over white as closer to the game's own warm palette.
var sackOutlineColor = color.RGBA{R: 240, G: 140, B: 30, A: 255}

// sackOutlineMaxWidth bounds the contour width in sprite pixels at very low zoom.
const sackOutlineMaxWidth = 4

// sackOutlineKey identifies one cached contour texture: the sprite texture it
// outlines and the contour width in sprite pixels.
type sackOutlineKey struct {
	sprite spriteTextureKey
	width  int
}

// sackOutlineWidth is the contour width in sprite pixels that lands as about
// one screen pixel at the given zoom: 1 from zoom 1 upward, wider below it.
func sackOutlineWidth(zoom float64) int {
	if zoom <= 0 || math.IsNaN(zoom) {
		return 1
	}
	w := int(math.Ceil(1 / zoom))
	if w < 1 {
		w = 1
	}
	if w > sackOutlineMaxWidth {
		w = sackOutlineMaxWidth
	}
	return w
}

// setSackHighlight adopts this tick's level of the sack highlight key. It is a
// drawing flag only; no simulation or saved state reads it.
func (v *Viewer) setSackHighlight(held bool) { v.sackHighlight = held }

// SackHighlighted reports whether sacks are drawn with their outline now.
func (v *Viewer) SackHighlighted() bool { return v.sackHighlight }

// sackOutlinePixels returns a copy of src enlarged by width pixels on every
// side, holding sackOutlineColor on each transparent pixel within width
// (Chebyshev distance) of an opaque one, and transparent elsewhere.
func sackOutlinePixels(src *image.RGBA, width int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w+2*width, h+2*width))
	opaque := func(x, y int) bool {
		if x < 0 || y < 0 || x >= w || y >= h {
			return false
		}
		return src.Pix[src.PixOffset(b.Min.X+x, b.Min.Y+y)+3] != 0
	}
	for y := -width; y < h+width; y++ {
		for x := -width; x < w+width; x++ {
			if opaque(x, y) {
				continue
			}
			near := false
			for dy := -width; dy <= width && !near; dy++ {
				for dx := -width; dx <= width; dx++ {
					if opaque(x+dx, y+dy) {
						near = true
						break
					}
				}
			}
			if near {
				o := dst.PixOffset(x+width, y+width)
				dst.Pix[o+0], dst.Pix[o+1], dst.Pix[o+2], dst.Pix[o+3] = sackOutlineColor.R, sackOutlineColor.G, sackOutlineColor.B, sackOutlineColor.A
			}
		}
	}
	return dst
}

func (v *Viewer) sackOutlineImage(f *terrain.StaticFrame, width int) *ebiten.Image {
	key := sackOutlineKey{sprite: v.spriteKey(f), width: width}
	if img, ok := v.sackOutlineImages[key]; ok {
		return img
	}
	img := ebiten.NewImageFromImage(sackOutlinePixels(v.spritePixels(f), width))
	if v.sackOutlineImages == nil {
		v.sackOutlineImages = make(map[sackOutlineKey]*ebiten.Image)
	}
	v.sackOutlineImages[key] = img
	return img
}

// drawSackOutline paints the contour of one sack entry: the outline texture,
// enlarged by its width on every side, at the sprite's own screen position.
func (v *Viewer) drawSackOutline(target imageTarget, s staticScreenRect) {
	width := sackOutlineWidth(v.cam.Zoom)
	zoom := v.cam.Zoom
	var op ebiten.DrawImageOptions
	op.Filter = ebiten.FilterNearest
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(s.X-float64(width)*zoom, s.Y-float64(width)*zoom)
	target.DrawImage(v.sackOutlineImage(s.Frame, width), &op)
}
