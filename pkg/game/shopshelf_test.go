package game

import (
	"testing"

	"againrom/pkg/data"
)

// The weapons and armour shelves drop a Weapon without sutableFor bit 0 and an
// Armor whose class field is 3..5; a Shield takes no test (SHOP-118).
func TestShopPlainShelfPoolAppliesSuitabilityAndClassTests(t *testing.T) {
	weapon := func(code data.ItemCode, fighter, mage bool) data.ShopCandidate {
		return data.ShopCandidate{Code: code, ItemKind: 2, EffectSlot: 1, Fighter: fighter,
			ForcedCast: !fighter && mage}
	}
	armour := func(code data.ItemCode, class int) data.ShopCandidate {
		return data.ShopCandidate{Code: code, ItemKind: 1, EffectSlot: class, Fighter: true}
	}
	kept := []data.ShopCandidate{
		weapon(0x0101, true, false),
		weapon(0x0102, true, true),
		armour(0x0201, 2), // Shield
		armour(0x0301, 0), // Armor row with no valid Slot
		armour(0x0302, 1),
		armour(0x0303, 2),
		armour(0x0304, 6),
		armour(0x0305, 12),
	}
	dropped := []data.ShopCandidate{
		weapon(0x0110, false, true), // staff: mage only
		weapon(0x0111, false, false),
		armour(0x0310, 3),
		armour(0x0311, 4), // Ring
		armour(0x0312, 5), // Amulet, the class of both beards
	}
	got := shopPlainShelfPool(append(append([]data.ShopCandidate{}, dropped...), kept...))
	if len(got) != len(kept) {
		t.Fatalf("plain shelf pool = %+v, want %+v", got, kept)
	}
	for i := range kept {
		if got[i] != kept[i] {
			t.Fatalf("plain shelf pool[%d] = %+v, want %+v", i, got[i], kept[i])
		}
	}
}
