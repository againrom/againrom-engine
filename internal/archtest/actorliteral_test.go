package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestLiveActorLiteralsMatchTheirBaseline measures the real tree: no actor
// origin writes a sim.Entity literal, and no other file's count rises.
func TestLiveActorLiteralsMatchTheirBaseline(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	report, err := LoadActorLiterals(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	if len(report.Files) == 0 {
		t.Fatal("measured no Entity literals at all; the loader or the module root is wrong")
	}
	if report.Files[actorConstructor] != 0 {
		t.Fatalf("%s is counted; it is the constructor file", actorConstructor)
	}
	for _, v := range CheckActorLiterals(report, CommittedActorLiterals) {
		t.Error(v)
	}
}

// TestActorOriginsAreNotInTheBaseline keeps the origin list and the baseline
// disjoint, so no baseline edit can admit a literal in a file that creates
// actors.
func TestActorOriginsAreNotInTheBaseline(t *testing.T) {
	for _, f := range actorOrigins {
		if _, ok := CommittedActorLiterals[f]; ok {
			t.Errorf("%s creates actors and must not hold a baseline count", f)
		}
	}
}

// TestCheckActorLiteralsRatchets holds the evaluator against a table.
func TestCheckActorLiteralsRatchets(t *testing.T) {
	base := map[string]int{"pkg/game/probe.go": 2}
	cases := []struct {
		name    string
		files   map[string]int
		want    int
		wantSub string
	}{
		{"equal", map[string]int{"pkg/game/probe.go": 2}, 0, ""},
		{"a probe added", map[string]int{"pkg/game/probe.go": 3}, 1, "baseline 2"},
		{"a probe removed", map[string]int{"pkg/game/probe.go": 1}, 1, "fell from 2 to 1"},
		{"a new file", map[string]int{"pkg/game/probe.go": 2, "pkg/game/new.go": 1}, 1, "pkg/game/new.go"},
		{"an origin literal", map[string]int{"pkg/game/probe.go": 2, "pkg/mapload/fromalm.go": 1}, 1, "build them with sim.NewActor"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CheckActorLiterals(ActorLiteralReport{Files: c.files}, base)
			if len(got) != c.want {
				t.Fatalf("violations = %v, want %d", got, c.want)
			}
			if c.wantSub != "" && !strings.Contains(strings.Join(got, "\n"), c.wantSub) {
				t.Fatalf("violations = %v, want one naming %q", got, c.wantSub)
			}
		})
	}
}
