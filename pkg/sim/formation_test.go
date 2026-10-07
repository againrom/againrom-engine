package sim

// THE FORMATION ARM: the centroid, the one flag, the per-member offsets and the
// group's slowest member.
//
// Every number asserted here reaches the digest, so this file is the story's
// heaviest: the centroid's rounding, the spread gate's boundary, the offsets'
// two narrowings and the minimum's initial value are each pinned in a shape a
// mutation can be seen to break.
//
// Nothing here reads a game install.

import (
	"os"
	"strings"
	"testing"
)

// fmBounds is roomy enough that a formation never meets an edge unless a test
// puts it there on purpose.
var fmBounds = Bounds{Width: 60, Height: 60}

func fmMove(id EntityID, x, y int32) Command {
	return Command{Kind: KindGroupMoveTo, Entity: id, X: x, Y: y}
}

// TestTheCentroidIsTheMeanRoundedToTheNearestCell is the arithmetic every offset
// below is measured from, and it is asserted as a TABLE rather than through a
// distribution, so a wrong rounding is a wrong number here and not a shifted
// formation somewhere else.
//
// The rows are chosen so that a TRUNCATING mean disagrees with at least one of
// them in each direction: the two-member rows sit on an exact half, and the
// three-member rows on a third either side of an integer. The tie goes UP and
// not away from zero, which is the half-cell in the summands meeting a floor
// division and is the one place our own arithmetic reaches past the original's
// unsigned coordinates.
func TestTheCentroidIsTheMeanRoundedToTheNearestCell(t *testing.T) {
	cases := []struct {
		name  string
		cells []int32
		want  int32
	}{
		{"one member is its own centroid", []int32{7}, 7},
		{"an exact half rounds up", []int32{0, 1}, 1},
		{"an exact half rounds up further out", []int32{10, 13}, 12},
		{"a third below rounds down", []int32{1, 1, 2}, 1},
		{"two thirds above rounds up", []int32{1, 2, 2}, 2},
		{"an exact integer stays", []int32{4, 6}, 5},
		{"a negative exact half ties UP, toward zero", []int32{-1, 0}, 0},
		{"a negative third rounds to the nearer", []int32{-2, -2, -1}, -2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ents := make([]Entity, len(c.cells))
			members := make([]int, len(c.cells))
			for i, v := range c.cells {
				// The SAME cells on both axes: the two are one arithmetic, and
				// mirroring one of them would only assert that the rounding is
				// symmetric about zero, which it deliberately is not.
				ents[i] = Entity{ID: EntityID(i + 1), X: v, Y: v}
				members[i] = i
			}
			gotX, gotY := groupCentroid(ents, members)
			if gotX != c.want || gotY != c.want {
				t.Errorf("cells %v give a centroid of (%d,%d), want (%d,%d)",
					c.cells, gotX, gotY, c.want, c.want)
			}
		})
	}
}

// TestAThreeUnitRowKeepsItsShape is the owner's own case: a row of three boxed
// and sent far away takes THREE DISTINCT destinations, each the ordered cell
// plus that unit's own displacement from the centroid — and the row is still a
// row when they get there.
func TestAThreeUnitRowKeepsItsShape(t *testing.T) {
	w := mustWorld(t, 1, fmBounds, []Entity{
		{ID: 1, X: 4, Y: 8},
		{ID: 2, X: 5, Y: 8},
		{ID: 3, X: 6, Y: 8},
	})
	Step(w, []Command{fmMove(1, 40, 40), fmMove(2, 40, 40), fmMove(3, 40, 40)})

	want := [][2]int32{{39, 40}, {40, 40}, {41, 40}}
	for i, e := range w.Entities() {
		if e.TargetX != want[i][0] || e.TargetY != want[i][1] {
			t.Fatalf("id %d holds (%d,%d), want (%d,%d)", e.ID, e.TargetX, e.TargetY, want[i][0], want[i][1])
		}
	}

	for tick := 0; tick < 400; tick++ {
		Step(w, nil)
		done := true
		for _, e := range w.Entities() {
			done = done && !e.HasTarget
		}
		if done {
			break
		}
	}
	ents := w.Entities()
	for i, e := range ents {
		if e.HasTarget {
			t.Fatalf("id %d never arrived: at (%d,%d) holding (%d,%d)", e.ID, e.X, e.Y, e.TargetX, e.TargetY)
		}
		if e.X != want[i][0] || e.Y != want[i][1] {
			t.Errorf("id %d came to rest at (%d,%d), want (%d,%d) — the row did not survive the walk",
				e.ID, e.X, e.Y, want[i][0], want[i][1])
		}
	}
}

