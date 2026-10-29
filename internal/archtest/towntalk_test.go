package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// The town talk of both games has one view producer and one page turn: in
// pkg/game only townTalkSeat.view names ui.TownTalkView or ui.TownTalkPage,
// and townTalkSeat is the only declarer of a turnTownTalk method.
func TestTownTalkHasOneProducer(t *testing.T) {
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
	views, turns := 0, 0
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
			if !ok {
				continue
			}
			recv := receiverName(fn.Recv)
			if fn.Name.Name == "turnTownTalk" {
				turns++
				if name != "pkg/game/towntalk.go" || recv != "townTalkSeat" {
					t.Errorf("%s:%s.%s is a second town talk page turn", name, recv, fn.Name.Name)
				}
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, named := sel.X.(*ast.Ident)
				if !named || pkg.Name != "ui" || (sel.Sel.Name != "TownTalkView" && sel.Sel.Name != "TownTalkPage") {
					return true
				}
				views++
				if name != "pkg/game/towntalk.go" || recv != "townTalkSeat" || fn.Name.Name != "view" {
					t.Errorf("%s:%s.%s produces a town talk view outside townTalkSeat.view (ui.%s)", name, recv, fn.Name.Name, sel.Sel.Name)
				}
				return true
			})
		}
	}
	if views == 0 || turns != 1 {
		t.Fatalf("pkg/game names the town talk view %d times and declares %d page turns, want the producer's and 1", views, turns)
	}
}
