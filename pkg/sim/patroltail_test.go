package sim

import (
	"encoding/binary"
	"testing"
)

// R2-ENGINE-1624: in the second game an empty route under order 0xa moves the
// patrol on to its next point when the actor stands below Chebyshev 3 of the
// point and keeps the point farther away. The first game keeps the point.
func TestPatrolEmptyRouteMovesOnOnlyNearItsPoint(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		for _, tc := range []struct {
			x    int32
			near bool
		}{{32, true}, {33, false}} {
			w := asWorld(t, second, asFighter(asHero, SelfSlot, 30, 40))
			w.commandPatrol([]int{0}, cell{x: tc.x, y: 40})
			w.clearOrder(0)
			w.answerRefusedWalk(0, searchStatic)
			e := w.entities[0]
			want, wantX := patrolLegTail, tc.x
			if second && tc.near {
				want, wantX = patrolLegHead, 30
			}
			if e.PatrolLeg != want || e.legX() != wantX || e.legY() != 40 {
				t.Fatalf("second=%v point %d: leg %d at (%d,%d), want leg %d at (%d,40)", second, tc.x, e.PatrolLeg, e.legX(), e.legY(), want, wantX)
			}
			if e.ActorState != actorStatePatrol || e.PostX != 30 || e.PostY != 40 {
				t.Fatalf("second=%v point %d: the empty route ended the patrol: state %d post (%d,%d)", second, tc.x, e.ActorState, e.PostX, e.PostY)
			}
			if second {
				o := w.savedOrder(asHero)
				if got := binary.LittleEndian.Uint16(o.Raw[2:]); got != uint16(wantX)|40<<8 {
					t.Fatalf("point %d: saved patrol point %#x, want (%d,40)", tc.x, got, wantX)
				}
			}
		}
	}
}

// A loaded order-0xa block takes the same tail: latch +4 = 1, activity +8 = 0
// and the point word +2 advanced only below Chebyshev 3 (R2-ENGINE-1624).
func TestLoadedPatrolEmptyRouteTailWritesItsOrderBlock(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		x    int32
		near bool
	}{{32, true}, {33, false}} {
		w := asWorld(t, true, asFighter(asHero, SelfSlot, 30, 40))
		w.commandPatrol([]int{0}, cell{x: tc.x, y: 40})
		o := w.savedOrder(asHero)
		o.Authored = false
		o.Raw[8] = 1
		w.clearOrder(0)
		w.answerRefusedWalk(0, searchStatic)
		want := uint16(tc.x) | 40<<8
		if tc.near {
			want = 30 | 40<<8
		}
		if got := binary.LittleEndian.Uint16(o.Raw[2:]); got != want || binary.LittleEndian.Uint32(o.Raw[4:]) != 1 || o.Raw[8] != 0 {
			t.Fatalf("point %d: word +2 %#x latch %d activity %d, want %#x, 1 and 0", tc.x, got, binary.LittleEndian.Uint32(o.Raw[4:]), o.Raw[8], want)
		}
	}
}
