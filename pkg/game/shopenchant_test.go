package game

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type enchantSpec struct {
	weight, cost, minimum, maximum int32
	usableBy                       int32
}

type enchantMagic map[int]enchantSpec

func (m enchantMagic) Len() int               { return 50 }
func (m enchantMagic) EntryName(i int) string { return "effect" }
func (m enchantMagic) EntryStrings(int) []string {
	return nil
}
func (m enchantMagic) EntryParams(i int) []int32 {
	p := make([]int32, 28)
	var endpoint int32
	for kind := 1; kind <= i; kind++ {
		endpoint += m[kind].weight
	}
	for column := 4; column < len(p); column++ {
		p[column] = endpoint
	}
	p[3] = 3
	if spec, ok := m[i]; ok {
		p[0], p[1], p[2] = spec.cost, spec.minimum, spec.maximum
		if spec.usableBy != 0 {
			p[3] = spec.usableBy
		}
	}
	return p
}

type enchantSpells map[int]int32

func (s enchantSpells) Len() int { return 21 }
func (s enchantSpells) EntryName(i int) string {
	return "spell"
}
func (s enchantSpells) EntryStrings(int) []string { return nil }
func (s enchantSpells) EntryParams(i int) []int32 {
	p := make([]int32, 21)
	p[20] = s[i]
	return p
}

type fixedDraws struct {
	values []int
	calls  []int
}

func (d *fixedDraws) Intn(n int) int {
	d.calls = append(d.calls, n)
	if len(d.values) == 0 {
		return 0
	}
	v := d.values[0]
	d.values = d.values[1:]
	if v < 0 || v >= n {
		panic("fixed draw outside requested range")
	}
	return v
}

func ordinaryEnchantTable(magic enchantMagic) *mapload.Table {
	return &mapload.Table{Magic: magic, Spells: enchantSpells{1: 1, 7: 1, 11: 1, 13: 1, 14: 1, 20: 1}}
}

func ordinaryCandidate() data.ShopCandidate {
	return data.ShopCandidate{Code: 0x0101, Price: 10, MagCap: 100, ItemKind: 2, EffectSlot: 1, Fighter: true}
}

func TestFixedShopDrawsProduceOneTwoAndThreeOrderedEffectsAtExactGates(t *testing.T) {
	table := ordinaryEnchantTable(enchantMagic{2: {weight: 1, cost: 1, minimum: 1, maximum: 100}})
	cases := []struct {
		name   string
		draws  []int
		bounds []int
	}{
		{"one", []int{0, 0, 50, 25}, []int{1, 100, 101, 101}},
		{"second gate 49", []int{0, 0, 49, 0, 0, 100}, []int{1, 100, 101, 1, 99, 101}},
		{"second gate 50 but third 24", []int{0, 0, 50, 24, 0, 0}, []int{1, 100, 101, 101, 1, 99}},
		{"third gate 25", []int{0, 0, 50, 25}, []int{1, 100, 101, 101}},
		{"three", []int{0, 0, 0, 0, 0, 0, 0, 0}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			draws := &fixedDraws{values: append([]int(nil), tc.draws...)}
			got, ok := shopEnchantedItem(ordinaryCandidate(), 100000, table, draws)
			want := []sim.ItemEffect{{Kind: 2, Operand: 1}}
			if !ok || !reflect.DeepEqual(got.Effects, want) {
				t.Fatalf("shopEnchantedItem = %+v, %v, want the single merged kind-2 effect", got, ok)
			}
			if tc.bounds != nil && !reflect.DeepEqual(draws.calls, tc.bounds) {
				t.Errorf("Intn bounds = %v, want %v", draws.calls, tc.bounds)
			}
		})
	}
}

func TestShopNullAndCastEarlyReturnsUseTheirExactDrawPrefixes(t *testing.T) {
	ordinary := enchantSpec{weight: 1, cost: 1, minimum: 1, maximum: 100}
	for _, tc := range []struct {
		name   string
		magic  enchantMagic
		draws  []int
		kind   uint8
		calls  int
		forced bool
	}{
		{"optional null", enchantMagic{2: ordinary, 3: {weight: 1}}, []int{0, 0, 0, 1}, 2, 4, false},
		{"optional cast discarded", enchantMagic{2: ordinary, 41: {weight: 1}}, []int{0, 0, 0, 1, 0, 0}, 2, 6, false},
		{"first cast retained", enchantMagic{41: {weight: 1}}, []int{0, 0, 0}, 41, 3, false},
		{"forced cast retained", enchantMagic{}, []int{0, 0}, 41, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			draws := &fixedDraws{values: append([]int(nil), tc.draws...)}
			candidate := ordinaryCandidate()
			candidate.ForcedCast = tc.forced
			got, ok := shopEnchantedItem(candidate, 100000, ordinaryEnchantTable(tc.magic), draws)
			if !ok || len(got.Effects) != 1 || got.Effects[0].Kind != tc.kind {
				t.Fatalf("shopEnchantedItem = %+v, %v, want one retained kind %d", got, ok, tc.kind)
			}
			if len(draws.calls) != tc.calls || len(draws.values) != 0 {
				t.Fatalf("draw calls=%v remaining=%v, want exactly %d calls", draws.calls, draws.values, tc.calls)
			}
		})
	}
}

