package game

import (
	"image"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// A ranged swing's own shot. The shooter's units.reg class names the
// projectiles.reg picture its swing releases and the swing tick the shot
// leaves on (ANIM-STATE-023). The shot is a World projectile record built on
// that tick (SAV-1129, SAV-1130), at a unit or at a structure (SAV-1197): the
// World advances it, SAVE writes it and a LOAD continues it. The world still
// resolves the blow on its own countdown, which is independent of the record
// (SAV-1132). DIV-1453 names the rules here that no claim states.

const unitShotDeformationPicture = 7

// unitShots is the map world's memory of ranged swings and of the records
// in flight: the wind-up that opened each entity's current run, and each
// smoke-leaving record's trail.
type unitShots struct {
	run map[sim.EntityID]sim.AttackPhase
	// trail is each smoke-leaving record's trail, oldest first: the point
	// each driver call started from, at most six (ANIM-140). No record stores
	// it, so a LOAD starts it empty (SAV-1193).
	trail map[uint16][]image.Point
	// premove is each smoke-leaving record's point before this tick's step.
	premove map[uint16]image.Point
}

// begin records the wind-up phase that opened id's run; end forgets it.
func (s *unitShots) begin(id sim.EntityID, phase sim.AttackPhase) {
	if s.run == nil {
		s.run = make(map[sim.EntityID]sim.AttackPhase)
	}
	s.run[id] = phase
}

func (s *unitShots) end(id sim.EntityID) { delete(s.run, id) }

// physical reports whether id's current run is a physical swing: one that
// opened in the charging wind-up and not in the casting one.
func (s *unitShots) physical(id sim.EntityID) bool { return s.run[id] == sim.AttackCharging }

// classShot is the picture class's ranged swing releases, the swing tick it
// leaves on, and whether this build draws that shot itself: from a loaded
// sheet, or as background deformation. A class naming no projectile, or one whose
// sheet did not load, answers drawn false and keeps the orange mark
// (entityDraws).
func (mw *mapWorld) classShot(class int32) (picture, delay int, drawn bool) {
	if mw.units == nil {
		return 0, 0, false
	}
	c := mw.units.Classes[class]
	if c == nil || c.Projectile == 0 {
		return 0, 0, false
	}
	drawn = c.Projectile == unitShotDeformationPicture && mw.projectiles.HasPicture(c.Projectile) || mw.projectiles.Sheet(c.Projectile) != nil
	return c.Projectile, c.ShootDelay, drawn
}

// advanceUnitShots releases this tick's shots: one per physical run, on the
// tick the swing clock reaches the class's release tick, while the target is
// in reach. Reach is the swing sound's own gate (advanceSwings): a shot that
// cannot land is not built.
//
// It runs after advanceSwings, whose clock and run memory it reads.
func (mw *mapWorld) advanceUnitShots() {
	ents := mw.world.EntityView()
	for _, e := range ents {
		if !mw.shots.physical(e.ID) || !e.Alive() || !e.HasAttackTarget || e.Reach <= 1 {
			continue
		}
		picture, delay, drawn := mw.classShot(mw.spellClientClass(e.ID, e.Class))
		if !drawn || mw.swing[e.ID] != unitShotSwingTick(delay) || !mw.swingTargetInReach(ents, e) {
			continue
		}
		mw.releaseUnitShot(e, picture, unitShotLate(delay))
	}
	mw.advanceShotTrails()
}

// unitShotSwingTick is the swing count a unit shot's record is built on. The
// clock's zero is the tick after the original's swing start, and the original
// builds the record ShootDelay ticks after it (SAV-1153).
func unitShotSwingTick(delay int) int { return max(delay-1, 0) }

// unitShotLate is the driver calls a record has missed when built: ShootDelay 0
// is created a tick before the clock's zero.
func unitShotLate(delay int) int {
	if delay == 0 {
		return 1
	}
	return 0
}

// releaseUnitShot builds the World record of e's shot: the class's picture and
// phase count and the release offset its facing selects. The World aims it.
func (mw *mapWorld) releaseUnitShot(e sim.Entity, picture, late int) {
	var phases uint16
	if sheet := mw.projectiles.Sheet(picture); sheet != nil && sheet.Phases > 0 && sheet.Phases <= 65535 {
		phases = uint16(sheet.Phases)
	}
	dx, dy := mw.shotOffset(mw.spellClientClass(e.ID, e.Class), e.Facing)
	id := mw.world.SavedProjectiles().FreeIndex
	var start []image.Point
	premove := func(x, y int32) { start = append(start, image.Pt(int(x), int(y))) }
	if !mw.world.ReleaseUnitShot(sim.UnitShot{Shooter: e.ID, Picture: int32(picture), Phases: phases,
		OffsetX: int32(dx), OffsetY: int32(dy), Late: late, PreMove: premove}) {
		return
	}
	if data.CastTrailSlot(picture) >= 0 {
		for _, at := range start {
			mw.shots.appendTrail(id, at)
		}
	}
}

// shotOffset is the displacement of a class's release point from its
// shooter's own point, in 256-per-cell units (SAV-1188): the cast producer's
// own class offset for the facing. A class with no ShootOffset array releases
// from the shooter's point.
func (mw *mapWorld) shotOffset(class int32, facing uint8) (dx, dy int) {
	off, _ := classShootOffset(mw.units.Classes[class], facing)
	return off.X, off.Y
}

// noteShotPreMoves records, before the world steps, the point each armed
// record of a smoke-leaving picture will start this tick's driver call from.
func (mw *mapWorld) noteShotPreMoves() {
	clear(mw.shots.premove)
	d := mw.world.SavedWorldEffectDrivers()
	if d == nil {
		return
	}
	armed := map[uint16]bool{}
	for _, row := range d.Projectiles {
		armed[row.ID] = !row.Retired
	}
	for _, p := range mw.world.SavedProjectiles().Items {
		if armed[p.ID] && data.CastTrailSlot(int(p.Picture)) >= 0 {
			if mw.shots.premove == nil {
				mw.shots.premove = make(map[uint16]image.Point)
			}
			mw.shots.premove[p.ID] = image.Pt(int(p.X), int(p.Y))
		}
	}
}

// appendTrail appends one point to a record's trail, dropping the oldest at
// the trail length (ANIM-140).
func (s *unitShots) appendTrail(id uint16, at image.Point) {
	if s.trail == nil {
		s.trail = make(map[uint16][]image.Point)
	}
	points := s.trail[id]
	if len(points) >= data.CastTrailLength {
		points = slices.Delete(points, 0, 1)
	}
	s.trail[id] = append(points, at)
}

// advanceShotTrails appends the pre-move point of each record that made a
// driver call this tick and forgets the trails of retired records.
func (mw *mapWorld) advanceShotTrails() {
	records := map[uint16]bool{}
	for _, p := range mw.world.SavedProjectiles().Items {
		records[p.ID] = true
	}
	for id, at := range mw.shots.premove {
		if records[id] {
			mw.shots.appendTrail(id, at)
		}
	}
	clear(mw.shots.premove)
	for id := range mw.shots.trail {
		if !records[id] {
			delete(mw.shots.trail, id)
		}
	}
	mw.forgetFlights(records)
}
