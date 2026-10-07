//go:build terrainseamgpu

package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

const terrainSeamGPUSize = 256

type terrainSeamTarget struct {
	dst       *ebiten.Image
	reference *ebiten.Image
}

func (t *terrainSeamTarget) DrawTriangles(vertices []ebiten.Vertex, indices []uint16, img *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	if t.reference != nil {
		adjusted := append([]ebiten.Vertex(nil), vertices...)
		if img.Bounds().Dx() == terrain.CellSize+2 {
			for i := range adjusted {
				adjusted[i].SrcX--
				adjusted[i].SrcY--
			}
		}
		vertices, img = adjusted, t.reference
	}
	t.dst.DrawTriangles(vertices, indices, img, op)
}

type terrainSeamGame struct {
	flat, displaced *Viewer
	stage           int
	done            bool
	err             error
}

func (g *terrainSeamGame) Layout(_, _ int) (int, int) {
	return terrainSeamGPUSize, terrainSeamGPUSize
}

func (g *terrainSeamGame) Update() error {
	if g.done {
		return ebiten.Termination
	}
	return nil
}

func (g *terrainSeamGame) Draw(screen *ebiten.Image) {
	g.stage++
	if g.stage < 3 || g.done {
		return
	}
	bg := color.RGBA{R: 255, B: 255, A: 255}
	cell := image.NewRGBA(image.Rect(0, 0, terrain.CellSize, terrain.CellSize))
	palette := [4]color.RGBA{{R: 252, G: 21, B: 23, A: 255}, {R: 25, G: 245, B: 43, A: 255}, {R: 19, G: 37, B: 240, A: 255}, {R: 236, G: 211, B: 25, A: 255}}
	for y := 0; y < terrain.CellSize; y++ {
		for x := 0; x < terrain.CellSize; x++ {
			cell.SetRGBA(x, y, palette[(x%2)+2*(y%2)])
		}
	}
	reference := ebiten.NewImageFromImage(cell)
	actual := ebiten.NewImage(terrainSeamGPUSize, terrainSeamGPUSize)
	old := ebiten.NewImage(terrainSeamGPUSize, terrainSeamGPUSize)
	defer reference.Dispose()
	defer actual.Dispose()
	defer old.Dispose()
	actualPixels := make([]byte, terrainSeamGPUSize*terrainSeamGPUSize*4)
	oldPixels := make([]byte, len(actualPixels))

	for _, mode := range []struct {
		name string
		view *Viewer
	}{{"flat", g.flat}, {"displaced", g.displaced}} {
		for _, zoom := range []float64{1, 1.2} {
			seam := 128.5
			v := mode.view
			v.cam.ViewW, v.cam.ViewH = terrainSeamGPUSize, terrainSeamGPUSize
			v.cam.SetZoom(zoom)
			v.cam.X, v.cam.Y = 32-seam/zoom, 32-seam/zoom
			actual.Fill(bg)
			old.Fill(bg)
			draw := v.drawFlat
			if mode.name == "displaced" {
				draw = v.drawDisplaced
			}
			draw(&terrainSeamTarget{dst: actual})
			draw(&terrainSeamTarget{dst: old, reference: reference})
			actual.ReadPixels(actualPixels)
			old.ReadPixels(oldPixels)
			lo := max(0, int(math.Ceil(seam-32*zoom))+2)
			hi := min(terrainSeamGPUSize, int(math.Floor(seam+64*zoom))-2)
			oldGaps, actualGaps, mismatches := 0, 0, 0
			for y := lo; y < hi; y++ {
				for x := lo; x < hi; x++ {
					i := (y*terrainSeamGPUSize + x) * 4
					a, b := actualPixels[i:i+4], oldPixels[i:i+4]
					isBG := func(p []byte) bool { return p[0] == bg.R && p[1] == bg.G && p[2] == bg.B && p[3] == bg.A }
					if isBG(a) {
						actualGaps++
					}
					if isBG(b) {
						oldGaps++
					} else if a[0] != b[0] || a[1] != b[1] || a[2] != b[2] || a[3] != b[3] {
						mismatches++
					}
				}
			}
			if actualGaps != 0 || mismatches != 0 || (zoom == 1 && oldGaps != 0) || (zoom == 1.2 && oldGaps == 0) {
				g.err = fmt.Errorf("%s zoom %.1f: old gaps %d, actual gaps %d, interior texel mismatches %d", mode.name, zoom, oldGaps, actualGaps, mismatches)
				g.done = true
				return
			}
		}
	}
	screen.Fill(bg)
	g.done = true
}

func TestMain(m *testing.M) {
	if os.Getenv("AGAINROM_TERRAIN_SEAM_GPU") != "1" {
		os.Exit(m.Run())
	}
	if code := m.Run(); code != 0 {
		os.Exit(code)
	}
	pal := color.Palette{color.RGBA{R: 252, G: 21, B: 23, A: 255}, color.RGBA{R: 25, G: 245, B: 43, A: 255}, color.RGBA{R: 19, G: 37, B: 240, A: 255}, color.RGBA{R: 236, G: 211, B: 25, A: 255}}
	cell := image.NewPaletted(image.Rect(0, 0, terrain.CellSize, terrain.CellSize), pal)
	for y := 0; y < terrain.CellSize; y++ {
		for x := 0; x < terrain.CellSize; x++ {
			cell.SetColorIndex(x, y, uint8((x%2)+2*(y%2)))
		}
	}
	ts := &terrain.Tileset{}
	ts.Slots[0] = &terrain.Strip{SubCells: []*image.Paletted{cell}}
	grid := altGrid(3, 3, make([]uint8, 9)...)
	flat, err := NewViewer("flat seam test", grid, ts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flat.SetFlat(true)
	flat.SetUnshaded(true)
	displaced, err := NewViewer("displaced seam test", grid, ts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	displaced.SetUnshaded(true)
	g := &terrainSeamGame{flat: flat, displaced: displaced}
	ebiten.SetWindowSize(terrainSeamGPUSize, terrainSeamGPUSize)
	ebiten.SetWindowPosition(-32000, -32000)
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGameWithOptions(g, &ebiten.RunGameOptions{InitUnfocused: true, SkipTaskbar: true}); err != nil && err != ebiten.Termination {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if g.err != nil {
		fmt.Fprintln(os.Stderr, g.err)
		os.Exit(1)
	}
	os.Exit(0)
}
