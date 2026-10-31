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

func readGlobalMapCallers(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				if fn.Name == "ReadGlobalMap" {
					out = append(out, name)
				}
			case *ast.SelectorExpr:
				if fn.Sel.Name == "ReadGlobalMap" {
					out = append(out, name)
				}
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

func TestReadGlobalMapCallerScanNamesEachCall(t *testing.T) {
	got := readGlobalMapCallers(t, map[string]string{
		"pkg/game/a.go": "package game\nfunc f() { ReadGlobalMap(nil); ReadGlobalMap(nil) }\n",
		"cmd/x/b.go":    "package main\nfunc g() { game.ReadGlobalMap(nil) }\n// ReadGlobalMap(nil)\n",
		"pkg/game/c.go": "package game\nfunc ReadGlobalMap(r *R) {}\n",
	})
	if strings.Join(got, ",") != "cmd/x/b.go,pkg/game/a.go,pkg/game/a.go" {
		t.Fatalf("readGlobalMapCallers = %v", got)
	}
}

// globalmap.reg is parsed in one place: the world-map catalog's reader.
func TestReadGlobalMapHasOneCaller(t *testing.T) {
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
	if got := readGlobalMapCallers(t, files); len(got) != 1 || got[0] != "pkg/game/worldmap.go" {
		t.Fatalf("ReadGlobalMap is called from %v; want one call, in the catalog reader in pkg/game/worldmap.go", got)
	}
}
