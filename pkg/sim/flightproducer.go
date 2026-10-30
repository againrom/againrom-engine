package sim

import "slices"

// FlightInputs is the one record producer's immutable input.
type FlightInputs struct {
	Pictures  map[int32]FlightPicture
	Classes   map[int32]FlightClass
	ClassRule FlightClassRule
	Profile   FlightProfile
}

// FlightProfile is how the games' producers differ; zero is the first game's.
// A moved caster's Teleport start record stands where he stands, not at his
// cast cell (MAGIC-TELEPORT-358), and a shot needs a picture the client draws.
type FlightProfile struct {
	Release              FlightRelease
	StartFromMovedCaster bool
	ShotsNeedArt         bool
}

// FlightRelease is a game's release clock: the step's, a shot on the actor's
// FlightRun, or the client's, after its observation and on its swing clock.
type FlightRelease uint8

const (
	ReleaseInStep FlightRelease = iota
	ReleaseAtClient
)

type FlightPicture struct {
	Present                  bool
	Phases                   uint16
	Homes                    bool
	LifeTicks, LifeDivisor   int32
	LifePhases, SecondRecord bool
	CellLife                 int32
	Drawn                    bool
}

type FlightClass struct {
	Projectile, ShootDelay, TileSize int32
	CenterX, CenterY                 int32
	SelectionWidth, SelectionHeight  int32
	ShootOffset                      []int32
}

// FlightClassRule answers an actor's flight class from his current values.
type FlightClassRule func(Entity) int32

// FlightRun is the physical release clock of the current attack run.
type FlightRun struct {
	Active      bool
	Kind, Phase AttackPhase
	Elapsed     uint32
}

// FlightStart is a release driver call's starting point for a cosmetic trail.
type FlightStart struct {
	ID      uint16
	Picture int32
	X, Y    int32
}

// Released is what a release outside the step observed.
type Released struct {
	FlightStarts    []FlightStart
	ProjectileMarks []ProjectileMark
}

// ShotClock answers a unit's swing count and whether his run is physical.
type ShotClock func(EntityID) (swing int, physical bool)

func (w *World) SetFlightInputs(in FlightInputs) {
	copy := FlightInputs{Pictures: make(map[int32]FlightPicture, len(in.Pictures)), Classes: make(map[int32]FlightClass, len(in.Classes)), ClassRule: in.ClassRule, Profile: in.Profile}
	for id, p := range in.Pictures {
		copy.Pictures[id] = p
	}
	for id, c := range in.Classes {
		c.ShootOffset = slices.Clone(c.ShootOffset)
		copy.Classes[id] = c
	}
	w.flightInputs = &copy
	w.SetBurstPhases(copy.Pictures[fireBallBurstPicture()].Phases)
}

func (w *World) HasFlightInputs() bool { return w.flightInputs != nil }

func (w *World) releasesInStep() bool {
	return w.flightInputs != nil && w.flightInputs.Profile.Release == ReleaseInStep
}

// SetClientClass stores the class a hero's visible equipment selects
// (HERO-APPEAR-042), as the front end derives it after an equipment change or a
// LOAD. An absent actor and one whose class is fixed refuse it.
func (w *World) SetClientClass(id EntityID, class int32) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 || w.entities[i].ClientClassFixed {
		return false
	}
	w.entities[i].Class = class
	return true
}

// EquipmentClassRule answers the class an actor's worn items select once a
// step's equipment command has applied; false keeps his stored class.
type EquipmentClassRule func(w *World, id EntityID) (int32, bool)

// SetEquipmentClassRule binds the rule the following steps call after each
// Equip, Unequip and DropWorn command; nil unbinds it.
func (w *World) SetEquipmentClassRule(rule EquipmentClassRule) { w.equipmentClass = rule }

