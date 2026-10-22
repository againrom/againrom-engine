package game

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/random"
)

func shopShelfUnits(items []ShopItem) int64 {
	var n int64
	for _, item := range items {
		n += int64(item.Count)
	}
	return n
}

func shopIsPotion(item ShopItem) bool {
	return item.Kind != 4 && item.Kind != 5
}

// assertShopRolledTimes checks got against base, the same shop and seed
// generated with one roll: the first three shelves hold n times the drawn
// units, the books shelf n times the random picks, the first roll is kept
// whole, and the further units are new draws rather than larger stacks.
func assertShopRolledTimes(t *testing.T, base, got *Shop, n int64) {
	t.Helper()
	if n == 1 {
		for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
			if !reflect.DeepEqual(got.Shelf(shelf), base.Shelf(shelf)) {
				t.Fatalf("shelf %s with one roll differs from the default stock", shelf)
			}
		}
		return
	}
	for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
		b, g := base.Shelf(shelf), got.Shelf(shelf)
		for _, want := range b {
			found := false
			for _, item := range g {
				if shopItemEqual(item, want) && item.Count >= want.Count {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("shelf %s lost the first roll's %v", shelf, want)
			}
		}
		if shelf == ShelfBooks {
			extra := shopShelfUnits(g) - shopShelfUnits(b)
			picked := slices.ContainsFunc(b, func(item ShopItem) bool { return !shopIsPotion(item) })
			if picked && (extra%(n-1) != 0 || extra/(n-1) < 1 || extra/(n-1) > 8) || !picked && extra != 0 {
				t.Fatalf("books shelf gained %d units over %d further rolls", extra, n-1)
			}
			var bp, gp []ShopItem
			for _, item := range b {
				if shopIsPotion(item) {
					bp = append(bp, item)
				}
			}
			for _, item := range g {
				if shopIsPotion(item) {
					gp = append(gp, item)
				}
			}
			if !reflect.DeepEqual(bp, gp) {
				t.Fatal("potion stacks changed with the multiplier", bp, gp)
			}
			continue
		}
		if len(b) == 0 {
			continue
		}
		if shopShelfUnits(g) != n*shopShelfUnits(b) {
			t.Fatalf("shelf %s holds %d units, want %d times %d", shelf, shopShelfUnits(g), n, shopShelfUnits(b))
		}
		scaled := make([]ShopItem, len(b))
		for i, item := range b {
			scaled[i] = item.Clone()
			scaled[i].Count *= int32(n)
		}
		if reflect.DeepEqual(g, scaled) {
			t.Fatalf("shelf %s is the one roll with every quantity multiplied by %d", shelf, n)
		}
	}
}

func TestShopStockMultiplierRollsEveryShelf(t *testing.T) {
	for _, seed := range []int64{1, 7, 42} {
		base := NewShop(1000)
		base.Generate(shopTable(), seed)
		for _, n := range []int32{1, 2, 3, 25} {
			got := NewShop(1000)
			got.stockMultiplier = n
			got.Generate(shopTable(), seed)
			assertShopRolledTimes(t, base, got, int64(n))
			if !got.Restock(shopTable(), seed) {
				t.Fatal("restock refused")
			}
			control := NewShop(1000)
			control.Restock(shopTable(), seed)
			assertShopRolledTimes(t, control, got, int64(n))
		}
	}
}

func TestShopStockMultiplierMagicEntriesAreOwnDraws(t *testing.T) {
	base := NewShop(1000)
	base.Generate(shopTable(), 42)
	got := NewShop(1000)
	got.stockMultiplier = 3
	got.Generate(shopTable(), 42)
	magic := got.Shelf(ShelfMagic)
	if len(magic) == 0 {
		t.Fatal("fixture has no magic stock")
	}
	kinds := map[string]bool{}
	for _, item := range magic {
		kinds[fmt.Sprint(item.Code, item.Effects)] = true
	}
	if len(kinds) < 2 {
		t.Fatal("three rolls produced one magic item kind")
	}
	t.Logf("magic shelf: %d entries with one roll, %d with three, %d distinct identities", len(base.Shelf(ShelfMagic)), len(magic), len(kinds))
}

func TestShopStockMultiplierLaunchSurvivesSessionReset(t *testing.T) {
	f, _ := shopRoom(t, nil)
	if err := f.SetShopStockMultiplier(5); err != nil {
		t.Fatal(err)
	}
	f.arriveInTown()
	base := NewShop(f.Shop.Ceiling())
	base.Generate(f.Table, f.Shop.drawSeed)
	assertShopRolledTimes(t, base, f.Shop, 5)
	f.resetSessionForNewGame()
	if f.runtime.shopStockMultiplier != 5 || f.townInstall().stockMultiplier != 5 {
		t.Fatal("session reset lost the launch option")
	}
	for _, n := range []int{0, -1, MaxShopStockMultiplier + 1} {
		if f.SetShopStockMultiplier(n) == nil || f.runtime.shopStockMultiplier != 5 {
			t.Fatal("invalid multiplier changed the launch option", n)
		}
	}
	if f.SetShopStockMultiplier(MaxShopStockMultiplier) != nil || f.runtime.shopStockMultiplier != MaxShopStockMultiplier {
		t.Fatal("the bound was refused")
	}
}

func TestShopStockMultiplierKeepsSharedRandomDraws(t *testing.T) {
	a := random.NewService(random.Session{Seed: 42, Shared: 42, Mode: random.Original})
	b := random.NewService(random.Session{Seed: 42, Shared: 42, Mode: random.Original})
	base, got := NewShop(1000), NewShop(1000)
	got.stockMultiplier = 5
	base.GenerateWith(shopTable(), 42, a.Stream(random.ShopStock))
	got.GenerateWith(shopTable(), 42, b.Stream(random.ShopStock))
	assertShopRolledTimes(t, base, got, 5)
	if !reflect.DeepEqual(a.Session(), b.Session()) {
		t.Fatal("further rolls consumed shared random draws")
	}
}

// shopStackShelfByScan is the stacking rule as a plain scan of every earlier
// element, the reference the keyed stacker must equal.
func shopStackShelfByScan(items []ShopItem) []ShopItem {
	var out []ShopItem
	for _, item := range items {
		found := false
		for i := range out {
			if shopItemEqual(out[i], item) {
				out[i].Count += item.Count
				found = true
				break
			}
		}
		if !found {
			out = append(out, item.Clone())
		}
	}
	return out
}

func TestShopStackShelfMatchesScan(t *testing.T) {
	s := NewShop(1000)
	s.stockMultiplier = 4
	s.Generate(shopTable(), 9)
	r := random.NewGo(3)
	var items []ShopItem
	for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
		items = append(items, s.Shelf(shelf)...)
	}
	mixed := make([]ShopItem, 0, 3*len(items))
	for i := 0; i < 3*len(items); i++ {
		item := items[r.Intn(len(items))].Clone()
		item.Count = int32(1 + r.Intn(3))
		mixed = append(mixed, item)
	}
	if got, want := shopStackShelf(mixed), shopStackShelfByScan(mixed); !reflect.DeepEqual(got, want) {
		t.Fatal("keyed stacking differs from the scan")
	}
}
