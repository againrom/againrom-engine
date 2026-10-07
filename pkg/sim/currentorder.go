package sim

import (
	"encoding/binary"
	"fmt"
)

// ProjectActorOrderTargets changes typed endpoints, preserving other bytes.
func ProjectActorOrderTargets(e Entity, o SavedActorOrder, attack, escort uint32) SavedActorOrder {
	attackOrder := o.State == 3 || o.Raw[8] == 2 || o.Raw[8] == 5 || o.Raw[8] == 6
	_, _, hasRequested := e.RequestedAttackTarget()
	if attackOrder && (hasRequested || o.Authored) {
		if !hasRequested {
			attack = 0
		}
		binary.LittleEndian.PutUint32(o.Raw[0xc:], attack)
	}
	if e.HasEscortTarget || o.Authored && (o.State == 8 || o.State == 0x11) {
		if !e.HasEscortTarget {
			escort = 0
		}
		binary.LittleEndian.PutUint32(o.Raw[0x10:], escort)
		o.Raw[0x70] = e.EscortRange
		if o.Raw[8] == 4 {
			binary.LittleEndian.PutUint32(o.Raw[0x18:], escort)
			o.Raw[0x14] = e.EscortRange
		}
	}
	return o
}

func ProjectActorRetreatOrder(e Entity, o SavedActorOrder) SavedActorOrder {
	if e.Retreat.Known {
		o.Raw[9] = e.Retreat.Progress
	}
	if e.ActorState == actorStateRetreat && e.Retreat.Known {
		o.State, o.Raw[8], o.Raw[9] = uint32(actorStateRetreat), 0, e.Retreat.Progress
		if e.Retreat.Pending {
			o.Raw[8] = 1
			binary.LittleEndian.PutUint16(o.Raw[2:], uint16(e.Retreat.X)|uint16(e.Retreat.Y)<<8)
		}
	}
	return o
}

// ProjectCastOrderOperands writes a current cast's operands. Progress 2
// restores its action from byte +0x5c (AI-RETREAT-272): 1 casts at the unit,
// 0 at the cell (DIV-1491).
func ProjectCastOrderOperands(raw [144]byte, target, spell uint32, x, y int32, radius uint8, atCell bool) [144]byte {
	raw[0x14] = radius
	binary.LittleEndian.PutUint32(raw[0x28:], target)
	binary.LittleEndian.PutUint32(raw[0x30:], spell)
	binary.LittleEndian.PutUint16(raw[0x3c:], uint16(uint8(x))|uint16(uint8(y))<<8)
	raw[0x5c] = 1
	if atCell {
		raw[0x5c] = 0
	}
	return raw
}

func ProjectMotionOrderTarget(raw [144]byte, e Entity) [144]byte {
	if e.HasTarget {
		binary.LittleEndian.PutUint16(raw[2:], uint16(e.TargetX)|uint16(e.TargetY)<<8)
	}
	return raw
}

// Native order insertion order is independent of archive traversal. The
// ordinary rows retain all their values while this identity-only permutation
// restores that sequence. Historical continuations omit every ordinal.
func restoreActionOrderOrdinals(groups *savedGroupState, rows []ActorContinuation) error {
	ordinals := map[EntityID]uint32{}
	for _, row := range rows {
		if row.Order != nil && row.Order.Ordinal != nil {
			ordinals[row.Entity] = *row.Order.Ordinal
		}
	}
	if len(ordinals) == 0 {
		return nil
	}
	if groups == nil || len(ordinals) != len(groups.Orders) {
		return fmt.Errorf("sim: incomplete current order ordinals")
	}
	ordered := make([]SavedActorOrder, len(groups.Orders))
	seen := make([]bool, len(ordered))
	for _, row := range groups.Orders {
		ordinal, ok := ordinals[row.Entity]
		if !ok || uint64(ordinal) >= uint64(len(ordered)) || seen[ordinal] {
			return fmt.Errorf("sim: invalid current order ordinal")
		}
		seen[ordinal], ordered[ordinal] = true, row
	}
	groups.Orders = ordered
	return nil
}
