package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestSpritePairBuildersStayShared(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for file, routes := range map[string]map[string]string{
		"pkg/ui/statics.go":          {"planeSprites": "spritePair", "drawStructuresFlat": "spritePair"},
		"pkg/ui/spellbolt.go":        {"drawSpellArt": "spritePair"},
		"pkg/game/sackboundaries.go": {"LoadSackBoundaries": "boundaryFrames"},
		"pkg/game/statics.go":        {"staticsOf": "boundaryFrames"},
		"pkg/game/structures.go":     {"LoadStructures": "boundaryFrames"},
		"pkg/game/units.go":          {"unitSet": "boundaryFrames"},
		"pkg/game/heroart.go":        {"LoadHeroBody": "boundaryFrames", "heroDyingBody": "boundaryFrames"},
		"pkg/game/projectiles.go":    {"projectileSet": "boundaryFrames"},
	} {
		b, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(token.NewFileSet(), file, b, 0)
		if err != nil {
			t.Fatal(err)
		}
		for function, builder := range routes {
			var body *ast.BlockStmt
			for _, d := range f.Decls {
				if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == function {
					body = fn.Body
				}
			}
			if body == nil {
				t.Fatalf("%s lost %s", file, function)
			}
			if !callsSpriteBuilder(body, builder) {
				t.Errorf("%s.%s must use %s", file, function, builder)
			}
			if callsSpriteBuilder(body, "absentBuilder") {
				t.Fatal("call check accepts an absent builder")
			}
		}
	}
}

func callsSpriteBuilder(body *ast.BlockStmt, builder string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if s, ok := call.Fun.(*ast.SelectorExpr); ok && s.Sel.Name == builder {
				found = true
			}
		}
		return true
	})
	return found
}
