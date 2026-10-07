package sim

// This file is 0091's own: how far a unit sees is the unit's, and everything
// that follows from putting a number on an entity rather than in a constant.
//
// The fixtures here name their ranges explicitly and never read one off anything
// production carries. That rule is 0090's mutation battery's, learned on this
// very file's predecessor: a test that took its distance from the constant it was
// exercising moved with the constant and pinned nothing about it.

import (
	"bytes"
	"testing"
)

var rngBounds = Bounds{Width: 45, Height: 45}

// rngWorld is a flat world with the given entities on it. Flat because this file
// is about the RANGE and not about the terrain: on relief the region is a
// function of two things and a difference between two regions would not say which
// one moved it.
func rngWorld(t *testing.T, ents ...Entity) *World {
	t.Helper()
	w, err := NewWorld(1, rngBounds, ModeCanonical, nil, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return w
}

// rngLit is the set of cells a stamp lights, as a sorted, comparable key set.
func rngLit(w *World, stamp []byte) map[cell]bool {
	out := map[cell]bool{}
	for x := int32(0); x < rngBounds.Width; x++ {
		for y := int32(0); y < rngBounds.Height; y++ {
			if c := (cell{x: x, y: y}); w.sightShows(stamp, c) {
				out[c] = true
			}
		}
	}
	return out
}

// TestAUnitMarchesAtItsOwnRange is AC-1 and AC-2: the range an entity carries is
// the range its march is seeded with, and nothing supplies one in its place.
//
// The zero case is the one worth having twice over. Zero is a RANGE and not an
// absence: such a unit lights the cell it stands in — the walk marks that before
// any ring and outside every test — and no other, because a budget of half a step
// fails the cheapest step there is.
func TestAUnitMarchesAtItsOwnRange(t *testing.T) {
	t.Parallel()

	centre := cell{x: 22, y: 22}
	for _, tc := range []struct {
		name  string
		r     uint8
		alone bool
	}{
		{"a range of zero lights its own cell alone", 0, true},
		{"a range of one lights more", 1, false},
		{"a range of nine lights more still", 9, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := rngWorld(t, Entity{ID: 1, X: centre.x, Y: centre.y, ScanRange: tc.r})
			if got := w.Entities()[0].ScanRange; got != tc.r {
				t.Fatalf("the entity carries a range of %d, want %d", got, tc.r)
			}
			lit := rngLit(w, w.groupSight(aiSight, []int{0}))
			if !lit[centre] {
				t.Error("a unit cannot see the cell it is standing in")
			}
			if tc.alone && len(lit) != 1 {
				t.Errorf("a unit at range 0 lights %d cell(s), want its own alone", len(lit))
			}
			if !tc.alone && len(lit) <= 1 {
				t.Errorf("a unit at range %d lights %d cell(s)", tc.r, len(lit))
			}
		})
	}

	// An entity built naming no range carries none, which is the same zero and
	// not a different one — there is no default anywhere for the constructor to
	// have applied.
	w := rngWorld(t, Entity{ID: 1, X: centre.x, Y: centre.y})
	if got := w.Entities()[0].ScanRange; got != 0 {
		t.Errorf("an entity built naming no range carries %d", got)
	}
}

func TestAWiderRangeContainsANarrowerOne(t *testing.T) {
	t.Parallel()

	centre := cell{x: 22, y: 22}
	var prev map[cell]bool
	for r := uint8(0); r <= 12; r++ {
		w := rngWorld(t, Entity{ID: 1, X: centre.x, Y: centre.y, ScanRange: r})
		lit := rngLit(w, w.groupSight(aiSight, []int{0}))
		for c := range prev {
			if !lit[c] {
				t.Fatalf("range %d lights (%d,%d) and range %d does not", r-1, c.x, c.y, r)
			}
		}
		if r > 0 && len(lit) <= len(prev) {
			t.Errorf("range %d lights %d cell(s) and range %d lights %d", r, len(lit), r-1, len(prev))
		}
		prev = lit
	}
}

