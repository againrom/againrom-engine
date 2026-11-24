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

// The map script compiler reads its game's rules from scriptprofile.Profile,
// not from the dialect: pkg/mapload/script.go names no sim.ScriptROM1 or
// sim.ScriptROM2 selector.
func TestScriptCompilerNamesNoDialect(t *testing.T) {
	path := filepath.Join(moduleRootForTest(t), "pkg", "mapload", "script.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), path, src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "sim" && strings.HasPrefix(sel.Sel.Name, "ScriptROM") {
			found = append(found, sel.Sel.Name)
		}
		return true
	})
	if len(found) != 0 {
		t.Fatalf("pkg/mapload/script.go names a game dialect %d times: %v; the rule belongs in scriptprofile.Profile", len(found), found)
	}
}
