package archtest

import (
	"go/ast"
	"go/types"
	"sort"
	"strconv"
)

// The one minimap plan producer and the one caller of the plan's picture builder.
const (
	minimapPlanPackage  = "againrom/pkg/ui"
	minimapPlanType     = "minimapPlan"
	minimapPlanProducer = "minimapProducePlan"
	minimapPlanPicture  = "composeMinimapPlan"
	minimapPlanCaller   = "minimapCompose"
)

// MinimapPlanSites lists what breaks the one minimap plan producer in the
// production code of pkg/ui. It reads types, so an elided element type, an
// alias, new, a declared variable, a field write, a function value and a
// method of the plan all count as what they are. A minimapPlan made or
// written outside minimapProducePlan, a second literal inside it, and a
// reference to composeMinimapPlan outside minimapCompose or a second one
// inside it are findings.
func MinimapPlanSites(m *TypeCheckedModule) []string {
	var sites []string
	literals, references := 0, 0
	cp := m.Packages[minimapPlanPackage]
	if cp == nil {
		return []string{"package " + minimapPlanPackage + " is not in the module"}
	}
	isPlan := func(t types.Type) bool {
		if t == nil {
			return false
		}
		t = types.Unalias(t)
		if p, ok := t.(*types.Pointer); ok {
			t = types.Unalias(p.Elem())
		}
		named, ok := t.(*types.Named)
		return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == minimapPlanPackage && named.Obj().Name() == minimapPlanType
	}
	for i, file := range cp.Files[:cp.NumProd] {
		rel := cp.RelPaths[i]
		at := func(n ast.Node) string { return rel + ":" + strconv.Itoa(m.Fset.Position(n.Pos()).Line) }
		visit := func(owner string, root ast.Node) {
			outside := func(n ast.Node, what string) {
				if owner != minimapPlanProducer {
					sites = append(sites, at(n)+" "+what+" outside "+minimapPlanProducer)
				}
			}
			ast.Inspect(root, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CompositeLit:
					if isPlan(cp.Info.Types[x].Type) {
						literals++
						outside(x, "writes a "+minimapPlanType)
					}
				case *ast.CallExpr:
					if id, ok := x.Fun.(*ast.Ident); ok && len(x.Args) == 1 {
						if b, _ := cp.Info.Uses[id].(*types.Builtin); b != nil && b.Name() == "new" && isPlan(cp.Info.Types[x.Args[0]].Type) {
							outside(x, "makes a "+minimapPlanType+" with new")
						}
					}
				case *ast.ValueSpec:
					if x.Type != nil && isPlan(cp.Info.Types[x.Type].Type) {
						outside(x, "declares a "+minimapPlanType+" variable")
					}
				case *ast.AssignStmt:
					for _, l := range x.Lhs {
						if sel, ok := l.(*ast.SelectorExpr); ok && isPlan(cp.Info.Types[sel.X].Type) {
							outside(l, "assigns a field of a "+minimapPlanType)
						}
					}
				case *ast.IncDecStmt:
					if sel, ok := x.X.(*ast.SelectorExpr); ok && isPlan(cp.Info.Types[sel.X].Type) {
						outside(x, "assigns a field of a "+minimapPlanType)
					}
				case *ast.Ident:
					if fn, _ := cp.Info.Uses[x].(*types.Func); fn != nil && fn.Name() == minimapPlanPicture && fn.Pkg() != nil && fn.Pkg().Path() == minimapPlanPackage {
						references++
						if owner != minimapPlanCaller {
							sites = append(sites, at(x)+" references "+minimapPlanPicture+" outside "+minimapPlanCaller)
						}
					}
				}
				return true
			})
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				visit(fn.Name.Name, fn)
				continue
			}
			visit("", decl)
		}
	}
	if literals != 1 {
		sites = append(sites, strconv.Itoa(literals)+" "+minimapPlanType+" literals in production code, want the one in "+minimapPlanProducer)
	}
	if references != 1 {
		sites = append(sites, strconv.Itoa(references)+" references to "+minimapPlanPicture+" in production code, want the one in "+minimapPlanCaller)
	}
	sort.Strings(sites)
	return sites
}
