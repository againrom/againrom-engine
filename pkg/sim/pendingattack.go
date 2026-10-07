package sim

import "fmt"

func (e *Entity) clearPendingAttack() {
	e.PendingAttackTarget, e.PendingAttackTargetKind, e.HasPendingAttackTarget = 0, AttackTargetUnit, false
}

func (e Entity) RequestedAttackTarget() (EntityID, AttackTargetKind, bool) {
	if e.HasPendingAttackTarget {
		return e.PendingAttackTarget, e.PendingAttackTargetKind, true
	}
	return e.AttackTarget, e.AttackTargetKind, e.HasAttackTarget
}

func (w *World) pendingAttackValid(i int) bool {
	e := w.entities[i]
	if e.PendingAttackTargetKind == AttackTargetStructure {
		at := indexOfStructure(w.structures, StructureID(e.PendingAttackTarget))
		return at >= 0 && w.structures[at].physicalTarget()
	}
	at := indexOfEntity(w.entities, e.PendingAttackTarget)
	return at >= 0 && at != i && w.entities[at].OrdinaryTargetable() && !w.entities[at].OffMap && !w.invisibleToActor(i, at) && !w.targetVetoed(i, at)
}

func (w *World) takePendingAttacks() {
	for i := range w.entities {
		e := &w.entities[i]
		if !e.HasPendingAttackTarget || !e.Alive() || e.OffMap {
			continue
		}
		if !w.pendingAttackValid(i) {
			e.clearPendingAttack()
			continue
		}
		if e.AttackPhase == AttackReady {
			w.orderAttack(i, e.PendingAttackTarget, e.PendingAttackTargetKind)
		}
	}
}

func (w *World) pendingAttackFault() error {
	for i, e := range w.entities {
		if !e.HasPendingAttackTarget {
			if e.PendingAttackTarget != 0 || e.PendingAttackTargetKind != AttackTargetUnit {
				return fmt.Errorf("sim: absent pending attack carries operands")
			}
			continue
		}
		if e.PendingAttackTargetKind > AttackTargetStructure || !e.Alive() && !e.Dying() || e.HasTarget || e.TargetX != 0 || e.TargetY != 0 || e.Stall != 0 || len(w.routes[i]) != 0 || w.usingStructure(e.ID) {
			return fmt.Errorf("sim: pending attack conflicts with actor or movement state")
		}
		if e.HasAttackTarget && e.AttackTarget == e.PendingAttackTarget && e.AttackTargetKind == e.PendingAttackTargetKind {
			return fmt.Errorf("sim: pending attack repeats its active victim")
		}
		if e.PendingAttackTargetKind == AttackTargetUnit {
			at := indexOfEntity(w.entities, e.PendingAttackTarget)
			if at < 0 || at == i || !w.entities[at].OrdinaryTargetable() {
				return fmt.Errorf("sim: pending attack names an invalid actor")
			}
		} else {
			at := indexOfStructure(w.structures, StructureID(e.PendingAttackTarget))
			if at < 0 || !w.structures[at].physicalTarget() {
				return fmt.Errorf("sim: pending attack names an invalid structure")
			}
		}
	}
	return nil
}
