package sim

// AttackTargetKind tags the handle stored in Entity.AttackTarget. Unit and
// structure handles overlap, including zero. The zero tag preserves old saves.
type AttackTargetKind uint8

const (
	AttackTargetUnit AttackTargetKind = iota
	AttackTargetStructure
)

// physicalTarget is the owner-directed admission/lifetime policy (DIV-546,
// DIV-547). UNIT-STRUCTORDER-062 proves the conditional pointer path, not all
// class admissions; UNIT-STRUCTSTOP-066 leaves destruction scheduling Unknown.
func (s Structure) physicalTarget() bool {
	return int16(s.Field42) > 0 && s.Width > 0 && s.Height > 0 && s.Attach&1 != 0
}

func (w *World) attackStructure(e Entity) (Structure, bool) {
	i := indexOfStructure(w.structures, StructureID(e.AttackTarget))
	if i < 0 || !w.structures[i].physicalTarget() {
		return Structure{}, false
	}
	return w.structures[i], true
}

// Building's combat token is one, independent of its blocking rectangle
// (UNIT-STRUCTREACH-063). This adapter carries geometry only, never identity,
// health, equipment or a place in the unit list.
func structurePosition(s Structure) Entity { return Entity{X: s.Col, Y: s.Row, TokenSize: 1} }

func (w *World) approachStructure(scratch *routeScratch, i int) {
	s, ok := w.attackStructure(w.entities[i])
	if !ok {
		w.entities[i].clearActiveAttack()
		w.restAt(scratch, i)
		return
	}
	w.removeAttachedSpell(w.entities[i].ID, w.armSpellID(15))
	if InStructureReach(w.entities[i], s) {
		w.turnToStructure(i, s)
		w.restAt(scratch, i)
		return
	}
	// The anchor is a static obstruction, unlike an occupied unit cell. Choose
	// a reachable firing position inside combat reach, not the footprint edge.
	// Preserve that destination during the walk. This bounded deterministic
	// routing policy is ours; original multi-cell approach success is Unknown.
	a := w.entities[i]
	if a.HasTarget {
		at := a
		at.X, at.Y = a.TargetX, a.TargetY
		if InStructureReach(at, s) && w.open(scratch, terrainRelation, i, at.X, at.Y) {
			return
		}
	}
	reaches := func(at cell) bool {
		e := a
		e.X, e.Y = at.x, at.y
		return InStructureReach(e, s)
	}
	if w.approachCell(scratch, i, cell{s.Col, s.Row}, structureAttackRadius(a), reaches) {
		return
	}
	// A completely unreachable target cannot run an eternal ready cycle.
	w.entities[i].clearAttack()
	w.restAt(scratch, i)
}

// Inverting the whole-cell strike predicate gives Reach + floor(TokenSize/2)
// for a size-one target. Reach alone omits valid firing cells for large actors.
// Keep the bounds wide, then still test exact reach and the complete footprint.
func structureAttackRadius(a Entity) int64 {
	return int64(a.Reach) + int64(max(1, a.TokenSize))/2
}

// InStructureReach is also the presentation's read-only reach predicate.
func InStructureReach(a Entity, s Structure) bool {
	return structureStrikeDistance(a, s) <= int32(a.Reach)
}

func structureStrikeDistance(a Entity, s Structure) int32 {
	return strikeDistanceForSizes(a, structurePosition(s), max(1, int32(a.TokenSize)), 1)
}

// resolveStructureBlow is the physical countdown consumer, not the effect
// consumer. UNIT-STRUCTDAMAGE-064: only the third combat pair participates,
// and a zero spread suppresses even a nonzero base and consumes no damage draw.
// UNIT-STRUCTDELIVERY-065: subtract as a word, without the effect path's clamp.
func (w *World) resolveStructureBlow(ai int) {
	a := w.entities[ai]
	si := indexOfStructure(w.structures, StructureID(a.AttackTarget))
	if si < 0 || !a.Alive() || !InStructureReach(a, w.structures[si]) {
		return
	}
	s := &w.structures[si]
	if s.MaxHealth == 0 || a.SecondaryDamage.Spread == 0 {
		return
	}
	damage := int32(a.SecondaryDamage.Base) + w.rng.uniform(int32(a.SecondaryDamage.Spread)) - 5
	if damage <= 0 {
		return
	}
	s.Field42 -= uint16(damage)
	// Keep the existing HP-to-ruin policy and stop before another subtraction
	// could wrap a ruin back to life. Original whole-world teardown is Unknown.
	if int16(s.Field42) <= 0 {
		for i := range w.entities {
			e := &w.entities[i]
			if e.HasPendingAttackTarget && e.PendingAttackTargetKind == AttackTargetStructure && e.PendingAttackTarget == a.AttackTarget {
				e.clearPendingAttack()
			}
			if e.HasAttackTarget && e.AttackTargetKind == AttackTargetStructure && e.AttackTarget == a.AttackTarget {
				e.clearActiveAttack()
				w.clearOrder(i)
			}
		}
	}
}
