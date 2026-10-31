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

func pictureLiteralTests(t *testing.T, files map[string]string) []string {
	t.Helper()
	picture := func(x ast.Expr) bool {
		switch x := x.(type) {
		case *ast.Ident:
			return x.Name == "Picture" || x.Name == "picture"
		case *ast.SelectorExpr:
			return x.Sel.Name == "Picture"
		}
		return false
	}
	number := func(x ast.Expr) bool {
		lit, ok := x.(*ast.BasicLit)
		return ok && lit.Kind == token.INT
	}
	var out []string
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/sim/") && !strings.HasPrefix(name, "pkg/game/") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if b, ok := n.(*ast.BinaryExpr); ok && (b.Op == token.EQL || b.Op == token.NEQ) &&
				(picture(b.X) && number(b.Y) || number(b.X) && picture(b.Y)) {
				out = append(out, name)
			}
			return true
		})
	}
	sort.Strings(out)
	return out
}

func TestPictureLiteralScanNamesEachTest(t *testing.T) {
	got := pictureLiteralTests(t, map[string]string{
		"pkg/sim/a.go":   "package sim\nfunc f(p P) bool { return p.Picture == 20 }\n",
		"pkg/game/b.go":  "package game\nfunc g(picture int32) bool { return 51 != picture }\n",
		"pkg/game/c.go":  "package game\n// p.Picture == 20\nfunc h(p P) bool { return p.Picture < 0 || p.Picture == n }\n",
		"pkg/rules/d.go": "package rules\nfunc k(p P) bool { return p.Picture == 30 }\n",
	})
	if strings.Join(got, ",") != "pkg/game/b.go,pkg/sim/a.go" {
		t.Fatalf("pictureLiteralTests = %v", got)
	}
}

// A projectile picture's capabilities are pkg/rules' picture rows: no source
// in pkg/sim or pkg/game tests a picture against a number.
func TestPictureCapabilitiesAreRows(t *testing.T) {
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
	if _, ok := files["pkg/sim/savedcontinuation.go"]; !ok {
		t.Fatal("pkg/sim/savedcontinuation.go is missing; the scan guards nothing")
	}
	for _, file := range pictureLiteralTests(t, files) {
		t.Errorf("%s tests a projectile picture against a number; read rules.PictureCapabilityOf", file)
	}
}
