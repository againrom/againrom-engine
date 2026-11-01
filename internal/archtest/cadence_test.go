package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckCadenceNamesEachShape plants one instance of each shape the scan
// refuses, and the shapes it admits.
func TestCheckCadenceNamesEachShape(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a Sub compared with a duration", "pkg/ui/a.go",
			"package ui\nfunc (c *x) paint(now time.Time) { if now.Sub(c.at) > gap { c.n++ } }\n", "an elapsed-time comparison"},
		{"time.Since compared with a duration", "pkg/ui/a.go",
			"package ui\nfunc paint() bool { return time.Since(start) >= blink }\n", "an elapsed-time comparison"},
		{"an elapsed variable compared", "pkg/ui/a.go",
			"package ui\nfunc paint(now time.Time) bool { d := now.Sub(last); return d < period }\n", "an elapsed-time comparison"},
		{"a deadline After", "pkg/ui/a.go",
			"package ui\nfunc paint(now time.Time) bool { return now.After(next) }\n", "a deadline test (After)"},
		{"a deadline Before", "pkg/ui/a.go",
			"package ui\nfunc paint(now time.Time) bool { return !now.Before(next) }\n", "a deadline test (Before)"},
		{"a millisecond clock difference", "pkg/ui/a.go",
			"package ui\nfunc (m *x) adv(nowMillis int64) bool { return nowMillis-m.lastTick > m.periodMillis }\n", "a clock difference compared with a period"},
		{"a tick modulo", "pkg/ui/a.go",
			"package ui\nfunc (v *x) frameOf() int { return v.tick % blinkPeriod }\n", "a count modulo a period"},
		{"the cadence file itself", "pkg/ui/cadence.go",
			"package ui\nfunc (c cadence) due(last, now time.Time) bool { return now.Sub(last) > c.period }\n", ""},
		{"a test file", "pkg/ui/a_test.go",
			"package ui\nfunc f(now time.Time) bool { return now.After(next) }\n", ""},
		{"outside pkg/ui", "pkg/town/a.go",
			"package town\nfunc f(now time.Time) bool { return now.Sub(last) > gap }\n", ""},
		{"a point difference", "pkg/ui/a.go",
			"package ui\nfunc f(p, q image.Point) image.Point { return p.Sub(q) }\n", ""},
		{"a due call", "pkg/ui/a.go",
			"package ui\nfunc f(now time.Time) bool { return cadence{period: gap}.due(last, now) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckCadence(map[string]string{tc.file: tc.src}) {
				if !strings.Contains(v.Reason, "nothing there needs it") {
					got = append(got, v.Reason)
				}
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckCadence = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckCadence = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

// TestCheckCadenceAllowsAMeasurementAndRefusesAStaleEntry pins the allow list.
func TestCheckCadenceAllowsAMeasurementAndRefusesAStaleEntry(t *testing.T) {
	src := "package ui\nfunc (d *doubleClick) inside(now time.Time) bool { return now.Sub(d.at) > d.window }\n"
	for _, v := range CheckCadence(map[string]string{"pkg/ui/doubleclick.go": src}) {
		if strings.Contains(v.Reason, "presentation clock test") {
			t.Fatalf("a listed measurement was refused: %s", v)
		}
	}
	vs := CheckCadence(map[string]string{"pkg/ui/a.go": "package ui\n"})
	if len(vs) != len(cadenceMeasurements) {
		t.Fatalf("CheckCadence over an empty package = %d violations, want %d stale entries", len(vs), len(cadenceMeasurements))
	}
	if vs := CheckCadence(nil); len(vs) != 1 {
		t.Fatalf("CheckCadence(nil) = %v, want one violation", vs)
	}
}

// TestPresentationCadencesHaveOneDescriptor is the live-tree half.
func TestPresentationCadencesHaveOneDescriptor(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckCadence(files) {
		t.Errorf("%s", v)
	}
}
