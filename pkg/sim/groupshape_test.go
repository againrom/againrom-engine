package sim

// What a group order IS, before formation forks it: which commands belong to
// one, which entities become its members, which cell it was given, and where a
// destination goes when it falls off the map.
//
// The tests here keep the group out of formation wherever the arm would confuse
// the measurement, so that what they measure is the order's shape and not the
// distribution. The distribution has its own file.
//
// Nothing here reads a game install.

import "testing"

// gsBounds is wide enough that a spread-out group is still on the map, so the
// spread gate and the clamp can be exercised without either standing in for the
// other.
var gsBounds = Bounds{Width: 40, Height: 40}

// gsMove is one member's share of a group order: the kind, the entity, the
// clicked cell and the tag.
func gsMove(id EntityID, x, y int32, tag uint32) Command {
	return Command{Kind: KindGroupMoveTo, Entity: id, X: x, Y: y, Group: tag}
}

// gsSpread is three units far enough apart to fail the spread gate, so every one
// of them takes the ordered cell unchanged and the arithmetic under test is the
// order's and not the formation's.
func gsSpread() []Entity {
	return []Entity{
		{ID: 1, X: 0, Y: 0},
		{ID: 2, X: 10, Y: 0},
		{ID: 3, X: 20, Y: 0},
	}
}

// TestOneTagIsOneOrderAndTwoTagsAreTwo is the tag's whole contract. Three
// commands under one tag are one order; a fourth under another tag is a second
// order, applied at its own position, and where the two name the same entity the
// later one wins — the same rule two single orders in one slice already obey.
func TestOneTagIsOneOrderAndTwoTagsAreTwo(t *testing.T) {
	w := mustWorld(t, 1, gsBounds, gsSpread())
	Step(w, []Command{
		gsMove(1, 30, 30, 7), gsMove(2, 30, 30, 7), gsMove(3, 30, 30, 7),
		gsMove(1, 5, 6, 9),
	})

	ents := w.Entities()
	if got := ents[0]; got.TargetX != 5 || got.TargetY != 6 {
		t.Errorf("id 1 holds (%d,%d), want (5,6) — the later tag's order must win",
			got.TargetX, got.TargetY)
	}
	for _, e := range ents[1:] {
		if e.TargetX != 30 || e.TargetY != 30 {
			t.Errorf("id %d holds (%d,%d), want (30,30)", e.ID, e.TargetX, e.TargetY)
		}
	}
}

// TestAGroupOrderTakesTheFirstCommandsCell pins that one destination reaches the
// whole group. A caller that varied the cell per command would be handing in an
// order no click can produce, and the members it named would still walk to the
// one cell the order carries.
func TestAGroupOrderTakesTheFirstCommandsCell(t *testing.T) {
	w := mustWorld(t, 1, gsBounds, gsSpread())
	Step(w, []Command{gsMove(1, 30, 31, 0), gsMove(2, 3, 4, 0), gsMove(3, 8, 9, 0)})

	for _, e := range w.Entities() {
		if e.TargetX != 30 || e.TargetY != 31 {
			t.Errorf("id %d holds (%d,%d), want the first command's (30,31)", e.ID, e.TargetX, e.TargetY)
		}
	}
}

// TestAnAbsentOrFelledMemberIsSkippedAndTheRestAreOrdered is what "ignored, not
// an error" means for a set: the order is still the order the surviving members
// were given, and it is issued even when the command that named the FIRST member
// is one of the ignored ones — which is why the group arm stands before the
// entity lookup.
func TestAnAbsentOrFelledMemberIsSkippedAndTheRestAreOrdered(t *testing.T) {
	w := mustWorld(t, 1, gsBounds, []Entity{
		{ID: 2, X: 0, Y: 0, HP: -1, MaxHP: 10},
		{ID: 3, X: 10, Y: 0},
		{ID: 4, X: 20, Y: 0},
	})
	// Id 1 is in no world at all, and it is the command the order begins at.
	Step(w, []Command{gsMove(1, 30, 30, 0), gsMove(2, 30, 30, 0), gsMove(3, 30, 30, 0), gsMove(4, 30, 30, 0)})

	ents := w.Entities()
	if ents[0].HasTarget {
		t.Errorf("the felled member holds (%d,%d) — a unit that is not alive takes no order",
			ents[0].TargetX, ents[0].TargetY)
	}
	for _, e := range ents[1:] {
		if !e.HasTarget || e.TargetX != 30 || e.TargetY != 30 {
			t.Errorf("id %d holds target %v (%d,%d), want (30,30)", e.ID, e.HasTarget, e.TargetX, e.TargetY)
		}
	}
}

// A duplicate naming is measured where double-counting would show — the
// centroid's divisor and the minimum — so its test lives beside the arm that
// computes them, in formation_test.go.

// TestAGroupDestinationIsClampedIntoTheMap covers both ends of both axes. A
// destination is clamped and never refused: an order off the map is an order to
// the nearest cell on it, which is what the routine that issues one member's
// destination does in the original.
func TestAGroupDestinationIsClampedIntoTheMap(t *testing.T) {
	cases := []struct {
		name         string
		x, y         int32
		wantX, wantY int32
	}{
		{"below both", -50, -1, 0, 0},
		{"past both", 400, 40, 39, 39},
		{"past one only", 400, 12, 39, 12},
		{"below one only", -3, 12, 0, 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := mustWorld(t, 1, gsBounds, []Entity{{ID: 1, X: 20, Y: 20}})
			Step(w, []Command{gsMove(1, c.x, c.y, 0)})

			got := w.Entities()[0]
			if got.TargetX != c.wantX || got.TargetY != c.wantY {
				t.Errorf("an order to (%d,%d) on a %dx%d map holds (%d,%d), want (%d,%d)",
					c.x, c.y, gsBounds.Width, gsBounds.Height, got.TargetX, got.TargetY, c.wantX, c.wantY)
			}
		})
	}
}

// TestTheGroupTagReachesNeitherTheBytesNorTheDigest is the negative half of the
// tag's contract, and it is the one a reader cannot check by inspection: two
// worlds advanced by the same order under different tags must be the same world,
// byte for byte.
func TestTheGroupTagReachesNeitherTheBytesNorTheDigest(t *testing.T) {
	run := func(tag uint32) *World {
		w := mustWorld(t, 1, gsBounds, gsSpread())
		Step(w, []Command{gsMove(1, 30, 30, tag), gsMove(2, 30, 30, tag), gsMove(3, 30, 30, tag)})
		return w
	}
	a, b := run(0), run(4294967295)

	if a.Hash() != b.Hash() {
		t.Errorf("the same order under two tags digests as %#016x and %#016x — the tag is state, "+
			"and it is not supposed to be", a.Hash(), b.Hash())
	}
}

// TestAGroupOrderWithNoSurvivingMemberDoesNothing is the empty case stated as
// behaviour rather than as a guard: the world it leaves is the world it was
// handed, tick and generator aside.
func TestAGroupOrderWithNoSurvivingMemberDoesNothing(t *testing.T) {
	w := mustWorld(t, 1, gsBounds, []Entity{{ID: 5, X: 3, Y: 3}})
	before := w.Entities()[0]
	Step(w, []Command{gsMove(1, 30, 30, 0), gsMove(2, 30, 30, 0)})

	if got := w.Entities()[0]; got != before {
		t.Errorf("a bystander went from %+v to %+v", before, got)
	}
}
