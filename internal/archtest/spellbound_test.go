package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// No spell bound in these files compares with a literal 28 or 29.
func TestSpellBoundHasNoLiteral(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"pkg/mapload/spell.go", "pkg/sim/spellbook.go", "pkg/game/shoproom.go",
		"pkg/formats/sav/city.go", "pkg/data/spellpresentation.go",
	} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(root, filepath.FromSlash(name)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.BinaryExpr:
				switch n.Op {
				case token.LSS, token.LEQ, token.GTR, token.GEQ:
					if spellLiteral(n.X) || spellLiteral(n.Y) {
						t.Errorf("%s: spell bound compared with a literal", fset.Position(n.Pos()))
					}
				}
			case *ast.KeyValueExpr:
				if key, ok := n.Key.(*ast.Ident); ok && key.Name == "LastSoundRow" {
					if _, lit := n.Value.(*ast.BasicLit); lit {
						t.Errorf("%s: LastSoundRow is a literal", fset.Position(n.Pos()))
					}
				}
			}
			return true
		})
	}
}

func spellLiteral(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && (lit.Value == "28" || lit.Value == "29")
}
