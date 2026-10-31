package ui

import "testing"

// TestParseMapSourceAnswersEachKind pins the kind, number and path the parser
// gives one source of each kind and a scenario source whose name is no number.
func TestParseMapSourceAnswersEachKind(t *testing.T) {
	for _, c := range []struct {
		source string
		want   MapSource
	}{
		{"new/64x64", MapSource{Kind: SourceNew, Text: "new/64x64", Path: "64x64"}},
		{"scenario/10.alm", MapSource{Kind: SourceScenario, Text: "scenario/10.alm", Path: "10.alm", Number: 10, HasNumber: true}},
		{"loose/maps/one.alm", MapSource{Kind: SourceLoose, Text: "loose/maps/one.alm", Path: "maps/one.alm"}},
		{`C:\maps\one.alm`, MapSource{Kind: SourceHost, Text: `C:\maps\one.alm`, Path: `C:\maps\one.alm`}},
		{"scenario/x.alm", MapSource{Kind: SourceScenario, Text: "scenario/x.alm", Path: "x.alm"}},
		{"", MapSource{Kind: SourceHost}},
	} {
		if got := ParseMapSource(c.source); got != c.want {
			t.Errorf("ParseMapSource(%q) = %+v, want %+v", c.source, got, c.want)
		}
	}
	for _, k := range []MapSourceKind{SourceNew, SourceScenario, SourceLoose, SourceHost} {
		key := MapSourceKey(k, "a/b.alm")
		if got := ParseMapSource(key); got.Kind != k || got.Path != "a/b.alm" {
			t.Errorf("ParseMapSource(MapSourceKey(%d)) = %+v", k, got)
		}
	}
}
