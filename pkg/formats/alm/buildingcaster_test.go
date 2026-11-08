package alm

import (
	"encoding/binary"
	"testing"
)

// A mode-4 record an element of a loot record links to (the sack binding) is
// not a caster; the link is 1-based and 0 names no record. The base does not
// compile this test.
func TestBuildingCastersSackBindingGate(t *testing.T) {
	body := binary.LittleEndian.AppendUint32(nil, 1) // element count
	body = append(body, make([]byte, 12)...)         // owner, cell
	body = binary.LittleEndian.AppendUint32(body, 0x0205)
	body = binary.LittleEndian.AppendUint16(body, 0)
	body = binary.LittleEndian.AppendUint32(body, 2) // element links record index 1
	m := &Map{Enchantments: []Enchantment{
		{Tag: 1, X: 5, Y: 5, A: 4, B: 12, SpellRaw: 1},
		{Tag: 1, X: 6, Y: 5, A: 4, B: 13, SpellRaw: 1},
	}, LootSection: LootSection{Body: body}}
	m.Meta.Word2C = 1
	got := m.BuildingCasters()
	if len(got) != 1 || got[0].Key != 12 {
		t.Fatalf("got %+v, want only record 0: record 1 is bound to a sack", got)
	}
	m.LootSection.Body, m.Meta.Word2C = nil, 0
	if got := m.BuildingCasters(); len(got) != 2 {
		t.Fatalf("got %+v, want both records with no sack binding", got)
	}
}

func TestBuildingCastersGates(t *testing.T) {
	spell := uint32(10) | 255<<16
	m := &Map{Enchantments: []Enchantment{
		{Tag: 7, X: 20, Y: 30, A: 4, B: 12, SpellRaw: spell}, // built
		{Tag: 7, X: 0, Y: 0, A: 5, B: 13, SpellRaw: spell},   // cell 0,0
		{Tag: 7, X: 20, Y: 30, A: 1, B: 14, SpellRaw: spell}, // no mask 4
		{Tag: 9, X: 5, Y: 0, A: 5, B: 15, C: 1, SpellRaw: 2}, // mask 4 in mode 5
	}}
	got := m.BuildingCasters()
	if len(got) != 2 {
		t.Fatalf("got %+v, want records 0 and 3", got)
	}
	want := BuildingCasterRecord{Key: 12, X: 20, Y: 30, Spell: 10, Power: 255, Reach: 7}
	if got[0] != want {
		t.Fatalf("record 0 = %+v, want %+v", got[0], want)
	}
	if got[1].Key != 15|1<<16 || got[1].Reach != 9 {
		t.Fatalf("record 3 = %+v", got[1])
	}
}
