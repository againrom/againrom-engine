package sim

import (
	"slices"
	"sort"
)

// unitShotSegment is the distance, in 256-per-cell units, that one driver
// call of a unit's shot covers (SAV-1130): the shot is built with
// actionsegments equal to the truncated shooter-to-target distance over it.
const unitShotSegment = 200

// projectileBuckets is the id hash's bucket count (SAV-1131).
const projectileBuckets = 17

// UnitShot names one released physical ranged shot. The caller owns what the
// installed registries decide: the class's projectile picture, its phase
// count and the release offset from the shooter's own point. The World owns
// the record: id, leaves, direction, driver and flight.
type UnitShot struct {
	Shooter EntityID
	Picture int32
	Phases  uint16
	// OffsetX and OffsetY displace the record from the shooter's point in
	// 256-per-cell units (SAV-1188).
	OffsetX, OffsetY int32
	// Late is the number of driver calls the record owes for ticks that
	// passed before the caller could build it; each runs after the first.
	Late int
	// PreMove, when set, observes the point each of the release's driver
	// calls starts from, the point a trail appends (ANIM-140).
	PreMove func(x, y int32)
}

// hasRuntimeID reports a source runtime identity: an actor without one is
// native, and its wire identity exists only in the Document.
func (b SourceBinding) hasRuntimeID() bool { return b.Class != 0 && b.RuntimeID != 0 }

// ProjectileTargetKey is the actiontarget value a record carries for target.
// An actor that holds a source runtime identity keeps it; any other actor is
// keyed by its entity id until the writer assigns the actor's runtime
// identity at SAVE.
func ProjectileTargetKey(target Entity) int32 {
	if target.SourceBinding.Class != 0 && target.SourceBinding.RuntimeID != 0 {
		return int32(target.SourceBinding.RuntimeID)
	}
	return int32(target.ID)
}

// ReleaseUnitShot builds the record a physical ranged swing's shot leaves as
// (SAV-1129, SAV-1130, SAV-1131) and runs its first driver call. The target is
// the shooter's attack target: a unit, or a structure, which the original
// finds in the same client hash and measures and homes on like a unit
// (SAV-1197). Any picture above 0 is admitted (SAV-1198); damage stays with the
// swing's own countdown (SAV-1132). It reports whether a record was built.
func (w *World) ReleaseUnitShot(s UnitShot) bool {
	if s.Picture < 1 {
		return false
	}
	at := indexOfEntity(w.entities, s.Shooter)
	if at < 0 {
		return false
	}
	shooter := w.entities[at]
	if !shooter.HasAttackTarget {
		return false
	}
	var tx, ty int32
	driver := SavedProjectileDriver{Phases: s.Phases, Target: shooter.AttackTarget, HasTarget: true}
	key := int32(0)
	switch shooter.AttackTargetKind {
	case AttackTargetUnit:
		to := indexOfEntity(w.entities, shooter.AttackTarget)
		if to < 0 {
			return false
		}
		target := w.entities[to]
		key = ProjectileTargetKey(target)
		tx, ty = w.savedProjectileTargetPoint(target)
	case AttackTargetStructure:
		to := indexOfStructure(w.structures, StructureID(shooter.AttackTarget))
		if to < 0 {
			return false
		}
		key = int32(shooter.AttackTarget)
		tx, ty = structureTargetPoint(w.structures[to])
		driver.TargetStructure = true
	default:
		return false
	}
	if key == 0 {
		return false
	}
	sx, sy := w.savedProjectileTargetPoint(shooter)
	dx, dy := int64(tx)-int64(sx), int64(ty)-int64(sy)
	segments := isqrt64(dx*dx+dy*dy) / unitShotSegment

	// Dir and actiondir start at zero: every driver call aims them (ANIM-139).
	record := SavedProjectile{
		X: sx + s.OffsetX, Y: sy + s.OffsetY, Picture: s.Picture,
		Action: 1, ActionTarget: key, ActionSegments: int32(segments),
	}
	i := w.insertProjectile(record, driver)
	w.driveNewRecord(i, 1+max(s.Late, 0), s.PreMove)
	return true
}

