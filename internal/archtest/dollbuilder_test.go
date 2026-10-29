package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"
)

func TestDollLayersHaveOneBuilder(t *testing.T) {
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
	count := 0
	for name, source := range files {
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		if f.Name.Name != "game" {
			continue
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "FigureDrawSteps" {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != "data" {
					return true
				}
				count++
				if name != "pkg/game/inventory.go" || fn.Name.Name != "composeDoll" {
					t.Errorf("%s:%s builds doll layers outside composeDoll", name, fn.Name.Name)
				}
				return true
			})
		}
	}
	if count != 2 {
		t.Fatalf("FigureDrawSteps references = %d, want the source and nil-source arms", count)
	}
}
