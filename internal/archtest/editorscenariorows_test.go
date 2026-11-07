package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"
)

// scenarioBuilderRefs names, for each map editor method the Scenario
// palette and the one-click adds rest on, the pkg/ui functions that may
// name it: the palette's rows come from the tree's row builder, a new
// record's proposals from the Add window's step offer, and the one-click
// trigger and variable from one builder, reached from the command row, the
// wizard's New and the palette's buttons, and run again by itself once the
// map is extended.
var scenarioBuilderRefs = map[string][]string{
	"treeMembers":   {"treeMatches", "treeRowsOf", "treeFollow"},
	"treeRowsOf":    {"treeRows", "scenarioRows"},
	"addAccepted":   {"addPlacement", "addNow"},
	"NewRecordStep": {"addOffer", "placeChoices"},
	"addNow":        {"menuAct", "newTrigger", "chromePress", "addNow"},
}

// checkScenarioBuilders reports, in one non-test pkg/ui source file, a
// selector naming one of those methods outside the functions allowed it; a
// table entry outside any function names "" as its function.
func checkScenarioBuilders(name, src string) []string {
	if !strings.HasPrefix(name, "pkg/ui/") {
		return nil
	}
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
	if err != nil {
		return []string{name + ": " + err.Error()}
	}
	var out []string
	look := func(fn string, n ast.Node) {
		ast.Inspect(n, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if allowed, ok := scenarioBuilderRefs[sel.Sel.Name]; ok && !slices.Contains(allowed, fn) {
					out = append(out, name+": "+fn+" names "+sel.Sel.Name)
				}
			}
			return true
		})
	}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			if fn.Body != nil {
				look(fn.Name.Name, fn.Body)
			}
			continue
		}
		look("", d)
	}
	return out
}

func TestEditorScenarioRowsAndOneClickTriggerHaveOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for name, src := range files {
		for _, v := range checkScenarioBuilders(name, src) {
			t.Error(v)
		}
		if strings.HasPrefix(name, "pkg/ui/") {
			for method := range scenarioBuilderRefs {
				if strings.Contains(src, "."+method+"(") {
					seen[method] = true
				}
			}
		}
	}
	for method := range scenarioBuilderRefs {
		if !seen[method] {
			t.Errorf("pkg/ui calls no %s: the guard has nothing to hold", method)
		}
	}
}

func TestEditorScenarioGuardSeesASecondBuilder(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		want      int
	}{
		{"pkg/ui/mapeditorpalettes.go", "package ui\nfunc (e *MapEditor) scenarioRows() { e.treeRowsOf(nil, \"\", nil) }", 0},
		{"pkg/ui/mapeditorpalettes.go", "package ui\nfunc (e *MapEditor) groupRows() { e.treeRowsOf(nil, \"\", nil) }", 1},
		{"pkg/ui/mapeditorpalettes.go", "package ui\nfunc (e *MapEditor) scenarioRows() { e.treeMembers(c) }", 1},
		{"pkg/ui/mapeditoradd.go", "package ui\nfunc (e *MapEditor) addOffer() { h.NewRecordStep(r, k) }", 0},
		{"pkg/ui/mapeditortrigger.go", "package ui\nfunc (e *MapEditor) triggerName() { h.NewRecordStep(r, \"name\") }", 1},
		{"pkg/ui/mapeditortrigger.go", "package ui\nfunc (e *MapEditor) addNow() { e.addAccepted(h, r, -1, p) }", 0},
		{"pkg/ui/mapeditorwizard.go", "package ui\nfunc (e *MapEditor) wizardNew() { e.addAccepted(h, r, -1, p) }", 1},
		{"pkg/ui/mapeditorwindow.go", "package ui\nfunc (e *MapEditor) menuAct(k string) { e.addNow() }", 0},
		{"pkg/ui/mapeditorkeys.go", "package ui\nvar rows = []keyRow{{act: (*MapEditor).addNow}}", 1},
		{"pkg/game/mapinspectionadd.go", "package game\nfunc (s *edits) addKey() { s.NewRecordStep(r, k) }", 0},
	} {
		if got := checkScenarioBuilders(tc.name, tc.src); len(got) != tc.want {
			t.Errorf("%s: %d findings %v, want %d", tc.src, len(got), got, tc.want)
		}
	}
}
