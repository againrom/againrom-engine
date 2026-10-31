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

// displayNameTableFiles are the resolver's name-table stage and the table loader.
var displayNameTableFiles = map[string]bool{
	"pkg/ui/displayname.go":   true,
	"pkg/game/installtext.go": true,
}

func unitNameIndexers(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	operand := func(x ast.Expr) bool {
		switch x := x.(type) {
		case *ast.Ident:
			return x.Name == "UnitNames" || x.Name == "unitNames"
		case *ast.SelectorExpr:
			return x.Sel.Name == "UnitNames" || x.Sel.Name == "unitNames"
		}
		return false
	}
	for name, source := range files {
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.IndexExpr:
				if operand(n.X) {
					out = append(out, name)
				}
			case *ast.SliceExpr:
				if operand(n.X) {
					out = append(out, name)
				}
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

func TestUnitNameIndexScanNamesEachIndexer(t *testing.T) {
	got := unitNameIndexers(t, map[string]string{
		"pkg/game/a.go": "package game\nfunc f(w W) string { return w.UnitNames[3] }\n",
		"pkg/game/b.go": "package game\nfunc g(unitNames []string) []string { return unitNames[1:] }\n",
		"pkg/ui/c.go":   "package ui\n// UnitNames[2] in a comment\nvar n = len(w.UnitNames)\n",
	})
	if strings.Join(got, ",") != "pkg/game/a.go,pkg/game/b.go" {
		t.Fatalf("unitNameIndexers = %v", got)
	}
}

// The character display name has one resolver.
func TestUnitNamesAreIndexedOnlyByTheDisplayNameResolver(t *testing.T) {
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
	for file := range displayNameTableFiles {
		if _, ok := files[file]; !ok {
			t.Fatalf("%s is missing; the scan guards nothing", file)
		}
	}
	for _, file := range unitNameIndexers(t, files) {
		if !displayNameTableFiles[file] {
			t.Errorf("%s reads a row of the unit name table outside the display name resolver", file)
		}
	}
}
