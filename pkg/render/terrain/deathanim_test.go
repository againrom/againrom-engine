package terrain_test

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// dyingAnim is a descriptor whose dying block is the only thing distinct about
// it: base 40, slot length 4, D directions. Every other field is left at its
// zero value on purpose — the death selection reads exactly three of them, and
// a selection that reached for a move base or an idle track would index 0 and
// be visible immediately.
func dyingAnim(d, slot int) terrain.UnitAnim {
	return terrain.UnitAnim{S: 16, D: d, DyingBase: 40, DyingSlot: slot}
}

// deathIndex is the sheet-contract rule, transcribed here: the direction rule
// for the animated blocks, then the block's own base plus that direction's
// whole slots plus the phase. It is this file's own derivation and never a
// call into the package.
func deathIndex(base, d, slot, oct, phase int) (frame int, mirror bool) {
	s, m := oct, false
	if d == 5 && oct > 4 {
		s, m = 8-oct, true
	}
	return base + s*slot + phase, m
}

// One dying frame is held for exactly two ticks, and the block plays once from
// its first frame to its last (AC-3, SC-3).
func TestADyingFrameIsHeldForTwoTicks(t *testing.T) {
	const slot, oct = 4, 6
	a := dyingAnim(8, slot)
	for tick := 0; tick < 2*slot; tick++ {
		want, wantMirror := deathIndex(40, 8, slot, oct, tick/2)
		frame, mirror, ok := terrain.SelectDeathFrame(a, 200, oct, tick)
		if !ok {
			t.Fatalf("tick %d: no frame, and this sheet holds 200", tick)
		}
		if frame != want || mirror != wantMirror {
			t.Errorf("tick %d: frame %d mirror %v, want %d %v", tick, frame, mirror, want, wantMirror)
		}
	}

	// The run is 2*slot ticks long and no shorter: the LAST frame of the block
	// is first reached at tick 2*(slot-1), which is 6 here, not 3 and not 4.
	last, _ := deathIndex(40, 8, slot, oct, slot-1)
	if frame, _, _ := terrain.SelectDeathFrame(a, 200, oct, 2*(slot-1)); frame != last {
		t.Errorf("the block reaches its last frame at tick %d as %d, want %d", 2*(slot-1), frame, last)
	}
	if frame, _, _ := terrain.SelectDeathFrame(a, 200, oct, 2*(slot-1)-1); frame == last {
		t.Errorf("the block reached its last frame a tick early, at %d", 2*(slot-1)-1)
	}
}

// Past the run the last dying frame is held, at every tick however large,
// and the answer never leaves its own direction's slot (AC-4, SC-4).
func TestPastTheRunTheLastDyingFrameIsHeld(t *testing.T) {
	const slot = 3
	a := dyingAnim(8, slot)
	for oct := 0; oct < 8; oct++ {
		want, _ := deathIndex(40, 8, slot, oct, slot-1)
		lo := 40 + oct*slot
		for _, tick := range []int{2 * slot, 2*slot + 1, 100, 5000, 1 << 30} {
			frame, _, ok := terrain.SelectDeathFrame(a, 200, oct, tick)
			if !ok {
				t.Fatalf("octant %d tick %d: no frame", oct, tick)
			}
			if frame != want {
				t.Errorf("octant %d tick %d: frame %d, want the block's last, %d", oct, tick, frame, want)
			}
			if frame < lo || frame >= lo+slot {
				t.Errorf("octant %d tick %d: frame %d is outside its own slot [%d, %d)",
					oct, tick, frame, lo, lo+slot)
			}
		}
	}
}

// A negative count — which no caller can produce — is the first frame, not an
// index before the block. Go's division truncates toward zero, so this is the
// clamp's answer and not the division's.
func TestANegativeCountIsTheFirstFrame(t *testing.T) {
	a := dyingAnim(8, 4)
	first, _ := deathIndex(40, 8, 4, 3, 0)
	for _, tick := range []int{-1, -2, -3, -99} {
		if frame, _, ok := terrain.SelectDeathFrame(a, 200, 3, tick); !ok || frame != first {
			t.Errorf("tick %d: frame %d ok %v, want %d true", tick, frame, ok, first)
		}
	}
}

