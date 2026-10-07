package mapload

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

type itemTestRow struct {
	name   string
	params []int32
}

type itemTestCollection []itemTestRow

func (c itemTestCollection) Len() int                  { return len(c) }
func (c itemTestCollection) EntryName(i int) string    { return c[i].name }
func (c itemTestCollection) EntryParams(i int) []int32 { return c[i].params }
func (c itemTestCollection) EntryStrings(int) []string { return nil }

type itemTestScale int

func (s itemTestScale) Len() int           { return int(s) }
func (itemTestScale) EntryName(int) string { return "" }
func (itemTestScale) EntryDoubles(int) []float64 {
	return []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}
}

func TestEquipmentCellParserPreservesOrderDuplicatesModesAndTheMalformedToken(t *testing.T) {
	spells := make(itemTestCollection, 15)
	spells[7].name = "Lightning"
	table := &Table{Spells: spells}
	head, effects, rejected, err := ParseItemCell(
		"Rare Staff {defence=7,defence=9,healthRegeneration=100:duration60,castSpell=Lightning:15,unknown=4}", table)
	if err != nil {
		t.Fatalf("ParseItemCell: %v", err)
	}
	want := []sim.ItemEffect{
		{Kind: 15, Operand: 7},
		{Kind: 15, Operand: 9},
		{Kind: 8, Mode: 1, Operand: 0x03c00064},
		{Kind: 41, Operand: uint32(7) | uint32(uint16(15))<<16},
	}
	if head != "Rare Staff" || rejected != 1 || !reflect.DeepEqual(effects, want) {
		t.Fatalf("head=%q rejected=%d effects=%+v, want %q/1/%+v", head, rejected, effects, "Rare Staff", want)
	}

	_, malformed, rejected, err := ParseItemCell(
		"Boots{defence=7,Rare Mithrill Chain Boots{defence=19}", table)
	if err != nil {
		t.Fatalf("malformed ParseItemCell: %v", err)
	}
	if rejected != 1 || !reflect.DeepEqual(malformed, []sim.ItemEffect{{Kind: 15, Operand: 7}}) {
		t.Fatalf("malformed token produced rejected=%d effects=%+v", rejected, malformed)
	}
}

func TestEquipmentCellParserAcceptsTheShippedWhitespaceBeforeEffectKeys(t *testing.T) {
	_, effects, rejected, err := ParseItemCell("Ring { protectionFire=18, protectionAir=25}", nil)
	if err != nil {
		t.Fatalf("ParseItemCell: %v", err)
	}
	want := []sim.ItemEffect{{Kind: 21, Operand: 18}, {Kind: 23, Operand: 25}}
	if !reflect.DeepEqual(effects, want) || rejected != 0 {
		t.Fatalf("effects/rejected = %+v/%d, want %+v/0", effects, rejected, want)
	}
}

