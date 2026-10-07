package sim

import (
	"encoding/binary"
	"sort"
)

// Only native code-constructed items without an explicit saved weight may
// acquire fresh constructor operands. A saved BaseItem with an equipment-like
// appearance remains that BaseItem, and a retained Weapon is never refilled.
func (w *World) sourceConstructItem(item ItemInstance) ItemInstance {
	if item.WeightPresent || item.SourceEquipment.Class != 0 {
		return item
	}
	i := sort.Search(len(w.itemWeights), func(i int) bool { return w.itemWeights[i].Code >= item.Code })
	if i == len(w.itemWeights) || w.itemWeights[i].Code != item.Code || w.itemWeights[i].Constructor.Class == 0 {
		return item
	}
	item.SourceEquipment = w.itemWeights[i].Constructor
	item.WeightPresent, item.Weight = true, int16(w.itemWeights[i].Weight)
	item.Kind = 1
	if item.SourceEquipment.Class == SourceWeapon {
		item.Kind = 2
	}
	return item
}

// EquipSourceCarried runs the ordinary source command without advancing a
// clock. Town uses the same ordered producer as mission KindEquip. A refusal
// commits neither an item prefix nor any intermediate derive.
func (w *World) EquipSourceCarried(id EntityID, index int, operations ...SourceEquipmentOperation) bool {
	trace, operation, ok := w.sourceEquipmentOperation(id, nil, operations)
	if !ok {
		return false
	}
	i := indexOfEntity(w.entities, id)
	if i < 0 || w.entities[i].ActorLoad.Source.Class == 0 {
		return false
	}
	n := w.sourceMutationCopy(i)
	if !n.sourceEquipCommandTraced(i, index, trace) || !n.savedMutationValid() {
		return false
	}
	*w = n
	operation.publish(trace, 0)
	return true
}

// EquipSourceItem attaches an item owned outside this actor's container.
// No invented take/insert pair changes the source accumulator before attach.
func (w *World) EquipSourceItem(id EntityID, item ItemInstance, operations ...SourceEquipmentOperation) bool {
	trace, operation, ok := w.sourceEquipmentOperation(id, &item, operations)
	if !ok || len(operation.PackAliases) != 0 {
		return false
	}
	i := indexOfEntity(w.entities, id)
	if i < 0 || w.entities[i].ActorLoad.Source.Class == 0 || !w.sourceMutationReady(i) {
		return false
	}
	n := w.sourceMutationCopy(i)
	if item.ObjectID != 0 {
		if n.savedObjects == nil {
			return false
		}
		row := n.savedObjects.item(item.ObjectID)
		var handles []uint64
		if operation.SessionHandle != 0 {
			handles = []uint64{operation.SessionHandle}
		}
		handle := n.savedObjects.sessionHandle(item.ObjectID, handles...)
		if row == nil || handle == 0 {
			return false
		}
		if _, err := n.savedObjects.ImportExternal(handle, n.savedObjects.stackForItem(item)); err != nil {
			return false
		}
	}
	ticket := trace.take(SourceEquipmentExternal, 0, false)
	old, oldTicket, ok := n.sourceAttachEquipmentTraced(i, item.Clone(), trace, ticket)
	if !ok || !old.Empty() && !n.sourcePutCarriedTraced(i, old, trace, oldTicket) || !n.sourceEquipmentLoad(i, 0) || !n.savedMutationValid() {
		return false
	}
	*w = n
	operation.publish(trace, 0)
	return true
}

// UnequipSource returns the removed instance after its removal effects and
// owned-Spell teardown. toPack=false leaves that value to the caller (tray or
// ground).
func (w *World) UnequipSource(id EntityID, slot int, toPack bool, operations ...SourceEquipmentOperation) (ItemInstance, bool) {
	trace, operation, ok := w.sourceEquipmentOperation(id, nil, operations)
	if !ok || len(operation.PackAliases) != 0 {
		return ItemInstance{}, false
	}
	i := indexOfEntity(w.entities, id)
	if i < 0 || w.entities[i].ActorLoad.Source.Class == 0 {
		return ItemInstance{}, false
	}
	n := w.sourceMutationCopy(i)
	item, ok := n.sourceUnequipCommandTraced(i, slot, toPack, trace)
	if !ok {
		return ItemInstance{}, false
	}
	var handle uint64
	if !toPack && item.ObjectID != 0 {
		var err error
		handle, err = n.savedObjects.ReserveExternal(item.ObjectID)
		if err != nil {
			return ItemInstance{}, false
		}
	}
	if !n.savedMutationValid() {
		return ItemInstance{}, false
	}
	*w = n
	operation.publish(trace, handle)
	return item, true
}

