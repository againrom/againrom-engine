package archtest

import (
	"go/ast"
	"strings"
	"testing"
)

// TestPathNewRequestHasOneCaller fails on a call to pathdialog.NewRequest
// outside Question.Request, the one route from a question to a Request.
func TestPathNewRequestHasOneCaller(t *testing.T) {
	fset, files := pathSources(t, "pkg", "cmd")
	allowed := 0
	eachFunc(files, func(rel string, fn *ast.FuncDecl) {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := ""
			switch f := call.Fun.(type) {
			case *ast.SelectorExpr:
				if x, ok := f.X.(*ast.Ident); ok && x.Name == "pathdialog" {
					name = f.Sel.Name
				}
			case *ast.Ident:
				if strings.HasPrefix(rel, "pkg/pathdialog/") {
					name = f.Name
				}
			}
			if name != "NewRequest" {
				return true
			}
			if rel == "pkg/pathdialog/pathdialog.go" && fn.Name.Name == "Request" && fn.Recv != nil {
				allowed++
				return true
			}
			t.Errorf("%s calls pathdialog.NewRequest; build the request from a pathdialog.Question with Request", fset.Position(call.Pos()))
			return true
		})
	})
	if allowed != 1 {
		t.Fatalf("Question.Request calls NewRequest %d times, want 1", allowed)
	}
}
