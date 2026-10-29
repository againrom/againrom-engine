package mapload_test

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

func TestZeroOwnedStockKeepsItsDecodedFormAndBaseHash(t *testing.T) {
	table := wearTable([]string{"Sword"}, wearWeapons(), wearShields(), wearArmors())
	code := uint16(0x0101)
	want := mapload.ItemInstanceFromCode(code, table)
	if reflect.DeepEqual(want, mapload.SourceConstructedItem(want, table)) {
		t.Fatal("fixture must distinguish decoded and source forms")
	}
	m := humanMap()
	m.Units[0].UnitID = 7
	m.FormatVersion, m.Meta.Word2C = 990, 1
	m.LootSection = alm.LootSection{Body: llJoin(llHead(2, 7, 0, 0, 0),
		llElement(uint32(code), 0, 0), llElement(uint32(code), 0, 0))}
	for _, difficulty := range []mapload.Difficulty{mapload.DifficultyEasy, mapload.DifficultyNormal, mapload.DifficultyHard} {
		w, err := mapload.FromALMWith(m, table, difficulty)
		if err != nil {
			t.Fatal(err)
		}
		items, _ := w.CarriedItems(0)
		if len(items) != 2 || !reflect.DeepEqual(items[0], want) || !reflect.DeepEqual(items[1], want) {
			t.Fatalf("zero element changed decoded form: %+v", items)
		}
		table.CarryStockOnly = true
		base, err := mapload.FromALMWith(m, table, difficulty)
		table.CarryStockOnly = false
		if err != nil || base.Hash() != w.Hash() {
			t.Fatal("zero elements changed the base constructor state", difficulty, err)
		}
	}
}

func TestOwnedStockUsesEquipmentDispatchAfterTheRow(t *testing.T) {
	table := wearTable([]string{"Sword", "Buckler"}, wearWeapons(), wearShields(), wearArmors())
	pike, err := data.ResolveWeapon("Pike", table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatal(err)
	}
	sword, _ := data.ResolveWeapon("Sword", table.Shapes, table.Materials, table.Weapons)
	shield, _ := data.ResolveShield("Buckler", table.Shapes, table.Materials, table.Shields)
	magic := make(defCollection, 9)
	magic[8] = defEntry{name: "PotionHealth", params: []int32{1}, strings: []string{"health=5"}}
	table.MagicItems = magic
	for _, dispatch := range []uint16{1, 2, 65535} {
		m := humanMap()
		if dispatch == 2 {
			m.Units[0].HasCurrentHP, m.Units[0].CurrentHP = true, 3
		}
		m.Units[0].UnitID = 7
		m.FormatVersion, m.Meta.Word2C = 990, 1
		m.LootSection = alm.LootSection{Body: llJoin(llHead(3, 7, 0, 0, 999),
			llElement(uint32(pike.Code), dispatch, 0), llElement(0x0e08, dispatch, 0), llElement(0x0e08, 0, 0))}
		if dispatch == 2 {
			m.LootSection.Body = llJoin(llHead(3, 7, 0, 0, 999),
				llElement(0x0e08, dispatch, 0), llElement(uint32(pike.Code), dispatch, 0), llElement(0x0e08, 0, 0))
		}
		w, err := mapload.FromALMWith(m, table, mapload.DifficultyNormal)
		if err != nil {
			t.Fatal(err)
		}
		worn, _ := w.Equipped(0)
		pack, _ := w.Carried(0)
		if worn[0] != uint16(pike.Code) || worn[1] != 0 || len(pack) != 3 || pack[0] != uint16(sword.Code) || pack[1] != uint16(shield.Code) || pack[2] != 0x0e08 {
			t.Fatalf("dispatch=%d: worn=%x pack=%x", dispatch, worn, pack)
		}
		if w.Tick() != 0 {
			t.Fatal("stock dispatch advanced the clock")
		}
		if dispatch == 2 && w.Entities()[0].HP != 8 {
			t.Fatal("stock potion did not apply through Item use", w.Entities()[0].HP)
		}
		table.CarryStockOnly = true
		legacy, err := mapload.FromALMWith(m, table, mapload.DifficultyNormal)
		table.CarryStockOnly = false
		if err != nil {
			t.Fatal(err)
		}
		before, _ := legacy.Equipped(0)
		carried, _ := legacy.Carried(0)
		pikeIndex := 0
		if dispatch == 2 {
			pikeIndex = 2
		}
		if before[0] != uint16(sword.Code) || before[1] != uint16(shield.Code) || len(carried) != 3 || carried[pikeIndex] != uint16(pike.Code) {
			t.Fatal("carry-only profile changed", before, carried)
		}
	}
}
