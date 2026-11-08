package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// RoomGestureEntries retain action timing: retained rooms latch a press and
// compare its release; entry rooms act on release. Combining them moves an action.
var RoomGestureEntries = map[string]string{
	"stepTown":         "retained-room press latch and entry-room release dispatch",
	"TownSurfaceClick": "retained-room double click and control release",
}

var roomContractWrappers = map[string]string{
	"ComposeTownSurface":   "ComposeTownRoom",
	"ComposeTownSquare":    "ComposeTownRoom",
	"TownSurfaceControlAt": "SurfaceControlAt",
	"buildButtonPanel":     "build",
}

// CheckRoomContract keeps room data construction, painting, hit testing and
// frame composition on their named contract, including compatibility wrappers.
func CheckRoomContract(files map[string]string) []Violation {
	names := make([]string, 0, len(files))
	for name := range files {
		if strings.HasPrefix(name, "pkg/") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the room contract scan found nothing to read"}}
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
		report := func(n ast.Node, reason string) {
			vs = append(vs, Violation{From: name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line), Reason: reason})
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if target, ok := roomContractWrappers[fn.Name.Name]; ok && !roomContractForward(fn, target) {
				report(fn, fn.Name.Name+" must only return the one contract's "+target+" call")
			}
			if fn.Name.Name == "townSurfacePanel" {
				report(fn, "retired townSurfacePanel builds a second room panel; use buildRoomPage")
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					callee := ""
					switch fun := call.Fun.(type) {
					case *ast.Ident:
						callee = fun.Name
					case *ast.SelectorExpr:
						callee = fun.Sel.Name
					}
					if callee == "DrawRoomPage" && !(name == "pkg/ui/roomframe.go" && fn.Name.Name == "ComposeTownRoom") && !(name == "pkg/game/secondtown.go" && fn.Name.Name == "Paint") {
						report(call, "second room frame composer calls DrawRoomPage; use ComposeTownRoom and roomPageView.Paint")
					}
				}
				return true
			})
			if name == "pkg/game/secondtown.go" && fn.Name.Name == "ControlAt" {
				ast.Inspect(fn, func(n ast.Node) bool {
					if call, ok := n.(*ast.CallExpr); ok {
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "In" || sel.Sel.Name == "SeatAt" || sel.Sel.Name == "ButtonAt") {
							report(call, "roomPageView.ControlAt hit-tests room geometry outside RoomPage.ControlAt")
						}
					}
					return true
				})
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.TypeSpec:
				if x.Name.Name == "TownSurfaceView" || x.Name.Name == "TownSquareView" {
					id, ok := x.Type.(*ast.Ident)
					if !x.Assign.IsValid() || !ok || id.Name != "TownRoomView" {
						report(x, x.Name.Name+" must alias TownRoomView")
					}
				}
			case *ast.CompositeLit:
				typ := ""
				switch t := x.Type.(type) {
				case *ast.Ident:
					typ = t.Name
				case *ast.SelectorExpr:
					typ = t.Sel.Name
				}
				if len(x.Elts) == 0 {
					break
				}
				if (typ == "RoomLayer" || typ == "RoomSchool" || typ == "RoomHit") && name != roomPageBuilder && name != roomPagePainter {
					report(x, "second room data builder constructs "+typ+"; use buildRoomPage")
				}
				if typ == "RoomPanel" && name != roomPageBuilder && name != "pkg/ui/roompanel.go" && name != "pkg/ui/widgetpanel.go" {
					report(x, "second room data builder constructs RoomPanel; use buildRoomPage")
				}
			case *ast.IfStmt, *ast.SwitchStmt:
				if name != roomPageBuilder && !strings.HasPrefix(name, "pkg/ui/room") {
					break
				}
				var cond ast.Node
				switch stmt := n.(type) {
				case *ast.IfStmt:
					cond = stmt.Cond
				case *ast.SwitchStmt:
					cond = stmt.Tag
				}
				if cond == nil {
					break
				}
				ast.Inspect(cond, func(c ast.Node) bool {
					if id, ok := c.(*ast.Ident); ok && (id.Name == "GameROM1" || id.Name == "GameROM2" || id.Name == "isSecondGame") {
						report(id, "game identity chooses a room contract path; use room data")
					}
					return true
				})
			}
			return true
		})
	}
	return vs
}

func roomContractForward(fn *ast.FuncDecl, target string) bool {
	if fn.Body == nil || len(fn.Body.List) != 1 {
		return false
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	call, ok := ret.Results[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name == target
	case *ast.SelectorExpr:
		return fun.Sel.Name == target
	}
	return false
}
