package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

const membershipFile = "pkg/mapload/membership.go"
const joinCatalogFile = "pkg/game/joincatalog.go"
const joinProfileFile = "pkg/base/edition.go"

var membershipFields = map[string]bool{
	"CompanionNPC": true, "MercenaryType": true, "CampaignHire": true, "CampaignUnit": true,
}

var membershipPredicates = map[string]bool{
	"Membership": true, "Hired": true, "UnitMember": true, "CampaignHireID": true,
	"TavernType": true, "CompanionID": true, "ValidMembership": true,
	"PartyIdentity": true, "CompanionIdentity": true, "DecodePartyIdentity": true,
	"inferredPartyID": true,
}

var joinReaders = map[string]bool{
	"joins": true, "joinMember": true, "addChapterCompanions": true,
	"joinChapterMember": true, "addPartyIdentity": true, "joinAtTalk": true,
	"addTalkIdentities": true, "addLeaveIdentities": true, "holdsCompanion": true,
}

var retiredJoinReaders = map[string]bool{
	"joinOnTalk": true, "carryTownCompanion": true, "carryCompanion": true,
}

var tavernLiteralGuardEntries = map[string]string{
	"pkg/game/tavern.go:siegeHireName": "type 2 names Ballista within the Units arm; every other input retains the Catapult fallback",
}

// CheckMembership guards membership decisions, the codec and the join reader.
// Wire copies remain data; aliases cannot turn a copied field into a predicate.
func CheckMembership(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned for party membership"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []Violation
	for _, name := range names {
		if !(strings.HasPrefix(name, "pkg/") || strings.HasPrefix(name, "cmd/")) || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			out = append(out, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		for _, decl := range f.Decls {
			owner := "package scope"
			if fn, ok := decl.(*ast.FuncDecl); ok {
				owner = fn.Name.Name
				if name != membershipFile && strings.HasPrefix(name, "pkg/mapload/") && membershipPredicates[owner] {
					out = append(out, Violation{From: name, Reason: owner + " is a second membership predicate or identity codec"})
				}
				if retiredJoinReaders[owner] || joinReaders[owner] && name != joinCatalogFile {
					out = append(out, Violation{From: name, Reason: owner + " is a second join catalog reader"})
				}
			}
			aliases := membershipAliases(decl)
			wordAliases := identityWordAliases(decl)
			context := strings.ToLower(name + " " + owner)
			tavern := strings.Contains(context, "tavern") || strings.Contains(context, "mercenary") || strings.Contains(context, "hire")
			emitted := map[token.Pos]bool{}
			report := func(n ast.Node, reason string) {
				if !emitted[n.Pos()] {
					emitted[n.Pos()] = true
					out = append(out, Violation{From: name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line), Reason: owner + " " + reason})
				}
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.BinaryExpr:
					if tavern && tavernLiteralGuardEntries[name+":"+owner] == "" && tavernTypeLiteral(x) {
						report(x, "tests a tavern type against a profile literal; use TavernType or TavernUnit")
					}
					if name != membershipFile && membershipRead(x, aliases) {
						report(x, "tests a membership field; use the membership predicates")
					}
					if name != membershipFile && x.Op == token.AND && integerIn(x, 65535) && identityWordRead(x, wordAliases) {
						report(x, "decodes a party identity word outside the identity codec")
					}
				case *ast.IfStmt:
					if name != membershipFile && membershipRead(x.Cond, aliases) {
						report(x.Cond, "tests a membership field; use the membership predicates")
					}
				case *ast.ForStmt:
					if name != membershipFile && x.Cond != nil && membershipRead(x.Cond, aliases) {
						report(x.Cond, "tests a membership field; use the membership predicates")
					}
				case *ast.SwitchStmt:
					if name != membershipFile && x.Tag != nil && membershipRead(x.Tag, aliases) {
						report(x.Tag, "switches on a membership field; use the membership predicates")
					}
				case *ast.SelectorExpr:
					if x.Sel.Name == "PartyProfile" && name != joinProfileFile ||
						x.Sel.Name == "JoinRows" && name != joinProfileFile && name != joinCatalogFile ||
						(x.Sel.Name == "TalkNPC" || x.Sel.Name == "TalkTopic" || x.Sel.Name == "TownGrant" || x.Sel.Name == "BankSlot") &&
							name != joinProfileFile && name != joinCatalogFile {
						report(x, "reads join rows outside the join catalog")
					}
				case *ast.CallExpr:
					fun := lastName(x.Fun)
					if name != membershipFile && (fun == "Contains" || fun == "Equal" || fun == "Compare" || fun == "DeepEqual") && membershipRead(x, aliases) {
						report(x, "compares a membership field; use the membership predicates")
					}
					if name != membershipFile && fun == "uint16" && identityWordRead(x, wordAliases) {
						report(x, "decodes a party identity word outside the identity codec")
					}
					if fun == "addPartyIdentity" && name != joinCatalogFile && !(name == "pkg/game/secondcensus.go" && owner == "secondCensusParty") {
						report(x, "adds a companion outside the join catalog")
					}
				case *ast.IndexExpr:
					if name != joinProfileFile && name != joinCatalogFile && isBank(x.X) && integerIn(x.Index, 773) {
						report(x, "reads the leave join slot outside the join catalog")
					}
				case *ast.ValueSpec:
					for _, id := range x.Names {
						if strings.EqualFold(id.Name, "tavernMercenaryTypes") && name != joinProfileFile {
							report(x, "states a tavern type range outside the profile")
						}
					}
				}
				return true
			})
		}
	}
	return out
}