func sourceWordAdd(b []byte, off int, amount int32) {
	binary.LittleEndian.PutUint16(b[off:], binary.LittleEndian.Uint16(b[off:])+uint16(amount))
}

func sourceDefenceAdd(dst []byte, item [22]byte, sign int32) {
	for off := 0; off < 16; off += 2 {
		sourceWordAdd(dst, off, sign*int32(binary.LittleEndian.Uint16(item[off:])))
	}
	for off := 16; off < 22; off++ {
		dst[off] += byte(sign * int32(item[off]))
	}
}

// L(w) in SAV-EQUIPORDER-552: the item's signed own-weight update, at its
// exact event boundary. Container helpers maintain their accumulator before
// this call; no final-loadout sum substitutes for an intermediate event.
func (w *World) sourceEquipmentLoad(i int, delta int32) bool {
	defer w.syncSavedPack(i)
	e := &w.entities[i]
	if e.Capacity == 0 {
		return false
	}
	old, capacity := e.Load, e.Capacity
	e.ActorLoad.OwnWeight += int16(delta)
	e.Load = e.ActorLoad.CurrentLoad()
	e.HumanMovement.Load = e.Load
	e.ActorLoad.Source = e.SourceNow()
	if old/capacity != e.Load/capacity {
		return w.deriveSource(i)
	}
	return true
}

func (w *World) sourcePutCarried(i int, item ItemInstance) bool {
	return w.sourcePutCarriedTraced(i, item, nil, 0)
}

func (w *World) sourcePutCarriedTraced(i int, item ItemInstance, trace *sourceEquipmentTrace, ticket uint32) bool {
	if trace != nil && trace.topology != nil {
		if !w.sourceTopologyPutCarried(i, item, trace, ticket) {
			return false
		}
	} else if trace != nil {
		index, ok := w.addCarriedUnmerged(i, StackItem(item, 1))
		if !ok {
			return false
		}
		trace.put(ticket, SourceEquipmentPack, index)
	} else if !w.addCarried(i, StackItem(item, 1)) {
		return false
	}
	w.entities[i].ActorLoad.Accumulator += int32(item.Weight) * int32(trace.quantity(ticket))
	w.syncSavedPack(i)
	return true
}

func sourceEquipmentSlot(item ItemInstance, actor SourceActor) (int, bool) {
	s := item.SourceEquipment
	if item.Empty() || !item.WeightPresent || s.EffectsUnsupported || s.Validate() != nil {
		return 0, false
	}
	switch s.Class {
	case SourceArmor:
		// Array indices1/2 have no ordinary command representation.
		return int(s.OwnKind), actor.Class == 2 && s.OwnKind >= 3 && s.OwnKind <= EquipSlots
	case SourceShield:
		return 2, s.DefinitionRow != 0
	case SourceWeapon:
		d := s.Definition
		return 1, actor.EquipmentRuntimePresent && d.Present && (d.AttackType >= 0 && d.AttackType <= 5 || d.AttackType == 11 || d.AttackType == 12)
	}
	return 0, false
}

func sourceAllowsShield(item ItemInstance) bool {
	d := item.SourceEquipment.Definition
	return item.SourceEquipment.Class == SourceWeapon && d.Present && d.AttackType < 10 && d.Hands != 2
}

func (w *World) sourcePrepareWeapon(item *ItemInstance) bool {
	return w.sourcePrepareWeaponTraced(item, nil, 0)
}

func (w *World) sourcePrepareWeaponTraced(item *ItemInstance, trace *sourceEquipmentTrace, ticket uint32) bool {
	for _, effect := range item.Effects {
		if effect.Kind != 41 {
			continue
		}
		rule, ok := w.findSpell(uint32(uint8(effect.Operand)))
		if !ok {
			return false
		}
		if !w.replaceObjectSpell(item, sourceSpellFromRule(rule)) {
			return false
		}
		return w.sourceTraceSpell(trace, ticket, item.SourceEquipment.Spell)
	}
	return true // no matching Effect does not clear an existing owned Spell
}

