package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"
)

// A hero's body, body directory and class have one derivation: outside
// pkg/data, only pkg/game's heroAppearance calls data.HeroAppearance.
func TestHeroAppearanceHasOneBuilder(t *testing.T) {
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
		if f.Name.Name == "data" {
			continue
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "HeroAppearance" {
					return true
				}
				if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "data" {
					return true
				}
				count++
				if name != "pkg/game/heroappearance.go" || fn.Name.Name != "heroAppearance" {
					t.Errorf("%s:%s derives a hero's appearance outside heroAppearance", name, fn.Name.Name)
				}
				return true
			})
		}
	}
	if count != 1 {
		t.Fatalf("%d non-test references to data.HeroAppearance, want 1", count)
	}
}
