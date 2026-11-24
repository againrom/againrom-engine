package sim

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

// msvcSteps is how many raw draws of the original stream (the CRT
// recurrence) lead from one state to another, or -1 past a bound.
func msvcSteps(from, to uint64) int {
	s := uint32(from)
	for k := 0; k < 1<<16; k++ {
		if uint64(s) == to {
			return k
		}
		s = s*214013 + 2531011
	}
	return -1
}

// orderedDraws answers every bounded draw with 0 and logs its width, the raw
// draws made before it in the step, and whether the recovery count asked it.
type orderedDraws struct {
	w     *World
	start uint64
	log   []string
}

func (s *orderedDraws) draw(n int32) int32 {
	pc := make([]uintptr, 8)
	k := runtime.Callers(2, pc)
	frames, recovery := runtime.CallersFrames(pc[:k]), false
	for {
		f, more := frames.Next()
		recovery = recovery || strings.HasSuffix(f.Function, ".castRecoveryTicks")
		if !more {
			break
		}
	}
	s.log = append(s.log, fmt.Sprintf("raw %d, draw %d recovery %v", msvcSteps(s.start, s.w.rng.state), n, recovery))
	return 0
}

// A second-game computer mage's Shield on itself draws once from its choice
// to the end of the recovery: the recovery's 0..3 draw, with no raw draw
// before it (R2-ENGINE-1577). The first game's draws are pinned unchanged.
func TestComputerMageSelfShieldDrawsOnce(t *testing.T) {
	for _, tc := range []struct {
		second bool
		want   []string
	}{
		{true, []string{"tick 9: raw 0, draw 3 recovery true"}},
		{false, []string{"tick 1: raw 2", "tick 9: raw 0, draw 3 recovery true"}},
	} {
		w, shield, _ := computerSupportFixture(t, tc.second, 7, 1, 60, true, true, 3)
		w.SetRandom(originalMode, 1)
		script := &orderedDraws{w: w}
		w.rng.scripted = script
		var got []string
		for tick := 1; tick <= 40; tick++ {
			script.start, script.log = w.rng.state, nil
			cast := false
			for _, e := range StepObserved(w, nil) {
				cast = cast || e.Caster == 1 && e.Spell == shield && e.Target == 1
			}
			for _, l := range script.log {
				got = append(got, fmt.Sprintf("tick %d: %s", tick, l))
			}
			if cast {
				break
			}
			if raw := msvcSteps(script.start, w.rng.state); raw != 0 {
				got = append(got, fmt.Sprintf("tick %d: raw %d", tick, raw))
			}
		}
		if strings.Join(got, "; ") != strings.Join(tc.want, "; ") {
			t.Errorf("second=%v: draws to the Shield's recovery %q, want %q", tc.second, got, tc.want)
		}
	}
}
