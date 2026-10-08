package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/render/terrain"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestShroudPreservesStructureArtAboveTheTerrainMesh(t *testing.T) {
	g := grid(16, 16)
	g.Structures = []terrain.StructureRecord{{Key: 1}}
	frame := staticsFrame(32, 32, 1)
	frame.Palette[1] = color.RGBA{R: 192, G: 64, B: 32, A: 255}
	set := &terrain.StructureSet{}
	set.Classes[1] = &terrain.StructureClass{TileWidth: 1, TileHeight: 1, FullHeight: 2, Frames: []*terrain.StaticFrame{frame, frame}}
	v := newStructureViewer(t, g, set, true)
	v.SetUnshaded(true)
	v.cam.X, v.cam.Y, v.cam.Zoom, v.cam.ViewW, v.cam.ViewH = 0, -32, 1, 32, 96
	plane := make([]byte, 16*16)
	for i := range plane {
		plane[i] = FogVisible
	}
	v.SetFog(plane, 16, 16)
	if v.FogRevealed() {
		t.Fatal("fixture bypasses the shroud")
	}
	draws, err := v.HeadlessArtDraws()
	if err != nil {
		t.Fatal(err)
	}
	var source color.RGBA
	for _, d := range draws {
		if d.Kind != "sprite" || d.Frame != frame {
			continue
		}
		x, y := d.Geometry.Apply(0, 0)
		if x == 0 && y == 0 {
			source = d.Pixels.RGBAAt(1, 1)
		}
	}
	if source.A != 255 || source.R == 0 {
		t.Fatalf("structure supplied no opaque colored art at the view top: %v", source)
	}
	mask, err := v.HeadlessShroudMask()
	if err != nil {
		t.Fatal(err)
	}
	output := image.NewRGBA(mask.Bounds())
	output.SetRGBA(1, 1, source)
	draw.Draw(output, output.Bounds(), mask, mask.Bounds().Min, draw.Over)
	if got := output.RGBAAt(1, 1); got != source {
		t.Fatalf("all-visible fog cuts structure art above the mesh: final %v, source %v, shroud %v", got, source, mask.RGBAAt(1, 1))
	}
}

func TestShroudStillHidesUnseenTerrain(t *testing.T) {
	v, err := NewViewer("shroud", grid(16, 16), &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	v.cam.X, v.cam.Y, v.cam.Zoom, v.cam.ViewW, v.cam.ViewH = 0, 0, 1, 32, 32
	v.SetFog(make([]byte, 16*16), 16, 16)
	mask, err := v.HeadlessShroudMask()
	if err != nil {
		t.Fatal(err)
	}
	if got := mask.RGBAAt(16, 16); got != (color.RGBA{A: 255}) {
		t.Fatalf("unseen terrain shroud = %v, want opaque black", got)
	}
	v.SetFogReveal(true)
	mask, err = v.HeadlessShroudMask()
	if err != nil {
		t.Fatal(err)
	}
	if got := mask.RGBAAt(16, 16); got != (color.RGBA{}) {
		t.Fatalf("reveal still draws shroud %v", got)
	}
}

func TestHeadlessShroudRasterInterpolatesAndCopiesTriangleAlpha(t *testing.T) {
	r := headlessShroudTarget{pixels: image.NewRGBA(image.Rect(0, 0, 4, 4))}
	r.Fill(color.Black)
	r.DrawTriangles([]ebiten.Vertex{
		{DstX: 0, DstY: 0, ColorA: 0},
		{DstX: 4, DstY: 0, ColorA: 1},
		{DstX: 0, DstY: 4, ColorA: 1},
	}, []uint16{0, 1, 2}, nil, &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendCopy})
	if r.err != nil {
		t.Fatal(r.err)
	}
	if got := r.pixels.RGBAAt(0, 0); got != (color.RGBA{A: 64}) {
		t.Fatalf("triangle centre alpha = %v, want 64 replacing the opaque fill", got)
	}
	if got := r.pixels.RGBAAt(3, 3); got != (color.RGBA{A: 255}) {
		t.Fatalf("pixel outside the triangle = %v, want retained opaque fill", got)
	}
}
