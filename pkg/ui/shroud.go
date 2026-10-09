package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Shroud is composited over the complete world, including structure overhangs.
// TERR-FOG-083 supplies the four vertex levels; TERR-FOG-084 supplies the gain.
// DIV-2578.
func (v *Viewer) drawShroud(dst *ebiten.Image) {
	if len(v.fogPlane) == 0 || v.FogRevealed() {
		return
	}
	size := image.Pt(v.cam.ViewW, v.cam.ViewH)
	if v.fogCanvas == nil || v.fogCanvas.Bounds().Size() != size {
		if v.fogCanvas != nil {
			v.fogCanvas.Deallocate()
		}
		v.fogCanvas = ebiten.NewImage(size.X, size.Y)
	}
	if v.fogInk == nil {
		v.fogInk = ebiten.NewImage(1, 1)
		v.fogInk.Fill(color.Black)
	}
	v.paintShroud(v.fogCanvas, v.fogInk)
	dst.DrawImage(v.fogCanvas, nil)
}

type shroudTarget interface {
	triangleTarget
	Fill(color.Color)
}

func (v *Viewer) paintShroud(dst shroudTarget, ink *ebiten.Image) {
	dst.Fill(color.Transparent)
	v.drawShroudTiles(dst, ink)
}

func (v *Viewer) drawShroudTiles(dst triangleTarget, ink *ebiten.Image) {
	op := &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendCopy}
	v.forEachDrawnTile(func(col, row int) {
		verts := flatTileVertices(v.cam, col, row)
		if v.Mode() == ModeDisplaced {
			verts = tileVertices(v.cam, v.proj, col, row)
		}
		for i, p := range [4]image.Point{{col, row}, {col + 1, row}, {col, row + 1}, {col + 1, row + 1}} {
			verts[i].SrcX, verts[i].SrcY = 0.5, 0.5
			verts[i].ColorR, verts[i].ColorG, verts[i].ColorB = 1, 1, 1
			verts[i].ColorA = 1 - fogScale(v.fogAt(p.X, p.Y))
		}
		dst.DrawTriangles(verts[:], quadIndices[:], ink, op)
	})
}
