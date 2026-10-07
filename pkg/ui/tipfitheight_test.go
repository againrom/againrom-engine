package ui

import (
	"image"
	"testing"
)

// TipPanelFitHeight's answer is the smallest whole-panel height at which the
// text draws whole. That answer is a fact about the widget, so it is stated
// here against an independent oracle: the linear search written out again,
// asking tipPanelTextFits directly for every candidate height exactly as the
// production search did before the wrap was hoisted out of it.

// tipPanelFitHeightOracle is the search as it stood: one wrap per candidate.
func tipPanelFitHeightOracle(t *testing.T, s string, width int) int {
	t.Helper()
	font := shopTipTestFont()
	const floor, ceiling = 20, 480
	if s == "" {
		return floor
	}
	for h := floor; h <= ceiling; h++ {
		if tipPanelTextFits(font, s, image.Rect(0, 0, width, h)) {
			return h
		}
	}
	return ceiling
}

// The hoisted wrap must not move a single answer. Each case is a width and a
// text whose wrapped shape differs: one that fits on one line, one that wraps
// to several, and one long enough to exhaust the 480-row ceiling at a narrow
// width and take the fallback.
func TestFitHeightAgreesWithThePerCandidateSearch(t *testing.T) {
	long := "one two three four five six seven eight nine ten eleven twelve " +
		"thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty"
	cases := []struct {
		name  string
		text  string
		width int
	}{
		{"one line", "hello", 312},
		{"a few lines", "one two three four five six seven eight nine ten", 120},
		{"many lines", long, 90},
		{"very narrow", long, 40},
		{"town width", long, 312},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want := tipPanelFitHeightOracle(t, c.text, c.width)
			got := TipPanelFitHeight(shopTipTestFont(), c.text, c.width)
			if got != want {
				t.Fatalf("TipPanelFitHeight(%q, width %d) = %d, per-candidate search says %d",
					c.text, c.width, got, want)
			}
		})
	}
}

// THE COST IS THE POINT OF THE CHANGE, so it is witnessed and not left to a
// benchmark nobody runs. The search walks from a 20-row floor to the answer,
// and the old code wrapped the text again at every step; the wrap is where
// every allocation in this call lives. Measuring both here and requiring an
// order of magnitude between them is what fails if the wrap moves back inside
// the loop: the two would then allocate the same.
func TestFitHeightWrapsOncePerSearchNotOncePerCandidate(t *testing.T) {
	font := shopTipTestFont()
	text := "one two three four five six seven eight nine ten eleven twelve " +
		"thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty"
	const width = 90

	h := TipPanelFitHeight(font, text, width)
	if h < 120 {
		t.Fatalf("fixture: fit height %d is too near the 20-row floor to search over; "+
			"the cost witness needs a text whose answer is many candidates away", h)
	}

	fast := testing.AllocsPerRun(20, func() {
		TipPanelFitHeight(font, text, width)
	})
	perCandidate := testing.AllocsPerRun(20, func() {
		const floor, ceiling = 20, 480
		for c := floor; c <= ceiling; c++ {
			if tipPanelTextFits(font, text, image.Rect(0, 0, width, c)) {
				return
			}
		}
	})
	t.Logf("fit height %d: %.0f allocations against the per-candidate search's %.0f", h, fast, perCandidate)
	if fast*10 > perCandidate {
		t.Fatalf("TipPanelFitHeight allocates %.0f objects against the per-candidate search's %.0f; "+
			"want at most a tenth, which is what one wrap instead of %d buys",
			fast, perCandidate, h-20+1)
	}
}
