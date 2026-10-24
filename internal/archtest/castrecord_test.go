package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"testing"
)

func TestCellSpellInspectionHasOneBuilder(t *testing.T) {
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
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				typ, ok := lit.Type.(*ast.SelectorExpr)
				if !ok || typ.Sel.Name != "InspectionRecord" {
					return true
				}
				for _, e := range lit.Elts {
					kv, ok := e.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := kv.Key.(*ast.Ident)
					if !ok || key.Name != "Kind" {
						continue
					}
					value, ok := kv.Value.(*ast.BasicLit)
					if !ok || value.Kind != token.STRING {
						continue
					}
					kind, _ := strconv.Unquote(value.Value)
					if kind != "Enchantment" {
						continue
					}
					count++
					if fn.Name.Name != "inspectionEnchantment" {
						t.Errorf("%s:%s builds another cell spell inspection", name, fn.Name.Name)
					}
				}
				return true
			})
		}
	}
	if count != 1 {
		t.Fatalf("%d cell spell inspection builders, want 1", count)
	}
}
