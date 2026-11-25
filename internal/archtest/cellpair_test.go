package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// cellPairWriters are the files under pkg/game that assign a field's cell
// pair, own-cell mark or cell word: the format dictionary's field row states
// them, so pkg/game only copies them.
func cellPairWriters(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/game/") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if as, ok := n.(*ast.AssignStmt); ok {
				for _, l := range as.Lhs {
					if sel, ok := l.(*ast.SelectorExpr); ok && (sel.Sel.Name == "CellPair" || sel.Sel.Name == "CellOwn") {
						out = append(out, name)
						break
					}
				}
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

func TestCellPairScanNamesAnAssigner(t *testing.T) {
	got := cellPairWriters(t, map[string]string{
		"pkg/game/a.go":    "package game\nfunc f(o []F) { o[0].CellPair, o[0].CellOwn = \"y\", true }\n",
		"pkg/game/b.go":    "package game\nfunc f(o []F) { x := F{CellPair: o[0].CellPair}; _ = x }\n",
		"pkg/mapedit/c.go": "package mapedit\nfunc f(o []F) { o[0].CellPair = \"y\" }\n",
	})
	if len(got) != 1 || got[0] != "pkg/game/a.go" {
		t.Fatalf("cellPairWriters = %v", got)
	}
}

// TestCellPairIsDeclaredByTheFieldRow holds pkg/game to copying a field's cell
// pair from the format dictionary.
func TestCellPairIsDeclaredByTheFieldRow(t *testing.T) {
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
	if got := cellPairWriters(t, files); len(got) != 0 {
		t.Fatalf("pkg/game assigns a cell pair: %v", got)
	}
}