func tavernTypeLiteral(x *ast.BinaryExpr) bool {
	switch x.Op {
	case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ:
	default:
		return false
	}
	isType := func(n ast.Expr) bool {
		if call, ok := ast.Unparen(n).(*ast.CallExpr); ok {
			n = call.Fun
		}
		switch lastName(n) {
		case "typ", "npc", "MercenaryType", "TavernType":
			return true
		}
		return false
	}
	for _, value := range []int{2, 3, 15, 16} {
		if isType(x.X) && integerIn(x.Y, value) || isType(x.Y) && integerIn(x.X, value) {
			return true
		}
	}
	return false
}

func membershipRead(n ast.Node, aliases map[string]bool) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			found = found || membershipFields[x.Sel.Name]
		case *ast.Ident:
			found = found || aliases[x.Name]
		}
		return !found
	})
	return found
}

func membershipAliases(n ast.Node) map[string]bool {
	aliases := map[string]bool{}
	for {
		before := len(aliases)
		ast.Inspect(n, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				if len(x.Lhs) == len(x.Rhs) {
					for i, rhs := range x.Rhs {
						if id, ok := x.Lhs[i].(*ast.Ident); ok && membershipRead(rhs, aliases) {
							aliases[id.Name] = true
						}
					}
				}
			case *ast.ValueSpec:
				if len(x.Names) == len(x.Values) {
					for i, rhs := range x.Values {
						if membershipRead(rhs, aliases) {
							aliases[x.Names[i].Name] = true
						}
					}
				}
			}
			return true
		})
		if len(aliases) == before {
			return aliases
		}
	}
}

func identityWordRead(n ast.Node, aliases map[string]bool) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			found = found || x.Sel.Name == "DisplayBacking"
		case *ast.Ident:
			found = found || aliases[x.Name]
		}
		return !found
	})
	return found
}

func identityWordAliases(n ast.Node) map[string]bool {
	aliases := map[string]bool{}
	for {
		before := len(aliases)
		ast.Inspect(n, func(n ast.Node) bool {
			if a, ok := n.(*ast.AssignStmt); ok && len(a.Lhs) == len(a.Rhs) {
				for i, rhs := range a.Rhs {
					if id, ok := a.Lhs[i].(*ast.Ident); ok && identityWordRead(rhs, aliases) {
						aliases[id.Name] = true
					}
				}
			}
			return true
		})
		if len(aliases) == before {
			return aliases
		}
	}
}

func integerIn(n ast.Node, want int) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.INT {
			v, err := strconv.ParseInt(lit.Value, 0, 64)
			found = found || err == nil && v == int64(want)
		}
		return !found
	})
	return found
}
