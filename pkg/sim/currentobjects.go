package sim

import (
	"encoding/binary"
	"fmt"
)

// ReplaceCurrentObjects rebinds an imported graph by explicit identities.
// Item values have already been read from their ordinary SAV nodes.
func (w *World) ReplaceCurrentObjects(registry *SavedObjects, identities map[SavedObjectID]SavedObjectID, values map[SavedObjectID]ItemStack, ground ...Sack) error {
	n := *w
	unbound := make(map[[2]int32]Sack, len(ground))
	for _, sack := range ground {
		key := [2]int32{sack.X, sack.Y}
		if sack.ObjectID != 0 {
			return fmt.Errorf("sim: ordinary ground values name a bound Sack")
		}
		if _, exists := unbound[key]; exists {
			return fmt.Errorf("sim: ordinary ground values repeat a Sack cell")
		}
		checked, err := normaliseSacks(w.bounds, []Sack{sack})
		if err != nil {
			return err
		}
		unbound[key] = checked[0]
	}
	n.savedObjects = registry.Clone()
	n.carried = make([][]ItemStack, len(w.carried))
	n.equipment = make([][EquipSlots]ItemInstance, len(w.equipment))
	rebind := func(v ItemInstance) (ItemInstance, error) {
		if v.ObjectID == 0 {
			return v.Clone(), nil
		}
		if value, ok := values[v.ObjectID]; ok {
			v = value.Instance()
		} else {
			v.ObjectID = identities[v.ObjectID]
		}
		return v.Clone(), v.ValidateWeight()
	}
	for i := range w.carried {
		n.carried[i] = cloneStacks(w.carried[i])
		for j, old := range n.carried[i] {
			v, err := rebind(old.Instance())
			if err != nil {
				return err
			}
			count := old.Count
			if value, present := values[old.ObjectID]; present {
				count = value.Count
			}
			n.carried[i][j] = StackItem(v, count)
		}
		for j, old := range w.equipment[i] {
			v, err := rebind(old)
			if err != nil {
				return err
			}
			n.equipment[i][j] = v
		}
	}
	n.sacks = make([]Sack, len(w.sacks))
	for i, old := range w.sacks {
		key := [2]int32{old.X, old.Y}
		if value, present := unbound[key]; present {
			if old.ObjectID != 0 {
				if id, explicit := identities[old.ObjectID]; !explicit || id != 0 {
					return fmt.Errorf("sim: ordinary ground values replace a bound Sack")
				}
			}
			n.sacks[i] = value
			delete(unbound, key)
			continue
		}
		n.sacks[i] = makeSack(old.X, old.Y, old.Gold, old.ItemInstances)
		n.sacks[i].ObjectID = identities[old.ObjectID]
		for j, v := range old.ItemInstances {
			var err error
			n.sacks[i].ItemInstances[j], err = rebind(v)
			if err != nil {
				return err
			}
		}
		if old.ObjectID != 0 {
			if container := w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: old.ObjectID}); container != nil && len(container.Items) != 0 {
				var units []ItemInstance
				at := 0
				for _, id := range container.Items {
					count := uint32(1)
					if id != 0 {
						count = w.savedObjects.item(id).Value.Count
					}
					if uint64(at)+uint64(count) > uint64(len(old.ItemInstances)) {
						return fmt.Errorf("sim: current Sack has incomplete ordinary units")
					}
					value := StackItem(n.sacks[i].ItemInstances[at], count)
					if current, present := values[id]; present {
						value = current
					}
					if uint64(len(units))+uint64(value.Count) > 1<<20 {
						return fmt.Errorf("sim: current Sack exceeds native unit capacity")
					}
					for range value.Count {
						units = append(units, value.Instance())
					}
					at += int(count)
				}
				if at != len(old.ItemInstances) {
					return fmt.Errorf("sim: current Sack has unmatched ordinary units")
				}
				n.sacks[i] = makeSack(old.X, old.Y, old.Gold, units)
				n.sacks[i].ObjectID = identities[old.ObjectID]
			}
		}
	}
	if len(unbound) != 0 {
		return fmt.Errorf("sim: ordinary ground values name an absent Sack")
	}
	if err := n.validateSavedObjects(); err != nil {
		return fmt.Errorf("sim: current object bindings: %w", err)
	}
	// Order+30 names the same Spell identity as the registry. An anchored
	// absent native key must replace its ordinary allocation in references too;
	// a changed ordinary pointer or Spell key has no such identity mapping.
	if w.savedObjects != nil && n.savedObjects != nil && n.savedGroups != nil {
		keys := map[uint32]uint32{}
		for _, old := range w.savedObjects.Spells {
			if current := n.savedObjects.spell(identities[old.ID]); old.This != 0 && current != nil && current.This != old.This {
				keys[old.This] = current.This
			}
		}
		if len(keys) != 0 {
			n.savedGroups = cloneSavedGroups(n.savedGroups)
			for i := range n.savedGroups.Orders {
				raw := &n.savedGroups.Orders[i].Raw
				if next, present := keys[binary.LittleEndian.Uint32(raw[0x30:])]; present {
					binary.LittleEndian.PutUint32(raw[0x30:], next)
				}
			}
		}
	}
	*w = n
	return nil
}
