package sim

import (
	"bytes"
	"testing"
)

// sfBounds is the small extent every fixture here is built against, large
// enough to hold every cell a case names and small enough to write a form
// out by hand.
var sfBounds = Bounds{Width: 10, Height: 10}

func mustLootWorld(t *testing.T, seed uint64, b Bounds, sacks []Sack) *World {
	t.Helper()
	w, err := NewLootWorld(seed, b, ModeCanonical, Terrain{}, nil, nil, Relations{}, sacks)
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	return w
}

// TestTwoEntriesOnOneCellMergeIntoOneSack is AC-4: two entries naming one
// cell produce one sack whose gold is the sum and whose item list is both
// lists in argument order.
func TestTwoEntriesOnOneCellMergeIntoOneSack(t *testing.T) {
	w := mustLootWorld(t, 1, sfBounds, []Sack{
		{X: 3, Y: 4, Gold: 100, Items: []uint16{0x0101, 0x0102}},
		{X: 3, Y: 4, Gold: 50, Items: []uint16{0x0203}},
	})
	got := w.Sacks()
	if len(got) != 1 {
		t.Fatalf("two entries on one cell produced %d sack(s), want 1: %+v", len(got), got)
	}
	want := Sack{X: 3, Y: 4, Gold: 150, Items: []uint16{0x0101, 0x0102, 0x0203}}
	if got[0].X != want.X || got[0].Y != want.Y || got[0].Gold != want.Gold ||
		!equalCodes(got[0].Items, want.Items) {
		t.Errorf("merged sack is %+v, want %+v", got[0], want)
	}
}

func TestThreeEntriesOnOneCellJoinInArgumentOrder(t *testing.T) {
	w := mustLootWorld(t, 1, sfBounds, []Sack{
		{X: 1, Y: 1, Items: []uint16{1}},
		{X: 1, Y: 1, Items: []uint16{2}},
		{X: 1, Y: 1, Items: []uint16{3}},
	})
	got := w.Sacks()
	if len(got) != 1 {
		t.Fatalf("got %d sack(s), want 1", len(got))
	}
	if !equalCodes(got[0].Items, []uint16{1, 2, 3}) {
		t.Errorf("items are %v, want [1 2 3] — argument order", got[0].Items)
	}
}

// TestThePurseWrapsAtThirtyTwoBits is D-8: two purses summing past the
// field's own width wrap, exactly as the original's addition does, rather
// than saturating or refusing.
func TestThePurseWrapsAtThirtyTwoBits(t *testing.T) {
	w := mustLootWorld(t, 1, sfBounds, []Sack{
		{X: 0, Y: 0, Gold: 0xffffffff},
		{X: 0, Y: 0, Gold: 2},
	})
	got := w.Sacks()
	if len(got) != 1 || got[0].Gold != 1 {
		t.Errorf("summed gold is %+v, want a single sack carrying 1 (wrapped)", got)
	}
}

func TestSacksAreOrderedAscendingByYThenX(t *testing.T) {
	w := mustLootWorld(t, 1, sfBounds, []Sack{
		{X: 5, Y: 2},
		{X: 1, Y: 5},
		{X: 9, Y: 2},
		{X: 0, Y: 0},
		{X: 2, Y: 2},
	})
	got := w.Sacks()
	want := [][2]int32{{0, 0}, {2, 2}, {5, 2}, {9, 2}, {1, 5}}
	if len(got) != len(want) {
		t.Fatalf("got %d sack(s), want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].X != want[i][0] || got[i].Y != want[i][1] {
			t.Errorf("sack %d is (%d,%d), want (%d,%d)", i, got[i].X, got[i].Y, want[i][0], want[i][1])
		}
	}
}

func TestAnOutOfBoundsSackIsRefusedNotClampedNotDropped(t *testing.T) {
	for _, tc := range []struct {
		name string
		s    Sack
	}{
		{"negative X", Sack{X: -1, Y: 0}},
		{"negative Y", Sack{X: 0, Y: -1}},
		{"X at the width", Sack{X: sfBounds.Width, Y: 0}},
		{"Y at the height", Sack{X: 0, Y: sfBounds.Height}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := NewLootWorld(1, sfBounds, ModeCanonical, Terrain{}, nil, nil, Relations{}, []Sack{tc.s})
			if err == nil {
				t.Fatalf("an out-of-bounds sack was accepted")
			}
			if w != nil {
				t.Errorf("a refused construction returned a world alongside its error")
			}
		})
	}

	// The control: the same cells at the bounds' own top-left legal corner
	// and the widest legal one are accepted, so the four cases above are
	// refused for standing outside the bounds and not because this fixture
	// cannot be built at all.
	if _, err := NewLootWorld(1, sfBounds, ModeCanonical, Terrain{}, nil, nil, Relations{},
		[]Sack{{X: 0, Y: 0}, {X: sfBounds.Width - 1, Y: sfBounds.Height - 1}}); err != nil {
		t.Errorf("the well-formed control was refused: %v", err)
	}
}

