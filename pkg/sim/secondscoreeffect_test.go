package sim

import "testing"

// R2-ENGINE-1495 and R2-ENGINE-1496: the ordinary scorer adds 0x7f for a
// candidate under effect 18; the stand-ground scorer reads no effect; the
// first game's scorer adds nothing here.
func TestTheSecondGameOrdinaryScoreAddsTheEffectCost(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{true, false} {
		self := asFighter(asHero, asSide, 30, 40)
		self.Facing, self.Mind = 64, 0
		w := asWorld(t, second, self, asFighter(asFoeA, SelfSlot, 30, 39))
		base, standing := w.candidateCost(0, 1, orderGuard), w.candidateCost(0, 1, orderStandGround)
		w.attached = append(w.attached, attachedEffect{Target: asFoeA, Spell: 20, Remaining: 100})
		if w.candidateCost(0, 1, orderGuard) != base {
			t.Fatalf("second %v: another effect changed the score", second)
		}
		w.attached = append(w.attached[:0], attachedEffect{Target: asFoeA, Spell: 18, Remaining: 100})
		want := base
		if second {
			want += 0x7f
		}
		if got := w.candidateCost(0, 1, orderGuard); got != want {
			t.Fatalf("second %v: ordinary score %d, want %d", second, got, want)
		}
		if got := w.candidateCost(0, 1, orderStandGround); got != standing {
			t.Fatalf("second %v: stand-ground score %d with the effect, %d without", second, got, standing)
		}
	}
}

// R2-ENGINE-1491: the hit hook's away cell scales both differences by 10 over
// the larger one and truncates.
func TestTheSecondGameAwayCellScalesByTheLargerAxis(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		ax, ay       int32
		vx, vy       int32
		wantX, wantY int32
	}{
		{"south-east", 10, 10, 13, 14, 20, 24},
		{"north-west", 30, 30, 27, 26, 20, 16},
		{"east", 10, 10, 12, 10, 22, 10},
		{"held inside the map", 60, 10, 62, 10, 63, 10},
	} {
		w := asWorld(t, true, asFighter(asHero, asSide, tc.ax, tc.ay), asFighter(asFoeA, SelfSlot, tc.vx, tc.vy))
		var o SavedActorOrder
		w.recordBlow(0, 1, &o)
		if got := (cell{int32(o.Strike.Away & 0xff), int32(o.Strike.Away >> 8)}); got != (cell{tc.wantX, tc.wantY}) {
			t.Errorf("%s: away cell %v, want (%d,%d)", tc.name, got, tc.wantX, tc.wantY)
		}
	}
}
