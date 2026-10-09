package game

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/sim"
)

// HERO-CADENCE-023/112/114 own +54/+58/+6c/+136 and the two
// intervening actor turns. The common deadline producer owns +138.
// No client animation frame is encoded here.
func projectSavedActorActions(state *SnapshotSAVDocument, world *sim.World) error {
	entities := world.Entities()
	byID := map[sim.EntityID]sim.Entity{}
	for _, e := range entities {
		byID[e.ID] = e
	}
	boundaryIdleMotion := map[sim.EntityID]bool{}
	motions, _, _, _ := world.SavedActorMotions()
	for _, motion := range motions {
		if motion.Current && motion.ActorAction == 0 {
			boundaryIdleMotion[motion.Entity] = true
		}
	}
	boundaryIdleOrder := map[sim.EntityID]bool{}
	_, orders, _ := world.SavedGroups()
	for _, order := range orders {
		if order.Raw[9] == 0 {
			boundaryIdleOrder[order.Entity] = true
		}
	}
	// The World drops an attack target that names a body below the targetable
	// floor. Until the World advances past the restore, such an order keeps the
	// words the LOAD read, so a save before the first tick does not turn a
	// restored order into a half-cleared one.
	restored := state.ActionTickSet && state.ActionTick == world.Tick()
	for _, b := range state.Actors {
		if b.Retired {
			continue
		}
		e, ok := byID[b.EntityID]
		if !ok {
			return fmt.Errorf("SAV action actor %d is absent", b.EntityID)
		}
		r := &state.Document.Objects[b.ObjectIndex-1]
		credit := uint32(0)
		var err error
		if e.HasKillCredit {
			credit, err = savedCurrentOrderActorKey(state, entities, e.KillCreditSource)
			if err != nil {
				return worldSaveUnsupportedf("actor %d credit: %v", e.ID, err)
			}
		}
		// SAV-908 repairs this raw actor word on world LOAD; Effect+44 has
		// a different, omitted serializer and is deliberately not inferred.
		savedObjectSetValue(r, "U40", credit)
		savedObjectSetValue(r, "U48", uint32(uint8(e.KillCreditSpell)))
		if !e.Alive() && !e.Dying() {
			if frozen, ok := state.FrozenOrders[e.ID]; ok {
				mustSetRaw(r, "U58", binary.LittleEndian.AppendUint32(nil, frozen.Phase))
				savedObjectSetValue(r, "U136", frozen.Complete)
			}
			continue
		}
		bodyOrder, keepLoadedOrder := state.BodyOrders[e.ID]
		if keepLoadedOrder {
			body, present := byID[bodyOrder.Target]
			keepLoadedOrder = restored && present && !body.OrdinaryTargetable() && !e.HasAttackTarget && !e.HasPendingAttackTarget && !e.Retreat.Known && e.PendingOrder.Kind == sim.PendingNone
		}
		if keepLoadedOrder {
			bodyKey, err := currentActionTargetKey(state, world, bodyOrder.Target, sim.AttackTargetUnit)
			if err != nil {
				return worldSaveUnsupportedf("actor %d restored order: %v", e.ID, err)
			}
			mustSetRaw(r, "U58", binary.LittleEndian.AppendUint32(nil, bodyOrder.Phase))
			savedObjectSetValue(r, "U5C", bodyKey)
			savedObjectSetValue(r, "U6C", bodyOrder.Countdown)
			savedObjectSetValue(r, "U136", bodyOrder.Complete)
		}
		key, phase, complete := uint32(0), uint32(0), uint32(0)
		if e.HasAttackTarget || e.Retreat.Known {
			if e.HasAttackTarget {
				key, err = currentActionTargetKey(state, world, e.AttackTarget, e.AttackTargetKind)
				if err != nil {
					return worldSaveUnsupportedf("actor %d attack: %v", e.ID, err)
				}
			}
			switch e.AttackPhase {
			case sim.AttackReady:
			case sim.AttackCharging, sim.AttackCasting:
				phase = 5
			case sim.AttackRelaxing:
				phase = 7
			case sim.AttackBoundaryOne:
				complete = 1
			case sim.AttackBoundaryTwo:
			default:
				return worldSaveUnsupportedf("actor %d attack phase is unavailable", e.ID)
			}
			// HERO-DYINGTICK-145: the per-tick dying branch force-clears U50/U54
			// every dying tick and never reaches the order machine, so a dying
			// actor's own U54 is always its idle value here, not the attacking
			// one the alive arm below writes; the motion producer already holds
			// that field for a dying actor, and this write must not override it.
			if e.Alive() && e.Transit == 0 && !e.Turning() && !world.ActorMotionActive(e.ID) {
				action := uint32(3)
				progress := byte(1)
				if e.AttackPhase == sim.AttackBoundaryTwo ||
					e.AttackPhase == sim.AttackBoundaryOne && boundaryIdleMotion[e.ID] && boundaryIdleOrder[e.ID] {
					action, progress = 0, 0
				}
				mustSetRaw(r, "U54", binary.LittleEndian.AppendUint32(nil, action))
				for i := range r.Raw {
					if r.Raw[i].Name == "U158" {
						r.Raw[i].Bytes[9] = progress
					}
				}
			}
		}
		if e.Retreat.Known && e.Retreat.Complete {
			complete = 1
		}
		if !keepLoadedOrder {
			mustSetRaw(r, "U58", binary.LittleEndian.AppendUint32(nil, phase))
		}
		if e.Retreat.Known && e.Alive() {
			order, err := savedMotionRaw(r, "U158", 148)
			if err != nil {
				return err
			}
			retained := sim.SavedActorOrder{}
			copy(retained.Raw[:], order)
			current := sim.ProjectActorRetreatOrder(e, retained)
			mustSetRaw(r, "U50", binary.LittleEndian.AppendUint32(nil, current.State))
			copy(order[2:4], current.Raw[2:4])
			copy(order[8:10], current.Raw[8:10])
			action := uint32(0)
			switch e.Retreat.Progress {
			case 0:
				if e.HasAttackTarget && e.AttackPhase != sim.AttackReady {
					action = 3
				}
			case 1:
				action = 3
			case 3:
				action = 1
			case 4:
				action = 0x1a
			}
			if e.Retreat.Progress != 2 {
				mustSetRaw(r, "U54", binary.LittleEndian.AppendUint32(nil, action))
			}
		}
		if !keepLoadedOrder {
			savedObjectSetValue(r, "U5C", key)
		}
		if e.HasPendingAttackTarget {
			pending, err := currentActionTargetKey(state, world, e.PendingAttackTarget, e.PendingAttackTargetKind)
			if err != nil {
				return worldSaveUnsupportedf("actor %d pending attack: %v", e.ID, err)
			}
			order, err := savedMotionRaw(r, "U158", 148)
			if err != nil {
				return err
			}
			binary.LittleEndian.PutUint32(order[0x0c:], pending)
		}
		// The wire timer is a byte. A wider native retained interval is
		// explicitly carried by Actions; never modulo-truncate its next tick.
		if !keepLoadedOrder {
			savedObjectSetValue(r, "U6C", uint32(min(max(e.AttackCountdown, 0), 255)))
			savedObjectSetValue(r, "U136", complete)
		}
		if p := e.PendingOrder; p.Kind != sim.PendingNone {
			order, err := savedMotionRaw(r, "U158", 148)
			if err != nil {
				return err
			}
			order[8] = 0
			switch p.Kind {
			case sim.PendingPickup:
				mustSetRaw(r, "U50", binary.LittleEndian.AppendUint32(nil, 2))
				binary.LittleEndian.PutUint16(order[0xa:], uint16(uint8(p.X))|uint16(uint8(p.Y))<<8)
				if p.RowAdmitted {
					order[8] = 7
					action := uint32(2)
					order[9] = 0
					if e.Transit != 0 || world.ActorMotionActive(e.ID) {
						action, order[9] = 1, 3
					}
					mustSetRaw(r, "U54", binary.LittleEndian.AppendUint32(nil, action))
				}
			case sim.PendingPickupComplete:
				mustSetRaw(r, "U50", binary.LittleEndian.AppendUint32(nil, 0xc))
				binary.LittleEndian.PutUint32(order[0x50:], 1)
				if p.RowAdmitted {
					order[9] = 0
					mustSetRaw(r, "U54", binary.LittleEndian.AppendUint32(nil, 0))
				}
			case sim.PendingActorCast, sim.PendingCellCast:
				code := uint32(0xd)
				if p.Kind == sim.PendingCellCast {
					code = 0xe
				}
				mustSetRaw(r, "U50", binary.LittleEndian.AppendUint32(nil, code))
				target, err := currentActionTargetKey(state, world, p.Target, sim.AttackTargetUnit)
				if err != nil {
					return err
				}
				if p.Kind == sim.PendingCellCast {
					target = 0
				}
				refs, _ := savedObjectRefs(r, "Spells")
				spellKey, reach := uint32(0), byte(0)
				if p.Spell >= 1 && int(p.Spell) <= len(refs) && refs[p.Spell-1] != 0 {
					sr := &state.Document.Objects[refs[p.Spell-1]-1]
					spellKey, err = savedStructureValue(sr, "This")
					if err != nil {
						return err
					}
					value, err := savedStructureValue(sr, "S09")
					if err != nil {
						return err
					}
					reach = byte(value)
				}
				var raw [144]byte
				copy(raw[:], order)
				raw = sim.ProjectCastOrderOperands(raw, target, spellKey, p.X, p.Y, reach, p.Kind == sim.PendingCellCast)
				copy(order, raw[:])
				if p.RowAdmitted {
					if spellKey != 0 {
						savedObjectSetValue(r, "U44", spellKey)
					}
					order[8], order[9] = 8, 2
					if p.Kind == sim.PendingCellCast {
						order[8] = 9
					}
					x, y := p.X, p.Y
					if p.Kind == sim.PendingActorCast {
						x, y = byID[p.Target].X, byID[p.Target].Y
					}
					mustSetRaw(r, "U54", binary.LittleEndian.AppendUint32(nil, code))
					savedObjectSetValue(r, "U5C", target)
					savedObjectSetValue(r, "U60", uint32(uint8(x)))
					savedObjectSetValue(r, "U61", uint32(uint8(y)))
					savedObjectSetValue(r, "U64", spellKey)
				}
			}
		}
	}
	return nil
}

