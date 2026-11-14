package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// The guard post's block scan is a chooseTarget row, not its own walk.
func TestPostEngageUsesTheRankingBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "pkg", "sim", "guardarm.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Name.Name != "postEngage" {
			continue
		}
		found = true
		calls := 0
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.RangeStmt, *ast.ForStmt:
				t.Error("postEngage walks its candidates outside chooseTarget")
			case *ast.CallExpr:
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "chooseTarget" {
					calls++
				}
			}
			return true
		})
		if calls != 1 {
			t.Errorf("postEngage calls chooseTarget %d times, want once", calls)
		}
	}
	if !found {
		t.Fatal("postEngage not found in pkg/sim/guardarm.go")
	}
}
