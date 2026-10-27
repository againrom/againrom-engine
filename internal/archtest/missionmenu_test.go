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

func TestMissionMenuLayersUseOneHost(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "pkg", "ui")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				name, ok := call.Fun.(*ast.Ident)
				if ok && name.Name == "blitMenuOverMap" {
					calls++
					if fn.Name.Name != "drawGameMenuOverMap" {
						t.Errorf("%s adds a mission-menu host in %s", fn.Name.Name, entry.Name())
					}
				}
				return true
			})
		}
	}
	if calls != 1 {
		t.Fatalf("mission-menu blit calls %d, want one shared host", calls)
	}
}
