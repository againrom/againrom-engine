package archtest

import (
	"os"
	"strings"
	"testing"
)

// Every minimap picture comes from one producer: one minimapPlan literal, in
// minimapProducePlan, and one reference to composeMinimapPlan, in minimapCompose.
func TestMinimapHasOnePlanProducer(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	module, err := loadTypeCheckedModule(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range MinimapPlanSites(module) {
		t.Error(s)
	}
}

// One planted instance per shape is named, and the one producer is not.
func TestMinimapPlanScanSeesEachShape(t *testing.T) {
	const head = `package ui
type minimapPlan struct{ cols int }
func composeMinimapPlan(p minimapPlan, n int) {}
func minimapProducePlan() minimapPlan { return minimapPlan{cols: 1} }
func minimapCompose() { composeMinimapPlan(minimapProducePlan(), 0) }
`
	for _, tc := range []struct{ name, src, want string }{
		{"the one producer", head, ""},
		{"a second literal elsewhere", head + `
func other() minimapPlan { return minimapPlan{cols: 2} }`, "writes a minimapPlan outside minimapProducePlan"},
		{"a package value", head + `
var p = minimapPlan{cols: 2}`, "writes a minimapPlan outside minimapProducePlan"},
		{"an elided element type", head + `
var p = []minimapPlan{{cols: 2}}`, "writes a minimapPlan outside minimapProducePlan"},
		{"an alias of the type", head + `
type mp = minimapPlan
var p = mp{cols: 2}`, "writes a minimapPlan outside minimapProducePlan"},
		{"new", head + `
func other() *minimapPlan { return new(minimapPlan) }`, "makes a minimapPlan with new outside"},
		{"a declared variable", head + `
func other() minimapPlan { var p minimapPlan; return p }`, "declares a minimapPlan variable outside"},
		{"a field written", head + `
func other(p *minimapPlan) { p.cols = 3 }`, "assigns a field of a minimapPlan outside"},
		{"a field counted", head + `
func other(p *minimapPlan) { p.cols++ }`, "assigns a field of a minimapPlan outside"},
		{"a second caller", head + `
func other(p minimapPlan) { composeMinimapPlan(p, 0) }`, "references composeMinimapPlan outside minimapCompose"},
		{"a function value", head + `
var f = composeMinimapPlan`, "references composeMinimapPlan outside minimapCompose"},
		{"a second literal in the producer", `package ui
type minimapPlan struct{ cols int }
func composeMinimapPlan(p minimapPlan, n int) {}
func minimapProducePlan() minimapPlan { _ = minimapPlan{}; return minimapPlan{cols: 1} }
func minimapCompose() { composeMinimapPlan(minimapProducePlan(), 0) }`, "2 minimapPlan literals in production code"},
		{"no caller", `package ui
type minimapPlan struct{ cols int }
func minimapProducePlan() minimapPlan { return minimapPlan{cols: 1} }`, "0 references to composeMinimapPlan"},
	} {
		module := checkedFixture(t, fixturePackage{path: minimapPlanPackage, files: map[string]string{"pkg/ui/x.go": tc.src}})
		got := strings.Join(MinimapPlanSites(module), "; ")
		if tc.want == "" && got != "" {
			t.Errorf("%s: reported %q", tc.name, got)
		}
		if tc.want != "" && !strings.Contains(got, tc.want) {
			t.Errorf("%s: got %q, want it to name %q", tc.name, got, tc.want)
		}
	}
}