// driveNewRecord runs a just-built record's first driver calls, observing each
// call's starting point through preMove (ANIM-140).
func (w *World) driveNewRecord(i, calls int, preMove func(x, y int32)) {
	for range calls {
		d := &w.savedWorldEffects.Projectiles[i]
		if p := w.savedProjectile(d.ID); preMove != nil && !d.Retired && p != nil && p.ActionSegments != 0 {
			preMove(p.X, p.Y)
		}
		w.stepSavedProjectile(d)
	}
}

// structureTargetPoint is the point a record aims at for a structure: the
// centre of its anchor cell. The original derives the point as for any
// drawable; whether a structure's anchor equals a unit's is open (SAV-1197,
// DIV-2846).
func structureTargetPoint(s Structure) (int32, int32) {
	return s.Col*256 + 128, s.Row*256 + 128
}

// insertProjectile adds one record to the carried store and its driver row: it
// takes the id the shared counter holds, moves the counter to id + 1 and places
// the id at the head of its hash bucket, the order the manager's serializer
// walks (SAV-1131, SAV-1146). A record already holding the id is replaced, as
// the manager replaces a node whose id is reused. It returns the driver row's
// index.
func (w *World) insertProjectile(record SavedProjectile, d SavedProjectileDriver) int {
	id := w.savedProjectiles.FreeIndex
	w.savedProjectiles.FreeIndex = id + 1
	if existing := w.savedProjectile(id); existing != nil {
		w.dropSavedProjectile(id)
	}
	record.ID, d.ID = id, id
	inOrder := len(w.savedProjectiles.Items) == len(w.savedProjectiles.IDs)
	for k := 0; inOrder && k < len(w.savedProjectiles.Items); k++ {
		inOrder = w.savedProjectiles.Items[k].ID == w.savedProjectiles.IDs[k]
	}
	w.savedProjectiles.Items = append(w.savedProjectiles.Items, record)
	w.savedProjectiles.IDs = insertProjectileID(w.savedProjectiles.IDs, id)
	// Items follow the saved order of IDs, the order a LOAD returns them in.
	order := w.savedProjectiles.IDs
	if inOrder {
		items := w.savedProjectiles.Items
		at := slices.Index(order, id)
		copy(items[at+1:], items[at:len(items)-1])
		items[at] = record
	} else {
		slices.SortStableFunc(w.savedProjectiles.Items, func(a, b SavedProjectile) int {
			return slices.Index(order, a.ID) - slices.Index(order, b.ID)
		})
	}
	if w.savedWorldEffects == nil {
		w.savedWorldEffects = &SavedWorldEffects{}
	}
	drivers := w.savedWorldEffects.Projectiles
	i := sort.Search(len(drivers), func(i int) bool { return drivers[i].ID >= id })
	if i < len(drivers) && drivers[i].ID == id {
		drivers[i] = d
	} else {
		drivers = slices.Insert(drivers, i, d)
	}
	w.savedWorldEffects.Projectiles = drivers
	return i
}

// dropSavedProjectile removes one record and its driver row, the way the
// manager replaces a node whose id is reused.
func (w *World) dropSavedProjectile(id uint16) {
	w.savedProjectiles.Items = slices.DeleteFunc(w.savedProjectiles.Items, func(p SavedProjectile) bool { return p.ID == id })
	w.savedProjectiles.IDs = slices.DeleteFunc(w.savedProjectiles.IDs, func(v uint16) bool { return v == id })
	if w.savedWorldEffects != nil {
		w.savedWorldEffects.Projectiles = slices.DeleteFunc(w.savedWorldEffects.Projectiles, func(d SavedProjectileDriver) bool { return d.ID == id })
	}
}

// insertProjectileID places a new id at the head of its hash bucket, the
// order the manager's serializer walks (SAV-1131). Existing ids keep their
// order.
func insertProjectileID(ids []uint16, id uint16) []uint16 {
	bucket := func(v uint16) int { return int(v>>4) % projectileBuckets }
	at := len(ids)
	for i, v := range ids {
		if bucket(v) >= bucket(id) {
			at = i
			break
		}
	}
	return slices.Insert(ids, at, id)
}

// isqrt64 is the largest n with n*n <= v, and 0 for any v at or below 0.
func isqrt64(v int64) int64 {
	if v <= 0 {
		return 0
	}
	n := int64(1)
	for n*n <= v {
		n <<= 1
	}
	lo, hi := n>>1, n
	for lo+1 < hi {
		mid := (lo + hi) / 2
		if mid*mid <= v {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo
}
