package scriptprofile

import (
	"fmt"
	"testing"
)

// The profiles hold the numbers the engine enforced before they were one
// table: three pairs, four actions, codes 0..5, 100 registers (1000 in the
// second game), 1000 latches and triggers, ids up to 5000.
func TestProfilesHoldTheEnginesLimits(t *testing.T) {
	want := Profile{Pairs: 3, Actions: 4, MaxCompare: 5, Registers: 100, Latches: 1000, Triggers: 1000, MaxID: 5000}
	if First != want {
		t.Errorf("First = %+v, want %+v", First, want)
	}
	want.Registers = 1000
	if Second != want {
		t.Errorf("Second = %+v, want %+v", Second, want)
	}
}

// The extended profile holds the owner's limits: 64 conditions and 64 actions
// per trigger in OR-ed groups, 4096 variables apart from the check results,
// 10000 triggers with a latch each, the original's codes and id cap.
func TestExtendedProfileHoldsTheOwnersLimits(t *testing.T) {
	want := Profile{Pairs: 64, Actions: 64, MaxCompare: 5, Latches: 10000, Triggers: 10000, MaxID: 5000,
		Variables: 4096, Groups: true}
	if Extended != want {
		t.Errorf("Extended = %+v, want %+v", Extended, want)
	}
}

func TestTriggerKeysSpellTheFieldTable(t *testing.T) {
	for i := 0; i < 4; i++ {
		for _, c := range []struct{ got, want string }{
			{LeftKey(i), fmt.Sprintf("left%d", i)}, {RightKey(i), fmt.Sprintf("right%d", i)},
			{CmpKey(i), fmt.Sprintf("cmp%d", i)}, {ActionKey(i), fmt.Sprintf("action%d", i)},
		} {
			if c.got != c.want {
				t.Errorf("key %q, want %q", c.got, c.want)
			}
		}
	}
}
