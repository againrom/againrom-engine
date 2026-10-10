package sim

// The native action owns these ordinary-facing carriers once it supersedes
// their imported continuation. Update them with that action, so SAVE does
// not become the first writer of the current mover or typed order endpoints.
func (w *World) syncCurrentActionCarriers() {
	w.syncNativeActorCells(nil)
	for i := range w.entities {
		w.syncCurrentActorCarriers(i)
	}
	for _, cast := range w.bookCasts {
		index := indexOfEntity(w.entities, cast.Caster)
		if index < 0 || cast.Spell < 1 || cast.Spell > 28 {
			continue
		}
		key := uint32(0)
		if w.savedObjects != nil {
			for _, root := range w.savedObjects.BookRoots {
				if root.Entity == cast.Caster {
					if spell := w.savedObjects.spell(root.Slots[cast.Spell-1]); spell != nil {
						key = spell.This
					}
					break
				}
			}
		}
		values := BookSlotValues(w.rules, w.entities[index], w.spells)
		w.syncCurrentCastOperands(cast.Caster, cast.Target, cast.AtCell, cast.X, cast.Y, key, values[cast.Spell-1].Range)
	}
	for _, cast := range w.scrollCasts {
		index := indexOfEntity(w.entities, cast.Caster)
		if index >= 0 && w.entities[index].PendingOrder.Kind == PendingScroll && !w.entities[index].PendingOrder.RowAdmitted {
			if order := w.savedOrder(cast.Caster); order != nil {
				target := uint32(0)
				if !cast.AtCell {
					target = w.currentActionKey(cast.Target, AttackTargetUnit)
				}
				order.Raw = ProjectCastOrderOperands(order.Raw, target, 0, cast.X, cast.Y, 0, cast.AtCell)
			}
		} else {
			w.syncCurrentCastOperands(cast.Caster, cast.Target, cast.AtCell, cast.X, cast.Y, 0, 0)
		}
	}
	for i := range w.entities {
		e := &w.entities[i]
		p := e.PendingOrder
		switch p.Kind {
		case PendingActorCast, PendingCellCast:
			key, reach := uint32(0), uint8(0)
			if p.Spell >= 1 && p.Spell <= 28 {
				values := BookSlotValues(w.rules, *e, w.spells)
				reach = values[p.Spell-1].Range
				if w.savedObjects != nil {
					for _, root := range w.savedObjects.BookRoots {
						if root.Entity == e.ID {
							if spell := w.savedObjects.spell(root.Slots[p.Spell-1]); spell != nil {
								key = spell.This
							}
							break
						}
					}
				}
			}
			if p.RowAdmitted {
				w.syncCurrentCastAction(e.ID, p.Kind == PendingCellCast)
			}
			if order := w.savedOrder(e.ID); order != nil {
				target := uint32(0)
				if p.Kind == PendingActorCast {
					target = w.currentActionKey(p.Target, AttackTargetUnit)
				}
				order.Raw = ProjectCastOrderOperands(order.Raw, target, key, p.X, p.Y, reach, p.Kind == PendingCellCast)
			}
		case PendingPickup, PendingPickupComplete:
			if m := w.motionFor(e.ID); m != nil && p.RowAdmitted {
				m.ActorAction = 0
				if p.Kind == PendingPickup {
					m.ActorAction = 2
					if e.Transit != 0 || w.motionActive(e.ID) {
						m.ActorAction = 1
					}
				}
			}
		}
	}
}

func (w *World) syncCurrentActorCarriers(i int) {
	e := &w.entities[i]
	// Current=false means a native producer superseded the imported motion.
	// Project that producer even when Issue names why the imported continuation
	// was invalidated; otherwise SAVE would retain and later resume stale motion.
	m := w.motionFor(e.ID)
	if m != nil && !m.Current && !e.OffMap && e.Alive() && m.Issue != "native movement continues the imported route" {
		next, err := ProjectActorMotion(*e, *m, w.Route(e.ID), false)
		if err == nil {
			next.Current, next.Active, next.Issue = m.Current, m.Active, m.Issue
			// An attack held in place carries action 3, the value SAVE writes
			// for it, so LOAD restores the World this carrier describes.
			if e.HasAttackTarget && e.Transit == 0 && !e.Turning() && e.AttackPhase != AttackBoundaryTwo {
				next.ActorAction = 3
			}
			*m = next
		}
	} else if m != nil && (m.ActorAction == 0xd || m.ActorAction == 0xe) && e.Alive() {
		m.ActorAction = 0
		if e.HasAttackTarget && e.Transit == 0 && !e.Turning() && e.AttackPhase != AttackBoundaryTwo {
			m.ActorAction = 3
		}
	}
	w.syncCurrentActorOrder(i)

	if e.HasAttackTarget && e.AttackPhase == AttackCasting && w.savedObjects != nil {
		item := w.savedObjects.item(w.equipment[i][0].ObjectID)
		if item != nil && item.Spell != 0 {
			w.syncCurrentCastAction(e.ID, false)
		}
	}
}

func (w *World) currentActionKey(id EntityID, kind AttackTargetKind) uint32 {
	if kind == AttackTargetUnit {
		if i := indexOfEntity(w.entities, id); i >= 0 {
			return w.entities[i].SourceBinding.Identity
		}
	} else if kind == AttackTargetStructure {
		for _, s := range w.savedStructures {
			if EntityID(s.ID) == id {
				return s.SourceKey
			}
		}
	}
	return 0
}

func (w *World) syncCurrentActorOrder(i int) {
	e := &w.entities[i]
	if order := w.savedOrder(e.ID); order != nil && (!order.Authored || e.HasAttackTarget || order.State == 3 || order.Raw[0xc]|order.Raw[0xd]|order.Raw[0xe]|order.Raw[0xf] != 0) {
		id, kind, _ := e.RequestedAttackTarget()
		*order = ProjectActorOrderTargets(*e, *order, w.currentActionKey(id, kind), w.currentActionKey(e.EscortTarget, AttackTargetUnit))
	}
}

func (w *World) syncCurrentCastAction(id EntityID, atCell bool) {
	if m := w.motionFor(id); m != nil {
		m.ActorAction = 0xd
		if atCell {
			m.ActorAction = 0xe
		}
	}
}

func (w *World) syncCurrentCastOperands(caster, target EntityID, atCell bool, x, y int32, spell uint32, radius uint8) {
	w.syncCurrentCastAction(caster, atCell)
	if order := w.savedOrder(caster); order != nil {
		var key uint32
		if !atCell {
			key = w.currentActionKey(target, AttackTargetUnit)
		}
		order.Raw = ProjectCastOrderOperands(order.Raw, key, spell, x, y, radius, atCell)
	}
}
