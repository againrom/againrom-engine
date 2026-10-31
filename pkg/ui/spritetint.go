package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

// spriteTintPercent is how much of the tint colour a tinted sprite's opaque
// pixel takes; the rest is the picture's own colour, so the picture stays
// readable under the mask.
const spriteTintPercent = 45

// spriteTint is a colour laid over one sprite's picture. The zero value tints
// nothing.
type spriteTint struct {
	Color color.RGBA
	On    bool
}

// spriteTintKey names one tinted texture: the picture it is made from, whether
// that picture is the stone-grey one, and the colour.
type spriteTintKey struct {
	sprite spriteTextureKey
	stone  bool
	color  color.RGBA
}

// SetSubjectTint sets the colour the sprite painter lays over the picture of
// each unit or structure the function answers for; nil, or an answer of false,
// draws the picture as it is. It is a presentation switch and changes no
// placement.
func (v *Viewer) SetSubjectTint(tint func(InspectionSubject) (color.RGBA, bool)) {
	v.subjectTint = tint
}

// tintOf is the tint of the sprite that belongs to subject.
func (v *Viewer) tintOf(subject InspectionSubject) spriteTint {
	if v.subjectTint == nil || subject.Kind == InspectionNone {
		return spriteTint{}
	}
	c, ok := v.subjectTint(subject)
	return spriteTint{Color: c, On: ok}
}

// tintPixels is src with c blended over every pixel that has any coverage, at
// spriteTintPercent. Alpha is kept, so the sprite's holes stay holes.
func tintPixels(src *image.RGBA, c color.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		for x := src.Rect.Min.X; x < src.Rect.Max.X; x++ {
			i := src.PixOffset(x, y)
			a := uint32(src.Pix[i+3])
			if a == 0 {
				continue
			}
			j := dst.PixOffset(x, y)
			for k, ch := range [3]uint8{c.R, c.G, c.B} {
				dst.Pix[j+k] = uint8((uint32(src.Pix[i+k])*(100-spriteTintPercent) + uint32(ch)*a/255*spriteTintPercent + 50) / 100)
			}
			dst.Pix[j+3] = src.Pix[i+3]
		}
	}
	return dst
}

// tintedPixels is the picture of f the painter draws under tint t: the lit
// picture, grey when stone, with the colour blended over it.
func (v *Viewer) tintedPixels(f *terrain.StaticFrame, stone bool, c color.RGBA) *image.RGBA {
	src := v.spritePixels(f)
	if stone {
		src = grayscaleRGBA(src)
	}
	return tintPixels(src, c)
}

// tintedImage is f's texture under a tint, built on its first draw and cached
// under the picture and the colour.
func (v *Viewer) tintedImage(f *terrain.StaticFrame, stone bool, c color.RGBA) *ebiten.Image {
	key := spriteTintKey{sprite: v.spriteKey(f), stone: stone, color: c}
	if img, ok := v.tintedImages[key]; ok {
		return img
	}
	img := ebiten.NewImageFromImage(v.tintedPixels(f, stone, c))
	if v.tintedImages == nil {
		v.tintedImages = make(map[spriteTintKey]*ebiten.Image)
	}
	v.tintedImages[key] = img
	return img
}
