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

// TestScriptNodeFieldsAreWrittenOnlyByPatchNode keeps the node record writer
// at one function: no non-test mapedit function other than patchNode may
// store a node field word.
func TestScriptNodeFieldsAreWrittenOnlyByPatchNode(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "pkg", "mapedit", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name == "patchNode" {
				continue
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "PutUint32" {
					return true
				}
				for _, arg := range call.Args {
					ast.Inspect(arg, func(m ast.Node) bool {
						if s, ok := m.(*ast.SelectorExpr); ok && strings.HasPrefix(s.Sel.Name, "ScriptNode") {
							t.Errorf("%s: %s writes %s outside patchNode", fset.Position(call.Pos()), fn.Name.Name, s.Sel.Name)
						}
						return true
					})
				}
				return true
			})
		}
	}
}
