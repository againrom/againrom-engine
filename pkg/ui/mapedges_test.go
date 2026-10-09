package ui

import (
	"fmt"
	"image"
	"math"
	"testing"
)

// TERR-216; DIV-2640.
func TestMapCameraLimitsAndTerrainWalk(t *testing.T) {
	const width, height = 64, 64
	spans := []struct {
		cols, rows, extraW, extraH int
	}{
		{15, 15, 0, 0},
		{20, 18, 0, 0},
		{27, 24, 0, 0},
		{20, 18, 17, 24},
	}
	reliefs := []struct {
		name string
		at   func(x, y int) uint8
	}{
		{"flat", nil},
		{"displaced zero", func(x, y int) uint8 { return 0 }},
		{"uniform raised", func(x, y int) uint8 { return 28 }},
		{"unequal raised", func(x, y int) uint8 { return uint8((7*x + 11*y + 37) % 128) }},
		{"uniform negative", func(x, y int) uint8 { return 240 }},
	}
	for _, relief := range reliefs {
		for _, span := range spans {
			name := fmt.Sprintf("%s/%dx%d+%dx%d", relief.name, span.cols, span.rows, span.extraW, span.extraH)
			t.Run(name, func(t *testing.T) {
				g := grid(width, height)
				g.Block = make([]uint8, width*height)
				if relief.at != nil {
					g.Altitudes = make([]uint8, width*height)
				}
				for y := 0; y < height; y++ {
					for x := 0; x < width; x++ {
						if x < 8 || x >= width-8 || y < 8 || y >= height-8 {
							g.Block[y*width+x] = 2
						}
						if relief.at != nil {
							g.Altitudes[y*width+x] = relief.at(x, y)
						}
					}
				}
				v := newViewer(t, g)
				v.cam.ViewW, v.cam.ViewH = span.cols*32+span.extraW, span.rows*32+span.extraH
				origin := 0
				if v.Mode() == ModeDisplaced {
					origin = v.proj.MinV
				}
				for _, edge := range []struct {
					name     string
					dx, dy   float64
					col, row int
				}{
					{"upper", 0, -1e9, 20, 8},
					{"lower", 0, 1e9, 20, height - 8 - span.rows},
					{"left", -1e9, 0, 8, 20},
					{"right", 1e9, 0, width - 8 - span.cols, 20},
				} {
					t.Run(edge.name, func(t *testing.T) {
						v.cam.X, v.cam.Y = 20*32, float64(20*32-origin)
						v.cam.Pan(edge.dx, edge.dy)
						wantX, wantY := float64(edge.col*32), float64(edge.row*32-origin)
						if edge.name == "right" {
							wantX -= float64(span.extraW)
						}
						if v.cam.X != wantX || v.cam.Y != wantY {
							t.Errorf("camera = (%v,%v), want (%v,%v), projection origin %d", v.cam.X, v.cam.Y, wantX, wantY, origin)
						}
						var cells []image.Point
						v.forEachDrawnTile(func(x, y int) { cells = append(cells, image.Pt(x, y)) })
						cols, rows := span.cols, span.rows
						if span.extraW > 0 {
							cols++
						}
						if span.extraH > 0 {
							rows++
						}
						wantCount := cols * (rows + 4)
						if len(cells) != wantCount {
							t.Errorf("terrain walk has %d cells, want %d", len(cells), wantCount)
						}
						for i, got := range cells {
							col := int(math.Floor(wantX / 32))
							row := int(math.Floor((wantY + float64(origin)) / 32))
							want := image.Pt(col+cols-1-i%cols, row+i/cols)
							if got != want {
								t.Errorf("terrain cell %d = %v, want %v; rows ascend and columns descend", i, got, want)
								break
							}
						}
					})
				}
			})
		}
	}
}

func TestMapCameraSpanChangesReapplyLimits(t *testing.T) {
	const width, height = 64, 64
	g := grid(width, height)
	g.Block = make([]uint8, width*height)
	g.Altitudes = make([]uint8, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			g.Altitudes[y*width+x] = 28
			if x < 8 || x >= width-8 || y < 8 || y >= height-8 {
				g.Block[y*width+x] = 2
			}
		}
	}
	v := newViewer(t, g)
	v.cam.ViewW, v.cam.ViewH = 511, 511
	v.cam.Pan(1e9, 1e9)
	if v.cam.X != 1281 || v.cam.Y != 1340 {
		t.Errorf("initial 511x511 lower/right = (%v,%v), want (1281,1340)", v.cam.X, v.cam.Y)
	}
	v.cam.ViewW, v.cam.ViewH = 647, 605
	v.cam.Clamp()
	if v.cam.X != 1145 || v.cam.Y != 1244 {
		t.Errorf("resized 647x605 lower/right = (%v,%v), want (1145,1244)", v.cam.X, v.cam.Y)
	}
	for _, zoom := range []float64{2, 0.75, 8} {
		v.cam.X, v.cam.Y = 1e9, 1e9
		v.cam.SetZoom(zoom)
		if wantX, wantY := 1792-647/zoom, 1820-math.Floor(605/(zoom*32))*32; v.cam.X != wantX || v.cam.Y != wantY {
			t.Errorf("zoom %v lower/right = (%v,%v), want (%v,%v)", zoom, v.cam.X, v.cam.Y, wantX, wantY)
		}
		v.cam.Pan(-1e9, -1e9)
		if v.cam.X != 256 || v.cam.Y != 284 {
			t.Errorf("zoom %v upper/left = (%v,%v), want (256,284)", zoom, v.cam.X, v.cam.Y)
		}
	}
}

func TestMapUpperLimitIgnoresGroundBlockingAndOccupants(t *testing.T) {
	const width, height = 64, 64
	for _, tc := range []struct {
		name              string
		blocked, occupied bool
	}{
		{"open empty", false, false},
		{"blocked empty", true, false},
		{"blocked occupied", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := grid(width, height)
			g.Block = make([]uint8, width*height)
			g.Altitudes = make([]uint8, width*height)
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					if tc.blocked {
						g.Block[y*width+x] = 1
					}
					if x < 8 || x >= width-8 || y < 8 || y >= height-8 {
						g.Block[y*width+x] |= 2
					}
				}
			}
			v := newViewer(t, g)
			if tc.occupied {
				v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(10, 8), Life: LifeAlive}})
			}
			v.cam.ViewW, v.cam.ViewH = 20*32, 18*32
			v.cam.Pan(-1e9, -1e9)
			if v.cam.X != 256 || v.cam.Y != 256 {
				t.Errorf("upper/left = (%v,%v), want (256,256)", v.cam.X, v.cam.Y)
			}
			var first image.Point
			count := 0
			v.forEachDrawnTile(func(x, y int) {
				if count == 0 {
					first = image.Pt(x, y)
				}
				count++
			})
			if first != image.Pt(27, 8) || count != 20*22 {
				t.Errorf("terrain first/count = %v/%d, want (27,8)/440", first, count)
			}
		})
	}
}
