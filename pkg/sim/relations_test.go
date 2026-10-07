package sim

import (
	"bytes"
	"testing"
)

// relFull is a matrix with one recognisable byte in every cell the index law can
// reach, built by a rule that is NOT the index law: the value at [i][j] is
// derived from i and j, so a transposed read produces a different byte at almost
// every cell rather than a plausible one.
func relFull() []byte {
	cells := make([]byte, relationLen)
	for i := 1; i < relationSlots; i++ {
		for j := 1; j < relationSlots; j++ {
			cells[i*relationSlots+j] = byte(3*i + 5*j)
		}
	}
	return cells
}

// relWorld is a one-entity world over the relation cells, through the root
// constructor.
func relWorld(t *testing.T, cells []byte) *World {
	t.Helper()
	rel, err := NewRelations(cells)
	if err != nil {
		t.Fatalf("NewRelations: %v", err)
	}
	w, err := NewRelatedWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 1, Y: 1, Owner: 1}}, nil, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

func TestARelationRefusesEveryLengthButItsOwn(t *testing.T) {
	t.Parallel()

	if _, err := NewRelations(nil); err != nil {
		t.Errorf("an empty relation was refused: %v", err)
	}
	if _, err := NewRelations(make([]byte, relationLen)); err != nil {
		t.Errorf("a full relation was refused: %v", err)
	}
	for _, n := range []int{1, relationSlots, relationLen - 1, relationLen + 1, 2 * relationLen} {
		if _, err := NewRelations(make([]byte, n)); err == nil {
			t.Errorf("a relation of %d byte(s) was accepted", n)
		}
	}
}

func TestTheRelationIsDirectionalAndReadsOnlyBitZero(t *testing.T) {
	t.Parallel()

	var r Relations
	r.Set(2, 3, 1)
	if !r.Hostile(2, 3) {
		t.Error("slot 2 is not hostile to slot 3 after bit 0 was written")
	}
	if r.Hostile(3, 2) {
		t.Error("writing [2][3] made slot 3 hostile to slot 2")
	}
	for v := 0; v < 256; v++ {
		r.Set(4, 5, byte(v))
		if got, want := r.Hostile(4, 5), v&1 == 1; got != want {
			t.Errorf("a relation byte of %#02x reads hostile=%v, want %v", v, got, want)
		}
		if got := r.Byte(4, 5); got != byte(v) {
			t.Errorf("a relation byte of %#02x reads back as %#02x — every bit is carried", v, got)
		}
	}
}

func TestASlotTheMatrixDoesNotHoldIsHostileToNothing(t *testing.T) {
	t.Parallel()

	r, err := NewRelations(relFull())
	if err != nil {
		t.Fatalf("NewRelations: %v", err)
	}
	before := append([]byte(nil), r.cells...)
	for _, slot := range []uint32{0, relationSlots, relationSlots + 1, 1 << 31} {
		r.Set(slot, 3, 0xff)
		r.Set(3, slot, 0xff)
		if r.Hostile(slot, 3) || r.Hostile(3, slot) {
			t.Errorf("slot %d took part in a relation", slot)
		}
		if r.Byte(slot, 3) != 0 || r.Byte(3, slot) != 0 {
			t.Errorf("slot %d read a nonzero byte", slot)
		}
	}
	if !bytes.Equal(before, r.cells) {
		t.Error("a write naming a slot the matrix does not hold changed the matrix")
	}
}

func TestTurnHostileGatesOnTheLowTwoBits(t *testing.T) {
	t.Parallel()

	const high = 0xfc // everything above bit 1: never read, never written
	cases := []struct {
		name      string
		low       byte
		flippable bool
	}{
		{"clear and clear: flippable", 0b00, true},
		{"hostile already", 0b01, false},
		{"locked", 0b10, false},
		{"hostile and locked", 0b11, false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var r Relations
			r.Set(2, 3, high|c.low)
			r.turnHostile(2, 3)

			want := high | c.low // declined: byte-for-byte unchanged
			if c.flippable {
				want = high | c.low | relationHostile // flippable: gains bit 0 and nothing else
			}
			if got := r.Byte(2, 3); got != want {
				t.Errorf("cell started %#02x, turnHostile left it %#02x, want %#02x",
					high|c.low, got, want)
			}
		})
	}
}

