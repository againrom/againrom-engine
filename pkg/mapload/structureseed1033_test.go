package mapload_test

// The Field42 SEED (1033 B3, adversarial pass 1): the initial value a world's
// structures carry, and where it comes from.
//
// `ALM-CLS-053` gives the type-4 spawn writing the Buildings table's healthMax
// into obj+0x44/+0x42, and +0x42 is the word check opcode 21 reads and instant
// opcode 26 writes. `SAV-BLDG-037` measures that pair in the owner's own
// original saves as equal non-zero values, at the healthMax figures
// `DAT-BLD-005` gives for those building kinds. `DAT-SCHEMA-004` puts healthMax
// at Buildings column 4, which is parameter position 3.
//
// The story first shipped this field at zero, on the reading that no claim gave
// it. That reading was wrong, and the consequence was visible on shipped
// content: mission 101's twelve placements all carry healthMax 1 against twelve
// script nodes testing the field below 1, so every one of them was true at tick
// 0 and the mission's counter objective completed with no player action.
//
// Fixtures reuse bldRow, bldRowOf, bldTable, structMap and place from
// structures_test.go. Nothing here reads a map file or an install.

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// slotBldHealthMax is the parameter position `ALM-CLS-053` names in the
// BUILDINGS table, spelt here from the claim and NOT imported from the
// implementation, on structures_test.go's own rule for the other four
// positions: a test that took the production constant would agree with
// whatever it became.
//
// `DAT-SCHEMA-004` puts healthMax at Buildings column 4, and a column is
// 1-based where a parameter position is 0-based. The neighbours agree:
// structures_test.go spells columns 5 and 6 as positions 4 and 5.
//
// The UNITS table has its own healthMax at a different position, spelt
// slotHealthMax in spawn_test.go. The two are not interchangeable.
const slotBldHealthMax = 3

// bldRowHealth is bldRow with an explicit value at slotBldHealthMax. bldRow leaves
// every position it does not name at -1, which is the shape of an unwritten
// cell.
func bldRowHealth(w, h int, blocking, attach uint32, health int32) []int32 {
	p := bldRow(w, h, blocking, attach)
	p[slotBldHealthMax] = health
	return p
}

// TestTheStructureFieldIsSeededFromTheDefinitionTable is the seed itself: one
// Structure per placement, in file order, ids ascending from zero, each
// carrying its own entry's healthMax.
//
// The three values are distinct and none is a Go zero value, so a build that
// seeded a neighbouring position, a constant, or nothing at all fails here
// rather than agreeing by coincidence.
func TestTheStructureFieldIsSeededFromTheDefinitionTable(t *testing.T) {
	t.Parallel()

	tbl := bldTable(map[int][]int32{
		1: bldRowHealth(1, 1, 0, 1, 300),
		2: bldRowHealth(1, 1, 0, 1, 30000),
		3: bldRowHealth(1, 1, 0, 1, 7),
	})
	m := structMap(
		place(1, 10, 10, nil),
		place(2, 10, 12, nil),
		place(3, 10, 14, nil),
	)

	got := mapload.Structures(m, tbl)
	if len(got) != 3 {
		t.Fatalf("built %d structures, want one per placement, 3", len(got))
	}
	for i, want := range []uint16{300, 30000, 7} {
		if got[i].ID != sim.StructureID(i) {
			t.Errorf("structure %d has id %d, want the record's own index %d", i, got[i].ID, i)
		}
		if got[i].Field42 != want {
			t.Errorf("structure %d carries Field42 %d, want its entry's own healthMax %d",
				i, got[i].Field42, want)
		}
		if got[i].MaxHealth != want || got[i].Col != 10 || got[i].Row != int32(10+2*i) ||
			got[i].Width != 1 || got[i].Height != 1 || got[i].Attach != 1 {
			t.Errorf("structure %d target state = %+v, want max %d at (10,%d), 1x1 attach 1",
				i, got[i], want, 10+2*i)
		}
	}
}

// TestTheStructureFieldSeedReadsPositionThreeAndNotItsNeighbours pins the
// position against the two beside it. scanRange sits at position 2 and the
// blocking set at position 4, and a seed reading either would still produce a
// plausible non-zero value on a shipped row.
func TestTheStructureFieldSeedReadsPositionThreeAndNotItsNeighbours(t *testing.T) {
	t.Parallel()

	// Every position carries a distinct value, so the field can only match one.
	row := []int32{11, 22, 33, 44, 55, 66}
	got := mapload.Structures(structMap(place(1, 10, 10, nil)),
		bldTable(map[int][]int32{1: row}))
	if len(got) != 1 {
		t.Fatalf("built %d structures, want 1", len(got))
	}
	if got[0].Field42 != uint16(row[slotBldHealthMax]) {
		t.Errorf("Field42 is %d, want position %d's own %d; the row is %v",
			got[0].Field42, slotBldHealthMax, row[slotBldHealthMax], row)
	}
}

