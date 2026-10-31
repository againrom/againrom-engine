package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"testing"
)

// groupRouteFiles read the group-route profile, never the game's identity.
var groupRouteFiles = []string{"pkg/sim/savedgroups.go", "pkg/sim/savedgroupsai.go", "pkg/sim/groupengaged.go"}

// groupRouteIdentityCalls names each SecondGame call.
func groupRouteIdentityCalls(t *testing.T, files map[string]string, names []string) []string {
	t.Helper()
	var out []string
	for _, name := range names {
		source, ok := files[name]
		if !ok {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "SecondGame" {
					out = append(out, fset.Position(call.Pos()).String())
				}
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

func TestGroupRouteIdentityCallsNameAPlantedCall(t *testing.T) {
	got := groupRouteIdentityCalls(t, map[string]string{
		"pkg/sim/savedgroups.go":  "package sim\nfunc f(w *World) bool { return w.SecondGame() }\n",
		"pkg/sim/groupengaged.go": "package sim\n// SecondGame in a comment\nfunc g(w *World) bool { return w.groupRoute().heldCircle }\n",
		"pkg/sim/other.go":        "package sim\nfunc h(w *World) bool { return w.SecondGame() }\n",
	}, groupRouteFiles)
	if len(got) != 1 || got[0] != "pkg/sim/savedgroups.go:2:32" {
		t.Fatalf("groupRouteIdentityCalls = %v", got)
	}
}

func TestSavedGroupsReadTheGroupRouteProfile(t *testing.T) {
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
	for _, f := range groupRouteFiles {
		if _, ok := files[f]; !ok {
			t.Fatalf("%s is missing; the scan guards nothing", f)
		}
	}
	for _, c := range groupRouteIdentityCalls(t, files, groupRouteFiles) {
		t.Errorf("a saved Group site asks the game's identity, not the group-route profile: %s", c)
	}
}
