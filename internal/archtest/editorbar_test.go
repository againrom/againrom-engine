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
		{"mapeditormode.go", "closeBarWindow", "barWindowSet"},
		{"mapeditorwindow.go", "barAct", "restoreBarWindows"},
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

func TestEditorBarRememberedSetsUseOneLeaveAndOpen(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "pkg", "ui", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	reads := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				idx, ok := n.(*ast.IndexExpr)
				if !ok {
					return true
				}
				sel, ok := idx.X.(*ast.SelectorExpr)
				if ok && sel.Sel.Name == "barSets" {
					reads++
					if fn.Name.Name != "closeBarWindow" && fn.Name.Name != "barAct" {
						t.Errorf("%s accesses a remembered set outside the bar leave/open path", fn.Name.Name)
					}
				}
				return true
			})
		}
	}
	if reads != 2 {
		t.Fatalf("%d remembered-set accesses, want the one leave and one open", reads)
	}
}
