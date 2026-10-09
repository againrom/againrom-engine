package ui

import "time"

// The two guided-cycle constants (TOWN-519): a hover freezes the cycle for
// 500 ms, then a step lands at the first paint more than 300 ms after the last.
const (
	guidedCycleHoverWait = 500 * time.Millisecond
	guidedCycleStepGap   = 300 * time.Millisecond
)

// guidedCycle is the one highlight cycle both generator pages run while their
// tip popup shows: pre-create over portraits, levels, then amulet and OK
// (TOWN-519), the detailed page over its five skills (TOWN-522). Targets are
// region codes, grouped by tip step. Its timers and index are process state:
// a page enter does not reset them.
type guidedCycle struct {
	started         bool
	hoverAt, stepAt time.Time
	index, length   int
	lastHovered     int
}

// paint runs one paint of the cycle and returns the index into steps[step]
// to highlight, or -1 for no highlight. hovered is the region under the
// pointer, or -1. A step outside steps draws nothing and stops stepping.
func (g *guidedCycle) paint(now time.Time, steps [][]int, step, hovered int) int {
	if !g.started {
		g.started, g.hoverAt, g.stepAt, g.length, g.lastHovered = true, now, now, 1, -1
	}
	if now.Sub(g.hoverAt) < guidedCycleHoverWait {
		g.stepAt = now
		return -1
	}
	if g.lastHovered != -1 {
		g.index = guidedCycleAfter(steps, g.lastHovered)
	}
	draw := -1
	if step >= 0 && step < len(steps) && len(steps[step]) > 0 {
		targets := steps[step]
		for _, r := range targets {
			if r == hovered {
				g.hoverAt, g.stepAt, g.lastHovered = now, now, hovered
				return -1
			}
		}
		g.length = len(targets)
		g.index = ((g.index % g.length) + g.length) % g.length
		draw = g.index
	} else {
		g.length = -1
	}
	g.lastHovered = -1
	if now.Sub(g.stepAt) > guidedCycleStepGap && g.length > 0 {
		g.index = (g.index + 1) % g.length
		g.stepAt = now
	}
	return draw
}

// guidedCycleAfter is the original's region-to-index table: the target after
// the hovered one in its own step, unwrapped.
func guidedCycleAfter(steps [][]int, region int) int {
	for _, targets := range steps {
		for i, r := range targets {
			if r == region {
				return i + 1
			}
		}
	}
	return 0
}
