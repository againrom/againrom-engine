package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// CastRequestFile is the one cast request boundary: the busy rule every cast
// issuer meets and the busy query every action that waits for a cast asks.
// Outside it, production pkg/sim code calls none of the busy folds, and no
// production pkg/ code compares a cast refusal.
const CastRequestFile = "pkg/sim/castrequest.go"

// The shapes CastRequestSites names.
const (
	ShapeCastFold    = "cast-fold"    // a call of a busy fold or an admission body
	ShapeRefusalTest = "refusal-test" // == or != with a cast refusal, or a case naming one
)

// castFoldNames are the busy folds and the two admission bodies that only
// the boundary calls. orderState.castBusy and retainsOldStrike are defined
// elsewhere and asked only there.
var castFoldNames = map[string]bool{
	"actorCastBusy": true, "actorActionBusy": true, "retainedCastBusy": true, "castBusy": true,
	"strikeHoldsCast": true, "replaceRetainedCast": true, "retainsOldStrike": true,
	"bookSpellAdmissionRange": true, "bookSpellCellAdmissionRange": true,
}

// CastRequestAllowList names the sites the guard admits, keyed "file shape",
// with their count and reason: a part of the busy rule with no neutral form
// in the boundary stands here. It is empty.
var CastRequestAllowList = map[string]struct {
	Count  int
	Reason string
}{}

// CastRequestSites reports every busy fold called and every cast refusal
// compared outside the boundary, one finding per site, less the allow list's
// admitted counts. The refusals are the boundary's own refusal constants, so
// a refusal added there is guarded at once.
func CastRequestSites(files map[string]string) []Violation {
	allow := map[string]int{}
	for key, a := range CastRequestAllowList {
		allow[key] = a.Count
	}
	return castRequestSitesAllowing(files, allow)
}

func castRequestSitesAllowing(files map[string]string, allow map[string]int) []Violation {
	src, ok := files[CastRequestFile]
	if !ok {
		return []Violation{{From: CastRequestFile, Reason: "the cast request boundary is missing; the scan guards nothing"}}
	}
	consts, values, err := refusalConsts(src)
	if err != nil {
		return []Violation{{From: CastRequestFile, Reason: "source does not parse: " + err.Error()}}
	}
	if len(consts) == 0 {
		return []Violation{{From: CastRequestFile, Reason: "the boundary declares no refusal constant; the refusal scan guards nothing"}}
	}
	refusal := func(e ast.Expr) bool {
		switch x := ast.Unparen(e).(type) {
		case *ast.Ident:
			return consts[x.Name]
		case *ast.BasicLit:
			s, err := strconv.Unquote(x.Value)
			return x.Kind == token.STRING && err == nil && values[s]
		}
		return false
	}
	var vs []Violation
	for _, name := range sortedNames(files) {
		if !strings.HasPrefix(name, "pkg/") || name == CastRequestFile {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		sim := strings.HasPrefix(name, "pkg/sim/")
		report := func(n ast.Node, shape, reason string) {
			key := name + " " + shape
			if allow[key] > 0 {
				allow[key]--
				return
			}
			at := name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line)
			vs = append(vs, Violation{From: at + " " + shape, Reason: reason})
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				fun := ""
				switch c := x.Fun.(type) {
				case *ast.Ident:
					fun = c.Name
				case *ast.SelectorExpr:
					fun = c.Sel.Name
				}
				if sim && castFoldNames[fun] {
					report(x, ShapeCastFold, fun+" called outside "+CastRequestFile+"; ask the cast request boundary")
				}
			case *ast.BinaryExpr:
				if (x.Op == token.EQL || x.Op == token.NEQ) && (refusal(x.X) || refusal(x.Y)) {
					report(x, ShapeRefusalTest, "a cast refusal compared outside "+CastRequestFile+"; ask its refusal predicate")
				}
			case *ast.CaseClause:
				for _, e := range x.List {
					if refusal(e) {
						report(e, ShapeRefusalTest, "a case names a cast refusal outside "+CastRequestFile+"; ask its refusal predicate")
					}
				}
			}
			return true
		})
	}
	return vs
}

// refusalConsts reads the boundary's constants named refusal..., and their
// string values.
func refusalConsts(src string) (map[string]bool, map[string]bool, error) {
	f, err := parser.ParseFile(token.NewFileSet(), CastRequestFile, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, nil, err
	}
	consts, values := map[string]bool{}, map[string]bool{}
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, s := range g.Specs {
			v := s.(*ast.ValueSpec)
			for i, id := range v.Names {
				if !strings.HasPrefix(id.Name, "refusal") || i >= len(v.Values) {
					continue
				}
				lit, ok := v.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				if s, err := strconv.Unquote(lit.Value); err == nil {
					consts[id.Name], values[s] = true, true
				}
			}
		}
	}
	return consts, values, nil
}
