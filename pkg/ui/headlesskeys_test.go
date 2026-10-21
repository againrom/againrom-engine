package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The refusal of an unknown key lists the keys HeadlessKey accepts: the list
// is the string cases of its switch, no more and no fewer, and an unknown
// name is refused with it.
func TestHeadlessKeyRefusalListsTheAcceptedKeys(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "headless.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var cases []string
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "HeadlessKey" {
			return true
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			sw, ok := n.(*ast.SwitchStmt)
			if !ok {
				return true
			}
			for _, c := range sw.Body.List {
				for _, e := range c.(*ast.CaseClause).List {
					if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						v, _ := strconv.Unquote(lit.Value)
						cases = append(cases, v)
					}
				}
			}
			return false
		})
		return false
	})
	// The switch also takes f5 to f8 by prefix test below its cases; the
	// names in a case list are all literal.
	got := slices.Clone(headlessKeyNames)
	slices.Sort(got)
	slices.Sort(cases)
	if !slices.Equal(got, cases) {
		t.Fatalf("headlessKeyNames and the cases of HeadlessKey differ:\nlist  %q\ncases %q", got, cases)
	}
	a := newTestApp(t, appRows(1), nil)
	err = a.HeadlessKey("zzz")
	if err == nil || !strings.Contains(err.Error(), `unknown key "zzz"; accepted keys: enter, `) || !strings.Contains(err.Error(), "alt- followed by a letter") {
		t.Fatalf("unknown key: %v", err)
	}
	if err := a.HeadlessKey("alt-q"); err != nil && strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("alt-q is refused as unknown: %v", err)
	}
}
