package mapload_test

import (
	"math"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// TestTheRulesTableAndTheFloatingPointCurveAgreeAtEveryLevel compares the one
// experience table with S(n) = ftol((1.1^n - 1) * 1000) computed in floating
// point over the whole original level range, and stops at the first level where
// they part.
func TestTheRulesTableAndTheFloatingPointCurveAgreeAtEveryLevel(t *testing.T) {
	for level := int32(0); level <= 100; level++ {
		want := int32((math.Pow(1.1, float64(level)) - 1) * 1000)
		if got := (sim.Rules{}).SkillXP(level); got != want {
			t.Fatalf("level %d: the table says %d, the curve says %d", level, got, want)
		}
		if got := data.SkillXPFor(level); got != want {
			t.Fatalf("level %d: data.SkillXPFor says %d, the curve says %d", level, got, want)
		}
	}
}

// TestTheThreeCheckpointsHold pins S(0), S(10) and S(100) as literals.
func TestTheThreeCheckpointsHold(t *testing.T) {
	for _, tc := range []struct {
		level int32
		want  int32
	}{
		{0, 0},
		{10, 1593},
		{100, 13779612},
	} {
		if got := data.SkillXPFor(tc.level); got != tc.want {
			t.Errorf("data.SkillXPFor(%d) = %d, want %d", tc.level, got, tc.want)
		}
		if got := (sim.Rules{}).SkillXP(tc.level); got != tc.want {
			t.Errorf("Rules.SkillXP(%d) = %d, want %d", tc.level, got, tc.want)
		}
	}
}
