package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
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

// roomDataTypes are the pkg/ui room data types and the production files that
// may write one with a literal.
var roomDataTypes = map[string][]string{
	"RoomLayer":  {roomPageBuilder, roomPagePainter},
	"RoomSchool": {roomPageBuilder, roomPagePainter},
	"RoomHit":    {roomPageBuilder, roomPagePainter},
	"RoomPanel":  {roomPageBuilder, "pkg/ui/roompanel.go", "pkg/ui/widgetpanel.go"},
}

// CheckRoomLiterals reports each production composite literal with elements
// of a room data type outside the files that build room data. It reads the
// literal's own type, so an elided element type (`[]ui.RoomHit{{...}}`), an
// alias and a pointer literal are the type they stand for.
func CheckRoomLiterals(m *TypeCheckedModule) []Violation {
	var vs []Violation
	for path, p := range m.Packages {
		if !strings.HasPrefix(path, "againrom/pkg/") {
			continue
		}
		for i, file := range p.Files[:p.NumProd] {
			rel := p.RelPaths[i]
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok || len(lit.Elts) == 0 {
					return true
				}
				t := p.Info.Types[lit].Type
				if t == nil {
					return true
				}
				t = types.Unalias(t)
				if ptr, ok := t.(*types.Pointer); ok {
					t = types.Unalias(ptr.Elem())
				}
				named, ok := t.(*types.Named)
				if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "againrom/pkg/ui" {
					return true
				}
				allowed, isRoomData := roomDataTypes[named.Obj().Name()]
				if isRoomData && !slices.Contains(allowed, rel) {
					vs = append(vs, Violation{From: rel + ":" + strconv.Itoa(m.Fset.Position(lit.Pos()).Line),
						Reason: "second room data builder constructs " + named.Obj().Name() + "; use buildRoomPage"})
				}
				return true
			})
		}
	}
	sort.Slice(vs, func(i, j int) bool { return vs[i].From < vs[j].From })
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