func (w *World) sourceWeaponFields(i int, item ItemInstance, sign int32) bool {
	s, q := w.entities[i].SourceNow(), item.SourceEquipment
	d := q.Definition
	if d.AttackType < 10 {
		s.Modifier[32] += byte(sign * int32(q.Attack[14]))
		s.Modifier[33] += byte(sign * int32(q.Attack[15]))
		sourceWordAdd(s.Modifier[:], 42, sign*int32(binary.LittleEndian.Uint16(q.Defence[:])))
		if sign > 0 {
			copy(s.Modifier[37:40], q.Attack[19:22])
		} else {
			clear(s.Modifier[37:40])
		}
	} else {
		s.Modifier[37] += byte(sign * int32(q.Attack[14]))
		s.Modifier[38] += byte(sign * int32(q.Attack[15]))
		s.Modifier[39] = 0
		if sign > 0 {
			s.Modifier[39] = uint8(d.AttackType - 10)
		}
	}
	if sign > 0 && d.AttackType >= 10 {
		copy(s.Modifier[18:20], s.Attack[2:4]) // General assignment, not addition
	} else {
		sourceWordAdd(s.Modifier[:], 18, sign*int32(binary.LittleEndian.Uint16(q.Attack[:])))
	}
	s.Attack[16] = 0
	if sign > 0 && d.AttackType < 10 {
		s.Attack[16] = uint8(d.AttackType)
	}
	w.publishSourceDerived(i, s)
	return w.deriveSource(i)
}

func (w *World) sourceEquipmentDefence(i int, item ItemInstance, sign int32) {
	s := w.entities[i].SourceNow()
	sourceDefenceAdd(s.Modifier[42:], item.SourceEquipment.Defence, sign)
	sourceDefenceAdd(s.Defence[:], item.SourceEquipment.Defence, sign)
	w.publishSourceDerived(i, s)
}

// sourceRemoveEquipment is the item virtual removal, not the outer command.
// SAV-EQUIPORDER-552/OBS-555: slot clear, L(-w), direct fields and Effects
// have different placement for all three classes. No implicit final derive.
func (w *World) sourceRemoveEquipment(i, slot int) (ItemInstance, bool) {
	return w.sourceRemoveEquipmentTraced(i, slot, nil, 0)
}

func (w *World) sourceRemoveEquipmentTraced(i, slot int, trace *sourceEquipmentTrace, ticket uint32) (ItemInstance, bool) {
	item := w.equipment[i][slot-1].Clone()
	actual, valid := sourceEquipmentSlot(item, w.entities[i].SourceNow())
	if !valid || actual != slot {
		return ItemInstance{}, false
	}
	if !w.takeWornObject(i, slot, item) {
		return ItemInstance{}, false
	}
	switch item.SourceEquipment.Class {
	case SourceArmor, SourceShield:
		if !w.sourceEquipmentLoad(i, -int32(item.Weight)) {
			return ItemInstance{}, false
		}
		w.sourceEquipmentDefence(i, item, -1)
		if item.SourceEquipment.Class == SourceArmor {
			w.equipment[i][slot-1] = ItemInstance{}
		}
		if !w.sourceEquipmentEffects(i, item, -1) {
			return ItemInstance{}, false
		}
	case SourceWeapon:
		if !w.sourceEquipmentEffects(i, item, -1) || !w.sourceWeaponFields(i, item, -1) {
			return ItemInstance{}, false
		}
		e := &w.entities[i]
		e.Reach -= item.SourceEquipment.OwnKind - 1
		e.AttackCharge, e.AttackRelax = 8, 4
		e.ActorLoad.Source = e.SourceNow()
		if !w.sourceEquipmentLoad(i, -int32(item.Weight)) {
			return ItemInstance{}, false
		}
		if !w.replaceObjectSpell(&item, SourceItemSpell{}) {
			return ItemInstance{}, false
		}
		if !w.sourceTraceSpell(trace, ticket, item.SourceEquipment.Spell) {
			return ItemInstance{}, false
		}
	}
	w.equipment[i][slot-1] = ItemInstance{}
	trace.detachWorn(slot - 1)
	if slot == 1 {
		syncWeaponItem(&w.entities[i], ItemInstance{})
	}
	return item, true
}

func (w *World) sourceAttachEquipment(i int, item ItemInstance) (ItemInstance, bool) {
	old, _, ok := w.sourceAttachEquipmentTraced(i, item, nil, 0)
	return old, ok
}

