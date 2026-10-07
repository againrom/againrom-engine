package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/sim"
)

// Current native targets are typed Entity fields, not the retained order's
// source address words. These supported setters reuse an existing actor/order
// record; this is not a generated actor or Group constructor.
// AI-PATROL-018 and SAV-PATROLCURSOR-571 own the current ring/cursor/latch.
// AI-FOLLOWSET-116, DEFEND-111 and FOLLOW-112 own +10, closing +18 and +70.
// SAV-HUMRESUME-460 requires stage zero for order reference repair and does not
// turn a missing lookup into null. Never emit a guessed target key.
func projectSavedCurrentOrder(state *SnapshotSAVDocument, entities []sim.Entity, index uint16, order sim.SavedActorOrder, worlds ...*sim.World) (string, error) {
	gap := func(detail string) (string, error) {
		return fmt.Sprintf("actor %d SAV order: %s", order.Entity, detail), nil
	}
	if state == nil || state.Document == nil || index == 0 || int(index) > len(state.Document.Objects) {
		return "", fmt.Errorf("saved SAV current order lacks its document/world/object")
	}
	if len(worlds) != 0 {
		return "", projectNativeOrder(state, worlds[0], entities, index, order)
	}
	if !order.Authored && order.State != 3 && order.State != 8 && order.State != 0x11 {
		return "", projectSavedOrderFields(&state.Document.Objects[index-1], order)
	}
	if order.Authored {
		switch order.State {
		case 3, 0xa, 0xb, 0xc, 8, 0x11:
		default:
			return gap("native-authored state has no original field producer")
		}
	}
	var actor sim.Entity
	found := false
	for _, e := range entities {
		if e.ID == order.Entity {
			actor, found = e, true
			break
		}
	}
	if !found {
		return gap("current actor is absent")
	}
	// AI-PURSUE-040: guard/acquire/patrol retain their outer policy while
	// inner 5/6 pursue the actor at +0c. The live typed target owns that word.
	pursuit := order.Raw[8] == 5 || order.Raw[8] == 6
	if order.State == 3 || pursuit || actor.HasAttackTarget && (order.State == 0xa || order.State == 0xb || order.State == 0xc) {
		// The first source dispatch has not run yet. Preserve its exact
		// source word; absence of a native target is not a null-key writer.
		if !order.Authored && !actor.HasAttackTarget {
			return "", projectSavedOrderFields(&state.Document.Objects[index-1], order)
		}
		if !actor.HasAttackTarget || actor.AttackTargetKind != sim.AttackTargetUnit || actor.AttackTarget == actor.ID {
			return gap("engagement lacks a distinct typed actor target")
		}
		stage, err := savedStructureValue(&state.Document.Objects[index-1], "Stage")
		if err != nil {
			return "", err
		}
		if stage != 0 || order.RepairStage != 0 {
			return gap("actor stage skips original order-key repair")
		}
		id, kind, requested := actor.RequestedAttackTarget()
		if !requested || kind != sim.AttackTargetUnit {
			return gap("engagement lacks a requested actor target")
		}
		key, err := savedCurrentOrderActorKey(state, entities, id)
		if err != nil {
			return gap(err.Error())
		}
		current := order
		binary.LittleEndian.PutUint32(current.Raw[0x0c:], key)
		current.Authored = false
		return "", projectSavedOrderFields(&state.Document.Objects[index-1], current)
	}
	// Pursuit/casts/pickup use different pointer slots and runtime substate.
	// Their typed producers must supply those fields, not borrow stale Raw.
	switch order.Raw[8] {
	case 0, 1, 0xb:
	case 4:
		if order.State != 8 && order.State != 0x11 {
			return gap("closing order has no typed escort authority")
		}
	default:
		return gap("active inner order requires another target producer")
	}
	if actor.HasAttackTarget {
		return gap(fmt.Sprintf("active attack target requires its own producer (outer %d inner %d)", order.State, order.Raw[8]))
	}
	current := order
	if order.State == 0xa && (len(order.Patrol) == 0 || !slices.Contains(order.Patrol, binary.LittleEndian.Uint16(order.Raw[2:]))) {
		return gap("patrol cursor is absent from the current ring")
	}
	if order.State == 8 || order.State == 0x11 {
		stage, err := savedStructureValue(&state.Document.Objects[index-1], "Stage")
		if err != nil {
			return "", err
		}
		if stage != 0 || order.RepairStage != 0 {
			return gap("actor stage skips original order-key repair")
		}
		target, bound := order.EscortTarget, order.EscortBound
		if order.Authored {
			target, bound = actor.EscortTarget, actor.HasEscortTarget
		}
		if !bound || target == actor.ID {
			return gap("escort has no distinct bound target")
		}
		key, err := savedCurrentOrderActorKey(state, entities, target)
		if err != nil {
			return gap(err.Error())
		}
		binary.LittleEndian.PutUint32(current.Raw[0x10:], key)
		if order.Authored {
			current.Raw[0x70] = actor.EscortRange
		}
		if current.Raw[8] == 4 {
			binary.LittleEndian.PutUint32(current.Raw[0x18:], key)
			current.Raw[0x14] = current.Raw[0x70]
			if current.Raw[0x14] == 0 {
				current.Raw[0x14] = actor.ScanRange
			}
		}
	}
	// The field helper still refuses an unprocessed authored record. Only this
	// bounded producer may hand it one with current typed references resolved.
	current.Authored = false
	return "", projectSavedOrderFields(&state.Document.Objects[index-1], current)
}

func savedCurrentOrderActorKey(state *SnapshotSAVDocument, entities []sim.Entity, target sim.EntityID) (uint32, error) {
	found := false
	for _, e := range entities {
		if e.ID == target {
			found = true
			break
		}
	}
	if !found {
		return 0, fmt.Errorf("target is no longer present")
	}
	var index uint16
	for _, a := range state.Actors {
		if a.EntityID == target && !a.Retired {
			index = a.ObjectIndex
			break
		}
	}
	if index == 0 && state.GroupBindings != nil {
		for _, m := range state.GroupBindings.Members {
			if m.Bound && m.EntityID == target {
				index = m.ObjectIndex
				break
			}
		}
	}
	if index == 0 || int(index) > len(state.Document.Objects) {
		return 0, fmt.Errorf("target lacks an exact persisted object binding")
	}
	key, err := savedStructureValue(&state.Document.Objects[index-1], "Identity")
	if err != nil || key == 0 {
		return 0, fmt.Errorf("target lacks a nonzero serializable key")
	}
	for i, record := range state.Document.Objects {
		for _, value := range record.Values {
			if (value.Name == "Identity" || value.Name == "This") && value.Value == key && i+1 != int(index) {
				return 0, fmt.Errorf("target key collides with a distinct object")
			}
		}
	}
	return key, nil
}
