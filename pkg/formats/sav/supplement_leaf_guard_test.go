package sav

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const supplementLeafFile = "pkg/formats/sav/supplement_leaf.go"

var supplementLeafLiteral = regexp.MustCompile(`(^|/)Againrom[A-Z][A-Za-z0-9]*$`)

var supplementPathConstants = map[string]bool{"NativeActionsPath": true, "NativeModsPath": true, "NativeRandomPath": true, "NativeSessionPath": true}

// supplementLeafViolations reports a leaf literal, a path constant or, in
// this package, a case-folded lookup outside the descriptor file.
func supplementLeafViolations(rel string, src []byte) ([]string, error) {
	if rel == supplementLeafFile {
		return nil, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rel, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	inSav := strings.HasPrefix(rel, "pkg/formats/sav/") && !strings.Contains(strings.TrimPrefix(rel, "pkg/formats/sav/"), "/")
	var out []string
	report := func(n ast.Node, what string) {
		out = append(out, fset.Position(n.Pos()).String()+": "+what)
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BasicLit:
			if n.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(n.Value)
			if err == nil && supplementLeafLiteral.MatchString(value) {
				report(n, "supplement leaf literal "+n.Value)
			}
			// A refusal sentence may name a leaf; a bare token may not.
			for _, d := range supplementLeaves {
				if err == nil && strings.Contains(value, d.name) && !strings.Contains(value, " ") && !supplementLeafLiteral.MatchString(value) {
					report(n, "literal carries supplement leaf name "+d.name)
				}
			}
		case *ast.Ident:
			if supplementPathConstants[n.Name] {
				report(n, "supplement leaf path constant "+n.Name)
			}
		case *ast.SelectorExpr:
			if x, ok := n.X.(*ast.Ident); ok && inSav && x.Name == "strings" && n.Sel.Name == "EqualFold" {
				report(n, "case-folded lookup outside the descriptor route")
			}
		}
		return true
	})
	return out, nil
}

func engineModuleRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatal("engine module root not found:", err)
	}
	return root
}

func TestSupplementLeavesHaveOneRoute(t *testing.T) {
	root := engineModuleRoot(t)
	files := 0
	sawDescriptor := false
	err := filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			if path != root {
				if name := e.Name(); name == "testdata" || strings.HasPrefix(name, ".") {
					return filepath.SkipDir
				}
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files++
		if rel == supplementLeafFile {
			sawDescriptor = true
			for _, d := range supplementLeaves {
				if !strings.Contains(string(src), `"`+d.name+`"`) {
					t.Errorf("descriptor file does not name leaf %s", d.name)
				}
			}
		}
		found, err := supplementLeafViolations(rel, src)
		if err != nil {
			return err
		}
		for _, v := range found {
			t.Error(v)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sawDescriptor || files < 100 {
		t.Fatalf("scan saw %d production files and descriptor %v", files, sawDescriptor)
	}
}

func TestSupplementLeafGuardRefusesEachBypass(t *testing.T) {
	for _, c := range []struct {
		name, rel, src string
	}{
		{"leaf name", "pkg/game/x.go", `package game; var n = "AgainromMods"`},
		{"leaf path", "pkg/game/x.go", `package game; const p = "/CurrentState/AgainromActions"`},
		{"new leaf path", "pkg/formats/sav/x.go", `package sav; const p = "/CurrentState/AgainromLater"`},
		{"leaf name inside a literal", "pkg/game/x.go", `package game; var n = "AgainromSeed:"`},
		{"path constant", "pkg/game/x.go", `package game; import "againrom/pkg/formats/sav"; var p = sav.NativeRandomPath`},
		{"own lookup", "pkg/formats/sav/x.go", `package sav; import "strings"; func f(a, b string) bool { return strings.EqualFold(a, b) }`},
		{"shape entry", "pkg/formats/sav/x.go", `package sav; var l = stateLeaf{"AgainromSeed", 6}`},
	} {
		got, err := supplementLeafViolations(c.rel, []byte(c.src))
		if err != nil {
			t.Fatal(c.name, err)
		}
		if len(got) == 0 {
			t.Errorf("%s: bypass not reported", c.name)
		}
	}
	for _, c := range []struct{ rel, src string }{
		{"pkg/game/x.go", `package game; var p = "/CurrentState/InBattle"`},
		{"pkg/game/x.go", `package game; import "strings"; func f(a, b string) bool { return strings.EqualFold(a, b) }`},
		{"pkg/game/x.go", `package game; var p = "Againrom/"`},
		{"pkg/formats/sav/x.go", `package sav; var e = "sav: AgainromSeed names a mode"`},
	} {
		if got, err := supplementLeafViolations(c.rel, []byte(c.src)); err != nil || len(got) != 0 {
			t.Errorf("%s reported %v %v", c.src, got, err)
		}
	}
}
