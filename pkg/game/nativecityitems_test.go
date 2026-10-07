package game

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestNativeCityItemsWriteOrderedDistinctEffectReferences(t *testing.T) {
	member := mapload.PartyMember{CarriedItems: []sim.ItemInstance{
		{Code: 0x0e14, Kind: 5, Price: 60000, Effects: []sim.ItemEffect{{Kind: 42, Operand: 8}}},
		{Code: 0x0e01, Kind: 3, Effects: []sim.ItemEffect{
			{Kind: 8, Mode: 1, Operand: 0x03c00064}, {Kind: 42, Operand: 26}, {Kind: 8, Mode: 1, Operand: 0x03c00064},
		}},
	}}
	var unit sav.CityUnitData
	seq := 0
	objects, err := nativeCityAttachItems(nil, &unit, member, nil, 123, &seq)
	if err != nil {
		t.Fatal(err)
	}
	want := [][][]byte{
		{{42, 0, 8, 0, 0, 0, 0}},
		{{8, 1, 100, 0, 192, 3, 0}, {42, 0, 26, 0, 0, 0, 0}, {8, 1, 100, 0, 192, 3, 0}},
	}
	if len(unit.Container) != len(want) {
		t.Fatalf("container refs = %v", unit.Container)
	}
	seen := map[uint32]bool{}
	for i, ref := range unit.Container {
		item := objects[ref-1].Item
		if len(item.Effects) != len(want[i]) {
			t.Fatalf("item %d lost effects: refs=%v", i, item.Effects)
		}
		for j, effectRef := range item.Effects {
			obj := objects[effectRef-1]
			if obj.Class != "Effect" || obj.Effect == nil || !bytes.Equal(obj.Effect.Fields, want[i][j]) {
				t.Fatalf("item %d effect %d lost ordered fields: %+v", i, j, obj)
			}
			key := binary.LittleEndian.Uint32(obj.Effect.Token[29:33])
			if key == 0 || seen[key] || obj.Effect.Token[16] != 0 {
				t.Fatalf("effect identity/state = %d/%d", key, obj.Effect.Token[16])
			}
			seen[key] = true
		}
	}
}

type nativeItemScale []float64

func (s nativeItemScale) Len() int             { return len(s) }
func (s nativeItemScale) EntryName(int) string { return "scale" }
func (s nativeItemScale) EntryDoubles(i int) []float64 {
	d := make([]float64, 9)
	d[8] = s[i]
	return d
}

func nativeItemTestTable() *mapload.Table {
	return &mapload.Table{Shapes: nativeItemScale{1, 1, 1, 1.5, 1},
		Materials: nativeItemScale{1, 1, 1, 1, 1, 1, 1, 1, 1, 2.5}, Spells: nativeItemSpells{}}
}

type nativeItemSpells struct{}

func (nativeItemSpells) Len() int                  { return 3 }
func (nativeItemSpells) EntryName(int) string      { return "spell" }
func (nativeItemSpells) EntryStrings(int) []string { return nil }
func (nativeItemSpells) EntryParams(i int) []int32 {
	p := make([]int32, 21)
	p[1], p[6], p[18] = int32(5+i), int32(7+i), 1
	return p
}

func nativeItemTestWeapon() sim.ItemInstance {
	return sim.ItemInstance{Code: 0x9167, Kind: 2, Weight: -9, WeightPresent: true,
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 7, OwnKind: 4,
			Attack: [24]byte{11, 0, 14: 3, 15: 2}, Defence: [22]byte{6}}}
}

