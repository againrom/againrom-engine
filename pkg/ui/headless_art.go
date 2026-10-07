package ui

import (
	"fmt"
	"image"

	"againrom/pkg/render/terrain"
	"github.com/hajimehoshi/ebiten/v2"
)

// HeadlessArtDraw records an actual drawArt submission and its uploaded source
// pixels. It is a draw-call witness, not GPU readback or a complete map frame.
type HeadlessArtDraw struct {
	Kind       string
	Frame      *terrain.StaticFrame
	Effect     *terrain.EffectFrame
	Pixels     *image.RGBA
	Geometry   ebiten.GeoM
	ColorScale ebiten.ColorScale
	Filter     ebiten.Filter
}

type headlessArtTarget struct {
	v     *Viewer
	draws []HeadlessArtDraw
	err   error
}

func (r *headlessArtTarget) DrawImage(img *ebiten.Image, op *ebiten.DrawImageOptions) {
	d := HeadlessArtDraw{Geometry: op.GeoM, ColorScale: op.ColorScale, Filter: op.Filter}
	for f, tex := range r.v.effectImages {
		if tex == img {
			d.Kind, d.Effect, d.Pixels = "effect", f, f.RGBA()
		}
	}
	for f, tex := range r.v.shadowMasks {
		if tex == img {
			d.Kind, d.Frame, d.Pixels = "shadow", f, terrain.ShadowMask(f)
		}
	}
	for key, tex := range r.v.staticImages {
		if tex == img {
			d.Kind, d.Frame, d.Pixels = "sprite", key.frame, headlessSpritePixels(key)
		}
	}
	for key, tex := range r.v.stoneImages {
		if tex == img {
			d.Kind, d.Frame, d.Pixels = "stone", key.frame, grayscaleRGBA(headlessSpritePixels(key))
		}
	}
	if d.Pixels == nil {
		r.err = fmt.Errorf("headless art: unidentified production texture")
	}
	r.draws = append(r.draws, d)
}

func headlessSpritePixels(key spriteTextureKey) *image.RGBA {
	if key.row == spriteUnlit {
		return key.frame.RGBA()
	}
	return key.frame.RGBALit(key.tint, key.row)
}

// HeadlessArtDraws runs the production art painter against a recorder. Unknown
// textures fail explicitly; terrain, overlay, shroud and HUD are not composed.
func (v *Viewer) HeadlessArtDraws() ([]HeadlessArtDraw, error) {
	r := headlessArtTarget{v: v}
	v.drawArt(&r)
	return r.draws, r.err
}
