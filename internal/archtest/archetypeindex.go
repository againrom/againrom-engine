package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The archetype-index scan: a hero's class and sex turn into a place 0..3 in
// one function, herokind.Of, and a table in another order reads it through
// herokind.Kind.In. It refuses axis-conditioned additions, ORs and shifts,
// and calls that derive a hero's sex or class from a cell ordinal.

var archetypeRoots = []string{"pkg/data/", "pkg/game/", "pkg/sim/", "pkg/mapload/", "pkg/ui/"}

var archetypeAxisName = regexp.MustCompile(`^(?i:is|hero)?(?i:mage|female)$|^NPCToken(Mage|Female)$`)

// CheckArchetypeIndex reports hero index and cell conversions outside
// pkg/herokind. files is keyed by module-relative slash path.
func CheckArchetypeIndex(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the archetype-index scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !archetypeRoot(name) || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if field, ok := n.(*ast.KeyValueExpr); ok {
				if key, ok := field.Key.(*ast.Ident); ok && archetypeAxisName.MatchString(key.Name) && ordinalHeroAxis(field.Value) {
					vs = append(vs, Violation{
						From:   name + ":" + strconv.Itoa(fset.Position(field.Pos()).Line),
						Reason: "a hero axis is built from a cell ordinal here; read herokind.AudienceOrder",
					})
				}
			}
			if call, ok := n.(*ast.CallExpr); ok {
				for _, arg := range call.Args {
					if ordinalHeroAxis(arg) || ordinalHeroCell(call, arg) {
						vs = append(vs, Violation{
							From:   name + ":" + strconv.Itoa(fset.Position(call.Pos()).Line),
							Reason: "a cell ordinal is turned into sex or class here; read herokind.AudienceOrder and Kind.Female or Kind.Mage",
						})
						break
					}
				}
			}
			stmt, ok := n.(*ast.IfStmt)
			if !ok || !readsHeroAxis(stmt.Cond) {
				return true
			}
			if add := addsToIndex(stmt.Body); add != nil {
				vs = append(vs, Violation{
					From:   name + ":" + strconv.Itoa(fset.Position(add.Pos()).Line),
					Reason: "a hero's class and sex are turned into an index here; call herokind.Of, and Kind.In for another order",
				})
			}
			return true
		})
	}
	return vs
}

func archetypeRoot(name string) bool {
	for _, r := range archetypeRoots {
		if strings.HasPrefix(name, r) {
			return true
		}
	}
	return false
}

func readsHeroAxis(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			found = found || archetypeAxisName.MatchString(n.Name)
		case *ast.SelectorExpr:
			found = found || archetypeAxisName.MatchString(n.Sel.Name)
		}
		return !found
	})
	return found
}

func ordinalHeroCell(call *ast.CallExpr, arg ast.Expr) bool {
	var name string
	switch f := call.Fun.(type) {
	case *ast.Ident:
		name = f.Name
	case *ast.SelectorExpr:
		name = f.Sel.Name
	}
	if name != "HeroCells" {
		return false
	}
	found := false
	ast.Inspect(arg, func(n ast.Node) bool {
		if index, ok := n.(*ast.IndexExpr); ok && archetypeLiteral(index.Index, "0", "1", "2", "3") {
			found = true
		}
		return !found
	})
	return found
}

func ordinalHeroAxis(arg ast.Expr) bool {
	arg = ast.Unparen(arg)
	if unary, ok := arg.(*ast.UnaryExpr); ok && unary.Op == token.NOT {
		return ordinalHeroAxis(unary.X)
	}
	binary, ok := arg.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	switch binary.Op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ, token.LAND, token.LOR:
	default:
		return false
	}
	found := false
	ast.Inspect(binary, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr, *ast.FuncLit:
			return false
		case *ast.BinaryExpr:
			found = found || (n.Op == token.LSS || n.Op == token.REM) && archetypeLiteral(n.Y, "2")
		}
		return !found
	})
	return found
}

func archetypeLiteral(e ast.Expr, values ...string) bool {
	lit, ok := ast.Unparen(e).(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return false
	}
	for _, value := range values {
		if lit.Value == value {
			return true
		}
	}
	return false
}

// addsToIndex answers the first statement of body that adds 1 or 2 to a
// number, ORs them into one or shifts by them, outside loops and closures.
func addsToIndex(body *ast.BlockStmt) ast.Node {
	var hit ast.Node
	ast.Inspect(body, func(n ast.Node) bool {
		if hit != nil {
			return false
		}
		switch n := n.(type) {
		case *ast.ForStmt, *ast.RangeStmt, *ast.FuncLit:
			return false // a loop counter or a closure is no index
		case *ast.IncDecStmt:
			if n.Tok == token.INC {
				hit = n
			}
		case *ast.AssignStmt:
			if len(n.Rhs) == 1 {
				if (n.Tok == token.ADD_ASSIGN || n.Tok == token.OR_ASSIGN || n.Tok == token.SHL_ASSIGN) && archetypeLiteral(n.Rhs[0], "1", "2") {
					hit = n
				}
				if shift, ok := ast.Unparen(n.Rhs[0]).(*ast.BinaryExpr); ok && shift.Op == token.SHL && archetypeLiteral(shift.Y, "1", "2") {
					hit = n
				}
			}
		}
		return hit == nil
	})
	return hit
}
