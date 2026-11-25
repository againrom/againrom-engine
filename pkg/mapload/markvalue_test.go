package mapload_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// The second game's terrain build leaves 1 for the flag, tile class 8 and
// water marks, 5 for a type-3 byte stored after them and 0x1f on the ring;
// by the move-kind masks 1 closes kind 1, 5 kinds 1 and 2, 0x1f all three.
func TestMarkValuesCloseMoveKinds(t *testing.T) {
	kinds := map[uint8]uint8{}
	for _, r := range mapload.SecondMarks.PlaneValueRows {
		if r.Word == "" {
			t.Fatalf("value %#x has no word", r.Value)
		}
		kinds[r.Value] = r.Kinds
	}
	if want := map[uint8]uint8{0: 0, 1: 0b001, 5: 0b011, 0x1f: 0b111}; len(kinds) != len(want) || kinds[0] != 0 || kinds[1] != want[1] || kinds[5] != want[5] || kinds[0x1f] != want[0x1f] {
		t.Fatalf("value kinds %v, want %v", kinds, want)
	}
	m := &alm.Map{Width: 24, Height: 24, Tiles: make([]uint16, 24*24), Overlay: make([]uint8, 24*24)}
	at := func(x, y int) int { return y*24 + x }
	m.Tiles[at(10, 10)] = alm.TileImpassable
	m.Tiles[at(11, 10)] = 0x1c1 // group 7, mid 0, low 1: weight 3
	m.Tiles[at(12, 10)] = 0x1f0 // group 7, mid 3, low 0: weight 5
	m.Tiles[at(13, 10)] = 0x1c0 // group 7, mid 0, low 0: weight 2, class 6
	m.Tiles[at(14, 10)] = 0x181 // group 6
	m.Tiles[at(15, 10)] = 0x211 // tile bits 0x200
	m.Tiles[at(10, 11)] = alm.TileImpassable
	m.Overlay[at(10, 11)] = 40
	m.Overlay[at(2, 2)] = 40
	arms := mapload.SecondMarks.GroundArms(m, nil)
	for _, c := range []struct {
		x, y  int
		value uint8
	}{{10, 10, 1}, {11, 10, 1}, {12, 10, 1}, {13, 10, 0}, {14, 10, 0}, {15, 10, 1}, {10, 11, 5}, {2, 2, 0x1f}, {12, 12, 0}} {
		if got := mapload.SecondMarks.MarkValue(arms[at(c.x, c.y)]); got != c.value {
			t.Errorf("cell %d,%d arms %#x value %#x, want %#x", c.x, c.y, arms[at(c.x, c.y)], got, c.value)
		}
	}
	if arms[at(11, 10)]&mapload.GroundClass8 == 0 || arms[at(13, 10)]&mapload.GroundClass8 != 0 {
		t.Fatalf("class-8 marks %#x, %#x", arms[at(11, 10)], arms[at(13, 10)])
	}
}