func TestTheFourOlderConstructorsBuildWorldsWithNoSacks(t *testing.T) {
	terrain, err := NewTerrainWorld(1, sfBounds, ModeCanonical, Terrain{}, nil, nil)
	if err != nil {
		t.Fatalf("NewTerrainWorld: %v", err)
	}
	related, err := NewRelatedWorld(1, sfBounds, ModeCanonical, Terrain{}, nil, nil, Relations{})
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	scripted, err := NewScriptedWorld(1, sfBounds, ModeCanonical, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	worlds := []*World{mustWorld(t, 1, sfBounds, nil), terrain, related, scripted}
	for i, w := range worlds {
		if got := w.Sacks(); len(got) != 0 {
			t.Errorf("constructor %d built a world holding %d sack(s), want none", i, len(got))
		}
	}
}

func TestSacksHandsBackACopy(t *testing.T) {
	w := mustLootWorld(t, 1, sfBounds, []Sack{{X: 1, Y: 1, Gold: 5, Items: []uint16{9}}})

	first := w.Sacks()
	first[0].Gold = 999
	first[0].Items[0] = 111
	first[0].X = 7

	second := w.Sacks()
	if second[0].Gold != 5 || second[0].Items[0] != 9 || second[0].X != 1 {
		t.Errorf("mutating one call's result reached the world: %+v", second[0])
	}

	// And the constructor's own argument is copied too: a caller's slice
	// stays the caller's to reuse or mutate.
	given := []Sack{{X: 2, Y: 2, Items: []uint16{1, 2}}}
	w2 := mustLootWorld(t, 1, sfBounds, given)
	given[0].Items[0] = 0xffff
	given[0].X = 8
	if got := w2.Sacks(); got[0].X != 2 || got[0].Items[0] != 1 {
		t.Errorf("mutating the constructor's argument reached the world: %+v", got[0])
	}
}

func TestAnItemCodeCrossesTheFormUnaltered(t *testing.T) {
	codes := []uint16{0x0000, 0xffff, 0x0f00, 0x00ff, 0x1234, 0x8000}
	w := mustLootWorld(t, 1, sfBounds, []Sack{{X: 4, Y: 4, Items: codes}})

	form := mustMarshal(t, w)
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	got := back.Sacks()
	if len(got) != 1 || !equalCodes(got[0].Items, codes) {
		t.Errorf("item codes crossed the form as %v, want %v", got, codes)
	}
}

// TestSackAtFindsExactlyTheOccupiedCells exercises the binary search over a
// scrambled-then-sorted list, including both ends of it and a cell no sack
// names.
func TestSackAtFindsExactlyTheOccupiedCells(t *testing.T) {
	w := mustLootWorld(t, 1, sfBounds, []Sack{
		{X: 5, Y: 5}, {X: 0, Y: 0}, {X: 9, Y: 9}, {X: 3, Y: 5}, {X: 7, Y: 5},
	})
	for _, tc := range []struct {
		x, y int32
		want bool
	}{
		{0, 0, true}, {9, 9, true}, {5, 5, true}, {3, 5, true}, {7, 5, true},
		{1, 1, false}, {5, 3, false}, {4, 5, false}, {9, 0, false}, {0, 9, false},
	} {
		if got := w.sackAt(tc.x, tc.y); got != tc.want {
			t.Errorf("sackAt(%d,%d) = %v, want %v", tc.x, tc.y, got, tc.want)
		}
	}
}

// TestSackAtOverAnEmptyListAnswersFalseEverywhere: no sack, no occupied
// cell, and the search must not panic over a zero-length list.
func TestSackAtOverAnEmptyListAnswersFalseEverywhere(t *testing.T) {
	w := mustWorld(t, 1, sfBounds, nil)
	for _, c := range [][2]int32{{0, 0}, {5, 5}, {-1, -1}} {
		if w.sackAt(c[0], c[1]) {
			t.Errorf("sackAt(%d,%d) over an empty list answered true", c[0], c[1])
		}
	}
}

// ---------------------------------------------------------------- AC-5

// TestWorldsWithSacksInAnyOrderAreOneWorld is AC-5: a world built with sacks
// named in a scrambled order equals — and hashes equal to — the same world
// built with them in ascending order, and a world differing only in one
// sack's gold hashes differently.
func TestWorldsWithSacksInAnyOrderAreOneWorld(t *testing.T) {
	ascending := []Sack{{X: 0, Y: 0, Gold: 1}, {X: 5, Y: 0, Gold: 2}, {X: 0, Y: 5, Gold: 3}}
	scrambled := []Sack{ascending[2], ascending[0], ascending[1]}

	a := mustLootWorld(t, 42, sfBounds, ascending)
	b := mustLootWorld(t, 42, sfBounds, scrambled)

	fa, fb := mustMarshal(t, a), mustMarshal(t, b)
	if !bytes.Equal(fa, fb) {
		t.Errorf("scrambled and ascending sacks marshal to\n % x\n % x", fb, fa)
	}
	if a.Hash() != b.Hash() {
		t.Errorf("scrambled and ascending sacks hash %#016x and %#016x", b.Hash(), a.Hash())
	}

	moved := []Sack{ascending[0], ascending[1], {X: 0, Y: 5, Gold: 4}}
	c := mustLootWorld(t, 42, sfBounds, moved)
	if c.Hash() == a.Hash() {
		t.Errorf("two worlds differing only in one sack's gold hash alike (%#016x)", a.Hash())
	}
}

// TestAWorldWithSacksMarshalsAndUnmarshalsToAnEqualWorld is AC-6's positive
// half.
func TestAWorldWithSacksMarshalsAndUnmarshalsToAnEqualWorld(t *testing.T) {
	w := mustLootWorld(t, 7, sfBounds, []Sack{
		{X: 1, Y: 1, Gold: 40, Items: []uint16{0x0101}},
		{X: 1, Y: 2, Gold: 0, Items: nil},
		{X: 8, Y: 8, Gold: 500000, Items: []uint16{0x0202, 0x0f10, 0xffff}},
	})
	form := mustMarshal(t, w)

	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round trip hashes %#016x, want %#016x", back.Hash(), w.Hash())
	}
	got, want := back.Sacks(), w.Sacks()
	if len(got) != len(want) {
		t.Fatalf("the round trip holds %d sack(s), want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].X != want[i].X || got[i].Y != want[i].Y || got[i].Gold != want[i].Gold ||
			!equalCodes(got[i].Items, want[i].Items) {
			t.Errorf("sack %d is %+v, want %+v", i, got[i], want[i])
		}
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round two): %v", err)
	}
	if !bytes.Equal(again, form) {
		t.Errorf("re-marshalling gave\n % x\nwant\n % x", again, form)
	}
}

