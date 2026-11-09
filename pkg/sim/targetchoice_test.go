package sim

import "testing"

func TestTargetChoiceKeepsGroupAndRepickTieRules(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		self := asFighter(asHero, SelfSlot, 30, 40)
		self.Facing = 64
		w := asWorld(t, second, self, asFoe(asFoeA, 31, 39), asFoe(asFoeB, 31, 41))
		if got := w.chooseTarget(0, []int{0, 1, 2}, orderStandGround, targetGroupScore); got != 1 {
			t.Fatalf("second=%v group tie=%d, want first candidate 1", second, got)
		}
		if got := w.reacquisitionVictim(0); got != 2 {
			t.Fatalf("second=%v re-pick tie=%d, want later candidate 2", second, got)
		}
		if got := w.chooseTarget(0, nil, orderStandGround, targetGroupScore); got != -1 {
			t.Fatalf("second=%v empty choice=%d", second, got)
		}
	}
}
