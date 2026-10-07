package mapload

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestNoItemEffectKindWritesDamageKindResistance(t *testing.T) {
	for _, kind := range effectKinds {
		for _, mode := range []uint8{0, 1, 2, 4, 8} {
			var equipped [sim.EquipSlots]sim.ItemInstance
			equipped[0].Effects = []sim.ItemEffect{{Kind: kind, Mode: mode, Operand: 37}}
			for _, fighter := range []bool{false, true} {
				var loadout data.Loadout
				ApplyItemEffects(&loadout, equipped, fighter)
				if loadout.Mod.Resistance != ([5]int32{}) {
					t.Fatalf("effect kind %d mode %d fighter=%v set Mod.Resistance = %v, want all zero",
						kind, mode, fighter, loadout.Mod.Resistance)
				}
			}
		}
	}
}
