package mapload

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestCheatItemKeepsFactoryEffectsAndExactInstalledNames(t *testing.T) {
	table := sourceActorConstructorTable()
	table.MagicItems = consumableRows{{}, {"Potion Custom", "health=+30", 47}}
	table.Names = data.ItemNames{0x0101: "Named Blade", 0x0201: "Named Blade"}
	for _, tc := range []struct {
		name string
		code uint16
		kind uint8
	}{
		{"Common Iron Blade", 0x0101, 2},
		{"Common Iron Shield", 0x0201, 1},
		{"Common Iron Coat", 0x0701, 1},
		{"Potion Custom", 0x0e01, 3},
		{"Named Blade", 0x0101, 2},
	} {
		item, ok := CheatItem(tc.name, table)
		if !ok || item.Code != tc.code || item.Kind != tc.kind {
			t.Errorf("%q: %+v present=%t", tc.name, item, ok)
		}
		if tc.code != 0x0e01 && (item.SourceEquipment.Class == 0 || !item.WeightPresent || item.Weight != 2) {
			t.Errorf("%q: constructor metadata is absent: %+v", tc.name, item)
		}
	}
	item, ok := CheatItem("Common Iron Blade {defence=7,defence=9}", table)
	if !ok || !reflect.DeepEqual(item.Effects, []sim.ItemEffect{{Kind: 15, Operand: 7}, {Kind: 15, Operand: 9}}) {
		t.Fatalf("equipment effects: %+v present=%t", item, ok)
	}
	potion, ok := CheatItem("Potion Custom", table)
	if !ok || potion.Price != 47 || !reflect.DeepEqual(potion.Effects, []sim.ItemEffect{{Kind: 6, Operand: 30}}) {
		t.Fatalf("potion payload: %+v present=%t", potion, ok)
	}
	for _, name := range []string{"", "potion custom", "unknown", "Common Iron Blade {unknown=7}", strings.Repeat("x", 4097)} {
		if _, ok := CheatItem(name, table); ok {
			t.Errorf("unsupported item name accepted: %q", name)
		}
	}
	table.UnitKeys, table.SpellArms, table.FreshPlayers = ServerUnitKeys, SecondGameSpellArms, NoFreshPlayers
	if second, ok := CheatItem("Potion Custom", table); !ok || !reflect.DeepEqual(second, potion) {
		t.Fatalf("second-game item name = %+v, %t; want the shared factory's item", second, ok)
	}
}

func cheatActorParams(width int, values map[int]int32) []int32 {
	params := make([]int32, width)
	for i := range params {
		params[i] = -1
	}
	for i, value := range values {
		params[i] = value
	}
	return params
}

func TestCheatItemMatchesExactInstalledCP866Alias(t *testing.T) {
	table := sourceActorConstructorTable()
	raw := "\xac\xa5\xe7"
	table.Names = data.ItemNames{0x0101: raw, 0x0201: raw}
	item, ok := CheatItem(raw, table)
	if !ok || item.Code != 0x0101 || item.SourceEquipment.Class == 0 || !item.WeightPresent || item.Weight != 2 {
		t.Fatalf("raw alias did not construct the first exact installed name: %+v found=%t", item, ok)
	}
	for _, name := range []string{"меч", "\x8c\xa5\xe7"} {
		if _, ok := CheatItem(name, table); ok {
			t.Fatalf("factory changed alphabet or case for %q", name)
		}
	}
	if table.Names[0x0101] != raw || table.Names[0x0201] != raw {
		t.Fatal("lookup changed the installed byte alphabet")
	}
}

func cheatActorTable() *Table {
	table := sourceActorConstructorTable()
	table.Units = ghostResistanceCollection{
		{},
		{name: "First Creature", params: cheatActorParams(38, map[int]int32{0: 23, 4: 40, 29: 64, 30: 2})},
		{name: "Named Creature", params: cheatActorParams(38, map[int]int32{0: 23, 4: 80, 29: 64, 30: 2}), strings: []string{"Common Iron Blade"}},
	}
	table.Humans = ghostResistanceCollection{
		{},
		{name: "First Person", params: cheatActorParams(26, map[int]int32{0: 10, 16: 7, 17: 1, 18: 0, 24: 100})},
		{name: "Named Person", params: cheatActorParams(26, map[int]int32{0: 30, 16: 7, 17: 2, 18: 0, 24: 100}), strings: []string{"Common Iron Blade", "Common Iron Shield"}},
	}
	return table
}

