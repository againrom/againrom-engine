package sim

import "slices"

// These opaque tokens identify current nodes only for this operation. They
// never enter ItemInstance.ObjectID or the World's persisted state.
// Sparse counts use an explicit shared occurrence's quantity, or one when
// zero. A Pack occurrence supplies its exact count; conflicts are invalid.
type SourceEquipmentTopology struct {
	Pack          []uint64
	Worn          [EquipSlots]uint64
	WornCounts    [EquipSlots]uint32
	External      uint64
	ExternalCount uint32
}

type sourceEquipmentTopologyTrace struct {
	actor    int
	pack     []uint32
	worn     [EquipSlots]uint32
	external uint32
	next     uint32
	tickets  map[uint32]uint32
	spells   map[uint32]SourceItemSpell
	counts   map[uint32]uint32
}

func (w *World) equipmentTopologyTrace(id EntityID, topology SourceEquipmentTopology, incoming *ItemInstance) (*sourceEquipmentTrace, bool) {
	i := indexOfEntity(w.entities, id)
	if i < 0 || w.savedObjects != nil || len(topology.Pack) != len(w.carried[i]) || incoming == nil && (topology.External != 0 || topology.ExternalCount != 0) {
		return nil, false
	}
	t := &sourceEquipmentTopologyTrace{actor: i, tickets: map[uint32]uint32{}, spells: map[uint32]SourceItemSpell{}, counts: map[uint32]uint32{}}
	ids, values := map[uint64]uint32{}, map[uint64]ItemInstance{}
	counts := map[uint64]uint32{}
	declareCount := func(token uint64, count uint32) bool {
		if count == 0 {
			return true
		}
		if token == 0 || count > MaxOriginalHoldingValues {
			return false
		}
		if old, exists := counts[token]; exists && old != count {
			return false
		}
		counts[token] = count
		return true
	}
	bind := func(token uint64, value ItemInstance, count uint32, counted bool) (uint32, bool) {
		if value.ObjectID != 0 || (token == 0) != value.Empty() {
			return 0, false
		}
		if token == 0 {
			return 0, true
		}
		if old, ok := values[token]; ok && !StackStateEqual(StackItem(old, 1), StackItem(value, 1)) {
			return 0, false
		}
		if counted {
			if count == 0 || !declareCount(token, count) {
				return 0, false
			}
		}
		if ids[token] == 0 {
			t.next++
			ids[token], values[token] = t.next, value.Clone()
		}
		t.counts[ids[token]] = counts[token]
		return ids[token], true
	}
	for at, value := range w.carried[i] {
		node, ok := bind(topology.Pack[at], value.Instance(), value.Count, true)
		if !ok {
			return nil, false
		}
		t.pack = append(t.pack, node)
	}
	for slot, token := range topology.Worn {
		if !declareCount(token, topology.WornCounts[slot]) {
			return nil, false
		}
	}
	if !declareCount(topology.External, topology.ExternalCount) {
		return nil, false
	}
	for slot, value := range w.equipment[i] {
		count := counts[topology.Worn[slot]]
		if count == 0 && !value.Empty() {
			count = 1
		}
		node, ok := bind(topology.Worn[slot], value, count, true)
		if !ok {
			return nil, false
		}
		t.worn[slot] = node
	}
	if incoming != nil {
		count := counts[topology.External]
		if count == 0 && !incoming.Empty() {
			count = 1
		}
		node, ok := bind(topology.External, *incoming, count, true)
		if !ok {
			return nil, false
		}
		t.external = node
	}
	return &sourceEquipmentTrace{topology: t}, true
}

func (t *sourceEquipmentTopologyTrace) take(ticket uint32, kind SourceEquipmentPlaceKind, index int, split bool) {
	var node uint32
	switch kind {
	case SourceEquipmentPack:
		node = t.pack[index]
		if split {
			t.counts[node]--
			t.next++
			node = t.next
			t.counts[node] = 1
		} else {
			t.pack = slices.Delete(t.pack, index, index+1)
		}
	case SourceEquipmentWorn:
		node = t.worn[index]
	case SourceEquipmentExternal:
		node, t.external = t.external, 0
	}
	t.tickets[ticket] = node
}

func (t *sourceEquipmentTopologyTrace) put(ticket uint32, kind SourceEquipmentPlaceKind, index int, merged bool) {
	node := t.tickets[ticket]
	switch kind {
	case SourceEquipmentPack:
		if !merged {
			t.pack = slices.Insert(t.pack, index, node)
		}
	case SourceEquipmentWorn:
		t.worn[index] = node
	case SourceEquipmentExternal:
		t.external = node
	}
}

func (t *sourceEquipmentTrace) detachWorn(slot int) {
	if t != nil && t.topology != nil {
		t.topology.worn[slot] = 0
	}
}

func (t *sourceEquipmentTrace) currentSpell(ticket uint32, item *ItemInstance) {
	if t != nil && t.topology != nil {
		if value, ok := t.topology.spells[t.topology.tickets[ticket]]; ok {
			item.SourceEquipment.Spell = value
		}
	}
}

func (w *World) sourceTraceSpell(t *sourceEquipmentTrace, ticket uint32, value SourceItemSpell) bool {
	t.spell(ticket, value)
	if t == nil || t.topology == nil {
		return true
	}
	g := t.topology
	node := g.tickets[ticket]
	g.spells[node] = value
	before, changed := w.beginLoadMutation(g.actor), false
	for at, id := range g.pack {
		if id == node {
			w.carried[g.actor][at].SourceEquipment.Spell = value
			changed = true
		}
	}
	for slot, id := range g.worn {
		if id == node {
			old := w.equipment[g.actor][slot]
			item := old.Clone()
			item.SourceEquipment.Spell = value
			w.equipment[g.actor][slot] = item
			w.applyEquipmentItemState(&w.entities[g.actor], old, item, w.spells, w.damageObservation)
			if slot == slotWeapon {
				syncWeaponItem(&w.entities[g.actor], item)
			}
			changed = true
		}
	}
	return !changed || w.finishLoadMutation(g.actor, before)
}

func (w *World) sourceTopologyPutCarried(i int, item ItemInstance, trace *sourceEquipmentTrace, ticket uint32) bool {
	if !w.hasActorContainer(i) {
		return false
	}
	g := trace.topology
	quantity := g.counts[g.tickets[ticket]]
	if quantity == 0 || quantity > MaxOriginalHoldingValues {
		return false
	}
	for at, held := range w.carried[i] {
		if held.Code == 0 || held.Count == 0 || g.pack[at] == g.tickets[ticket] || !savedNativeMerge(held, StackItem(item, quantity)) {
			continue
		}
		if uint64(held.Count)+uint64(quantity) > MaxOriginalHoldingValues {
			return false
		}
		for alias, node := range g.pack {
			if node == g.pack[at] {
				w.carried[i][alias].Count += quantity
				if alias != at {
					w.entities[i].ActorLoad.Accumulator += int32(held.Weight) * int32(quantity)
				}
			}
		}
		g.counts[g.pack[at]] += quantity
		trace.put(ticket, SourceEquipmentPack, at, true)
		return true
	}
	index, ok := w.addCarriedUnmerged(i, StackItem(item, quantity))
	if !ok {
		return false
	}
	trace.put(ticket, SourceEquipmentPack, index)
	return true
}