func sfValidWithSacks(t *testing.T) []byte {
	t.Helper()
	w := mustLootWorld(t, 3, sfBounds, []Sack{
		{X: 2, Y: 2, Gold: 10},
		{X: 4, Y: 4, Gold: 20, Items: []uint16{7}},
	})
	return mustMarshal(t, w)
}

// sfSackSectionAt is where sfValidWithSacks' sack section begins: past the
// header, three 100-cell planes, no entity record, no route and the empty
// group section this fixture's ownerless world writes — every span written
// out from the contract rather than read back off the encoder, on this
// package's own rule for a byte-form fixture.
const sfSackSectionAt = headerLen + 3*100 + groupCountLen

func TestUnmarshalRefusesTheFourSackShapesFR15(t *testing.T) {
	valid := sfValidWithSacks(t)
	if got := valid[sfSackSectionAt : sfSackSectionAt+4]; !bytes.Equal(got, []byte{2, 0, 0, 0}) {
		t.Fatalf("the fixture's sack count is % x at offset %d, want a count of 2 — "+
			"sfSackSectionAt no longer lands on the section", got, sfSackSectionAt)
	}

	cases := []struct {
		name string
		data []byte
	}{
		// Cut six bytes into the section — the count and two bytes of the
		// first sack's head — and a relation-shaped tail spliced back on so
		// the form's OVERALL length still passes the header's own check and
		// the cut is isolated to decodeSacks' own truncation branch, the one
		// the per-sack bounds check above this table guards (a sack whose
		// own item list overruns what an EARLIER sack's items already left
		// could otherwise run this reader past the buffer instead of
		// refusing it).
		{"a truncated section",
			append(append([]byte(nil), valid[:sfSackSectionAt+6]...), make([]byte, relationLen)...)},
		{"a sack out of bounds", withU32(valid, sfSackSectionAt+4, uint32(sfBounds.Width))},
		// Sack 0's cell moved onto sack 1's own (4,4): both in bounds, so
		// this isolates the duplicate-cell shape from the bounds refusal
		// above.
		{"two sacks on one cell", withU32(withU32(valid, sfSackSectionAt+4+4, 4), sfSackSectionAt+4, 4)},
		// Sack 0's Y moved past sack 1's (4 -> 6), so the pair descends
		// while both stay in bounds and on distinct cells — isolating
		// "not ascending" from the duplicate-cell shape above.
		{"a list that is not ascending", withU32(valid, sfSackSectionAt+8, 6)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var w World
			if err := w.UnmarshalBinary(tc.data); err == nil {
				t.Fatalf("accepted % x", tc.data)
			}
			if len(w.Entities()) != 0 || len(w.Sacks()) != 0 {
				t.Errorf("a refused decode left a world behind: %+v", w)
			}
		})
	}

	// The one that must NOT be refused, so the table above is not passing by
	// refusing everything.
	var w World
	if err := w.UnmarshalBinary(valid); err != nil {
		t.Errorf("the unspoiled form was refused: %v", err)
	}
}

func equalCodes(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