// TestTheStructureFieldSeedTruncatesToSixteenBits is the width of the
// destination `ALM-CLS-053` names. The store reproduces that width rather than
// widening it here.
//
// No shipped entry reaches the truncation: healthMax runs 0..30000 over all 66
// entries on both preserved roots, so this fixture is its only witness. A
// negative cell takes the same path -- -1 is what an unwritten column reads as,
// and it truncates to 0xffff, which check opcode 21 then sign-extends back to
// -1.
func TestTheStructureFieldSeedTruncatesToSixteenBits(t *testing.T) {
	t.Parallel()

	tbl := bldTable(map[int][]int32{
		1: bldRowHealth(1, 1, 0, 1, 0x10001),
		2: bldRowHealth(1, 1, 0, 1, -1),
	})
	got := mapload.Structures(structMap(place(1, 10, 10, nil), place(2, 10, 12, nil)), tbl)
	if len(got) != 2 {
		t.Fatalf("built %d structures, want 2", len(got))
	}
	if got[0].Field42 != 1 {
		t.Errorf("0x10001 seeded as %d, want its low 16 bits, 1", got[0].Field42)
	}
	if got[1].Field42 != 0xffff {
		t.Errorf("-1 seeded as %#04x, want 0xffff", got[1].Field42)
	}
}

// TestTheStructureFieldSeedLeavesZeroWhereTheTableSaysNothing is the set of
// misses. A placement whose class key names no entry, one whose entry is too
// narrow to carry the position, one naming the reserved entry 0, and a table
// with no Buildings collection at all: each keeps the zero value, and each
// still yields a Structure.
//
// The list is one entry per placed record whatever the table says, on the
// entity list's own precedent. None of these is reached on shipped content --
// `ALM-CLS-053` lands all 3141 shipped kinds on a named entry, and no shipped
// entry is short.
func TestTheStructureFieldSeedLeavesZeroWhereTheTableSaysNothing(t *testing.T) {
	t.Parallel()

	tbl := bldTable(map[int][]int32{
		1: bldRowHealth(1, 1, 0, 1, 700),
		2: bldRowOf(3, 1, 1, 0, 1), // three parameters: too narrow for position 3
	})
	for _, tc := range []struct {
		name string
		obj  alm.Object
	}{
		{"a class key naming no entry", place(200, 10, 10, nil)},
		{"an entry too narrow to carry the position", place(2, 10, 10, nil)},
		{"the reserved entry 0", place(0, 10, 10, nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := mapload.Structures(structMap(tc.obj), tbl)
			if len(got) != 1 {
				t.Fatalf("built %d structures, want 1 whatever the table says", len(got))
			}
			if got[0].Field42 != 0 {
				t.Errorf("Field42 is %d, want 0", got[0].Field42)
			}
		})
	}

	for name, none := range map[string]*mapload.Table{"nil table": nil, "no Buildings": {}} {
		got := mapload.Structures(structMap(place(1, 10, 10, nil)), none)
		if len(got) != 1 {
			t.Fatalf("%s: built %d structures, want 1", name, len(got))
		}
		if got[0].Field42 != 0 {
			t.Errorf("%s: Field42 is %d, want 0", name, got[0].Field42)
		}
	}
}

// TestTheStructureSeedIsPureAndTotal is the property the footprint pass's own
// resolution already has and this seed shares: no shape of map or table yields
// an error, the table is never written through, and the same pair always yields
// the same list.
func TestTheStructureSeedIsPureAndTotal(t *testing.T) {
	t.Parallel()

	tbl := bldTable(map[int][]int32{1: bldRowHealth(2, 1, 0b01, 0b11, 1234)})
	before := append([]int32(nil), tbl.Buildings.EntryParams(1)...)

	m := structMap(place(1, 10, 10, nil), place(200, 10, 12, nil))
	first := mapload.Structures(m, tbl)
	second := mapload.Structures(m, tbl)

	if len(first) != len(second) {
		t.Fatalf("two calls built %d and %d structures", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("structure %d differs between two calls: %+v against %+v", i, first[i], second[i])
		}
	}
	after := tbl.Buildings.EntryParams(1)
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("the seed wrote through the collection at position %d: %d became %d",
				i, before[i], after[i])
		}
	}

	if got := mapload.Structures(nil, tbl); got != nil {
		t.Errorf("a nil map built %d structures, want none", len(got))
	}
	if got := mapload.Structures(structMap(), tbl); got != nil {
		t.Errorf("a map with no placements built %d structures, want none", len(got))
	}
}
