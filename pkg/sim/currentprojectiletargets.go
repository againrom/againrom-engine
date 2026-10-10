package sim

import "fmt"

func (w *World) restoreCurrentProjectileTargets(incoming *World) error {
	if w.savedWorldEffects == nil {
		return nil
	}
	current := make(map[EntityID]int32, len(w.entities))
	for _, e := range w.entities {
		current[e.ID] = ProjectileTargetKey(e)
	}
	wires := make(map[EntityID]int32, len(incoming.entities))
	owners, ambiguous := map[int32]EntityID{}, map[int32]bool{}
	for _, e := range incoming.entities {
		key := ProjectileTargetKey(e)
		if key <= 0 || key > 65535 {
			continue
		}
		wires[e.ID] = key
		if owner, exists := owners[key]; exists && owner != e.ID {
			ambiguous[key] = true
		}
		owners[key] = e.ID
	}
	held := make(map[EntityID]int32, len(w.originalDead))
	for _, dead := range w.originalDead {
		held[dead.ID] = int32(dead.Source.State.RuntimeID)
	}
	indices := make(map[uint16]int, len(w.savedProjectiles.Items))
	for i, p := range w.savedProjectiles.Items {
		indices[p.ID] = i
	}
	updates := map[int]int32{}
	for _, d := range w.savedWorldEffects.Projectiles {
		if d.Retired || !d.HasTarget || d.TargetDetached || d.TargetStructure {
			continue
		}
		index, present := indices[d.ID]
		if !present {
			continue
		}
		p := w.savedProjectiles.Items[index]
		wire, matched := wires[d.Target]
		value, live := current[d.Target]
		if !matched || !live || p.ActionTarget != wire || value == 0 || value == wire {
			continue
		}
		if retained := held[d.Target]; retained != 0 && retained != value {
			continue
		}
		if ambiguous[wire] {
			return fmt.Errorf("sim: current projectile target has an aliased wire coordinate")
		}
		if previous, exists := updates[index]; exists && previous != value {
			return fmt.Errorf("sim: current projectile target has conflicting live drivers")
		}
		updates[index] = value
	}
	if len(updates) != 0 {
		w.savedProjectiles = cloneSavedProjectiles(w.savedProjectiles)
		for index, value := range updates {
			w.savedProjectiles.Items[index].ActionTarget = value
		}
	}
	return nil
}
