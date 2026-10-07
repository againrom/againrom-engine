package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"fmt"
)

func validateSnapshotItemWeights(s Snapshot) error {
	check := func(p mapload.PartyMember) error {
		sets := [][]sim.ItemInstance{p.CarriedItems, p.WornItems[:]}
		if p.Carry != nil {
			sets = append(sets, p.Carry.ItemInstances, p.Carry.EquippedItems[:])
		}
		if p.OriginalHuman != nil {
			sets = append(sets, p.OriginalHuman.Inventory, p.OriginalHuman.Equipment[:])
		}
		for _, items := range sets {
			for _, item := range items {
				if err := item.ValidateWeight(); err != nil {
					return fmt.Errorf("party %q: %w", p.ID, err)
				}
			}
		}
		return nil
	}
	for _, p := range s.Party {
		if err := check(p); err != nil {
			return err
		}
	}
	if s.OriginalCity != nil {
		for _, b := range s.OriginalCity.Bindings {
			if err := check(b.Baseline); err != nil {
				return err
			}
		}
	}
	return nil
}
