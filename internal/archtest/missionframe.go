package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The mission frame scan: the frame is one plan, missionLayers, walked by
// the window executor (drawLayers) and the CPU executor
// (headlessMissionFrame), and the final-frame filter rule is one table,
// stageOf in pkg/ui/framestage.go. In pkg/ui production code it refuses a
// filter name compared (==, != or a switch case) outside framestage.go, and
// any use of missionLayers outside its declaration and the two executors.

var missionFilterNames = map[string]bool{"filterSmooth": true, "filterSharp": true, "filterPixel": true}

var missionLayerWalkers = map[string]bool{"drawLayers": true, "headlessMissionFrame": true}

// CheckMissionFrame reports every second filter rule and every second walk
// of the mission frame plan. files is keyed by module-relative slash path.
func CheckMissionFrame(files map[string]string) []Violation {
	var names []string
	for name := range files {
		if strings.HasPrefix(name, "pkg/ui/") && !strings.Contains(strings.TrimPrefix(name, "pkg/ui/"), "/") &&
			!strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var vs []Violation
	declared := false
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		filterName := func(e ast.Expr) bool {
			id, ok := e.(*ast.Ident)
			return ok && missionFilterNames[id.Name]
		}
		for _, d := range f.Decls {
			walker := false
			var own map[*ast.Ident]bool
			switch d := d.(type) {
			case *ast.FuncDecl:
				walker = missionLayerWalkers[d.Name.Name]
			case *ast.GenDecl:
				own = map[*ast.Ident]bool{}
				for _, sp := range d.Specs {
					if vsp, ok := sp.(*ast.ValueSpec); ok {
						for _, id := range vsp.Names {
							if id.Name == "missionLayers" {
								own[id], declared = true, true
							}
						}
					}
				}
			}
			ast.Inspect(d, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.BinaryExpr:
					if (n.Op == token.EQL || n.Op == token.NEQ) && (filterName(n.X) || filterName(n.Y)) && name != "pkg/ui/framestage.go" {
						vs = append(vs, Violation{From: at(n), Reason: "a frame filter compared outside framestage.go; ask stageOf"})
					}
				case *ast.CaseClause:
					for _, e := range n.List {
						if filterName(e) && name != "pkg/ui/framestage.go" {
							vs = append(vs, Violation{From: at(e), Reason: "a frame filter switched on outside framestage.go; ask stageOf"})
						}
					}
				case *ast.Ident:
					if n.Name == "missionLayers" && !walker && !own[n] {
						vs = append(vs, Violation{From: at(n), Reason: "missionLayers used outside drawLayers and headlessMissionFrame"})
					}
				}
				return true
			})
		}
	}
	if !declared {
		vs = append(vs, Violation{From: "pkg/ui", Reason: "no missionLayers declaration scanned - the mission frame scan found nothing to read"})
	}
	return vs
}
