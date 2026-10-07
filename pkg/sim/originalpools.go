package sim

import "fmt"

// OriginalActorPools changes only the four already-persisted pool fields.
// Entity identity has been resolved by the importer, not by a first-match scan.
type OriginalActorPools struct {
	ID                       EntityID
	HP, MaxHP, Mana, MaxMana int32
}

// ImportOriginalActorPools installs a complete validated batch after fresh
// construction and rearming, before publication or a gameplay tick. It does not
// replay effects, heal, reset regeneration, or perform death transitions.
func (w *World) ImportOriginalActorPools(pools []OriginalActorPools, current ...EntityID) error {
	if w == nil {
		return fmt.Errorf("original actor pools: no world")
	}
	currentIDs := make(map[EntityID]bool, len(current))
	for _, id := range current {
		if currentIDs[id] {
			return fmt.Errorf("original actor pools: repeated current entity %d", id)
		}
		currentIDs[id] = true
	}
	indices := make([]int, len(pools))
	seen := make(map[EntityID]bool, len(pools))
	for i, pool := range pools {
		if seen[pool.ID] {
			return fmt.Errorf("original actor pools: repeated entity %d", pool.ID)
		}
		seen[pool.ID] = true
		index := indexOfEntity(w.entities, pool.ID)
		if index < 0 {
			return fmt.Errorf("original actor pools: missing entity %d", pool.ID)
		}
		e := w.entities[index]
		if !e.Alive() && !originalDyingEntity(e) {
			return fmt.Errorf("original actor pools: entity %d is not living or dying", pool.ID)
		}
		// Profiles precede these final pool writes. Validate against the saved
		// maximum, never a fresh template maximum that this batch replaces.
		if e.CurrentProfileBasis == ProfileOriginalCurrent && int16(e.ManaRegenPeriod) == 0 && pool.MaxMana != 0 {
			return fmt.Errorf("original actor pools: entity %d has nonzero mana maximum with zero regeneration divisor", pool.ID)
		}
		if currentIDs[pool.ID] {
			// Explicit current nodes receive wire words here. Anchored width
			// residues restore signed/wide values in the continuation transaction.
			if pool.HP < -32768 || pool.HP > 32767 || pool.MaxHP < 0 || pool.MaxHP > 65535 ||
				pool.Mana < 0 || pool.Mana > 65535 || pool.MaxMana < 0 || pool.MaxMana > 65535 {
				return fmt.Errorf("original actor pools: entity %d exceeds current wire widths", pool.ID)
			}
		} else {
			validHP := pool.HP > 0
			if originalDyingEntity(e) {
				validHP = pool.HP <= 0 && pool.MaxHP > 0
			}
			if !validHP || pool.MaxHP < pool.HP || pool.Mana < 0 || pool.MaxMana < pool.Mana {
				return fmt.Errorf("original actor pools: entity %d has unsupported pools HP %d/%d mana %d/%d",
					pool.ID, pool.HP, pool.MaxHP, pool.Mana, pool.MaxMana)
			}
		}
		indices[i] = index
	}
	for _, id := range current {
		if !seen[id] {
			return fmt.Errorf("original actor pools: current entity %d is absent from batch", id)
		}
	}
	for i, pool := range pools {
		e := &w.entities[indices[i]]
		hp, maxHP := pool.HP, pool.MaxHP
		if currentIDs[pool.ID] {
			// A current actor admitted with a full health pair keeps it while
			// the saved words are that pair's low words.
			if int16(e.HP) == int16(pool.HP) {
				hp = e.HP
			}
			if uint16(e.MaxHP) == uint16(pool.MaxHP) {
				maxHP = e.MaxHP
			}
		}
		e.HP, e.MaxHP, e.Mana, e.MaxMana = hp, maxHP, pool.Mana, pool.MaxMana
	}
	return nil
}
