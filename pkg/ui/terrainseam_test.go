package ui

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

type terrainTextureCall struct {
	vertices []ebiten.Vertex
	bounds   image.Rectangle
}

func TestPaddedCellPixelsKeepCheckerTexelsAndDuplicateEdges(t *testing.T) {
	src := image.NewRGBA(image.Rect(4, 7, 36, 39))
	colors := [4]color.RGBA{
		{R: 252, G: 21, B: 23, A: 255},
		{R: 25, G: 245, B: 43, A: 255},
		{R: 19, G: 37, B: 240, A: 255},
		{R: 236, G: 211, B: 25, A: 255},
	}
	for y := 0; y < terrain.CellSize; y++ {
		for x := 0; x < terrain.CellSize; x++ {
			src.SetRGBA(x+4, y+7, colors[(x%2)+2*(y%2)])
		}
	}
	got := paddedCellPixels(src)
	if got.Bounds() != image.Rect(0, 0, 34, 34) {
		t.Fatalf("padded checker bounds = %v", got.Bounds())
	}
	for y := 0; y < 34; y++ {
		for x := 0; x < 34; x++ {
			sx := min(31, max(0, x-1))
			sy := min(31, max(0, y-1))
			if want := src.RGBAAt(sx+4, sy+7); got.RGBAAt(x, y) != want {
				t.Fatalf("padded checker (%d,%d) = %v, source (%d,%d) = %v", x, y, got.RGBAAt(x, y), sx, sy, want)
			}
		}
	}
}

type terrainTextureTarget struct {
	calls []terrainTextureCall
}

func (r *terrainTextureTarget) DrawTriangles(vertices []ebiten.Vertex, _ []uint16, img *ebiten.Image, _ *ebiten.DrawTrianglesOptions) {
	r.calls = append(r.calls, terrainTextureCall{
		vertices: append([]ebiten.Vertex(nil), vertices...),
		bounds:   img.Bounds(),
	})
}

func TestTerrainDrawKeepsTheFullCellInsideAnOpaqueBorder(t *testing.T) {
	g := altGrid(3, 3, make([]uint8, 9)...)
	g.Tiles[1] = slotWord(1)
	v, err := NewViewer("terrain border", g, litTileset())
	if err != nil {
		t.Fatal(err)
	}

	for _, mode := range []string{"flat", "displaced"} {
		v.SetFlat(mode == "flat")
		r := &terrainTextureTarget{}
		if mode == "flat" {
			v.drawFlat(r)
		} else {
			v.drawDisplaced(r)
		}
		var cells [][2]int
		v.forEachDrawnTile(func(x, y int) { cells = append(cells, [2]int{x, y}) })
		if len(r.calls) != len(cells) || len(cells) != 9 {
			t.Fatalf("%s drew %d tiles for %d cells, want all 9", mode, len(r.calls), len(cells))
		}
		for i, cell := range cells {
			call := r.calls[i]
			if want := image.Rect(0, 0, terrain.CellSize+2, terrain.CellSize+2); call.bounds != want {
				t.Fatalf("%s tile %v texture bounds = %v, want %v", mode, cell, call.bounds, want)
			}
			var raw [4]ebiten.Vertex
			if mode == "flat" {
				raw = flatTileVertices(v.cam, cell[0], cell[1])
			} else {
				raw = tileVertices(v.cam, v.proj, cell[0], cell[1])
			}
			for j, got := range call.vertices {
				if got.DstX != raw[j].DstX || got.DstY != raw[j].DstY ||
					got.SrcX != raw[j].SrcX+1 || got.SrcY != raw[j].SrcY+1 {
					t.Fatalf("%s tile %v corner %d = %+v, want unchanged destination and source %+v", mode, cell, j, got, raw[j])
				}
			}
		}
	}
}
