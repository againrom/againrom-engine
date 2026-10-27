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
	}
	return vs
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
