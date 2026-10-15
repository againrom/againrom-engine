package sim

import (
	"encoding/binary"
	"slices"
)

// AI-374. A terminal victim ends the engage state and its executor row.
func (w *World) stopTargetOrder(i int) {
	e := &w.entities[i]
	e.clearActiveAttack()
	if e.HeldOrder.Kind == HeldOrderBody {
		e.HeldOrder = HeldOrder{}
	}
	w.clearOrder(i)
	e.clearTurn()
	if e.ActorState == actorStateEngage || e.ActorState == 0xd || e.ActorState == 0xe {
		e.ActorState, e.PostX, e.PostY = actorStateAcquire, e.X, e.Y
	}
	if o := w.savedOrder(e.ID); o != nil {
		o.authorNative()
		o.State, o.Raw[8] = uint32(e.ActorState), 0
		binary.LittleEndian.PutUint16(o.Raw[:], uint16(e.PostX)|uint16(e.PostY)<<8)
		o.Raw[0x14] = uint8(e.Reach)
		binary.LittleEndian.PutUint32(o.Raw[0x50:], 1)
	}
	if m := w.motionFor(e.ID); m != nil {
		m.ActorAction = 0
	}
}

func (w *World) orderTargetPresent(e *Entity, o *SavedActorOrder, cast bool) bool {
	if !cast {
		if id, kind, ok := e.RequestedAttackTarget(); ok {
			if kind == AttackTargetUnit {
				return indexOfEntity(w.entities, id) >= 0
			}
			return indexOfStructure(w.structures, StructureID(id)) >= 0
		}
	} else {
		if e.PendingOrder.Kind == PendingActorCast {
			return indexOfEntity(w.entities, e.PendingOrder.Target) >= 0
		}
		for _, c := range w.bookCasts {
			if c.Caster == e.ID && !c.AtCell {
				return indexOfEntity(w.entities, c.Target) >= 0
			}
		}
		for _, c := range w.scrollCasts {
			if c.Caster == e.ID && !c.AtCell {
				return indexOfEntity(w.entities, c.Target) >= 0
			}
		}
	}
	at := 0xc
	if cast {
		at = 0x28
	}
	key := binary.LittleEndian.Uint32(o.Raw[at:])
	if key == 0 {
		return false
	}
	for i := range w.entities {
		if w.entities[i].SourceBinding.Identity == key {
			return true
		}
	}
	for _, target := range w.originalDead {
		if target.Source.Identity == key {
			return true
		}
	}
	if !cast {
		for _, target := range w.savedStructures {
			if target.SourceKey == key {
				return true
			}
		}
	}
	return false
}

// DIV-3310. Unresolved target rows become the established stand state on LOAD.
func (w *World) normaliseSavedTargetOrders() {
	if w.SecondGame() {
		return
	}
	for i := range w.entities {
		e := &w.entities[i]
		if e.Alive() && w.hasMissingTargetCast(e) {
			if !w.dropMissingTargetCast(i) {
				continue
			}
			e = &w.entities[i]
			if !w.actorCastBusy(i) {
				if _, _, requested := e.RequestedAttackTarget(); requested {
					if order := w.savedOrder(e.ID); order != nil {
						order.authorNative()
						order.Raw[8], order.Raw[0x14] = 5, uint8(e.Reach)
						w.syncCurrentActorCarriers(i)
					}
				} else {
					w.stopTargetOrder(i)
				}
			}
		}
		o := w.savedOrder(e.ID)
		if !e.Alive() || o == nil {
			continue
		}
		row := ProjectActorRetreatOrder(*e, *o).Raw[8]
		cast := row == 8
		if row != 5 && row != 6 && !cast {
			continue
		}
		if !w.orderTargetPresent(e, o, cast) {
			if !cast && e.HasTarget {
				if _, _, requested := e.RequestedAttackTarget(); !requested {
					w.syncSavedDestination(i)
					continue
				}
			}
			if cast {
				if !w.dropMissingTargetCast(i) {
					continue
				}
			}
			w.stopTargetOrder(i)
		}
	}
}

func (w *World) hasMissingTargetCast(e *Entity) bool {
	if w.SecondGame() {
		return false
	}
	if p := e.PendingOrder; p.Kind == PendingActorCast && indexOfEntity(w.entities, p.Target) < 0 {
		return true
	}
	for _, c := range w.bookCasts {
		if c.Caster == e.ID && !c.AtCell && indexOfEntity(w.entities, c.Target) < 0 {
			return true
		}
	}
	for _, c := range w.scrollCasts {
		if c.Caster == e.ID && !c.AtCell && indexOfEntity(w.entities, c.Target) < 0 {
			return true
		}
	}
	return false
}

func (w *World) dropMissingTargetCast(i int) bool {
	id := w.entities[i].ID
	for _, c := range w.scrollCasts {
		if c.Caster == id && !c.AtCell && indexOfEntity(w.entities, c.Target) < 0 && !w.cancelScroll(i) {
			return false
		}
	}
	w.bookCasts = slices.DeleteFunc(w.bookCasts, func(c bookCast) bool {
		return c.Caster == id && !c.AtCell && indexOfEntity(w.entities, c.Target) < 0
	})
	if p := w.entities[i].PendingOrder; p.Kind == PendingActorCast && indexOfEntity(w.entities, p.Target) < 0 {
		w.entities[i].PendingOrder = PendingOrder{}
	}
	state := w.orderStateAt(i)
	if !state.booked() && !state.scrollPending() && !state.queued {
		w.entities[i].CastWait = 0
	}
	return true
}
