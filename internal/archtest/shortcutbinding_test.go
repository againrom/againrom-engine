package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShortcutBindingsHaveOneWriter(t *testing.T) {
	root := moduleRootForTest(t)
	paths, err := filepath.Glob(filepath.Join(root, "pkg/ui/*.go"))
	if err != nil {
		t.Fatal(err)
	}
	writes := 0
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(src), "quickSpells") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				a, ok := n.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for _, lhs := range a.Lhs {
					index, ok := lhs.(*ast.IndexExpr)
					if !ok {
						continue
					}
					field, ok := index.X.(*ast.SelectorExpr)
					if !ok || field.Sel.Name != "quickSpells" {
						continue
					}
					writes++
					if fn.Name.Name != "bindShortcut" {
						t.Errorf("%s: %s writes a shortcut outside bindShortcut", path, fn.Name.Name)
					}
				}
				return true
			})
		}
	}
	if writes != 2 {
		t.Fatalf("found %d slot writes, want clear and assign in the shared writer", writes)
	}
}
