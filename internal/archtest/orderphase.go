package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The order-phase scan: outside pkg/sim/orderstate.go, production pkg/sim code
// asks what an actor's order is doing through orderState and not through the
// carriers. A site that reads or changes a carrier's fields is listed here with
// the number of such sites in its file; a new site in a file not listed, or a
// further site in a listed file, fails the scan.
const orderPhaseRuleFile = "pkg/sim/orderstate.go"

// orderCarrierAllowance counts, per file, the calls of a record lookup that
// read or change the record's own fields: a clock sum, a delivery payment, a
// release rebinding, a record insert, a decoder walk.
var orderCarrierAllowance = map[string]int{
	"pkg/sim/actionguard.go:bookCastIndex":         1,
	"pkg/sim/activity.go:bookCastIndex":            1,
	"pkg/sim/activity.go:scrollIndex":              1,
	"pkg/sim/creaturecast.go:bookCastIndex":        1,
	"pkg/sim/spell.go:bookCastIndex":               3,
	"pkg/sim/spelldelivery.go:bookCastIndex":       2,
	"pkg/sim/spelldeliverybinary.go:bookCastIndex": 1,
	"pkg/sim/usescroll.go:scrollIndex":             2,
	"pkg/sim/world.go:scrollIndex":                 1,
}

// orderWaitAllowance counts, per file, the comparisons of CastWait that treat
// it as a timer: a decoder check, a floor on a copied remaining count and the
// guard before a decrement.
var orderWaitAllowance = map[string]int{
	"pkg/sim/binary.go": 1,
	"pkg/sim/spell.go":  2,
}

// orderPairAllowance counts, per file, the lines naming both queued cast
// kinds: the one method that tests the pair and a dispatch over the kinds.
var orderPairAllowance = map[string]int{
	"pkg/sim/pendingorder.go":   1,
	"pkg/sim/currentactions.go": 1,
}

// CheckOrderPhaseReaders reports every in-flight predicate of a cast written
// out beside its carriers instead of read from orderState.
func CheckOrderPhaseReaders(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the order-phase scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/sim/") || name == orderPhaseRuleFile {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		calls := map[string]int{}
		waits := 0
		pairLines := map[int]bool{}
		kindLines := map[int]int{}
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
				switch fun {
				case "bookCastIndex", "scrollIndex":
					calls[fun]++
					if calls[fun] > orderCarrierAllowance[name+":"+fun] {
						vs = append(vs, Violation{From: at(x), Reason: fun + " outside orderstate.go in a file or count not listed as a record reader; ask orderState"})
					}
				case "scrollInFlight":
					vs = append(vs, Violation{From: at(x), Reason: "scrollInFlight was replaced by orderState.scrollBegun"})
				}
			case *ast.BinaryExpr:
				switch x.Op {
				case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ:
					for _, side := range []ast.Expr{x.X, x.Y} {
						if sel, ok := side.(*ast.SelectorExpr); ok && sel.Sel.Name == "CastWait" {
							waits++
							if waits > orderWaitAllowance[name] {
								vs = append(vs, Violation{From: at(x), Reason: "a CastWait comparison outside orderstate.go; ask orderState.recoveryWaiting"})
							}
						}
					}
				}
			case *ast.Ident:
				if x.Name == "PendingActorCast" || x.Name == "PendingCellCast" {
					line := fset.Position(x.Pos()).Line
					bit := 1
					if x.Name == "PendingCellCast" {
						bit = 2
					}
					kindLines[line] |= bit
					if kindLines[line] == 3 && !pairLines[line] {
						pairLines[line] = true
						if len(pairLines) > orderPairAllowance[name] {
							vs = append(vs, Violation{From: name + ":" + strconv.Itoa(line), Reason: "the queued cast pair written out; use PendingOrder.IsCastRow"})
						}
					}
				}
			}
			return true
		})
	}
	return vs
}
