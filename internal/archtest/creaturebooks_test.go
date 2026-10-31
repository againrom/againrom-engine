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

// retiredCreatureBookNames are the first game's fixed book table and the
// second game's switch around it; the profile's CreatureBooks replaces both.
var retiredCreatureBookNames = map[string]bool{
	"dragonBooks": true, "daemonBook": true, "fixedCreatureBook": true,
	"addFixedCreatureBook": true, "ClassCreatureBooks": true, "classCreatureBooks": true,
}

func retiredCreatureBookUses(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && retiredCreatureBookNames[id.Name] {
				out = append(out, name+":"+id.Name)
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

func TestRetiredCreatureBookScanNamesEachUse(t *testing.T) {
	got := retiredCreatureBookUses(t, map[string]string{
		"pkg/mapload/x.go": "package mapload\nvar dragonBooks = 1\nfunc f(t *T) bool { return t.ClassCreatureBooks }\n",
		"pkg/game/y.go":    "package game\ntype g struct{ classCreatureBooks bool }\n",
	})
	want := []string{"pkg/game/y.go:classCreatureBooks", "pkg/mapload/x.go:ClassCreatureBooks", "pkg/mapload/x.go:dragonBooks"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("retiredCreatureBookUses = %v", got)
	}
}

// The creature book source is one per profile: dragonBooks and the flag that
// switched it off are gone from every non-test source.
func TestDragonBooksIsGone(t *testing.T) {
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
	if _, ok := files["pkg/mapload/spell.go"]; !ok {
		t.Fatal("pkg/mapload/spell.go is missing; the scan guards nothing")
	}
	if got := retiredCreatureBookUses(t, files); len(got) != 0 {
		t.Fatalf("retired creature book names in use: %v", got)
	}
}
