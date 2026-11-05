package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestEditorBarWindowsShareStateAndClose(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []struct{ file, function, callee string }{
		{"mapeditorwindow.go", "barEntries", "barWindow"},
		{"mapeditorwindow.go", "barAct", "barWindow"},
		{"mapeditorwindow.go", "barAct", "closeBarWindow"},
		{"mapeditorwindow.go", "openRecords", "closeBarWindow"},
		{"mapeditormode.go", "openPanel", "closeBarWindow"},
		{"mapeditorbrush.go", "objectBrush", "closeBarWindow"},
		{"mapeditormode.go", "closeBarWindow", "closeList"},
		{"mapeditormode.go", "closeBarWindow", "leave"},
	} {
		src, err := os.ReadFile(filepath.Join(root, "pkg", "ui", route.file))
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(token.NewFileSet(), route.file, src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Name.Name != route.function || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == route.callee {
						found = true
					}
				}
				return true
			})
		}
		if !found {
			t.Errorf("%s must read %s", route.function, route.callee)
		}
	}
}
