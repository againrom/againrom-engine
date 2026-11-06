package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The room page scan: a tavern or inn page of either game is laid out by
// buildRoomPage and placeRoomPerson in roomPageBuilder from the room's seat
// data and painted and hit-tested by ui.DrawRoomPage and RoomPage.SeatAt in
// roomPagePainter. A read of the seat grid or the card and figure placement
// anywhere else under pkg/ lays a page out a second time, a read of a built
// seat's rectangle anywhere else paints or hit-tests it a second time, and a
// numeric image.Rect or image.Pt in either file puts a page's geometry back
// into code. The scan follows a Seats value through a local name, a range
// variable, an element taken by index and a *SeatSpec parameter.
const (
	roomPageBuilder = "pkg/game/roompageview.go"
	roomPagePainter = "pkg/ui/roompage.go"
)

// roomSeatGeometry is the seat data a page layout reads: the grid and its
// rectangle answer, and the card and figure placement.
var roomSeatGeometry = map[string]bool{"First": true, "Step": true, "RowStep": true, "Columns": true, "Rows": true,
	"Count": true, "Seat": true, "Card": true, "Figure": true}

// CheckRoomPage reports every production read of seat geometry through a
// Seats field or a name bound to one outside roomPageBuilder and pkg/town,
// every read of a built seat's Rect outside roomPageBuilder and
// roomPagePainter, and every image.Rect or image.Pt with a number literal in
// the builder and the painter. files is keyed by module-relative slash path,
// as LoadDrawnTextSources returns it.
func CheckRoomPage(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the room page scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/") || strings.HasPrefix(name, "pkg/town/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		builder := name == roomPageBuilder || name == roomPagePainter
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok && builder {
				if v, ok := numberInRoomPageGeometry(call); ok {
					vs = append(vs, Violation{From: at(call), Reason: v})
				}
			}
			return true
		})
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				vs = append(vs, roomSeatReads(fn, name, at)...)
			}
		}
		vs = append(vs, roomPageShapes(f, name, at)...)
	}
	return vs
}

// Shared chrome, text bands and canvas bounds are not room controls. Counts
// prevent adding a control rectangle inside an admitted painter.
var roomShellGeometry = map[string]int{
	"TownContentRegion": 1, "TownUpperRegion": 1, "TownWideUpperRegion": 1, "TownCharacterRegion": 1,
	"townButtonLabelRect": 1, "townButtonValueRect": 1, "townSurfaceMessageRect": 1,
	"drawCharacterPaneBody": 1, "ComposeTownSurface": 4, "shiftedCardBackground": 3,
}

