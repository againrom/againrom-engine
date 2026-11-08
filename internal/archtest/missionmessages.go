package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

const missionMessagesFile = "pkg/game/secondgamenotices.go"

// These are storage and observation ports, not delivery decisions.
var missionMessagePorts = map[string]map[string]bool{
	"pkg/game/residuecarrier.go":  {"pendingMessages": true},
	"pkg/game/scenario.go":        {"Sample": true, "ReservedMessageNumber": true},
	"pkg/game/mapcheck.go":        {"ReservedMessageNumber": true},
	"pkg/game/mapmessagepages.go": {"ReservedMessageNumber": true},
}

// CheckMissionMessages keeps one settle function, queue and special-number
// interpreter. Campaign services cannot replace delivery with a game hook.
func CheckMissionMessages(files map[string]string) []Violation {
	var names []string
	for name := range files {
		if strings.HasPrefix(name, "pkg/game/") && !strings.Contains(strings.TrimPrefix(name, "pkg/game/"), "/") &&
			strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return []Violation{{From: "pkg/game", Reason: "no sources scanned for mission messages"}}
	}
	sort.Strings(names)
	var vs []Violation
	settles := 0
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		report := func(n ast.Node, reason string) {
			vs = append(vs, Violation{From: name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line), Reason: reason})
		}
		outside := func(n ast.Node, symbol string) {
			if name != missionMessagesFile && !missionMessagePorts[name][symbol] {
				report(n, symbol+" outside the mission message pipeline")
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.ValueSpec:
				for _, id := range n.Names {
					lower := strings.ToLower(id.Name)
					if !strings.Contains(lower, "message") && !strings.Contains(lower, "notice") {
						continue
					}
					for _, value := range n.Values {
						if _, composite := value.(*ast.CompositeLit); !composite {
							continue
						}
						ast.Inspect(value, func(x ast.Node) bool {
							if missionMessageNumber(x) {
								outside(id, "special-number table "+id.Name)
							}
							return true
						})
					}
				}
			case *ast.CaseClause:
				for _, value := range n.List {
					if missionMessageNumber(value) {
						outside(value, "special-number dispatch")
					}
				}
			case *ast.BinaryExpr:
				for _, expr := range []ast.Expr{n.X, n.Y} {
					if id, ok := expr.(*ast.Ident); ok && (id.Name == "ReservedMessageNumber" || id.Name == "ObjectiveEventDone" || id.Name == "ObjectiveEventMode4") {
						outside(id, id.Name)
					}
				}
			case *ast.FuncDecl:
				lower := strings.ToLower(n.Name.Name)
				if strings.Contains(lower, "settle") && (strings.Contains(lower, "notice") || strings.Contains(lower, "message")) {
					settles++
					if name != missionMessagesFile || n.Name.Name != "settleNotices" || receiverName(n.Recv) != "mapWorld" {
						report(n, "second mission message settle function: "+n.Name.Name)
					}
				}
				if n.Name.Name == "observeSecondGameMessages" || n.Name.Name == "acknowledgeNotice" || n.Name.Name == "scriptMessages" {
					report(n, "per-game mission message hook: "+n.Name.Name)
				}
			case *ast.SelectorExpr:
				if n.Sel.Name == "pendingMessages" {
					outside(n, n.Sel.Name)
				}
				if name == missionMessagesFile && (n.Sel.Name == "GameROM1" || n.Sel.Name == "GameROM2" || n.Sel.Name == "GameOf" || n.Sel.Name == "FirstData") {
					report(n, "game identity branch in mission message delivery")
				}
			case *ast.CallExpr:
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
					switch sel.Sel.Name {
					case "latchOutcome":
						outside(n, "latchOutcome")
					case "Sample":
						if recv, ok := sel.X.(*ast.SelectorExpr); ok && recv.Sel.Name == "ann" {
							outside(n, "Sample")
						}
					}
				}
			case *ast.TypeSpec:
				if n.Name.Name == "missionNotices" {
					if st, ok := n.Type.(*ast.StructType); ok {
						for _, field := range st.Fields.List {
							array, ok := field.Type.(*ast.ArrayType)
							if !ok || array.Len != nil {
								continue
							}
							id, ok := array.Elt.(*ast.Ident)
							if ok && id.Name == "int32" {
								for _, id := range field.Names {
									if id.Name != "pendingMessages" {
										report(id, "second mission message queue: "+id.Name)
									}
								}
							}
						}
					}
				}
				if n.Name.Name == "campaignService" {
					if it, ok := n.Type.(*ast.InterfaceType); ok {
						for _, field := range it.Methods.List {
							for _, id := range field.Names {
								if id.Name == "scriptMessages" || id.Name == "settleNotices" || id.Name == "acknowledgeNotice" {
									report(id, "campaignService mission delivery hook: "+id.Name)
								}
							}
						}
					}
				}
			}
			return true
		})
	}
	if settles != 1 {
		vs = append(vs, Violation{From: "pkg/game", Reason: "mission message settle functions: " + strconv.Itoa(settles) + ", want 1"})
	}
	return vs
}

func missionMessageNumber(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.BasicLit:
		if n.Kind == token.INT {
			v, err := strconv.ParseInt(n.Value, 0, 64)
			return err == nil && (v == 250 || v == 253 || v == 254 || v == 255)
		}
	case *ast.Ident:
		return n.Name == "ReservedMessageNumber"
	case *ast.SelectorExpr:
		return n.Sel.Name == "ObjectiveEventDone" || n.Sel.Name == "ObjectiveEventMode4"
	}
	return false
}
