package ui

import (
	"image"
	"testing"
	"time"

	"againrom/pkg/ui/systemclick"
)

// clickStep is one tick of primary and secondary edges at a point and an
// offset from the start.
type clickStep struct {
	at                         time.Duration
	p                          image.Point
	press, release, rightPress bool
}

func runClicks(d *doubleClick, steps []clickStep) []bool {
	start := time.Unix(1_700_000_000, 0)
	var out []bool
	for _, s := range steps {
		in := appInput{CursorX: s.p.X, CursorY: s.p.Y, PrimaryPressed: s.press, PrimaryReleased: s.release, SecondaryPressed: s.rightPress}
		double := d.observe(in, start.Add(s.at))
		if s.press {
			out = append(out, double)
		}
	}
	return out
}

func pairClick(at time.Duration, x, y int) []clickStep {
	p := image.Pt(x, y)
	return []clickStep{{at: at, p: p, press: true}, {at: at + 10*time.Millisecond, p: p, release: true}}
}

func pairClicks(parts ...[]clickStep) []clickStep {
	var out []clickStep
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// The detector pairs presses by the system's setting, read through an
// injected platform reading: 300 ms and a 6 by 4 pixel rectangle here.
func TestDoubleClickDetector(t *testing.T) {
	setting := systemclick.Setting{Time: 300 * time.Millisecond, Width: 6, Height: 4}
	ms := time.Millisecond
	for _, tc := range []struct {
		name  string
		steps []clickStep
		want  []bool
	}{
		{"inside the time and the rectangle", pairClicks(pairClick(0, 100, 100), pairClick(300*ms, 103, 102)), []bool{false, true}},
		{"outside the time", pairClicks(pairClick(0, 100, 100), pairClick(301*ms, 100, 100)), []bool{false, false}},
		{"a pointer moved past the width", pairClicks(pairClick(0, 100, 100), pairClick(50*ms, 104, 100)), []bool{false, false}},
		{"a pointer moved past the height", pairClicks(pairClick(0, 100, 100), pairClick(50*ms, 100, 97)), []bool{false, false}},
		{"no release between the presses", []clickStep{{at: 0, p: image.Pt(5, 5), press: true}, {at: 50 * ms, p: image.Pt(5, 5), press: true}}, []bool{false, false}},
		{"the press after a double starts a pair", pairClicks(pairClick(0, 9, 9), pairClick(40*ms, 9, 9), pairClick(80*ms, 9, 9), pairClick(120*ms, 9, 9)), []bool{false, true, false, true}},
		{"a moved press starts its own pair", pairClicks(pairClick(0, 0, 0), pairClick(40*ms, 50, 50), pairClick(80*ms, 50, 50)), []bool{false, false, true}},
		{"a secondary press between ends the pair", pairClicks(pairClick(0, 9, 9), []clickStep{{at: 20 * ms, p: image.Pt(9, 9), rightPress: true}}, pairClick(40*ms, 9, 9)), []bool{false, false}},
		{"a held button released and pressed in one tick", []clickStep{{at: 0, p: image.Pt(9, 9), press: true}, {at: 40 * ms, p: image.Pt(9, 9), press: true, release: true}}, []bool{false, true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := doubleClick{read: func() systemclick.Setting { return setting }}
			got := runClicks(&d, tc.steps)
			if len(got) != len(tc.want) {
				t.Fatalf("presses %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("presses %v, want %v", got, tc.want)
				}
			}
		})
	}
}

// The setting is read once per press pair, at its first press, so the
// second press is judged by the setting its first press read.
func TestDoubleClickReadsTheSettingOncePerPair(t *testing.T) {
	reads := 0
	settings := []systemclick.Setting{{Time: 100 * time.Millisecond, Width: 4, Height: 4}, {Time: time.Second, Width: 4, Height: 4}}
	d := doubleClick{read: func() systemclick.Setting { s := settings[reads%2]; reads++; return s }}
	got := runClicks(&d, pairClicks(pairClick(0, 1, 1), pairClick(200*time.Millisecond, 1, 1), pairClick(900*time.Millisecond, 1, 1)))
	if want := []bool{false, false, true}; got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || reads != 2 {
		t.Fatalf("presses %v with %d reads, want %v with 2", got, reads, want)
	}
}

// Without a platform reading, every App uses the stated fallback.
func TestDoubleClickFallback(t *testing.T) {
	if systemclick.Fallback != (systemclick.Setting{Time: 500 * time.Millisecond, Width: 4, Height: 4}) {
		t.Fatalf("fallback %+v", systemclick.Fallback)
	}
	var d doubleClick
	got := runClicks(&d, pairClicks(pairClick(0, 10, 10), pairClick(500*time.Millisecond, 12, 8)))
	if !got[1] {
		t.Fatal("a second press inside 500 ms and 2 pixels is no double click")
	}
}
