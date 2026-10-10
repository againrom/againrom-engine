package sim

// teleportProjectilePicture is the cast picture whose producer builds a second
// record by copy and whose driver arm writes no position (ANIM-147, ANIM-149).
const teleportProjectilePicture = 60

// CastRecord names one record a cast leaves as. The caller owns what the
// installed registries decide: the picture, its phase count, the start point
// the caster class's launch rule gives and the segment count of the
// producer's picture switch. The World owns the record: id, leaves, driver and
// flight. Damage stays with the cast's own delivery.
type CastRecord struct {
	Caster  EntityID
	Picture int32
	Phases  uint16
	// X and Y are the start point in 256-per-cell units.
	X, Y int32
	// Target is the actor a homing picture aims at; without one the record
	// aims at AimX, AimY.
	Target     EntityID
	HasTarget  bool
	AimX, AimY int32
	// Dir is the caster's sixteen-way direction. Client is set for a record a
	// client arm builds from a message (0x8b, 0x8c): it starts at actionphase
	// -1 and keeps dir and actiondir at the constructor's 0 (ANIM-144).
	Dir      int32
	Client   bool
	Segments int32
	Owner    uint32
	// PreMove observes the point the first driver call starts from.
	PreMove func(x, y int32)
}

// ReleaseCast builds the record a cast producer builds at the cast frame and
// runs its first driver call (ANIM-147, ANIM-144, SAV-1129). A record with 0
// segments takes its id from the shared counter and retires on that call, so
// no SAVE can hold it (SAV-1204). It answers the record's id and false when
// nothing was built: a homing target that is not in the World builds nothing,
// as the producer returns when the target is not in the client hash.
func (w *World) ReleaseCast(c CastRecord) (uint16, bool) {
	if c.Picture < 0 || c.Segments < 0 {
		return 0, false
	}
	record := SavedProjectile{X: c.X, Y: c.Y, Picture: c.Picture, Action: 1,
		ActionX: c.AimX, ActionY: c.AimY, ActionSegments: c.Segments}
	driver := SavedProjectileDriver{Phases: c.Phases, Owner: c.Owner}
	if c.HasTarget {
		to := indexOfEntity(w.entities, c.Target)
		if to < 0 {
			return 0, false
		}
		target := w.entities[to]
		key := ProjectileTargetKey(target)
		if key == 0 {
			return 0, false
		}
		record.ActionTarget = key
		record.ActionX, record.ActionY = w.savedProjectileTargetPoint(target)
		driver = SavedProjectileDriver{Phases: c.Phases, Owner: c.Owner, Target: target.ID, HasTarget: true}
	}
	if c.Client {
		record.ActionPhase = -1
	} else {
		record.Dir, record.ActionDir = c.Dir&15, c.Dir&15
		if at := indexOfEntity(w.entities, c.Caster); c.HasTarget && at >= 0 {
			cx, cy := w.savedProjectileTargetPoint(w.entities[at])
			record.ActionDir = ProjectileDirection(record.ActionX-cx, record.ActionY-cy)
		}
	}
	id := w.savedProjectiles.FreeIndex
	i := w.insertProjectile(record, driver)
	w.driveNewRecord(i, 1, c.PreMove)
	return id, true
}

// AreaBurst names one record a staged area stage builds at one accepted cell
// (ANIM-148): the odd burst picture at the cell centre, no target, started at
// actionphase -1 with the sender's segment count.
type AreaBurst struct {
	CellX, CellY int32
	Picture      int32
	Segments     int32
	Phases       uint16
	Owner        uint32
}

// ReleaseAreaBurst builds one staged burst record and runs its first driver
// call. It answers the record's id.
func (w *World) ReleaseAreaBurst(b AreaBurst) (uint16, bool) {
	if b.Picture < 0 || b.Segments < 0 {
		return 0, false
	}
	id := w.savedProjectiles.FreeIndex
	i := w.insertBurstRecord(b)
	w.driveNewRecord(i, 1, nil)
	return id, true
}

// ProjectilePoint is the point a record measures an actor from, in
// 256-per-cell units: its current fine position, its stride's interpolated
// point, or its cell centre.
func (w *World) ProjectilePoint(id EntityID) (int32, int32, bool) {
	at := indexOfEntity(w.entities, id)
	if at < 0 {
		return 0, 0, false
	}
	x, y := w.savedProjectileTargetPoint(w.entities[at])
	return x, y, true
}
