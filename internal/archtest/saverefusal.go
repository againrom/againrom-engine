package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

var saveRefusalRules = map[string]bool{"townSavePoint": true, "exportable": true, "refuseCompleted": true}

// checkSaveRefusal reports an error a save point rule builds and a refusal
// row outside pkg/game/saverefusal.go.
func checkSaveRefusal(files map[string]string) (rules int, vs []string) {
	for name, src := range files {
		if !strings.HasPrefix(name, "pkg/game/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, name+": "+err.Error())
			continue
		}
		for _, d := range f.Decls {
			fn, _ := d.(*ast.FuncDecl)
			rule := fn != nil && fn.Recv != nil && saveRefusalRules[fn.Name.Name]
			if rule {
				rules++
			}
			ast.Inspect(d, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CallExpr:
					sel, ok := x.Fun.(*ast.SelectorExpr)
					if !ok || !rule {
						return true
					}
					if pkg, ok := sel.X.(*ast.Ident); ok && (pkg.Name == "errors" && sel.Sel.Name == "New" || pkg.Name == "fmt" && sel.Sel.Name == "Errorf") {
						vs = append(vs, name+": "+fn.Name.Name+" builds an error; a closed save point is a row of its game's refusal table")
					}
				case *ast.CompositeLit:
					if id, ok := x.Type.(*ast.Ident); ok && id.Name == "saveRefusal" && name != "pkg/game/saverefusal.go" {
						vs = append(vs, name+": writes a saveRefusal row outside the refusal tables in pkg/game/saverefusal.go")
					}
				}
				return true
			})
		}
	}
	return rules, vs
}

const savePointFile = "pkg/ui/savepoint.go"

// savePointCalls are the gate's helpers and savePointPredicates the screen
// tests its rows make.
var (
	savePointCalls      = map[string]bool{"townCanSave": true, "atGameSavePoint": true, "showSaveRefusal": true}
	savePointPredicates = map[string]bool{"popupOpen": true, "townDialogue": true, "missionShopOpen": true,
		"NoticeOpen": true, "currentDialoguePointerOwner": true}
)

// checkSavePoint reports, in pkg/ui outside savepoint.go: a call of a save
// point predicate or of the refusal display, a CanSave assertion, an
// admission request, a write of the refusal line, and a function (a
// declaration or a literal, each on its own) that reads a save input and
// tests a gate predicate. It also counts the savePoint calls it read.
func checkSavePoint(files map[string]string) (asks int, vs []string) {
	for name, src := range files {
		if !strings.HasPrefix(name, "pkg/ui/") || strings.Contains(name[len("pkg/ui/"):], "/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, name+": "+err.Error())
			continue
		}
		gate := name == savePointFile
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				callee := calleeName(x)
				if callee == "savePoint" && !gate {
					asks++
				}
				if savePointCalls[callee] && !gate {
					vs = append(vs, at(x)+": calls "+callee+"; a save point is asked through savePoint in "+savePointFile)
				}
			case *ast.TypeAssertExpr:
				if it, ok := x.Type.(*ast.InterfaceType); ok && !gate && declaresMethod(it, "CanSave") {
					vs = append(vs, at(x)+": asserts CanSave; whether a town can save is a row of savePoint")
				}
			case *ast.CompositeLit:
				if id, ok := x.Type.(*ast.Ident); ok && id.Name == "SaveRequest" && !gate && setsField(x, "Admission") {
					vs = append(vs, at(x)+": asks the save admission; a refusal is shown by savePoint's asking rows")
				}
			case *ast.AssignStmt:
				for _, l := range x.Lhs {
					if sel, ok := l.(*ast.SelectorExpr); ok && sel.Sel.Name == "saveRefused" && !gate {
						vs = append(vs, at(x)+": writes the refusal line; only savePoint's asking rows show a refusal")
					}
				}
			case *ast.FuncDecl:
				if !gate && x.Body != nil {
					if p := savePointDecision(x.Body); p != "" {
						vs = append(vs, at(x)+": "+x.Name.Name+" reads a save input and tests "+p+"; ask savePoint")
					}
				}
			case *ast.FuncLit:
				if !gate {
					if p := savePointDecision(x.Body); p != "" {
						vs = append(vs, at(x)+": a function literal reads a save input and tests "+p+"; ask savePoint")
					}
				}
			}
			return true
		})
	}
	return asks, vs
}

// savePointDecision names the gate predicate a function body tests when the
// same body, without its nested literals, also reads a save input.
func savePointDecision(body *ast.BlockStmt) string {
	reads, tested := false, ""
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.SelectorExpr:
			switch x.Sel.Name {
			case "SaveGame", "QuickSave":
				reads = true
			case "Poll":
				if s, ok := x.X.(*ast.SelectorExpr); ok && s.Sel.Name == "Autosave" {
					reads = true
				}
			}
		case *ast.Ident:
			if x.Name == "gameMenuSave" {
				reads = true
			}
		case *ast.CallExpr:
			if c := calleeName(x); savePointPredicates[c] && tested == "" {
				tested = c
			}
		}
		return true
	})
	if !reads {
		return ""
	}
	return tested
}

func declaresMethod(it *ast.InterfaceType, method string) bool {
	if it.Methods == nil {
		return false
	}
	for _, m := range it.Methods.List {
		for _, id := range m.Names {
			if id.Name == method {
				return true
			}
		}
	}
	return false
}

func setsField(lit *ast.CompositeLit, field string) bool {
	for _, e := range lit.Elts {
		if kv, ok := e.(*ast.KeyValueExpr); ok {
			if id, ok := kv.Key.(*ast.Ident); ok && id.Name == field {
				return true
			}
		}
	}
	return false
}
