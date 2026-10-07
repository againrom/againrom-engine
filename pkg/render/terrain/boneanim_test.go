package terrain

// The bone selection (0089 AC-12, AC-13): the block a body is drawn from
// once its fall is over.

import "testing"

// bnAnim is a descriptor with every block named and a tail base a reader can
// check by hand: S 16, D 8, so the tail sits at 16 + 8*(MB+MV+AT+DY).
func bnAnim(d, bone int) UnitAnim {
	return UnitAnim{S: 16, D: d, TailBase: 100, BoneSlot: bone, DyingSlot: 3, DyingBase: 60}
}

func TestTheBoneFrameIsTheDirectionsOwnSlot(t *testing.T) {
	t.Parallel()

	const frames = 1000
	a := bnAnim(8, 5)
	seen := map[int]bool{}
	for oct := 0; oct < 8; oct++ {
		for stage, want := range map[int]int{2: 0, 3: 1, 4: 2} {
			got, mirror, ok := SelectBoneFrame(a, frames, oct, stage)
			if !ok {
				t.Fatalf("octant %d stage %d: no frame", oct, stage)
			}
			if mirror {
				t.Errorf("octant %d: mirrored at the eight-way layout", oct)
			}
			if wantFrame := a.TailBase + oct*a.BoneSlot + want; got != wantFrame {
				t.Errorf("octant %d stage %d: frame %d, want %d", oct, stage, got, wantFrame)
			}
			lo := a.TailBase + oct*a.BoneSlot
			if got < lo || got >= lo+a.BoneSlot {
				t.Errorf("octant %d stage %d: frame %d is outside [%d,%d)",
					oct, stage, got, lo, lo+a.BoneSlot)
			}
			seen[got] = true
		}
	}
	if len(seen) != 24 {
		t.Errorf("the eight octants at three stages gave %d distinct frames, want 24 — "+
			"an index with no direction term answers the same three for all of them", len(seen))
	}

	// The five-way layout: the upper octants fold onto the lower slots and draw
	// reflected, which is the animated blocks' own rule and not a second one.
	b := bnAnim(5, 4)
	for oct := 0; oct <= 4; oct++ {
		got, mirror, ok := SelectBoneFrame(b, frames, oct, 2)
		if !ok || mirror || got != b.TailBase+oct*b.BoneSlot {
			t.Errorf("octant %d: frame %d mirror %v ok %v, want %d unmirrored",
				oct, got, mirror, ok, b.TailBase+oct*b.BoneSlot)
		}
	}
	for oct := 5; oct < 8; oct++ {
		got, mirror, ok := SelectBoneFrame(b, frames, oct, 2)
		if !ok || !mirror || got != b.TailBase+(8-oct)*b.BoneSlot {
			t.Errorf("octant %d: frame %d mirror %v ok %v, want %d mirrored",
				oct, got, mirror, ok, b.TailBase+(8-oct)*b.BoneSlot)
		}
	}
}

func TestTheBoneSelectionRefusesWhatItCannotDraw(t *testing.T) {
	t.Parallel()

	a := bnAnim(8, 5)
	for _, tc := range []struct {
		name   string
		anim   UnitAnim
		frames int
		oct    int
		stage  int
	}{
		{"stage 0, which is a living unit", a, 1000, 0, 0},
		{"stage 1, which is still the fall", a, 1000, 0, 1},
		{"a negative stage", a, 1000, 0, -3},
		{"no bone block at all", bnAnim(8, 0), 1000, 0, 3},
		{"an absent phase count, clamped to none", bnAnim(8, -1), 1000, 0, 3},
		{"a sheet that cannot hold the index", a, 100, 7, 4},
		{"a sheet with no frames", a, 0, 0, 2},
		{"an octant past the eight", a, 100, 40, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frame, mirror, ok := SelectBoneFrame(tc.anim, tc.frames, tc.oct, tc.stage)
			if ok {
				t.Errorf("answered frame %d (mirror %v)", frame, mirror)
			}
			if frame != 0 || mirror {
				t.Errorf("a refusal answered frame %d mirror %v, want 0 and false", frame, mirror)
			}
		})
	}
}

func TestTheBoneSelectionIsPure(t *testing.T) {
	t.Parallel()

	a := bnAnim(5, 3)
	for oct := -2; oct < 10; oct++ {
		for stage := -2; stage < 8; stage++ {
			f1, m1, o1 := SelectBoneFrame(a, 200, oct, stage)
			f2, m2, o2 := SelectBoneFrame(a, 200, oct, stage)
			if f1 != f2 || m1 != m2 || o1 != o2 {
				t.Fatalf("octant %d stage %d answered twice as (%d,%v,%v) and (%d,%v,%v)",
					oct, stage, f1, m1, o1, f2, m2, o2)
			}
		}
	}
}
