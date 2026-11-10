package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Executors ask the admission which arm owns an actor, not the carriers.
func TestExecutorGatesReadTheAdmission(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "pkg", "sim", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	gates := map[string]bool{"stepBookCasts": true, "stepScrollCasts": true, "takePendingOrders": true, "executorProgress": true, "pendingLogicalProgress": true, "stepWorld": true}
	views := map[string]bool{"executorProgress": true, "pendingLogicalProgress": true}
	seen := map[string]bool{}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Body != nil && fn.Name.Name == "stepWorld" {
				seen["stepWorld"] = true
				asked := map[string]bool{}
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					switch sel.Sel.Name {
					case "motionActive":
						t.Errorf("stepWorld tests motionActive beside the admission")
					case "waits":
						if id, ok := call.Args[0].(*ast.Ident); ok {
							asked[id.Name] = true
						}
					}
					return true
				})
				for _, gate := range []string{"executeMove", "executeStrike"} {
					if !asked[gate] {
						t.Errorf("stepWorld does not ask the admission for %s", gate)
					}
				}
				continue
			}
			if !ok || fn.Body == nil || !gates[fn.Name.Name] {
				continue
			}
			seen[fn.Name.Name] = true
			admits := false
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					switch sel.Sel.Name {
					case "stoneCursed", "motionActive":
						t.Errorf("%s tests %s beside the admission", fn.Name.Name, sel.Sel.Name)
					case "admission":
						admits = true
					}
				}
				return true
			})
			if views[fn.Name.Name] && !admits {
				t.Errorf("%s does not read the admission", fn.Name.Name)
			}
		}
	}
	for name := range gates {
		if !seen[name] {
			t.Errorf("executor gate %s not found", name)
		}
	}
}