func roomPageShapes(f *ast.File, name string, at func(ast.Node) string) []Violation {
	var vs []Violation
	imageNames := map[string]bool{"image": true}
	for _, imp := range f.Imports {
		if imp.Path.Value == `"image"` && imp.Name != nil {
			imageNames[imp.Name.Name] = true
		}
	}
	rectangle := func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Rect" {
				if id, ok := sel.X.(*ast.Ident); ok && imageNames[id.Name] {
					return hasNumberLiteral(n)
				}
			}
		case *ast.CompositeLit:
			if sel, ok := n.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Rectangle" {
				if id, ok := sel.X.(*ast.Ident); ok && imageNames[id.Name] {
					return hasNumberLiteral(n)
				}
			}
		}
		return false
	}
	methods := map[string]map[string]bool{}
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Recv != nil {
				typ := fn.Recv.List[0].Type
				if ptr, ok := typ.(*ast.StarExpr); ok {
					typ = ptr.X
				}
				if id, ok := typ.(*ast.Ident); ok {
					if methods[id.Name] == nil {
						methods[id.Name] = map[string]bool{}
					}
					methods[id.Name][fn.Name.Name] = true
				}
			}
			count := 0
			ast.Inspect(fn, func(n ast.Node) bool {
				if name == "pkg/ui/townshell.go" && rectangle(n) {
					count++
					if count > roomShellGeometry[fn.Name.Name] {
						vs = append(vs, Violation{From: at(n), Reason: "Go rectangle for room control in " + fn.Name.Name + "; control geometry belongs to the room description"})
					}
				}
				if call, ok := n.(*ast.CallExpr); ok && isRoomPersonSource(name) && fn.Name.Name != "buildRoomPatron" {
					if id, ok := call.Fun.(*ast.Ident); ok && (id.Name == "partyPanelSubject" || id.Name == "composeMemberPortrait") {
						vs = append(vs, Violation{From: at(call), Reason: "second room person builder; use buildRoomPatron with source profile data"})
					}
				}
				return true
			})
		}
	}
	for typ, m := range methods {
		if strings.HasPrefix(name, "pkg/game/") && typ != "roomPageView" && m["Paint"] && (m["ControlAt"] || m["TipAt"]) {
			vs = append(vs, Violation{From: name, Reason: "second room page adapter " + typ + "; use roomPageView"})
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ValueSpec:
			if name == "pkg/ui/townshell.go" {
				for _, id := range n.Names {
					count := 0
					for _, v := range n.Values {
						ast.Inspect(v, func(child ast.Node) bool {
							if rectangle(child) {
								count++
								if count > roomShellGeometry[id.Name] {
									vs = append(vs, Violation{From: at(child), Reason: "Go rectangle for room control " + id.Name + "; control geometry belongs to the room description"})
								}
							}
							return true
						})
					}
				}
			}
		case *ast.TypeSpec:
			if !strings.HasPrefix(name, "pkg/game/") || n.Name.Name == "roomPageView" {
				break
			}
			if body, ok := n.Type.(*ast.StructType); ok && (len(body.Fields.List) == 1 || methods[n.Name.Name]["Paint"]) {
				for _, field := range body.Fields.List {
					typ := field.Type
					if ptr, ok := typ.(*ast.StarExpr); ok {
						typ = ptr.X
					}
					if sel, ok := typ.(*ast.SelectorExpr); ok && sel.Sel.Name == "Scene" {
						vs = append(vs, Violation{From: at(n), Reason: "second room page adapter " + n.Name.Name + " holds a scene; use roomPageView"})
					}
				}
			}
		case *ast.CompositeLit:
			typ := ""
			switch t := n.Type.(type) {
			case *ast.SelectorExpr:
				typ = t.Sel.Name
			case *ast.Ident:
				typ = t.Name
			}
			if (typ == "RoomSeat" || typ == "RoomPage" || typ == "RoomBackdrop") && len(n.Elts) > 0 && name != roomPageBuilder && name != roomPagePainter {
				vs = append(vs, Violation{From: at(n), Reason: "second room page builder constructs " + typ + "; use buildRoomPage"})
			}
			if typ == "roomPageView" && name != "pkg/game/roompage.go" {
				vs = append(vs, Violation{From: at(n), Reason: "second room page reader; use townScreen.roomView"})
			}
		case *ast.IfStmt:
			if name == roomPageBuilder || name == "pkg/game/secondtown.go" || name == "pkg/ui/townshell.go" {
				ast.Inspect(n.Cond, func(child ast.Node) bool {
					if id, ok := child.(*ast.Ident); ok && (id.Name == "GameROM1" || id.Name == "GameROM2" || id.Name == "isSecondGame") {
						vs = append(vs, Violation{From: at(id), Reason: "game identity chooses a room page path; use room profile data"})
					}
					return true
				})
			}
		}
		return true
	})
	return vs
}

func isRoomPersonSource(name string) bool {
	return name == "pkg/game/townshell.go" || name == "pkg/game/taverntalk.go" || name == "pkg/game/secondinnpatron.go"
}

// numberInRoomPageGeometry answers the reason when call is image.Rect or
// image.Pt with a number literal argument.
func numberInRoomPageGeometry(call *ast.CallExpr) (string, bool) {
	fun, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || fun.Sel.Name != "Rect" && fun.Sel.Name != "Pt" {
		return "", false
	}
	if pkg, ok := fun.X.(*ast.Ident); !ok || pkg.Name != "image" {
		return "", false
	}
	for _, arg := range call.Args {
		if hasNumberLiteral(arg) {
			return "image." + fun.Sel.Name + " with a number in the room page builder; page geometry is room data", true
		}
	}
	return "", false
}

// seatNames are the names a function binds to a Seats value (grids) and to
// one element of a page's seats (elems).
type seatNames struct{ grids, elems map[string]bool }

// isSeats reports whether e is a Seats field, a slice of one, or a name
// bound to one.
func (s seatNames) isSeats(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.ParenExpr:
		return s.isSeats(e.X)
	case *ast.SliceExpr:
		return s.isSeats(e.X)
	case *ast.SelectorExpr:
		return e.Sel.Name == "Seats"
	case *ast.Ident:
		return s.grids[e.Name]
	}
	return false
}

