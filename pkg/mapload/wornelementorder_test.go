package mapload

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// TestWornElementalEffectsApplyInConstructionOrder pins ITEM-253, ITEM-254
// and ITEM-255 at construction: the weapon in cell 0, the shield in cell 1,
// then the armour; each elemental effect replaces base, spread and kind
// whole, a zero base included, so the last worn item's effect decides and
// replaces a ranged weapon's own element.
func TestWornElementalEffectsApplyInConstructionOrder(t *testing.T) {
	var equipped [sim.EquipSlots]sim.ItemInstance
	equipped[0] = sim.ItemInstance{Code: 0x0201, Effects: []sim.ItemEffect{{Kind: 44, Operand: 5 | 7<<8}}}
	equipped[1] = sim.ItemInstance{Code: 0x0301, Effects: []sim.ItemEffect{{Kind: 45, Operand: 9 | 2<<8}}}
	equipped[4] = sim.ItemInstance{Code: 0x0401, Effects: []sim.ItemEffect{{Kind: 47, Operand: 3 << 8}}}
	var loadout data.Loadout
	ApplyItemEffects(&loadout, equipped, true)
	want := data.SecondaryDamage{Base: 0, Spread: 3, Selector: 3}
	if loadout.Mod.SecondaryDamage != want || !loadout.Mod.HasSecondaryDamage {
		t.Fatalf("worn effects = %+v set=%v, want the armour's Earth %+v", loadout.Mod.SecondaryDamage, loadout.Mod.HasSecondaryDamage, want)
	}
	w := data.Weapon{AttackType: 12, DamageBase: 20, DamageSpread: 10, Range: 6}
	loadout.Weapon = &w
	h := data.Hero{Body: 30, Reaction: 30}
	if got := h.Recompute(data.Profile{}, loadout).SecondaryDamage; got != want {
		t.Fatalf("ranged weapon then armour = %+v, want the armour's %+v", got, want)
	}
}
