package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// CheckSaveScreen forbids the retired screen name and a second live save
// screen holder in flow or its page data, including aliases and containers.
func CheckSaveScreen(files map[string]string) []Violation {
	types := map[string]ast.Expr{}
	positions := map[ast.Node]string{}
	var violations []Violation
	var names []string
	for name := range files {
		if strings.HasPrefix(name, "pkg/ui/") && !strings.Contains(strings.TrimPrefix(name, "pkg/ui/"), "/") && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return []Violation{{From: "pkg/ui", Reason: "no save screen sources scanned"}}
	}
	for _, name := range names {
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			violations = append(violations, Violation{From: name, Reason: err.Error()})
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil {
				return false
			}
			positions[n] = name + ":" + strconv.Itoa(set.Position(n.Pos()).Line)
			if id, ok := n.(*ast.Ident); ok && id.Name == "saveDialog" {
				violations = append(violations, Violation{From: positions[n], Reason: "saveDialog is a retired save screen identifier"})
			}
			if typ, ok := n.(*ast.TypeSpec); ok {
				types[typ.Name.Name] = typ.Type
			}
			return true
		})
	}
	var holders func(ast.Expr, string, map[string]bool)
	holders = func(expr ast.Expr, path string, seen map[string]bool) {
		switch typ := expr.(type) {
		case *ast.Ident:
			if seen[typ.Name] {
				return
			}
			seen[typ.Name] = true
			defer delete(seen, typ.Name)
			body := types[typ.Name]
			if typ.Name == "savePageState" || saveScreenStateShape(body) {
				if path != "flow.saveScreen" {
					violations = append(violations, Violation{From: positions[expr], Reason: path + " is a second save screen holder; flow.saveScreen owns the open screen"})
				}
				return
			}
			holders(body, path, seen)
		case *ast.StarExpr:
			holders(typ.X, path, seen)
		case *ast.ArrayType:
			holders(typ.Elt, path, seen)
		case *ast.MapType:
			holders(typ.Key, path, seen)
			holders(typ.Value, path, seen)
		case *ast.ChanType:
			holders(typ.Value, path, seen)
		case *ast.ParenExpr:
			holders(typ.X, path, seen)
		case *ast.StructType:
			for _, field := range typ.Fields.List {
				if len(field.Names) == 0 {
					holders(field.Type, path+".embedded", seen)
				}
				for _, id := range field.Names {
					holders(field.Type, path+"."+id.Name, seen)
				}
			}
		}
	}
	for _, name := range []string{"flow", "pageSeams"} {
		holders(types[name], name, map[string]bool{})
	}
	return violations
}

func saveScreenStateShape(expr ast.Expr) bool {
	st, ok := expr.(*ast.StructType)
	if !ok {
		return false
	}
	request, page, list, window := false, false, false, false
	for _, field := range st.Fields.List {
		typ := field.Type
		if ptr, ok := typ.(*ast.StarExpr); ok {
			typ = ptr.X
		}
		if id, ok := typ.(*ast.Ident); ok {
			request = request || id.Name == "SaveRequest"
			page = page || id.Name == "SavePage"
			list = list || id.Name == "Picker"
			window = window || id.Name == "loadWindow"
		}
	}
	return list && (request || page && window)
}
