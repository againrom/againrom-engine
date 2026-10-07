package alm

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestType9DecodesEveryHeaderFieldAndOrderedElement(t *testing.T) {
	appendRecord := func(out []byte, tag, x, y uint32, a, b, c uint16, spell uint32,
		elements ...EnchantmentElement) []byte {
		out = binary.LittleEndian.AppendUint32(out, tag)
		out = binary.LittleEndian.AppendUint32(out, x)
		out = binary.LittleEndian.AppendUint32(out, y)
		out = binary.LittleEndian.AppendUint16(out, a)
		out = binary.LittleEndian.AppendUint16(out, b)
		out = binary.LittleEndian.AppendUint16(out, c)
		out = binary.LittleEndian.AppendUint32(out, spell)
		out = binary.LittleEndian.AppendUint32(out, uint32(len(elements)))
		for _, e := range elements {
			out = binary.LittleEndian.AppendUint16(out, e.Kind)
			out = binary.LittleEndian.AppendUint16(out, e.Low)
			out = binary.LittleEndian.AppendUint16(out, e.High)
		}
		return out
	}
	var body []byte
	body = appendRecord(body, 0x11223344, 0, 0, 1, 5, 12, 0x000f0007,
		EnchantmentElement{Kind: 41, Low: 3, High: 8},
		EnchantmentElement{Kind: 25, Low: 4, High: 4})
	body = appendRecord(body, 0xaabbccdd, 9, 10, 0, 0, 0, 0)
	m := &Map{TileMarkers: TileMarkers{Count: 2, Body: body}}
	m.present[9] = true
	if err := m.decodeEnchantments(); err != nil {
		t.Fatalf("decodeEnchantments: %v", err)
	}
	want := []Enchantment{
		{Tag: 0x11223344, A: 1, B: 5, C: 12, SpellRaw: 0x000f0007,
			Elements: []EnchantmentElement{{Kind: 41, Low: 3, High: 8}, {Kind: 25, Low: 4, High: 4}}},
		{Tag: 0xaabbccdd, X: 9, Y: 10, Elements: []EnchantmentElement{}},
	}
	if !reflect.DeepEqual(m.Enchantments, want) {
		t.Fatalf("Enchantments = %+v, want %+v", m.Enchantments, want)
	}
}

func TestMalformedHistoricalType9BodyRemainsRawAndUndecoded(t *testing.T) {
	m := &Map{TileMarkers: TileMarkers{Count: 0x55667788, Body: []byte{1, 2, 3}}}
	m.present[9] = true
	if err := m.decodeEnchantments(); err != nil {
		t.Fatalf("decodeEnchantments changed the historical acceptance surface: %v", err)
	}
	if m.Enchantments != nil || !reflect.DeepEqual(m.TileMarkers.Body, []byte{1, 2, 3}) {
		t.Fatalf("malformed raw section was rewritten: raw=%v decoded=%+v", m.TileMarkers.Body, m.Enchantments)
	}
}

func TestType9AllocationCapacityIsBoundedByTheBody(t *testing.T) {
	for _, tc := range []struct {
		count   uint32
		bodyLen int
		want    int
	}{
		{count: 0x55667788, bodyLen: 3, want: 0},
		{count: 0x55667788, bodyLen: 2 * enchantmentHeadSize, want: 2},
		{count: 1, bodyLen: 2 * enchantmentHeadSize, want: 1},
	} {
		if got := enchantmentCapacity(tc.count, tc.bodyLen); got != tc.want {
			t.Errorf("enchantmentCapacity(%d, %d) = %d, want %d", tc.count, tc.bodyLen, got, tc.want)
		}
	}
}
