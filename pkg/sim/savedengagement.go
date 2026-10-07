package sim

import "encoding/binary"

// SAV-HUMRESUME-460 repairs order+0c only at source stage zero. A missing
// lookup preserves its source word; it never becomes a native EntityID.
// Resolve against persisted construction identity on every cold native LOAD
// as well. A dead record contributes only when its entity is materialized.
func savedSourceActor(key uint32, entities []Entity, dead []originalDeadRecord) (EntityID, bool) {
	if key == 0 {
		return 0, false
	}
	matches := map[EntityID]bool{}
	for _, e := range entities {
		s := e.SourceBinding
		if s.ActorClass() >= 1 && s.ActorClass() <= 3 && s.Identity == key {
			matches[e.ID] = true
		}
	}
	for _, d := range dead {
		if d.Source.Identity == key && d.Source.Class >= 1 && d.Source.Class <= 3 && indexOfEntity(entities, d.ID) >= 0 {
			matches[d.ID] = true
		}
	}
	if len(matches) == 1 {
		for id := range matches {
			return id, true
		}
	}
	return 0, false
}

func (w *World) savedEngagementTarget(o SavedActorOrder) (EntityID, bool) {
	if o.Authored {
		i := indexOfEntity(w.entities, o.Entity)
		if i < 0 {
			return 0, false
		}
		e := w.entities[i]
		id, kind, present := e.RequestedAttackTarget()
		return id, present && kind == AttackTargetUnit && id != e.ID && indexOfEntity(w.entities, id) >= 0
	}
	if o.RepairStage != 0 {
		return 0, false
	}
	id, ok := savedSourceActor(binary.LittleEndian.Uint32(o.Raw[0x0c:]), w.entities, w.originalDead)
	return id, ok && id != o.Entity
}

// AI-STATE-011 dispatches state3 with order+0c. AI-PURSUE-040 writes the
// named-engage substate and current reach; orderAttack owns this engine's
// existing pursuit/attack cycle and preserves a reissued target's progress.
func (w *World) savedEngage(i int, o *SavedActorOrder) {
	id, ok := w.savedEngagementTarget(*o)
	if e := &w.entities[i]; ok && e.HasPendingAttackTarget && e.AttackPhase != AttackReady {
		o.Raw[8], o.Raw[0x14] = 5, uint8(e.Reach)
		return
	}
	if !ok || !w.orderAttack(i, id) {
		return
	}
	e := &w.entities[i]
	e.ActorState = actorStateEngage
	o.Raw[8], o.Raw[0x14] = 5, uint8(e.Reach)
}

// Both tactical producers publish the accepted typed action. A cast or
// stone refusal must not leave a latent state3 order that revives later.
func (w *World) publishEngagementOrder(i int) {
	e := &w.entities[i]
	e.ActorState = actorStateEngage
	if o := w.syncSavedActorCommand(i); o != nil {
		o.Raw[8], o.Raw[0x14] = 0, uint8(e.Reach)
	}
}
