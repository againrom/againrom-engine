package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestEditorTextDeletionUsesSharedModel(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct{ file, function, builder string }{
		{"editorinput.go", "textKey", "Erase"},
		{"mapeditorkeys.go", "keyBackspace", "EditText"},
	} {
		file := filepath.Join(root, "pkg", "ui", row.file)
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		found, calls := false, 0
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != row.function {
				continue
			}
			found = true
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.SliceExpr:
					t.Errorf("%s slices text outside the shared model", row.function)
				case *ast.CallExpr:
					switch name := x.Fun.(type) {
					case *ast.Ident:
						if name.Name == row.builder {
							calls++
						}
					case *ast.SelectorExpr:
						if name.Sel.Name == row.builder {
							calls++
						}
					}
				}
				return true
			})
		}
		if !found || calls != 1 {
			t.Errorf("%s must call %s once, got %d", row.function, row.builder, calls)
		}
	}
}
