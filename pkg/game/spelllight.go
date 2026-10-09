package game

import (
	"image"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Projectile light: pictures 10, 12, 13, 34 and 36 stamp the vertex light
// grid, rebuilt on every push (MAGIC-269, MAGIC-272). Client draw state only
// (MAGIC-274). Stamps overwrite in draw order (MAGIC-270, DIV-2661).

// boltLightStep is the stamp byte per phase of a path picture (MAGIC-270).
const boltLightStep = 10

// boltLightPhases is a path object's phase on calls 1..13 of normal caster
// construction; the direct route's five calls take the first five (MAGIC-272).
var boltLightPhases = [...]uint8{4, 3, 2, 1, 0, 1, 2, 1, 0, 1, 2, 3, 4}

// boltLightPhase is the phase at age; a later age holds the last value.
func boltLightPhase(age int) uint8 {
	return boltLightPhases[min(max(age, 0), len(boltLightPhases)-1)]
}

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
// at u8(10*phase) (MAGIC-270). The points are the drawn figure's (DIV-2659).
func pathLightStamps(out []ui.LightStamp, points []image.Point, phase uint8, bounds sim.Bounds) []ui.LightStamp {
	level := uint8(boltLightStep * int(phase))
	for _, p := range points {
		c := lightCell(p)
		if c.X < 0 || c.Y < 0 || c.X >= int(bounds.Width) || c.Y >= int(bounds.Height) {
			continue
		}
		for _, v := range [4]image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
			out = append(out, ui.LightStamp{Vertex: c.Add(v), Level: level})
		}
	}
	return out
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

// boltLightStamps stamps one spellBolt where and as it is drawn.
func (mw *mapWorld) boltLightStamps(out []ui.LightStamp, b spellBolt, bounds sim.Bounds) []ui.LightStamp {
	if b.picture == healingPicture || b.age < b.delay {
		return out
	}
	if data.CastDrawsPath(b.picture) {
		_, _, points := mw.pathFigure(b)
		return pathLightStamps(out, points, boltLightPhase(b.age), bounds)
	}
	age, life := b.age-b.delay, b.life-b.delay
	num := min(age+1, life)
	phase := -1
	if b.picture == fireBallBurstPicture {
		phase = mw.burstLightPhase(age)
	}
	return pictureLightStamps(out, b.picture, lightCell(castShotPoint(b.from, b.to, num, life, b.launch)), phase)
}

// burstLightPhase is the explosion phase spellDraw draws at age, or -1.
func (mw *mapWorld) burstLightPhase(age int) int {
	sheet := mw.projectiles.Sheet(fireBallBurstPicture)
	if sheet == nil {
		return -1
	}
	phase, ok := terrain.EffectPhase(sheet.Clock, age, sheet.Phases)
	if !ok {
		return -1
	}
	if age == data.BurstLife(fireBallBurstPicture)-1 {
		phase = sheet.Phases - 1
	}
	return phase
}

// objectLightStamps is this push's stamps: objects in draw order, then Wall
// of Fire, written after the object calls (MAGIC-272).
func (mw *mapWorld) objectLightStamps(ents []sim.Entity) []ui.LightStamp {
	if mw.projectiles == nil {
		return mw.wallFireLightStamps(nil)
	}
	bounds := mw.world.Bounds()
	var out []ui.LightStamp
	for _, b := range mw.bolts {
		out = mw.boltLightStamps(out, b, bounds)
	}
	out = mw.savedProjectileLightStamps(out)
	for _, b := range mw.weaponBolts(ents) {
		if data.CastDrawsPath(b.picture) {
			out = mw.boltLightStamps(out, b, bounds)
			continue
		}
		// weaponBoltDraws draws a flying picture at swing/charge, not (age+1).
		out = pictureLightStamps(out, b.picture, lightCell(castShotPoint(b.from, b.to, b.age, b.life, b.launch)), -1)
	}
	for _, s := range mw.shots.flying {
		b := s.spellBolt
		out = pictureLightStamps(out, b.picture, lightCell(castShotPoint(b.from, b.to, b.age+1, b.life, b.launch)), -1)
	}
	return mw.wallFireLightStamps(out)
}

// savedProjectileLightStamps stamps the armed World projectile records.
func (mw *mapWorld) savedProjectileLightStamps(out []ui.LightStamp) []ui.LightStamp {
	d := mw.world.SavedWorldEffectDrivers()
	if d == nil {
		return out
	}
	armed := map[uint16]bool{}
	for _, row := range d.Projectiles {
		if !row.Retired {
			armed[row.ID] = true
		}
	}
	for _, p := range mw.world.SavedProjectiles().Items {
		if !armed[p.ID] {
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
