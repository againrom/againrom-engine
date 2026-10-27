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
// data and painted by ui.DrawRoomPage in roomPagePainter. A read of the seat
// grid or the card and figure placement anywhere else under pkg/ lays a page
// out a second time, and a numeric image.Rect or image.Pt in either file
// puts a page's geometry back into code.
const (
	roomPageBuilder = "pkg/game/roompageview.go"
	roomPagePainter = "pkg/ui/roompage.go"
)

// roomSeatGeometry is the seat data a page layout reads: the grid and its
// rectangle answer, and the card and figure placement.
var roomSeatGeometry = map[string]bool{"First": true, "Step": true, "RowStep": true, "Columns": true, "Rows": true,
	"Count": true, "Seat": true, "Card": true, "Figure": true}

// CheckRoomPage reports every production read of seat geometry through a
// Seats field outside roomPageBuilder and pkg/town, and every image.Rect or
// image.Pt with a number literal in the builder and the painter. files is
// keyed by module-relative slash path, as LoadDrawnTextSources returns it.
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
			switch n := n.(type) {
			case *ast.SelectorExpr:
				if inner, ok := n.X.(*ast.SelectorExpr); ok && inner.Sel.Name == "Seats" && roomSeatGeometry[n.Sel.Name] && name != roomPageBuilder {
					vs = append(vs, Violation{From: at(n), Reason: "reads Seats." + n.Sel.Name + "; a tavern or inn page is laid out by buildRoomPage and placeRoomPerson"})
				}
			case *ast.CallExpr:
				if !builder {
					return true
				}
				fun, ok := n.Fun.(*ast.SelectorExpr)
				if !ok || fun.Sel.Name != "Rect" && fun.Sel.Name != "Pt" {
					return true
				}
				if pkg, ok := fun.X.(*ast.Ident); !ok || pkg.Name != "image" {
					return true
				}
				for _, arg := range n.Args {
					if hasNumberLiteral(arg) {
						vs = append(vs, Violation{From: at(n), Reason: "image." + fun.Sel.Name + " with a number in the room page builder; page geometry is room data"})
						break
					}
				}
			}
			return true
		})
	}
	return vs
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
