package sim

import "slices"

type SourceEquipmentPlaceKind uint8

const (
	SourceEquipmentPack SourceEquipmentPlaceKind = iota + 1
	SourceEquipmentWorn
	SourceEquipmentExternal
)

// Pack and Worn use zero-based indices. External has index zero.
type SourceEquipmentPlace struct {
	Kind  SourceEquipmentPlaceKind
	Index int
}

type SourceEquipmentEventKind uint8

const (
	SourceEquipmentTake SourceEquipmentEventKind = iota + 1
	SourceEquipmentPut
	SourceEquipmentSpellWrite
)

// Ticket identifies one detached occurrence within a single operation. It is
// neither a native object handle nor a persisted graph identity. Split is a
// one-unit take from a stack whose count exceeded one; Merged identifies an
// existing destination position selected by the gameplay merge predicate.
type SourceEquipmentEvent struct {
	Kind   SourceEquipmentEventKind
	Ticket uint32
	Place  SourceEquipmentPlace
	Split  bool
	Merged bool
	// Count is the moved quantity for topology receipts; legacy receipts use zero.
	Count uint32
	// Spell belongs only to SpellWrite events. A zero value is an explicit
	// teardown. It is an operation result, never a persisted second value.
	Spell SourceItemSpell
}

type SourceEquipmentReceipt struct {
	Events []SourceEquipmentEvent
	// Reservation is the independent session handle for a bound item returned
	// to an external caller. Unbound values keep both native handles absent.
	Reservation uint64
}

// The optional operation is ephemeral. Receipt is replaced only after success.
// PackAliases names exact other Pack positions for a receipt without topology.
type SourceEquipmentOperation struct {
	Topology      *SourceEquipmentTopology
	Receipt       *SourceEquipmentReceipt
	SessionHandle uint64
	PackAliases   []int
}

func (w *World) sourceEquipmentOperation(id EntityID, incoming *ItemInstance, operations []SourceEquipmentOperation) (*sourceEquipmentTrace, SourceEquipmentOperation, bool) {
	if len(operations) == 0 {
		return nil, SourceEquipmentOperation{}, true
	}
	if len(operations) != 1 {
		return nil, SourceEquipmentOperation{}, false
	}
	op := operations[0]
	if op.Topology != nil && len(op.PackAliases) != 0 || op.SessionHandle != 0 && (incoming == nil || incoming.ObjectID == 0) {
		return nil, op, false
	}
	if op.Topology != nil {
		t, ok := w.equipmentTopologyTrace(id, *op.Topology, incoming)
		return t, op, ok
	}
	if op.Receipt != nil || len(op.PackAliases) != 0 {
		return &sourceEquipmentTrace{aliases: slices.Clone(op.PackAliases)}, op, true
	}
	return nil, op, true
}

func (op SourceEquipmentOperation) publish(t *sourceEquipmentTrace, reservation uint64) {
	if op.Receipt != nil {
		*op.Receipt = SourceEquipmentReceipt{Events: slices.Clone(t.events), Reservation: reservation}
	}
}

type sourceEquipmentTrace struct {
	events   []SourceEquipmentEvent
	next     uint32
	aliases  []int
	topology *sourceEquipmentTopologyTrace
}

func (t *sourceEquipmentTrace) validAliases(stacks []ItemStack, selected int) bool {
	if t != nil && t.topology != nil {
		t.aliases = nil
		for at, node := range t.topology.pack {
			if at != selected && node == t.topology.pack[selected] {
				t.aliases = append(t.aliases, at)
			}
		}
	}
	if t == nil || len(t.aliases) == 0 {
		return true
	}
	item := stacks[selected]
	if item.ObjectID != 0 {
		return false
	}
	for i, index := range t.aliases {
		if index < 0 || index >= len(stacks) || index == selected || slices.Contains(t.aliases[:i], index) || !StackStateEqual(item, stacks[index]) {
			return false
		}
	}
	return true
}

func (t *sourceEquipmentTrace) take(kind SourceEquipmentPlaceKind, index int, split bool) uint32 {
	if t == nil {
		return 0
	}
	t.next++
	event := SourceEquipmentEvent{Kind: SourceEquipmentTake, Ticket: t.next, Place: SourceEquipmentPlace{kind, index}, Split: split}
	if t.topology != nil {
		t.topology.take(t.next, kind, index, split)
		event.Count = t.quantity(t.next)
	}
	t.events = append(t.events, event)
	return t.next
}

func (t *sourceEquipmentTrace) put(ticket uint32, kind SourceEquipmentPlaceKind, index int, merge ...bool) {
	if t != nil {
		merged := len(merge) != 0 && merge[0]
		event := SourceEquipmentEvent{Kind: SourceEquipmentPut, Ticket: ticket, Place: SourceEquipmentPlace{kind, index}, Merged: merged}
		if t.topology != nil {
			event.Count = t.quantity(ticket)
			t.topology.put(ticket, kind, index, merged)
		}
		t.events = append(t.events, event)
	}
}

func (t *sourceEquipmentTrace) quantity(ticket uint32) uint32 {
	if t != nil && t.topology != nil {
		return t.topology.counts[t.topology.tickets[ticket]]
	}
	return 1
}

func (t *sourceEquipmentTrace) spell(ticket uint32, value SourceItemSpell) {
	if t != nil {
		t.events = append(t.events, SourceEquipmentEvent{Kind: SourceEquipmentSpellWrite, Ticket: ticket, Spell: value})
	}
}

// CloneSplitItemValue copies one unit's scalar payload and applies the owned
// Spell constructor. The caller assigns the new identity separately. Missing
// rules leave the same explicit ID-only Spell that registry splits retain.
func CloneSplitItemValue(item ItemInstance, rules []SpellRule) ItemInstance {
	out := item.Clone()
	out.ObjectID = 0
	if spell := out.SourceEquipment.Spell; spell.Present {
		out.SourceEquipment.Spell = SourceItemSpell{Present: true, ID: spell.ID}
		for _, rule := range rules {
			if rule.ID == uint16(spell.ID) {
				out.SourceEquipment.Spell = sourceSpellFromRule(rule)
				break
			}
		}
	}
	return out
}