// The document's existing actor bindings provide the map. An unresolved raw
// attribution key becomes null (SAV-908); no archive ordinal, ALM ID or
// Token.Reference substitutes for it. The action decoder admits physical
// phases only. Other original actions retain their existing admission policy.
func importSavedActorActions(ms *Mission, state *SnapshotSAVDocument) error {
	if state == nil || state.Document == nil {
		return nil
	}
	byKey := map[uint32]sim.EntityID{}
	for _, b := range state.Actors {
		key, err := savedStructureValue(&state.Document.Objects[b.ObjectIndex-1], "Identity")
		if err != nil {
			return err
		}
		if _, exists := byKey[key]; exists || key == 0 {
			return fmt.Errorf("SAV action actor key is ambiguous")
		}
		byKey[key] = b.EntityID
	}
	// The byte form's own UnmarshalBinary (binary.go, normaliseTargetReferences)
	// silently drops every attack reference naming a body no longer
	// OrdinaryTargetable (HP <= -10; target.go's clearInvalidTargetReferences).
	// That is a standing world invariant, not something this reader may leave
	// to a later round trip to enforce: a decode that restores a target on such
	// a body would sit correct in memory right up until the first native
	// SAVE/LOAD silently stripped it (witnessed on the synthetic-terminal-
	// zero-timer fixture, mapID 54/entity 25 at HP -10), which is a decode bug
	// here, not a byte-form one. Applying the same admission at decode time
	// keeps the restored state consistent whether or not a round trip ever runs.
	byIDForTargeting := map[sim.EntityID]sim.Entity{}
	for _, e := range ms.World.Entities() {
		byIDForTargeting[e.ID] = e
	}
	var batch []sim.OriginalActorAction
	var bodyOrders map[sim.EntityID]SnapshotSAVBodyOrder
	bodyKeys := map[uint32]sim.EntityID{}
	for _, d := range ms.World.OriginalDeadActors() {
		if _, actor := byKey[d.Source.Identity]; !actor && d.Source.Identity != 0 {
			bodyKeys[d.Source.Identity] = d.ID
		}
	}
	// A late corpse has no action record; its frozen words are read here.
	frozen := map[sim.EntityID]SnapshotSAVFrozenOrder{}
	for i := range state.Document.Objects {
		r := &state.Document.Objects[i]
		key, err := savedStructureValue(r, "Identity")
		body, ok := bodyKeys[key]
		if err != nil || !ok || r.Class != "Human" && r.Class != "Unit" {
			continue
		}
		phase, err := savedMotionRaw(r, "U58", 4)
		if err != nil {
			return err
		}
		complete, _ := savedStructureValue(r, "U136")
		frozen[body] = SnapshotSAVFrozenOrder{Phase: binary.LittleEndian.Uint32(phase), Complete: complete}
	}
	for _, b := range state.Actors {
		r := &state.Document.Objects[b.ObjectIndex-1]
		value := func(name string) uint32 { v, _ := savedStructureValue(r, name); return v }
		a := sim.OriginalActorAction{Entity: b.EntityID, CreditSpell: int8(value("U48"))}
		a.Credit, a.HasCredit = byKey[value("U40")]
		// HERO-DYINGTICK-145/AI-332: a dying (Stage != 0) record's order block
		// is not special-cased by LOAD. Order::Serialize copies its own fixed
		// block through one raw archive read/write pair regardless of stage,
		// so a dying actor's attack target/phase/countdown restore exactly the
		// same way an alive one's do; the per-tick dying branch that never
		// reaches the order machine is why they hold the frozen value in the
		// first place, not a reason for this reader to skip them.
		actionRaw, err := savedMotionRaw(r, "U54", 4)
		if err != nil {
			return err
		}
		phaseRaw, err := savedMotionRaw(r, "U58", 4)
		if err != nil {
			return err
		}
		order, err := savedMotionRaw(r, "U158", 148)
		if err != nil {
			return err
		}
		stateRaw, err := savedMotionRaw(r, "U50", 4)
		if err != nil {
			return err
		}
		switch v := binary.LittleEndian.Uint32(stateRaw); v {
		case 3, 8, 0xa, 0xb, 0xc, 0x11:
			// savedActorStateSupported's own raw order-state set (1, 3, 8, 0xa,
			// 0xb, 0xc; savedgroupsai.go) is a DIFFERENT enum from sim's own
			// Entity.ActorState (world.go): the two agree at engage(3),
			// defend(8), patrol(0xa), guard(0xb) and acquire(0xc), but
			// ActorState has no code 1 at all (pickup-completion is 2, never
			// reached by this raw continuation field) — actorStateDefined
			// rejects it and the native encoder refuses on that account
			// (ActorRegistry1111/game0001.sav, CurrentWorldProjectileRefusal1170,
			// entity 8/56 witnessed with raw U50=1). Forwarding raw 1 here was
			// carrying the wrong enum's membership across; omit it so that raw
			// state leaves ActorState at its no-override zero value instead.
			a.ActorState, a.PostX, a.PostY = uint8(v), int32(order[0]), int32(order[1])
		}
		action, phase := binary.LittleEndian.Uint32(actionRaw), binary.LittleEndian.Uint32(phaseRaw)
		if source, ok := byIDForTargeting[b.EntityID]; ok && !source.Alive() && !source.Dying() {
			frozen[b.EntityID] = SnapshotSAVFrozenOrder{Phase: phase, Complete: value("U136")}
		}
		if action == 0 || action == 1 || action == 2 || action == 3 {
			a.Target, a.HasTarget = byKey[value("U5C")]
			if body, ok := bodyKeys[value("U5C")]; ok && !a.HasTarget && (phase == 5 || phase == 7) {
				// A late corpse has no action record; its order stays on the body.
				if bodyOrders == nil {
					bodyOrders = map[sim.EntityID]SnapshotSAVBodyOrder{}
				}
				bodyOrders[b.EntityID] = SnapshotSAVBodyOrder{Target: body, Phase: phase, Countdown: value("U6C"), Complete: value("U136")}
			}
			if a.HasTarget && a.Target == b.EntityID {
				// A boundary-phase idle actor (AreaDamage1164/1111/1170 corpus
				// fixtures, witnessed at full HP with U54=U58=U6C=0) carries its
				// own Identity key back in U5C: this raw record's null-target
				// convention is self-reference, not the omitted 0 that "SAV
				// action actor key is ambiguous" above already reserves.
				// sim.ImportOriginalActorActions treats target==self as
				// definitionally invalid (never a real attack), so this parses
				// as no restored target rather than a hard load failure.
				a.HasTarget = false
			}
			if a.HasTarget {
				if source, ok := byIDForTargeting[b.EntityID]; !ok || (!source.Alive() && !source.Dying()) {
					// HERO-DYINGTICK-145's frozen-order window is bounded by
					// Dying() itself: projectSavedActorActions stops updating
					// U54/U58/U5C/U6C/U158 the same tick a source leaves it
					// (its own !Alive()&&!Dying() gate), so a record for a
					// source that has decayed past that point carries only a
					// stale leftover from an earlier stage, not a continuing
					// order. sim.ImportOriginalActorActions refuses such a
					// source outright rather than silently drop it, so this
					// reader normalises it first, the same way a self-
					// reference already is above.
					a.HasTarget = false
				}
			}
			if a.HasTarget {
				if target, ok := byIDForTargeting[a.Target]; !ok || !target.OrdinaryTargetable() {
					// The dying/late-corpse import that ran ahead of this one
					// (applyOriginalDead/applyOriginalDying, both pre-existing)
					// already carried this body past the finishing-blow floor;
					// no live attacker keeps a real order on it. Same normal-
					// isation as the self-reference case above.
					if ok && (phase == 5 || phase == 7) {
						if bodyOrders == nil {
							bodyOrders = map[sim.EntityID]SnapshotSAVBodyOrder{}
						}
						bodyOrders[b.EntityID] = SnapshotSAVBodyOrder{Target: a.Target, Phase: phase, Countdown: value("U6C"), Complete: value("U136")}
					}
					a.HasTarget = false
				}
			}
			if a.HasTarget {
				a.Countdown = int32(value("U6C"))
				switch phase {
				case 5:
					a.Phase = sim.AttackCharging
				case 7:
					a.Phase = sim.AttackRelaxing
				case 0:
					if value("U136") != 0 {
						a.Phase = sim.AttackBoundaryOne
					} else if action == 0 && order[9] == 0 {
						a.Phase = sim.AttackBoundaryTwo
					}
				default:
					a.HasTarget = false
				}
			}
		}
		if a.HasTarget && a.Phase != sim.AttackReady && a.ActorState == 3 && value("Stage") == 0 && order[9] == 1 && (order[8] == 0 || order[8] == 2 || order[8] == 5 || order[8] == 6) {
			pending, present := byKey[binary.LittleEndian.Uint32(order[0x0c:])]
			actor, actorPresent := byIDForTargeting[a.Entity]
			if target, ok := byIDForTargeting[pending]; present && ok && target.OrdinaryTargetable() && pending != a.Entity && pending != a.Target && actorPresent && !actor.HasTarget && actor.Transit == 0 {
				a.PendingAttackTarget, a.HasPendingAttackTarget = pending, true
			}
		}
		batch = append(batch, a)
	}
	if err := ms.World.ImportOriginalActorActions(batch); err != nil {
		return err
	}
	state.ActionTick, state.ActionTickSet, state.BodyOrders = ms.World.Tick(), true, bodyOrders
	if len(frozen) != 0 {
		state.FrozenOrders = frozen
	}
	return nil
}
