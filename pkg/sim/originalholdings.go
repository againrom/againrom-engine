package sim

import "fmt"

// OriginalActorStock carries compact canonical stacks; no count is expanded
// before validation. The bound also covers later native flat compatibility
// projections and their deep-copied effects, across the entire batch.
const MaxOriginalHoldingValues = 1 << 20

type OriginalActorStock struct {
	ID        EntityID
	Carried   []ItemStack
	Equipped  [EquipSlots]ItemInstance
	LoadState *ActorLoadSnapshot
}

// ImportOriginalActorStock replaces a validated living-actor batch atomically.
// It restores holdings/cache/load only: no equip effects, teaching, pool
// deltas, or synthetic innate items. An independent innate/legacy pair cannot
// coexist with an item cast in the current one-source representation.
func (w *World) ImportOriginalActorStock(batch []OriginalActorStock) error {
	if w == nil {
		return fmt.Errorf("sim: original holdings require a world")
	}
	entities := append([]Entity(nil), w.entities...)
	seen := make(map[EntityID]bool)
	var values uint64
	for _, stock := range batch {
		i := indexOfEntity(entities, stock.ID)
		if i < 0 || seen[stock.ID] {
			return fmt.Errorf("sim: original holdings absent/repeated actor %d", stock.ID)
		}
		if w.actorHasSavedItems(i) {
			return fmt.Errorf("sim: original holdings cannot replace bound object ownership")
		}
		seen[stock.ID] = true
		e := &entities[i]
		if stock.LoadState != nil {
			if err := stock.LoadState.Validate(); err != nil {
				return err
			}
			if !stock.LoadState.Inventory.ContainerPresent && len(stock.Carried) != 0 {
				return fmt.Errorf("sim: absent original container carries items")
			}
		}
		if !e.Alive() && !originalDyingEntity(*e) {
			return fmt.Errorf("sim: original holdings actor %d is not living or dying", stock.ID)
		}
		for _, stack := range stock.Carried {
			if (stack.Code == 0 || stack.Count == 0) && !emptyOrderedStack(stack) {
				return fmt.Errorf("sim: original holdings actor %d has zero code/count", stock.ID)
			}
			if err := stack.Instance().ValidateWeight(); err != nil {
				return fmt.Errorf("sim: original holdings actor %d carried item: %w", stock.ID, err)
			}
			values += uint64(stack.Count) * (1 + uint64(len(stack.Effects)))
			if values > MaxOriginalHoldingValues {
				return fmt.Errorf("sim: original holdings exceed %d item/effect values", MaxOriginalHoldingValues)
			}
		}
		for _, item := range stock.Equipped {
			if item.Code == 0 && !item.Empty() {
				return fmt.Errorf("sim: original holdings actor %d has empty equipment residue", stock.ID)
			}
			if err := item.ValidateWeight(); err != nil {
				return fmt.Errorf("sim: original holdings actor %d equipped item: %w", stock.ID, err)
			}
			values += 1 + uint64(len(item.Effects))
			if values > MaxOriginalHoldingValues {
				return fmt.Errorf("sim: original holdings exceed %d item/effect values", MaxOriginalHoldingValues)
			}
		}
		weapon := stock.Equipped[slotWeapon]
		if _, _, cast := weapon.CastSpell(); cast && (e.WeaponSpellSource == WeaponSpellInnate || e.WeaponSpellSource == WeaponSpellLegacy) {
			return fmt.Errorf("sim: original holdings actor %d item spell conflicts with independent source", stock.ID)
		}
		syncWeaponItem(e, weapon)
		if err := validateWeaponSource(*e, weapon); err != nil {
			return fmt.Errorf("sim: original holdings actor %d: %w", stock.ID, err)
		}
	}
	for _, stock := range batch {
		i := indexOfEntity(entities, stock.ID)
		w.entities[i] = entities[i]
		w.carried[i] = cloneStacks(stock.Carried)
		if stock.LoadState != nil {
			stock.LoadState.apply(&w.entities[i])
		}
		w.equipment[i] = cloneEquipment(stock.Equipped)
		w.recomputeLoad(i)
	}
	return nil
}