// reclassAfterEquipment stores the class the bound rule answers for entity i
// after an equipment command, so the rest of the step, its flight producer
// included, reads the class the new equipment selects (ANIM-107). A fixed
// class keeps its value, as SetClientClass refuses it.
func (w *World) reclassAfterEquipment(i int) {
	if w.equipmentClass == nil || i < 0 || i >= len(w.entities) || w.entities[i].ClientClassFixed {
		return
	}
	if class, ok := w.equipmentClass(w, w.entities[i].ID); ok {
		w.entities[i].Class = class
	}
}

func (w *World) flightClass(id EntityID) FlightClass {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return FlightClass{}
	}
	e := w.entities[i]
	class := e.Class
	if rule := w.flightInputs.ClassRule; rule != nil {
		class = rule(e)
	}
	return w.flightInputs.Classes[class]
}

// shotOffset is 8*(the facing's ShootOffset pair-Center) (MAGIC-261,
// SAV-1188, DIV-2656); false without a complete pair.
func (c FlightClass) shotOffset(facing uint8) (int32, int32, bool) {
	i := (int(facing>>4) - 8) & 14
	if len(c.ShootOffset) < i+2 {
		return 0, 0, false
	}
	return 8 * (c.ShootOffset[i] - c.CenterX), 8 * (c.ShootOffset[i+1] - c.CenterY), true
}

func (c FlightClass) selectionOffset() (int32, int32) {
	return c.SelectionWidth/2 - c.CenterX, c.SelectionHeight/2 - c.CenterY
}

// castOffset is the shot offset, or the selection fallback without one and
// for a two-record picture, plus 128*(TileSize-1) (DIV-2663).
func (c FlightClass) castOffset(facing uint8, second bool) (int32, int32) {
	x, y, ok := c.shotOffset(facing)
	if !ok || second {
		x, y = c.selectionOffset()
	}
	a := (max(c.TileSize, 1) - 1) * 128
	return x + a, y + a
}

// shotTick is the run's swing count a class's shot leaves on (SAV-1153).
func (c FlightClass) shotTick() int64 { return max(int64(c.ShootDelay)-1, 0) }

func (p FlightPicture) segments(dx, dy int32) int32 {
	if p.LifeDivisor > 0 {
		return max(1, int32(256*isqrt64(int64(dx)*int64(dx)+int64(dy)*int64(dy)))/p.LifeDivisor)
	}
	if p.LifePhases {
		return int32(p.Phases)
	}
	return p.LifeTicks
}

func (w *World) flightPreMove(obs *castObs, id uint16, picture int32) func(int32, int32) {
	return func(x, y int32) {
		obs.flightStarts = append(obs.flightStarts, FlightStart{ID: id, Picture: picture, X: x, Y: y})
	}
}

// flightMarks keeps first-call marks no step reports.
func (w *World) flightMarks(obs *castObs) func(ProjectileMark) {
	if w.castObservation != nil {
		return nil
	}
	return func(m ProjectileMark) { obs.projectileMarks = append(obs.projectileMarks, m) }
}

func (w *World) produceCast(ev *CastEvent, obs *castObs) {
	ev.Produced = true
	if ev.Rider {
		return
	}
	picture := int32(ev.Spell)*2 + 8
	p := w.flightInputs.Pictures[picture]
	c := w.flightClass(ev.Caster)
	x, y, ok := w.ProjectilePoint(ev.Caster)
	ci := indexOfEntity(w.entities, ev.Caster)
	moved := ci < 0 || w.entities[ci].X != ev.FromX || w.entities[ci].Y != ev.FromY
	if !ok || p.SecondRecord && !w.flightInputs.Profile.StartFromMovedCaster && moved {
		x, y = ev.FromX*256+128, ev.FromY*256+128
	}
	dx, dy := c.castOffset(ev.Facing, p.SecondRecord)
	rec := CastRecord{Caster: ev.Caster, Picture: picture, Phases: p.Phases, X: x + dx, Y: y + dy, AimX: ev.ToX*256 + 128, AimY: ev.ToY*256 + 128,
		Dir: ProjectileDirection(ev.ToX-ev.FromX, ev.ToY-ev.FromY), Segments: p.segments(ev.ToX-ev.FromX, ev.ToY-ev.FromY), Owner: ev.Owner, Mark: w.flightMarks(obs)}
	target, _ := w.Entity(ev.Target)
	if !ev.AtCell && p.Homes && (ev.Target != 0 || ProjectileTargetKey(target) != 0) {
		rec.Target, rec.HasTarget = ev.Target, true
	}
	release := func() {
		rec.PreMove = w.flightPreMove(obs, w.savedProjectiles.FreeIndex, picture)
		if id, ok := w.ReleaseCast(rec); ok {
			ev.Records = append(ev.Records, id)
		}
	}
	release()
	if p.SecondRecord {
		dx, dy = c.selectionOffset()
		rec.X, rec.Y = rec.AimX+dx, rec.AimY+dy
		release()
	}
}

