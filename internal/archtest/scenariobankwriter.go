package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// scenarioBankWriter is the one non-test file that stores into a scenario bank
// slot. The second game's bank is sim.ScenarioBank; every other store goes
// through sim.SetScenario, sim.WriteObjective, sim.EnterLocation or
// sim.NewGameScenarioBank.
const scenarioBankWriter = "pkg/sim/scenariobank.go"

// CheckScenarioBankWriter reports every store into a scenario bank slot in a
// non-test pkg file other than scenarioBankWriter. It is a source check by
// name: a slot is an index expression over an identifier or field named bank,
// and a store is an assignment, a compound assignment, an increment or taking
// the slot's address. files is keyed by module-relative slash path, as
// LoadDrawnTextSources returns it.
func CheckScenarioBankWriter(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the scenario bank scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		if name != scenarioBankWriter && strings.HasPrefix(name, "pkg/") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
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
		report := func(n ast.Node) {
			vs = append(vs, Violation{From: name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line),
				Reason: "stores into a scenario bank slot; go through the writer in " + scenarioBankWriter})
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				if n.Tok == token.DEFINE {
					return true
				}
				for _, l := range n.Lhs {
					if isBankSlot(l) {
						report(l)
					}
				}
			case *ast.IncDecStmt:
				if isBankSlot(n.X) {
					report(n.X)
				}
			case *ast.UnaryExpr:
				if n.Op == token.AND && isBankSlot(n.X) {
					report(n)
				}
			}
			return true
		})
	}
	return vs
}

// isBankSlot reports whether e is an index over a name or field called bank.
func isBankSlot(e ast.Expr) bool {
	ix, ok := ast.Unparen(e).(*ast.IndexExpr)
	if !ok {
		return false
	}
	switch x := ast.Unparen(ix.X).(type) {
	case *ast.Ident:
		return x.Name == "bank"
	case *ast.SelectorExpr:
		return x.Sel.Name == "bank"
	}
	return false
}
