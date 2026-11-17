package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// The editor ink scan: every colour an editor window shows is named in the
// one ink table, pkg/ui/editorink.go. A color.RGBA literal with elements in
// another editor file (editor*, mapeditor*, popup*) is a second table,
// unless the file is listed below as remaining. The list may only shrink.

const editorInkTable = "pkg/ui/editorink.go"

// editorInkRemaining are the editor files that still name their own
// colours, with the count each holds.
var editorInkRemaining = map[string]int{
	"pkg/ui/mapeditorpassability.go": 4,
	"pkg/ui/mapeditorcapture.go":     2,
}

func editorInkSurface(name string) bool {
	base, ok := strings.CutPrefix(name, "pkg/ui/")
	if !ok || strings.Contains(base, "/") || !strings.HasSuffix(base, ".go") || strings.HasSuffix(base, "_test.go") {
		return false
	}
	return strings.HasPrefix(base, "editor") || strings.HasPrefix(base, "mapeditor") || strings.HasPrefix(base, "popup")
}

// CheckEditorInk reports the editor files that name a colour outside the
// ink table, and a remaining file whose count changed.
func CheckEditorInk(files map[string]string) []Violation {
	var vs []Violation
	seen := map[string]bool{}
	for _, name := range sortedKeys(files) {
		if !editorInkSurface(name) || name == editorInkTable {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "parse: " + err.Error()})
			continue
		}
		n := 0
		ast.Inspect(f, func(node ast.Node) bool {
			lit, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if list, ok := lit.Type.(*ast.ArrayType); ok && isColorRGBA(list.Elt) {
				n += len(lit.Elts) // a list of colours: every element is one
				return false
			}
			if isColorRGBA(lit.Type) && len(lit.Elts) > 0 && numbersOnly(lit.Elts) {
				n++
			}
			return true
		})
		want, listed := editorInkRemaining[name]
		seen[name] = listed
		switch {
		case !listed && n > 0:
			vs = append(vs, Violation{From: name, Reason: "names a colour outside the editor ink table " + editorInkTable})
		case listed && n != want:
			vs = append(vs, Violation{From: name, Reason: "remaining editor colours changed; lower editorInkRemaining or move them to the table"})
		}
	}
	for name := range editorInkRemaining {
		if !seen[name] {
			vs = append(vs, Violation{From: name, Reason: "listed in editorInkRemaining but not scanned; drop the entry"})
		}
	}
	return vs
}

func isColorRGBA(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "RGBA" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "color"
}

// numbersOnly reports a literal written in numbers, not a colour computed
// from others.
func numbersOnly(elts []ast.Expr) bool {
	for _, e := range elts {
		if _, ok := e.(*ast.BasicLit); !ok {
			return false
		}
	}
	return true
}
