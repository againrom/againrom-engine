package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestEditorChoiceListsUseOneRowBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	// The object window's row and the field prompt open their list through
	// openChoiceList, which builds a choiceList and its rows.
	for file, name := range map[string]string{"mapeditoradd.go": "addList", "editorchoices.go": "openChoiceList", "mapeditorobject.go": "objectRowPress", "mapeditorprompt.go": "openPromptList"} {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "pkg", "ui", file), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		found, shared := false, false
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != name {
				continue
			}
			found = true
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if literal, ok := n.(*ast.CompositeLit); ok {
					if id, ok := literal.Type.(*ast.Ident); ok && id.Name == "PickerRow" {
						t.Errorf("%s builds choice rows outside choiceRows", file)
					}
				}
				if call, ok := n.(*ast.CallExpr); ok {
					if method, ok := call.Fun.(*ast.SelectorExpr); ok && method.Sel.Name == "openChoiceList" {
						shared = true
					}
					if method, ok := call.Fun.(*ast.SelectorExpr); ok && method.Sel.Name == "rows" {
						switch x := method.X.(type) {
						case *ast.SelectorExpr:
							shared = shared || x.Sel.Name == "choices"
						case *ast.Ident:
							shared = shared || name == "openChoiceList"
						}
					}
				}
				return true
			})
		}
		if !found || !shared {
			t.Errorf("%s.%s must build its list through choiceList.rows", file, name)
		}
	}
}