func TestCheatActorUsesItsExactRowAndCompleteConstructor(t *testing.T) {
	table := cheatActorTable()
	for _, tc := range []struct {
		name     string
		hero     bool
		humanoid bool
		typeID   int32
		body     uint16
		face     uint8
	}{
		{"Named Creature", false, false, 64, 23, 2},
		{"Named Creature", true, false, 64, 23, 2},
		{"Named Person", false, true, 7, 30, 2},
		{"Named Person", true, true, 7, 30, 2},
	} {
		t.Run(tc.name+fmtBool(tc.hero), func(t *testing.T) {
			e, pack, worn, err := CheatActor(tc.name, tc.hero, table, DifficultyNormal)
			if err != nil {
				t.Fatal(err)
			}
			if e.Humanoid != tc.humanoid || e.TypeID != tc.typeID || e.SourceBinding.TokenRow != 2 || e.SourceBinding.Face != tc.face || e.ActorLoad.Source.Stats[0] != tc.body {
				t.Fatalf("exact definition metadata: %+v", e)
			}
			if !e.ActorLoad.Present || !e.ActorLoad.ContainerPresent || e.ActorLoad.Source.Class == 0 || e.NativeBasis.HasValues() {
				t.Fatal("constructor has competing or missing source basis", e)
			}
			if err := e.SourceBinding.Validate(e); err != nil {
				t.Fatal(err)
			}
			if worn[0].Code != 0x0101 || worn[0].SourceEquipment.Class == 0 || !worn[0].WeightPresent {
				t.Fatal("ordinary worn weapon is missing", worn)
			}
			if e.Load != int32(e.ActorLoad.OwnWeight)+e.ActorLoad.Accumulator/2 || e.ActorLoad.Source.Stats[6] != uint16(e.Load) {
				t.Fatalf("constructor load does not carry its stock: %d %+v %v", e.Load, e.ActorLoad, pack)
			}
			if !tc.humanoid && (e.HP != 80 || e.MaxHP != 80) {
				t.Fatalf("duplicate type selected another row: HP=%d/%d", e.HP, e.MaxHP)
			}
		})
	}
	for _, tc := range []struct {
		name string
		hero bool
		diff Difficulty
	}{
		{"", false, DifficultyNormal},
		{"named person", true, DifficultyNormal},
		{"unknown", false, DifficultyNormal},
		{"Named Creature", false, 0},
	} {
		if _, _, _, err := CheatActor(tc.name, tc.hero, table, tc.diff); err == nil {
			t.Errorf("unsupported actor accepted: %+v", tc)
		}
	}
	// A second-game placement resolves its row by the server id column. Both
	// persons carry server id 100; the named row is the one built. A creature
	// row too short to hold a server id cannot be keyed.
	table.UnitKeys, table.SpellArms, table.FreshPlayers = ServerUnitKeys, SecondGameSpellArms, NoFreshPlayers
	e, _, _, err := CheatActor("Named Person", true, table, DifficultyNormal)
	if err != nil || !e.Humanoid || e.SourceBinding.TokenRow != 2 || e.ActorLoad.Source.Stats[0] != 30 {
		t.Fatalf("second-game person = row %d body %d humanoid %t, %v; want the named row", e.SourceBinding.TokenRow, e.ActorLoad.Source.Stats[0], e.Humanoid, err)
	}
	if _, _, _, err := CheatActor("Named Creature", false, table, DifficultyNormal); err == nil {
		t.Fatal("second-game creature without a server id accepted")
	}
}

