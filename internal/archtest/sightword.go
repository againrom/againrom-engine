package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

var sightWordFiles = map[string]bool{"pkg/sim/sight.go": true}

// CheckSightWordBuilder reports SightSubCells calls outside pkg/rules, pkg/data, sim/sight.go.
func CheckSightWordBuilder(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the sight word scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if strings.HasPrefix(name, "pkg/rules/") || strings.HasPrefix(name, "pkg/data/") || sightWordFiles[name] {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			fun := ""
			switch c := call.Fun.(type) {
			case *ast.Ident:
				fun = c.Name
			case *ast.SelectorExpr:
				fun = c.Sel.Name
			}
			if fun == "SightSubCells" {
				vs = append(vs, Violation{From: name + ":" + strconv.Itoa(fset.Position(call.Pos()).Line),
					Reason: "SightSubCells outside pkg/rules, pkg/data and pkg/sim/sight.go is a second computation of the sight word; call rules.SightWordIn"})
			}
			return true
		})
	}
	return vs
}
