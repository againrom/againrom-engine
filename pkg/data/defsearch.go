package data

// Collection is one of the definition table's by-value collections as a search
// reads it: a length, and per index an entry's name bytes and its parameter
// array.
//
// It is an INTERFACE and not a struct so that the tier which parses the table
// stays off this package's import list. A parsed collection satisfies it by
// having the three methods, no copy of the entries is made to cross the
// boundary, and nothing here can reach a field the format tier did not mean to
// expose. Index it with the collection's own subscript: the searched
// collections are one-based, so index 0 is the reserved empty entry.
type Collection interface {
	Len() int
	EntryName(i int) string
	EntryParams(i int) []int32

	// EntryStrings is the entry's trailing strings — the equipment names a row
	// carries after its parameters. An entry whose collection writes none
	// answers nil.
	//
	// It is part of THIS interface rather than a narrower one beside it because
	// an entry of that file is a name, a parameter array and a run of trailing
	// strings, and an interface that admitted only the first two was narrower
	// than the thing it models: a reader looking for the equipment had nowhere
	// to look, and a type assertion at the call site would have folded "this
	// table carries no equipment" onto "this table's type does not admit any".
	EntryStrings(i int) []string
}

// The key columns the three searches read. They are slot numbers, i.e. the
// column one to the right of the one they name, and they are the ONLY meaning
// this file reads into a parameter.
const (
	unitTypeIDSlot    = 0x1d
	unitFaceSlot      = 0x1e
	humanTypeIDSlot   = 0x10
	humanServerIDSlot = 0x18
	unitServerIDSlot  = 55
)

// NotFound is what a search reports when no entry matches. It is index 0, which
// a one-based collection reserves and never writes, so "nothing matched" and
// "the first entry matched" can never be the same answer.
const NotFound = 0

// FindUnit walks c ascending from index 1 and returns the first entry whose
// typeID and face columns equal both keys. FIRST MATCH WINS: the shipped
// population happens to be unique on that pair, but the engine's own search is a
// first-match walk, so a file that repeats a key is answered rather than
// refused.
func FindUnit(c Collection, typeID, face int32) int {
	return ascending(c, func(p []int32) bool {
		return slotEquals(p, unitTypeIDSlot, typeID) && slotEquals(p, unitFaceSlot, face)
	})
}

// FindHumanByType walks c ascending from index 1 and returns the first entry
// whose typeID column equals the key.
func FindUnitByServerID(c Collection, serverID int32) int {
	return ascending(c, func(p []int32) bool {
		return slotEquals(p, unitServerIDSlot, serverID)
	})
}

func FindHumanByType(c Collection, typeID int32) int {
	return ascending(c, func(p []int32) bool {
		return slotEquals(p, humanTypeIDSlot, typeID)
	})
}

// FindHumanByServerID walks c DOWNWARD from its last entry and returns the first
// entry whose serverID column equals the key.
//
// The direction is the engine's and it is observable: where two entries carry
// one server id, this reaches the LATER one while the two ascending searches
// above would reach the earlier.
func FindHumanByServerID(c Collection, serverID int32) int {
	return descending(c, func(p []int32) bool {
		return slotEquals(p, humanServerIDSlot, serverID)
	})
}

func FindHumanByName(c Collection, name string) int {
	if c == nil {
		return NotFound
	}
	for i := 1; i < c.Len(); i++ {
		if n := c.EntryName(i); n != "" && n == name {
			return i
		}
	}
	return NotFound
}

// ascending walks the written entries from index 1 upward; descending walks them
// from the last down to index 1. Both skip an EMPTY-NAMED entry — a collection
// is sized by its count and the rows that were never written carry no name — and
// both build nothing: no index, no map, no first pass.
//
// A nil collection is no entries, which is what lets a caller hold "no table"
// without a branch of its own.
func ascending(c Collection, match func(p []int32) bool) int {
	if c == nil {
		return NotFound
	}
	for i := 1; i < c.Len(); i++ {
		if hit(c, i, match) {
			return i
		}
	}
	return NotFound
}

func descending(c Collection, match func(p []int32) bool) int {
	if c == nil {
		return NotFound
	}
	for i := c.Len() - 1; i >= 1; i-- {
		if hit(c, i, match) {
			return i
		}
	}
	return NotFound
}

func hit(c Collection, i int, match func(p []int32) bool) bool {
	return c.EntryName(i) != "" && match(c.EntryParams(i))
}

// slotEquals reports whether p's slot holds want. A row too short to have that
// column cannot carry that key and so cannot match — which is how the sixty-odd
// entries that ship with no parameter array at all stay out of every result
// without being special-cased anywhere.
func slotEquals(p []int32, slot int, want int32) bool {
	return slot < len(p) && p[slot] == want
}
