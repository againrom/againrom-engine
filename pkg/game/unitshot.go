package game

import (
	"image"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A ranged swing's own shot. The shooter's units.reg class names the
// projectiles.reg picture its swing releases and the swing tick the shot
// leaves on (ANIM-STATE-023). A shot at a unit is a World projectile record
// built on that tick (SAV-1129, SAV-1130): the World advances it, SAVE writes
// it and a LOAD continues it. The world still resolves the blow on its own
// countdown, which is independent of the record (SAV-1132). A shot at a
// structure is a presentation object kept by this tier: it lives
// ftol(distance)/200 ticks and travels toward its target (ANIM-PROJ-025), and
// it is drawn as a cast's picture is (ANIM-PROJ-026). DIV-1453 names the rules
// here that no claim states.

// unitShotSegment is the distance, in the engine's 256-per-cell units, that
// one tick of a unit shot's life stands for (ANIM-PROJ-025).
const unitShotSegment = 200

const unitShotDeformationPicture = 7

// unitShotRegistryTop is the highest picture whose projectile driver applies
// no damage (SAV-1132); a shot of a higher picture stays a presentation
// object.
const unitShotRegistryTop = 12

// unitShotOffsetScale is the 256-per-cell units that one sprite pixel of a
// class's release offset stands for (SAV-1130).
const unitShotOffsetScale = 8

// unitShot is one structure-bound shot in flight: a cast object's own fields.
type unitShot struct {
	spellBolt
	target    sim.EntityID
	structure bool
}

// unitShots is the map world's memory of ranged swings: the wind-up that
// opened each entity's current run, and the shots already released.
type unitShots struct {
	run    map[sim.EntityID]sim.AttackPhase
	flying []unitShot
	// trail is the last positions of each record whose picture leaves smoke,
	// newest first. It is presentation only: no record stores it.
	trail map[uint16][]image.Point
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

// advanceUnitShots ages the presentation shots in flight, retires the
// finished ones and re-aims the rest at their targets' current cells, then
// releases this tick's: one per physical run, on the tick the swing clock
// reaches the class's release tick, while the target is in reach. A shot at a
// unit becomes a World record; a shot at a structure becomes a presentation
// object. Reach is the swing sound's own gate (advanceSwings): a shot that
// cannot land is not drawn.
//
// It runs after advanceSwings, whose clock and run memory it reads.
func (mw *mapWorld) advanceUnitShots() {
	live := mw.shots.flying[:0]
	for _, s := range mw.shots.flying {
		s.age++
		if s.age >= s.life {
			continue
		}
		if !s.structure {
			if t, ok := mw.entity(s.target); ok {
				s.to = image.Pt(int(t.X), int(t.Y))
			}
		}
		live = append(live, s)
	}
	mw.shots.flying = live

	ents := mw.world.EntityView()
	for _, e := range ents {
		if !mw.shots.physical(e.ID) || !e.Alive() || !e.HasAttackTarget || e.Reach <= 1 {
			continue
		}
		picture, delay, drawn := mw.classShot(mw.spellClientClass(e.ID, e.Class))
		record := e.AttackTargetKind == sim.AttackTargetUnit && picture <= unitShotRegistryTop
		at := delay
		if record {
			at = unitShotSwingTick(delay)
		}
		if !drawn || mw.swing[e.ID] != at || !mw.swingTargetInReach(ents, e) {
			continue
		}
		if record {
			mw.releaseUnitShot(e, picture, unitShotLate(delay))
			continue
		}
		to, ok := mw.attackTargetCell(e)
		if !ok {
			continue
		}
		from := image.Pt(int(e.X), int(e.Y))
		life := castDistance(from, to) / unitShotSegment
		if life < 1 {
			life = 1
		}
		mw.shots.flying = append(mw.shots.flying, unitShot{
			spellBolt: spellBolt{from: from, to: to, picture: picture, owner: e.Owner,
				life: life, facing: e.Facing},
			target:    e.AttackTarget,
			structure: e.AttackTargetKind == sim.AttackTargetStructure,
		})
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
// phase count, the release offset its facing selects and the direction the
// shot starts toward.
func (mw *mapWorld) releaseUnitShot(e sim.Entity, picture, late int) {
	var phases uint16
	if sheet := mw.projectiles.Sheet(picture); sheet != nil && sheet.Phases > 0 && sheet.Phases <= 65535 {
		phases = uint16(sheet.Phases)
	}
	dx, dy := mw.shotOffset(mw.spellClientClass(e.ID, e.Class), e.Facing)
	if !mw.world.ReleaseUnitShot(sim.UnitShot{Shooter: e.ID, Picture: int32(picture), Phases: phases,
		OffsetX: int32(dx), OffsetY: int32(dy), Dir: shotDirection, Late: late}) {
		return
	}
	if data.CastTrailSlot(picture) >= 0 {
		if mw.shots.trail == nil {
			mw.shots.trail = make(map[uint16][]image.Point)
		}
		mw.shots.trail[mw.world.SavedProjectiles().FreeIndex-1] = nil
	}
}

// shotDirection is the direction leaf of a shot starting along (dx, dy): the
// sheet facing that vector selects, in the leaf's own bias (the draw subtracts
// 8 and keeps four bits).
func shotDirection(dx, dy int) int32 {
	return int32(terrain.EffectFacing(dx, dy)+8) & 0xf
}

// shotOffset is the displacement of a class's release point from its
// shooter's own point, in 256-per-cell units (SAV-1130): the class's offset
// pair for the facing, less the class centre, eight units per pixel. A class
// declaring no complete offset array releases from the shooter's point.
func (mw *mapWorld) shotOffset(class int32, facing uint8) (dx, dy int) {
	c := mw.units.Classes[class]
	if c == nil {
		return 0, 0
	}
	at := 2 * ((sim.FacingDir(facing) + 4) & 7)
	if len(c.ShootOffset) < at+2 {
		return 0, 0
	}
	return (c.ShootOffset[at] - c.CenterX) * unitShotOffsetScale, (c.ShootOffset[at+1] - c.CenterY) * unitShotOffsetScale
}

// advanceShotTrails appends this tick's position of every record released
// here whose picture leaves smoke to its trail, bounded at the trail length,
// and forgets retired records. A record restored from a SAV has no history.
func (mw *mapWorld) advanceShotTrails() {
	records := map[uint16]sim.SavedProjectile{}
	for _, p := range mw.world.SavedProjectiles().Items {
		records[p.ID] = p
	}
	for id, trail := range mw.shots.trail {
		p, ok := records[id]
		if !ok {
			delete(mw.shots.trail, id)
			continue
		}
		points := append([]image.Point{{X: int(p.X), Y: int(p.Y)}}, trail...)
		if len(points) > data.CastTrailLength {
			points = points[:data.CastTrailLength]
		}
		mw.shots.trail[id] = points
	}
}

// unitShotDraws is every presentation shot in flight as the viewer draws it, through the
// cast object's own sprite and trail producers, so a picture reads the same
// from a swing as from a book. A shot of life N stands (age+1)/N of the way
// from the shooter's hand and on its target at its last drawn tick.
func (mw *mapWorld) unitShotDraws() []ui.SpellBolt {
	var out []ui.SpellBolt
	for _, s := range mw.shots.flying {
		b := s.spellBolt
		pos := castShotPoint(b.from, b.to, b.age+1, b.life, b.facing)
		if d, ok := mw.spellDraw(b.picture, b.from, b.to, pos, b.age, b.owner); ok {
			out = append(out, d)
		}
		out = append(out, mw.trailDraws(b)...)
	}
	return out
}
