package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func TestSourceEquipment1110EveryOwnerAndDistinctOperands(t *testing.T) {
	item := ItemInstance{Code: 0x0101, Kind: 2, WeightPresent: true, Weight: 7,
		SourceEquipment: SourceEquipment{Class: SourceWeapon, DefinitionRow: 37, OwnKind: 3,
			Attack:     [24]byte{11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34},
			Defence:    [22]byte{41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62},
			Definition: SourceWeaponDefinition{Present: true, AttackType: 11, Hands: 2}}}
	other := item.Clone()
	other.SourceEquipment.Attack[21]++
	w := mustStockedWorld(t, 21, []Entity{{ID: 1, X: 4, Y: 4}, {ID: 2, X: 4, Y: 4}}, []Stock{{ID: 1, ItemInstances: []ItemInstance{item, item, other}}})
	if len(w.carried[0]) != 2 || w.carried[0][0].Count != 2 {
		t.Fatal("distinct retained operands merged")
	}
	check := func() {
		t.Helper()
		b, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var fresh World
		if err := fresh.UnmarshalBinary(b); err != nil {
			t.Fatal(err)
		}
		if fresh.Hash() != w.Hash() {
			t.Fatal("native source equipment changed")
		}
		w = &fresh
	}
	check()
	if err := w.MoveCarried(1, 2, item.Code, 1); err != nil {
		t.Fatal(err)
	}
	check()
	w.equip(1, 0, 1, 0) // native actor: instance retention, not source arithmetic
	if !reflect.DeepEqual(w.equipment[1][0], item) {
		t.Fatal("equip lost exact source operands")
	}
	check()
	w.dropFromEquipment(1, 1, 4, 4)
	if !reflect.DeepEqual(w.Sacks()[0].ItemInstances[0], item) {
		t.Fatal("ground owner lost source operands")
	}
	check()
	if err := w.TakeSack(2, 4, 4); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.carried[1][0].Instance(), item) {
		t.Fatal("pickup lost source operands")
	}
	// The detached scroll owner shares the same generic instance persistence.
	w.scrollCasts = []ScrollCast{{Caster: 2, Item: item}}
	b := make([]byte, sourceEquipmentLen)
	if _, err := binary.Encode(b, binary.LittleEndian, item.SourceEquipment); err != nil {
		t.Fatal(err)
	}
	if len(b) != 77 || b[0] != 1 || b[1] != 37 || b[2] != 3 || !bytes.Equal(b[3:27], item.SourceEquipment.Attack[:]) || !bytes.Equal(b[27:49], item.SourceEquipment.Defence[:]) || b[49] != 1 || binary.LittleEndian.Uint32(b[50:54]) != 11 || binary.LittleEndian.Uint32(b[54:]) != 2 {
		t.Fatal("literal source item layout")
	}
	seen := false
	w.eachSourceEquipment(func(_ uint32, _ uint16, s *SourceEquipment) {
		if s == &w.scrollCasts[0].Item.SourceEquipment {
			seen = true
		}
	})
	if !seen {
		t.Fatal("detached owner omitted")
	}
}

func TestSourceEquipment1110MalformedPrefixIsAtomic(t *testing.T) {
	item := ItemInstance{Code: 0x0101, SourceEquipment: SourceEquipment{Class: SourceWeapon, DefinitionRow: 40}}
	w := mustStockedWorld(t, 1, []Entity{{ID: 1}}, []Stock{{ID: 1, ItemInstances: []ItemInstance{item}}})
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	end := len(b) - entityIDFloorLen - spellDeliverySpanLen - 65 - 2500 - 4
	span := binary.LittleEndian.Uint32(b[end:])
	if span != 0x80000055 {
		t.Fatalf("literal source-only prefix span %x", span)
	}
	start := end - 85
	for name, alter := range map[string]func([]byte){
		"oversized prefix":    func(b []byte) { binary.LittleEndian.PutUint32(b[start:], 0xffffffff) },
		"ordinal":             func(b []byte) { binary.LittleEndian.PutUint32(b[start+4:], 0xffffffff) },
		"class":               func(b []byte) { b[start+8] = 4 },
		"absence":             func(b []byte) { b[start+8] = 0 },
		"boolean":             func(b []byte) { b[start+8+49] = 2 },
		"Spell boolean":       func(b []byte) { b[start+8+70] = 2 },
		"unsupported boolean": func(b []byte) { b[start+8+76] = 2 },
		"definition residue":  func(b []byte) { b[start+8+50] = 11 },
	} {
		t.Run(name, func(t *testing.T) {
			bad := append([]byte(nil), b...)
			alter(bad)
			before := w.Hash()
			if err := w.UnmarshalBinary(bad); err == nil || w.Hash() != before {
				t.Fatal("malformed source prefix changed receiver", err)
			}
		})
	}
}

func TestSourceEquipment1110ConstructorConflictIsAtomic(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1}}, nil)
	row := ItemWeight{Code: 0x0701, Weight: 2, Constructor: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
	if err := w.DeclareItemWeights([]ItemWeight{row, {Code: row.Code, Weight: row.Weight}}); err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	row.Constructor.Defence[0] = 9
	if err := w.DeclareItemWeights([]ItemWeight{row}); err == nil || w.Hash() != before {
		t.Fatal("conflicting constructor committed")
	}
	if err := w.DeclareItemWeights([]ItemWeight{{Code: row.Code, Weight: row.Weight}}); err != nil || w.Hash() != before {
		t.Fatal("plain re-declaration erased constructor")
	}
}
