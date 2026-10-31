package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"testing"
)

const windowApplyFile = "pkg/ui/windowmode.go"

func windowSizeSetters(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
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
				if call, ok := n.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "SetWindowSize" {
						out = append(out, name+":"+fn.Name.Name)
					}
				}
				return true
			})
		}
	}
	sort.Strings(out)
	return out
}

func TestWindowSizeScanNamesEachCaller(t *testing.T) {
	got := windowSizeSetters(t, map[string]string{
		windowApplyFile:     "package ui\nfunc applyWindow() { ebiten.SetWindowSize(1, 2) }\n",
		"cmd/x/main.go":     "package main\nfunc run() { ebiten.SetWindowSize(1, 2); ebiten.SetWindowSizeLimits(1, 2, -1, -1) }\n",
		"pkg/ui/another.go": "package ui\nfunc f() { ebiten.SetWindowTitle(\"\") }\n",
	})
	if len(got) != 2 || got[0] != "cmd/x/main.go:run" || got[1] != windowApplyFile+":applyWindow" {
		t.Fatalf("windowSizeSetters = %v", got)
	}
}

// SetWindowSize has applyWindow as its only production caller.
func TestProgramWindowFormHasOneApplyFunction(t *testing.T) {
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
	if _, ok := files[windowApplyFile]; !ok {
		t.Fatalf("%s is missing; the scan guards nothing", windowApplyFile)
	}
	got := windowSizeSetters(t, files)
	if len(got) != 1 || got[0] != windowApplyFile+":applyWindow" {
		t.Fatalf("SetWindowSize callers = %v, want only %s:applyWindow", got, windowApplyFile)
	}
}