func (w *World) produceScriptCast(ev *ScriptCastEvent, obs *castObs) {
	ev.Produced = true
	picture := int32(ev.Spell)*2 + 8
	p := w.flightInputs.Pictures[picture]
	segments := int32(0)
	switch w.SpellArm(ev.Spell) {
	case 13:
		segments = 5
	case 14:
		segments = p.segments(0, 0)
	default:
		return
	}
	rec := CastRecord{Picture: picture, Phases: p.Phases, X: ev.FromX*256 + 128, Y: ev.FromY*256 + 128, AimX: ev.ToX*256 + 128, AimY: ev.ToY*256 + 128, Client: true, Segments: segments}
	rec.PreMove = w.flightPreMove(obs, w.savedProjectiles.FreeIndex, picture)
	if id, ok := w.ReleaseCast(rec); ok {
		ev.Records = append(ev.Records, id)
	}
}

func (w *World) produceAreaBursts(paint *AreaPaint) {
	paint.Produced = true
	picture := int32(paint.Spell)*2 + 9
	p := w.flightInputs.Pictures[picture]
	if !p.Present {
		return
	}
	for _, c := range paint.Cells {
		w.ReleaseAreaBurst(AreaBurst{CellX: c.X, CellY: c.Y, Picture: picture, Segments: p.CellLife, Phases: p.Phases, Owner: paint.Owner})
	}
}

func (w *World) produceEvents(obs *castObs) {
	for i := range obs.scriptCasts {
		if !obs.scriptCasts[i].Produced {
			w.produceScriptCast(&obs.scriptCasts[i], obs)
		}
	}
	for i := range obs.casts {
		if !obs.casts[i].Produced {
			w.produceCast(&obs.casts[i], obs)
		}
	}
	for i := range obs.paints {
		if !obs.paints[i].Produced {
			w.produceAreaBursts(&obs.paints[i])
		}
	}
}

// ReleaseCasts builds the records of observed casts no step built.
func (w *World) ReleaseCasts(events []CastEvent) Released {
	return w.releaseObserved(castObs{casts: events})
}

// ReleaseScriptCasts is ReleaseCasts for cell-source casts.
func (w *World) ReleaseScriptCasts(events []ScriptCastEvent) Released {
	return w.releaseObserved(castObs{scriptCasts: events})
}

// ReleaseAreaPaints is ReleaseCasts for staged cells.
func (w *World) ReleaseAreaPaints(paints []AreaPaint) {
	w.releaseObserved(castObs{paints: paints})
}

func (w *World) releaseObserved(obs castObs) Released {
	if w.flightInputs == nil {
		return Released{}
	}
	w.produceEvents(&obs)
	return Released{FlightStarts: obs.flightStarts, ProjectileMarks: obs.projectileMarks}
}

// ReleaseClientShots builds each shot the client's swing clock puts on its
// class's release tick, for a World that releases at the client.
func (w *World) ReleaseClientShots(clock ShotClock) Released {
	if w.flightInputs == nil || w.flightInputs.Profile.Release != ReleaseAtClient || clock == nil {
		return Released{}
	}
	var obs castObs
	for i := range w.entities {
		e := w.entities[i]
		if w.HasEffectArm(e.ID, 20) || !e.Alive() || !e.HasAttackTarget || e.Reach <= 1 {
			continue
		}
		swing, physical := clock(e.ID)
		if !physical {
			continue
		}
		c := w.flightClass(e.ID)
		if c.Projectile <= 0 || w.flightInputs.Profile.ShotsNeedArt && !w.flightInputs.Pictures[c.Projectile].Drawn {
			continue
		}
		if int64(swing) != c.shotTick() || !w.shotInReach(e) {
			continue
		}
		w.releaseShot(e, c, &obs)
	}
	return Released{FlightStarts: obs.flightStarts}
}

