package sim

// The two budget rules, driven directly, and the pins this revision owes.
//
// The far search's budget used to be a function of the straight line: the whole
// allowance for LEAVING that line was a quarter of the distance ALONG it, so a
// route that went round rather than through failed for being round. It is a flat
// thousand now, and the near search keeps the scaled form, because a search
// aimed at a sub-goal four cells off is exactly where a bound shaped like the
// straight line is the right shape.
//
// Both fixtures here are small enough to read. What they show is a difference
// between the two RULES on one world, so neither leans on a map big enough that
// the answer has to be taken on trust.

import "testing"

// ------------------------------------------------- the rule the far search took

// pocketBounds is a three by nine with a column of eight blocked cells and one
// gap at the bottom, and it is the whole argument in one picture.
//
//	. # .        The target is (2,0), TWO cells from the start. The only walk
//	. # .        to it is down one side, through the gap and back up the other:
//	. # .        sixteen steps, so sixteen generations.
//	. # .
//	. # .        The scaled rule allows max(3, 2>>2) + 2 = 5. The flat rule
//	. # .        allows a thousand. Nothing about the terrain, the relation or
//	. # .        the goal differs between the two calls below - only the rule -
//	. # .        so the refusal was never the map's.
//	. . .
var pocketBounds = Bounds{Width: 3, Height: 9}

func pocketGrid() []byte {
	g := make([]byte, pocketBounds.Width*pocketBounds.Height)
	for y := int32(0); y < 8; y++ {
		g[y*pocketBounds.Width+1] = blockGround
	}
	return g
}

func TestTheFlatBudgetReachesAGoalTheScaledOneCannot(t *testing.T) {
	start, target := cell{0, 0}, cell{2, 0}

	if got := generationBudget(start, target, scaledBudget, true); got != 5 {
		t.Fatalf("the scaled budget for this fixture is %d, want 5", got)
	}
	if got := generationBudget(start, target, flatBudget, true); got != 1000 {
		t.Fatalf("the flat budget is %d, want 1000", got)
	}

	w := routeWorld(t, pocketBounds, pocketGrid(), []Entity{{ID: 1, X: start.x, Y: start.y}})
	s := newRouteScratch(w)

	if route, ok := w.canonicalRoute(s, 0, terrainRelation, noWindow, scaledBudget, exactGoal, target.x, target.y); ok {
		t.Errorf("the scaled rule found a route it has five generations for: %s", fmtRoute(route))
	}

	route, ok := w.canonicalRoute(s, 0, terrainRelation, noWindow, flatBudget, exactGoal, target.x, target.y)
	if !ok {
		t.Fatal("the flat rule found no route through the gap")
	}
	if len(route) != 16 {
		t.Errorf("the walk round the column is %d hops, want 16: %s", len(route), fmtRoute(route))
	}
	if last := route[len(route)-1]; last != target {
		t.Errorf("the route ends at (%d,%d), want the target", last.x, last.y)
	}
}

// TestTheFlatBudgetIsFlat is what makes the far search's bound independent of
// how far the order pointed: the same number at every distance, including the
// two where the scaled rule would be the LARGER of the pair.
func TestTheFlatBudgetIsFlat(t *testing.T) {
	for _, target := range []cell{{0, 0}, {8, 3}, {255, 255}, {800, 0}, {900, 0}} {
		if got := generationBudget(cell{0, 0}, target, flatBudget, true); got != flatGenerations {
			t.Errorf("the flat budget to (%d,%d) is %d, want %d", target.x, target.y, got, flatGenerations)
		}
	}
	// The crossover is a real bound and not a formality: at 900 the scaled rule
	// would allow more than the flat one does.
	if scaled := generationBudget(cell{0, 0}, cell{900, 0}, scaledBudget, true); scaled <= flatGenerations {
		t.Errorf("the scaled budget at D=900 is %d, and this row exists because it exceeds %d",
			scaled, flatGenerations)
	}
}

// ------------------------------------------------- the channel

// The channel fixture: a forty by thirty-two split top to bottom by a wall at
// x = 20, open only across the last two rows, and a unit ordered from one side
// to the other.
//
//	@ . . . # . . . T        The straight line between (0,0) and (39,0) is 39
//	. . . . # . . . .        cells. The only land route goes down to row 30,
//	. . . . # . . . .        across, and back up: 62 hops, so 23 rings more
//	. . . . # . . . .        than the straight line.
//	. . . . # . . . .
//	. . . . . . . . .        The old allowance for leaving that line was
//	. . . . . . . . .        max(5, 39>>2) = 9 rings, and 23 is not 9.
//
// It is defined here because AC-14's recorded run below needs a world whose far
// search fails on the tree before this change; channel_test.go asserts the
// fixture's own claims and walks a unit across it.
var chanBounds = Bounds{Width: 40, Height: 32}

