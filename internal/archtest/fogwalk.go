package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The fog walk scan: pkg/sim's fogStamp is the one fog walk of both games.
// It alone names the fog reader fogSight, it marches once, and it reads no
// game identity: the first game takes the second game's walk as data.
const (
	fogWalkFile   = "pkg/sim/sight.go"
	fogWalkStamp  = "fogStamp"
	fogWalkReader = "fogSight"
)

// fogWalkIdentity is what a branch on the running game reads in pkg/sim.
var fogWalkIdentity = map[string]bool{"SecondGame": true, "rom2": true}

// CheckFogWalk reports every production use of fogSight outside fogStamp,
// a fogStamp that marches other than once, and a fogStamp that reads the
// running game. files is keyed by module-relative slash path, as
// LoadDrawnTextSources returns it.
func CheckFogWalk(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the fog walk scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	marches, found := 0, false
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
		for _, decl := range f.Decls {
			owner := ""
			fn, _ := decl.(*ast.FuncDecl)
			if fn != nil {
				owner = fn.Name.Name
			}
			stamp := name == fogWalkFile && owner == fogWalkStamp
			found = found || stamp
			at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
			ast.Inspect(decl, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.ValueSpec:
					// The reader's own declaration is not a use.
					if name == fogWalkFile && fn == nil {
						for _, v := range x.Values {
							ast.Inspect(v, func(m ast.Node) bool {
								if id, ok := m.(*ast.Ident); ok && id.Name == fogWalkReader {
									vs = append(vs, Violation{From: at(id), Reason: "the fog reader is built from itself"})
								}
								return true
							})
						}
						return false
					}
				case *ast.Ident:
					if x.Name == fogWalkReader && !stamp {
						vs = append(vs, Violation{From: at(x), Reason: fogWalkReader + " is used outside fogStamp; fogStamp is the one fog walk of both games"})
					}
					if stamp && fogWalkIdentity[x.Name] {
						vs = append(vs, Violation{From: at(x), Reason: "fogStamp reads the running game (" + x.Name + "); the first game takes the second game's walk as data"})
					}
				case *ast.CallExpr:
					if stamp && fogWalkCallsReader(x) {
						marches++
					}
				}
				return true
			})
		}
	}
	if !found {
		vs = append(vs, Violation{From: fogWalkFile, Reason: "no fogStamp in pkg/sim/sight.go; the fog walk scan guards nothing"})
	} else if marches != 1 {
		vs = append(vs, Violation{From: fogWalkFile, Reason: "fogStamp marches " + strconv.Itoa(marches) + " times; the fog walk is one march"})
	}
	return vs
}

// fogWalkCallsReader reports a call taking fogSight as an argument.
func fogWalkCallsReader(c *ast.CallExpr) bool {
	for _, a := range c.Args {
		if id, ok := a.(*ast.Ident); ok && id.Name == fogWalkReader {
			return true
		}
	}
	return false
}
