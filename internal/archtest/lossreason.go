package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The loss reason scan: latchOutcome is the one writer of a mission's
// outcome and of its failure reason, which it takes from missionLossReason,
// the one producer; partyFailureReason alone reads the edition's failure
// reasons, and the loss notice's outcomeText reads no script counter.
const (
	lossLatch    = "pkg/game/world.go:latchOutcome"
	lossProducer = "missionLossReason"
	lossParty    = "pkg/game/world.go:partyFailureReason"
)

var lossLatchFields = map[string]bool{"outcome": true, "lossReason": true}

var lossEditionFields = map[string]bool{"FailureReason": true, "HeroFailureReason": true}

// CheckLossReason reports every production write of a mission outcome or
// failure reason outside latchOutcome, a failure reason latchOutcome takes
// from anything but missionLossReason, an edition failure reason read
// outside partyFailureReason, and an outcomeText that reads the script
// counters. files is keyed by module-relative slash path, as
// LoadDrawnTextSources returns it.
func CheckLossReason(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the loss reason scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/") || strings.HasPrefix(name, "pkg/base/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		game := strings.HasPrefix(name, "pkg/game/")
		for _, decl := range f.Decls {
			owner := "package scope"
			fn, _ := decl.(*ast.FuncDecl)
			if fn != nil {
				owner = fn.Name.Name
			}
			at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
			ast.Inspect(decl, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.AssignStmt:
					if game {
						vs = append(vs, lossAssign(name+":"+owner, at(x), x)...)
					}
				case *ast.SelectorExpr:
					if lossEditionFields[x.Sel.Name] && name+":"+owner != lossParty {
						vs = append(vs, Violation{From: at(x), Reason: owner + " reads the edition's " + x.Sel.Name + "; partyFailureReason alone turns a fall into a failure reason"})
					}
				case *ast.CallExpr:
					if game && owner == "outcomeText" && lossCallName(x) == "ScriptCounters" {
						vs = append(vs, Violation{From: at(x), Reason: "outcomeText reads the script counters; the loss notice reads its reason from missionLossReason through lossReason"})
					}
				}
				return true
			})
		}
	}
	return vs
}

// lossAssign checks one assignment in pkg/game.
func lossAssign(fn, at string, a *ast.AssignStmt) []Violation {
	var vs []Violation
	for i, lhs := range a.Lhs {
		sel, ok := lhs.(*ast.SelectorExpr)
		if !ok || !lossLatchFields[sel.Sel.Name] {
			continue
		}
		if fn != lossLatch {
			vs = append(vs, Violation{From: at, Reason: fn[strings.LastIndex(fn, ":")+1:] + " writes the mission's " + sel.Sel.Name + "; latchOutcome is its one writer"})
			continue
		}
		if sel.Sel.Name != "lossReason" {
			continue
		}
		if len(a.Rhs) != len(a.Lhs) {
			vs = append(vs, Violation{From: at, Reason: "latchOutcome writes lossReason from a multi-value expression; it takes missionLossReason's result"})
			continue
		}
		if call, ok := a.Rhs[i].(*ast.CallExpr); !ok || lossCallName(call) != lossProducer {
			vs = append(vs, Violation{From: at, Reason: "latchOutcome writes lossReason from something other than missionLossReason, the one loss reason producer"})
		}
	}
	return vs
}

// lossCallName is the called function's or method's name.
func lossCallName(c *ast.CallExpr) string {
	switch f := c.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}
