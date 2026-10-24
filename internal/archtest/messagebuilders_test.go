package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func messageRouteCalls(source, function, callee string, messageCase bool) bool {
	f, err := parser.ParseFile(token.NewFileSet(), "route.go", source, 0)
	if err != nil {
		return false
	}
	for _, declaration := range f.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Name.Name != function || fn.Body == nil {
			continue
		}
		var body ast.Node = fn.Body
		if messageCase {
			body = nil
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				c, ok := n.(*ast.CaseClause)
				if !ok || len(c.List) != 1 || len(c.Body) != 1 {
					return true
				}
				id, ok := c.List[0].(*ast.Ident)
				if _, ret := c.Body[0].(*ast.ReturnStmt); ok && id.Name == "wpMessage" && ret {
					body = c
				}
				return true
			})
		}
		if body == nil {
			return false
		}
		found := false
		ast.Inspect(body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
					found = found || selector.Sel.Name == callee
				}
			}
			return true
		})
		return found
	}
	return false
}

func TestMessageRoutesUseSharedBuilders(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []struct {
		file, function, callee string
		messageCase            bool
	}{
		{"pkg/ui/mapeditorwizard.go", "wizardChoices", "parameterChoices", true},
		{"pkg/ui/mapeditorwizard.go", "wizardSpec", "parameterRows", false},
		{"pkg/ui/mapeditortrigger.go", "scriptSpec", "parameterRows", false},
		{"pkg/ui/mapeditormessage.go", "createMessage", "CreateMessage", false},
		{"pkg/game/mapmessagepages.go", "CreateMessage", "NewMessage", false},
		{"pkg/game/mapmessagepages.go", "CreateMessage", "SetMessage", false},
	} {
		source := files[route.file]
		if !messageRouteCalls(source, route.function, route.callee, route.messageCase) {
			t.Errorf("%s:%s must call the shared %s builder", route.file, route.function, route.callee)
		}
		broken := strings.ReplaceAll(source, "."+route.callee+"(", ".otherBuilder(")
		if messageRouteCalls(broken, route.function, route.callee, route.messageCase) {
			t.Errorf("%s guard accepted a removed delegation", route.function)
		}
	}
}
