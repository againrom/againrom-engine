package alm

import (
	"reflect"
	"testing"
)

func TestCellBindingsSelectAndNarrowTheCellArm(t *testing.T) {
	r := Enchantment{X: 0x102, Y: 0x203, A: 3, B: 99, C: 88, SpellRaw: 0x9876010d,
		Elements: []EnchantmentElement{{Kind: 0x104, Low: 0x205, High: 91}, {Kind: 0x306, Low: 0x407, High: 92}}}
	m := &Map{Enchantments: []Enchantment{r}}
	want := []CellBinding{{X: 2, Y: 3, Spell: 13, Power: 0x76, SourceX: 4, SourceY: 5, LastX: 6, LastY: 7}}
	got, err := m.CellBindings()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("CellBindings = %+v, %v; want %+v", got, err, want)
	}
	for _, a := range []uint16{4, 5, 0xffff} {
		r.A = a
		m.Enchantments = []Enchantment{r}
		if got, err := m.CellBindings(); err != nil || len(got) != 0 {
			t.Fatalf("building/non-cell arm A=%d became %+v: %v", a, got, err)
		}
	}
	r.X, r.Y, r.A, r.Elements = 0, 0, 0, nil
	m.Enchantments = []Enchantment{r}
	if got, err := m.CellBindings(); err != nil || len(got) != 0 {
		t.Fatalf("zero coordinate item arm became %+v: %v", got, err)
	}
	r.Y = 1 // OR, not AND: either nonzero coordinate selects the cell arm.
	m.Enchantments = []Enchantment{r}
	if _, err := m.CellBindings(); err == nil {
		t.Fatal("missing source/last tail accepted")
	}
	m = &Map{TileMarkers: TileMarkers{Count: 0xffffffff, Body: []byte{1}}}
	m.present[9] = true
	if err := m.decodeEnchantments(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.CellBindings(); err == nil {
		t.Fatal("malformed huge count silently disabled cell bindings")
	}
}
