package ui

// Enter has two keys: the main one and the numpad's. Windows posts the same
// virtual key for both and marks the numpad one only with the extended-key
// flag, which the engine's key layer turns into a second key. Every reader of
// Enter has to take both, and no decoded key handler of the original reads the
// flag, so the two act alike.
//
// The binding is an Ebitengine call and no test may open a window, so it is read
// from the source (settingskeys_test.go); the reading itself and the routes it
// feeds are driven in numpadenter_test.go.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestNoReaderTakesTheMainEnterKeyAlone finds every production declaration in
// pkg and cmd that names the main Enter key and requires the same declaration
// to name the numpad's Enter key too. A new reader that takes one key fails
// here and names the declaration.
func TestNoReaderTakesTheMainEnterKeyAlone(t *testing.T) {
	var alone []string
	scanned := 0
	for _, root := range []string{filepath.Join("..", "..", "pkg"), filepath.Join("..", "..", "cmd")} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !strings.Contains(string(src), "KeyEnter") {
				return nil
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			scanned++
			for _, decl := range f.Decls {
				var main, numpad bool
				ast.Inspect(decl, func(n ast.Node) bool {
					sel, ok := n.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "ebiten" {
						switch sel.Sel.Name {
						case "KeyEnter":
							main = true
						case "KeyNumpadEnter", "KeyKPEnter":
							numpad = true
						}
					}
					return true
				})
				if main && !numpad {
					name := "declaration"
					switch d := decl.(type) {
					case *ast.FuncDecl:
						name = d.Name.Name
					case *ast.GenDecl:
						name = d.Tok.String()
					}
					alone = append(alone, filepath.ToSlash(path)+": "+name)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
	if scanned == 0 {
		t.Fatal("no production file names an Enter key; the scan found nothing to check")
	}
	sort.Strings(alone)
	for _, where := range alone {
		t.Errorf("%s reads ebiten.KeyEnter without ebiten.KeyNumpadEnter, so the numpad's Enter reaches nothing there", where)
	}
}

// TestTheEnterBindingReadsBothEnterKeysOnThePressEdge is the App's own reading:
// the Enter field of readAppInput is the press edge of either Enter key, one
// true per press, as every other key in that snapshot is.
func TestTheEnterBindingReadsBothEnterKeysOnThePressEdge(t *testing.T) {
	expr, ok := bindingSource(t)["Enter"]
	if !ok {
		t.Fatal("readAppInput names no Enter field, so no key reaches Enter")
	}
	const want = "enterPressed(inpututil.IsKeyJustPressed)"
	if expr != want {
		t.Errorf("Enter is bound to %q, want %q: the press edge of both Enter keys", expr, want)
	}
}
