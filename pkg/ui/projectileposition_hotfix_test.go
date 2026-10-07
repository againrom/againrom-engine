package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

func TestProjectileCoordinateSpacesKeepSpriteAndDeformationTogether(t *testing.T) {
	v := overlayViewer(t, 8, 8, 256, 256)
	v.SetFlat(true)
	v.cam.X, v.cam.Y, v.cam.Zoom = 0, 0, 1
	sheet := &terrain.EffectSheet{CenterX: 2, CenterY: 3, Frames: []*terrain.EffectFrame{{Width: 8, Height: 8}}}
	for _, tc := range []struct {
		name      string
		absolute  bool
		pos, want image.Point
	}{
		{"cell-directed", false, image.Pt(256, 512), image.Pt(48, 80)},
		{"carried-fine", true, image.Pt(384, 640), image.Pt(48, 80)},
		{"low-fine", true, image.Pt(43, 77), image.Pt(5, 9)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := SpellBolt{Pos: tc.pos, AbsolutePosition: tc.absolute, Sheet: sheet, Cell: image.Pt(1, 2), To: image.Pt(1, 2)}
			v.SetSpellBolts([]SpellBolt{b})
			p := v.spellArtPlacements()
			if len(p) != 1 || int(p[0].X)+2 != tc.want.X || int(p[0].Y)+3 != tc.want.Y {
				t.Fatal("sprite geometry", p, tc.want)
			}
			b.Effect, b.Sheet = SpellBackgroundDeformation, nil
			d, ok := v.deformationPlacement(b)
			if !ok || d.Center != tc.want {
				t.Fatal("deformation geometry", d.Center, tc.want)
			}
		})
	}
}