func TestAGroupsStampIsOneMarchPerMemberAtItsOwnRange(t *testing.T) {
	t.Parallel()

	near := Entity{ID: 1, X: 12, Y: 22, ScanRange: 3}
	far := Entity{ID: 2, X: 32, Y: 22, ScanRange: 10}

	w := rngWorld(t, near, far)
	both := rngLit(w, w.groupSight(aiSight, []int{0, 1}))
	only0 := rngLit(w, w.groupSight(aiSight, []int{0}))
	only1 := rngLit(w, w.groupSight(aiSight, []int{1}))

	for _, m := range []map[cell]bool{only0, only1} {
		for c := range m {
			if !both[c] {
				t.Errorf("(%d,%d) is lit by a member and not by its group", c.x, c.y)
			}
		}
	}
	if len(both) != len(only0)+len(only1) {
		t.Errorf("the group lights %d cell(s) and its two members %d and %d; the two regions overlap, "+
			"which this fixture places them too far apart to do", len(both), len(only0), len(only1))
	}
	// And the two members really do see different amounts, so the union above is
	// not a superset by accident.
	if len(only1) <= len(only0) {
		t.Errorf("the member at range 10 lights %d cell(s) against %d for the member at range 3",
			len(only1), len(only0))
	}
}

// TestTheNoticeRadiusFollowsThePairAndNotEitherTerm is AC-4.
//
// Three members, arranged so that the member furthest from the centroid, the
// member with the widest range, and the member whose SUM is largest are three
// DIFFERENT members. That is the whole discrimination: with one range for
// everybody the three questions had the same answer, so a build that took the
// maximum distance and added a range, or took the maximum range and added a
// distance, was indistinguishable from the law's own expression.
func TestTheNoticeRadiusFollowsThePairAndNotEitherTerm(t *testing.T) {
	t.Parallel()

	// Centroid of (0,0), (40,0) and (14,0) is (18,0). Distances 18, 22 and 4;
	// ranges 6, 5 and 30. So the furthest member is the second, the widest range
	// is the third, and the largest pair is the third's 34.
	ents := []Entity{
		{ID: 1, X: 0, Y: 0, ScanRange: 6},
		{ID: 2, X: 40, Y: 0, ScanRange: 5},
		{ID: 3, X: 14, Y: 0, ScanRange: 30},
	}
	members := []int{0, 1, 2}
	cx, cy := groupCentroid(ents, members)
	if cx != 18 || cy != 0 {
		t.Fatalf("the centroid is (%d,%d), which is not the geometry this case is built on", cx, cy)
	}
	if got, want := noticeBase(ents, members, cx, cy), uint8(34); got != want {
		t.Errorf("the notice base is %d, want %d — the largest of 18+6, 22+5 and 4+30",
			got, want)
	}
	// The two wrong readings this case exists to separate, written out so the
	// numbers above cannot drift into agreeing with one of them.
	if want := uint8(22 + 30); uint8(34) == want {
		t.Fatal("the widest distance plus the widest range is the same number as the largest pair")
	}
}

