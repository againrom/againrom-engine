package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// CheckPanelProfileReader reports each json.Unmarshal call in the source of
// the editor's side panels outside readPanelProfile, the one reader of the
// panel profile file, and reports a source that has no readPanelProfile.
func CheckPanelProfileReader(name string, src any) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil, err
	}
	var findings []string
	found := false
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		reader := fn.Recv == nil && fn.Name.Name == "readPanelProfile"
		found = found || reader
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || reader {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Unmarshal" {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "json" {
				findings = append(findings, fmt.Sprintf("%s: %s decodes json outside readPanelProfile", fset.Position(call.Pos()), fn.Name.Name))
			}
			return true
		})
	}
	if !found {
		findings = append(findings, name+": readPanelProfile is not declared")
	}
	return findings, nil
}
