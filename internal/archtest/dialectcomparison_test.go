package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// keptDialectComparisons are the production comparisons with a script dialect
// that stay: the chooser, and the checks that a persisted or decoded dialect
// is valid or consents to the state beside it. Any other file asks the game
// through the game profile.
var keptDialectComparisons = map[string]int{
	"pkg/sim/gameprofile.go":     1,
	"pkg/sim/rom2statebinary.go": 3,
}

func isDialectName(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident:
		return strings.HasPrefix(x.Name, "ScriptROM")
	case *ast.SelectorExpr:
		return strings.HasPrefix(x.Sel.Name, "ScriptROM")
	}
	return false
}

func isOpsName(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.SelectorExpr:
		return x.Sel.Name == "OpsFirst" || x.Sel.Name == "OpsSecond"
	}
	return false
}

// A rule that differs by game is a field of the game profile (pkg/sim) or of
// the script vocabulary (scriptprofile), not a comparison with the dialect or
// the operation table at the point of use.
func TestDialectComparisonsStayWithTheirChooser(t *testing.T) {
	root := moduleRootForTest(t)
	got := map[string]int{}
	for _, dir := range []string{"sim", "game", "mapload"} {
		base := filepath.Join(root, "pkg", dir)
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			ast.Inspect(f, func(n ast.Node) bool {
				b, ok := n.(*ast.BinaryExpr)
				if !ok || (b.Op != token.EQL && b.Op != token.NEQ) {
					return true
				}
				if isDialectName(b.X) || isDialectName(b.Y) {
					got[rel]++
				}
				if isOpsName(b.X) || isOpsName(b.Y) {
					t.Errorf("%s compares an operation table; the rule belongs in scriptprofile.Vocabulary", rel)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for file, n := range got {
		if keptDialectComparisons[file] != n {
			t.Errorf("%s compares a script dialect %d times, %d kept; a game rule belongs in the game profile", file, n, keptDialectComparisons[file])
		}
	}
	for file, n := range keptDialectComparisons {
		if got[file] != n {
			t.Errorf("%s compares a script dialect %d times, %d kept", file, got[file], n)
		}
	}
}
