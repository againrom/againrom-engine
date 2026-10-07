package data

import (
	"reflect"
	"testing"
)

// The placement byte's lookup, over a synthetic registry (SC-1, AC-1).
//
// enumerate_test.go pins ByID over a sparse ID domain. These pin the ONE offset
// above it: which byte of a map's object layer names which class, and that
// asking never changes an answer. Nothing here reads an install — internal/synth
// writes the .reg byte stream and pkg/formats/reg parses it back, as everywhere
// else in this package.

// threeObjects is AC-1's registry: IDs 0, 1 and 2. Each class carries an Index
// of its own, so a lookup landing on the neighbouring class is visible as a
// value and not merely as a pointer that is not the expected one; the animation
// pair on one of them gives the purity check below a slice to compare, a struct
// copy being the thing that would alias one.
func threeObjects(t *testing.T) *ObjectClasses {
	t.Helper()
	return loadObjects(t, objectsReg(t, 3,
		regDir("Object0", regInt("ID", 0), regInt("File", 0), regInt("Index", 10)),
		regDir("Object1", regInt("ID", 1), regInt("File", 0), regInt("Index", 11),
			regInts("AnimationTime", 4, 5), regInts("AnimationFrame", 6, 7)),
		regDir("Object2", regInt("ID", 2), regInt("File", 0), regInt("Index", 12)),
	))
}

// SC-1 (AC-1). AC-1's four bytes, exactly. What byte 0 discriminates here is a
// lookup applying no offset at all, which would draw Object0 on every cell that
// holds nothing; a subtraction done at byte width, wrapping 0 to 255, misses on
// a three-class registry for a reason of its own and is ruled out by the
// widening in the code rather than by this row.
func TestByCodeNamesTheClassOneBelow(t *testing.T) {
	cs := threeObjects(t)

	for _, tc := range []struct {
		code byte
		id   int32 // the ID the byte resolves to, or -1 for a miss
		why  string
	}{
		{0, -1, "no object — and NOT the class whose ID is 0"},
		{1, 0, "the first class, which byte 1 and not byte 0 names"},
		{3, 2, "the third class, one below the byte"},
		{83, -1, "ID 82: past this fixture, and past all 82 shipped classes"},
	} {
		c, ok := cs.ByCode(tc.code)
		switch {
		case tc.id < 0:
			if ok {
				t.Errorf("ByCode(%d) hit the class whose ID is %d, want a miss — %s",
					tc.code, c.ID, tc.why)
			}
		case !ok:
			t.Errorf("ByCode(%d) missed, want the class whose ID is %d — %s",
				tc.code, tc.id, tc.why)
		default:
			if c.ID != tc.id {
				t.Errorf("ByCode(%d) hit ID %d, want %d — %s", tc.code, c.ID, tc.id, tc.why)
				continue
			}
			// One collection and one set of pointers: the byte reaches the same
			// class the ID does, rather than any class that compares equal to it.
			if want, _ := cs.ByID(tc.id); c != want {
				t.Errorf("ByCode(%d) is not ByID(%d) — the lookup must go through the collection",
					tc.code, tc.id)
			}
		}
	}
}

// SC-1. The lookup is total and pure: every one of the 256 bytes answers,
// exactly the three the fixture loads hit, and the collection is unchanged
// afterwards. The comparison is against a SECOND load of the same registry
// rather than against a snapshot taken from the first — a snapshot of the
// class structs would share their slices, and a mutated element would
// compare equal to itself.
func TestByCodeIsTotalAndMutatesNothing(t *testing.T) {
	cs := threeObjects(t)
	untouched := threeObjects(t)

	var hits []int
	for code := 0; code < 256; code++ {
		c, ok := cs.ByCode(byte(code))
		if !ok {
			continue
		}
		hits = append(hits, code)
		if c.ID != int32(code)-1 {
			t.Errorf("ByCode(%d) hit ID %d, want %d", code, c.ID, code-1)
		}
	}
	if want := []int{1, 2, 3}; !reflect.DeepEqual(hits, want) {
		t.Errorf("bytes hitting a class = %v, want %v — three classes, three bytes, "+
			"and byte 0 among them would mean a cell holding nothing draws Object0", hits, want)
	}

	if !reflect.DeepEqual(cs, untouched) {
		t.Error("the collection differs from a freshly loaded one after 256 lookups; " +
			"ByCode must read the classes and write none")
	}
}
