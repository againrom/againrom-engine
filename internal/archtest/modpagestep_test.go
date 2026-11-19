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

// TestModPageHasOneStep states that a mod window page opens its edit only
// through modPageEdit: no page file calls modWritable, and only the step's
// file builds a missing-row error from a page's err.gone word.
func TestModPageHasOneStep(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "pkg", "game", "mapeditormod*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		base := filepath.Base(path)
		if strings.HasSuffix(base, "_test.go") || base == "mapeditormodpage.go" {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := ""
			switch fn := call.Fun.(type) {
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			case *ast.Ident:
				name = fn.Name
			}
			if name == "modWritable" {
				t.Errorf("%s: calls modWritable; a page edit opens through modPageEdit", base)
			}
			if name == "Errorf" || name == "New" {
				ast.Inspect(call, func(m ast.Node) bool {
					if lit, ok := m.(*ast.BasicLit); ok && strings.Contains(lit.Value, "err.gone") {
						t.Errorf("%s: builds a missing-row error from %s; use modGone", base, lit.Value)
					}
					return true
				})
			}
			return true
		})
	}
}
