package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type headlessShroudTarget struct {
	pixels *image.RGBA
	err    error
}

func (r *headlessShroudTarget) Fill(c color.Color) {
	draw.Draw(r.pixels, r.pixels.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
}

func (r *headlessShroudTarget) DrawTriangles(vertices []ebiten.Vertex, indices []uint16, _ *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	if r.err != nil {
		return
	}
	if op == nil || op.Blend != ebiten.BlendCopy || len(indices)%3 != 0 {
		r.err = fmt.Errorf("headless shroud: unsupported triangle blend or indices")
		return
	}
	for _, v := range vertices {
		if math.IsNaN(float64(v.DstX)) || math.IsNaN(float64(v.DstY)) || math.IsInf(float64(v.DstX), 0) || math.IsInf(float64(v.DstY), 0) {
			r.err = fmt.Errorf("headless shroud: nonfinite triangle position")
			return
		}
	}
	for i := 0; i < len(indices); i += 3 {
		if int(indices[i]) >= len(vertices) || int(indices[i+1]) >= len(vertices) || int(indices[i+2]) >= len(vertices) {
			r.err = fmt.Errorf("headless shroud: triangle index outside vertices")
			return
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
				alpha := wa*float64(a.ColorA) + wb*float64(b.ColorA) + wc*float64(c.ColorA)
				r.pixels.SetRGBA(x, y, color.RGBA{A: uint8(math.Round(min(max(alpha, 0), 1) * 255))})
			}
		}
	}
}

// HeadlessShroudMask rasterizes the production fill and triangle submissions.
// It samples pixel centres; GPU edge coverage and rounding remain unmeasured.
func (v *Viewer) HeadlessShroudMask() (*image.RGBA, error) {
	if v.cam.ViewW <= 0 || v.cam.ViewH <= 0 {
		return nil, fmt.Errorf("headless shroud: empty viewport")
	}
	r := headlessShroudTarget{pixels: image.NewRGBA(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH))}
	if len(v.fogPlane) > 0 && !v.FogRevealed() {
		v.paintShroud(&r, nil)
	}
	return r.pixels, r.err
}
