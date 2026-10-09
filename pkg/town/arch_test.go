package town

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The composer holds no fact of any game: its non-test source imports only
// the standard library, carries no integer literal beyond the counting,
// channel and bit-width values below, no float or rune literal, and no string
// literal that could name a resource, a sound or a path. The entry-name
// formats ("%s/%d") are the only strings with a slash.
var allowedInts = map[string]bool{"0": true, "1": true, "2": true, "3": true, "4": true, "8": true, "10": true, "16": true, "32": true, "64": true}

func entryFormat(s string) bool {
	rest := strings.NewReplacer("%s", "", "%d", "").Replace(s)
	return strings.Trim(rest, "/") == ""
}

func TestComposerHoldsNoGameFact(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	checked := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		checked++
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if first, _, _ := strings.Cut(path, "/"); strings.Contains(first, ".") || first == "againrom" {
				t.Errorf("%s imports %s; the composer imports only the standard library", name, path)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.ImportSpec:
				return false
			case *ast.BasicLit:
				switch x.Kind {
				case token.INT:
					if !allowedInts[x.Value] {
						t.Errorf("%s: integer literal %s", fset.Position(x.Pos()), x.Value)
					}
				case token.FLOAT, token.IMAG, token.CHAR:
					t.Errorf("%s: literal %s", fset.Position(x.Pos()), x.Value)
				case token.STRING:
					s, _ := strconv.Unquote(x.Value)
					if !entryFormat(s) && (strings.Contains(s, "/") || strings.Contains(s, ".") || strings.Contains(s, "\\")) {
						t.Errorf("%s: string literal %q looks like a path or resource key", fset.Position(x.Pos()), s)
					}
				}
			}
			return true
		})
	}
	if checked == 0 {
		t.Fatal("no source checked")
	}
}
