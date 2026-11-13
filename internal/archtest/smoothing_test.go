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

func TestSmoothingUsesOneStoreKeyAndFilterSelector(t *testing.T) {
	root := moduleRootForTest(t)
	for _, name := range []string{"pkg/game", "pkg/ui", "cmd/starter"} {
		files, err := filepath.Glob(filepath.Join(root, name, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			f, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if s, ok := n.(*ast.BasicLit); ok && s.Kind == token.STRING && s.Value == `"FrameSmoothing"` && filepath.Base(path) != "tipstore.go" {
					t.Errorf("%s restores an independent frame preference", path)
				}
				return true
			})
			if filepath.Base(path) != "mapzoom.go" && filepath.Base(path) != "devicescale.go" && filepath.Base(path) != "framesmoothing.go" {
				continue
			}
			for _, d := range f.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || (fn.Name.Name != "mapFilter" && fn.Name.Name != "canvasFilter" && fn.Name.Name != "frameFilter") {
					continue
				}
				if len(fn.Body.List) != 1 {
					t.Errorf("%s %s decides a filter outside filterOf", path, fn.Name)
					continue
				}
				ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
				if !ok || len(ret.Results) != 1 {
					t.Errorf("%s %s is not a selector call", path, fn.Name)
					continue
				}
				call, ok := ret.Results[0].(*ast.CallExpr)
				if !ok {
					t.Errorf("%s %s bypasses filterOf", path, fn.Name)
					continue
				}
				id, ok := call.Fun.(*ast.Ident)
				if !ok || id.Name != "filterOf" {
					t.Errorf("%s %s bypasses filterOf", path, fn.Name)
				}
			}
		}
	}
}
