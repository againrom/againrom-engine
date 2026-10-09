package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"slices"

	"againrom/pkg/render/terrain"
)

// statusBar is one overhead bar the frame shows: the unit's anchor cell, whose
// relief lift the bar takes, the bar's kind, its native map rectangle with the
// unit's displacement already added, how many interior columns its pool fills,
// and whether it is faded because its unit is not selected.
type statusBar struct {
	Cell           image.Point
	Kind           terrain.StatusBarKind
	Rect           image.Rectangle
	Fill           int
	Faded          bool
	Value, Maximum int
}

// statusBars admits visible non-dead units with positive pool maxima.
// Selection and Show Health choose opaque or blended interiors. TERR-225
// Walking and relief displacement remain independent of damage jolt.
func (v *Viewer) statusBars() []statusBar {
	var bars []statusBar
	for _, e := range v.entities {
		if e.Life == LifeDead || !v.entityStatusShown(e.ID) {
			continue
		}
		if !v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
			continue
		}
		faded := !slices.Contains(v.sel, e.ID)
		shift := v.entityShift(e)
		for _, pool := range [...]struct {
			kind           terrain.StatusBarKind
			value, maximum int
		}{
			{terrain.HealthBar, e.HP, e.MaxHP},
			{terrain.ManaBar, e.Mana, e.MaxMana},
		} {
			rect, ok := terrain.StatusBarRect(pool.kind, e.Cell.X, e.Cell.Y, v.grid.Width, v.grid.Height, e.TokenSize, e.Art)
			if !ok {
				continue
			}
			fill, ok := terrain.StatusBarFill(pool.kind, rect.Dx(), pool.value, pool.maximum)
			if !ok {
				continue
			}
			bars = append(bars, statusBar{Cell: e.Cell, Kind: pool.kind, Rect: rect.Add(shift), Fill: fill, Faded: faded, Value: pool.value, Maximum: pool.maximum})
		}
	}
	return bars
}

// statusBarPasses keeps run order and merges adjacent equal operations.
// Placement takes the unit cell's lift, camera and zoom.
func (v *Viewer) statusBarPasses() []overlayPass {
	var passes []overlayPass
	var runs []terrain.StatusBarRun
	for _, bar := range v.statusBars() {
		lift := v.cellLift(bar.Cell)
		runs = terrain.AppendStatusBarRuns(runs[:0], bar.Kind, bar.Rect, bar.Value, bar.Maximum, bar.Faded)
		for _, run := range runs {
			r, in := v.placeLifted(lift, run.Rect)
			if !in {
				continue
			}
			i := len(passes) - 1
			if i < 0 || passes[i].Color != run.Color || passes[i].HalfAdd != run.HalfAdd {
				i = len(passes)
				passes = append(passes, overlayPass{Color: run.Color, HalfAdd: run.HalfAdd})
			}
			passes[i].Rects = append(passes[i].Rects, r)
		}
	}
	return passes
}

// HeadlessStatusBarFrame composes the mission viewport on the CPU for a
// release witness: the render tier's terrain raster sampled through the
// camera, the flat structures and the plane's body placements, then this
// frame's overlay passes, the status bars among them. Rectangles cover pixel
// centres and use source-over or packed half-add as declared. under is the
// same composite before the passes. It is a diagnostic and not GPU readback:
// edge sampling can differ, and lighting, shadows, shroud and the HUD are not
// composed.
func (v *Viewer) HeadlessStatusBarFrame() (frame, under *image.RGBA, err error) {
	var ground *terrain.Render
	if v.Mode() == ModeDisplaced {
		ground, err = terrain.CompositeProjectedLit(v.set, v.grid, v.grid.Altitudes, v.Sun(), 1)
	} else {
		ground, err = terrain.CompositeLit(v.set, v.grid, v.grid.Altitudes, v.Sun(), 1)
	}
	if err != nil {
		return nil, nil, err
	}
	under = image.NewRGBA(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH))
	draw.Draw(under, under.Bounds(), image.NewUniform(color.RGBA{A: 0xff}), image.Point{}, draw.Src)
	for y := 0; y < v.cam.ViewH; y++ {
		for x := 0; x < v.cam.ViewW; x++ {
			wx, wy := v.cam.ScreenToWorld(float64(x)+0.5, float64(y)+0.5)
			p := image.Pt(int(math.Floor(wx)), int(math.Floor(wy)))
			if p.In(ground.Image.Bounds()) {
				under.SetRGBA(x, y, ground.Image.RGBAAt(p.X, p.Y))
			}
		}
	}
	for _, p := range v.structurePlacements() {
		if p.Class == nil || !p.Class.Flat || p.Frame == nil {
			continue
		}
		if r, ok := spriteScreenRect(p.Rect(), v.cam); ok {
			editorBlit(under, v.spritePixels(p.Frame), r, false)
		}
	}
	for _, s := range v.planeSprites() {
		if s.Frame != nil {
			editorBlit(under, v.spritePixels(s.Frame), s.screenRect, s.Mirror)
		}
	}
	frame = image.NewRGBA(under.Bounds())
	copy(frame.Pix, under.Pix)
	for _, p := range v.overlayPasses() {
		for _, r := range p.Rects {
			if p.HalfAdd {
				blendStatusBarRect(frame, r, p.Color)
			} else {
				blendScreenRect(frame, r, p.Color)
			}
		}
	}
	return frame, under, nil
}

// blendScreenRect blends one premultiplied colour source-over into the pixels
// whose centres r contains.
func blendScreenRect(dst *image.RGBA, r screenRect, c color.RGBA) {
	x0, x1 := int(math.Ceil(r.X-0.5)), int(math.Ceil(r.X+r.W-0.5))
	y0, y1 := int(math.Ceil(r.Y-0.5)), int(math.Ceil(r.Y+r.H-0.5))
	box := image.Rect(x0, y0, x1, y1).Intersect(dst.Bounds())
	keep := 255 - uint32(c.A)
	over := func(s, d uint8) uint8 { return uint8(uint32(s) + (uint32(d)*keep+127)/255) }
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			d := dst.RGBAAt(x, y)
			dst.SetRGBA(x, y, color.RGBA{R: over(c.R, d.R), G: over(c.G, d.G), B: over(c.B, d.B), A: 0xff})
		}
	}
}
