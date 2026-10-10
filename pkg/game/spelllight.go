package game

import (
	"image"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Projectile light: pictures 10, 12, 13, 34 and 36 stamp the vertex light
// grid, rebuilt on every push (MAGIC-269, MAGIC-272). Client draw state only
// (MAGIC-274). Stamps overwrite in draw order (MAGIC-270, DIV-2661).

// boltLightStep is the stamp byte per phase of a path picture (MAGIC-270).
const boltLightStep = 10

// Fire Arrow and Fire Ball flight stamp at a fixed level (MAGIC-271).
const (
	flightLightLevel     = 16
	fireArrowLightRadius = 0
	fireBallLightRadius  = 1
	fireArrowPicture     = 10
	fireBallPicture      = 12
)

// explosionLight is the explosion's radius and level by phase (MAGIC-271).
var explosionLight = [...]struct {
	radius int
	level  uint8
}{{1, 16}, {2, 8}, {3, 0}, {3, 0}, {3, 8}, {3, 16}, {3, 24}, {3, 32}, {3, 40}, {3, 46}, {3, 46}}

// wallFireLightRadius is a Wall of Fire cell's stamp (MAGIC-UNITLIGHT-057,
// footprint MAGIC-271).
const wallFireLightRadius = 1

// pathLightStamps writes the four vertices of each in-map path point's cell
// at u8(10*phase) (MAGIC-270). The points are the stored figure's native
// display points; the column is x>>5 and the row is the ground row under the
// point (DIV-2659).
func (mw *mapWorld) pathLightStamps(out []ui.LightStamp, points []image.Point, phase int, bounds sim.Bounds) []ui.LightStamp {
	level := uint8(boltLightStep * phase)
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

// pictureLightStamps stamps one non-path object at cell.
func pictureLightStamps(out []ui.LightStamp, picture int, cell image.Point, phase int) []ui.LightStamp {
	switch picture {
	case fireArrowPicture:
		return pointLightStamps(out, cell, fireArrowLightRadius, flightLightLevel)
	case fireBallPicture:
		return pointLightStamps(out, cell, fireBallLightRadius, flightLightLevel)
	case fireBallBurstPicture:
		if phase >= 0 && phase < len(explosionLight) {
			e := explosionLight[phase]
			return pointLightStamps(out, cell, e.radius, e.level)
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
	for _, p := range records {
		if data.CastDrawsPath(int(p.Picture)) {
			for _, b := range mw.recordPaths(p, owners[p.ID]) {
				_, _, points := mw.pathFigure(b)
				out = mw.pathLightStamps(out, points, int(p.Phase), bounds)
			}
			continue
		}
		cell := image.Pt(floorDiv(int(p.X), ui.ShotScale), floorDiv(int(p.Y), ui.ShotScale))
		out = pictureLightStamps(out, int(p.Picture), cell, int(p.Phase))
	}
	return out
}

// wallFireLightStamps stamps each Wall of Fire cell at its flicker level.
func (mw *mapWorld) wallFireLightStamps(out []ui.LightStamp) []ui.LightStamp {
	for _, effect := range mw.cellEffectsByArm() {
		if effect.Spell != 3 {
			continue
		}
		for _, c := range effect.Cells {
			cell := image.Pt(int(c[0]), int(c[1]))
			out = pointLightStamps(out, cell, wallFireLightRadius, wallFireLightLevel(mw.scene, cell))
		}
	}
	return out
}
