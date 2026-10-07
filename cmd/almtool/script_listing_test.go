package main

// The `script` verb's trigger listing: the band note it prints for a failed
// Target_Unit reference.
//
// WHAT THIS FILE DOES NOT COVER, said here rather than implied by its absence.
// The listing's other half is a JOIN -- compiled check subscript i is condition
// i, and that condition's own ID is the authored node id the binder report
// prints -- and it is pinned in pkg/mapload, at the binder that establishes it,
// by TestConditionSubscriptIsListOrderAndUnresolvedCarriesTheAuthoredNodeID.
// Pinning it here as well would test this file against a copy of the fact
// rather than against the fact.

import (
	"strings"
	"testing"
)

// TestUnitFailureNoteSeparatesTheThreeBands drives every arm of the note and
// both edges of the hero band. The three bands are three different findings and
// the earlier draft of this listing printed them as one line; the edges are
// where an off-by-one would put a map's own dangling reference into the hero
// band, or the last ordinal into the name table.
func TestUnitFailureNoteSeparatesTheThreeBands(t *testing.T) {
	cases := []struct {
		name string
		v    uint32
		want string
	}{
		{
			"a placed record's own id",
			42,
			"map unit 42 is not in this map's unit table",
		},
		{
			"the last value below the band",
			10000,
			"map unit 10000 is not in this map's unit table",
		},
		{
			"the band's own low edge is ordinal 1, which a party resolves",
			10001,
			"hero ordinal 1, which resolves when the caller supplies a party and did not here",
		},
		{
			"the next ordinal names nobody in a one-player build",
			10002,
			"hero ordinal 2, which names nobody in a one-player build even with a party",
		},
		{
			"the band's own high edge is still an ordinal",
			11000,
			"hero ordinal 1000, which names nobody in a one-player build even with a party",
		},
		{
			"one past the high edge is the name table",
			11001,
			"name-table id 11001, a table this tree does not carry",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := unitFailureNote(c.v); got != c.want {
				t.Errorf("value %d notes %q, want %q", c.v, got, c.want)
			}
		})
	}
}

func TestUnitFailureNoteNamesOrdinalOneOnlyAtTheBandEdge(t *testing.T) {
	one, two := unitFailureNote(10001), unitFailureNote(10002)
	if one == two {
		t.Fatalf("ordinal 1 and ordinal 2 both note %q", one)
	}
	if want := "resolves when the caller supplies a party"; !strings.Contains(one, want) {
		t.Errorf("ordinal 1 notes %q, which does not say %q", one, want)
	}
	if want := "names nobody in a one-player build"; !strings.Contains(two, want) {
		t.Errorf("ordinal 2 notes %q, which does not say %q", two, want)
	}
}