func TestRangeEffectConsumesDiscardedBaseAndSpreadDraws(t *testing.T) {
	magic := enchantMagic{44: {weight: 1, cost: 1, minimum: 1, maximum: 100}}
	draws := &fixedDraws{values: []int{2, 4, 2}}
	got, ok := shopOrdinaryEffect(magic, 44, 200000, 100, draws)
	if !ok {
		t.Fatal("shopOrdinaryEffect refused a feasible range")
	}
	want := generatedEffect{effect: sim.ItemEffect{Kind: 44, Operand: 5 | 3<<8}, points: 8}
	if got != want || len(draws.calls) != 3 {
		t.Fatalf("range = %+v calls=%v, want %+v and three payload draws", got, draws.calls, want)
	}
}

func TestEffectPriceUsesAggregateAdditionAndClamp(t *testing.T) {
	table := ordinaryEnchantTable(enchantMagic{2: {weight: 1, cost: 1, minimum: 1, maximum: 1}})
	for _, tc := range []struct {
		base, ceiling int32
		want          int32
	}{
		{10, 100000, 10 + shopNonCastPrice(1)},
		{9_999_990, 10_000_000, 9_999_999},
	} {
		candidate := ordinaryCandidate()
		candidate.Price = tc.base
		draws := &fixedDraws{values: []int{0, 0, 100, 100}}
		got, ok := shopEnchantedItem(candidate, tc.ceiling, table, draws)
		if !ok || got.Price != tc.want {
			t.Errorf("base %d: item=%+v ok=%v, want price %d", tc.base, got, ok, tc.want)
		}
	}
}

func TestMagicShelfUsesTheFullTwoHundredAndOneFailureAttempts(t *testing.T) {
	invalidMagic := enchantMagic{2: {weight: 1}}
	table := &mapload.Table{
		Shapes: unitScale(data.ShopShapes), Materials: unitScale(data.ShopMaterials),
		Weapons: shopCollection{{}, {name: "axe", price: 40, masks: [data.ShopShapes]uint16{1}}},
		Magic:   invalidMagic,
	}
	draws := &fixedDraws{}
	shop := NewShop(1000)
	shop.generate(table, draws)
	if got := len(shop.Shelf(ShelfMagic)); got != 0 {
		t.Fatalf("invalid Magic shelf holds %d row(s)", got)
	}
	// 100 weapon draws, then 201 Magic attempts. Each Magic attempt draws a
	// base candidate and a cumulative kind before the zero-cost row rejects.
	if got, want := len(draws.calls), 100+201*2; got != want {
		t.Fatalf("generator made %d draws, want %d (the inclusive 0..200 attempt run)", got, want)
	}
}

func TestShopCellsKeepDistinctEffectsAndCloneAcrossEveryModelTransfer(t *testing.T) {
	a := ShopItem{Code: 0x0101, Kind: 2, Effects: []sim.ItemEffect{{Kind: 15, Operand: 4}}, Price: 90, Count: 1}
	b := ShopItem{Code: 0x0101, Kind: 2, Effects: []sim.ItemEffect{{Kind: 15, Operand: 5}}, Price: 90, Count: 1}
	folded := shopStackShelf([]ShopItem{a, b, a})
	if len(folded) != 2 || folded[0].Count != 2 || folded[1].Count != 1 {
		t.Fatalf("effect-aware shelf fold = %+v", folded)
	}
	a.Effects[0].Operand = 99
	if folded[0].Effects[0].Operand != 4 {
		t.Fatal("shelf fold retained an input effect alias")
	}

	shop := &Shop{ceiling: 1000}
	shop.shelves[ShelfWeapons] = []ShopItem{folded[0].Clone()}
	if !shop.TakeFromShelf(ShelfWeapons, 0, 1) {
		t.Fatal("TakeFromShelf refused the enchanted item")
	}
	view := shop.Table()
	view[0].ShopItem.Effects[0].Operand = 77
	if shop.Table()[0].ShopItem.Effects[0].Operand != 4 {
		t.Fatal("Table exposed an effect alias")
	}
	shop.ClearTable()
	if shelf := shop.Shelf(ShelfWeapons); len(shelf) != 1 || shelf[0].Count != 2 || shelf[0].Effects[0].Operand != 4 {
		t.Fatalf("cancellation did not restore the complete shelf instance: %+v", shelf)
	}
	shop.TakeFromShelf(ShelfWeapons, 0, 1)
	bought, _, ok := shop.Buy(1000)
	if !ok || len(bought) != 1 || bought[0].Effects[0].Operand != 4 {
		t.Fatalf("buy did not return the complete instance: %+v", bought)
	}
	put := ShopItem{Code: b.Code, Kind: b.Kind, Effects: append([]sim.ItemEffect(nil), b.Effects...), Price: b.Price, Count: 2}
	if !shop.PutOnTable(put) {
		t.Fatal("PutOnTable refused the enchanted item")
	}
	put.Effects[0].Operand = 88
	off, ok := shop.TakeOffTable(0, 1)
	if !ok || off.ShopItem.Effects[0].Operand != 5 {
		t.Fatalf("TakeOffTable did not return the complete cloned instance: %+v", off)
	}
	off.ShopItem.Effects[0].Operand = 66
	if table := shop.Table(); len(table) != 1 || table[0].ShopItem.Count != 1 || table[0].ShopItem.Effects[0].Operand != 5 {
		t.Fatalf("partial TakeOffTable exposed an internal effect alias: %+v", table)
	}
	if _, ok := shop.Sell(); !ok {
		t.Fatal("Sell refused the enchanted item")
	}
	magic := shop.Shelf(ShelfMagic)
	if len(magic) != 1 || magic[0].Count != 1 || magic[0].Effects[0].Operand != 5 {
		t.Fatalf("sell collapsed a distinct enchantment: %+v", magic)
	}
	if weapons := shop.Shelf(ShelfWeapons); len(weapons) != 1 || weapons[0].Count != 1 || weapons[0].Effects[0].Operand != 4 {
		t.Fatalf("sell changed the remaining shelf instance: %+v", weapons)
	}
}

