package mapload_test

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestSourceActorPersonDecodesWireAppearance(t *testing.T) {
	table := joinTable()
	rows := table.Humans.(defCollection)
	for len(rows) < 6 {
		rows = append(rows, rows[1])
	}
	table.Humans = rows
	for _, tc := range []struct {
		name      string
		typ, face uint8
		dir       string
		index     int
		mage      bool
	}{
		{"fighter", 8, 24, "mfighter", 24, false},
		{"archer", 14, 0x80 | 10, "ffighter", 10, false},
		{"male mage", 24, 3, "mmage", 3, true},
		{"female mage", 24, 0x80 | 3, "fmage", 3, true},
		{"noncomposing human", 26, 131, "mfighter", 131, false},
		{"hero fighter", 0x21, 4, "mfighter", 4, false},
		{"hero female fighter", 0x22, 4, "ffighter", 4, false},
		{"hero mage", 0x23, 5, "mmage", 5, true},
		{"hero female mage", 0x24, 5, "fmage", 5, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The DAT row deliberately disagrees: saved type and face are live
			// identity; its separate Gender column cannot decode an actor byte.
			e := sim.Entity{ID: 42, SourceBinding: sim.SourceBinding{Class: 2, TokenRow: 1, TypeID: uint16(tc.typ), Face: tc.face}}
			person, err := mapload.SourceActorPerson(e, "Enemy", table)
			if err != nil {
				t.Fatal(err)
			}
			if person.FigureDir != tc.dir || person.FigureFace != tc.index || person.Mage != tc.mage {
				t.Fatalf("wire type=%x face=%x: got %s/%d mage=%v, want %s/%d mage=%v", tc.typ, tc.face, person.FigureDir, person.FigureFace, person.Mage, tc.dir, tc.index, tc.mage)
			}
		})
	}
}