func TestOneFlagGatesBothTheDistributionAndTheTerm(t *testing.T) {
	cases := []struct {
		name string
		gap  int32
		want bool
	}{
		{"four cells apart", 4, true},
		{"five cells apart", 5, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := mustWorld(t, 1, fmBounds, []Entity{
				{ID: 1, X: 20, Y: 20, Speed: 30},
				{ID: 2, X: 20, Y: 20 + c.gap, Speed: 9},
			})
			Step(w, []Command{fmMove(1, 40, 40), fmMove(2, 40, 40)})

			ents := w.Entities()
			distributed := ents[1].TargetX != ents[0].TargetX || ents[1].TargetY != ents[0].TargetY
			termed := ents[0].GroupSpeed != 0

			if distributed != c.want || termed != c.want {
				t.Errorf("a gap of %d: distributed=%v termed=%v, want both %v — one flag gates both, "+
					"so they may never disagree", c.gap, distributed, termed, c.want)
			}
			for _, e := range ents {
				if got := e.GroupSpeed != 0; got != c.want {
					t.Errorf("id %d carries a term (%v) where the group's flag is %v", e.ID, got, c.want)
				}
			}
		})
	}
}

// TestADistributedDestinationIsClamped closes the case a distribution creates
// that a plain order cannot: the ordered cell is on the map and the OFFSET takes
// a member off it. The member is clamped, not dropped and not refused.
func TestADistributedDestinationIsClamped(t *testing.T) {
	w := mustWorld(t, 1, Bounds{Width: 10, Height: 10}, []Entity{
		{ID: 1, X: 4, Y: 4},
		{ID: 2, X: 6, Y: 4},
	})
	// The centroid is (5,4); offsets are -1 and +1, and the ordered cell is the
	// map's right edge, so id 2's own destination is one cell past it.
	Step(w, []Command{fmMove(1, 9, 4), fmMove(2, 9, 4)})

	ents := w.Entities()
	if ents[0].TargetX != 8 || ents[0].TargetY != 4 {
		t.Errorf("id 1 holds (%d,%d), want (8,4)", ents[0].TargetX, ents[0].TargetY)
	}
	if ents[1].TargetX != 9 || ents[1].TargetY != 4 {
		t.Errorf("id 2 holds (%d,%d), want (9,4) — a distributed destination off the map is clamped",
			ents[1].TargetX, ents[1].TargetY)
	}
}

func TestAGroupOfOneIsTheIdentity(t *testing.T) {
	for _, at := range [][2]int32{{0, 0}, {17, 3}, {59, 59}} {
		w := mustWorld(t, 1, fmBounds, []Entity{{ID: 1, X: at[0], Y: at[1], Speed: 23}})
		Step(w, []Command{fmMove(1, 30, 31)})

		got := w.Entities()[0]
		if got.TargetX != 30 || got.TargetY != 31 {
			t.Errorf("a group of one at (%d,%d) holds (%d,%d), want the ordered (30,31)",
				at[0], at[1], got.TargetX, got.TargetY)
		}
		if got.GroupSpeed != 23 {
			t.Errorf("a group of one at (%d,%d) carries a term of %d, want its own speed 23",
				at[0], at[1], got.GroupSpeed)
		}
	}
}

// TestADuplicateNamingIsCountedOnce measures the duplicate where double-counting
// would show — the centroid's divisor and the minimum. Naming the fast unit
// twice must move neither.
func TestADuplicateNamingIsCountedOnce(t *testing.T) {
	once := mustWorld(t, 1, fmBounds, []Entity{
		{ID: 1, X: 20, Y: 20, Speed: 30},
		{ID: 2, X: 21, Y: 20, Speed: 9},
	})
	twice := mustWorld(t, 1, fmBounds, []Entity{
		{ID: 1, X: 20, Y: 20, Speed: 30},
		{ID: 2, X: 21, Y: 20, Speed: 9},
	})
	Step(once, []Command{fmMove(1, 40, 40), fmMove(2, 40, 40)})
	Step(twice, []Command{fmMove(1, 40, 40), fmMove(1, 40, 40), fmMove(2, 40, 40)})

	if a, b := snap(once), snap(twice); !equalState(a, b) {
		t.Errorf("naming a member twice changed the order:\n %+v\nagainst\n %+v", b, a)
	}
	if got := twice.Entities()[0].GroupSpeed; got != 9 {
		t.Errorf("the group term is %d, want the slower member's 9", got)
	}
}

