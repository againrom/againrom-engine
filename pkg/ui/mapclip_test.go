package ui

import (
	"fmt"
	"image"
	"math"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type mapCoverageTarget struct {
	headlessShroudTarget
}

func (r *mapCoverageTarget) DrawTriangles(vertices []ebiten.Vertex, indices []uint16, _ *ebiten.Image, _ *ebiten.DrawTrianglesOptions) {
	vertices = append([]ebiten.Vertex(nil), vertices...)
	for i := range vertices {
		vertices[i].ColorA = 1
	}
	r.headlessShroudTarget.DrawTriangles(vertices, indices, nil, &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendCopy})
}

func TestMapViewportTerrainCoversFractionalSpansAndOrigins(t *testing.T) {
	const width, height = 512, 512
	for _, relief := range []struct {
		name string
		alt  int
	}{
		{"flat", -1},
		{"displaced", 0},
		{"raised", 127},
	} {
		g := grid(width, height)
		g.Block = make([]uint8, width*height)
		if relief.alt >= 0 {
			g.Altitudes = make([]uint8, width*height)
		}
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				if x < 8 || x >= width-8 || y < 8 || y >= height-8 {
					g.Block[y*width+x] = borderBit
				}
				if relief.alt >= 0 {
					g.Altitudes[y*width+x] = uint8(relief.alt)
				}
			}
		}
		v := newViewer(t, g)
		for _, view := range []image.Point{{864, 768}, {1205, 677}, {1760, 1024}} {
			for _, zoom := range []float64{0.125, 0.75, 1, 1.2, 2, 8} {
				for _, position := range []struct {
					name   string
					dx, dy float64
				}{
					{"middle", 0, 0},
					{"left", -1e9, 0},
					{"right", 1e9, 0},
					{"top", 0, -1e9},
					{"bottom", 0, 1e9},
				} {
					if relief.alt > 0 && position.name == "top" {
						continue
					}
					t.Run(fmt.Sprintf("%s/%dx%d/zoom%g/%s", relief.name, view.X, view.Y, zoom, position.name), func(t *testing.T) {
						v.cam.ViewW, v.cam.ViewH = view.X, view.Y
						v.cam.SetZoom(zoom)
						v.cam.X, v.cam.Y = 8000.75, 8000.25
						v.cam.Pan(position.dx, position.dy)
						r := &mapCoverageTarget{headlessShroudTarget{pixels: image.NewRGBA(image.Rect(0, 0, view.X, view.Y))}}
						if relief.alt < 0 {
							v.drawFlat(r)
						} else {
							v.drawDisplaced(r)
						}
						if r.err != nil {
							t.Fatal(r.err)
						}
						for y := 0; y < view.Y; y++ {
							for x := 0; x < view.X; x++ {
								if r.pixels.RGBAAt(x, y).A != 255 {
									t.Fatalf("terrain leaves pixel (%d,%d) unpainted, camera (%g,%g), zoom %g", x, y, v.cam.X, v.cam.Y, zoom)
								}
							}
						}
					})
				}
			}
		}
	}
}

func TestMapWheelZoomKeepsViewportCentre(t *testing.T) {
	v := newViewer(t, grid(512, 512))
	v.frameW, v.frameH = 1365, 768
	v.syncMapViewport()
	v.cam.CenterOn(8192, 8192)
	sx, sy := float64(v.cam.ViewW)/2, float64(v.cam.ViewH)/2
	wx, wy := v.cam.ScreenToWorld(sx, sy)
	for _, wheel := range []float64{-1, 1} {
		v.step(Input{CursorX: 177, CursorY: 211, WheelY: wheel}, time.Unix(1, 0))
		x, y := v.cam.ScreenToWorld(sx, sy)
		if math.Abs(x-wx) > 1e-6 || math.Abs(y-wy) > 1e-6 {
			t.Fatalf("wheel %g moved centre from (%g,%g) to (%g,%g)", wheel, wx, wy, x, y)
		}
	}
	if math.Abs(v.cam.Zoom-1) > 1e-6 {
		t.Fatalf("zoom out and back = %g, want 1", v.cam.Zoom)
	}
}

func TestMapZoomBeyondDrawableSpanCentresTheMap(t *testing.T) {
	g := grid(64, 64)
	g.Block = make([]uint8, 64*64)
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			if x < 8 || x >= 56 || y < 8 || y >= 56 {
				g.Block[y*64+x] = borderBit
			}
		}
	}
	v := newViewer(t, g)
	v.cam.ViewW, v.cam.ViewH = 864, 768
	v.cam.SetZoom(0.125)
	for _, pan := range []float64{-1e9, 1e9} {
		v.cam.Pan(pan, pan)
		x, y := v.cam.ScreenToWorld(432, 384)
		if x != 1024 || y != 1024 {
			t.Fatalf("oversized view centre = (%g,%g), want drawable centre (1024,1024)", x, y)
		}
	}
}

func TestMapEditorWheelKeepsCursorAnchor(t *testing.T) {
	v := newViewer(t, grid(512, 512))
	v.editorView = true
	v.cam.CenterOn(8192, 8192)
	wx, wy := v.cam.ScreenToWorld(177, 211)
	v.step(Input{CursorX: 177, CursorY: 211, WheelY: -1}, time.Unix(1, 0))
	x, y := v.cam.ScreenToWorld(177, 211)
	if math.Abs(x-wx) > 1e-6 || math.Abs(y-wy) > 1e-6 {
		t.Fatalf("editor cursor anchor moved from (%g,%g) to (%g,%g)", wx, wy, x, y)
	}
}
