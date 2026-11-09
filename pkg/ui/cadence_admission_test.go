package ui

import (
	"fmt"
	"image"
	"os"
	"strings"
	"testing"
	"time"
)

// admissionClock is the fixed sequence of clock readings, in milliseconds
// from an arbitrary origin, every admission instance is asked in turn. It has
// exact period boundaries, a repeat and a step back.
var admissionClock = []int64{0, 100, 400, 499, 500, 501, 600, 999, 1000, 1001, 1002, 1500, 1501, 2000, 2023, 2024, 2025, 3000, 3001, 5000, 5001, 4000, 7000, 31000}

func admissionAt(ms int64) time.Time {
	return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(ms) * time.Millisecond)
}

// admissionTrace records the answer of each admission instance at each clock
// reading. It is the same text before and after the cadence descriptor.
func admissionTrace() string {
	var b strings.Builder
	line := func(name string, ms int64, format string, a ...any) {
		fmt.Fprintf(&b, "%s @%d: %s\n", name, ms, fmt.Sprintf(format, a...))
	}

	var g guidedCycle
	steps := [][]int{{10, 11, 12}, {20, 21}}
	for i, ms := range admissionClock {
		hovered := -1
		if i == 6 || i == 14 {
			hovered = 11
		}
		line("guided", ms, "%d", g.paint(admissionAt(ms), steps, i/12, hovered, 300*time.Millisecond, 400*time.Millisecond))
	}

	var c caretBlink
	for _, ms := range admissionClock {
		c.tick(admissionAt(ms))
		line("caret", ms, "%v", c.on())
	}

	var n nameCaret
	for i, ms := range admissionClock {
		n.restart = i == 13
		n.paint(admissionAt(ms), i != 9, 500*time.Millisecond)
		line("namecaret", ms, "%v", n.visible())
	}

	var v voiceMemory
	for _, ms := range admissionClock {
		line("voice", ms, "%v", v.claim(admissionAt(ms), 500*time.Millisecond))
	}

	h := tooltipController{delay: 500}
	for _, ms := range admissionClock {
		h.observe(admissionAt(ms), image.Pt(1, 1), "k", ms == 3000)
		line("tooltip", ms, "%v", h.visible("k"))
	}

	for _, period := range []int64{66, 100} {
		m := NewCursorManager()
		m.SetRegistry(NewCursorRegistry([]CursorSlot{{Name: "x", FrameCount: 4, PeriodMillis: period}}))
		m.SetCursor("x")
		for _, ms := range admissionClock {
			m.Advance(ms + 1000)
			line(fmt.Sprintf("cursor%d", period), ms, "%d", m.frameIndex)
		}
	}

	var t TextInputState
	for i, ms := range admissionClock {
		held := map[string]bool{"left": i < 20}
		line("repeat", ms, "%v", t.Repeat(admissionAt(ms), held, gateAtLeast.reached))
	}
	return b.String()
}

func TestAdmissionTraceUnchanged(t *testing.T) {
	got := admissionTrace()
	if out := os.Getenv("ADMISSION_TRACE_RECORD"); out != "" {
		if err := os.WriteFile(out, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile("testdata/admission_trace.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("admission trace changed:\n%s", got)
	}
}
