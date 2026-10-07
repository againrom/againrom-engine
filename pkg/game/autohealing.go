package game

import (
	"fmt"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func carriedAutoHealing(party []mapload.PartyMember) (uint32, bool, error) {
	var percent uint32
	present := false
	for _, member := range party {
		if member.Carry == nil || member.Carry.LiveLoad == nil {
			continue
		}
		source := member.Carry.LiveLoad.Inventory.Source
		if source.Class == 0 || !source.HasOwner {
			continue
		}
		if present && percent != source.ManaReservePercent {
			return 0, false, fmt.Errorf("carried party has inconsistent player autohealing percentages")
		}
		percent, present = source.ManaReservePercent, true
	}
	return percent, present, nil
}

func (mw *mapWorld) autoHealingSetting() int {
	mode := -1
	if percent, present := mw.world.AutoHealing(sim.SelfSlot); present {
		switch percent {
		case 100:
			mode = 0
		case 50:
			mode = 1
		case 0:
			mode = 2
		}
	}
	for _, c := range mw.pending {
		if c.Kind == sim.KindPlayerParameter && c.Player == sim.SelfSlot && sim.PlayerParameter(c.X) == sim.PlayerParameterAutoHealing {
			mode = int(c.Y)
		}
	}
	return mode
}

func (mw *mapWorld) checkAutoHealing(value int) error {
	players, present := mw.world.SavedGroupPlayers()
	if present {
		count := 0
		for _, p := range players {
			if p.Slot == sim.SelfSlot {
				count++
			}
		}
		if count != 1 {
			return fmt.Errorf("autohealing requires one exact player identity")
		}
	}
	probe := *mw.world
	if !probe.SetAutoHealing(sim.SelfSlot, int32(value)) {
		return fmt.Errorf("autohealing could not derive the party's mana floors")
	}
	return nil
}
