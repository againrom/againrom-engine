package terrain_test

// The object layer's step and frame selector (0031 AC-2, AC-4, AC-9's
// selector half, SC-2).
//
// Both functions are pure integer arithmetic over their arguments, so every
// fixture here is a literal and every expectation is COMPUTED FROM THE CONTRACT
// INSIDE THE TEST — never read back from the function under test. The step's
// expectation in particular is written as the spec's own expression, with the
// operands in the spec's own order, so a stagger transposed onto water's
// (col+1)*row disagrees with it at every cell off the diagonal.

import (
	"testing"

	"againrom/pkg/render/terrain"
)

func objectStepWant(col, row int, counter uint32, period int) int {
	s := (int(counter) + col*(row+1)) % period
	if s < 0 {
		s += period
	}
	return s
}

// objectTimeline is AC-4's cycle: seven fours over the values 0..6, the shipped
// shape, 28 steps long. Written out by hand for the same reason the data tier's
// expansion test writes it out.
func objectTimeline() []int {
	var t []int
	for v := 0; v <= 6; v++ {
		for i := 0; i < 4; i++ {
			t = append(t, v)
		}
	}
	return t
}

// TestObjectStepMatchesTheContract is AC-2: four cells over a period of 28, at
// every counter 0..31 — one full cycle and four steps past its wrap.
//
// The four cells are not four samples. (0,0) has no offset at all; (1,0) and
// (0,1) are the transposition pair, which is the whole reason this test exists;
// (3,5) is an interior cell whose offset exceeds the period, so the reduction is
// exercised rather than assumed.
func TestObjectStepMatchesTheContract(t *testing.T) {
	const period = 28
	cells := [][2]int{{0, 0}, {1, 0}, {0, 1}, {3, 5}}

	for _, c := range cells {
		col, row := c[0], c[1]
		for counter := uint32(0); counter < 32; counter++ {
			want := objectStepWant(col, row, counter, period)
			got := terrain.ObjectStep(col, row, counter, period)
			if got != want {
				t.Errorf("ObjectStep(%d, %d, %d, %d) = %d, want %d",
					col, row, counter, period, got, want)
			}
			// Two evaluations of one input agree: the function holds nothing between
			// calls.
			if again := terrain.ObjectStep(col, row, counter, period); again != got {
				t.Errorf("ObjectStep(%d, %d, %d, %d) = %d then %d — not a function",
					col, row, counter, period, got, again)
			}
		}
	}
}

// TestObjectStepStaggerIsNotTransposed is SC-2's second half, stated as its own
// fact: (1,0) and (0,1) select DIFFERENT steps at every counter.
//
// A transposed stagger — water's (col+1)*row — swaps exactly these two
// cells, so this assertion alone does not catch it; the expectation in the
// test above does.
func TestObjectStepStaggerIsNotTransposed(t *testing.T) {
	const period = 28
	for counter := uint32(0); counter < 64; counter++ {
		a := terrain.ObjectStep(1, 0, counter, period)
		b := terrain.ObjectStep(0, 1, counter, period)
		if a == b {
			t.Fatalf("counter %d: cells (1,0) and (0,1) both step to %d — the stagger does not separate them",
				counter, a)
		}
	}
}

// TestObjectStepIsTotal is AC-9's share of the step: a period of 0, a period of
// 1, the counter at its maximum, and a cell at the origin. Nothing divides by
// zero and every answer lies inside its own period.
func TestObjectStepIsTotal(t *testing.T) {
	if got := terrain.ObjectStep(3, 5, 100, 0); got != 0 {
		t.Errorf("ObjectStep at period 0 = %d, want 0 — no division", got)
	}
	if got := terrain.ObjectStep(3, 5, 100, -7); got != 0 {
		t.Errorf("ObjectStep at a negative period = %d, want 0", got)
	}
	if got := terrain.ObjectStep(3, 5, 100, 1); got != 0 {
		t.Errorf("ObjectStep at period 1 = %d, want 0 — the only step there is", got)
	}

	const maxCounter = ^uint32(0)
	for _, c := range [][2]int{{0, 0}, {255, 255}, {3, 5}} {
		got := terrain.ObjectStep(c[0], c[1], maxCounter, 28)
		if got < 0 || got >= 28 {
			t.Errorf("ObjectStep(%d, %d, %d, 28) = %d, outside [0, 28)", c[0], c[1], maxCounter, got)
		}
		if want := objectStepWant(c[0], c[1], maxCounter, 28); got != want {
			t.Errorf("ObjectStep(%d, %d, max, 28) = %d, want %d", c[0], c[1], got, want)
		}
	}
}