func TestTurnHostileTwiceEqualsOnceOnTheSamePair(t *testing.T) {
	t.Parallel()

	var r Relations
	r.turnHostile(4, 7)
	once := append([]byte(nil), r.cells...)

	r.turnHostile(4, 7)
	if !bytes.Equal(r.cells, once) {
		t.Error("a second turnHostile on an already-hostile pair changed the matrix")
	}
	r.turnHostile(4, 7)
	r.turnHostile(4, 7)
	if !bytes.Equal(r.cells, once) {
		t.Error("repeated turnHostile calls on an already-hostile pair changed the matrix")
	}
}

func TestTurnHostileTouchesOnlyItsOwnCell(t *testing.T) {
	t.Parallel()

	r, err := NewRelations(relFull())
	if err != nil {
		t.Fatalf("NewRelations: %v", err)
	}
	// [2][6] = 3*2 + 5*6 = 36 = 0x24: low two bits clear, so this cell flips.
	const from, to = 2, 6
	if r.cells[from*relationSlots+to]&0b11 != 0 {
		t.Fatalf("test setup: cell [%d][%d] is not flippable", from, to)
	}
	before := append([]byte(nil), r.cells...)

	r.turnHostile(from, to)

	target := from*relationSlots + to
	if r.cells[target] != before[target]|relationHostile {
		t.Errorf("cell [%d][%d] did not gain bit 0: was %#02x, now %#02x",
			from, to, before[target], r.cells[target])
	}
	for i := range r.cells {
		if i == target {
			continue
		}
		if r.cells[i] != before[i] {
			t.Errorf("cell at offset %d moved from %#02x to %#02x though turnHostile named a different cell",
				i, before[i], r.cells[i])
		}
	}
}

func TestTurnHostileOnASlotTheMatrixDoesNotHoldIsANoOp(t *testing.T) {
	t.Parallel()

	var r Relations
	for _, slot := range []uint32{0, relationSlots, relationSlots + 1, 1 << 31} {
		r.turnHostile(slot, 3)
		r.turnHostile(3, slot)
	}
	if r.cells != nil {
		t.Error("a call naming a slot the matrix does not hold materialised the matrix")
	}

	full, err := NewRelations(relFull())
	if err != nil {
		t.Fatalf("NewRelations: %v", err)
	}
	before := append([]byte(nil), full.cells...)
	for _, slot := range []uint32{0, relationSlots, relationSlots + 1, 1 << 31} {
		full.turnHostile(slot, 3)
		full.turnHostile(3, slot)
	}
	if !bytes.Equal(before, full.cells) {
		t.Error("a call naming a slot the matrix does not hold changed a materialised matrix")
	}
}

func TestNoRunOfTurnHostileCallsLowersAByte(t *testing.T) {
	t.Parallel()

	r, err := NewRelations(relFull())
	if err != nil {
		t.Fatalf("NewRelations: %v", err)
	}
	prev := append([]byte(nil), r.cells...)
	pairs := [][2]uint32{
		{1, 2}, {2, 1}, {3, 3}, {4, 9}, {9, 4}, {1, 2},
		{5, 5}, {2, 1}, {49, 1}, {1, 49}, {0, 7}, {7, 0},
	}
	for _, p := range pairs {
		r.turnHostile(p[0], p[1])
		for i, b := range r.cells {
			if lost := prev[i] &^ b; lost != 0 {
				t.Fatalf("after turnHostile(%d,%d), cell at offset %d lost bit(s) %#02x (was %#02x, now %#02x)",
					p[0], p[1], i, lost, prev[i], b)
			}
		}
		prev = append([]byte(nil), r.cells...)
	}
}