func (w *World) sourceAttachEquipmentTraced(i int, item ItemInstance, trace *sourceEquipmentTrace, ticket uint32) (ItemInstance, uint32, bool) {
	item = w.sourceConstructItem(item)
	slot, valid := sourceEquipmentSlot(item, w.entities[i].SourceNow())
	if !valid {
		return ItemInstance{}, 0, false
	}
	if slot == 1 && !w.sourcePrepareWeaponTraced(&item, trace, ticket) {
		return ItemInstance{}, 0, false
	}
	var displaced ItemInstance
	var displacedTicket uint32
	if !w.equipment[i][slot-1].Empty() {
		var ok bool
		displacedTicket = trace.take(SourceEquipmentWorn, slot-1, false)
		displaced, ok = w.sourceRemoveEquipmentTraced(i, slot, trace, displacedTicket)
		if !ok {
			return ItemInstance{}, 0, false
		}
	}
	// A shield stands alone. A weapon that fills both hands and a shield cannot
	// be worn together: whichever arrives removes the other into the pack, before
	// the displaced same-slot item follows it there.
	opposite := 0
	switch {
	case slot == 1 && !sourceAllowsShield(item):
		opposite = 2
	case slot == 2 && !w.equipment[i][0].Empty() && !sourceAllowsShield(w.equipment[i][0]):
		opposite = 1
	}
	if opposite != 0 && !w.equipment[i][opposite-1].Empty() {
		oppositeTicket := trace.take(SourceEquipmentWorn, opposite-1, false)
		removed, ok := w.sourceRemoveEquipmentTraced(i, opposite, trace, oppositeTicket)
		if !ok {
			return ItemInstance{}, 0, false
		}
		if !w.sourcePutCarriedTraced(i, removed, trace, oppositeTicket) {
			return ItemInstance{}, 0, false
		}
	}
	if item.ObjectID != 0 {
		// Evicting another root of the incoming node can tear down its
		// owned Spell. The registry node owns the current incoming value.
		row := w.savedObjects.item(item.ObjectID)
		if row == nil {
			return ItemInstance{}, 0, false
		}
		item = row.Value.Instance()
	}
	trace.currentSpell(ticket, &item)
	if !w.putWornObject(i, slot, item) {
		return ItemInstance{}, 0, false
	}
	w.equipment[i][slot-1] = item
	trace.put(ticket, SourceEquipmentWorn, slot-1)
	if slot == 1 {
		syncWeaponItem(&w.entities[i], item)
		if !w.sourceWeaponFields(i, item, 1) {
			return ItemInstance{}, 0, false
		}
		e := &w.entities[i]
		d := item.SourceEquipment.Definition
		if e.ActorLoad.Source.Class == 2 {
			if d.Charge != -1 {
				e.AttackCharge = int32(uint8(d.Charge))
			}
			if d.Relax != -1 {
				e.AttackRelax = int32(uint8(d.Relax))
			}
		}
		e.Reach += item.SourceEquipment.OwnKind - 1
		e.ActorLoad.Source = e.SourceNow()
	} else {
		w.sourceEquipmentDefence(i, item, 1)
	}
	if !w.sourceEquipmentLoad(i, int32(item.Weight)) || !w.sourceEquipmentEffects(i, item, 1) {
		return ItemInstance{}, 0, false
	}
	return displaced, displacedTicket, true
}

func (w *World) sourceEquipCommand(i, index int) bool {
	return w.sourceEquipCommandTraced(i, index, nil)
}

func (w *World) sourceEquipCommandTraced(i, index int, trace *sourceEquipmentTrace) bool {
	if !w.sourceMutationReady(i) || !w.hasActorContainer(i) || index < 0 || index >= len(w.carried[i]) || !trace.validAliases(w.carried[i], index) {
		return false
	}
	item := w.sourceConstructItem(w.carried[i][index].Instance())
	if _, ok := sourceEquipmentSlot(item, w.entities[i].SourceNow()); !ok {
		return false
	}
	w.entities[i].ActorLoad.Accumulator -= int32(item.Weight)
	w.entities[i].ActorLoad.InsertIndex = uint32(index)
	split := w.carried[i][index].Count > 1
	if split && item.ObjectID != 0 {
		// A node quantity change also removes one unit from every other
		// occurrence. The selected occurrence already ran its source callback.
		for j, held := range w.carried[i] {
			if j != index && held.ObjectID == item.ObjectID {
				w.entities[i].ActorLoad.Accumulator -= int32(item.Weight)
			}
		}
	} else if split && trace != nil {
		for _, alias := range trace.aliases {
			w.carried[i][alias].Count--
			w.entities[i].ActorLoad.Accumulator -= int32(item.Weight)
		}
	}
	taken, ok := w.takeCarriedObject(i, index, false)
	if !ok {
		return false
	}
	ticket := trace.take(SourceEquipmentPack, index, split)
	if split && taken.ObjectID == 0 {
		taken = StackItem(CloneSplitItemValue(taken.Instance(), w.spells), 1)
	}
	item = w.sourceConstructItem(taken.Instance())
	w.syncSavedPack(i)
	displaced, displacedTicket, ok := w.sourceAttachEquipmentTraced(i, item, trace, ticket)
	if !ok || !displaced.Empty() && !w.sourcePutCarriedTraced(i, displaced, trace, displacedTicket) {
		return false
	}
	return w.sourceEquipmentLoad(i, 0)
}

