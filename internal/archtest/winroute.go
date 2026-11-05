package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The session mission identity and the one win finisher of pkg/game. Every
// production read of the guest record's mission, of its return and of a mod
// mission by number goes through the identity file; one finishWon exists, the
// mission seam's, and the campaign service declares no per-game win step.
const (
	winRouteIdentityFile = "pkg/game/sessionmission.go"
	winRouteFinisherFile = "pkg/game/winfinish.go"
)

// winRouteIdentityCalls are the method calls that decide a mission's
// identity, each with the arity the identity's own calls have.
var winRouteIdentityCalls = map[string]int{
	"guestMission":  0,
	"playing":       1,
	"returnMission": 1,
	"guestEntry":    1,
	"ByNumber":      1,
	"Replacing":     1,
}

// winRouteServiceSteps are the win steps the campaign service no longer
// declares: what follows a win is the finisher's rows.
var winRouteServiceSteps = map[string]bool{"finishWon": true, "route": true, "returnToTown": true}

// CheckWinRoute reports each identity call outside the identity file, each
// finishWon but the seam's, a finisher outside its file, and a win step in
// the campaign service. files maps module-relative paths to source; only
// production files of pkg/game are read.
func CheckWinRoute(files map[string]string) []Violation {
	names := make([]string, 0, len(files))
	for name := range files {
		if strings.HasPrefix(name, "pkg/game/") && !strings.Contains(strings.TrimPrefix(name, "pkg/game/"), "/") &&
			strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return []Violation{{From: "pkg/game", Reason: "no sources scanned - the win route scan found nothing to read"}}
	}
	sort.Strings(names)
	var vs []Violation
	var finishWon, finishWin []string
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(p token.Pos) string { return name + ":" + strconv.Itoa(fset.Position(p).Line) }
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				sel, ok := n.Fun.(*ast.SelectorExpr)
				if !ok || name == winRouteIdentityFile {
					return true
				}
				if arity, ok := winRouteIdentityCalls[sel.Sel.Name]; ok && len(n.Args) == arity {
					vs = append(vs, Violation{From: at(n.Pos()),
						Reason: sel.Sel.Name + " call outside the session mission identity; read newMissionIdentity's answer"})
				}
			case *ast.FuncDecl:
				switch n.Name.Name {
				case "finishWon":
					if receiverName(n.Recv) != "frontTransitions" {
						vs = append(vs, Violation{From: at(n.Pos()),
							Reason: "finishWon on " + receiverName(n.Recv) + "; the one finishWon is the mission seam's (frontTransitions)"})
					}
					finishWon = append(finishWon, at(n.Pos()))
				case "finishWin":
					if name != winRouteFinisherFile {
						vs = append(vs, Violation{From: at(n.Pos()), Reason: "finishWin outside " + winRouteFinisherFile})
					}
					finishWin = append(finishWin, at(n.Pos()))
				}
			case *ast.TypeSpec:
				it, ok := n.Type.(*ast.InterfaceType)
				if !ok || n.Name.Name != "campaignService" {
					return true
				}
				for _, m := range it.Methods.List {
					for _, id := range m.Names {
						if winRouteServiceSteps[id.Name] {
							vs = append(vs, Violation{From: at(id.Pos()),
								Reason: "campaignService declares the win step " + id.Name + "; what follows a win is the finisher's rows"})
						}
					}
				}
			}
			return true
		})
	}
	if len(finishWon) != 1 {
		vs = append(vs, Violation{From: "pkg/game", Reason: "finishWon declared " + strconv.Itoa(len(finishWon)) + " times, want 1: " + strings.Join(finishWon, ", ")})
	}
	if len(finishWin) != 1 {
		vs = append(vs, Violation{From: "pkg/game", Reason: "finishWin declared " + strconv.Itoa(len(finishWin)) + " times, want 1: " + strings.Join(finishWin, ", ")})
	}
	return vs
}
