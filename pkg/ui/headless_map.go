package ui

import (
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// HeadlessMapFrame composes a w by h map window centred on cell on the CPU:
// the production terrain triangles with their corner scales, then the
// production art submissions source-over with their colour scales, nearest
// sampling at pixel centres. Shroud, HUD, diagnostics and GPU blend modes are
// not composed. The camera is restored afterwards. The art submissions are
// returned with the frame.
func (v *Viewer) HeadlessMapFrame(cell image.Point, w, h int) (*image.RGBA, []HeadlessArtDraw, error) {
	saved := *v.cam
	defer func() { *v.cam = saved }()
	v.cam.ViewW, v.cam.ViewH = w, h
	v.cam.SetZoom(1)
	v.cam.CenterOn(v.cellWorldCentre(cell))
	v.refreshHoverLighting()
	v.refreshStructureLighting()
	r := &headlessTerrainTarget{v: v, pix: image.NewRGBA(image.Rect(0, 0, w, h)), sources: map[*ebiten.Image]*image.RGBA{}}
	if v.Mode() == ModeDisplaced {
		v.drawDisplaced(r)
	} else {
		v.drawFlat(r)
	}
	if r.err != nil {
		return nil, nil, r.err
	}
	draws, err := v.HeadlessArtDraws()
	if err != nil {
		return nil, nil, err
	}
	for _, d := range draws {
		compositeHeadlessArt(r.pix, d)
	}
	return r.pix, draws, nil
}

type headlessTerrainTarget struct {
	v       *Viewer
	pix     *image.RGBA
	sources map[*ebiten.Image]*image.RGBA
	err     error
}

func (r *headlessTerrainTarget) source(texture *ebiten.Image) *image.RGBA {
	if src := r.sources[texture]; src != nil {
		return src
	}
	for key, cached := range r.v.cache {
		if cached != texture {
			continue
		}
		src := r.v.set.Slot(key.slot).SubCell(key.sub)
		if src == nil {
			return nil
		}
		var dirt *image.Paletted
		if key.dirt != 0 {
			dirt = r.v.set.Dirt.SubCell(int(key.dirt) - 1)
		}
		pixels := paddedCellPixels(scorchedCellPixels(src, dirt, key.tint))
		r.sources[texture] = pixels
		return pixels
	}
	return nil
}

func (r *headlessTerrainTarget) DrawTriangles(vertices []ebiten.Vertex, indices []uint16, texture *ebiten.Image, _ *ebiten.DrawTrianglesOptions) {
	src := r.source(texture)
	if src == nil {
		r.err = errors.New("headless map: unidentified terrain texture")
		return
	}
	for i := 0; i+2 < len(indices); i += 3 {
		a, b, c := vertices[indices[i]], vertices[indices[i+1]], vertices[indices[i+2]]
		ax, ay, bx, by, cx, cy := float64(a.DstX), float64(a.DstY), float64(b.DstX), float64(b.DstY), float64(c.DstX), float64(c.DstY)
		den := (by-cy)*(ax-cx) + (cx-bx)*(ay-cy)
		if den == 0 {
			continue
		}
		box := image.Rect(int(math.Floor(min(ax, bx, cx))), int(math.Floor(min(ay, by, cy))),
			int(math.Ceil(max(ax, bx, cx))), int(math.Ceil(max(ay, by, cy)))).Intersect(r.pix.Bounds())
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				px, py := float64(x)+0.5, float64(y)+0.5
				wa := ((by-cy)*(px-cx) + (cx-bx)*(py-cy)) / den
				wb := ((cy-ay)*(px-cx) + (ax-cx)*(py-cy)) / den
				wc := 1 - wa - wb
				if wa < -1e-9 || wb < -1e-9 || wc < -1e-9 {
					continue
				}
				at := func(a, b, c float32) float64 { return wa*float64(a) + wb*float64(b) + wc*float64(c) }
				s := src.RGBAAt(int(math.Floor(at(a.SrcX, b.SrcX, c.SrcX))), int(math.Floor(at(a.SrcY, b.SrcY, c.SrcY))))
				scale := func(ch uint8, a, b, c float32) uint8 {
					return uint8(math.Round(min(255, max(0, float64(ch)*at(a, b, c)))))
				}
				r.pix.SetRGBA(x, y, color.RGBA{scale(s.R, a.ColorR, b.ColorR, c.ColorR), scale(s.G, a.ColorG, b.ColorG, c.ColorG),
					scale(s.B, a.ColorB, b.ColorB, c.ColorB), 255})
			}
		}
	}
}

// compositeHeadlessArt draws one art submission source-over through its
// geometry's inverse and its colour scale.
func compositeHeadlessArt(dst *image.RGBA, d HeadlessArtDraw) {
	inv := d.Geometry
	if !inv.IsInvertible() {
		return
	}
	inv.Invert()
	sb := d.Pixels.Bounds()
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range [4][2]float64{{0, 0}, {float64(sb.Dx()), 0}, {0, float64(sb.Dy())}, {float64(sb.Dx()), float64(sb.Dy())}} {
		x, y := d.Geometry.Apply(p[0], p[1])
		minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
	}
	box := image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX)), int(math.Ceil(maxY))).Intersect(dst.Bounds())
	cs := [4]float64{float64(d.ColorScale.R()), float64(d.ColorScale.G()), float64(d.ColorScale.B()), float64(d.ColorScale.A())}
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			sx, sy := inv.Apply(float64(x)+0.5, float64(y)+0.5)
			p := image.Pt(sb.Min.X+int(math.Floor(sx)), sb.Min.Y+int(math.Floor(sy)))
			if !p.In(sb) {
				continue
			}
			s := d.Pixels.RGBAAt(p.X, p.Y)
			if s.A == 0 {
				continue
			}
			src := [4]float64{float64(s.R) * cs[0], float64(s.G) * cs[1], float64(s.B) * cs[2], float64(s.A) * cs[3]}
			o := dst.RGBAAt(x, y)
			keep := 1 - min(src[3], 255)/255
			mix := func(sc float64, dc uint8) uint8 { return uint8(math.Round(min(255, max(0, sc+float64(dc)*keep)))) }
			dst.SetRGBA(x, y, color.RGBA{mix(src[0], o.R), mix(src[1], o.G), mix(src[2], o.B), 255})
		}
	}
}