// isSeat reports whether e is one element of a Seats value, or its address.
func (s seatNames) isSeat(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.ParenExpr:
		return s.isSeat(e.X)
	case *ast.UnaryExpr:
		return e.Op == token.AND && s.isSeat(e.X)
	case *ast.IndexExpr:
		return s.isSeats(e.X)
	case *ast.Ident:
		return s.elems[e.Name]
	}
	return false
}

// isSeatSpec reports whether t names the room data's seat type, or a pointer
// to it.
func isSeatSpec(t ast.Expr) bool {
	switch t := t.(type) {
	case *ast.StarExpr:
		return isSeatSpec(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name == "SeatSpec"
	case *ast.Ident:
		return t.Name == "SeatSpec"
	}
	return false
}

// bind records the names fn binds to a Seats value, until no more are found.
func (s *seatNames) bind(fn *ast.FuncDecl) {
	add := func(m map[string]bool, e ast.Expr) bool {
		id, ok := e.(*ast.Ident)
		if !ok || id.Name == "_" || m[id.Name] {
			return false
		}
		m[id.Name] = true
		return true
	}
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			if isSeatSpec(field.Type) {
				for _, id := range field.Names {
					add(s.grids, id)
				}
			}
		}
	}
	for changed := true; changed; {
		changed = false
		ast.Inspect(fn, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				if len(n.Lhs) != len(n.Rhs) {
					return true
				}
				for i, rhs := range n.Rhs {
					switch {
					case s.isSeats(rhs):
						changed = add(s.grids, n.Lhs[i]) || changed
					case s.isSeat(rhs):
						changed = add(s.elems, n.Lhs[i]) || changed
					}
				}
			case *ast.ValueSpec:
				if isSeatSpec(n.Type) {
					for _, id := range n.Names {
						changed = add(s.grids, id) || changed
					}
				}
				if len(n.Names) == len(n.Values) {
					for i, v := range n.Values {
						switch {
						case s.isSeats(v):
							changed = add(s.grids, n.Names[i]) || changed
						case s.isSeat(v):
							changed = add(s.elems, n.Names[i]) || changed
						}
					}
				}
			case *ast.RangeStmt:
				if s.isSeats(n.X) && n.Value != nil {
					changed = add(s.elems, n.Value) || changed
				}
			}
			return true
		})
	}
}

// roomSeatReads reports the seat reads of fn that only the builder and the
// painter may make.
func roomSeatReads(fn *ast.FuncDecl, name string, at func(ast.Node) string) []Violation {
	if name == roomPageBuilder || fn.Body == nil {
		return nil
	}
	s := seatNames{grids: map[string]bool{}, elems: map[string]bool{}}
	s.bind(fn)
	var vs []Violation
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch {
		case roomSeatGeometry[sel.Sel.Name] && isSeatsField(sel.X):
			vs = append(vs, Violation{From: at(sel), Reason: "reads Seats." + sel.Sel.Name + "; a tavern or inn page is laid out by buildRoomPage and placeRoomPerson"})
		case roomSeatGeometry[sel.Sel.Name] && isBoundGrid(sel.X, s):
			vs = append(vs, Violation{From: at(sel), Reason: "reads " + sel.Sel.Name + " of a name bound to Seats; a tavern or inn page is laid out by buildRoomPage and placeRoomPerson"})
		case sel.Sel.Name == "Rect" && name != roomPagePainter && s.isSeat(sel.X):
			vs = append(vs, Violation{From: at(sel), Reason: "reads the Rect of a built page's seat; ui.DrawRoomPage paints it and RoomPage.SeatAt hit-tests it"})
		}
		return true
	})
	return vs
}

// isSeatsField reports whether e is a Seats field selector.
func isSeatsField(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Seats"
}

// isBoundGrid reports whether e is a name the function bound to a Seats value.
func isBoundGrid(e ast.Expr, s seatNames) bool {
	id, ok := e.(*ast.Ident)
	return ok && s.grids[id.Name]
}

func hasNumberLiteral(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && (lit.Kind == token.INT || lit.Kind == token.FLOAT) {
			found = true
		}
		return !found
	})
	return found
}
