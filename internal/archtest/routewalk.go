package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// The route witness scan: each game's step table is rows of one shape, and
// one routeWitness method serves each stage kind in both games. It refuses,
// in pkg/game, a routeWitness method named first*, and in
// pkg/game/routewitness*.go each shape a per-game walk took: a value named
// second, a game compared or switched on, a profile's HasFirstData or
// GameOf, a nil test of the witness's town or chapter field, and a row of a
// []routeStage literal that holds a function instead of naming its kind.

const (
	routeWalkDir    = "pkg/game/"
	routeWalkPrefix = "routewitness"
	routeWalkType   = "routeWitness"
	routeWalkRow    = "routeStage"
)

// routeWalkGames are the game names a per-game branch compares with.
var routeWalkGames = map[string]bool{"GameROM1": true, "GameROM2": true}

// routeWalkProfileReads are the profile reads that tell the games apart.
var routeWalkProfileReads = map[string]bool{"HasFirstData": true, "GameOf": true}

// CheckRouteWalk reports every second path to the route witness's stage
// builder in files, keyed by module-relative path.
func CheckRouteWalk(files map[string]string) []Violation {
	names := make([]string, 0, len(files))
	for name := range files {
		if strings.HasPrefix(name, routeWalkDir) && !strings.Contains(strings.TrimPrefix(name, routeWalkDir), "/") && strings.HasSuffix(name, ".go") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var vs []Violation
	scanned := 0
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		routeWalkMethods(f, at, &vs)
		if strings.HasPrefix(path.Base(name), routeWalkPrefix) {
			scanned++
			routeWalkBranches(f, at, &vs)
		}
	}
	if scanned == 0 {
		vs = append(vs, Violation{From: routeWalkDir, Reason: "no pkg/game/routewitness*.go scanned - the route witness scan found nothing to read"})
	}
	return vs
}

// routeWalkMethods refuses a routeWitness method named first*.
func routeWalkMethods(f *ast.File, at func(ast.Node) string, vs *[]Violation) {
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 || !routeWalkReceiver(fd.Recv.List[0].Type) {
			continue
		}
		if n := fd.Name.Name; strings.HasPrefix(n, "first") && len(n) > len("first") && unicode.IsUpper(rune(n[len("first")])) {
			*vs = append(*vs, Violation{From: at(fd.Name), Reason: "routeWitness." + n + " is a first-game stage walk beside the shared one; give the stage kind's one method the row's data"})
		}
	}
}

func routeWalkReceiver(e ast.Expr) bool {
	if s, ok := e.(*ast.StarExpr); ok {
		e = s.X
	}
	id, ok := e.(*ast.Ident)
	return ok && id.Name == routeWalkType
}

// routeWalkBranches refuses each per-game branch shape in a route witness
// file.
func routeWalkBranches(f *ast.File, at func(ast.Node) string, vs *[]Violation) {
	add := func(n ast.Node, reason string) { *vs = append(*vs, Violation{From: at(n), Reason: reason}) }
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			if n.Name == "second" {
				add(n, "a value named second picks a game inside the route witness; put the difference in the stage rows' data")
			}
		case *ast.SelectorExpr:
			if routeWalkProfileReads[n.Sel.Name] {
				add(n, n.Sel.Name+" tells the games apart inside the route witness; the profile's campaign service answers the table in routeStages")
			}
		case *ast.BinaryExpr:
			if n.Op != token.EQL && n.Op != token.NEQ {
				return true
			}
			for _, pair := range [2][2]ast.Expr{{n.X, n.Y}, {n.Y, n.X}} {
				if g := routeWalkGame(pair[0]); g != "" {
					add(n, "a comparison with "+g+" branches on the game inside the route witness; put the difference in the stage rows' data")
					break
				}
				if fld := routeWalkTownField(pair[0]); fld != "" && routeWalkNil(pair[1]) {
					add(n, "a nil test of the witness's "+fld+" field branches on the game; the witness holds one town screen")
					break
				}
			}
		case *ast.SwitchStmt:
			if n.Tag != nil && routeWalkGameCall(n.Tag) {
				add(n, "a switch on the game branches inside the route witness; put the difference in the stage rows' data")
			}
		case *ast.CompositeLit:
			if !routeWalkRows(n.Type) {
				return true
			}
			for _, row := range n.Elts {
				ast.Inspect(row, func(m ast.Node) bool {
					switch m := m.(type) {
					case *ast.FuncLit:
						add(m, "a stage row holds a function literal; the row names its stage kind and carries data")
						return false
					case *ast.SelectorExpr:
						if p, ok := m.X.(*ast.ParenExpr); ok && routeWalkReceiver(p.X) {
							add(m, "a stage row holds the method "+m.Sel.Name+"; the row names its stage kind and carries data")
						}
					}
					return true
				})
			}
		}
		return true
	})
}

// routeWalkGame names the game constant e selects, or "".
func routeWalkGame(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.ParenExpr:
		return routeWalkGame(x.X)
	case *ast.Ident:
		if routeWalkGames[x.Name] {
			return x.Name
		}
	case *ast.SelectorExpr:
		if routeWalkGames[x.Sel.Name] {
			return x.Sel.Name
		}
	case *ast.CallExpr:
		if routeWalkGameCall(x) {
			return "Game()"
		}
	}
	return ""
}

// routeWalkGameCall reports whether e is a call of a Game method.
func routeWalkGameCall(e ast.Expr) bool {
	c, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	s, ok := c.Fun.(*ast.SelectorExpr)
	return ok && s.Sel.Name == "Game"
}

// routeWalkTownField names the town or chapter field e selects, or "".
func routeWalkTownField(e ast.Expr) string {
	if p, ok := e.(*ast.ParenExpr); ok {
		return routeWalkTownField(p.X)
	}
	if s, ok := e.(*ast.SelectorExpr); ok && (s.Sel.Name == "town" || s.Sel.Name == "chapter") {
		return s.Sel.Name
	}
	return ""
}

func routeWalkNil(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "nil"
}

// routeWalkRows reports whether t is []routeStage.
func routeWalkRows(t ast.Expr) bool {
	a, ok := t.(*ast.ArrayType)
	if !ok {
		return false
	}
	id, ok := a.Elt.(*ast.Ident)
	return ok && id.Name == routeWalkRow
}
