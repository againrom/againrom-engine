package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

func TestReleaseShopStockMultiplierDefaultMatchesMain(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := shopOrderFront(t)
	h := sha256.New()
	for _, ceiling := range shelfTestCeilings(t, f) {
		for _, seed := range []int64{1, 7, 42} {
			s := NewShop(ceiling)
			s.Generate(f.Table, seed)
			explicit := NewShop(ceiling)
			explicit.stockMultiplier = 1
			explicit.Generate(f.Table, seed)
			assertShopRolledTimes(t, s, explicit, 1)
			if err := json.NewEncoder(h).Encode(s.shelves); err != nil {
				t.Fatal(err)
			}
		}
	}
	const mainDigest = "b8b288e13ec90e4a898eb50a59a7d5e89114dace965103d763fe48083e690dda"
	if got := fmt.Sprintf("%x", h.Sum(nil)); got != mainDigest {
		t.Fatalf("default stock digest %s, want main %s", got, mainDigest)
	}
	t.Logf("main stock digest %s; %d ceilings, three seeds, four shelves", mainDigest, len(shelfTestCeilings(t, f)))
}

func TestReleaseShopStockMultiplierRollsEveryShelf(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := shopOrderFront(t)
	for _, ceiling := range shelfTestCeilings(t, f) {
		for _, seed := range []int64{1, 7, 42} {
			base := NewShop(ceiling)
			base.Generate(f.Table, seed)
			ns := []int32{3}
			if seed == 42 {
				ns = append(ns, MaxShopStockMultiplier)
			}
			for _, n := range ns {
				got := NewShop(ceiling)
				got.stockMultiplier = n
				start := time.Now()
				got.Generate(f.Table, seed)
				took := time.Since(start)
				assertShopRolledTimes(t, base, got, int64(n))
				for _, shelf := range got.shelves {
					for _, item := range shelf {
						if width := f.tipFont().Advance(ui.GroupDigits(int64(item.Count))) + 1; width > 69 {
							t.Fatalf("quantity %d needs %d pixels; cell allows 69", item.Count, width)
						}
					}
				}
				if n == MaxShopStockMultiplier {
					t.Logf("ceiling %d: N=%d generated in %v; entries %d/%d/%d/%d, one roll %d/%d/%d/%d", ceiling, n, took,
						len(got.shelves[ShelfArmour]), len(got.shelves[ShelfWeapons]), len(got.shelves[ShelfMagic]), len(got.shelves[ShelfBooks]),
						len(base.shelves[ShelfArmour]), len(base.shelves[ShelfWeapons]), len(base.shelves[ShelfMagic]), len(base.shelves[ShelfBooks]))
				}
			}
		}
	}
}
