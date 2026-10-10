package game

import (
	"image"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Every object in flight is one World projectile record: a cast, a staged
// burst, a Fire_Ball burst, a unit's shot at a unit or a structure, and a
// record a LOAD restored. This tier hands the World what the installed
// registries decide at the release (picture, phase count, start point,
// segments), keeps the look a record has no field for, and draws the records
// through one producer (savedProjectileScene).

// flightLook is the presentation a record carries no field for: a path
// picture's figure seed, its link tag and the victims a Prismatic Spray links.
// A LOAD starts every record without one, as the original's LOAD loses a
// record's word list and point array (SAV-1202).
type flightLook struct {
	seed  uint32
	tag   int
	links []image.Point
}

// recordPhases is the phase count a picture's installed sheet holds, the
// driver's frame clock, or 0 for a picture with no usable sheet.
func (mw *mapWorld) recordPhases(picture int) uint16 {
	if sheet := mw.projectiles.Sheet(picture); sheet != nil && sheet.Phases > 0 && sheet.Phases <= 65535 {
		return uint16(sheet.Phases)
	}
	return 0
}

// cellCentre is a cell's centre in 256-per-cell units.
func cellCentre(c image.Point) (int32, int32) {
	return int32(c.X)*256 + 128, int32(c.Y)*256 + 128
}

// releaseCast builds the record one applied cast leaves as (ANIM-147): the
// picture 2 × spell + 8 at the caster class's launch point (MAGIC-261), aimed
// at the caster's target when the picture's row homes and at the target
// cell's centre otherwise (MAGIC-288), with the segment count of the
// producer's picture switch. A picture outside that switch gets 0 segments:
// its record takes an id and retires on its first call (SAV-1204). Teleport
// builds a second record at the destination (MAGIC-265).
func (mw *mapWorld) releaseCast(ev sim.CastEvent, index int) {
	spell := int(ev.Spell)
	picture := data.CastPicture(spell)
	from := image.Pt(int(ev.FromX), int(ev.FromY))
	to := image.Pt(int(ev.ToX), int(ev.ToY))
	class := mw.casterClass(ev.Caster)
	x, y, ok := mw.world.ProjectilePoint(ev.Caster)
	if !ok {
		x, y = cellCentre(from)
	}
	launch := castLaunch(class, ev.Facing, picture)
	aimX, aimY := cellCentre(to)
	rec := sim.CastRecord{Caster: ev.Caster, Picture: int32(picture), Phases: mw.recordPhases(picture),
		X: x + int32(launch.X), Y: y + int32(launch.Y), AimX: aimX, AimY: aimY,
		Dir: int32(ev.Facing >> 4), Segments: int32(data.CastFlight(picture, castDistance(from, to))), Owner: ev.Owner}
	if !ev.AtCell && ev.Target != 0 && mw.projectiles.Homes(picture) {
		rec.Target, rec.HasTarget = ev.Target, true
	}
	look := flightLook{seed: mw.visualCastSeed(ev.Caster, ev.Target, spell, ev.AtCell), tag: index % chainTagCount}
	for _, v := range ev.Victims {
		look.links = append(look.links, image.Pt(int(v.X), int(v.Y)))
	}
	if len(look.links) == 0 && data.CastDrawsPath(picture) {
		look.links = []image.Point{to}
	}
	mw.releaseRecord(rec, look)
	if picture == teleportPicture {
		arrive := image.Point{}
		if class != nil {
			arrive = castSelectionDelta(class)
		}
		rec.X, rec.Y = aimX+int32(arrive.X), aimY+int32(arrive.Y)
		mw.releaseRecord(rec, look)
	}
}

// releaseScriptCast builds the record a client arm builds for a cast whose
// source is a cell (0x8b, 0x8c; SAV-1199): Lightning's direct record of 5
// segments and Prismatic Spray's of 13, from the source cell's centre,
// started at actionphase -1 (MAGIC-281).
func (mw *mapWorld) releaseScriptCast(ev sim.ScriptCastEvent, seed uint32) {
	from := image.Pt(int(ev.FromX), int(ev.FromY))
	to := image.Pt(int(ev.ToX), int(ev.ToY))
	var picture, segments int
	look := flightLook{seed: seed}
	switch mw.world.SpellArm(uint16(ev.Spell)) {
	case 13:
		picture, segments = data.CastPicture(13), 5
		look.links = []image.Point{to}
	case 14:
		picture, segments = data.CastPicture(14), data.CastFlight(data.PicturePathSecond, 0)
		for _, v := range ev.Victims {
			look.links = append(look.links, image.Pt(int(v.X), int(v.Y)))
		}
	default:
		return
	}
	x, y := cellCentre(from)
	aimX, aimY := cellCentre(to)
	mw.releaseRecord(sim.CastRecord{Picture: int32(picture), Phases: mw.recordPhases(picture),
		X: x, Y: y, AimX: aimX, AimY: aimY, Client: true, Segments: int32(segments)}, look)
}

// releaseRecord hands one cast record to the World and keeps its look and the
// trail point its first call started from.
func (mw *mapWorld) releaseRecord(rec sim.CastRecord, look flightLook) {
	var start []image.Point
	rec.PreMove = func(x, y int32) { start = append(start, image.Pt(int(x), int(y))) }
	id, ok := mw.world.ReleaseCast(rec)
	if !ok {
		return
	}
	if mw.flights == nil {
		mw.flights = make(map[uint16]flightLook)
	}
	mw.flights[id] = look
	if data.CastTrailSlot(int(rec.Picture)) >= 0 {
		for _, at := range start {
			mw.shots.appendTrail(id, at)
		}
	}
}

// releaseAreaBursts builds the record a staged stage leaves at each accepted
// cell: the odd picture 2 × spell + 9 of spells 4, 9 and 21, for the sender's
// segment count (ANIM-148). A picture the registry does not hold builds none.
func (mw *mapWorld) releaseAreaBursts(p sim.AreaPaint) {
	picture := data.BurstPicture(int(p.Spell))
	if !mw.projectiles.HasPicture(picture) {
		return
	}
	for _, c := range p.Cells {
		mw.world.ReleaseAreaBurst(sim.AreaBurst{CellX: c.X, CellY: c.Y, Picture: int32(picture),
			Segments: int32(data.BurstLife(picture)), Phases: mw.recordPhases(picture), Owner: p.Owner})
	}
}

// recordPaths is a path record's figures this call, one per link, each as the
// path producer reads it. A record with no look draws one figure to its aim
// point, and a Prismatic Spray record none: LOAD empties its word list
// (SAV-1202).
func (mw *mapWorld) recordPaths(p sim.SavedProjectile, owner uint32) []spellBolt {
	look, ok := mw.flights[p.ID]
	if !ok {
		if int(p.Picture) == data.PicturePathSecond {
			return nil
		}
		look = flightLook{seed: uint32(p.ID) + 1}
		look.links = []image.Point{image.Pt(floorDiv(int(p.ActionX), ui.ShotScale), floorDiv(int(p.ActionY), ui.ShotScale))}
	}
	cell := image.Pt(floorDiv(int(p.X), ui.ShotScale), floorDiv(int(p.Y), ui.ShotScale))
	base := spellBolt{from: cell, picture: int(p.Picture), owner: owner, age: int(p.ActionPhase), phase: int(p.Phase),
		launch: image.Pt(int(p.X)-cell.X*ui.ShotScale-ui.ShotScale/2, int(p.Y)-cell.Y*ui.ShotScale-ui.ShotScale/2)}
	out := make([]spellBolt, 0, len(look.links))
	for k, to := range look.links {
		b := base
		b.to = to
		b.seed = look.seed + uint32(k)*2654435761
		b.tag = (look.tag + k) % chainTagCount
		if len(look.links) > 1 {
			b.tag = k % chainTagCount
		}
		out = append(out, b)
	}
	return out
}

// forgetFlights drops the looks of records the World no longer holds.
func (mw *mapWorld) forgetFlights(records map[uint16]bool) {
	for id := range mw.flights {
		if !records[id] {
			delete(mw.flights, id)
		}
	}
}