// TestTheGroupTermIsTheMinimumInTheOriginalsOwnWidths pins the loop rather than
// the idea of a minimum: its initial value bounds the answer, its comparison is
// strict, its input is a signed 16-bit read and its output a low byte. The last
// three rows are unreachable from the shipped speed column and reachable by
// customising it, which is why they are here.
func TestTheGroupTermIsTheMinimumInTheOriginalsOwnWidths(t *testing.T) {
	cases := []struct {
		name   string
		speeds []int32
		want   uint8
	}{
		{"the slowest member wins", []int32{30, 9, 19}, 9},
		{"equal speeds are that speed", []int32{19, 19}, 19},
		{"every member faster than the initial value keeps it", []int32{300, 400}, groupSpeedInit},
		{"a speed at the initial value does not displace it", []int32{groupSpeedInit, 300}, groupSpeedInit},
		{"one below the initial value does", []int32{groupSpeedInit - 1, 300}, groupSpeedInit - 1},
		{"a speed past sixteen bits is compared at its low half", []int32{65636}, 100},
		{"a negative speed becomes the unsigned low byte", []int32{-1}, 255},
		{"and a slower member still displaces that", []int32{-1, 19}, 19},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ents := make([]Entity, len(c.speeds))
			members := make([]int, len(c.speeds))
			for i, s := range c.speeds {
				ents[i] = Entity{ID: EntityID(i + 1), Speed: s}
				members[i] = i
			}
			if got := groupMinSpeed(ents, members); got != c.want {
				t.Errorf("speeds %v give %d, want %d", c.speeds, got, c.want)
			}
		})
	}
}

// TestAFormationCrossesAtItsSlowestMembersRate is the term reaching a walk. Both
// members of a mixed pair cross at the slow one's transit length in formation,
// and at their own out of it.
//
// BOTH FIXTURES ARE BUILT SO THAT EVERY FIRST STEP IS STRAIGHT. A transit is
// longer on a diagonal, so a mover that detoured round a companion would report
// a length this test would read as a different rate — the measurement would be
// of the near search and not of the rate at all. In formation the pair walks two
// ADJACENT COLUMNS, each straight down its own; out of formation it walks ONE
// column six cells apart, the faster one in front and pulling away.
func TestAFormationCrossesAtItsSlowestMembersRate(t *testing.T) {
	const slow, fast = 8, 40

	slowTransit := transitOf(rateOf(DomainGround, slow, 0, 0, 0, 0), false)
	fastTransit := transitOf(rateOf(DomainGround, fast, 0, 0, 0, 0), false)
	if slowTransit == fastTransit {
		t.Fatalf("the two speeds give the same transit of %d — the fixture separates nothing", slowTransit)
	}

	// Two adjacent columns: the centroid is the right-hand one, the offsets are
	// -1 and 0, and each member's own destination is straight below it.
	in := mustWorld(t, 1, fmBounds, []Entity{
		{ID: 1, X: 20, Y: 20, Speed: slow},
		{ID: 2, X: 21, Y: 20, Speed: fast},
	})
	Step(in, []Command{fmMove(1, 21, 50), fmMove(2, 21, 50)})
	for _, e := range in.Entities() {
		if e.GroupSpeed != slow {
			t.Fatalf("in formation id %d carries a term of %d, want the slow member's %d",
				e.ID, e.GroupSpeed, slow)
		}
		if int32(e.TransitTotal) != slowTransit {
			t.Errorf("in formation id %d crossed in %d tick(s), want the slow member's %d",
				e.ID, e.TransitTotal, slowTransit)
		}
	}

	// One column, six cells apart: nobody is within the spread threshold of the
	// centroid, so there is no distribution and no term.
	out := mustWorld(t, 1, fmBounds, []Entity{
		{ID: 1, X: 20, Y: 20, Speed: slow},
		{ID: 2, X: 20, Y: 26, Speed: fast},
	})
	Step(out, []Command{fmMove(1, 20, 50), fmMove(2, 20, 50)})
	ents := out.Entities()
	for _, e := range ents {
		if e.GroupSpeed != 0 {
			t.Fatalf("out of formation id %d carries a term of %d, want none", e.ID, e.GroupSpeed)
		}
	}
	if int32(ents[0].TransitTotal) != slowTransit || int32(ents[1].TransitTotal) != fastTransit {
		t.Errorf("out of formation the pair crossed in %d and %d tick(s), want %d and %d — "+
			"each at its own rate", ents[0].TransitTotal, ents[1].TransitTotal, slowTransit, fastTransit)
	}
}

