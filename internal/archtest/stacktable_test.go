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

// The stack a shop draws a stacking row at is declared once, in
// pkg/shopstock's stack table. No shop file of either game declares a stack
// range of its own, and both fills join a pick through shopstock.Filling.
func TestShopStackTableIsDeclaredOnce(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, pattern := range []string{"pkg/game/shop*.go", "pkg/secondshop/*.go"} {
		names, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, names...)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			s, ok := n.(*ast.StructType)
			if !ok {
				return true
			}
			// A stack range is a least with a most or a span, or a
			// drawn-once flag.
			has := map[string]bool{}
			for _, field := range s.Fields.List {
				for _, id := range field.Names {
					has[strings.ToLower(id.Name)] = true
				}
			}
			if has["least"] && (has["most"] || has["span"]) || has["once"] {
				t.Errorf("%s declares a stack range; the stack table is shopstock's", fset.Position(s.Pos()))
			}
			return true
		})
	}
	for name, want := range map[string][]string{
		"pkg/game/shopconsumables.go":  {"shopstock.PotionStack", "shopstock.ScrollStack"},
		"pkg/game/shopstockprofile.go": {"shopstock.Filling("},
		"pkg/secondshop/fill.go":       {"shopstock.PotionStack", "shopstock.ScrollStack", "shopstock.UnitStack", "shopstock.Filling("},
		"pkg/shopstock/stacks.go":      {"var stackTable = "},
	} {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		for _, w := range want {
			if !strings.Contains(string(b), w) {
				t.Errorf("%s does not name %s", name, w)
			}
		}
	}
}
