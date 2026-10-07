package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

// HealSprite is one instance of the shipped healing sheet in a successful
// target-local Heal burst. It is deliberately a separate consumer from
// SpellBolt: picture 20's ordinary travelling arm performs no blit, while this
// semantic effect places several frames around the actor actually healed.
type HealSprite struct {
	Cell  image.Point
	Pos   image.Point
	Sheet *terrain.EffectSheet
	Frame int
	Owner uint32
}

func (v *Viewer) SetHealSprites(p []HealSprite) {
	v.healSprites = append(v.healSprites[:0], p...)
}

func (v *Viewer) HealSprites() int { return len(v.healSprites) }

// healArtPlacements applies the same fog, relief, camera, centring and clipping
// transform as ordinary projectile art, but only to the target-local semantic
// Heal consumer. Keeping the lists separate prevents picture 20 from becoming
// visible through the deliberately suppressed travelling-projectile arm.
func (v *Viewer) healArtPlacements() []effectScreenRect {
	var out []effectScreenRect
	for _, p := range v.healSprites {
		f := p.Sheet.Frame(p.Frame)
		if f == nil || f.Width <= 0 || f.Height <= 0 ||
			!v.fogGateEntity(p.Owner, p.Cell.X, p.Cell.Y) {
			continue
		}
		px, py := EffectGroundPoint(p.Pos, p.Sheet)
		r, ok := v.placeArm(p.Cell, image.Rect(px, py, px+f.Width, py+f.Height))
		if !ok {
			continue
		}
		out = append(out, effectScreenRect{screenRect: r, Frame: f})
	}
	return out
}

func (v *Viewer) drawHealArt(target imageTarget) {
	zoom := v.cam.Zoom
	for _, s := range v.healArtPlacements() {
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		op.GeoM.Scale(zoom, zoom)
		op.GeoM.Translate(s.X, s.Y)
		target.DrawImage(v.effectImage(s.Frame), &op)
	}
}
