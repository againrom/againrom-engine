package game

import (
	"againrom/pkg/sim"
	"encoding/binary"
	"fmt"
)

func currentActionTargetKey(state *SnapshotSAVDocument, w *sim.World, id sim.EntityID, kind sim.AttackTargetKind) (uint32, error) {
	if kind == sim.AttackTargetUnit {
		for _, e := range w.Entities() {
			if e.ID == id {
				return savedCurrentOrderActorKey(state, w.Entities(), id)
			}
		}
		return 0, nil // removed target: exact absent endpoint in the supplement
	}
	if kind != sim.AttackTargetStructure {
		return 0, fmt.Errorf("invalid current target class")
	}
	rows, _, _ := w.SavedStructures()
	present := false
	for _, s := range rows {
		if sim.EntityID(s.ID) == id {
			present = true
			found := false
			for _, r := range state.Document.Objects {
				key, err := savedStructureValue(&r, "Identity")
				if err == nil && key == s.SourceKey {
					if found {
						return 0, fmt.Errorf("ambiguous structure target")
					}
					found = true
				}
			}
			if found && s.SourceKey != 0 {
				return s.SourceKey, nil
			}
		}
	}
	if !present {
		return 0, nil
	}
	return 0, fmt.Errorf("current structure target has no exact object")
}

// Authored orders already own their current raw scalar/ring operands. This
// pass translates only typed endpoints. Repair-stage and runtime scheduling
// differences remain in the supplement; they cannot leave stale target keys.
func projectNativeOrder(state *SnapshotSAVDocument, w *sim.World, entities []sim.Entity, index uint16, o sim.SavedActorOrder) error {
	var e sim.Entity
	found := false
	for _, v := range entities {
		if v.ID == o.Entity {
			e, found = v, true
			break
		}
	}
	if !found {
		return fmt.Errorf("current order actor is absent")
	}
	var attack, escort uint32
	var err error
	id, kind, requested := e.RequestedAttackTarget()
	if requested {
		attack, err = currentActionTargetKey(state, w, id, kind)
		if err != nil {
			return err
		}
	}
	if e.HasEscortTarget {
		escort, err = currentActionTargetKey(state, w, e.EscortTarget, sim.AttackTargetUnit)
		if err != nil {
			return err
		}
	}
	// A body's retained order still names the state it held alive. Death
	// reset the entity to guard and cleared its patrol, and the current state
	// is what SAV writes: that state and no ring (DIV-1432).
	if !e.Alive() && sim.LivingOnlyActorState(o.State) {
		o.State, o.Patrol = uint32(e.ActorState), nil
	}
	o = sim.ProjectActorOrderTargets(e, o, attack, escort)
	o = sim.ProjectActorRetreatOrder(e, o)
	o.Authored = false
	return projectSavedOrderFields(&state.Document.Objects[index-1], o)
}

