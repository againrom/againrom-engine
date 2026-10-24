package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

const textModelFile = "pkg/ui/editortextmodel.go"

var textInputRoutes = map[string]string{
	"pkg/ui/chargen.go:Chargen.EditName":              "EditText",
	"pkg/ui/save_dialog.go:flow.editSaveText":         "EditText",
	"pkg/ui/savepage_save.go:savePageState.edit":      "EditText",
	"pkg/ui/goldmodal.go:goldState.edit":              "EditText",
	"pkg/ui/goldmodal.go:goldState.move":              "edit",
	"pkg/ui/goldmodal.go:goldState.backspace":         "edit",
	"pkg/ui/goldmodal.go:goldState.deleteForward":     "edit",
	"pkg/ui/goldmodal.go:goldState.insert":            "edit",
	"pkg/ui/goldmodal.go:goldState.clampEdit":         "edit",
	"pkg/ui/cheatinput.go:App.stepCheatChat":          "EditText",
	"cmd/starter/app.go:app.typed":                    "EditText",
	"cmd/starter/app.go:app.backspace":                "EditText",
	"pkg/ui/mapeditormove.go:MapEditor.TextInput":     "insertText",
	"pkg/ui/editorinput.go:MapEditor.insertText":      "typeText",
	"pkg/ui/mapeditorworkspace.go:MapEditor.keyClear": "replace",
}

// CheckTextInputRoutes pins the known field adapters to the one editing model.
func CheckTextInputRoutes(files map[string]string) []Violation {
	var vs []Violation
	seen := map[string]bool{}
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/ui/") && !strings.HasPrefix(name, "cmd/starter/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: err.Error()})
			continue
		}
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			receiver := ""
			if fn.Recv != nil {
				t := fn.Recv.List[0].Type
				if star, ok := t.(*ast.StarExpr); ok {
					t = star.X
				}
				if id, ok := t.(*ast.Ident); ok {
					receiver = id.Name + "."
				}
			}
			if name != textModelFile && (receiver == "TextInputState." || fn.Name.Name == "EditText") {
				vs = append(vs, Violation{From: name, Reason: "text model implementation outside " + textModelFile})
			}
			key := name + ":" + receiver + fn.Name.Name
			want := textInputRoutes[key]
			if want == "" {
				continue
			}
			seen[key] = true
			delegates, inline := false, false
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.SliceExpr:
					inline = true
				case *ast.CallExpr:
					callee := ""
					switch f := node.Fun.(type) {
					case *ast.Ident:
						callee = f.Name
					case *ast.SelectorExpr:
						callee = f.Sel.Name
					}
					delegates = delegates || callee == want
					inline = inline || callee == "append" || callee == "copy"
				}
				return true
			})
			if !delegates || inline {
				vs = append(vs, Violation{From: key, Reason: "text field must delegate to " + want + " without inline editing"})
			}
		}
	}
	for key := range textInputRoutes {
		if !seen[key] {
			vs = append(vs, Violation{From: key, Reason: "text input route missing"})
		}
	}
	if _, ok := files[textModelFile]; !ok {
		vs = append(vs, Violation{From: textModelFile, Reason: "text model missing"})
	}
	sort.Slice(vs, func(i, j int) bool { return vs[i].From < vs[j].From })
	return vs
}