// TestARelationSurvivesTheFormAndReachesTheDigest is AC-1: the relation is
// canonical state, so it round-trips whole and one byte of it moves the digest.
//
// The byte moved is bit 1's, NOT bit 0's, and that is the point of choosing it:
// nothing in this build reads bit 1, so a field carried only as far as its reader
// would pass an equivalent test written over bit 0 and fail this one.
func TestARelationSurvivesTheFormAndReachesTheDigest(t *testing.T) {
	t.Parallel()

	w := relWorld(t, relFull())
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var got World
	if err := got.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if !bytes.Equal(got.Relations().cells, relFull()) {
		t.Error("the relation did not survive the byte form")
	}
	if got.Hash() != w.Hash() {
		t.Error("a decoded world hashes differently from the world it was cut from")
	}

	moved := relFull()
	moved[7*relationSlots+9] ^= 1 << 1
	other := relWorld(t, moved)
	if other.Hash() == w.Hash() {
		t.Error("two worlds differing in one relation byte hash the same")
	}
	if bytes.Equal(mustMarshal(t, other), form) {
		t.Error("two worlds differing in one relation byte marshal to the same bytes")
	}
}

// TestNoRelationAndAnAllZeroRelationAreOneWorld is AC-2. The two ways to build a
// world nobody authored a relation for are one world in the bytes and in the
// digest — not merely one in behaviour — which is the materialisation rule the
// three per-cell planes already stand under.
func TestNoRelationAndAnAllZeroRelationAreOneWorld(t *testing.T) {
	t.Parallel()

	unnamed := relWorld(t, nil)
	zeroed := relWorld(t, make([]byte, relationLen))
	if !bytes.Equal(mustMarshal(t, unnamed), mustMarshal(t, zeroed)) {
		t.Error("a world naming no relation and one naming the zero matrix marshal differently")
	}
	if unnamed.Hash() != zeroed.Hash() {
		t.Error("a world naming no relation and one naming the zero matrix hash differently")
	}
	// And the older constructors reach exactly that world, which is what leaves
	// every caller written before this story where it was.
	older, err := NewTerrainWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 1, Y: 1, Owner: 1}}, nil)
	if err != nil {
		t.Fatalf("NewTerrainWorld: %v", err)
	}
	if older.Hash() != unnamed.Hash() {
		t.Error("the constructor that names no relation builds a different world from the one that names none")
	}
}

// TestTheRelationADecoderIsHandedIsCopiedNotAliased is the half of the copy rule
// a round trip cannot see: the caller's bytes and the world's are separate
// arrays, in both directions, so mutating either afterwards reaches nothing.
func TestTheRelationADecoderIsHandedIsCopiedNotAliased(t *testing.T) {
	t.Parallel()

	cells := relFull()
	w := relWorld(t, cells)
	before := w.Hash()
	for i := range cells {
		cells[i] = 0xff
	}
	if w.Hash() != before {
		t.Error("mutating the slice a world was built from changed the world")
	}
	out := w.Relations()
	for i := range out.cells {
		out.cells[i] = 0xff
	}
	if w.Hash() != before {
		t.Error("mutating the relation a world handed back changed the world")
	}
}

// TestAFormTooShortForItsRelationIsRefused is the decoder's length rule at the
// one boundary the block introduces. A buffer one byte short of the block is the
// sharp case: every section in front of it is well formed, so nothing but the
// length check stands between it and a world whose relation was read out of its
// own script section.
func TestAFormTooShortForItsRelationIsRefused(t *testing.T) {
	t.Parallel()

	form := mustMarshal(t, relWorld(t, relFull()))
	for _, n := range []int{1, relationLen / 2, relationLen} {
		var w World
		if err := w.UnmarshalBinary(form[:len(form)-n]); err == nil {
			t.Errorf("a form %d byte(s) short of its relation was accepted", n)
		}
	}
}
