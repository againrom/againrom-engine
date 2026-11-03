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
// herokind.Kind.In. In pkg/data, pkg/game, pkg/sim and pkg/mapload it refuses
// an if whose condition reads the class or the sex (a name archetypeAxisName
// matches) when its body adds 1 or 2 to a number or ORs 1 or 2 into one: a
// second index function.

var archetypeRoots = []string{"pkg/data/", "pkg/game/", "pkg/sim/", "pkg/mapload/"}

var archetypeAxisName = regexp.MustCompile(`^(?i:is|hero)?(?i:mage|female)$|^NPCToken(Mage|Female)$`)

// CheckArchetypeIndex reports every if that builds a hero archetype index
// outside pkg/herokind. files is keyed by module-relative slash path.
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

// addsToIndex answers the first statement of body that adds 1 or 2 to a
// number or ORs 1 or 2 into one, outside loops and closures.
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
			if (n.Tok == token.ADD_ASSIGN || n.Tok == token.OR_ASSIGN) && len(n.Rhs) == 1 {
				if lit, ok := n.Rhs[0].(*ast.BasicLit); ok && lit.Kind == token.INT && (lit.Value == "1" || lit.Value == "2") {
					hit = n
				}
			}
		}
		return hit == nil
	})
	return hit
}
