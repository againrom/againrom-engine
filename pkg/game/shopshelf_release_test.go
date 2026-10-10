package game

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/sim"
)

// The two Common Amulet codes whose installed names are Magic Beard and Beard
// (SHOP-119). Production code names neither; these tests use them only to
// witness that the shelf tests remove them.
const (
	shelfTestMagicBeard data.ItemCode = 0x0502
	shelfTestBeard      data.ItemCode = 0x1502
)

// shelfTestCeilings is every ShopMaxPrice the installed campaign ships plus a
// ceiling above every item value.
func shelfTestCeilings(t *testing.T, f *FrontEnd) []int32 {
	t.Helper()
	seen := map[int32]bool{1_000_000_000: true}
	for _, ch := range f.Campaign.Value().Chapters {
		if ch.ShopMax > 0 {
			seen[int32(ch.ShopMax)] = true
		}
	}
	if len(seen) < 2 {
		t.Fatal("installed campaign ships no ShopMaxPrice")
	}
	out := make([]int32, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func shelfTestCodes(pool []data.ShopCandidate) map[data.ItemCode]data.ShopCandidate {
	out := make(map[data.ItemCode]data.ShopCandidate, len(pool))
	for _, c := range pool {
		out[c.Code] = c
	}
	return out
}

// shelfTestMinus is the codes of a absent from b, sorted.
func shelfTestMinus(a, b map[data.ItemCode]data.ShopCandidate) []data.ItemCode {
	var out []data.ItemCode
	for code := range a {
		if _, ok := b[code]; !ok {
			out = append(out, code)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// shelfTestPreviousPool is the stock rule this engine applied before the
// shelf tests: both beard codes dropped from every shelf and a ForcedCast
// weapon dropped from the plain weapons shelf.
func shelfTestPreviousPool(pool []data.ShopCandidate, magic bool) []data.ShopCandidate {
	var out []data.ShopCandidate
	for _, c := range pool {
		if c.Code == shelfTestMagicBeard || c.Code == shelfTestBeard || c.ForcedCast && !magic {
			continue
		}
		out = append(out, c)
	}
	return out
}

// Each shelf's candidate population before and after the shelf tests, by
// code, at every shipped ceiling. Only Ring and Amulet codes leave the armour
// shelf, no weapons code changes, and the Magic Items population gains exactly
// the two beards, which the first-effect rule then rejects. The counts at the
// four ceilings SHOP-120 states match its weapons and armour counts.
func TestReleaseShopShelfPopulationsFollowShelfTests(t *testing.T) {
	f := releaseFront(t)
	claimed := map[int32][2]int{1_000: {46, 87}, 10_000: {70, 143}, 100_000: {106, 189}, 1_000_000: {128, 204}}
	for _, ceiling := range shelfTestCeilings(t, f) {
		rawArmour := shopArmourPool(f.Table, ceiling)
		rawWeapons := shopWeaponPool(f.Table, ceiling)
		rawMagic := append(append([]data.ShopCandidate{}, rawArmour...), rawWeapons...)
		before := [numShopShelves]map[data.ItemCode]data.ShopCandidate{
			ShelfArmour:  shelfTestCodes(shelfTestPreviousPool(rawArmour, false)),
			ShelfWeapons: shelfTestCodes(shelfTestPreviousPool(rawWeapons, false)),
			ShelfMagic:   shelfTestCodes(shelfTestPreviousPool(rawMagic, true)),
		}
		after := [numShopShelves]map[data.ItemCode]data.ShopCandidate{
			ShelfArmour:  shelfTestCodes(shopPlainShelfPool(rawArmour)),
			ShelfWeapons: shelfTestCodes(shopPlainShelfPool(rawWeapons)),
			ShelfMagic:   shelfTestCodes(rawMagic),
		}
		for _, shelf := range []ShopShelf{ShelfWeapons, ShelfArmour, ShelfMagic} {
			left := shelfTestMinus(before[shelf], after[shelf])
			entered := shelfTestMinus(after[shelf], before[shelf])
			var lines []string
			for _, c := range left {
				lines = append(lines, fmt.Sprintf("-%#04x", uint16(c)))
			}
			for _, c := range entered {
				lines = append(lines, fmt.Sprintf("+%#04x", uint16(c)))
			}
			t.Logf("ceiling %d %s: before %d after %d; %s", ceiling, shelf, len(before[shelf]), len(after[shelf]),
				strings.Join(lines, ", "))
			switch shelf {
			case ShelfWeapons:
				if len(left)+len(entered) != 0 {
					t.Errorf("ceiling %d weapons shelf changed: left %v entered %v", ceiling, left, entered)
				}
			case ShelfArmour:
				if len(entered) != 0 {
					t.Errorf("ceiling %d armour shelf gained %v", ceiling, entered)
				}
				for _, code := range left {
					c := before[shelf][code]
					if c.ItemKind != 1 || c.EffectSlot < 3 || c.EffectSlot > 5 {
						t.Errorf("ceiling %d armour shelf lost %v, not an Armor of class 3..5: %+v", ceiling, code, c)
					}
				}
				for code, c := range after[shelf] {
					if c.ItemKind == 1 && c.EffectSlot >= 3 && c.EffectSlot <= 5 {
						t.Errorf("ceiling %d armour shelf keeps class %d code %v", ceiling, c.EffectSlot, code)
					}
				}
			case ShelfMagic:
				var want []data.ItemCode
				for _, code := range []data.ItemCode{shelfTestMagicBeard, shelfTestBeard} {
					if _, ok := after[shelf][code]; ok {
						want = append(want, code)
					}
				}
				if len(left) != 0 || len(entered) != len(want) || len(want) != 0 && entered[0] != want[0] ||
					len(want) == 2 && entered[1] != want[1] {
					t.Errorf("ceiling %d Magic Items population left %v entered %v, want entered %v", ceiling, left, entered, want)
				}
			}
		}
		for _, code := range []data.ItemCode{shelfTestMagicBeard, shelfTestBeard} {
			if c, ok := shelfTestCodes(rawArmour)[code]; ok {
				if _, kept := after[ShelfArmour][code]; kept || c.EffectSlot != 5 {
					t.Errorf("ceiling %d beard %v class %d kept=%v on the armour shelf", ceiling, code, c.EffectSlot, kept)
				}
			}
		}
		if ceiling >= 1_000_000 {
			if _, ok := after[ShelfMagic][shelfTestMagicBeard]; !ok {
				t.Errorf("ceiling %d raw walk lacks Magic Beard; the witness would prove nothing", ceiling)
			}
			if _, ok := after[ShelfMagic][shelfTestBeard]; !ok {
				t.Errorf("ceiling %d raw walk lacks Beard; the witness would prove nothing", ceiling)
			}
		}
		want, ok := claimed[ceiling]
		if ceiling > 1_000_000 {
			want, ok = claimed[1_000_000], true
		}
		if ok && (len(after[ShelfWeapons]) != want[0] || len(after[ShelfArmour]) != want[1]) {
			t.Errorf("ceiling %d weapons/armour codes %d/%d, SHOP-120 states %d/%d", ceiling,
				len(after[ShelfWeapons]), len(after[ShelfArmour]), want[0], want[1])
		}
	}
}

// No generated shelf holds either beard at any shipped ceiling, and the
// Magic Items draw rejects each beard every time it is drawn because neither
// can take a first effect (SHOP-119).
func TestReleaseGeneratedShelvesNeverHoldABeard(t *testing.T) {
	f := releaseFront(t)
	ceilings := shelfTestCeilings(t, f)
	for _, ceiling := range ceilings {
		for seed := int64(1); seed <= 30; seed++ {
			s := NewShop(ceiling)
			s.Generate(f.Table, seed)
			for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
				for _, item := range s.Shelf(shelf) {
					if item.Code == shelfTestMagicBeard || item.Code == shelfTestBeard {
						t.Fatalf("ceiling %d seed %d: %s shelf holds %v", ceiling, seed, shelf, item.Code)
					}
				}
			}
		}
	}
	raw := shelfTestCodes(shopArmourPool(f.Table, 1_000_000_000))
	for code, magCap := range map[data.ItemCode]int32{shelfTestMagicBeard: 0, shelfTestBeard: 3} {
		c, ok := raw[code]
		if !ok || c.MagCap != magCap || c.EffectSlot != 5 || !c.Fighter {
			t.Fatalf("raw candidate %v = %+v (present %v), want MagCap %d, class 5, fighter column", code, c, ok, magCap)
		}
		for _, ceiling := range ceilings {
			for seed := int64(1); seed <= 200; seed++ {
				if item, ok := shopEnchantedItem(c, ceiling, f.Table, random.NewGo(seed)); ok {
					t.Fatalf("ceiling %d seed %d: %v took a first effect: %+v", ceiling, seed, code, item)
				}
			}
		}
	}
}

// A hero sells each beard made by the #create route in a town shop: the sale
// is credited and the beard lands on the armour shelf like any other armour.
func TestReleaseSoldBeardIsCreditedAndShelvedAsArmour(t *testing.T) {
	installed := releaseFront(t)
	for _, code := range []data.ItemCode{shelfTestMagicBeard, shelfTestBeard} {
		name, ok := installed.Table.Names.NameFor(code)
		if !ok || name == "" {
			t.Fatalf("installed item names lack %v", code)
		}
		item, ok := mapload.CheatItem(name, installed.Table)
		if !ok || data.ItemCode(item.Code) != code || item.Price <= 0 {
			t.Fatalf("#create %q = %+v (ok %v), want code %v with a positive price", name, item, ok, code)
		}
		f, screen := shopRoom(t, nil)
		f.Table = installed.Table
		f.Shop = NewShop(1_000_000)
		f.Shop.Generate(f.Table, 7)
		f.Carried[0].ID = "hero"
		cityShopSetTestPack(t, f, 0, sim.StackItem(item, 1))
		cityShopGraph(t, f)
		count := func(shelf ShopShelf) int32 {
			n := int32(0)
			for _, stock := range f.Shop.Shelf(shelf) {
				if stock.Code == code {
					n += stock.Count
				}
			}
			return n
		}
		var before [numShopShelves]int32
		for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
			before[shelf] = count(shelf)
		}
		if action := screen.shopFromPack(0, true); strings.Contains(action.Msg, "cannot") || len(f.Shop.Table()) != 1 {
			t.Fatalf("%v: staging refused: %+v table %+v", code, action, f.Shop.Table())
		}
		gold := f.Town.Gold()
		if action := screen.shopSell(); strings.Contains(action.Msg, "cannot") {
			t.Fatalf("%v: sale refused: %+v", code, action)
		}
		if paid := int32(f.Town.Gold() - gold); paid != shopHalfUp(item.Price) || paid <= 0 {
			t.Fatalf("%v: sale paid %d, want %d", code, paid, shopHalfUp(item.Price))
		}
		for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
			want := before[shelf]
			if shelf == ShelfArmour {
				want++
			}
			if got := count(shelf); got != want {
				t.Fatalf("%v: %s shelf holds %d after the sale, want %d", code, shelf, got, want)
			}
		}
		t.Logf("%v %q: price %d, paid %d, armour shelf", code, name, item.Price, shopHalfUp(item.Price))
	}
}