func TestEquipmentInstanceStoresAggregateOrdinaryAndCastEffectPrices(t *testing.T) {
	magic := make(itemTestCollection, 50)
	magic[12].params = []int32{10}
	magic[23].params = []int32{10}
	spells := make(itemTestCollection, 8)
	spells[7].params = make([]int32, 21)
	spells[7].params[20] = 500
	table := &Table{Magic: magic, Spells: spells}

	bow := equipmentInstance(data.ComposeItemCode(9, 1, 2, 20),
		[]sim.ItemEffect{{Kind: 12, Operand: 22}}, itemTestCollection{
			{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			{name: "Short Bow", params: []int32{0, 0, 19200}},
		}, table)
	if bow.Price != 69538 {
		t.Fatalf("to-hit bow stored price = %d, want 69538", bow.Price)
	}
	ring := equipmentInstance(data.ComposeItemCode(5, 4, 2, 1),
		[]sim.ItemEffect{{Kind: 23, Operand: 16}}, itemTestCollection{{}, {params: []int32{0, 0, 1600}}}, table)
	if ring.Price != 29810 {
		t.Fatalf("air ring stored price = %d, want 29810", ring.Price)
	}
	staff := storedItemPrice(30000, []sim.ItemEffect{{Kind: 41, Operand: 7 | uint32(90)<<16}}, table)
	if staff != 1002460 {
		t.Fatalf("cast staff stored price = %d, want 1002460", staff)
	}
}

func TestApplyItemEffectsFoldsTheReachableConsumerUnionOnceInStoredOrder(t *testing.T) {
	effects := []sim.ItemEffect{
		{Kind: 2, Operand: 3}, {Kind: 3, Operand: 4}, {Kind: 4, Operand: 5}, {Kind: 5, Operand: 6},
		{Kind: 7, Operand: 7}, {Kind: 8, Operand: 8}, {Kind: 10, Operand: 10}, {Kind: 11, Operand: 11},
		{Kind: 12, Operand: 12}, {Kind: 13, Operand: 13}, {Kind: 14, Operand: 14},
		{Kind: 15, Operand: ^uint32(2)}, {Kind: 16, Operand: 16}, {Kind: 17, Operand: 17},
		{Kind: 18, Operand: 18}, {Kind: 19, Operand: 19},
		{Kind: 21, Operand: 21}, {Kind: 22, Operand: 22}, {Kind: 23, Operand: 23},
		{Kind: 24, Operand: 24}, {Kind: 25, Operand: 25},
		{Kind: 32, Operand: 2}, {Kind: 33, Operand: 3},
		{Kind: 44, Operand: 5 | 7<<8}, {Kind: 44, Operand: 9 | 2<<8},
		{Kind: 49, Operand: 4},
	}
	var equipped [sim.EquipSlots]sim.ItemInstance
	equipped[2] = sim.ItemInstance{Code: 0x0311, Kind: 1, Effects: effects}
	loadout := data.Loadout{}
	ApplyItemEffects(&loadout, equipped, false)
	m := loadout.Mod
	if m.Body != 3 || m.Mind != 4 || m.Reaction != 5 || m.Spirit != 6 ||
		m.HealthMax != 7 || m.HealthRegeneration != 8 || m.ManaMax != 10 || m.ManaRegeneration != 11 ||
		m.ToHit != 12 || m.DamageBase != 17 || m.DamageSpread != 14 || m.Defence != -3 ||
		m.Absorption != 16 || m.Speed != 17 || m.RotationSpeed != 18 || m.Sight != 19 {
		t.Fatalf("scalar effect fold = %+v", m)
	}
	if m.Protection != [5]int32{21, 22, 23, 24, 25} ||
		m.SkillBonus[data.SkillGeneral] != 2 || m.SkillBonus[data.SkillBlade] != 3 {
		t.Fatalf("family effect fold = %+v", m)
	}
	if m.SecondaryDamage != (data.SecondaryDamage{Base: 9, Spread: 2, Selector: 0}) {
		t.Fatalf("ordered elemental replacement = %+v", m.SecondaryDamage)
	}
}

func TestApplyItemEffectsKeepsOneOrderedSecondaryDamageTriple(t *testing.T) {
	fire := sim.ItemEffect{Kind: 44, Operand: 5 | 7<<8}
	water := sim.ItemEffect{Kind: 45, Operand: 9 | 2<<8}
	for _, tc := range []struct {
		name    string
		effects []sim.ItemEffect
		want    data.SecondaryDamage
	}{
		{"fire then water", []sim.ItemEffect{fire, water}, data.SecondaryDamage{Base: 9, Spread: 2, Selector: 1}},
		{"water then fire", []sim.ItemEffect{water, fire}, data.SecondaryDamage{Base: 5, Spread: 7, Selector: 0}},
		{"same kind replaces", []sim.ItemEffect{fire, {Kind: 44, Operand: 11 | 3<<8}}, data.SecondaryDamage{Base: 11, Spread: 3, Selector: 0}},
		{"unrelated effect does not replace", []sim.ItemEffect{water, {Kind: 15, Operand: 4}}, data.SecondaryDamage{Base: 9, Spread: 2, Selector: 1}},
		{"none", []sim.ItemEffect{{Kind: 15, Operand: 4}}, data.SecondaryDamage{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var equipped [sim.EquipSlots]sim.ItemInstance
			equipped[0] = sim.ItemInstance{Code: 0x0201, Effects: tc.effects}
			var loadout data.Loadout
			ApplyItemEffects(&loadout, equipped, true)
			if loadout.Mod.SecondaryDamage != tc.want {
				t.Fatalf("secondary damage = %+v, want %+v", loadout.Mod.SecondaryDamage, tc.want)
			}
			wantSet := tc.name != "none"
			if loadout.Mod.HasSecondaryDamage != wantSet {
				t.Fatalf("secondary damage set = %v, want %v", loadout.Mod.HasSecondaryDamage, wantSet)
			}
		})
	}

	var zeroEquipped [sim.EquipSlots]sim.ItemInstance
	zeroEquipped[0] = sim.ItemInstance{Effects: []sim.ItemEffect{{Kind: 44}}}
	var zeroLoadout data.Loadout
	ApplyItemEffects(&zeroLoadout, zeroEquipped, true)
	if zeroLoadout.Mod.SecondaryDamage != (data.SecondaryDamage{}) || !zeroLoadout.Mod.HasSecondaryDamage {
		t.Fatalf("zero Fire effect = %+v set=%v, want zero triple with explicit presence",
			zeroLoadout.Mod.SecondaryDamage, zeroLoadout.Mod.HasSecondaryDamage)
	}

	var equipped [sim.EquipSlots]sim.ItemInstance
	equipped[0] = sim.ItemInstance{Code: 0x0201, Effects: []sim.ItemEffect{fire}}
	equipped[1] = sim.ItemInstance{Code: 0x0301, Effects: []sim.ItemEffect{water}}
	var loadout data.Loadout
	ApplyItemEffects(&loadout, equipped, true)
	if want := (data.SecondaryDamage{Base: 9, Spread: 2, Selector: 1}); loadout.Mod.SecondaryDamage != want {
		t.Fatalf("later equipment slot secondary damage = %+v, want %+v", loadout.Mod.SecondaryDamage, want)
	}
}

func TestALMLinkBuildsTheCompleteItemAndZeroLinkStaysPlain(t *testing.T) {
	magicItems := make(itemTestCollection, 32)
	magicItems[31] = itemTestRow{name: "Treasure!", params: []int32{10000, 1}}
	table := &Table{MagicItems: magicItems}
	m := &alm.Map{Enchantments: []alm.Enchantment{{
		A: 1, B: 5, C: 12, SpellRaw: uint32(7) | uint32(uint16(15))<<16,
		Elements: []alm.EnchantmentElement{{Kind: 41, Low: 3}, {Kind: 25, Low: 4}},
	}}}
	linked, err := lootItemInstance(alm.LootElement{Code: 0x0e1f, TileMarkerIndex: 1}, m, table)
	if err != nil {
		t.Fatalf("lootItemInstance: %v", err)
	}
	wantEffects := []sim.ItemEffect{
		{Kind: 44, Operand: 5 | 12<<8},
		{Kind: 41, Operand: uint32(7) | uint32(uint16(15))<<16},
		{Kind: 49, Operand: 3},
		{Kind: 25, Operand: 4},
	}
	if linked.Code != 0x0e1f || linked.Price != 10000 || !reflect.DeepEqual(linked.Effects, wantEffects) {
		t.Fatalf("linked item = %+v, want effects %+v and price 10000", linked, wantEffects)
	}
	plain, err := lootItemInstance(alm.LootElement{Code: 0x0e1f}, m, table)
	if err != nil {
		t.Fatalf("zero-link lootItemInstance: %v", err)
	}
	if plain.Price != 10000 || len(plain.Effects) != 0 {
		t.Fatalf("zero-link Treasure = %+v, want value 10000 and no effect", plain)
	}
}

func TestALMLinkKeepsScalarMagnitudesAndRepricesTheCompleteItem(t *testing.T) {
	weapons := make(itemTestCollection, 21)
	weapons[20] = itemTestRow{name: "Short Bow", params: []int32{0, 0, 207}}
	magic := make(itemTestCollection, 22)
	magic[12].params = []int32{7}
	magic[21].params = []int32{7}
	table := &Table{
		Shapes: itemTestScale(2), Materials: itemTestScale(9),
		Weapons: weapons, Magic: magic,
	}
	m := &alm.Map{Enchantments: []alm.Enchantment{{
		Elements: []alm.EnchantmentElement{
			{Kind: 12, Low: 5, High: 0},
			{Kind: 21, Low: 5, High: 0},
		},
	}}}

	got, err := lootItemInstance(alm.LootElement{Code: 0x8134, TileMarkerIndex: 1}, m, table)
	if err != nil {
		t.Fatalf("lootItemInstance: %v", err)
	}
	wantEffects := []sim.ItemEffect{{Kind: 12, Operand: 5}, {Kind: 21, Operand: 5}}
	if got.Code != 0x8134 || got.Kind != 2 || got.Price != 8957 || !reflect.DeepEqual(got.Effects, wantEffects) {
		t.Fatalf("linked Short Bow = %+v, want kind 2, value 8957 and effects %+v", got, wantEffects)
	}

	var equipped [sim.EquipSlots]sim.ItemInstance
	equipped[0] = got
	var loadout data.Loadout
	ApplyItemEffects(&loadout, equipped, true)
	if loadout.Mod.ToHit != 5 || loadout.Mod.Protection[0] != 5 {
		t.Fatalf("linked scalar fold = %+v, want Attack +5 and Fire protection +5", loadout.Mod)
	}
}

func TestItemInstanceFromCodeAssignsConcreteKindsAndMagicItemSignedValue(t *testing.T) {
	magicItems := make(itemTestCollection, 7)
	magicItems[6] = itemTestRow{name: "Potion Health Regeneration", params: []int32{-1}}
	table := &Table{MagicItems: magicItems}
	for _, tc := range []struct {
		code  uint16
		kind  uint8
		price int32
	}{
		{0x0101, 2, 0},
		{0x0201, 1, 0},
		{0x0301, 1, 0},
		{0x0e06, 3, -1},
	} {
		got := ItemInstanceFromCode(tc.code, table)
		if got.Kind != tc.kind || got.Price != tc.price {
			t.Errorf("ItemInstanceFromCode(%#04x) = %+v, want kind %d price %d", tc.code, got, tc.kind, tc.price)
		}
	}
}

func TestReadableBookRepricesFromSpellBookCostAndPreservesAnIncompleteTable(t *testing.T) {
	spells := make(itemTestCollection, 27)
	spells[26].params = make([]int32, 22)
	spells[26].params[21] = 735
	book := sim.ItemInstance{
		Code: 0x0e17, Kind: 5, Price: 200,
		Effects: []sim.ItemEffect{{Kind: 42, Mode: 0, Operand: 26}},
	}
	if got := RepriceItemInstance(book, &Table{Spells: spells}); got != 735 {
		t.Fatalf("readable book value = %d, want installed slot-21 cost 735", got)
	}
	spells[26].params = make([]int32, 21)
	if got := RepriceItemInstance(book, &Table{Spells: spells}); got != 200 {
		t.Fatalf("book against incomplete Spells row = %d, want stored value 200", got)
	}
}

func TestALMLoadedReadableBookUsesSpellBookCostInsteadOfMagicItemsBase(t *testing.T) {
	magicItems := make(itemTestCollection, 24)
	magicItems[23] = itemTestRow{name: "Book Astral", params: []int32{64201}}
	spells := make(itemTestCollection, 27)
	spells[26].params = make([]int32, 22)
	spells[26].params[21] = 735
	table := &Table{MagicItems: magicItems, Spells: spells}
	m := &alm.Map{Enchantments: []alm.Enchantment{{SpellRaw: 26}}}

	got, err := lootItemInstance(alm.LootElement{Code: 0x0e17, TileMarkerIndex: 1}, m, table)
	if err != nil {
		t.Fatalf("lootItemInstance: %v", err)
	}
	wantEffects := []sim.ItemEffect{{Kind: 42, Operand: 26}}
	if got.Code != 0x0e17 || got.Kind != 5 || got.Price != 735 || !reflect.DeepEqual(got.Effects, wantEffects) {
		t.Fatalf("ALM book = %+v, want kind 5, value 735 and effects %+v", got, wantEffects)
	}
}

func TestPartyCanonicalHalvesIndependentlyOverrideLegacyProjections(t *testing.T) {
	carried := sim.ItemInstance{Code: 0x0301, Effects: []sim.ItemEffect{{Kind: 15, Operand: 7}}}
	worn := sim.ItemInstance{Code: 0x0402, Effects: []sim.ItemEffect{{Kind: 20, Operand: 3}}}

	withCanonicalCarry := PartyMember{Carry: &Carry{
		Items:         []uint16{0xffff},
		ItemInstances: []sim.ItemInstance{carried},
		Equipped:      [sim.EquipSlots]uint16{0xeeee},
	}}
	gotCarried := MemberCarriedItems(withCanonicalCarry, nil)
	gotWorn := MemberItemEquipment(withCanonicalCarry, nil)
	if len(gotCarried) != 1 || !sim.ItemEqual(gotCarried[0], carried) || gotWorn[0].Code != 0xeeee {
		t.Fatalf("canonical carried/legacy worn = %+v/%+v", gotCarried, gotWorn[0])
	}

	withCanonicalWorn := PartyMember{Carry: &Carry{
		Items:         []uint16{0x0303},
		Equipped:      [sim.EquipSlots]uint16{0xffff},
		EquippedItems: [sim.EquipSlots]sim.ItemInstance{worn},
	}}
	gotCarried = MemberCarriedItems(withCanonicalWorn, nil)
	gotWorn = MemberItemEquipment(withCanonicalWorn, nil)
	if len(gotCarried) != 1 || gotCarried[0].Code != 0x0303 || !sim.ItemEqual(gotWorn[0], worn) {
		t.Fatalf("legacy carried/canonical worn = %+v/%+v", gotCarried, gotWorn[0])
	}
	if got, ok := EquipmentFromParty(withCanonicalWorn).Code(1); !ok || got != data.ItemCode(worn.Code) {
		t.Fatalf("loadout code = %#x/%v, want canonical %#x", got, ok, worn.Code)
	}

	gotCarried[0].Effects = append(gotCarried[0].Effects, sim.ItemEffect{Kind: 2})
	gotWorn[0].Effects[0].Operand = 99
	if len(withCanonicalWorn.Carry.ItemInstances) != 0 || withCanonicalWorn.Carry.EquippedItems[0].Effects[0].Operand != 3 {
		t.Fatal("canonical party readers aliased the member")
	}
}
