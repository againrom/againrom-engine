package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The character generator holds no fact of any game: its source files carry
// no integer literal beyond the counting and bit-width values below, no float
// or rune literal, and no string literal that could name a resource, a sound
// or a path. Every coordinate, mask byte, text slot, timing and key comes
// from a generator description.
var generatorAllowedInts = map[string]bool{"0": true, "1": true, "2": true, "3": true, "4": true, "8": true, "10": true, "16": true, "32": true, "64": true,
	// An opaque alpha and the printable ASCII bounds of the name encoder.
	"0xff": true, "0x20": true, "0x7f": true}

// generatorSources are the generator's own files: the description, the model,
// its pages, its caret, its sparkle and its highlight cycle. The headless
// surface (its harness clock and ASCII folding) and the shared sound voices
// are not generator facts.
var generatorSources = []string{"generator.go", "chargen.go", "chargen_page.go", "chargen_caret.go", "chargen_sparkle.go", "guidedcycle.go"}

func TestGeneratorHoldsNoGameFact(t *testing.T) {
	fset := token.NewFileSet()
	for _, name := range generatorSources {
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.ImportSpec:
				return false
			case *ast.BasicLit:
				switch x.Kind {
				case token.INT:
					if !generatorAllowedInts[x.Value] {
						t.Errorf("%s: integer literal %s", fset.Position(x.Pos()), x.Value)
					}
				case token.FLOAT, token.IMAG, token.CHAR:
					t.Errorf("%s: literal %s", fset.Position(x.Pos()), x.Value)
				case token.STRING:
					s, _ := strconv.Unquote(x.Value)
					if s != "/" && !strings.ContainsAny(s, " %") && strings.ContainsAny(s, "/.\\") {
						t.Errorf("%s: string literal %q looks like a path or resource key", fset.Position(x.Pos()), s)
					}
				}
			}
			return true
		})
	}
}
