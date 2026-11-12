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

func TestBaseMenuActionsUseTheModRegistration(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "pkg", "game")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	reads := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(src), "baseMenuActions") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), entry.Name(), src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && id.Name == "baseMenuActions" {
					reads++
					if fn.Name.Name != "ModScreens" || entry.Name() != "modscreens.go" {
						t.Errorf("base menu actions read outside registration: %s:%s", entry.Name(), fn.Name.Name)
					}
				}
				return true
			})
		}
	}
	if reads != 1 {
		t.Fatalf("base menu action readers=%d, want one registration", reads)
	}
}