// AI-ORDER-039/MAGIC-221 own the live order-8/9 endpoints and Spell range.
// MAGIC-CADENCE-126/127 own cast state d/e, phase5/7 and completion. The
// native one-shot/payment/retry scheduler is separately typed, not inferred
// by executing an original callback at SAVE or LOAD.
func projectCurrentCastWire(state *SnapshotSAVDocument, w *sim.World) error {
	actions := w.Actions()
	actors := map[sim.EntityID]uint16{}
	for _, b := range state.Actors {
		if !b.Retired {
			actors[b.EntityID] = b.ObjectIndex
		}
	}
	items := savedObjectIndices(state.Objects)
	for _, e := range w.Entities() {
		index := actors[e.ID]
		if index == 0 {
			continue
		}
		r := &state.Document.Objects[index-1]
		// ITEM-CASTSTATE-056: U68 is an archive borrow, U64 a reminted
		// Spell key. An active weapon cast borrows its current held object.
		if e.HasAttackTarget && e.AttackPhase == sim.AttackCasting && !e.PendingOrder.RowAdmitted {
			equipped, _ := w.EquippedItems(e.ID)
			item := items[equipped[0].ObjectID]
			if item != 0 {
				refs, _ := savedObjectRefs(&state.Document.Objects[item-1], "WeaponSpell")
				if len(refs) == 1 && refs[0] != 0 {
					key, err := savedStructureValue(&state.Document.Objects[refs[0]-1], "This")
					if err != nil {
						return err
					}
					savedObjectSetValue(r, "U64", key)
					savedObjectSetRefs(r, "U68", []uint16{item}, false)
					mustSetRaw(r, "U54", binary.LittleEndian.AppendUint32(nil, 0xd))
				}
			}
		}
	}
	write := func(caster, target sim.EntityID, atCell bool, x, y int32, spell uint16, remaining, phase, progress uint8, complete bool, book, retain bool) error {
		index := actors[caster]
		if index == 0 {
			return fmt.Errorf("current cast has no caster object")
		}
		r := &state.Document.Objects[index-1]
		key := uint32(0)
		if !atCell {
			var err error
			key, err = savedCurrentOrderActorKey(state, w.Entities(), target)
			if err != nil {
				// A target removed during the same tick is explicitly absent in
				// the continuation, not a dangling numeric pointer in the wire.
				for _, e := range w.Entities() {
					if e.ID == target {
						return err
					}
				}
			}
		}
		spellKey := uint32(0)
		rangeByte := byte(0)
		if book {
			refs, _ := savedObjectRefs(r, "Spells")
			if spell >= 1 && int(spell) <= len(refs) && refs[spell-1] != 0 {
				sr := &state.Document.Objects[refs[spell-1]-1]
				var err error
				spellKey, err = savedStructureValue(sr, "This")
				if err != nil {
					return err
				}
				rangeValue, err := savedStructureValue(sr, "S09")
				if err != nil {
					return err
				}
				rangeByte = byte(rangeValue)
			}
		}
		stateCode, inner := uint32(0xd), byte(8)
		if atCell {
			stateCode, inner = 0xe, 9
		}
		wirePhase := uint32(0)
		if phase == 1 {
			wirePhase = 5
		}
		if phase == 2 {
			wirePhase = 7
		}
		// An armed creature cast is an order that has not installed its cast:
		// the actor stays in its walking state and the order sits at progress 0.
		armed := book && phase == sim.BookPhaseApproach
		if !armed {
			mustSetRaw(r, "U54", binary.LittleEndian.AppendUint32(nil, stateCode))
			mustSetRaw(r, "U58", binary.LittleEndian.AppendUint32(nil, wirePhase))
			savedObjectSetValue(r, "U5C", key)
			savedObjectSetValue(r, "U60", uint32(uint8(x)))
			savedObjectSetValue(r, "U61", uint32(uint8(y)))
			savedObjectSetValue(r, "U64", spellKey)
			// Native consumables are detached reservations, not the researched
			// weapon-borrow route. Their complete current object/child graph lives
			// in the typed supplement (DIV-1369); never retain an obsolete borrow.
			savedObjectSetRefs(r, "U68", []uint16{0}, false)
			savedObjectSetValue(r, "U6C", uint32(remaining))
			savedObjectSetValue(r, "U136", 0)
			if complete {
				savedObjectSetValue(r, "U136", 1)
			}
		}
		order, err := savedMotionRaw(r, "U158", 148)
		if err != nil {
			return err
		}
		order[8], order[9], order[0x15] = inner, 2, progress
		if armed {
			order[9] = 0
		}
		var raw [144]byte
		copy(raw[:], order)
		raw = sim.ProjectCastOrderOperands(raw, key, spellKey, x, y, rangeByte, atCell)
		if retain {
			// The creature cast selectors store 1 in the order's retention flag.
			raw[0x60] = 1
		}
		copy(order, raw[:])
		for _, e := range w.Entities() {
			if e.ID == caster && e.Retreat.Known {
				current := sim.ProjectActorRetreatOrder(e, sim.SavedActorOrder{Raw: raw})
				mustSetRaw(r, "U50", binary.LittleEndian.AppendUint32(nil, current.State))
				copy(order[2:4], current.Raw[2:4])
				copy(order[8:10], current.Raw[8:10])
				break
			}
		}
		return nil
	}
	for _, c := range actions.Books {
		caster, _ := w.Entity(c.Caster)
		if err := write(c.Caster, c.Target, c.AtCell, c.X, c.Y, c.Spell, c.Remaining, c.Phase, c.Progress, c.Complete, true,
			c.Retained && creatureHoldsSlots(caster)); err != nil {
			return err
		}
	}
	for _, c := range actions.Scrolls {
		id, _, ok := sim.ScrollSpell(c.Item)
		if !ok {
			return fmt.Errorf("current reservation is not a scroll")
		}
		if e, ok := w.Entity(c.Caster); ok && e.PendingOrder.Kind == sim.PendingScroll {
			index := actors[c.Caster]
			if index == 0 {
				return fmt.Errorf("pending scroll has no caster object")
			}
			r := &state.Document.Objects[index-1]
			savedObjectSetRefs(r, "U68", []uint16{items[c.Item.ObjectID]}, false)
			stateCode, target := uint32(0xd), uint32(0)
			if c.AtCell {
				stateCode = 0xe
			} else {
				var err error
				target, err = savedCurrentOrderActorKey(state, w.Entities(), c.Target)
				if err != nil {
					return err
				}
			}
			mustSetRaw(r, "U50", binary.LittleEndian.AppendUint32(nil, stateCode))
			order, err := savedMotionRaw(r, "U158", 148)
			if err != nil {
				return err
			}
			rule, _ := w.Spell(uint32(id))
			var raw [144]byte
			copy(raw[:], order)
			raw = sim.ProjectCastOrderOperands(raw, target, 0, c.X, c.Y, rule.MaxRange, c.AtCell)
			copy(order, raw[:])
			order[8] = 0
			if e.PendingOrder.RowAdmitted {
				order[8], order[9] = 8, 2
				if c.AtCell {
					order[8] = 9
				}
				mustSetRaw(r, "U54", binary.LittleEndian.AppendUint32(nil, stateCode))
				savedObjectSetValue(r, "U5C", target)
				savedObjectSetValue(r, "U60", uint32(uint8(c.X)))
				savedObjectSetValue(r, "U61", uint32(uint8(c.Y)))
				savedObjectSetValue(r, "U64", 0)
			}
			continue
		}
		phase := uint8(0)
		if c.Started {
			phase = 1
		}
		if err := write(c.Caster, c.Target, c.AtCell, c.X, c.Y, id, c.Remaining, phase, 0, false, false, false); err != nil {
			return err
		}
	}
	return nil
}
