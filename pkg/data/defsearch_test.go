package data

import "testing"

// testEntry and testCollection are a definition collection built in test code.
// Index 0 is the reserved empty entry a one-based collection carries, written
// out here rather than synthesised, so a search that started at 0 would reach an
// entry that exists and is nameless.
type testEntry struct {
	name    string
	params  []int32
	strings []string
}

type testCollection []testEntry

func (c testCollection) Len() int                    { return len(c) }
func (c testCollection) EntryName(i int) string      { return c[i].name }
func (c testCollection) EntryParams(i int) []int32   { return c[i].params }
func (c testCollection) EntryStrings(i int) []string { return c[i].strings }

// row builds a parameter array long enough to carry every key column, with the
// named slots set and every other cell at the sentinel. Rows are built by slot
// number so a test never depends on which column sits where.
func row(slots map[int]int32) []int32 {
	p := sentinelRow(38)
	for s, v := range slots {
		p[s] = v
	}
	return p
}

func unitRow(typeID, face int32) []int32 {
	return row(map[int]int32{unitTypeIDSlot: typeID, unitFaceSlot: face})
}

func humanRow(typeID, serverID int32) []int32 {
	return row(map[int]int32{humanTypeIDSlot: typeID, humanServerIDSlot: serverID})
}

// TestUnitSearchTakesBothColumnsAndTheFirstMatch: the units search is on the
// pair, ascending, first match.
func TestUnitSearchTakesBothColumnsAndTheFirstMatch(t *testing.T) {
	c := testCollection{
		{},                                 // 0: the reserved entry
		{name: "A", params: unitRow(9, 1)}, // 1
		{name: "B", params: unitRow(9, 2)}, // 2: same type, other face
		{name: "C", params: unitRow(9, 2)}, // 3: the duplicate the earlier one beats
		{name: "D", params: unitRow(7, 1)}, // 4
	}
	for _, tc := range []struct {
		typeID, face int32
		want         int
	}{
		{9, 1, 1},
		{9, 2, 2}, // the EARLIER of the two on this key
		{7, 1, 4},
		{9, 3, NotFound}, // the type is there, the face is not
		{5, 1, NotFound},
	} {
		if got := FindUnit(c, tc.typeID, tc.face); got != tc.want {
			t.Errorf("FindUnit(%d, %d) = %d, want %d", tc.typeID, tc.face, got, tc.want)
		}
	}
}

// TestAnEmptyNameIsSkipped: a collection is sized by its count, so the rows that
// were never written are present and nameless. One sitting on the key must be
// walked past, not returned.
func TestAnEmptyNameIsSkipped(t *testing.T) {
	c := testCollection{
		{},
		{name: "", params: unitRow(9, 1)},     // written key, no name
		{name: "real", params: unitRow(9, 1)}, // the entry the search must reach
	}
	if got := FindUnit(c, 9, 1); got != 2 {
		t.Errorf("FindUnit reached %d, want 2 — the empty-named entry before it must be skipped", got)
	}

	h := testCollection{{}, {name: "", params: humanRow(3, 40)}, {name: "real", params: humanRow(3, 40)}}
	if got := FindHumanByType(h, 3); got != 2 {
		t.Errorf("FindHumanByType reached %d, want 2", got)
	}
	if got := FindHumanByServerID(h, 40); got != 2 {
		t.Errorf("FindHumanByServerID reached %d, want 2", got)
	}
}

// TestTheServerIDSearchRunsDownward is the direction made observable: two
// entries on one server id, and the downward search reaches the LATER one where
// an ascending search would reach the earlier.
func TestTheServerIDSearchRunsDownward(t *testing.T) {
	c := testCollection{
		{},
		{name: "early", params: humanRow(1, 500)},
		{name: "middle", params: humanRow(2, 501)},
		{name: "late", params: humanRow(3, 500)},
	}
	if got := FindHumanByServerID(c, 500); got != 3 {
		t.Errorf("FindHumanByServerID = %d, want 3 — the walk runs from the last entry down", got)
	}
	if got := FindHumanByServerID(c, 501); got != 2 {
		t.Errorf("FindHumanByServerID = %d, want 2", got)
	}
	if got := FindHumanByServerID(c, 999); got != NotFound {
		t.Errorf("FindHumanByServerID = %d, want NotFound", got)
	}
	// And the type search over the same collection walks the other way.
	if got := FindHumanByType(c, 3); got != 3 {
		t.Errorf("FindHumanByType = %d, want 3", got)
	}
}

