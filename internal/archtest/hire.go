package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The hire scan: pkg/game/hire.go holds the hire rows of both games, the one
// producer hireMembers and the one transaction hireDeal.run. Outside that
// file no production source names a second-game hire slot (bank 511..571 or
// the slot constants), sets a hire field of a campaign actor spec, writes a
// first-game hire flag except where hireWriters names it, or adds a method
// to hireDeal; and no source anywhere declares a retired second producer.
const hireFile = "pkg/game/hire.go"

var hireSlotNames = map[string]bool{"secondHireSlots": true, "secondHireSelection": true, "secondHireAvailability": true, "secondHireTown": true}

var retiredHireBuilders = map[string]bool{"mercenaryHire": true, "mercenaryReturn": true, "hireOrFire": true, "buildMercenarySquad": true,
	"buildSiegeSquad": true, "missionParty": true, "innHires": true, "keyedCampaignActor": true}

// hireWriters names the functions that may write or hand out a first-game
// hire flag outside hire.go: the travel restore copies a validated record,
// and squadDeal hands the flag to the transaction.
var hireWriters = map[string]bool{"pkg/game/towntravel.go:restoreTowns": true, "pkg/game/tavern.go:squadDeal": true}

// CheckHire reports every hire shape outside the hire rows. files is keyed
// by module-relative slash path, as LoadDrawnTextSources returns it.
func CheckHire(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the hire scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/") || strings.HasSuffix(name, "_test.go") {
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
			owner := "package scope"
			if fn, ok := decl.(*ast.FuncDecl); ok {
				owner = fn.Name.Name
				if retiredHireBuilders[owner] {
					vs = append(vs, Violation{From: at(fn), Reason: owner + " is a retired second hire producer; hires are built by hireMembers from the hire rows"})
				}
				if recv := receiverName(fn.Recv); recv == "hireDeal" && (name != hireFile || owner != "run") {
					vs = append(vs, Violation{From: at(fn), Reason: "hireDeal." + owner + " is a second hire transaction; hireDeal.run in " + hireFile + " is the only one"})
				}
			}
			if name == hireFile || !strings.HasPrefix(name, "pkg/game/") {
				continue
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.Ident:
					if hireSlotNames[x.Name] {
						vs = append(vs, Violation{From: at(x), Reason: owner + " names the hire slot " + x.Name + "; hire slots are read through the hire rows"})
					}
				case *ast.IndexExpr:
					if isBank(x.X) && holdsHireSlot(x.Index) {
						vs = append(vs, Violation{From: at(x), Reason: owner + " indexes a hire slot of the bank; hire slots are read through the hire rows"})
					}
				case *ast.CallExpr:
					if lastName(x.Fun) == "SetScenario" && len(x.Args) > 1 && holdsHireSlot(x.Args[1]) {
						vs = append(vs, Violation{From: at(x), Reason: owner + " sets a hire slot of the bank; hire slots are written through the hire rows"})
					}
				case *ast.AssignStmt:
					for _, lhs := range x.Lhs {
						if isHireFlag(lhs) && !hireWriters[name+":"+owner] {
							vs = append(vs, Violation{From: at(x), Reason: owner + " writes a first-game hire flag; hireDeal.run is the hire transaction"})
						}
						if sel, ok := ast.Unparen(lhs).(*ast.SelectorExpr); ok && sel.Sel.Name == "Hire" {
							vs = append(vs, Violation{From: at(x), Reason: owner + " sets Hire of a campaign actor spec; hireMembers builds every hire"})
						}
					}
				case *ast.UnaryExpr:
					if x.Op == token.AND && isHireFlag(x.X) && !hireWriters[name+":"+owner] {
						vs = append(vs, Violation{From: at(x), Reason: owner + " takes a first-game hire flag's address; hireDeal.run is the hire transaction"})
					}
				case *ast.CompositeLit:
					if lastName(x.Type) != "CampaignActorSpec" {
						return true
					}
					for _, e := range x.Elts {
						if kv, ok := e.(*ast.KeyValueExpr); ok {
							if key, ok := kv.Key.(*ast.Ident); ok && (key.Name == "Hire" || key.Name == "MercenaryType") {
								vs = append(vs, Violation{From: at(x), Reason: owner + " sets " + key.Name + " of a campaign actor spec; hireMembers builds every hire"})
							}
						}
					}
				}
				return true
			})
		}
	}
	return vs
}

// isBank reports whether e names a scenario bank.
func isBank(e ast.Expr) bool {
	if s, ok := ast.Unparen(e).(*ast.StarExpr); ok {
		e = s.X
	}
	return lastName(e) == "bank"
}

// isHireFlag reports whether e is an element of a first-game hire flag list.
func isHireFlag(e ast.Expr) bool {
	ix, ok := e.(*ast.IndexExpr)
	return ok && lastName(ix.X) == "mercHired"
}

// holdsHireSlot reports whether an index expression holds an integer
// literal in the hire slot range 511..571.
func holdsHireSlot(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.INT {
			if v, err := strconv.Atoi(lit.Value); err == nil && v >= 511 && v <= 571 {
				found = true
			}
		}
		return !found
	})
	return found
}
