package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

const gridEntryFile = "pkg/mapedit/grid.go"

// gridCellWriters lists the pkg/mapedit functions that call setCell.
func gridCellWriters(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/mapedit/") {
			continue
		}
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
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "setCell" {
					out = append(out, name+":"+fn.Name.Name)
				}
				return true
			})
		}
	}
	sort.Strings(out)
	return out
}

func TestGridCellWriterScanNamesEachCaller(t *testing.T) {
	got := gridCellWriters(t, map[string]string{
		gridEntryFile:         "package mapedit\nfunc (e *Editor) SetCell() error { return e.setCell() }\n",
		"pkg/mapedit/fill.go": "package mapedit\nfunc (e *Editor) Fill() error { return e.setCell() }\n",
		"pkg/game/other.go":   "package game\nfunc f() { x.setCell() }\n",
	})
	if len(got) != 2 || got[0] != "pkg/mapedit/fill.go:Fill" || got[1] != gridEntryFile+":SetCell" {
		t.Fatalf("gridCellWriters = %v", got)
	}
}

// setCell has SetCell as its only caller.
func TestEditorGridMutationHasOneEntry(t *testing.T) {
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
	if _, ok := files[gridEntryFile]; !ok {
		t.Fatalf("%s is missing; the scan guards nothing", gridEntryFile)
	}
	got := gridCellWriters(t, files)
	if len(got) != 1 || got[0] != gridEntryFile+":SetCell" {
		t.Fatalf("setCell callers = %v, want only %s:SetCell", got, gridEntryFile)
	}
}
