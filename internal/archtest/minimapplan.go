package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// The one minimap plan producer and the one caller of the plan's picture builder.
const (
	minimapPlanType     = "minimapPlan"
	minimapPlanProducer = "minimapProducePlan"
	minimapPlanPicture  = "composeMinimapPlan"
	minimapPlanCaller   = "minimapCompose"
)

// MinimapPlanSites lists what breaks the one minimap plan producer in the
// production code of pkg/ui: a minimapPlan literal outside minimapProducePlan
// or a second one inside it, and a call of composeMinimapPlan outside
// minimapCompose or a second call inside it.
func MinimapPlanSites(root string) ([]string, error) {
	const dir = "pkg/ui"
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil {
		return nil, err
	}
	sources := map[string][]byte{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(dir), name))
		if err != nil {
			return nil, err
		}
		sources[dir+"/"+name] = src
	}
	return minimapPlanSitesIn(sources), nil
}

func minimapPlanSitesIn(sources map[string][]byte) []string {
	var sites []string
	literals, calls := 0, 0
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, name, sources[name], parser.SkipObjectResolution)
		if err != nil {
			sites = append(sites, name+" does not parse: "+err.Error())
			continue
		}
		visit := func(owner string, root ast.Node) {
			ast.Inspect(root, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CompositeLit:
					if id, ok := x.Type.(*ast.Ident); ok && id.Name == minimapPlanType {
						literals++
						if owner != minimapPlanProducer {
							sites = append(sites, name+":"+strconv.Itoa(fset.Position(x.Pos()).Line)+" writes a "+minimapPlanType+" outside "+minimapPlanProducer)
						}
					}
				case *ast.CallExpr:
					if id, ok := x.Fun.(*ast.Ident); ok && id.Name == minimapPlanPicture {
						calls++
						if owner != minimapPlanCaller {
							sites = append(sites, name+":"+strconv.Itoa(fset.Position(x.Pos()).Line)+" calls "+minimapPlanPicture+" outside "+minimapPlanCaller)
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
	if calls != 1 {
		sites = append(sites, strconv.Itoa(calls)+" calls of "+minimapPlanPicture+" in production code, want the one in "+minimapPlanCaller)
	}
	sort.Strings(sites)
	return sites
}
