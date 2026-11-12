package game

import (
	"fmt"
	"testing"

	"againrom/pkg/random"
)

func secondShopStockUnits(s *Shop) (units int) {
	for _, shelf := range s.stock() {
		for _, it := range shelf {
			units += int(it.Count)
		}
	}
	return units
}

// The launch's shop stock multiplier reaches the second game's shop: a Sleep
// at a factor of three rolls every shelf again, the shared stock stream is
// consumed as with one, and a factor of one is the unchanged stock.
func TestSecondShopStockMultiplierReachesTheShop(t *testing.T) {
	for _, id := range []int{1, 2, 3} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			f, s := sleepTown(id)
			one, ones := sleepTown(id)
			s.switches, ones.switches = &f.runtime, &one.runtime
			// Ten scrolls, so a further roll brings a scroll the first did
			// not; a repeated scroll adds nothing (DIV-4870).
			for _, rows := range []sleepCollection{f.Table.MagicItems.(sleepCollection), one.Table.MagicItems.(sleepCollection)} {
				for i := 2; i <= 10; i++ {
					rows.secondShopTestCollection[i+5], rows.secondShopTestCollection[i+34] = []int32{10}, []int32{10}
				}
			}
			f.runtime.shopStockMultiplier = 3
			one.runtime.shopStockMultiplier = 1
			chooseSleep(t, s)
			chooseSleep(t, ones)
			base, got := one.Town.second.shop, f.Town.second.shop
			if a, b := secondShopStockUnits(&base), secondShopStockUnits(&got); b <= a {
				t.Fatalf("units with three rolls %d, with one %d", b, a)
			}
			if a, b := f.randomService().Stream(random.ShopStock).Raw(), one.randomService().Stream(random.ShopStock).Raw(); a != b {
				t.Fatal("the multiplier consumed another number of stock draws", a, b)
			}
		})
	}
}