func TestSelectObjectFrameArms(t *testing.T) {
	const (
		index      = 3
		frameCount = 12
	)
	tl := objectTimeline()
	if len(tl) != 28 {
		t.Fatalf("fixture timeline has %d steps, want 28", len(tl))
	}

	for _, counter := range []uint32{0, 1, 5, 13, 27, 28, 100} {
		col, row := 10, 6

		// Arm 2: the switch on and the cycle open.
		want := index + tl[objectStepWant(col, row, counter, len(tl))]
		got := terrain.SelectObjectFrame(tl, index, frameCount, col, row, counter, true, true)
		if got != want {
			t.Errorf("open cycle at counter %d: frame %d, want %d", counter, got, want)
		}

		if got := terrain.SelectObjectFrame(tl, index, frameCount, col, row, counter, true, false); got != index {
			t.Errorf("closed cycle at counter %d: frame %d, want %d", counter, got, index)
		}

		// Arm 1: the switch off — sheet frame 0, open or closed alike.
		for _, open := range []bool{false, true} {
			if got := terrain.SelectObjectFrame(tl, index, frameCount, col, row, counter, false, open); got != 0 {
				t.Errorf("animation off at counter %d (open=%v): frame %d, want 0", counter, open, got)
			}
		}
	}
}

// TestSelectObjectFrameOutOfRangeDrawsFrameZero is AC-4's fourth clause and
// C-3: a timeline naming a value that carries the selection past the sheet
// answers sheet frame 0 — never nothing, never a panic.
//
// The plain arm is checked at the same refusal, because the guard is the
// selector's last act and must judge every arm rather than the animated one
// alone.
func TestSelectObjectFrameOutOfRangeDrawsFrameZero(t *testing.T) {
	const frameCount = 12

	// A one-step timeline naming 40: 3 + 40 = 43, far outside a 12-frame sheet.
	if got := terrain.SelectObjectFrame([]int{40}, 3, frameCount, 1, 1, 0, true, true); got != 0 {
		t.Errorf("out-of-range cycle selection = %d, want 0", got)
	}
	// The plain arm, out of range on its own.
	if got := terrain.SelectObjectFrame(nil, 40, frameCount, 1, 1, 0, true, false); got != 0 {
		t.Errorf("out-of-range Index = %d, want 0", got)
	}
	// A negative Index — pkg/data's absent-everywhere default is -1, so this is
	// the shape a class omitting the key arrives in, not a defensive case.
	if got := terrain.SelectObjectFrame(nil, -1, frameCount, 1, 1, 0, true, false); got != 0 {
		t.Errorf("negative Index = %d, want 0", got)
	}
	// A sheet of no frames: every index is outside it, including 0.
	if got := terrain.SelectObjectFrame(nil, 0, 0, 1, 1, 0, true, false); got != 0 {
		t.Errorf("frame at frameCount 0 = %d, want 0", got)
	}
}

func TestSelectObjectFrameHasNoFourthArm(t *testing.T) {
	const (
		index      = 5
		frameCount = 12
	)
	for _, tl := range [][]int{nil, {}} {
		for _, counter := range []uint32{0, 7, ^uint32(0)} {
			if got := terrain.SelectObjectFrame(tl, index, frameCount, 4, 9, counter, true, true); got != index {
				t.Errorf("empty timeline, gate open, counter %d: frame %d, want %d", counter, got, index)
			}
			if got := terrain.SelectObjectFrame(tl, index, frameCount, 4, 9, counter, false, true); got != 0 {
				t.Errorf("empty timeline, gate open, animation off, counter %d: frame %d, want 0", counter, got)
			}
		}
	}
}

func TestSelectObjectFrameIsAFunction(t *testing.T) {
	tl := objectTimeline()
	for _, animate := range []bool{false, true} {
		for _, open := range []bool{false, true} {
			for counter := uint32(0); counter < 30; counter++ {
				a := terrain.SelectObjectFrame(tl, 3, 12, 10, 6, counter, animate, open)
				b := terrain.SelectObjectFrame(tl, 3, 12, 10, 6, counter, animate, open)
				if a != b {
					t.Fatalf("animate=%v open=%v counter=%d: %d then %d", animate, open, counter, a, b)
				}
			}
		}
	}
}

func TestSelectObjectFrameCellsDisagree(t *testing.T) {
	tl := objectTimeline()
	differed := false
	for counter := uint32(0); counter < 28 && !differed; counter++ {
		a := terrain.SelectObjectFrame(tl, 0, 12, 1, 0, counter, true, true)
		b := terrain.SelectObjectFrame(tl, 0, 12, 0, 1, counter, true, true)
		differed = a != b
	}
	if !differed {
		t.Error("cells (1,0) and (0,1) selected the same frame at every counter of one period")
	}
}
