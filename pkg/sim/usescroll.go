package sim

import "sort"

// ScrollCast owns a detached single item from admission until release or refund.
// Armed UI state owns no item. The original slot and complete instance survive
// native saves; no learned-spell bit or mana balance substitutes for the item.
type ScrollCast struct {
	Caster, Target  EntityID
	X, Y            int32
	Index           uint16
	AtCell, Started bool
	Remaining       uint8
	Item            ItemInstance
	Reservation     uint64
}

func (w *World) ScrollCasts() []ScrollCast {
	out := append([]ScrollCast(nil), w.scrollCasts...)
	for i := range out {
		out[i].Item = out[i].Item.Clone()
	}
	return out
}

func (w *World) scrollIndex(id EntityID) (int, bool) {
	i := sort.Search(len(w.scrollCasts), func(i int) bool { return w.scrollCasts[i].Caster >= id })
	return i, i < len(w.scrollCasts) && w.scrollCasts[i].Caster == id
}

func (w *World) scrollInFlight(id EntityID) bool {
	i, ok := w.scrollIndex(id)
	return ok && w.scrollCasts[i].Started
}

func ScrollSpell(item ItemInstance) (uint16, int32, bool) {
	if item.Kind != 4 || item.Code>>8&15 != 14 || item.Price < 0 || len(item.Effects) == 0 || len(item.Effects) > 64 || item.Effects[0].Kind != 41 {
		return 0, 0, false
	}
	e := item.Effects[0]
	id := uint16(uint8(e.Operand))
	return id, int32(uint8(e.Operand >> 16)), id != 0
}

func (w *World) beginScroll(ci, index int, target EntityID, x, y int32, atCell bool) bool {
	if ci < 0 || ci >= len(w.entities) || index < 0 || index >= len(w.carried[ci]) || index > 65535 {
		return false
	}
	e := w.entities[ci]
	if e.HP <= 0 || e.OffMap || w.stoneCursed(ci) || w.motionActive(e.ID) || w.actorCastBusy(ci) {
		return false
	}
	if !w.sourceMutationReady(ci) {
		return false
	}
	item := w.carried[ci][index].Instance()
	id, _, ok := ScrollSpell(item)
	rule, found := w.findSpell(uint32(id))
	if !ok || !found || !spellApplicable(rule) {
		return false
	}
	if atCell {
		// TargetsUnit is the installed target parameter, not area shape.
		if rule.TargetsUnit {
			return false
		}
		if _, ok := cellIndexIn(w.bounds, x, y); !ok {
			return false
		}
	} else {
		ti := indexOfEntity(w.entities, target)
		if ti < 0 || !spellTargetable(w.entities[ti], rule) || prismaticBodyPrimary(w.entities[ti], rule) || w.entities[ti].OffMap {
			return false
		}
		x, y = w.entities[ti].X, w.entities[ti].Y
	}
	live := w
	if w.savedObjects != nil {
		n := w.sourceMutationCopy(ci)
		w = &n
	}
	var reservation uint64
	if item.ObjectID != 0 {
		before := w.beginLoadMutation(ci)
		taken, ok := w.takeCarriedObject(ci, index, false)
		if !ok || !w.finishLoadMutation(ci, before) {
			return false
		}
		var err error
		reservation, err = w.savedObjects.ReserveExternal(taken.ObjectID)
		if err != nil {
			return false
		}
		item = taken.Instance()
	} else if !w.consumeCarriedUnit(ci, index) {
		return false
	}
	progress := w.pendingLogicalProgress(ci)
	if w.retainsOldStrike(ci) {
		w.entities[ci].releaseBetweenCycles(PendingScroll)
		w.entities[ci].PendingOrder = PendingOrder{Kind: PendingScroll, RowAdmitted: progress == 0}
	} else {
		w.entities[ci].clearAttack()
	}
	w.clearOrder(ci)
	w.commandGroup([]int{ci}, orderMove, cell{x: x, y: y})
	c := ScrollCast{Caster: e.ID, Target: target, X: x, Y: y, AtCell: atCell, Index: uint16(index), Item: item.Clone(), Reservation: reservation}
	if atCell {
		c.Target = 0
	}
	k, _ := w.scrollIndex(e.ID)
	w.scrollCasts = append(w.scrollCasts, ScrollCast{})
	copy(w.scrollCasts[k+1:], w.scrollCasts[k:])
	w.scrollCasts[k] = c
	if !w.savedMutationValid() {
		return false
	}
	if w != live {
		*live = *w
	}
	return true
}

