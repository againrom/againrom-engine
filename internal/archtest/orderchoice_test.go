package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestOrderTargetSelectionHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for path, name := range map[string]string{
		"escort.go": "acquireStanding", "savedgroupsai.go": "savedDecision", "reacquire.go": "reacquisitionVictim",
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "pkg", "sim", path), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		found, calls := false, 0
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != name {
				continue
			}
			found = true
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				method, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				switch method.Sel.Name {
				case "chooseTarget":
					calls++
				case "candidateCost", "facingArc":
					t.Errorf("%s.%s ranks outside chooseTarget", path, name)
				}
				return true
			})
		}
		if !found || calls != 1 {
			t.Errorf("%s.%s has %d chooser calls, found=%v", path, name, calls, found)
		}
	}
}