func (w *World) sourceUnequipCommand(i, slot int, toPack bool) (ItemInstance, bool) {
	return w.sourceUnequipCommandTraced(i, slot, toPack, nil)
}

func (w *World) sourceUnequipCommandTraced(i, slot int, toPack bool, trace *sourceEquipmentTrace) (ItemInstance, bool) {
	if !w.sourceMutationReady(i) || slot < 1 || slot > EquipSlots || !w.hasActorContainer(i) {
		return ItemInstance{}, false
	}
	ticket := trace.take(SourceEquipmentWorn, slot-1, false)
	item, ok := w.sourceRemoveEquipmentTraced(i, slot, trace, ticket)
	if !ok {
		return ItemInstance{}, false
	}
	if toPack && !w.sourcePutCarriedTraced(i, item, trace, ticket) {
		return ItemInstance{}, false
	}
	if !toPack {
		trace.put(ticket, SourceEquipmentExternal, 0)
	}
	return item, w.sourceEquipmentLoad(i, 0)
}

// The death routine calls removal virtuals and inserts each return; it is not
// command22 and has no command's extra L(0). ITEM-DEATH-012/CORPSE-034 retain
// the weapon when its bound parameter15 is zero. Container destruction creates
// fresh bookkeeping, never a delta inferred from the old item population.
func (w *World) sourceTerminalLoot(i int, x, y int32, validCell bool) bool {
	observation := w.damageObservation
	w.damageObservation = nil
	defer func() { w.damageObservation = observation }()
	if !w.sourceMutationReady(i) || !w.hasActorContainer(i) {
		return false
	}
	e := &w.entities[i]
	if !validCell && !e.SuppressCorpseLoot {
		return true
	}
	for _, slot := range []int{2, 1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12} {
		item := w.equipment[i][slot-1]
		if item.Empty() || slot >= 3 && e.ActorLoad.Source.Class != 2 {
			continue
		}
		if slot == 1 {
			if !item.SourceEquipment.Definition.Present {
				return false
			}
			if item.innateWeapon() {
				continue
			}
		}
		removed, ok := w.sourceRemoveEquipment(i, slot)
		if !ok || !w.sourcePutCarried(i, removed) {
			return false
		}
	}
	if e.SuppressCorpseLoot {
		if w.savedObjects != nil && !w.disposeSavedPack(i) {
			return false
		}
		w.carried[i] = nil
		e.ActorLoad.Accumulator, e.ActorLoad.InsertIndex = 0, 10000
	}
	if validCell {
		gold := w.deathGold(i)
		if gold != 0 || len(w.carried[i]) != 0 {
			if w.savedObjects != nil {
				if !w.dropSavedPackAt(i, gold, x, y) {
					return false
				}
			} else {
				// All fallible producer work is done. The caller's candidate owns
				// this sack slice too, including an existing destination's items.
				w.sacks = w.Sacks()
				w.pourSack(x, y, gold, expandItems(w.carried[i]))
			}
		}
	}
	w.carried[i] = nil
	e.ActorLoad.ContainerPresent = true
	e.ActorLoad.Accumulator, e.ActorLoad.InsertIndex = 0, 10000
	e.ActorLoad.Source = e.SourceNow()
	w.syncSavedPack(i)
	if e.CurrentProfileBasis == ProfileNative {
		// The native sheet finishes equipment removal with a current derive,
		// including empty-container load and the body's death adjustment.
		e.Load = e.ActorLoad.CurrentLoad()
		e.ActorLoad.Source = e.SourceNow()
		if !w.deriveSource(i) {
			return false
		}
		if e.Decay != DecayNone {
			e.Defence >>= 1
		}
		e.ActorLoad.Source = e.SourceNow()
	}
	return true
}
