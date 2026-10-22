package sim

import (
	"cmp"
	"fmt"
	"slices"
)

// PassiveBody is what a terminal actor still wore and carried when the World
// removed its body. Nothing in play reads it: it drops no loot, returns no
// entity and enters no digest. SAVE writes it on the actor's dead record and a
// LOAD of that record restores it. Body is the removed actor itself, kept for a
// departed native actor without a map placement, which the SAV writer cannot
// reconstruct from the map and the tables.
type PassiveBody struct {
	ID      EntityID
	Body    *Entity `json:",omitempty"`
	Worn    [EquipSlots]ItemInstance
	Carried []ItemStack `json:",omitempty"`
}

const (
	maxPassiveCarried = 65535
	maxPassiveCount   = 65535
)

// Clone returns a copy that shares no slice or pointer with b.
func (b PassiveBody) Clone() PassiveBody {
	for slot := range b.Worn {
		b.Worn[slot] = b.Worn[slot].Clone()
	}
	carried := make([]ItemStack, len(b.Carried))
	for i, st := range b.Carried {
		carried[i] = st.Clone()
	}
	if len(carried) == 0 {
		carried = nil
	}
	b.Carried = carried
	if b.Body != nil {
		body := *b.Body
		b.Body = &body
	}
	return b
}

func (b PassiveBody) empty() bool {
	for _, item := range b.Worn {
		if !item.Empty() {
			return false
		}
	}
	return len(b.Carried) == 0
}

// passiveItem is an item value in the form its SAV record gives back: no
// registry or record identity, a stated weight, and no weapon definition or
// effects marker, which an item record does not carry. The SAV writer gives
// every passive item a fresh record.
func (w *World) passiveItem(item ItemInstance) ItemInstance {
	item = item.Clone()
	item.ObjectID, item.NativeRecord = 0, nil
	if !item.WeightPresent {
		item.Weight, item.WeightPresent = int16(w.itemWeightOf(item.Code)), true
	}
	item.SourceEquipment.Definition = SourceWeaponDefinition{}
	item.SourceEquipment.EffectsUnsupported = false
	return item
}

// passiveStacks is a container in passive item form, a stack wider than one
// record's count split as the SAV writer splits it.
func (w *World) passiveStacks(carried []ItemStack) []ItemStack {
	var out []ItemStack
	for _, st := range carried {
		item := w.passiveItem(st.Instance())
		for count := st.Count; count != 0; {
			part := min(count, uint32(maxPassiveCount))
			out = append(out, StackItem(item, part))
			count -= part
		}
	}
	return out
}

func passiveItemForm(item ItemInstance) bool {
	return item.ObjectID == 0 && item.NativeRecord == nil && item.Code != 0 && item.WeightPresent && item.ValidateWeight() == nil &&
		item.SourceEquipment.Definition == (SourceWeaponDefinition{}) && !item.SourceEquipment.EffectsUnsupported
}

// passiveEntity is the removed actor without references into the World.
func passiveEntity(e Entity) Entity {
	e.attackNotice = attackNotice{}
	e.NativeBasis = NativeActorBasis{}
	e.clearKillCredit()
	e.clearAttack()
	e.clearPendingAttack()
	e.HasEscortTarget, e.EscortTarget = false, 0
	e.Pursuit.Held, e.Pursuit.Victim = false, 0
	if e.PendingOrder.Kind == PendingActorCast {
		e.PendingOrder = PendingOrder{}
	}
	if e.HeldOrder.Kind == HeldOrderBody {
		e.HeldOrder = HeldOrder{}
	}
	return e
}

// PassiveBodies returns the retained contents in ID order.
func (w *World) PassiveBodies() []PassiveBody {
	out := make([]PassiveBody, len(w.passiveBodies))
	for i, b := range w.passiveBodies {
		out[i] = b.Clone()
	}
	return out
}

// retainPassiveBody keeps the worn and carried items of entity i when its
// removal left a terminal row with a body. It runs after the terminal row is
// written and before the entity leaves the slices.
func (w *World) retainPassiveBody(i int) {
	e := w.entities[i]
	if !w.passiveBodyOwner(e.ID) {
		return
	}
	b := PassiveBody{ID: e.ID}
	for slot, item := range w.equipment[i] {
		if !item.Empty() {
			b.Worn[slot] = w.passiveItem(item)
		}
	}
	b.Carried = w.passiveStacks(w.carried[i])
	if len(b.Carried) > maxPassiveCarried {
		b.Carried = b.Carried[:maxPassiveCarried]
	}
	if w.heldWeaponOnly(e.ID, b.Worn) {
		// A Unit record whose one worn object is its held weapon loads that
		// weapon as the dead row's own held weapon, which this capture
		// does not write. DIV-3953.
		b.Worn[slotWeapon] = ItemInstance{}
	}
	if b.empty() {
		return
	}
	if e.MapUnitID == 0 && e.SourceBinding.Class == 0 {
		body := passiveEntity(e)
		b.Body = &body
	}
	rows := slices.DeleteFunc(slices.Clone(w.passiveBodies), func(r PassiveBody) bool { return r.ID == e.ID })
	w.passiveBodies = append(rows, b)
	w.sortPassiveBodies()
}