func (w *World) cancelScroll(ci int) bool {
	k, ok := w.scrollIndex(w.entities[ci].ID)
	if !ok {
		return true
	}
	if !w.sourceMutationReady(ci) || !w.hasActorContainer(ci) {
		return false
	}
	if w.savedObjects != nil || w.entities[ci].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(ci)
		if !n.refundScroll(ci, k) || !n.savedMutationValid() {
			return false
		}
		*w = n
		return true
	}
	return w.refundScroll(ci, k)
}

func (w *World) refundScroll(ci, k int) bool {
	c := w.scrollCasts[k]
	before := w.beginLoadMutation(ci)
	w.scrollCasts = append(w.scrollCasts[:k], w.scrollCasts[k+1:]...)
	i := min(int(c.Index), len(w.carried[ci]))
	if c.Item.ObjectID != 0 {
		row := w.savedObjects.item(c.Item.ObjectID)
		if row == nil || !w.savedObjects.HasRoot(row.ID, SavedObjectOwner{Kind: SavedOwnerSession, SessionHandle: c.Reservation}) {
			return false
		}
		if _, err := w.savedObjects.ImportExternal(c.Reservation, StackItem(c.Item, 1)); err != nil {
			return false
		}
		var merge SavedObjectID
		if i < len(w.carried[ci]) && w.carried[ci][i].Count < 65535 {
			merge = w.carried[ci][i].ObjectID
		}
		if !w.putCarriedObjectAt(ci, StackItem(c.Item, 1), i, merge) {
			return false
		}
	} else if i < len(w.carried[ci]) && w.carried[ci][i].ObjectID == 0 && ItemEqual(w.carried[ci][i].Instance(), c.Item) && w.carried[ci][i].Price == c.Item.Price && w.carried[ci][i].Count < 65535 {
		w.carried[ci][i].Count++
	} else {
		w.carried[ci] = append(w.carried[ci], ItemStack{})
		copy(w.carried[ci][i+1:], w.carried[ci][i:])
		w.carried[ci][i] = StackItem(c.Item, 1)
	}
	if !w.finishLoadMutation(ci, before) {
		return false
	}
	w.clearOrder(ci)
	if w.entities[ci].PendingOrder.Kind == PendingScroll {
		w.entities[ci].PendingOrder = PendingOrder{}
	}
	return true
}

func (w *World) stepScrollCasts(obs *castObs, interrupted map[EntityID]bool) map[EntityID]bool {
	var released map[EntityID]bool
	for k := 0; k < len(w.scrollCasts); {
		c := &w.scrollCasts[k]
		if interrupted[c.Caster] {
			k++
			continue
		}
		ci := indexOfEntity(w.entities, c.Caster)
		if ci < 0 {
			if !w.retireSessionObject(c.Item, c.Reservation) {
				k++
				continue
			}
			w.scrollCasts = append(w.scrollCasts[:k], w.scrollCasts[k+1:]...)
			continue
		}
		id, power, _ := ScrollSpell(c.Item)
		rule, exists := w.findSpell(uint32(id))
		if !exists || w.entities[ci].HP <= 0 || w.entities[ci].OffMap {
			if !w.cancelScroll(ci) {
				k++
			}
			continue
		}
		if !c.AtCell {
			ti := indexOfEntity(w.entities, c.Target)
			if ti < 0 || !spellTargetable(w.entities[ti], rule) || prismaticBodyPrimary(w.entities[ti], rule) || w.entities[ti].OffMap {
				if !w.cancelScroll(ci) {
					k++
				}
				continue
			}
			c.X, c.Y = w.entities[ti].X, w.entities[ti].Y
		}
		e := &w.entities[ci]
		if e.PendingOrder.Kind == PendingScroll {
			if !w.pendingOrderReady(ci) {
				k++
				continue
			}
			if !e.PendingOrder.RowAdmitted {
				e.PendingOrder.RowAdmitted = true
				k++
				continue
			}
			if w.stoneCursed(ci) || w.motionActive(e.ID) {
				k++
				continue
			}
			e.clearActiveAttack()
			e.PendingOrder = PendingOrder{}
			e.Retreat = RetreatContinuation{}
		}
		if w.stoneCursed(ci) || w.motionActive(e.ID) {
			k++
			continue
		}
		if !c.Started {
			if cellOf(*e).chebyshevTo(cell{x: c.X, y: c.Y}) > spellRangeUnder(w.rules, rule, power) {
				if !e.HasTarget {
					// A prior search gave up. Preserve the item on this authored
					// failure boundary rather than inventing original destruction.
					if c.Remaining != 0 {
						if !w.cancelScroll(ci) {
							k++
						}
						continue
					}
					c.Remaining = 1
				}
				if !e.HasTarget || e.TargetX != c.X || e.TargetY != c.Y {
					w.routes[ci] = nil
					e.TargetX, e.TargetY, e.HasTarget = c.X, c.Y, true
				}
				k++
				continue
			}
			if e.Transit != 0 || e.CastWait != 0 {
				k++
				continue
			}
			w.clearOrder(ci)
			c.Started, c.Remaining = true, castWindupTicks(*e)
			w.turnToward(ci, c.X-e.X, c.Y-e.Y)
			w.startSpellAction(ci)
			k++
			continue
		}
		if e.Turning() {
			k++
			continue
		}
		if c.Remaining > 0 {
			c.Remaining--
		}
		if c.Remaining > 0 {
			k++
			continue
		}
		state := *c
		if !w.retireSessionObject(state.Item, state.Reservation) {
			// A failed reservation transaction has not completed the cast.
			// Keep a valid retry phase without running effect callbacks.
			c.Remaining = 1
			k++
			continue
		}
		// Commit before effect callbacks: damage/Control Spirit can change
		// actor storage. An ineffective completed cast still consumes its item.
		w.scrollCasts = append(w.scrollCasts[:k], w.scrollCasts[k+1:]...)
		w.releaseScroll(ci, state, rule, power, obs)
		ci = indexOfEntity(w.entities, state.Caster)
		if ci >= 0 && w.entities[ci].Alive() {
			w.entities[ci].CastWait = w.bookRecoveryTicks(ci, rule)
		}
		if released == nil {
			released = make(map[EntityID]bool)
		}
		released[state.Caster] = true
	}
	return released
}

