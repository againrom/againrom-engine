package sim

import "fmt"

// THE RELATION: which roster slot treats which other roster slot as an enemy.
//
// It is the first question the engagement decision asks and the only one whose
// answer no rule of this package can derive — a group looks at everything it can
// see and keeps what this says is hostile. With nothing written nobody is
// hostile to anybody and no unit ever finds an enemy, which is exactly the state
// every world built before this file existed was in (AI-DIPLO-004).
//
// IT IS DIRECTIONAL, and that is a measurement rather than a convenience. Over
// the 38 shipped maps 102 of 866 ordered off-diagonal pairs disagree with their
// mirror on the hostility bit, across 19 maps (AI-DIPLO-005): guards attack
// monsters that will not start a fight with guards. A symmetric store is the
// wrong shape, and it is the wrong shape in a way that would look right on most
// maps.

// relationSlots is how many roster slots the relation describes per side.
//
// It is a DECODED LIMIT and not a capacity of ours: the session constructor
// zeroes 625 dwords at the matrix's base — 2500 bytes, 50 by 50 — and every
// index into it is taken at stride 50 (AI-DIPLO-004). So fifty is how many
// players the thing being reconstructed can hold a relation for, and raising it
// is a customisation that moves the size of a block a save writes verbatim
// (AI-DIPLO-085). No shipped map ships more than sixteen players.
const relationSlots = 50

// relationLen is the matrix's whole size in bytes, and it is a compile-time
// constant of this build — which is why the byte form carries the block with no
// length of its own to disagree with it.
const relationLen = relationSlots * relationSlots

// relationHostile is bit 0: "the row's slot treats the column's slot as an
// enemy". It is THE ONLY BIT any rule in this build reads.
//
// The others are carried and never interpreted. Bit 1 locks the pair against
// being turned hostile by combat — it is the one thing that stops a blow,
// since turnHostile declines to set bit 0 wherever bit 1 is already set — and
// bit 2 is read by the engine's own mask and used by no shipped map; bit 2
// alone still belongs to a writer this story does not implement, and masking
// it away here would make two matrices the byte form distinguishes into one
// world.
const relationHostile = 1 << 0

// relationLocked is bit 1: the pair is allied and stays that way, against
// combat. turnHostile reads it: a cell with bit 1 set is never FLIPPABLE, so
// the lock holds against every blow.
const relationLocked = 1 << 1

// Relations is one world's relation, indexed [me][him] by roster slot.
//
// THE ZERO VALUE IS THE EMPTY RELATION and carries no storage at all. That is
// what makes "a world named no relation" and "a world named the all-zero matrix"
// one world in the fields, in the bytes and in the digest, with no flag
// recording which way it was built — the rule the three per-cell planes already
// follow, reached here without materialising 2500 bytes for every world that
// will never write one.
//
// It is a TYPE and not a bare slice on the world because the stride is the
// failure worth designing out. A transposed index is not a crash and not a
// refusal: it is a different, entirely plausible game, in which the guards
// attack the monsters that would not have attacked them. One type means one
// place computes the offset.
type Relations struct {
	// cells is the matrix in row-major order at stride relationSlots, or nil for
	// the empty relation. Nothing outside this file reads it directly.
	cells []byte
}

// NewRelations returns the relation carrying cells: the EMPTY relation when
// cells is empty, and a copy of them when there are exactly relationLen.
//
// Any other length is refused rather than padded or truncated, which is
// newGrid's own rule and is what keeps the encoding injective — a matrix
// silently completed here would make two inputs one world.
//
// It COPIES, so the caller's slice stays the caller's to reuse or mutate and a
// relation cannot change under a world that is being advanced.
func NewRelations(cells []byte) (Relations, error) {
	if len(cells) == 0 {
		return Relations{}, nil
	}
	if len(cells) != relationLen {
		return Relations{}, fmt.Errorf("sim: relation carries %d byte(s), want %d for %d slots",
			len(cells), relationLen, relationSlots)
	}
	return Relations{cells: append([]byte(nil), cells...)}, nil
}

// relationIndex is the offset of cell [from][to], and whether the matrix holds
// one at all.
//
// It is the ONE place the stride is written down. A slot outside the matrix has
// no cell — slot 0 because it names no roster entry at all, and a slot at or
// past relationSlots because the matrix ends there — and both answer the same
// way, so no caller has two out-of-range rules to keep in agreement.
func relationIndex(from, to uint32) (int, bool) {
	if from == 0 || to == 0 || from >= relationSlots || to >= relationSlots {
		return 0, false
	}
	return int(from)*relationSlots + int(to), true
}

// Set writes v at [from][to], materialising the matrix if this is the first
// write.
//
// A slot the matrix does not hold is IGNORED rather than refused. The caller is
// a map loader replaying the engine's own store, and the engine's store is a
// byte write into a fixed block: a roster naming a slot past the block writes
// nowhere and carries on, and an error here would turn a map the original loads
// into a map this tree will not.
func (r *Relations) Set(from, to uint32, v byte) {
	i, ok := relationIndex(from, to)
	if !ok {
		return
	}
	if r.cells == nil {
		r.cells = make([]byte, relationLen)
	}
	r.cells[i] = v
}

