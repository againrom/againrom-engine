package ui

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
	"github.com/hajimehoshi/ebiten/v2"
)

type mapClipTarget struct {
	t       *testing.T
	view    *Viewer
	pixels  *image.RGBA
	painted []bool
	sources map[*ebiten.Image]*image.RGBA
	draws   int
}

func (r *mapClipTarget) source(texture *ebiten.Image) *image.RGBA {
	if src := r.sources[texture]; src != nil {
		return src
	}
	for key, cached := range r.view.cache {
		if cached != texture {
			continue
		}
		src := r.view.set.Slot(key.slot).SubCell(key.sub)
		if src == nil {
			r.t.Fatal("installed terrain draw selected a placeholder")
		}
		var dirt *image.Paletted
		if key.dirt != 0 {
			dirt = r.view.set.Dirt.SubCell(int(key.dirt) - 1)
		}
		pixels := paddedCellPixels(scorchedCellPixels(src, dirt, key.tint))
		if pixels.Bounds() != texture.Bounds() {
			r.t.Fatal("submitted terrain texture has different bounds from its CPU source")
		}
		r.sources[texture] = pixels
		return pixels
	}
	r.t.Fatal("production submitted an unidentified terrain texture")
	return nil
}

func (r *mapClipTarget) DrawTriangles(vertices []ebiten.Vertex, indices []uint16, texture *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	if op == nil || op.Filter != ebiten.FilterNearest || len(indices)%3 != 0 {
		r.t.Fatal("unsupported terrain triangle submission")
	}
	src := r.source(texture)
	r.draws++
	for i := 0; i < len(indices); i += 3 {
		if int(indices[i]) >= len(vertices) || int(indices[i+1]) >= len(vertices) || int(indices[i+2]) >= len(vertices) {
			r.t.Fatal("terrain index outside submitted vertices")
		}
		a, b, c := vertices[indices[i]], vertices[indices[i+1]], vertices[indices[i+2]]
		ax, ay, bx, by, cx, cy := float64(a.DstX), float64(a.DstY), float64(b.DstX), float64(b.DstY), float64(c.DstX), float64(c.DstY)
		den := (by-cy)*(ax-cx) + (cx-bx)*(ay-cy)
		if den == 0 {
			continue
		}
		box := image.Rect(int(math.Floor(min(ax, bx, cx))), int(math.Floor(min(ay, by, cy))), int(math.Ceil(max(ax, bx, cx))), int(math.Ceil(max(ay, by, cy)))).Intersect(r.pixels.Bounds())
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				px, py := float64(x)+0.5, float64(y)+0.5
				wa := ((by-cy)*(px-cx) + (cx-bx)*(py-cy)) / den
				wb := ((cy-ay)*(px-cx) + (ax-cx)*(py-cy)) / den
				wc := 1 - wa - wb
				if wa < -1e-9 || wb < -1e-9 || wc < -1e-9 {
					continue
				}
				interpolate := func(a, b, c float32) float64 { return wa*float64(a) + wb*float64(b) + wc*float64(c) }
				sx, sy := int(math.Floor(interpolate(a.SrcX, b.SrcX, c.SrcX))), int(math.Floor(interpolate(a.SrcY, b.SrcY, c.SrcY)))
				if !image.Pt(sx, sy).In(src.Bounds()) {
					r.t.Fatal("terrain source sample outside submitted texture")
				}
				pixel := src.RGBAAt(sx, sy)
				scale := func(channel uint8, a, b, c float32) uint8 {
					return uint8(math.Round(min(255, max(0, float64(channel)*interpolate(a, b, c)))))
				}
				r.pixels.SetRGBA(x, y, color.RGBA{scale(pixel.R, a.ColorR, b.ColorR, c.ColorR), scale(pixel.G, a.ColorG, b.ColorG, c.ColorG), scale(pixel.B, a.ColorB, b.ColorB, c.ColorB), scale(pixel.A, a.ColorA, b.ColorA, c.ColorA)})
				r.painted[y*r.pixels.Bounds().Dx()+x] = true
			}
		}
	}
}

func mapClipOutputPath(root, output string) (string, error) {
	if !filepath.IsAbs(output) {
		return "", fmt.Errorf("absolute AGAINROM_MAP_CLIP_WITNESS_DIR required")
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	parent, suffix := filepath.Clean(output), ""
	for {
		if _, err := os.Lstat(parent); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return "", err
		}
		next := filepath.Dir(parent)
		if next == parent {
			return "", fmt.Errorf("map clip output has no existing ancestor")
		}
		suffix = filepath.Join(filepath.Base(parent), suffix)
		parent = next
	}
	parent, err = filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	output = filepath.Join(parent, suffix)
	rel, err := filepath.Rel(root, output)
	if err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
		return "", fmt.Errorf("map clip output is inside the installed asset root")
	}
	return filepath.Join(output, fmt.Sprintf("%x", sha256.Sum256([]byte(filepath.ToSlash(root))))), nil
}

