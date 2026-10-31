package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// seeInvisibleTests names each pkg/sim production file outside sight.go that
// tests the SeeInvisible radius: the field read inside a comparison or any
// other binary expression. Copies for construction and persistence stay.
func seeInvisibleTests(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/sim/") || name == "pkg/sim/sight.go" {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			bin, ok := n.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			found := false
			ast.Inspect(bin, func(m ast.Node) bool {
				if sel, ok := m.(*ast.SelectorExpr); ok && sel.Sel.Name == "SeeInvisible" {
					found = true
				}
				return !found
			})
			if found {
				out = append(out, name)
				return false
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

func TestSeeInvisibleScanNamesEachRadiusTest(t *testing.T) {
	got := seeInvisibleTests(t, map[string]string{
		"pkg/sim/engage.go": "package sim\nfunc f(a, b E) bool { return d(a, b) <= int64(a.SeeInvisible) }\n",
		"pkg/sim/binary.go": "package sim\nfunc g(b []byte, e E) { b[251] = e.SeeInvisible; _ = E{SeeInvisible: e.SeeInvisible} }\n",
		"pkg/sim/sight.go":  "package sim\nfunc h(a E) bool { return 0 <= a.SeeInvisible }\n",
		"pkg/game/world.go": "package game\nfunc k(a E) bool { return 0 <= a.SeeInvisible }\n",
		"pkg/sim/effect.go": "package sim\nfunc m(a E) bool { return a.SeeInvisible+1 > 2 }\n",
	})
	want := []string{"pkg/sim/effect.go", "pkg/sim/engage.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("seeInvisibleTests = %v", got)
	}
}

// The target perception query in pkg/sim/sight.go holds the one SeeInvisible
// radius test.
func TestTargetPerceptionHasOneQuery(t *testing.T) {
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
	if !strings.Contains(files["pkg/sim/sight.go"], "func (w *World) perceives(q perception) bool") {
		t.Fatal("the perception query left pkg/sim/sight.go; the scan guards nothing")
	}
	if got := seeInvisibleTests(t, files); len(got) != 0 {
		t.Fatalf("SeeInvisible radius tests outside the perception query: %v", got)
	}
}
