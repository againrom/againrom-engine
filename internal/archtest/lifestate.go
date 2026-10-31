package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// Outside lifeStateRuleFile, production pkg/sim and pkg/game compare no .HP
// with a literal 0; they ask a sim.LifeState by name.
const lifeStateRuleFile = "pkg/sim/world.go"

// CheckLifeState reports each ordered .HP comparison with a literal 0.
func CheckLifeState(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the life-state scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	isHP := func(x ast.Expr) bool {
		sel, ok := x.(*ast.SelectorExpr)
		return ok && sel.Sel.Name == "HP"
	}
	isZero := func(x ast.Expr) bool {
		lit, ok := x.(*ast.BasicLit)
		return ok && lit.Kind == token.INT && lit.Value == "0"
	}
	var vs []Violation
	for _, name := range names {
		if !(strings.HasPrefix(name, "pkg/sim/") || strings.HasPrefix(name, "pkg/game/")) ||
			strings.HasSuffix(name, "_test.go") || name == lifeStateRuleFile {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			b, ok := n.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			switch b.Op {
			case token.LSS, token.LEQ, token.GTR, token.GEQ:
			default:
				return true
			}
			if isHP(b.X) && isZero(b.Y) || isZero(b.X) && isHP(b.Y) {
				vs = append(vs, Violation{
					From:   name + ":" + strconv.Itoa(fset.Position(b.Pos()).Line),
					Reason: "a raw .HP comparison with 0 outside " + lifeStateRuleFile + "; ask a sim.LifeState by name",
				})
			}
			return true
		})
	}
	return vs
}