const chanWallX int32 = 20
const chanOpenFrom int32 = 30
const chanTargetX, chanTargetY int32 = 39, 0

func chanGrid() []byte {
	g := make([]byte, chanBounds.Width*chanBounds.Height)
	for y := int32(0); y < chanOpenFrom; y++ {
		g[y*chanBounds.Width+chanWallX] = blockGround
	}
	return g
}

// ------------------------------------------------- AC-14

// acFourteenDigests is the run recorded from the tree BEFORE the budget moved:
// the channel world, canonical mode, seed 1, one unit at (0,0) ordered to
// (39,0) on tick 1 and advanced eight ticks.
//
// On that tree the far search found nothing, the order was cleared in the first
// tick, and the unit never moved. These are LITERALS, pasted from that run and
// never regenerated from the tree they judge — a pin recomputed from the build
// it is meant to catch is not a pin. They are expected to be WRONG now, at every
// tick, which is the whole assertion.
//
// The digests move from tick to tick even on the old tree, because a world's
// generator advances whether or not anything walks. So "they differ" is not by
// itself interesting; what is interesting is that they differ from the FIRST
// tick, and that the unit that stood still now walks.
var acFourteenDigests = [8]uint64{
	0xd60bee51854cd8e5,
	0xc9314265087767f4,
	0x6669ec2f0513a75b,
	0x6b3b3225daa2e452,
	0x0873dbefd73f23b9,
	0xfb9930035a69b2c8,
	0x98d1d9cd5705f22f,
	0xa2f63ff4ae1d36a6,
}

// TestTheRecordedRunDiffersFromTheFirstTick is AC-14's second half: one world
// advanced by the same commands on a build before this revision and on this one,
// differing exactly where a far search used to fail.
func TestTheRecordedRunDiffersFromTheFirstTick(t *testing.T) {
	w := mustWorldGrid(t, 1, chanBounds, ModeCanonical, chanGrid(), []Entity{{ID: 1, X: 0, Y: 0}})

	for tick := 1; tick <= len(acFourteenDigests); tick++ {
		if tick == 1 {
			Step(w, []Command{{Entity: 1, X: chanTargetX, Y: chanTargetY}})
		} else {
			Step(w, nil)
		}
		if got := w.Hash(); got == acFourteenDigests[tick-1] {
			t.Errorf("tick %d hashes 0x%016x, which is what the tree before this change hashed. "+
				"The far search that used to fail here is meant to succeed now", tick, got)
		}
	}

	// The unit walked where it stood. On the old tree it was at (0,0) with its
	// order cleared on tick 1; here it holds the order and has moved off.
	e := w.Entities()[0]
	if e.X == 0 && e.Y == 0 {
		t.Errorf("the unit is still at (0,0) as %+v — it stood still on the old tree and is meant to walk", e)
	}
	if !e.HasTarget {
		t.Errorf("the unit is %+v with no target after 8 ticks, so its order was cleared as it used to be", e)
	}
	t.Logf("after 8 ticks the unit is at (%d,%d) holding %d route cell(s), where it stood at (0,0)",
		e.X, e.Y, len(w.routes[0]))
}

// TestTheByteFormsVersionIsUnmovedByThisRevision is AC-14's first half, and it
// is stated as what it is.
//
// THE VERSION LITERAL IS GONE, and its removal is this merge's, not either
// story's. It read "the version this revision inherited and left alone" and had
// to be edited by every unrelated bump since — 0089 and 0091 each changed it, in
// parallel, and it was one of the sixteen files their merge collided in. A
// literal cannot say "this revision moved no byte": it fails when SOMEBODY ELSE
// legitimately takes the next version, and it holds when a story adds a field and
// forgets to bump. What witnesses the claim is the check below — the form this
// world marshals to opens at whatever version the build writes — plus the byte
// forms and digests pinned elsewhere in the package, which move if any byte does.
//
// The rest of AC-14's first half is every other byte form and digest this tree
// pins, which is the rest of this package's suite and is not restated here.
func TestTheByteFormsVersionIsUnmovedByThisRevision(t *testing.T) {
	w := mustWorldGrid(t, 1, chanBounds, ModeCanonical, chanGrid(), []Entity{{ID: 1, X: 0, Y: 0}})
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("marshalling the channel world: %v", err)
	}
	if form[0] != formatVersion {
		t.Errorf("the form's version byte is %d, want %d", form[0], formatVersion)
	}
}
