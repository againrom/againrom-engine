package game

import (
	"testing"

	"againrom/pkg/data"
)

func TestGeneratedShopPoolsExcludeOrdinaryStaffsAndMagicBeard(t *testing.T) {
	plain := data.ShopCandidate{Code: 0x0101}
	staff := data.ShopCandidate{Code: 0x0202, ForcedCast: true}
	beard := data.ShopCandidate{Code: magicBeardItemCode}
	plainBeard := data.ShopCandidate{Code: beardItemCode}
	pool := []data.ShopCandidate{plain, staff, beard, plainBeard}

	ordinary := shopStockPool(pool, false)
	if len(ordinary) != 1 || ordinary[0] != plain {
		t.Fatalf("ordinary pool = %+v, want only the non-staff ordinary item", ordinary)
	}
	magic := shopStockPool(pool, true)
	if len(magic) != 2 || magic[0] != plain || magic[1] != staff {
		t.Fatalf("magic pool = %+v, want ordinary item and staff but no Magic Beard", magic)
	}
}