func (w *World) releaseScroll(ci int, c ScrollCast, rule SpellRule, power int32, obs *castObs) {
	a := w.entities[ci]
	if c.AtCell || c.Target != a.ID {
		w.removeAttachedSpell(a.ID, w.armSpellID(15))
	}
	if c.AtCell {
		applied := false
		if rule.arm() == 26 {
			applied = true
			obs.recordAt(w, ci, rule, c.X, c.Y)
			if w.attachFootprint(ci, c.X, c.Y) {
				w.entities[ci].X, w.entities[ci].Y = c.X, c.Y
				w.invalidateActorMotion(w.entities[ci].ID, "native teleport supersedes original movement")
				w.entities[ci].clearStride()
			}
		} else if rule.Area {
			applied = w.landAreaFacing(rule, uint16(power), a.ID, true, a.X, a.Y, c.X, c.Y, a.Facing, true, obs)
		}
		if applied {
			w.markSpellEffect(ci, rule.ID)
			if rule.arm() != 26 {
				obs.recordAt(w, ci, rule, c.X, c.Y)
			}
		}
		return
	}
	ti := indexOfEntity(w.entities, c.Target)
	if ti < 0 {
		return
	}
	var victims []CellPoint
	applied := false
	if rule.arm() == 14 {
		victims = w.applyPrismaticItem(ci, ti, rule, power, true)
		applied = len(victims) != 0
	} else if rule.Area {
		applied = w.landAreaAimed(areaAim{Target: c.Target, Has: true}, rule, uint16(power), a.ID, true, a.X, a.Y, c.X, c.Y, a.Facing, true, obs)
	} else if rule.arm() == controlSpiritSpellID {
		// The arm finds its corpse by the target's cell and never reads the
		// target object (MAGIC-251).
		if corpse := w.controlSpiritCorpseAt(w.entities[ti].X, w.entities[ti].Y, ti); corpse >= 0 &&
			w.pointEffectRefusal(ci, corpse, rule, power) == "" {
			applied = w.ordinaryEffect(ci, corpse, rule, power)
		}
	} else if w.pointEffectRefusal(ci, ti, rule, power) == "" {
		applied = w.ordinaryEffect(ci, ti, rule, power)
	}
	ci, ti = indexOfEntity(w.entities, a.ID), indexOfEntity(w.entities, c.Target)
	if applied && ci >= 0 {
		w.markSpellEffect(ci, rule.ID)
		if ti >= 0 {
			if rule.Delivery != 2 {
				w.markSpellEffect(ti, rule.ID)
			}
			obs.recordFrom(w, ci, ti, rule, false, a.X, a.Y)
		}
		obs.recordVictims(victims)
	}
}
