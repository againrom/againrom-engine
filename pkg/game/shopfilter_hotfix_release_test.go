package game

import (
	"testing"

	"againrom/pkg/data"
)

func TestReleaseGeneratedShopFiltersStaffsAndMagicBeardByInstalledIdentity(t *testing.T) {
	f := releaseFront(t)
	name, named := f.Table.Names.NameFor(magicBeardItemCode)
	if !named || name == "" {
		t.Fatalf("installed item-name table has no row for special code %#04x", uint16(magicBeardItemCode))
	}

	armour := shopArmourPool(f.Table, 1_000_000)
	weapons := shopWeaponPool(f.Table, 1_000_000)
	all := append(append([]data.ShopCandidate{}, armour...), weapons...)
	foundBeard, foundStaff := false, false
	for _, candidate := range all {
		foundBeard = foundBeard || candidate.Code == magicBeardItemCode
		foundStaff = foundStaff || candidate.ForcedCast
	}
	if !foundBeard || !foundStaff {
		t.Fatalf("installed raw pools: Magic Beard=%v staff=%v; want both populations before filtering", foundBeard, foundStaff)
	}
	plainName, plainNamed := f.Table.Names.NameFor(beardItemCode)
	if !plainNamed || plainName == "" {
		t.Fatalf("installed item-name table has no row for beard code %#04x", uint16(beardItemCode))
	}
	foundPlain := false
	for _, candidate := range all {
		foundPlain = foundPlain || candidate.Code == beardItemCode
	}
	if !foundPlain {
		t.Fatal("installed raw pools hold no plain Beard before filtering")
	}
	for _, candidate := range shopStockPool(armour, false) {
		if candidate.Code == beardItemCode || candidate.Code == magicBeardItemCode {
			t.Fatalf("armour stock retained %#04x", uint16(candidate.Code))
		}
	}
	for _, candidate := range shopStockPool(weapons, false) {
		if candidate.ForcedCast || candidate.Code == magicBeardItemCode {
			t.Fatalf("ordinary weapon stock retained forbidden candidate %+v", candidate)
		}
	}
	magic := shopStockPool(all, true)
	keptStaff := false
	for _, candidate := range magic {
		if candidate.Code == magicBeardItemCode || candidate.Code == beardItemCode {
			t.Fatalf("magic stock retained %q (%#04x)", name, uint16(candidate.Code))
		}
		keptStaff = keptStaff || candidate.ForcedCast
	}
	if !keptStaff {
		t.Fatal("magic stock lost every staff; enchanted staffs must remain possible")
	}
}
