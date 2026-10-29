package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// The town hero pane has one producer in both games: in pkg/game only
// heroPane reads the pane's corner art and names the shop pane session.
func TestTownHeroPaneHasOneProducer(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	count := map[string]int{}
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/game/") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, named := sel.X.(*ast.Ident)
				switch {
				case sel.Sel.Name == "CharacterPaneShop" && named && pkg.Name == "ui":
				case sel.Sel.Name == "characterCorners":
				default:
					return true
				}
				count[sel.Sel.Name]++
				if name != "pkg/game/heropane.go" || fn.Name.Name != "heroPane" {
					t.Errorf("%s:%s produces a town hero pane outside heroPane (%s)", name, fn.Name.Name, sel.Sel.Name)
				}
				return true
			})
		}
	}
	if count["CharacterPaneShop"] != 1 || count["characterCorners"] != 1 {
		t.Fatalf("pkg/game names ui.CharacterPaneShop %d times and characterCorners %d times, want 1 each",
			count["CharacterPaneShop"], count["characterCorners"])
	}
}