// TestReservedIndexZeroIsNeverReturned: entry 0 exists, and a search that walked
// from it would answer 0, which is the same word as "nothing matched".
func TestReservedIndexZeroIsNeverReturned(t *testing.T) {
	c := testCollection{
		{name: "reserved-but-named", params: unitRow(4, 1)},
		{name: "written", params: unitRow(5, 1)},
	}
	if got := FindUnit(c, 4, 1); got != NotFound {
		t.Errorf("FindUnit reached %d; the walk starts at index 1", got)
	}
	h := testCollection{{name: "reserved-but-named", params: humanRow(4, 60)}}
	if got := FindHumanByType(h, 4); got != NotFound {
		t.Errorf("FindHumanByType reached %d; the walk starts at index 1", got)
	}
	if got := FindHumanByServerID(h, 60); got != NotFound {
		t.Errorf("FindHumanByServerID reached %d; the walk stops at index 1", got)
	}
}

// TestARowTooShortForTheKeyCannotMatch: sixty-odd shipped entries carry no
// parameter array at all, and the searches must walk past them rather than
// panic or match.
func TestARowTooShortForTheKeyCannotMatch(t *testing.T) {
	full := row(map[int]int32{unitTypeIDSlot: 0, unitFaceSlot: 0, humanTypeIDSlot: 0})
	c := testCollection{
		{},
		{name: "no params", params: nil},
		{name: "short", params: []int32{0, 0, 0}},
		{name: "full", params: full},
	}
	if got := FindUnit(c, 0, 0); got != 3 {
		t.Errorf("FindUnit = %d, want 3 — a row shorter than the key column cannot carry the key", got)
	}
	if got := FindHumanByType(c, 0); got != 3 {
		t.Errorf("FindHumanByType = %d, want 3", got)
	}
}

// TestNoCollectionIsNoMatch: a caller holding no table needs no branch of its
// own, and a missing collection is reported as no match, never as an error.
func TestNoCollectionIsNoMatch(t *testing.T) {
	if got := FindUnit(nil, 1, 1); got != NotFound {
		t.Errorf("FindUnit(nil) = %d, want NotFound", got)
	}
	if got := FindHumanByType(nil, 1); got != NotFound {
		t.Errorf("FindHumanByType(nil) = %d, want NotFound", got)
	}
	if got := FindHumanByServerID(nil, 1); got != NotFound {
		t.Errorf("FindHumanByServerID(nil) = %d, want NotFound", got)
	}
	empty := testCollection{{}}
	if got := FindUnit(empty, 1, 1); got != NotFound {
		t.Errorf("FindUnit over a collection of the reserved entry alone = %d, want NotFound", got)
	}
}

// TestTheThreeSearchesReadThreeDifferentColumns pins that the keys are not one
// column read three times: a row keyed only on the units pair is invisible to
// both humans searches, and a row keyed only on the humans columns is invisible
// to the units search.
func TestTheThreeSearchesReadThreeDifferentColumns(t *testing.T) {
	unitOnly := testCollection{{}, {name: "u", params: unitRow(11, 2)}}
	if got := FindHumanByType(unitOnly, 11); got != NotFound {
		t.Errorf("the humans type search matched a units row at %d", got)
	}
	if got := FindHumanByServerID(unitOnly, 11); got != NotFound {
		t.Errorf("the server-id search matched a units row at %d", got)
	}
	humanOnly := testCollection{{}, {name: "h", params: humanRow(11, 2)}}
	if got := FindUnit(humanOnly, 11, 2); got != NotFound {
		t.Errorf("the units search matched a humans row at %d", got)
	}
	if got := FindHumanByType(humanOnly, 11); got != 1 {
		t.Errorf("FindHumanByType = %d, want 1", got)
	}
	if got := FindHumanByServerID(humanOnly, 2); got != 1 {
		t.Errorf("FindHumanByServerID = %d, want 1", got)
	}
}

func TestTheUnitServerIDSearchRunsUpward(t *testing.T) {
	wide := func(id int32) []int32 {
		p := sentinelRow(unitServerIDSlot + 1)
		p[unitServerIDSlot] = id
		return p
	}
	c := testCollection{
		{},
		{name: "", params: wide(7)},
		{name: "early", params: wide(7)},
		{name: "late", params: wide(7)},
		{name: "short", params: []int32{7}},
	}
	if got := FindUnitByServerID(c, 7); got != 2 {
		t.Errorf("FindUnitByServerID = %d, want 2", got)
	}
	if got := FindUnitByServerID(c, 8); got != NotFound {
		t.Errorf("FindUnitByServerID = %d, want NotFound", got)
	}
}
