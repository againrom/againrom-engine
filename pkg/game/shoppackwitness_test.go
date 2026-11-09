package game

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

func TestCityShopPackWritesKeepSAVBytes(t *testing.T) {
	var record strings.Builder
	for _, action := range []string{"drag", "purchase", "staged"} {
		t.Run(action, func(t *testing.T) {
			f := shopSaveFixture(t, 2)
			screen := f.TownScreen().(*townScreen)
			screen.room, screen.shopMember = roomShop, 0
			if action == "drag" {
				first := cityMemberStacks(f.Carried[0], f.Table)[0]
				other := sim.ItemInstance{Code: 0xe0e, Kind: 1, Price: 11, WeightPresent: true}
				cityShopSetTestPack(t, f, 0, first, sim.StackItem(other, 3))
				cityShopGraph(t, f)
				roots := slices.Clone(cityShopRoots(t, f, 0).Pack)
				screen.shopMoveWithinPack(0, 1, true)
				if got := cityShopRoots(t, f, 0).Pack; !slices.Equal(got, []sim.SavedObjectID{roots[1], roots[0]}) {
					t.Fatalf("drag did not swap pack roots: %v -> %v", roots, got)
				}
			} else {
				gold := f.Town.Gold()
				shopSaveStage(t, screen, 0, 0, true)
				f.Shop.shelves[ShelfMagic] = []ShopItem{{Code: 0xe0e, Kind: 1, Price: 11, Count: 2, WeightPresent: true}}
				screen.shopShelf = ShelfMagic
				screen.shopFromShelf(0, true)
				if len(f.Shop.table) != 2 {
					t.Fatal("the owned and merchant stacks did not reach the table")
				}
				if action == "purchase" {
					screen.shopBuy()
					if f.Town.Gold() != gold-22 || len(f.Shop.table) != 1 {
						t.Fatal("purchase did not charge for the merchant stack once")
					}
				}
			}
			raw := cityProjectionSave(t, f)
			fmt.Fprintf(&record, "%s %d %x\n", action, len(raw), sha256.Sum256(raw))
			if dir := os.Getenv("AGAINROM_SHOP_PACK_WITNESS_DIR"); dir != "" {
				path := filepath.Join(dir, action+".sav")
				if os.Getenv("AGAINROM_WRITE_SHOP_PACK_WITNESS") != "" {
					if err := os.WriteFile(path, raw, 0o644); err != nil {
						t.Fatal(err)
					}
				} else {
					want, err := os.ReadFile(path)
					if err != nil || !bytes.Equal(raw, want) {
						t.Fatalf("SAV differs from base bytes %s: %v", path, err)
					}
				}
			}
		})
	}
	const golden = "testdata/shoppackwitness.golden"
	if os.Getenv("AGAINROM_WRITE_SHOP_PACK_WITNESS") != "" {
		if err := os.WriteFile(golden, []byte(record.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil || strings.ReplaceAll(string(want), "\r", "") != record.String() {
		t.Fatalf("shop SAV byte witness differs: %v\n%s", err, record.String())
	}
}