func TestNativeCityItemKnownFieldsUseConstructorOperands(t *testing.T) {
	for _, class := range []uint8{sim.SourceWeapon, sim.SourceArmor, sim.SourceShield} {
		item := nativeItemTestWeapon()
		item.SourceEquipment.Class = class
		if class != sim.SourceWeapon {
			item.SourceEquipment.Attack = [24]byte{}
		}
		if class == sim.SourceShield {
			item.SourceEquipment.OwnKind = 0
		}
		obj, err := nativeCityItemObject(item, 123, 456, nativeItemTestTable())
		if err != nil {
			t.Fatal(err)
		}
		// Literal 0x9167 selects shape3/material9. 1.5*2.5 truncates to3;
		// weight -9 is independently stored as the signed low word.
		want := []byte{0x67, 0x91, 1, 0, 2, 3, 9, 3, 0, 0xf7, 0xff, 0}
		if !bytes.Equal(obj.Item.Fields, want) {
			t.Fatalf("class%d fields=%x want=%x", class, obj.Item.Fields, want)
		}
		if got := binary.LittleEndian.Uint16(obj.Item.Token[17:19]); got != 0x21 {
			t.Fatalf("Item marker changed to %#x", got)
		}
	}
	for _, tc := range []struct {
		product float64
		want    uint16
	}{{-1.75, 0xffff}, {65536.75, 0}} {
		table := nativeItemTestTable()
		table.Shapes = nativeItemScale{1, 1, 1, tc.product}
		table.Materials = nativeItemScale{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
		v, err := nativeCityItemConstructionFor(0x9167, table)
		if err != nil || v.F48 != tc.want {
			t.Fatalf("product%g=%+v err=%v, want F48=%d", tc.product, v, err, tc.want)
		}
	}
}

func TestNativeCityBaseItemWeightDefaultsOnlyWhenAbsent(t *testing.T) {
	member := mapload.PartyMember{CarriedItems: []sim.ItemInstance{
		{Code: 0x0e14, Kind: 5},
		{Code: 0x0e01, Kind: 3, WeightPresent: true},
		{Code: 0x0e02, Kind: 3, WeightPresent: true, Weight: -7},
	}}
	var unit sav.CityUnitData
	seq := 0
	objects, err := nativeCityAttachItems(nil, &unit, member, nil, 123, &seq)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []int16{1, 0, -7} {
		fields := objects[unit.Container[i]-1].Item.Fields
		if got := int16(binary.LittleEndian.Uint16(fields[9:11])); got != want {
			t.Fatalf("base item%d weight=%d want=%d", i, got, want)
		}
	}
	if int32(unit.ContainerTails[1]) != -6 {
		t.Fatalf("container accumulator=%d, want -6", int32(unit.ContainerTails[1]))
	}
}

func TestNativeCityWeaponSpellOwnsDistinctCurrentFields(t *testing.T) {
	item := nativeItemTestWeapon()
	item.Effects = []sim.ItemEffect{{Kind: 42, Operand: 6}, {Kind: 41, Operand: 0x12340002}, {Kind: 41, Operand: 1}}
	member := mapload.PartyMember{KnownSpells: 1 << 2}
	member.WornItems[0] = item
	var unit sav.CityUnitData
	seq := 0
	objects, err := nativeCityAttachItems(nil, &unit, member, nativeItemTestTable(), 123, &seq)
	if err != nil {
		t.Fatal(err)
	}
	weapon := objects[unit.Reference74-1].Item
	if weapon.WeaponExtra == 0 || weapon.WeaponExtra == unit.Spells[1] {
		t.Fatalf("weapon/book Spell refs=%d/%v", weapon.WeaponExtra, unit.Spells)
	}
	fields := objects[weapon.WeaponExtra-1].Spell.Fields
	if !bytes.Equal(fields[:5], []byte{2, 9, 1, 7, 0}) {
		t.Fatalf("first kind41 Spell fields=%x", fields)
	}
	seen := map[uint32]bool{}
	for _, obj := range objects {
		var identity uint32
		switch {
		case obj.Item != nil:
			identity = binary.LittleEndian.Uint32(obj.Item.Token[29:33])
		case obj.Effect != nil:
			identity = binary.LittleEndian.Uint32(obj.Effect.Token[29:33])
		case obj.Spell != nil:
			identity = binary.LittleEndian.Uint32(obj.Spell.Fields[5:9])
		}
		if identity == 0 || seen[identity] {
			t.Fatalf("non-distinct object identity=%d", identity)
		}
		seen[identity] = true
	}
	if len(weapon.Effects) != 3 || objects[weapon.Effects[1]-1].Effect.Fields[0] != 41 ||
		binary.LittleEndian.Uint32(objects[weapon.Effects[1]-1].Effect.Fields[2:6]) != 0x12340002 {
		t.Fatalf("weapon Effect order/operand changed: %v", weapon.Effects)
	}
	// Current saved Spell bytes win even when its Effect names another row.
	member.KnownSpells = 0
	member.WornItems[0].SourceEquipment.Spell = sim.SourceItemSpell{Present: true, ID: 25, Range: 33, Defensive: 7, ManaCost: 0xfffd}
	unit, seq = sav.CityUnitData{}, 0
	objects, err = nativeCityAttachItems(nil, &unit, member, nativeItemTestTable(), 123, &seq)
	if err != nil {
		t.Fatal(err)
	}
	weapon = objects[unit.Reference74-1].Item
	if got := objects[weapon.WeaponExtra-1].Spell.Fields[:5]; !bytes.Equal(got, []byte{25, 33, 7, 0xfd, 0xff}) {
		t.Fatalf("current owned Spell fields=%x", got)
	}
	// Moving the fresh weapon into the pack does not run its equip producer.
	member = mapload.PartyMember{CarriedItems: []sim.ItemInstance{item}}
	unit, seq = sav.CityUnitData{}, 0
	objects, err = nativeCityAttachItems(nil, &unit, member, nativeItemTestTable(), 123, &seq)
	if err != nil || objects[unit.Container[0]-1].Item.WeaponExtra != 0 {
		t.Fatalf("unequipped weapon gained a Spell: err=%v", err)
	}
}

func TestNativeCityItemsRefuseMissingAndRetainedOperands(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *mapload.Table
		edit  func(*sim.ItemInstance)
		want  string
	}{
		{"shape operands", nil, func(*sim.ItemInstance) {}, "F48 constructor operands"},
		{"nonfinite", &mapload.Table{Shapes: nativeItemScale{1, 1, 1, math.Inf(1)}, Materials: nativeItemScale{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}}, func(*sim.ItemInstance) {}, "F48 constructor product"},
		{"first spell", nativeItemTestTable(), func(i *sim.ItemInstance) {
			i.Effects = []sim.ItemEffect{{Kind: 41, Operand: 29}, {Kind: 41, Operand: 1}}
		}, "first kind41 id 29"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := nativeItemTestWeapon()
			tc.edit(&item)
			member := mapload.PartyMember{}
			member.WornItems[0] = item
			var unit sav.CityUnitData
			seq := 0
			_, err := nativeCityAttachItems(nil, &unit, member, tc.table, 123, &seq)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want %q", err, tc.want)
			}
		})
	}
	// ObjectID belongs to the current in-memory graph. A newly constructed SAV
	// receives its own file-local identity instead of refusing the checkpoint.
	identityItem := nativeItemTestWeapon()
	identityItem.ObjectID = 77
	identityMember := mapload.PartyMember{}
	identityMember.WornItems[0] = identityItem
	identityUnit, identitySeq := sav.CityUnitData{}, 0
	identityObjects, err := nativeCityAttachItems(nil, &identityUnit, identityMember, nativeItemTestTable(), 123, &identitySeq)
	if err != nil {
		t.Fatalf("current object identity refused: %v", err)
	}
	gotIdentity := binary.LittleEndian.Uint32(identityObjects[identityUnit.Reference74-1].Item.Token[29:33])
	if gotIdentity == 0 || gotIdentity == uint32(identityItem.ObjectID) {
		t.Fatalf("constructed SAV identity = %d, want a fresh nonzero file-local identity", gotIdentity)
	}
	member := mapload.PartyMember{Carry: &mapload.Carry{OrderedStacks: []sim.ItemStack{sim.StackItem(sim.PlainItem(0x0e01), 3)}}}
	var unit sav.CityUnitData
	seq := 0
	if _, err := nativeCityAttachItems(nil, &unit, member, nil, 123, &seq); err == nil || !strings.Contains(err.Error(), "container state") {
		t.Fatalf("stack flattened without a precise refusal: %v", err)
	}
	member.Carry = &mapload.Carry{ItemInstances: []sim.ItemInstance{{Code: 0x0e1c, Kind: 5}}}
	unit, seq = sav.CityUnitData{}, 0
	objects, err := nativeCityAttachItems(nil, &unit, member, nil, 123, &seq)
	if err != nil || len(objects) != 1 || binary.LittleEndian.Uint16(objects[0].Item.Fields[9:11]) != 1 || unit.ContainerTails[1] != 1 {
		t.Fatalf("unbound native Carry singleton constructor: %v %+v", err, unit.ContainerTails)
	}
}

