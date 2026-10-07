package archtest

import (
	"os"
	"testing"
)

// TestLiveCompositionMatchesItsBaseline measures the real pkg/game and holds
// both ratcheted numbers against the committed record. A field added to a
// component, or a function that starts reaching a third component, fails
// here; driving either down requires writing the new number into
// composition_baseline.go in the same commit.
func TestLiveCompositionMatchesItsBaseline(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	report, err := LoadComposition(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	if len(report.Components) == 0 {
		t.Fatal("measured no owned components; the loader or the module root is wrong")
	}
	sum := 0
	for _, c := range report.Components {
		if c.Fields == 0 {
			t.Errorf("component %s declares no fields; a component with nothing in it owns no lifecycle", c.Name)
		}
		sum += c.Fields
	}
	if sum != report.Fields {
		t.Fatalf("component field counts sum to %d, total says %d", sum, report.Fields)
	}
	for _, v := range CheckComposition(report, CommittedComposition) {
		t.Error(v)
	}
}

// TestCheckCompositionRatchets holds the evaluator itself against a table, so
// the direction of each number is decided here rather than by whichever tree
// happens to be checked out.
func TestCheckCompositionRatchets(t *testing.T) {
	base := CompositionBaseline{Fields: 81, Coordinators: 18}
	cases := []struct {
		name    string
		report  CompositionReport
		want    int
		wantSub string
	}{
		{"equal", reportOf(81, 18), 0, ""},
		{"a field added", reportOf(82, 18), 1, "rose from 81 to 82"},
		{"a field removed", reportOf(80, 18), 1, "fell from 81 to 80"},
		{"a new coordination point", reportOf(81, 19), 1, "rose from 18 to 19"},
		{"a coordination point removed", reportOf(81, 17), 1, "fell from 18 to 17"},
		{"both moved", reportOf(80, 17), 2, "fell"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CheckComposition(c.report, base)
			if len(got) != c.want {
				t.Fatalf("violations = %v, want %d", got, c.want)
			}
			if c.wantSub != "" && !contains(got, c.wantSub) {
				t.Fatalf("violations = %v, want one naming %q", got, c.wantSub)
			}
		})
	}
}

func reportOf(fields, coordinators int) CompositionReport {
	r := CompositionReport{Fields: fields}
	for i := 0; i < coordinators; i++ {
		r.Coordination = append(r.Coordination, FileComponents{File: "x.go"})
	}
	return r
}

func contains(list []string, sub string) bool {
	for _, s := range list {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}