func repeatShopItem(t *testing.T, magic enchantMagic, values []int) (ShopItem, *fixedDraws) {
	t.Helper()
	draws := &fixedDraws{values: append([]int(nil), values...)}
	got, ok := shopEnchantedItem(ordinaryCandidate(), 100000, ordinaryEnchantTable(magic), draws)
	if !ok {
		t.Fatal("shopEnchantedItem refused a feasible item")
	}
	return got, draws
}

func TestShopRepeatedKindOverwritesTheHeldPayloadInPlace(t *testing.T) {
	magic := enchantMagic{2: {weight: 1, cost: 1, minimum: 1, maximum: 100}}
	for _, tc := range []struct {
		name   string
		values []int
		bounds []int
		want   uint32
	}{
		{"lower", []int{0, 49, 0, 0, 9, 100}, []int{1, 100, 101, 1, 50, 101}, 10},
		{"higher", []int{0, 9, 0, 0, 49, 100}, []int{1, 100, 101, 1, 90, 101}, 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, draws := repeatShopItem(t, magic, tc.values)
			if want := []sim.ItemEffect{{Kind: 2, Operand: tc.want}}; !reflect.DeepEqual(got.Effects, want) {
				t.Fatalf("effects = %+v, want %+v", got.Effects, want)
			}
			if wantPrice := 10 + shopNonCastPrice(int32(tc.want)); got.Price != wantPrice {
				t.Errorf("price = %d, want %d from the stored payload only", got.Price, wantPrice)
			}
			if !reflect.DeepEqual(draws.calls, tc.bounds) {
				t.Errorf("Intn bounds = %v, want %v", draws.calls, tc.bounds)
			}
		})
	}
}

func TestShopUsableByConflictDropsTheNewEffect(t *testing.T) {
	magic := enchantMagic{
		2: {weight: 1, cost: 1, minimum: 1, maximum: 100, usableBy: 1},
		3: {weight: 1, cost: 1, minimum: 1, maximum: 100, usableBy: 2},
	}
	got, draws := repeatShopItem(t, magic, []int{0, 4, 0, 1, 6, 100})
	if want := []sim.ItemEffect{{Kind: 2, Operand: 5}}; !reflect.DeepEqual(got.Effects, want) {
		t.Fatalf("effects = %+v, want %+v", got.Effects, want)
	}
	if wantPrice := 10 + shopNonCastPrice(5); got.Price != wantPrice {
		t.Errorf("price = %d, want %d", got.Price, wantPrice)
	}
	if want := []int{2, 100, 101, 2, 95, 101}; !reflect.DeepEqual(draws.calls, want) {
		t.Errorf("Intn bounds = %v, want %v", draws.calls, want)
	}
}

func TestShopNewKindAppendsAndEachKindKeepsOneEntry(t *testing.T) {
	magic := enchantMagic{
		2: {weight: 1, cost: 1, minimum: 1, maximum: 100},
		3: {weight: 1, cost: 2, minimum: 1, maximum: 100},
	}
	got, draws := repeatShopItem(t, magic, []int{0, 4, 0, 1, 5, 0, 0, 9})
	want := []sim.ItemEffect{{Kind: 2, Operand: 10}, {Kind: 3, Operand: 6}}
	if !reflect.DeepEqual(got.Effects, want) {
		t.Fatalf("effects = %+v, want %+v", got.Effects, want)
	}
	if wantPrice := 10 + shopNonCastPrice(10+12); got.Price != wantPrice {
		t.Errorf("price = %d, want %d", got.Price, wantPrice)
	}
	if len(draws.calls) != 8 {
		t.Errorf("draw count = %d, want 8", len(draws.calls))
	}
}