// turnHostile sets bit 0 at [from][to] when that cell is FLIPPABLE — its low
// two bits both clear — and leaves every other cell byte-for-byte alone. A
// cell already hostile, or locked by bit 1, is declined rather than
// overwritten, and no bit above bit 0 is ever written. Materialises the
// matrix on that write, exactly as Set does.
//
// A slot the matrix does not hold is a NO-OP, not a refusal —
// relationIndex's own rule, reused rather than re-decided here, so there is
// no second out-of-range test to keep in agreement. Called with such a slot
// on an empty relation, it does not materialise the matrix either: nothing
// is written, so there is nothing to allocate for.
//
// IT TAKES ONE DIRECTION. The caller makes the second call for the mirror
// cell; a symmetric helper here would flip both cells under one test and get
// exactly the pairs that matter wrong — one already hostile, the other
// still clear.
func (r *Relations) turnHostile(from, to uint32) {
	i, ok := relationIndex(from, to)
	if !ok {
		return
	}
	if r.cells != nil && r.cells[i]&(relationHostile|relationLocked) != 0 {
		return
	}
	if r.cells == nil {
		r.cells = make([]byte, relationLen)
	}
	r.cells[i] |= relationHostile
}

// changeRelation is instant opcode 10's write: the relation cell at
// [from][to] becomes (cell &^ 3) + p2 — a READ-MODIFY-WRITE and not an
// assignment, so bits 2 to 7 survive untouched. p2 is ADDED rather than
// or-ed in, so a p2 of 4 or more carries into those surviving bits and the
// sum truncates in the byte exactly as every other byte in this matrix does;
// nothing here clamps it.
//
// IT IS turnHostile's SIBLING, materialising the matrix on first write and
// going through relationIndex for the same out-of-range no-op, on the same
// grounds Set and turnHostile already state. It is deliberately UNLIKE
// turnHostile in the one place that matters: turnHostile declines a pair
// that is already hostile or locked by bit 1, and this write tests neither
// before it stores.
func (r *Relations) changeRelation(from, to uint32, p2 int32) {
	i, ok := relationIndex(from, to)
	if !ok {
		return
	}
	if r.cells == nil {
		r.cells = make([]byte, relationLen)
	}
	r.cells[i] = byte(int32(r.cells[i]&^3) + p2)
}

// Byte is the whole byte at [from][to], and zero where the matrix holds no such
// cell. Every bit above bit 0 reaches a caller through this and through nothing
// else.
func (r Relations) Byte(from, to uint32) byte {
	i, ok := relationIndex(from, to)
	if !ok || r.cells == nil {
		return 0
	}
	return r.cells[i]
}

// Hostile reports whether slot from treats slot to as an enemy: bit 0 of
// [from][to].
//
// IT IS NOT SYMMETRIC and it is not reflexive-by-rule. A map forces its own
// diagonal to 2, whose bit 0 is clear, so a slot is not hostile to itself on any
// map that authored one — but that is the map's doing rather than this
// function's, and a caller that wrote a 1 there would get a slot at war with
// itself, exactly as the original would.
func (r Relations) Hostile(from, to uint32) bool {
	return r.Byte(from, to)&relationHostile != 0
}

// Locked reports whether the relation from slot from toward slot to carries
// relationLocked — bit 1 — regardless of what bit 0 says.
func (r Relations) Locked(from, to uint32) bool {
	return r.Byte(from, to)&relationLocked != 0
}

// materialised is r with its matrix PRESENT: a fresh copy of relationLen bytes,
// all-zero where r carried none.
//
// It is newGrid's rule and it is here for newGrid's reason. A world takes its
// relation through this at construction and hands one back through it at every
// read, so inside a world there is exactly ONE representation of "nobody is
// hostile to anybody" — not a nil that behaves like a zeroed matrix and a zeroed
// matrix, kept in agreement by whoever remembers. Nothing downstream can ask
// which way a world was built, because nothing stores it.
//
// It copies for the reason the entity slice and the three planes do: the
// caller's bytes stay the caller's, and a relation cannot change under a world
// that is being advanced.
func (r Relations) materialised() Relations {
	if r.cells == nil {
		return Relations{cells: make([]byte, relationLen)}
	}
	return Relations{cells: append([]byte(nil), r.cells...)}
}

// encodeInto writes the matrix's relationLen bytes at the front of b.
//
// The nil arm keeps the function TOTAL rather than covering a state a world can
// be in: a world's relation is materialised at construction, so the encoder
// never reaches it. An all-zero block is what the empty relation is, and the
// caller's buffer is freshly allocated and therefore already one.
func (r Relations) encodeInto(b []byte) {
	if r.cells != nil {
		copy(b, r.cells)
	}
}
