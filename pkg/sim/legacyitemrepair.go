package sim

// LegacyLinkedItemPopulation identifies one canonical World item container.
type LegacyLinkedItemPopulation uint8

const (
	LegacyLinkedItemCarried LegacyLinkedItemPopulation = iota + 1
	LegacyLinkedItemEquipped
	LegacyLinkedItemSack
)

// LegacyLinkedItemRepair describes one scalar effect that an old linked-item
// producer stored with the magnitude's negated byte beside the magnitude.
// Position is the carried-stack index, equipment-slot index, or sack-item
// index according to Population. Count is greater than one only for a carried
// stack.
type LegacyLinkedItemRepair struct {
	Population  LegacyLinkedItemPopulation
	Entity      EntityID
	SackX       int32
	SackY       int32
	Position    int
	EffectIndex int
	Count       uint32
	Code        uint16
	Kind        uint8
	Before      uint32
	After       uint32
}

// legacyLinkedScalarOperand recognizes the exact old underflow encoding. Only
// permanent scalar effects use one signed magnitude in the whole operand.
// Spell payloads, timed modes, and range effects have independent meanings for
// the same bytes and are excluded.
func legacyLinkedScalarOperand(effect ItemEffect) (uint32, bool) {
	if effect.Mode != 0 || !((effect.Kind >= 1 && effect.Kind <= 40) || effect.Kind == 49) {
		return 0, false
	}
	if effect.Operand&0xffff0000 != 0 {
		return 0, false
	}
	magnitude := uint8(effect.Operand)
	if magnitude == 0 || uint8(effect.Operand>>8) != uint8(-int(magnitude)) {
		return 0, false
	}
	return uint32(magnitude), true
}

func inspectLegacyLinkedItem(out []LegacyLinkedItemRepair, item ItemInstance,
	population LegacyLinkedItemPopulation, entity EntityID, sackX, sackY int32,
	position int, count uint32) []LegacyLinkedItemRepair {
	for effectIndex, effect := range item.Effects {
		restored, ok := legacyLinkedScalarOperand(effect)
		if !ok {
			continue
		}
		out = append(out, LegacyLinkedItemRepair{
			Population:  population,
			Entity:      entity,
			SackX:       sackX,
			SackY:       sackY,
			Position:    position,
			EffectIndex: effectIndex,
			Count:       count,
			Code:        item.Code,
			Kind:        effect.Kind,
			Before:      effect.Operand,
			After:       restored,
		})
	}
	return out
}

// InspectLegacyLinkedItems returns every effect RepairLegacyLinkedItems would
// repair. It does not mutate the World or expose an Effects slice.
func InspectLegacyLinkedItems(w *World) []LegacyLinkedItemRepair {
	if w == nil {
		return nil
	}
	var out []LegacyLinkedItemRepair
	for entityIndex, stacks := range w.carried {
		for position, stack := range stacks {
			out = inspectLegacyLinkedItem(out, stack.Instance(), LegacyLinkedItemCarried,
				w.entities[entityIndex].ID, 0, 0, position, stack.Count)
		}
	}
	for entityIndex, equipment := range w.equipment {
		for slot, item := range equipment {
			out = inspectLegacyLinkedItem(out, item, LegacyLinkedItemEquipped,
				w.entities[entityIndex].ID, 0, 0, slot, 1)
		}
	}
	for _, sack := range w.sacks {
		for position, item := range sack.ItemInstances {
			out = inspectLegacyLinkedItem(out, item, LegacyLinkedItemSack, 0,
				sack.X, sack.Y, position, 1)
		}
	}
	return out
}

// RepairLegacyLinkedItemInstance repairs one complete item instance carrying
// the legacy scalar-underflow signature. reprice sees the fully repaired
// clone; nil preserves the stored price. The second call is inert.
func RepairLegacyLinkedItemInstance(item *ItemInstance, reprice func(ItemInstance) int32) bool {
	if item == nil {
		return false
	}
	repaired := false
	for i := range item.Effects {
		operand, ok := legacyLinkedScalarOperand(item.Effects[i])
		if !ok {
			continue
		}
		item.Effects[i].Operand = operand
		repaired = true
	}
	if repaired && reprice != nil {
		item.Price = reprice(item.Clone())
	}
	return repaired
}

// RepairLegacyLinkedItems repairs the complete canonical World item
// population. reprice receives each repaired item after all of its effects are
// repaired. A nil reprice preserves the stored price. The returned records are
// the read-only pre-repair census, and a second call returns no records.
func RepairLegacyLinkedItems(w *World, reprice func(ItemInstance) int32) []LegacyLinkedItemRepair {
	repairs := InspectLegacyLinkedItems(w)
	if len(repairs) == 0 {
		return nil
	}

	for entityIndex := range w.carried {
		changed := false
		for position := range w.carried[entityIndex] {
			stack := w.carried[entityIndex][position]
			item := stack.Instance()
			if !RepairLegacyLinkedItemInstance(&item, reprice) {
				continue
			}
			w.carried[entityIndex][position] = StackItem(item, stack.Count)
			changed = true
		}
		if changed {
			w.carried[entityIndex] = foldContainer(w.carried[entityIndex])
		}
	}
	for entityIndex := range w.equipment {
		for slot := range w.equipment[entityIndex] {
			item := w.equipment[entityIndex][slot].Clone()
			if RepairLegacyLinkedItemInstance(&item, reprice) {
				w.equipment[entityIndex][slot] = item
			}
		}
	}
	for sackIndex := range w.sacks {
		for position := range w.sacks[sackIndex].ItemInstances {
			item := w.sacks[sackIndex].ItemInstances[position].Clone()
			if RepairLegacyLinkedItemInstance(&item, reprice) {
				w.sacks[sackIndex].ItemInstances[position] = item
			}
		}
	}
	return repairs
}
