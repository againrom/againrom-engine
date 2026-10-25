package game

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"againrom/internal/firstgame"
)

// TestReleaseShopStockHoldsOneEntryPerEffectKind generates stock for every
// shop price ceiling of the installed campaign over 100 seeds and requires no
// item to hold two effects of one kind. When AGAINROM_SHOP_CENSUS_OUT names a
// file, it also lists every generated item there.
func TestReleaseShopStockHoldsOneEntryPerEffectKind(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := shopOrderFront(t)
	var out strings.Builder
	items, repeats := 0, 0
	sample := ""
	for _, ceiling := range shelfTestCeilings(t, f) {
		for seed := int64(1); seed <= 100; seed++ {
			s := NewShop(ceiling)
			s.Generate(f.Table, seed)
			for shelf := range s.shelves {
				for i, item := range s.shelves[shelf] {
					items++
					seen := map[uint8]bool{}
					repeated := false
					for _, e := range item.Effects {
						if seen[e.Kind] {
							repeated = true
						}
						seen[e.Kind] = true
					}
					line := fmt.Sprintf("%d %d %d %d code=%d price=%d effects=%v", ceiling, seed, shelf, i, item.Code, item.Price, item.Effects)
					if repeated {
						repeats++
						if sample == "" {
							sample = line
						}
					}
					out.WriteString(line + "\n")
				}
			}
		}
	}
	if path := os.Getenv("AGAINROM_SHOP_CENSUS_OUT"); path != "" {
		if err := os.WriteFile(path, []byte(out.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d cells, %d with two effects of one kind; first: %s", items, repeats, sample)
	if repeats != 0 {
		t.Fatalf("%d stocked cells hold two effects of one kind", repeats)
	}
}