// TestTheSurvivorsKeepADeadMembersRate is the defect at its sharpest, and it is
// the thing the owner is asked to look at. A slow unit is killed mid-walk and its
// companion goes on crawling at its speed for the rest of the order; a new plain
// order restores its own.
//
// The pair walks two adjacent columns for the reason the rate test does: what is
// measured is a transit length, and a detour would lengthen one.
func TestTheSurvivorsKeepADeadMembersRate(t *testing.T) {
	const slow, fast = 8, 40

	w := mustWorld(t, 1, fmBounds, []Entity{
		{ID: 1, X: 20, Y: 20, Speed: slow, HP: 10, MaxHP: 10},
		{ID: 2, X: 21, Y: 20, Speed: fast, HP: 10, MaxHP: 10},
	})
	Step(w, []Command{fmMove(1, 21, 50), fmMove(2, 21, 50)})
	Step(w, []Command{{Kind: KindKill, Entity: 1}})

	survivor := w.Entities()[1]
	if survivor.GroupSpeed != slow {
		t.Fatalf("the survivor carries a term of %d, want the dead member's %d", survivor.GroupSpeed, slow)
	}
	if dead := w.Entities()[0]; dead.GroupSpeed != 0 {
		t.Fatalf("the dead member carries a term of %d, want none — death unlinks the MEMBER, "+
			"and it is the group's byte that survives", dead.GroupSpeed)
	}

	slowTransit := transitOf(rateOf(DomainGround, slow, 0, 0, 0, 0), false)
	fastTransit := transitOf(rateOf(DomainGround, fast, 0, 0, 0, 0), false)
	for tick := 0; tick < 4*int(slowTransit); tick++ {
		Step(w, nil)
		e := w.Entities()[1]
		if e.TransitTotal != 0 && int32(e.TransitTotal) != slowTransit {
			t.Fatalf("at tick %d the survivor crossed in %d tick(s), want the dead member's %d "+
				"(its own would be %d)", tick, e.TransitTotal, slowTransit, fastTransit)
		}
	}

	Step(w, []Command{{Entity: 2, X: 21, Y: 20}})
	if got := w.Entities()[1]; got.GroupSpeed != 0 {
		t.Errorf("after a plain order the survivor still carries a term of %d — a new order is a "+
			"fresh group, and a fresh group carries none", got.GroupSpeed)
	}
}

func TestTheGroupOrderHasNoMemberCountBranch(t *testing.T) {
	src, err := os.ReadFile("group.go")
	if err != nil {
		t.Fatalf("reading the group order's own source: %v", err)
	}
	const needle = "len(members)"
	if got := strings.Count(string(src), needle); got != 3 {
		t.Errorf("group.go mentions %s %d time(s), and exactly 3 are accounted for: "+
			"issueGroupDestination's and commandGroup's own empty-order guards, and the "+
			"centroid's divisor. A fourth is a member-count arm, which FR-8 says may not exist",
			needle, got)
	}
}

// TestTheFormationOffsetNarrowsToASignedByte exercises the pair of widths the
// order carries an offset through, DIRECTLY, because nothing an order can be
// given reaches them: the spread gate admits displacements in [-2, +2] only, and
// both narrowings are transparent over all five.
//
// They are exercised anyway because they are decoded — the offset is stored 16
// bits wide and read back a byte — and because they are the story's own
// customisation limit: under a formation mode this tree does not model, a member
// standing 128 cells from its centroid is sent in the opposite direction.
func TestTheFormationOffsetNarrowsToASignedByte(t *testing.T) {
	cases := []struct {
		delta, want int32
	}{
		{0, 0}, {2, 2}, {-2, -2},
		{127, 127},          // the last displacement a byte holds
		{128, -128},         // and the first that wraps
		{-129, 127},         // the other end of the same wrap
		{255, -1}, {256, 0}, // one whole byte on
		{32768, 0}, {32767, -1}, // and where the 16-bit field itself wraps
		{-32768, 0},
	}
	for _, c := range cases {
		if got := formationOffset(c.delta); got != c.want {
			t.Errorf("a displacement of %d offsets by %d, want %d", c.delta, got, c.want)
		}
	}
}