// heldWeaponOnly is an original dead Unit whose one worn item is its held
// weapon.
func (w *World) heldWeaponOnly(id EntityID, worn [EquipSlots]ItemInstance) bool {
	if worn[slotWeapon].Empty() {
		return false
	}
	for slot, item := range worn {
		if slot != slotWeapon && !item.Empty() {
			return false
		}
	}
	for _, row := range w.originalDead {
		if row.ID == id {
			return row.Source.Class == 1
		}
	}
	return false
}

func (w *World) sortPassiveBodies() {
	slices.SortFunc(w.passiveBodies, func(a, b PassiveBody) int { return cmp.Compare(a.ID, b.ID) })
}

// passiveBodyOwner is a terminal row whose body the original keeps: a retained
// original dead actor at its terminal stage, or a departed native actor that
// does not leave its body at teardown.
func (w *World) passiveBodyOwner(id EntityID) bool {
	for _, row := range w.currentTerminalActors {
		if row.ID == id {
			return !row.LeavesNoBody()
		}
	}
	for _, row := range w.originalDead {
		if row.ID == id && row.terminal.Stage != 0 {
			return true
		}
	}
	return false
}

// RestorePassiveBodies adopts the contents a LOAD read from dead records. A
// row replaces the row of the same ID; a row without items removes it. Item
// identities are dropped. It changes nothing on refusal.
func (w *World) RestorePassiveBodies(rows []PassiveBody) error {
	if w == nil {
		return fmt.Errorf("sim: passive body restore has no world")
	}
	next := slices.Clone(w.passiveBodies)
	for _, r := range rows {
		r = r.Clone()
		for slot := range r.Worn {
			if !r.Worn[slot].Empty() {
				r.Worn[slot] = w.passiveItem(r.Worn[slot])
			}
		}
		r.Carried = w.passiveStacks(r.Carried)
		if r.Body != nil {
			body := passiveEntity(*r.Body)
			r.Body = &body
		}
		next = slices.DeleteFunc(next, func(held PassiveBody) bool { return held.ID == r.ID })
		if !r.empty() {
			next = append(next, r)
		}
	}
	slices.SortFunc(next, func(a, b PassiveBody) int { return cmp.Compare(a.ID, b.ID) })
	held := w.passiveBodies
	w.passiveBodies = next
	if err := w.passiveBodiesFault(); err != nil {
		w.passiveBodies = held
		return err
	}
	return nil
}

func (w *World) passiveBodiesFault() error {
	if len(w.passiveBodies) > len(w.originalDead)+len(w.currentTerminalActors) {
		return fmt.Errorf("sim: more passive bodies than terminal rows")
	}
	for n, b := range w.passiveBodies {
		if n > 0 && b.ID <= w.passiveBodies[n-1].ID || indexOfEntity(w.entities, b.ID) >= 0 || !w.passiveBodyOwner(b.ID) || b.empty() ||
			len(b.Carried) > maxPassiveCarried {
			return fmt.Errorf("sim: invalid passive body %d", b.ID)
		}
		if b.Body != nil {
			terminal := false
			for _, row := range w.currentTerminalActors {
				terminal = terminal || row.ID == b.ID && row.MapUnitID == 0
			}
			if !terminal || b.Body.ID != b.ID || b.Body.MapUnitID != 0 || b.Body.SourceBinding.Class != 0 {
				return fmt.Errorf("sim: passive body %d has no departed native owner", b.ID)
			}
		}
		for _, item := range b.Worn {
			if item.Empty() {
				continue
			}
			if !passiveItemForm(item) {
				return fmt.Errorf("sim: invalid passive worn item on %d", b.ID)
			}
		}
		for _, st := range b.Carried {
			if st.Count == 0 || st.Count > maxPassiveCount || !passiveItemForm(st.Instance()) {
				return fmt.Errorf("sim: invalid passive carried item on %d", b.ID)
			}
		}
	}
	return nil
}
