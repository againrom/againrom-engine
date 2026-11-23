package sim

import "testing"

// AI-COST-071: the first game's ordinary scorer adds 0x7f (minimum 1) for a
// candidate under spell 20; the stand-ground scorer reads no effect; the
// second game's scorer adds nothing for spell 20.
func TestTheFirstGameOrdinaryScoreAddsTheEffectCost(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		self := asFighter(asHero, asSide, 30, 40)
		self.Facing, self.Mind = 64, 0
		w := asWorld(t, second, self, asFighter(asFoeA, SelfSlot, 30, 39))
		base, standing := w.candidateCost(0, 1, orderGuard), w.candidateCost(0, 1, orderStandGround)
		w.attached = append(w.attached, attachedEffect{Target: asFoeA, Spell: 19, Remaining: 100})
		if w.candidateCost(0, 1, orderGuard) != base {
			t.Fatalf("second %v: another effect changed the score", second)
		}
		w.attached = append(w.attached[:0], attachedEffect{Target: asFoeA, Spell: 20, Remaining: 100})
		want := base
		if !second {
			want = max(base+0x7f, 1)
		}
		if got := w.candidateCost(0, 1, orderGuard); got != want {
			t.Fatalf("second %v: ordinary score %d, want %d", second, got, want)
		}
		if got := w.candidateCost(0, 1, orderStandGround); got != standing {
			t.Fatalf("second %v: stand-ground score %d with the effect, %d without", second, got, standing)
		}
	}
}
