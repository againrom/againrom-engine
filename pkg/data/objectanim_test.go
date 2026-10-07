package data

import (
	"slices"
	"testing"
)

// The object class's expanded cycle over hand-written literals (0031 AC-1,
// SC-1).
//
// Timeline() is a pure function of one resolved class's two arrays — no
// registry, no loader, no sheet, no IO — so every fixture below is a class
// literal and every expected timeline is written out BY HAND. An expectation
// produced by the expansion under test would pin nothing: it would agree with
// whatever the walk happened to do, including a walk that dropped a round or
// ran off the shorter array.

// objectClassWith is the fixture: a class carrying one AnimationTime /
// AnimationFrame pair and nothing else. Every other field is left at its zero
// value, because none of them reaches the expansion.
func objectClassWith(times, frames []int32) *ObjectClass {
	return &ObjectClass{AnimationTime: times, AnimationFrame: frames}
}

// TestObjectTimelineExpansion is AC-1's four pairs, each on a case of its own.
func TestObjectTimelineExpansion(t *testing.T) {
	cases := []struct {
		name   string
		times  []int32
		frames []int32
		want   []int
	}{
		{
			// The spec's own I/O example: seven fours over the values 0..6.
			// Written out in full — 28 entries — rather than generated, so a
			// walk that emitted 7 entries or 28 zeroes fails here.
			name:   "seven fours over 0..6",
			times:  []int32{4, 4, 4, 4, 4, 4, 4},
			frames: []int32{0, 1, 2, 3, 4, 5, 6},
			want: []int{
				0, 0, 0, 0,
				1, 1, 1, 1,
				2, 2, 2, 2,
				3, 3, 3, 3,
				4, 4, 4, 4,
				5, 5, 5, 5,
				6, 6, 6, 6,
			},
		},
		{
			// A zero duration in the middle: 5 three times, 6 not at all, 7
			// twice. The round is still consumed, so 7 pairs with 2 and not
			// with 6 — a walk that skipped the zero entry without advancing
			// both heads would give 5,5,5,6,6.
			name:   "a zero duration drops its value but not its round",
			times:  []int32{3, 0, 2},
			frames: []int32{5, 6, 7},
			want:   []int{5, 5, 5, 7, 7},
		},
		{
			// The value array is shorter, so the walk ends after one round.
			name:   "the shorter array ends the walk",
			times:  []int32{2, 2},
			frames: []int32{9},
			want:   []int{9, 9},
		},
		{
			// No durations at all: no cycle, whatever the values say.
			name:   "no durations is no cycle",
			times:  nil,
			frames: []int32{1},
			want:   nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := objectClassWith(c.times, c.frames).Timeline()
			if !slices.Equal(got, c.want) {
				t.Errorf("Timeline() = %v, want %v", got, c.want)
			}
			// The length IS the period, so it is asserted as its own fact rather than
			// left implied by the slice comparison: a consumer reduces its step by
			// exactly this number.
			if len(got) != len(c.want) {
				t.Errorf("period = %d, want %d", len(got), len(c.want))
			}
		})
	}
}

// TestObjectTimelineIgnoresPhases pins the clause the baseline got wrong: a
// class's Phases scalar is set on classes carrying no arrays at all, so it
// cannot say whether a class has a cycle and Timeline() must not read it.
//
// Both directions are asserted, because either alone is satisfiable by
// accident: a class with a large Phases and no arrays has NO cycle, and a
// class with Phases 0 and a usable pair HAS one.
func TestObjectTimelineIgnoresPhases(t *testing.T) {
	loud := &ObjectClass{Phases: 17}
	if got := loud.Timeline(); len(got) != 0 {
		t.Errorf("Timeline() = %v over Phases 17 and no arrays, want empty — Phases is not a cycle", got)
	}

	quiet := &ObjectClass{Phases: 0, AnimationTime: []int32{2}, AnimationFrame: []int32{3}}
	if got, want := quiet.Timeline(), []int{3, 3}; !slices.Equal(got, want) {
		t.Errorf("Timeline() = %v over Phases 0 and a usable pair, want %v", got, want)
	}
}

// TestObjectTimelineIsDerivedNotHeld pins that the expansion holds no state
// between calls and returns a fresh slice each time: two calls on one class
// agree, and writing through one answer does not move the other.
func TestObjectTimelineIsDerivedNotHeld(t *testing.T) {
	c := objectClassWith([]int32{2, 1}, []int32{4, 5})
	first := c.Timeline()
	if want := []int{4, 4, 5}; !slices.Equal(first, want) {
		t.Fatalf("Timeline() = %v, want %v", first, want)
	}
	first[0] = 99
	if got := c.Timeline(); !slices.Equal(got, []int{4, 4, 5}) {
		t.Errorf("Timeline() = %v after writing through an earlier answer, want [4 4 5]", got)
	}
}

// TestObjectTimelineNegativeDuration is the other half of "non-positive": the
// clause says non-positive and the fixtures above test only zero.
func TestObjectTimelineNegativeDuration(t *testing.T) {
	got := objectClassWith([]int32{-4, 2}, []int32{1, 2}).Timeline()
	if want := []int{2, 2}; !slices.Equal(got, want) {
		t.Errorf("Timeline() = %v over a negative duration, want %v", got, want)
	}
}
