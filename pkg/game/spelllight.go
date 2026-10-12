package game

import (
	"image"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Projectile light: the identifiers the game's set gives a light kind stamp
// the vertex light grid, rebuilt on every push (MAGIC-269, MAGIC-272;
// R2-ENGINE-448). Client draw state only (MAGIC-274). Stamps overwrite in
// draw order (MAGIC-270, DIV-2661).

// pathLightStamps writes the four vertices of each in-map path point's cell
// at u8(step*phase), the set's PathLightStep (MAGIC-270). The points are the stored figure's native
// display points; the column is x>>5 and the row is the ground row under the
// point (DIV-2659).
func (mw *mapWorld) pathLightStamps(out []ui.LightStamp, points []image.Point, phase int, bounds sim.Bounds) []ui.LightStamp {
	level := mw.spells().PathLightStep * uint8(phase)
	for _, p := range points {
		c, ok := mw.displayLightCell(p)
		if !ok || c.X < 0 || c.Y < 0 || c.X >= int(bounds.Width) || c.Y >= int(bounds.Height) {
			continue
		}
		for _, v := range [4]image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
			out = append(out, ui.LightStamp{Vertex: c.Add(v), Level: level})
		}
	}
	return out
}

// displayLightCell is the cell under a native display point: column x>>5 and
// the viewer's ground row, or y>>5 with no viewer.
func (mw *mapWorld) displayLightCell(p image.Point) (image.Point, bool) {
	col := p.X >> 5
	if mw.view == nil {
		return image.Pt(col, p.Y>>5), true
	}
	row, ok := mw.view.DisplayRow(p.X, p.Y)
	return image.Pt(col, row), ok
}

// pointLightStamps is the point helper (MAGIC-271): for i, j in 0..radius with
// i*i+j*j < radius*(radius+1) (1 at radius 0) it writes x in {c+1+i, c-i},
// y in {r+1+j, r-j}. Dynamic lighting off skips it (MAGIC-273).
func pointLightStamps(out []ui.LightStamp, cell image.Point, radius int, level uint8) []ui.LightStamp {
	limit := radius * (radius + 1)
	if radius == 0 {
		limit = 1
	}
	for j := 0; j <= radius; j++ {
		for i := 0; i <= radius; i++ {
			if i*i+j*j >= limit {
				continue
			}
			for _, y := range [2]int{cell.Y + 1 + j, cell.Y - j} {
				for _, x := range [2]int{cell.X + 1 + i, cell.X - i} {
					out = append(out, ui.LightStamp{Vertex: image.Pt(x, y), Level: level, Point: true})
				}
			}
		}
	}
	return out
}

// lightCell is the cell holding a cell-relative ShotScale point.
func lightCell(p image.Point) image.Point {
	half := ui.ShotScale / 2
	return image.Pt(floorDiv(p.X+half, ui.ShotScale), floorDiv(p.Y+half, ui.ShotScale))
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// pictureLightStamps stamps one non-path object at cell by its light kind in
// set: a fixed point stamp, or the burst step of its phase.
func pictureLightStamps(out []ui.LightStamp, set *data.SpellPresentation, picture int, cell image.Point, phase int) []ui.LightStamp {
	switch e := set.Effect(int32(picture)); e.LightKind {
	case data.LightPoint:
		return pointLightStamps(out, cell, e.Light.Radius, e.Light.Level)
	case data.LightBurst:
		if step, ok := set.BurstLightAt(phase); ok {
			return pointLightStamps(out, cell, step.Radius, step.Level)
		}
	}
	return out
}

// objectLightStamps is this push's stamps: objects in draw order, then Wall
// of Fire, written after the object calls (MAGIC-272).
func (mw *mapWorld) objectLightStamps(ents []sim.Entity) []ui.LightStamp {
	if mw.projectiles == nil {
		return mw.wallFireLightStamps(nil)
	}
	bounds := mw.world.Bounds()
	var out []ui.LightStamp
	out = mw.savedProjectileLightStamps(out, bounds)
	return mw.wallFireLightStamps(out)
}

// savedProjectileLightStamps stamps the armed World projectile records, a
// path record at each of its figures' points.
func (mw *mapWorld) savedProjectileLightStamps(out []ui.LightStamp, bounds sim.Bounds) []ui.LightStamp {
	records, owners := mw.armedRecords()
	set := mw.spells()
	for _, p := range records {
		if set.DrawsPath(p.Picture) {
			if set.Effect(p.Picture).LightKind != data.LightPath {
				continue
			}
			for _, b := range mw.recordPaths(p, owners[p.ID]) {
				_, _, points := mw.pathFigure(b)
				out = mw.pathLightStamps(out, points, int(p.Phase), bounds)
			}
			continue
		}
		cell := image.Pt(floorDiv(int(p.X), ui.ShotScale), floorDiv(int(p.Y), ui.ShotScale))
		out = pictureLightStamps(out, set, int(p.Picture), cell, int(p.Phase))
	}
	return out
}

// wallFireLightStamps stamps each cell of a retained effect whose overlay
// identifier the set lights as a cell (the first game's Wall of Fire,
// MAGIC-UNITLIGHT-057, footprint MAGIC-271) at its flicker level.
func (mw *mapWorld) wallFireLightStamps(out []ui.LightStamp) []ui.LightStamp {
	set := mw.spells()
	for _, effect := range mw.world.CellEffects() {
		picture, _ := set.OverlayPicture(int(effect.Spell))
		e := set.Effect(int32(picture))
		if e.LightKind != data.LightCell {
			continue
		}
		for _, c := range effect.Cells {
			cell := image.Pt(int(c[0]), int(c[1]))
			out = pointLightStamps(out, cell, e.Light.Radius, wallFireLightLevel(mw.scene, cell))
		}
	}
	return out
}
