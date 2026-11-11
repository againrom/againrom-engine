package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEffectAttachmentHasOneTransaction(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "pkg", "sim")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.SelectorExpr:
					if n.Sel.Name == "opposite" || n.Sel.Name == "attachEffectTransaction" {
						got[name+":"+fn.Name.Name+":"+n.Sel.Name]++
					}
				case *ast.Ident:
					if n.Name == "keepsLargerOnRepeat" {
						got[name+":"+fn.Name.Name+":"+n.Name]++
					}
				}
				return true
			})
		}
	}
	want := map[string]int{
		"effect.go:attachEffectTransaction:opposite":                 1,
		"effect.go:attachEffectTransaction:keepsLargerOnRepeat":      1,
		"effect.go:attachEffect:attachEffectTransaction":             1,
		"sourceeffect.go:attachSourceEffect:attachEffectTransaction": 1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("attachment decision sites %v, want %v", got, want)
	}
}
