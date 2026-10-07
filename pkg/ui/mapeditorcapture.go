package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"againrom/pkg/render/terrain"
)

// HeadlessFrame composes the editor's actual rail and retained body placements
// over the render tier's CPU terrain raster. It is a diagnostic, not GPU
// readback: triangle edge sampling can differ and projected shadows are omitted.
// No native window, input synthesis or simulation is needed.
func (e *MapEditor) HeadlessFrame() (*image.RGBA, string, error) {
	if e.doc == nil {
		return nil, "", fmt.Errorf("no map open")
	}
	v := e.doc.Viewer
	var ground *terrain.Render
	var err error
	if v.Mode() == ModeDisplaced {
		ground, err = terrain.CompositeProjectedLit(v.set, v.grid, v.grid.Altitudes, v.Sun(), 1)
	} else {
		ground, err = terrain.CompositeLit(v.set, v.grid, v.grid.Altitudes, v.Sun(), 1)
	}
	if err != nil {
		return nil, "", err
	}
	pic := image.NewRGBA(image.Rect(0, 0, v.frameW, v.frameH))
	draw.Draw(pic, pic.Bounds(), image.NewUniform(editorInk), image.Point{}, draw.Src)
	canvas := pic.SubImage(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH)).(*image.RGBA)
	for y := 0; y < v.cam.ViewH; y++ {
		for x := 0; x < v.cam.ViewW; x++ {
			wx, wy := v.cam.ScreenToWorld(float64(x)+0.5, float64(y)+0.5)
			p := image.Pt(int(math.Floor(wx)), int(math.Floor(wy)))
			if p.In(ground.Image.Bounds()) {
				canvas.SetRGBA(x, y, ground.Image.RGBAAt(p.X, p.Y))
			}
		}
	}
	for _, p := range v.structurePlacements() {
		if p.Class == nil || !p.Class.Flat || p.Frame == nil {
			continue
		}
		if r, ok := spriteScreenRect(p.Rect(), v.cam); ok {
			editorBlit(canvas, v.spritePixels(p.Frame), r, false)
		}
	}
	for _, s := range v.planeSprites() {
		if s.Frame != nil {
			editorBlit(canvas, v.spritePixels(s.Frame), s.screenRect, s.Mirror)
		}
	}
	for _, s := range e.markerLines() {
		dx, dy := float64(s.x1-s.x0), float64(s.y1-s.y0)
		n := max(1, int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy)))))
		for i := 0; i <= n; i++ {
			canvas.SetRGBA(int(float64(s.x0)+dx*float64(i)/float64(n)), int(float64(s.y0)+dy*float64(i)/float64(n)), s.color)
		}
	}
	draw.Draw(pic, image.Rect(v.cam.ViewW, 0, v.frameW, v.frameH), e.PanelImage(v.frameH), image.Point{}, draw.Src)
	out := image.NewRGBA(image.Rect(0, 0, e.window.X, e.window.Y))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.RGBA{0, 0, 0, 255}), image.Point{}, draw.Src)
	ox, oy := v.place.Origin()
	editorBlit(out, pic, screenRect{X: ox, Y: oy, W: float64(v.frameW) * v.place.Scale(), H: float64(v.frameH) * v.place.Scale()}, false)
	return out, "CPU terrain + retained body placements + actual rail; no GPU readback; shadows omitted", nil
}

func editorBlit(dst, src *image.RGBA, r screenRect, mirror bool) {
	if src == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	box := image.Rect(int(math.Floor(r.X)), int(math.Floor(r.Y)), int(math.Ceil(r.X+r.W)), int(math.Ceil(r.Y+r.H))).Intersect(dst.Bounds())
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			sx := int((float64(x) + 0.5 - r.X) / r.W * float64(src.Bounds().Dx()))
			sy := int((float64(y) + 0.5 - r.Y) / r.H * float64(src.Bounds().Dy()))
			if mirror {
				sx = src.Bounds().Dx() - 1 - sx
			}
			c := src.RGBAAt(src.Rect.Min.X+sx, src.Rect.Min.Y+sy)
			if c.A == 0 {
				continue
			}
			if c.A == 255 {
				dst.SetRGBA(x, y, c)
				continue
			}
			old := dst.RGBAAt(x, y)
			a := uint32(c.A)
			dst.SetRGBA(x, y, color.RGBA{uint8(uint32(c.R) + uint32(old.R)*(255-a)/255), uint8(uint32(c.G) + uint32(old.G)*(255-a)/255), uint8(uint32(c.B) + uint32(old.B)*(255-a)/255), 255})
		}
	}
}
