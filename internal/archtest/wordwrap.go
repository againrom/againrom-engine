package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The word-wrap scan: every multi-line text in pkg/ui is broken by the one line
// breaker in pkg/ui/linebreak.go. A surface is a row of that breaker (a
// lineRule) and an adapter that hands it a measure and a width. Outside that
// file the scan refuses the shapes a second breaker is made of:
//
//   - a function named for wrapping (hostedWrap returns a width: the exception);
//   - strings.Fields or strings.FieldsFunc;
//   - strings.Split, SplitN or Cut at a space;
//   - an Index, LastIndex, IndexByte, LastIndexByte or IndexRune of a space.
//
// It reads parsed syntax, so it proves these shapes are absent, not that no
// breaker can be spelled another way.

const wordWrapDir = "pkg/ui/"
const wordWrapHome = "pkg/ui/linebreak.go"

var wordWrapSplitters = map[string]bool{"Fields": true, "FieldsFunc": true}
var wordWrapSpaceCalls = map[string]bool{
	"Split": true, "SplitN": true, "Cut": true,
	"Index": true, "LastIndex": true, "IndexByte": true, "LastIndexByte": true, "IndexRune": true,
}

func wordWrapSpaceArg(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	if !ok {
		return false
	}
	switch lit.Kind {
	case token.STRING:
		s, err := strconv.Unquote(lit.Value)
		return err == nil && s == " "
	case token.CHAR:
		r, _, _, err := strconv.UnquoteChar(lit.Value[1:len(lit.Value)-1], '\'')
		return err == nil && r == ' '
	}
	return false
}

// CheckWordWrap reports every second-breaker shape in pkg/ui outside
// linebreak.go. Files are keyed by module-relative slash path.
func CheckWordWrap(files map[string]string) []Violation {
	names := make([]string, 0, len(files))
	for name := range files {
		if strings.HasPrefix(name, wordWrapDir) && name != wordWrapHome {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return []Violation{{From: "pkg/ui", Reason: "no sources scanned - the word-wrap scan found nothing to read"}}
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(p token.Pos) string { return name + ":" + strconv.Itoa(fset.Position(p).Line) }
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				low := strings.ToLower(n.Name.Name)
				if strings.Contains(low, "wrap") && !strings.Contains(low, "unwrap") && n.Name.Name != "hostedWrap" {
					vs = append(vs, Violation{From: at(n.Pos()), Import: n.Name.Name,
						Reason: "a function that wraps text; build the lines with a lineRule in linebreak.go"})
				}
			case *ast.CallExpr:
				sel, ok := n.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != "strings" {
					return true
				}
				if wordWrapSplitters[sel.Sel.Name] {
					vs = append(vs, Violation{From: at(n.Pos()), Import: "strings." + sel.Sel.Name,
						Reason: "a text split into words outside the line breaker; use lineWords, gapWords or collapseGaps"})
				}
				if wordWrapSpaceCalls[sel.Sel.Name] {
					for _, a := range n.Args {
						if wordWrapSpaceArg(a) {
							vs = append(vs, Violation{From: at(n.Pos()), Import: "strings." + sel.Sel.Name,
								Reason: "a search or split at a space outside the line breaker"})
							break
						}
					}
				}
			}
			return true
		})
	}
	return vs
}