func (w *World) shotInReach(e Entity) bool {
	switch e.AttackTargetKind {
	case AttackTargetStructure:
		if at := indexOfStructure(w.structures, StructureID(e.AttackTarget)); at >= 0 {
			return InStructureReach(e, w.structures[at])
		}
	case AttackTargetUnit:
		if at := indexOfEntity(w.entities, e.AttackTarget); at >= 0 {
			return InReach(e, w.entities[at])
		}
	}
	return false
}

// releaseShot: a ShootDelay-0 shot has missed one driver call.
func (w *World) releaseShot(e Entity, c FlightClass, obs *castObs) {
	dx, dy, _ := c.shotOffset(e.Facing)
	late := 0
	if c.ShootDelay == 0 {
		late = 1
	}
	w.ReleaseUnitShot(UnitShot{Shooter: e.ID, Picture: c.Projectile, Phases: w.flightInputs.Pictures[c.Projectile].Phases, OffsetX: dx, OffsetY: dy, Late: late, PreMove: w.flightPreMove(obs, w.savedProjectiles.FreeIndex, c.Projectile)})
}

func (w *World) produceUnitShots(obs *castObs) {
	for i := range w.entities {
		e := &w.entities[i]
		c := w.flightClass(e.ID)
		if c.Projectile <= 0 || e.Reach <= 1 {
			e.FlightRun = FlightRun{}
			continue
		}
		if w.HasEffectArm(e.ID, 20) {
			continue
		}
		r := e.FlightRun
		if !e.Alive() || e.Turning() || !e.HasAttackTarget {
			e.FlightRun = FlightRun{}
			continue
		}
		windup := e.AttackPhase == AttackCharging || e.AttackPhase == AttackCasting
		if windup && e.AttackPhase != r.Phase {
			r = FlightRun{Active: true, Kind: e.AttackPhase, Elapsed: 0}
		} else if r.Active {
			if r.Elapsed < ^uint32(0) {
				r.Elapsed++
			}
		} else if windup {
			n, ok := w.WindUpElapsed(e.ID)
			if !ok {
				n = 1
			}
			r = FlightRun{Active: true, Kind: e.AttackPhase, Elapsed: uint32(n)}
		}
		if r.Active {
			r.Phase = e.AttackPhase
		} else {
			r = FlightRun{}
		}
		e.FlightRun = r
		if !r.Active || r.Kind != AttackCharging || e.Reach <= 1 {
			continue
		}
		if int64(r.Elapsed) != c.shotTick() || !w.shotInReach(*e) {
			continue
		}
		w.releaseShot(*e, c, obs)
	}
}

func (w *World) prepareFlightRuns() {
	if !w.releasesInStep() {
		return
	}
	for i := range w.entities {
		e := &w.entities[i]
		c := w.flightClass(e.ID)
		if c.Projectile <= 0 || e.Reach <= 1 {
			e.FlightRun = FlightRun{}
			continue
		}
		if e.FlightRun.Active || e.AttackPhase != AttackCharging && e.AttackPhase != AttackCasting {
			continue
		}
		n, ok := w.WindUpElapsed(e.ID)
		if !ok {
			n = 1
		}
		e.FlightRun = FlightRun{Active: true, Kind: e.AttackPhase, Phase: e.AttackPhase, Elapsed: uint32(n)}
	}
}

func (w *World) produceFlights(obs *castObs) {
	if !w.releasesInStep() {
		return
	}
	w.produceEvents(obs)
	w.produceUnitShots(obs)
}
