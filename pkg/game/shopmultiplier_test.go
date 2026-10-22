package game

import (
	"reflect"
	"testing"

	"againrom/pkg/random"
)

func assertShopStockMultiple(t *testing.T, base, got *Shop, n int32) {
	t.Helper()
	for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
		want := base.Shelf(shelf)
		for i := range want {
			want[i].Count *= n
		}
		if !reflect.DeepEqual(got.Shelf(shelf), want) {
			t.Fatalf("shelf %s is not exactly %d times the baseline stock", shelf, n)
		}
	}
}

func TestShopStockMultiplierGenerationAndRestock(t *testing.T) {
	base := NewShop(1000)
	base.Generate(shopTable(), 42)
	for _, n := range []int32{1, 5, MaxShopStockMultiplier} {
		got := NewShop(1000)
		got.stockMultiplier = n
		got.Generate(shopTable(), 42)
		assertShopStockMultiple(t, base, got, n)
		if !got.Restock(shopTable(), 42) {
			t.Fatal("restock refused")
		}
		control := NewShop(1000)
		control.Restock(shopTable(), 42)
		assertShopStockMultiple(t, control, got, n)
	}
}

func TestShopStockMultiplierLaunchSurvivesSessionReset(t *testing.T) {
	f, _ := shopRoom(t, nil)
	if err := f.SetShopStockMultiplier(5); err != nil {
		t.Fatal(err)
	}
	f.arriveInTown()
	base := NewShop(f.Shop.Ceiling())
	base.Generate(f.Table, f.Shop.drawSeed)
	assertShopStockMultiple(t, base, f.Shop, 5)
	f.resetSessionForNewGame()
	if f.runtime.shopStockMultiplier != 5 || f.townInstall().stockMultiplier != 5 {
		t.Fatal("session reset lost the launch option")
	}
	for _, n := range []int{0, -1, MaxShopStockMultiplier + 1} {
		if f.SetShopStockMultiplier(n) == nil || f.runtime.shopStockMultiplier != 5 {
			t.Fatal("invalid multiplier changed the launch option", n)
		}
	}
}

func TestShopStockMultiplierKeepsSharedRandomDraws(t *testing.T) {
	a := random.NewService(random.Session{Seed: 42, Shared: 42, Mode: random.Original})
	b := random.NewService(random.Session{Seed: 42, Shared: 42, Mode: random.Original})
	base, got := NewShop(1000), NewShop(1000)
	got.stockMultiplier = 5
	base.GenerateWith(shopTable(), 42, a.Stream(random.ShopStock))
	got.GenerateWith(shopTable(), 42, b.Stream(random.ShopStock))
	assertShopStockMultiple(t, base, got, 5)
	if !reflect.DeepEqual(a.Session(), b.Session()) {
		t.Fatal("multiplier consumed extra shared random draws")
	}
}
