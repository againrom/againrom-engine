package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
)

// animGateClass is a resolved structure class over a 2 x 1 rectangle of
// FullHeight 3 — six grid cells — carrying the given animation keys.
func animGateClass(phases int32, mask string, times, frames []int32) *data.StructureClass {
	return &data.StructureClass{
		ID: 1, TileWidth: 2, TileHeight: 1, FullHeight: 3,
		Phases: phases, AnimMask: mask, AnimTime: times, AnimFrame: frames,
	}
}

func animGateResult(c *data.StructureClass) *terrain.StructureClass {
	sc := &terrain.StructureClass{TileWidth: 2, TileHeight: 1, FullHeight: 3}
	structureAnimation(c, sc)
	return sc
}

// AC-2, over the four precondition shapes and the two the registry cannot carry.
func TestStructureAnimationGateIsAConjunction(t *testing.T) {
	times, frames := []int32{2, 1}, []int32{1, 2} // expands to [1 1 2], period 3

	for _, tc := range []struct {
		name     string
		class    *data.StructureClass
		animates bool
	}{
		{"Phases 1 with a mask and a pair", animGateClass(1, "x-x-x-", times, frames), false},
		{"Phases 0", animGateClass(0, "x-x-x-", times, frames), false},
		{"the registry's own default for an omitted Phases", animGateClass(-1, "x-x-x-", times, frames), false},
		{"a mask of the RECTANGLE's length", animGateClass(3, "x-", times, frames), false},
		{"a mask one byte long", animGateClass(3, "x", times, frames), false},
		{"a mask one byte too long", animGateClass(3, "x-x-x-x", times, frames), false},
		{"no mask at all", animGateClass(3, "", times, frames), false},
		{"no animation pair", animGateClass(3, "x-x-x-", nil, nil), false},
		{"a pair whose every time is non-positive", animGateClass(3, "x-x-x-", []int32{0, 0}, frames), false},
		{"all three present", animGateClass(3, "x-x-x-", times, frames), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sc := animGateResult(tc.class)
			if got := len(sc.Timeline) > 0; got != tc.animates {
				t.Fatalf("animates = %t, want %t (timeline %v)", got, tc.animates, sc.Timeline)
			}
			if tc.animates {
				return
			}
			// A FAILED GATE LEAVES NOTHING BEHIND. The mask is refused whole:
			// nothing truncates it, pads it or indexes it, so there is no rank
			// array for a later reader to mistake for a valid one.
			if len(sc.Rank) != 0 || sc.Live != 0 {
				t.Errorf("a failed gate left rank %v and live %d", sc.Rank, sc.Live)
			}
		})
	}
}

// The ranks the gate computes: '-' retires a cell, every other byte leaves it
// live, and the rank is EXCLUSIVE — the first live cell is at offset 0 of its
// phase, not at offset 1.
func TestStructureRanksCountLiveCellsExclusively(t *testing.T) {
	for _, tc := range []struct {
		mask string
		rank []int
		live int
	}{
		{"------", []int{-1, -1, -1, -1, -1, -1}, 0},
		{"xxxxxx", []int{0, 1, 2, 3, 4, 5}, 6},
		{"x-x-x-", []int{0, -1, 1, -1, 2, -1}, 3},
		{"-xx--x", []int{-1, 0, 1, -1, -1, 2}, 3},
	} {
		t.Run(tc.mask, func(t *testing.T) {
			rank, live := structureRanks(tc.mask)
			if live != tc.live {
				t.Errorf("live = %d, want %d", live, tc.live)
			}
			if len(rank) != len(tc.rank) {
				t.Fatalf("rank = %v, want %v", rank, tc.rank)
			}
			for i := range tc.rank {
				if rank[i] != tc.rank[i] {
					t.Fatalf("rank = %v, want %v", rank, tc.rank)
				}
			}
		})
	}
}
