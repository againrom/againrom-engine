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

// spriteFocusPercent is the share of a focused tint: the one player the map
// maker picked stands out with a stronger blend than the ordinary tint, and
// its colour is lightened by spriteFocusLightPercent toward white.
const (
	spriteFocusPercent      = 75
	spriteFocusLightPercent = 25
)

// spriteTint is a colour laid over one sprite's picture. The zero value tints
// nothing. Focus asks for the brighter, stronger blend.
type spriteTint struct {
	Color color.RGBA
	On    bool
	Focus bool
}

// spriteTintKey names one tinted texture: the picture it is made from, whether
// that picture is the stone-grey one, the colour and whether it is focused.
type spriteTintKey struct {
	sprite spriteTextureKey
	stone  bool
	color  color.RGBA
	focus  bool
}

// SetSubjectTint sets the colour the sprite painter lays over the picture of
// each unit or structure the function answers for; nil, or an answer of false,
// draws the picture as it is. It is a presentation switch and changes no
// placement.
func (v *Viewer) SetSubjectTint(tint func(InspectionSubject) spriteTint) {
	v.subjectTint = tint
}

// tintOf is the tint of the sprite that belongs to subject.
func (v *Viewer) tintOf(subject InspectionSubject) spriteTint {
	if v.subjectTint == nil || subject.Kind == InspectionNone {
		return spriteTint{}
	}
	return v.subjectTint(subject)
}

// tintPixels is src with c blended over every pixel that has any coverage, at
// spriteTintPercent, or, when focus is set, with c lightened at
// spriteFocusPercent. Alpha is kept, so the sprite's holes stay holes.
func tintPixels(src *image.RGBA, c color.RGBA, focus bool) *image.RGBA {
	share := uint32(spriteTintPercent)
	chans := [3]uint32{uint32(c.R), uint32(c.G), uint32(c.B)}
	if focus {
		share = spriteFocusPercent
		for k := range chans {
			chans[k] += (255 - chans[k]) * spriteFocusLightPercent / 100
		}
	}
	dst := image.NewRGBA(src.Bounds())
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		for x := src.Rect.Min.X; x < src.Rect.Max.X; x++ {
			i := src.PixOffset(x, y)
			a := uint32(src.Pix[i+3])
			if a == 0 {
				continue
			}
			j := dst.PixOffset(x, y)
			for k, ch := range chans {
				dst.Pix[j+k] = uint8((uint32(src.Pix[i+k])*(100-share) + ch*a/255*share + 50) / 100)
			}
			dst.Pix[j+3] = src.Pix[i+3]
		}
	}
	return dst
}

// tintedPixels is the picture of f the painter draws under tint t: the lit
// picture, grey when stone, with the colour blended over it.
func (v *Viewer) tintedPixels(f *terrain.StaticFrame, stone bool, c color.RGBA, focus bool, rows ...int) *image.RGBA {
	src := v.spritePixels(f, rows...)
	return tintPixels(src, c, focus)
}

// tintedImage is f's texture under a tint, built on its first draw and cached
// under the picture and the colour.
func (v *Viewer) tintedImage(f *terrain.StaticFrame, stone bool, c color.RGBA, focus bool, rows ...int) *ebiten.Image {
	key := spriteTintKey{sprite: v.spriteKey(f, rows...), stone: stone, color: c, focus: focus}
	if img, ok := v.tintedImages[key]; ok {
		return img
	}
	img := ebiten.NewImageFromImage(v.tintedPixels(f, stone, c, focus, rows...))
	if v.tintedImages == nil {
		v.tintedImages = make(map[spriteTintKey]*ebiten.Image)
	}
	v.tintedImages[key] = img
	return img
}
