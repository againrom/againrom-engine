package game

// 0146's two far sides: what ui.MapStance and ui.MapMarch put on the pending
// queue, and what they do not do.
//
// It reuses group_order_test.go's hand-built field and driver whole — a 9x5 open
// field, four units, no schedule, no map stream and no window — so what is
// observed here is the conversion and the queue alone.

import (
	"testing"

	"againrom/pkg/sim"
)

// vocSeams is the driver with a stance and a march already issued, and the
// pending queue as those two calls left it.
func vocPending(t *testing.T, issue func(*mapWorld)) []sim.Command {
	t.Helper()
	mw := groupDriver(t)
	before := mw.world.Hash()
	issue(mw)
	if got := mw.world.Hash(); got != before {
		t.Fatalf("issuing moved the digest %#x -> %#x — an order reached the world outside an advance", before, got)
	}
	return mw.pending
}

func TestTheStanceSeamConvertsTheBoolToAnOrderByte(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		guard bool
		want  int32
	}{
		{"guard", true, sim.OrderGuard},
		{"stand ground", false, sim.OrderStandGround},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := vocPending(t, func(mw *mapWorld) {
				mw.stance(uint32(groupIDs[0]), tc.guard)
				mw.stance(uint32(groupIDs[1]), tc.guard)
			})
			if len(got) != 2 {
				t.Fatalf("the queue holds %d commands, want 2", len(got))
			}
			for _, c := range got {
				if c.Kind != sim.KindGroupStance {
					t.Fatalf("command kind is %d, want the stance kind (%d)", c.Kind, sim.KindGroupStance)
				}
				if c.X != tc.want {
					t.Fatalf("the order byte is %d, want %d", c.X, tc.want)
				}
				if c.Y != 0 {
					t.Fatalf("Y is %d — this kind carries no cell", c.Y)
				}
			}
			if got[0].Group != got[1].Group {
				t.Fatalf("the two commands carry tags %d and %d — one key press is one group order",
					got[0].Group, got[1].Group)
			}
		})
	}
}

func TestTheMarchSeamConvertsTheBoolToAKind(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		patrol bool
		want   uint8
	}{
		{"patrol", true, sim.KindGroupPatrolTo},
		{"march", false, sim.KindGroupSwarmTo},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := vocPending(t, func(mw *mapWorld) {
				mw.march(uint32(groupIDs[0]), tc.patrol, 6, 3)
				mw.march(uint32(groupIDs[1]), tc.patrol, 6, 3)
			})
			if len(got) != 2 {
				t.Fatalf("the queue holds %d commands, want 2", len(got))
			}
			for _, c := range got {
				if c.Kind != tc.want {
					t.Fatalf("command kind is %d, want %d", c.Kind, tc.want)
				}
				if c.X != 6 || c.Y != 3 {
					t.Fatalf("the cell is (%d,%d), want (6,3)", c.X, c.Y)
				}
			}
			if got[0].Group != got[1].Group {
				t.Fatalf("the two commands carry tags %d and %d — one press is one group order",
					got[0].Group, got[1].Group)
			}
		})
	}
}

func TestTwoDifferentOrdersNeverShareATag(t *testing.T) {
	t.Parallel()

	got := vocPending(t, func(mw *mapWorld) {
		mw.stance(uint32(groupIDs[0]), true)
		mw.stance(uint32(groupIDs[1]), true)
		mw.enqueue(uint32(groupIDs[2]), 6, 3)
		mw.march(uint32(groupIDs[3]), false, 6, 3)
	})
	if len(got) != 4 {
		t.Fatalf("the queue holds %d commands, want 4", len(got))
	}
	if got[0].Group != got[1].Group {
		t.Fatal("the two stances did not share a tag")
	}
	for _, pair := range [][2]int{{0, 2}, {0, 3}, {2, 3}} {
		if got[pair[0]].Group == got[pair[1]].Group {
			t.Fatalf("commands %d (kind %d) and %d (kind %d) share tag %d",
				pair[0], got[pair[0]].Kind, pair[1], got[pair[1]].Kind, got[pair[0]].Group)
		}
	}
}

// TestEveryVocabularySeamMarksItsEntityCommanded is the mark strike and enqueue
// already make: a unit told to hold ground has been given something to do, so
// the placeholder schedule must not overwrite it.
func TestEveryVocabularySeamMarksItsEntityCommanded(t *testing.T) {
	t.Parallel()

	mw := groupDriver(t)
	mw.stance(uint32(groupIDs[0]), true)
	mw.march(uint32(groupIDs[1]), true, 6, 3)
	for _, id := range groupIDs[:2] {
		if !mw.commanded[sim.EntityID(id)] {
			t.Errorf("entity %d was ordered and is not marked commanded", id)
		}
	}
	for _, id := range groupIDs[2:] {
		if mw.commanded[sim.EntityID(id)] {
			t.Errorf("entity %d was never ordered and is marked commanded", id)
		}
	}
}
