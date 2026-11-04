package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// hostModeFields are the town.HostSpec fields that carry a mode. Each has a
// named type in pkg/town and is compared against that type's constants only.
var hostModeFields = map[string]bool{
	"Shop": true, "TrackSource": true, "Rows": true, "Header": true, "Footer": true,
	"Talk": true, "Pages": true, "Art": true, "AudioRoom": true, "Leave": true,
}

// talkCursorHome is the one struct that holds the town talk cursor.
const talkCursorHome = "Town"

// CheckHostModes reports, over production sources as LoadDrawnTextSources
// returns them, each comparison or switch of a host mode field against a
// string literal in pkg/game and pkg/town, each second struct field of type
// talkCursor, and each method named talkCursor. A host value is a hostRules()
// call, a selector ending in .Host, or a name assigned from either in the
// same function.
func CheckHostModes(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the host mode scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	homes := 0
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/game/") && !strings.HasPrefix(name, "pkg/town/") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv != nil && d.Name.Name == "talkCursor" {
					vs = append(vs, Violation{From: at(d), Reason: "method talkCursor is a second talk cursor home; the cursor lives on " + talkCursorHome})
				}
				if d.Body != nil {
					vs = append(vs, hostModeLiterals(d.Body, at)...)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, fld := range st.Fields.List {
						if !isTalkCursorType(fld.Type) {
							continue
						}
						if ts.Name.Name == talkCursorHome && strings.HasPrefix(name, "pkg/game/") {
							homes++
							continue
						}
						vs = append(vs, Violation{From: at(fld), Reason: ts.Name.Name + " holds a talkCursor field, a second talk cursor home; the cursor lives on " + talkCursorHome})
					}
				}
			}
		}
	}
	if homes != 1 {
		vs = append(vs, Violation{From: "pkg/game", Reason: "the talk cursor home " + talkCursorHome + " holds " + strconv.Itoa(homes) + " talkCursor fields, want 1"})
	}
	return vs
}

func isTalkCursorType(e ast.Expr) bool {
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "talkCursor"
}

// hostValue reports whether e reads a HostSpec: hostRules(), x.Host, *x.Host
// or a name in held.
func hostValue(e ast.Expr, held map[string]bool) bool {
	switch x := e.(type) {
	case *ast.ParenExpr:
		return hostValue(x.X, held)
	case *ast.StarExpr:
		return hostValue(x.X, held)
	case *ast.CallExpr:
		sel, ok := x.Fun.(*ast.SelectorExpr)
		return ok && sel.Sel.Name == "hostRules"
	case *ast.SelectorExpr:
		return x.Sel.Name == "Host"
	case *ast.Ident:
		return held[x.Name]
	}
	return false
}

// hostModeField returns the mode field e selects off a host value.
func hostModeField(e ast.Expr, held map[string]bool) (string, bool) {
	if p, ok := e.(*ast.ParenExpr); ok {
		return hostModeField(p.X, held)
	}
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || !hostModeFields[sel.Sel.Name] || !hostValue(sel.X, held) {
		return "", false
	}
	return sel.Sel.Name, true
}

func isStringLiteral(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING
}

func hostModeLiterals(body *ast.BlockStmt, at func(ast.Node) string) []Violation {
	held := map[string]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		if as, ok := n.(*ast.AssignStmt); ok && len(as.Lhs) == len(as.Rhs) {
			for i, l := range as.Lhs {
				if id, ok := l.(*ast.Ident); ok && hostValue(as.Rhs[i], held) {
					held[id.Name] = true
				}
			}
		}
		return true
	})
	var vs []Violation
	report := func(n ast.Node, field string) {
		vs = append(vs, Violation{From: at(n), Reason: "host mode " + field + " is compared with a string literal; compare it with a pkg/town constant"})
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BinaryExpr:
			if x.Op != token.EQL && x.Op != token.NEQ {
				return true
			}
			if field, ok := hostModeField(x.X, held); ok && isStringLiteral(x.Y) {
				report(x, field)
			} else if field, ok := hostModeField(x.Y, held); ok && isStringLiteral(x.X) {
				report(x, field)
			}
		case *ast.SwitchStmt:
			if x.Tag == nil {
				return true
			}
			field, ok := hostModeField(x.Tag, held)
			if !ok {
				return true
			}
			for _, c := range x.Body.List {
				for _, e := range c.(*ast.CaseClause).List {
					if isStringLiteral(e) {
						report(e, field)
					}
				}
			}
		}
		return true
	})
	return vs
}
