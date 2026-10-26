package sim

import "slices"

// FlightInputs is immutable installed input for the first game's record producers.
type FlightInputs struct {
	Pictures  map[int32]FlightPicture
	Classes   map[int32]FlightClass
	ClassRule FlightClassRule
}

type FlightPicture struct {
	Present                  bool
	Phases                   uint16
	Homes                    bool
	LifeTicks, LifeDivisor   int32
	LifePhases, SecondRecord bool
	CellLife                 int32
}

type FlightClass struct {
	Projectile, ShootDelay, TileSize int32
	CenterX, CenterY                 int32
	SelectionWidth, SelectionHeight  int32
	ShootOffset                      []int32
}

// FlightClassRule reads only current actor/equipment values and immutable inputs.
type FlightClassRule func(Entity, [EquipSlots]uint16) int32

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

func (w *World) SetFlightInputs(in FlightInputs) {
	copy := FlightInputs{Pictures: make(map[int32]FlightPicture, len(in.Pictures)), Classes: make(map[int32]FlightClass, len(in.Classes)), ClassRule: in.ClassRule}
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

func (w *World) flightClass(id EntityID) FlightClass {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return FlightClass{}
	}
	e := w.entities[i]
	class := e.Class
	if rule := w.flightInputs.ClassRule; rule != nil {
		class = rule(e, equipmentCodes(w.equipment[i]))
	}
	return w.flightInputs.Classes[class]
}

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

func (c FlightClass) castOffset(facing uint8, second bool) (int32, int32) {
	x, y, ok := c.shotOffset(facing)
	if !ok || second {
		x, y = c.selectionOffset()
	}
	a := (max(c.TileSize, 1) - 1) * 128
	return x + a, y + a
}

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
	if !ok || p.SecondRecord && (ci < 0 || w.entities[ci].X != ev.FromX || w.entities[ci].Y != ev.FromY) {
		x, y = ev.FromX*256+128, ev.FromY*256+128
	}
	dx, dy := c.castOffset(ev.Facing, p.SecondRecord)
	rec := CastRecord{Caster: ev.Caster, Picture: picture, Phases: p.Phases, X: x + dx, Y: y + dy, AimX: ev.ToX*256 + 128, AimY: ev.ToY*256 + 128,
		Dir: ProjectileDirection(ev.ToX-ev.FromX, ev.ToY-ev.FromY), Segments: p.segments(ev.ToX-ev.FromX, ev.ToY-ev.FromY), Owner: ev.Owner}
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

func (w *World) produceAreaBursts(paint AreaPaint) {
	picture := int32(paint.Spell)*2 + 9
	p := w.flightInputs.Pictures[picture]
	if !p.Present {
		return
	}
	for _, c := range paint.Cells {
		w.ReleaseAreaBurst(AreaBurst{CellX: c.X, CellY: c.Y, Picture: picture, Segments: p.CellLife, Phases: p.Phases, Owner: paint.Owner})
	}
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
		if int64(r.Elapsed) != max(int64(c.ShootDelay)-1, 0) {
			continue
		}
		inReach := false
		if e.AttackTargetKind == AttackTargetStructure {
			if at := indexOfStructure(w.structures, StructureID(e.AttackTarget)); at >= 0 {
				inReach = InStructureReach(*e, w.structures[at])
			}
		} else if at := indexOfEntity(w.entities, e.AttackTarget); at >= 0 {
			inReach = InReach(*e, w.entities[at])
		}
		if !inReach {
			continue
		}
		dx, dy, _ := c.shotOffset(e.Facing)
		late := 0
		if c.ShootDelay == 0 {
			late = 1
		}
		w.ReleaseUnitShot(UnitShot{Shooter: e.ID, Picture: c.Projectile, Phases: w.flightInputs.Pictures[c.Projectile].Phases, OffsetX: dx, OffsetY: dy, Late: late, PreMove: w.flightPreMove(obs, w.savedProjectiles.FreeIndex, c.Projectile)})
	}
}

func (w *World) prepareFlightRuns() {
	if w.SecondGame() || w.flightInputs == nil {
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
	if w.SecondGame() || w.flightInputs == nil {
		return
	}
	for i := range obs.scriptCasts {
		w.produceScriptCast(&obs.scriptCasts[i], obs)
	}
	for i := range obs.casts {
		w.produceCast(&obs.casts[i], obs)
	}
	for _, p := range obs.paints {
		w.produceAreaBursts(p)
	}
	w.produceUnitShots(obs)
}
