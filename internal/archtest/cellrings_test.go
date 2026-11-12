package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestCellRingWalkHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	readers := map[string]bool{"settleFor": false, "OrderSackCollect": false}
	builders := 0
	for _, name := range []string{"route.go", "sackpickup.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "pkg", "sim", name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if fn.Name.Name == "walkCellRings" {
				builders++
			}
			if _, reader := readers[fn.Name.Name]; !reader {
				continue
			}
			calls := 0
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "walkCellRings" {
						calls++
					}
				}
				if _, loop := n.(*ast.ForStmt); loop {
					t.Errorf("%s walks cells outside walkCellRings", fn.Name.Name)
				}
				return true
			})
			readers[fn.Name.Name] = calls == 1
		}
	}
	if builders != 1 {
		t.Fatalf("found %d cell ring builders, want one", builders)
	}
	for reader, calls := range readers {
		if !calls {
			t.Errorf("%s does not use the ring builder once", reader)
		}
	}
}
