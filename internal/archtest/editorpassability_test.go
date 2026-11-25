package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

func TestEditorPassabilityHasOneMarkAndDisplayRoute(t *testing.T) {
	root := moduleRootForTest(t)
	for _, path := range []string{"pkg/mapload/groundarms.go", "pkg/game/mapeditrow.go", "pkg/ui/mapeditorpassability.go"} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(root, path), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				switch id.Name {
				case "MarkedArms", "MarkArmRows", "groundArms", "armRows", "passValues", "passValueAt", "passValueFill":
					t.Errorf("%s: separate passability route %s", fset.Position(id.Pos()), id.Name)
				}
			}
			return true
		})
	}
}