// The direction rule is the animated blocks' own, at both layouts (AC-5, SC-5).
func TestTheDeathSelectionTakesTheAnimatedBlocksDirectionRule(t *testing.T) {
	for _, d := range []int{8, 5} {
		a := dyingAnim(d, 2)
		for oct := 0; oct < 8; oct++ {
			want, wantMirror := deathIndex(40, d, 2, oct, 0)
			frame, mirror, ok := terrain.SelectDeathFrame(a, 200, oct, 0)
			if !ok {
				t.Fatalf("D %d octant %d: no frame", d, oct)
			}
			if frame != want || mirror != wantMirror {
				t.Errorf("D %d octant %d: frame %d mirror %v, want %d %v",
					d, oct, frame, mirror, want, wantMirror)
			}
		}
	}
	// At D 5 the upper octants fold and mirror; at D 8 nothing mirrors. Stated
	// as a count so the two rules cannot both be the same rule.
	mirrored := func(d int) (n int) {
		for oct := 0; oct < 8; oct++ {
			if _, m, _ := terrain.SelectDeathFrame(dyingAnim(d, 2), 200, oct, 0); m {
				n++
			}
		}
		return n
	}
	if got := mirrored(8); got != 0 {
		t.Errorf("%d of 8 octants mirror at D 8, want 0", got)
	}
	if got := mirrored(5); got != 3 {
		t.Errorf("%d of 8 octants mirror at D 5, want 3 — octants 5, 6 and 7", got)
	}
}

// The two refusals, and what each answers with (AC-5, SC-5).
func TestTheDeathSelectionRefusesWhatItCannotDraw(t *testing.T) {
	for _, tc := range []struct {
		label string
		anim  terrain.UnitAnim
		count int
		tick  int
	}{
		{"no dying block at all", dyingAnim(8, 0), 200, 0},
		{"an absent phase, clamped to no block", dyingAnim(8, 0), 200, 7},
		{"a sheet shorter than the index", dyingAnim(8, 4), 40, 0},
		{"a sheet of no frames", dyingAnim(8, 4), 0, 0},
		{"a negative frame count", dyingAnim(8, 4), -5, 0},
		{"a base below zero", terrain.UnitAnim{S: 16, D: 8, DyingBase: -9, DyingSlot: 2}, 200, 0},
	} {
		frame, mirror, ok := terrain.SelectDeathFrame(tc.anim, tc.count, 0, tc.tick)
		if ok {
			t.Errorf("%s: answered frame %d, want no frame", tc.label, frame)
		}
		if frame != 0 || mirror {
			t.Errorf("%s: a refusal carried frame %d mirror %v, want 0 false", tc.label, frame, mirror)
		}
	}

	// The last drawable index is answered and the first undrawable one is not,
	// so the guard is a half-open bound and not an off-by-one either way.
	a := dyingAnim(8, 4)
	last, _ := deathIndex(40, 8, 4, 7, 3)
	if _, _, ok := terrain.SelectDeathFrame(a, last+1, 7, 6); !ok {
		t.Errorf("a sheet of exactly %d frames refused index %d, its own last", last+1, last)
	}
	if _, _, ok := terrain.SelectDeathFrame(a, last, 7, 6); ok {
		t.Errorf("a sheet of %d frames answered index %d, which it does not hold", last, last)
	}
}

// Equal inputs give equal answers, and no input panics: the function reads
// nothing but its arguments.
func TestTheDeathSelectionIsTotalAndRepeatable(t *testing.T) {
	for _, d := range []int{0, 1, 5, 8, 16, -3} {
		for _, slot := range []int{-1, 0, 1, 5} {
			for oct := -2; oct < 10; oct++ {
				for _, tick := range []int{-7, 0, 1, 9, 1 << 20} {
					a := dyingAnim(d, slot)
					f1, m1, ok1 := terrain.SelectDeathFrame(a, 60, oct, tick)
					f2, m2, ok2 := terrain.SelectDeathFrame(a, 60, oct, tick)
					if f1 != f2 || m1 != m2 || ok1 != ok2 {
						t.Fatalf("D %d slot %d oct %d tick %d: two calls answered %d/%v/%v and %d/%v/%v",
							d, slot, oct, tick, f1, m1, ok1, f2, m2, ok2)
					}
					if ok1 && (f1 < 0 || f1 >= 60) {
						t.Fatalf("D %d slot %d oct %d tick %d: answered %d against a 60-frame sheet",
							d, slot, oct, tick, f1)
					}
				}
			}
		}
	}
}