func TestReleaseMapViewportPaintsInstalledTerrain(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: map clip witness requires installed terrain")
	}
	source, err := vfs.Open([]string{filepath.Join(root, "scenario.res"), filepath.Join(root, "graphics.res")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := source.ReadFile("scenario/20.alm")
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	g := terrain.Grid{Width: m.Width, Height: m.Height, Tiles: terrain.RenderTileWords(m.Tiles), Altitudes: m.Altitudes, Block: mapload.Passability(m)}
	v, err := NewViewer("map clip witness", g, terrain.LoadTileset(source))
	if err != nil {
		t.Fatal(err)
	}
	v.SetAnimated(false)
	v.SetUnshaded(true)
	output := os.Getenv("AGAINROM_MAP_CLIP_WITNESS_DIR")
	if output != "" {
		root, err = filepath.Abs(root)
		if err != nil {
			t.Fatal(err)
		}
		output, err = mapClipOutputPath(root, output)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(output, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, flat := range []bool{false, true} {
		v.SetFlat(flat)
		mode := map[bool]string{false: "displaced", true: "flat"}[flat]
		for _, size := range []image.Point{{864, 768}, {1205, 678}} {
			v.cam.ViewW, v.cam.ViewH = size.X, size.Y
			for _, zoom := range []float64{0.75, 1, 1.2, 2} {
				for _, position := range []struct {
					name   string
					dx, dy float64
				}{{"middle", 0, 0}, {"right", 1e9, 0}, {"bottom", 0, 1e9}, {"right-bottom", 1e9, 1e9}} {
					name := fmt.Sprintf("%s/%dx%d/zoom%g/%s", mode, size.X, size.Y, zoom, position.name)
					t.Run(name, func(t *testing.T) {
						v.cam.SetZoom(zoom)
						v.cam.CenterOn(v.cam.WorldW()/2+9.25, v.cam.WorldH()/2+13.5)
						v.cam.Pan(position.dx, position.dy)
						r := &mapClipTarget{t: t, view: v, pixels: image.NewRGBA(image.Rect(0, 0, size.X, size.Y)), painted: make([]bool, size.X*size.Y), sources: make(map[*ebiten.Image]*image.RGBA)}
						if flat {
							v.drawFlat(r)
						} else {
							v.drawDisplaced(r)
						}
						coloredColumns, coloredRows := make([]bool, size.X), make([]bool, size.Y)
						for y := 0; y < size.Y; y++ {
							for x := 0; x < size.X; x++ {
								pixel := r.pixels.RGBAAt(x, y)
								if !r.painted[y*size.X+x] || pixel.A != 255 {
									t.Fatalf("viewport pixel (%d,%d) left unpainted at camera (%g,%g): %v", x, y, v.cam.X, v.cam.Y, pixel)
								}
								if pixel.R != 0 || pixel.G != 0 || pixel.B != 0 {
									coloredColumns[x], coloredRows[y] = true, true
								}
							}
						}
						for x, painted := range coloredColumns {
							if !painted {
								t.Fatalf("black viewport column %d at camera (%g,%g)", x, v.cam.X, v.cam.Y)
							}
						}
						for y, painted := range coloredRows {
							if !painted {
								t.Fatalf("black viewport row %d at camera (%g,%g)", y, v.cam.X, v.cam.Y)
							}
						}
						t.Logf("mission 20: %d viewport pixels painted; %d nonblack columns, %d nonblack rows; %d terrain submissions; camera (%g,%g)", size.X*size.Y, size.X, size.Y, r.draws, v.cam.X, v.cam.Y)
						if output != "" {
							label := strings.ReplaceAll(name, "/", "-")
							file, err := os.Create(filepath.Join(output, label+".png"))
							if err != nil {
								t.Fatal(err)
							}
							encodeErr, closeErr := png.Encode(file, r.pixels), file.Close()
							if encodeErr != nil || closeErr != nil {
								t.Fatal(encodeErr, closeErr)
							}
							proof := map[string]any{"mission": 20, "mode": mode, "viewport": size, "zoom": zoom, "position": position.name, "camera": [2]float64{v.cam.X, v.cam.Y}, "painted_pixels": size.X * size.Y, "nonblack_columns": size.X, "nonblack_rows": size.Y, "terrain_submissions": r.draws, "frame_sha256": fmt.Sprintf("%x", sha256.Sum256(r.pixels.Pix)), "raster_limit": "CPU pixel-centre raster of production terrain triangles; excludes structures, shroud, HUD and GPU coverage"}
							data, err := json.MarshalIndent(proof, "", "  ")
							if err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(filepath.Join(output, label+".json"), data, 0644); err != nil {
								t.Fatal(err)
							}
						}
					})
				}
			}
		}
	}
}
