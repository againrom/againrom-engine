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

// Hero index and cell conversions belong to pkg/herokind.

var archetypeRoots = []string{"pkg/data/", "pkg/game/", "pkg/sim/", "pkg/mapload/", "pkg/ui/"}

var archetypeAxisName = regexp.MustCompile(`^(?i:is|hero)?(?i:mage|female)$|^NPCToken(Mage|Female)$`)
var archetypeBitName = regexp.MustCompile(`^Hero(Woman|Man)(Mage|Fighter)$`)
var archetypeTagName = regexp.MustCompile(`(?i)^(at|hero|axes|herotag|heroindex)$`)

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
			if bits, ok := n.(*ast.BinaryExpr); ok && archetypeBitCopy(bits) {
				vs = append(vs, Violation{
					From:   name + ":" + strconv.Itoa(fset.Position(bits.Pos()).Line),
					Reason: "hero bits are decoded here; read herokind.Kind or HeroAudience.Kinds",
				})
			}
			if decl, ok := n.(*ast.ValueSpec); ok && archetypeBitOrder(decl) {
				vs = append(vs, Violation{
					From:   name + ":" + strconv.Itoa(fset.Position(decl.Pos()).Line),
					Reason: "a hero bit order is declared here; read herokind.AudienceOrder",
				})
			}
			if cells, ok := n.(*ast.CompositeLit); ok && archetypeMaskCells(cells) {
				vs = append(vs, Violation{
					From:   name + ":" + strconv.Itoa(fset.Position(cells.Pos()).Line),
					Reason: "hero cells are grouped by masks here; read herokind.AudienceOrder and Kind axes",
				})
			}
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

func archetypeBitCopy(e *ast.BinaryExpr) bool {
	if e.Op != token.AND {
		return false
	}
	if archetypeHasAudienceBit(e) {
		return true
	}
	for _, pair := range [][2]ast.Expr{{e.X, e.Y}, {e.Y, e.X}} {
		if !archetypeLiteral(pair[1], "1", "2") {
			continue
		}
		switch tag := ast.Unparen(pair[0]).(type) {
		case *ast.Ident:
			if archetypeTagName.MatchString(tag.Name) {
				return true
			}
		case *ast.BinaryExpr:
			if tag.Op == token.SUB && archetypeLiteral(tag.Y, "0x21", "33") {
				return true
			}
		}
	}
	return false
}

func archetypeHasAudienceBit(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && archetypeBitName.MatchString(id.Name) {
			found = true
		}
		return !found
	})
	return found
}

func archetypeBitOrder(decl *ast.ValueSpec) bool {
	hero := false
	if typ, ok := decl.Type.(*ast.Ident); ok && typ.Name == "HeroAudience" {
		hero = true
	}
	for _, name := range decl.Names {
		hero = hero || archetypeBitName.MatchString(name.Name)
	}
	if !hero {
		return false
	}
	found := false
	for _, value := range decl.Values {
		ast.Inspect(value, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				found = found || n.Name == "iota"
			case *ast.BinaryExpr:
				found = found || n.Op == token.SHL && archetypeLiteral(n.Y, "0", "1", "2", "3")
			}
			return !found
		})
	}
	return found
}

func archetypeMaskCells(cells *ast.CompositeLit) bool {
	array, ok := cells.Type.(*ast.ArrayType)
	if !ok {
		return false
	}
	name := ""
	switch typ := array.Elt.(type) {
	case *ast.Ident:
		name = typ.Name
	case *ast.SelectorExpr:
		name = typ.Sel.Name
	}
	if name != "HeroAudience" {
		return false
	}
	for _, cell := range cells.Elts {
		if group, ok := ast.Unparen(cell).(*ast.BinaryExpr); ok && group.Op == token.OR && archetypeHasAudienceBit(group) {
			return true
		}
	}
	return false
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

// addsToIndex finds additions, ORs and shifts outside loops and closures.
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