func TestCheatActorPrefersUnitsForSharedNamesAndHeroFlag(t *testing.T) {
	table := cheatActorTable()
	units := table.Units.(ghostResistanceCollection)
	humans := table.Humans.(ghostResistanceCollection)
	units[2].name, humans[2].name = "Shared Name", "Shared Name"
	for _, hero := range []bool{false, true} {
		e, _, _, err := CheatActor("Shared Name", hero, table, DifficultyNormal)
		if err != nil {
			t.Fatal(err)
		}
		if e.Humanoid || e.TypeID != 64 || e.ActorLoad.Source.Class != 1 || e.SourceBinding.ActorClass() != 1 || e.SourceBinding.TokenRow != 2 || e.HP != 80 || e.ActorLoad.Source.Stats[0] != 23 {
			t.Fatalf("hero=%t selected Human or changed the Units constructor: %+v", hero, e)
		}
	}
}

func TestCheatActorFallsBackToHumansWhenUnitTypeIsZero(t *testing.T) {
	table := cheatActorTable()
	units := table.Units.(ghostResistanceCollection)
	humans := table.Humans.(ghostResistanceCollection)
	units[2].name, humans[2].name = "Shared Name", "Shared Name"
	units[2].params[29] = 0
	for _, hero := range []bool{false, true} {
		e, _, _, err := CheatActor("Shared Name", hero, table, DifficultyNormal)
		if err != nil {
			t.Fatal(err)
		}
		if !e.Humanoid || e.TypeID != 7 || e.ActorLoad.Source.Class != 2 || e.SourceBinding.ActorClass() != 2 || e.SourceBinding.TokenRow != 2 || e.ActorLoad.Source.Stats[0] != 30 {
			t.Fatalf("hero=%t did not take the type-zero Human fallback: %+v", hero, e)
		}
	}
}

func fmtBool(value bool) string {
	if value {
		return "/hero"
	}
	return "/ordinary"
}

func TestCheatActorCanSummonAndColdLoadWithoutNativeBacking(t *testing.T) {
	for _, tc := range []struct {
		name string
		hero bool
	}{
		{"Named Creature", false},
		{"Named Creature", true},
		{"Named Person", false},
		{"Named Person", true},
	} {
		t.Run(tc.name+fmtBool(tc.hero), func(t *testing.T) {
			e, pack, worn, err := CheatActor(tc.name, tc.hero, cheatActorTable(), DifficultyNormal)
			if err != nil {
				t.Fatal(err)
			}
			e.Owner = sim.SelfSlot
			w, err := sim.NewWorld(Seed, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			BindSourceDerive(w)
			id, err := w.CheatSummon(e, pack, worn, 20, 20)
			if err != nil {
				t.Fatal(err)
			}
			live, found := w.Entity(id)
			if !found || live.ActorLoad.Source.Class == 0 {
				t.Fatal("summon did not retain its source constructor")
			}
			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold sim.World
			if err := cold.UnmarshalBinary(form); err != nil {
				t.Fatal(err)
			}
			restored, found := cold.Entity(id)
			if !found || restored.ActorLoad.Source != live.ActorLoad.Source || restored.SourceBinding != live.SourceBinding || restored.NativeClass != (sim.NativeClass{}) || restored.NativeTraining != (sim.NativeTraining{}) || cold.Hash() != w.Hash() {
				t.Fatal("cold load changed source constructor or retained competing native backing")
			}
			held, _ := cold.CarriedStacks(id)
			equipped, _ := cold.EquippedItems(id)
			if !reflect.DeepEqual(held, pack) || !reflect.DeepEqual(equipped, worn) {
				t.Fatal("cold load changed constructor inventory or equipment")
			}
		})
	}
}

func TestCheatCreatureKeepsOrdinaryDifficulty(t *testing.T) {
	for _, tc := range []struct {
		diff Difficulty
		hp   int32
	}{
		{DifficultyEasy, 52}, {DifficultyNormal, 80}, {DifficultyHard, 120},
	} {
		e, _, _, err := CheatActor("Named Creature", false, cheatActorTable(), tc.diff)
		if err != nil || e.HP != tc.hp || e.MaxHP != tc.hp || e.ActorLoad.Source.Stats[8] != uint16(tc.hp) {
			t.Errorf("difficulty %d: HP=%d/%d err=%v, want %d", tc.diff, e.HP, e.MaxHP, err, tc.hp)
		}
	}
}
