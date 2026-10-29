package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestEditorNPCChoicesHaveOneBuilder(t *testing.T) {
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
	count := 0
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/game/map") {
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
				loop, ok := n.(*ast.RangeStmt)
				if !ok {
					return true
				}
				call, ok := loop.X.(*ast.CallExpr)
				if !ok {
					return true
				}
				records, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || records.Sel.Name != "Records" {
					return true
				}
				catalog, ok := records.X.(*ast.SelectorExpr)
				if !ok || catalog.Sel.Name != "NPCCatalog" {
					return true
				}
				count++
				if name != "pkg/game/mapinspectionnpc.go" || fn.Name.Name != "npcChoices" {
					t.Errorf("%s:%s builds NPC choices outside npcChoices", name, fn.Name.Name)
				}
				return true
			})
		}
	}
	if count != 1 {
		t.Fatalf("NPC catalog choice walks = %d, want 1", count)
	}
}
