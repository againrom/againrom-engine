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
	Cell  image.Point
	Kind  terrain.StatusBarKind
	Rect  image.Rectangle
	Fill  int
	Faded bool
}

// statusBars is every bar this frame shows, in the snapshot's order, health
// before mana within a unit.
//
// WHICH UNITS CARRY ONE is the contract's two clauses read off the seam: the
// unit is NOT DEAD — the simulation's own answer, never re-derived from the
// numbers beside it — and the pool's maximum is positive, which a unit with no
// such pool fails. A downed unit keeps its bars; an empty pool draws the caps
// alone. Show Health (DIV-333) decides for an unselected unit, and a unit is
// drawn only while fogGateEntity lets its sprite be seen, so a bar never stands
// over a unit the fog hides.
//
// A SELECTED UNIT'S BARS ARE OPAQUE AND ANY OTHER UNIT'S ARE FADED: caps
// opaque, interior at half opacity, as the owner's screenshot of the original
// shows. The bars take the unit's walking displacement, as its sprite does, and
// not its damage jolt (DIV-488). The unit's class sizes them: a class wider
// than one cell spans its selection box over the body (terrain.StatusBarRect).
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
			fill, ok := terrain.StatusBarFill(rect.Dx(), pool.value, pool.maximum)
			if !ok {
				continue
			}
			bars = append(bars, statusBar{Cell: e.Cell, Kind: pool.kind, Rect: rect.Add(shift), Fill: fill, Faded: faded})
		}
	}
	return bars
}

// statusBarPasses turns statusBars into overlay passes: one pass per colour,
// in the order the colours first occur, each run placed through the lift of
// its unit's cell and the camera exactly as a mark on that cell is placed.
// The runs are native pixels, so the camera's zoom scales a bar with the map.
//
// One pass per colour draws two overlapping bars colour by colour rather than
// unit by unit. Bars overlap only while one unit passes another, and a pass
// per colour keeps the frame at no more than twenty-one bar passes.
func (v *Viewer) statusBarPasses() []overlayPass {
	var passes []overlayPass
	var runs []terrain.StatusBarRun
	for _, bar := range v.statusBars() {
		lift := v.cellLift(bar.Cell)
		runs = terrain.AppendStatusBarRuns(runs[:0], bar.Kind, bar.Rect, bar.Fill, bar.Faded)
		for _, run := range runs {
			r, in := v.placeLifted(lift, run.Rect)
			if !in {
				continue
			}
			i := slices.IndexFunc(passes, func(p overlayPass) bool { return p.Color == run.Color })
			if i < 0 {
				i = len(passes)
				passes = append(passes, overlayPass{Color: run.Color})
			}
			passes[i].Rects = append(passes[i].Rects, r)
		}
	}
	return passes
}

// HeadlessStatusBarFrame composes the mission viewport on the CPU for a
// release witness: the render tier's terrain raster sampled through the
// camera, the flat structures and the plane's body placements, then this
// frame's overlay passes, the status bars among them, each rectangle covering
// the pixels whose centres it contains and blended source-over. under is the
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
			blendScreenRect(frame, r, p.Color)
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
