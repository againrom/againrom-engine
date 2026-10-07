package game

// The play census's silence accounting (1033, adversarial pass 1 W-2).
//
// The census is the instrument pipeline/check-milestone.sh reports this
// build's script-gap number from, so a silence it does not count is a gap
// nothing measures. ScriptSilenceNoStructure was added by 1033 and the
// observe switch listed the reasons that existed before it, so the new
// reason was counted nowhere. These tests assert the property that failed
// rather than the one arm that was missing.

import (
	"testing"

	"againrom/pkg/sim"
)

// silenceReasons is every reason pkg/sim declares today. It is a hand list
// and it cannot see a reason added later; what covers that case is the
// production switch, whose second arm is a default rather than a list, so a
// reason nobody added here still lands in the unresolved bucket and is still
// counted.
var silenceReasons = []sim.ScriptSilence{
	sim.ScriptSilenceUnsupported,
	sim.ScriptSilenceNoUnit,
	sim.ScriptSilenceNoGroup,
	sim.ScriptSilenceNoPlayer,
	sim.ScriptSilenceNoStructure,
}

// TestThePlayCensusCountsEverySilenceReason is the property W-2 names: every
// reason lands in exactly one of the census's two buckets, and none is
// dropped. Unsupported is the arm this build cannot evaluate; every other
// reason is an unresolved reference.
func TestThePlayCensusCountsEverySilenceReason(t *testing.T) {
	t.Parallel()

	for _, why := range silenceReasons {
		var c playCensus
		c.observe(&sim.ScriptTrace{
			Silent: []sim.ScriptSilentCheck{{Check: 1, Op: 21, Register: 3, Why: why}},
		})
		inChecks, inUnresolved := c.checks[21], c.unresolved[21]
		if inChecks+inUnresolved != 1 {
			t.Errorf("reason %d landed in checks=%d unresolved=%d, want exactly one of the two; "+
				"a reason counted in neither is a silence the milestone census cannot see",
				why, inChecks, inUnresolved)
		}
		wantUnsupported := why == sim.ScriptSilenceUnsupported
		if (inChecks == 1) != wantUnsupported {
			t.Errorf("reason %d landed in the unsupported-arm bucket=%v, want %v",
				why, inChecks == 1, wantUnsupported)
		}
	}
}

// TestThePlayCensusCountsAnUnresolvedStructureReference is the instance the
// pass found: a check-21 node whose structure reference resolved to nothing.
func TestThePlayCensusCountsAnUnresolvedStructureReference(t *testing.T) {
	t.Parallel()

	var c playCensus
	c.observe(&sim.ScriptTrace{
		Silent: []sim.ScriptSilentCheck{{Check: 1, Op: 21, Register: 3, Why: sim.ScriptSilenceNoStructure}},
	})
	if got := c.unresolved[21]; got != 1 {
		t.Errorf("unresolved count for check op 21 is %d, want 1", got)
	}
	// It is NOT an unsupported arm: this build evaluates check 21. Counting it
	// there would inflate the number check-milestone.sh reports as the
	// script gap.
	if got := c.checks[21]; got != 0 {
		t.Errorf("unsupported-arm count for check op 21 is %d, want 0", got)
	}
}
