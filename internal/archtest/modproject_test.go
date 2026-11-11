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

func TestEditorModWritesUseProjectWriter(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "pkg", "game", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		name := filepath.Base(path)
		if strings.HasSuffix(name, "_test.go") || name == "mapedittestplay_files.go" || !(strings.HasPrefix(name, "mapedit") || strings.HasPrefix(name, "mapinspection") || name == "maptextpages.go" || name == "modmapsave.go" || name == "modspeakers.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if fn, ok := call.Fun.(*ast.SelectorExpr); ok {
				if pkg, ok := fn.X.(*ast.Ident); ok && (pkg.Name == "os" && (fn.Sel.Name == "WriteFile" || fn.Sel.Name == "OpenFile" || fn.Sel.Name == "Create" || fn.Sel.Name == "Rename") || pkg.Name == "hostfile" && fn.Sel.Name == "Commit") {
					t.Errorf("%s writes outside the staged project writer", name)
				}
			}
			if fn, ok := call.Fun.(*ast.Ident); ok && (fn.Name == "writeModMap" || fn.Name == "writeModFile" || fn.Name == "writeSpeakerFiles") {
				t.Errorf("%s calls retired writer %s", name, fn.Name)
			}
			return true
		})
	}
}