// TestTheRangeIsCanonicalState is AC-9, AC-10 and AC-11.
//
// Three claims in one place because they are one claim: the byte is in the form,
// it is in the digest, and it comes back whole. A field the encoder wrote and the
// decoder dropped would pass the first two and fail the third; one the digest
// skipped would pass the first and third and fail the second.
func TestTheRangeIsCanonicalState(t *testing.T) {
	t.Parallel()

	// NO VERSION LITERAL. The claim is "the range is in the form", and a literal
	// version number cannot say that — it holds when some other story adds a
	// field and forgets to bump, and it fails when another lane legitimately
	// takes the next number, which is what 0089 did with 17 while this branch was
	// open. What the record WIDTH says is structural and is this story's own: one
	// byte more than the record the branch merged.
	//
	// THE TRIPWIRE HAS NOW FIRED AGAIN, CORRECTLY: 230 is the record 0166
	// leaves it at, seven bytes past 0164's 223 — one from widening the spell
	// mark's remaining ticks to a word, and six from the escort triple. It is
	// re-pinned here rather than deleted, on corpseloot_test.go's own
	// precedent for a literal like this one.
	if entityLen != 492 {
		t.Errorf("the entity record is %d bytes; the 100 this story's own claim was measured against "+
			"plus every later story's own widening, 1001's actor tail, corpse-loot property, kill "+
			"attribution, 1025's load and capacity, 1029's authored map id, 1039's resistance, 1037's withdrawal thresholds, later saved actor fields and 1111's source binding is 492", entityLen)
	}
	// And the form this build writes opens at the version this build defines,
	// whatever number that is.
	probe := rngWorld(t, Entity{ID: 1, X: 1, Y: 1, ScanRange: 6})
	if form, err := probe.MarshalBinary(); err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	} else if form[0] != formatVersion {
		t.Errorf("the form opens at %d, want %d", form[0], formatVersion)
	}

	// Two worlds alike in everything but one range: different bytes, different
	// digests.
	a := rngWorld(t, Entity{ID: 1, X: 5, Y: 5, ScanRange: 6})
	b := rngWorld(t, Entity{ID: 1, X: 5, Y: 5, ScanRange: 7})
	fa, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	fb, err := b.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if bytes.Equal(fa, fb) {
		t.Error("two worlds whose ranges differ marshal to the same bytes")
	}
	if a.Hash() == b.Hash() {
		t.Error("two worlds whose ranges differ hash alike")
	}

	for v := 0; v < 256; v++ {
		w := rngWorld(t, Entity{ID: 1, X: 5, Y: 5, ScanRange: uint8(v)})
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("range %d: MarshalBinary: %v", v, err)
		}
		var back World
		if err := back.UnmarshalBinary(form); err != nil {
			t.Fatalf("range %d: UnmarshalBinary: %v", v, err)
		}
		if got := back.Entities()[0].ScanRange; got != uint8(v) {
			t.Errorf("a range of %d decoded to %d", v, got)
		}
	}
}

// TestTheNoticeRadiusNarrowsPerMemberAndNotAfterTheMaximum is the field's own
// WIDTH, and it is the one claim here a mutation survived until it was written.
//
// The law's group record holds the maximum distance, the maximum sight and their
// combined maximum in three CONSECUTIVE BYTES, so every term is narrowed on the
// way in — per member, before the comparison. Narrowing the maximum instead is
// indistinguishable on every world this tree can load, because a map is at most
// 136 cells on a side and no sum can reach 256 there; the two part company only
// past that, and then they disagree about WHICH MEMBER WINS rather than by a
// multiple of 256.
//
// It is therefore a CUSTOMISATION LIMIT measured rather than an assumption: this
// world is built by hand at bounds no loader produces, which is the only way the
// difference is observable at all.
func TestTheNoticeRadiusNarrowsPerMemberAndNotAfterTheMaximum(t *testing.T) {
	t.Parallel()

	// Two members on a line from the centre, at 300 and at 200, neither seeing
	// anything. Narrowed per member the pair is 44 and 200, so the NEARER member
	// decides; narrowed after the maximum it is 300 truncated to 44, so the
	// further one does and the answer is four times smaller.
	ents := []Entity{{ID: 1, X: 300, Y: 0}, {ID: 2, X: 200, Y: 0}}
	got := noticeBase(ents, []int{0, 1}, 0, 0)
	if want := uint8(200); got != want {
		t.Errorf("the notice base is %d, want %d — each member's pair is narrowed to a byte "+
			"before the maximum, so 300 enters as 44 and the member at 200 wins", got, want)
	}
	if wrong := uint8(300 % 256); got == wrong {
		t.Errorf("the base is %d, which is what narrowing AFTER the maximum gives", got)
	}
}