// A flat pack writes one record per cell: two equal Potions are one record
// counted two (ITEM-MERGE-129, ITEM-SAVE-014).
func TestNativeCityItemsWriteEqualFlatUnitsAsOneRecord(t *testing.T) {
	potion := sim.ItemInstance{Code: 0x0e01, Kind: 3, Price: 71, Weight: 2, WeightPresent: true}
	other := potion
	other.Code = 0x0e02
	member := mapload.PartyMember{CarriedItems: []sim.ItemInstance{potion, other, potion}}
	var unit sav.CityUnitData
	seq := 0
	objects, err := nativeCityAttachItems(nil, &unit, member, nil, 123, &seq)
	if err != nil {
		t.Fatal(err)
	}
	if len(unit.Container) != 2 {
		t.Fatalf("container records %v, want one per cell", unit.Container)
	}
	for i, want := range [][2]uint16{{0x0e01, 2}, {0x0e02, 1}} {
		fields := objects[unit.Container[i]-1].Item.Fields
		if got := [2]uint16{binary.LittleEndian.Uint16(fields[:2]), binary.LittleEndian.Uint16(fields[2:4])}; got != want {
			t.Fatalf("record %d code/count %x, want %x", i, got, want)
		}
	}
	if int32(unit.ContainerTails[1]) != 6 {
		t.Fatalf("container accumulator %d, want 6", int32(unit.ContainerTails[1]))
	}
}
