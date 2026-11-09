package archtest

import (
	"os"
	"strings"
	"testing"
)

// Every minimap picture comes from one producer: one minimapPlan literal, in
// minimapProducePlan, and one call of composeMinimapPlan, in minimapCompose.
func TestMinimapHasOnePlanProducer(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	sites, err := MinimapPlanSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Error(s)
	}
}

// One planted instance per shape is named, and the one producer is not.
func TestMinimapPlanScanSeesEachShape(t *testing.T) {
	const producer = `package ui
func minimapProducePlan() minimapPlan { return minimapPlan{cols: 1} }
func minimapCompose() { composeMinimapPlan(minimapProducePlan(), 0) }`
	for _, tc := range []struct{ name, src, want string }{
		{"the one producer", producer, ""},
		{"a second literal elsewhere", producer + `
func other() minimapPlan { return minimapPlan{cols: 2} }`, "writes a minimapPlan outside minimapProducePlan"},
		{"a package value", producer + `
var p = minimapPlan{cols: 2}`, "writes a minimapPlan outside minimapProducePlan"},
		{"a second caller", producer + `
func other() { composeMinimapPlan(p, 0) }`, "calls composeMinimapPlan outside minimapCompose"},
		{"a second literal in the producer", `package ui
func minimapProducePlan() minimapPlan { _ = minimapPlan{}; return minimapPlan{cols: 1} }
func minimapCompose() { composeMinimapPlan(minimapProducePlan(), 0) }`, "2 minimapPlan literals in production code"},
		{"no caller", `package ui
func minimapProducePlan() minimapPlan { return minimapPlan{cols: 1} }`, "0 calls of composeMinimapPlan"},
	} {
		got := strings.Join(minimapPlanSitesIn(map[string][]byte{"pkg/ui/x.go": []byte(tc.src)}), "; ")
		if tc.want == "" && got != "" {
			t.Errorf("%s: reported %q", tc.name, got)
		}
		if tc.want != "" && !strings.Contains(got, tc.want) {
			t.Errorf("%s: got %q, want it to name %q", tc.name, got, tc.want)
		}
	}
}
