package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// editorElementSources is every production file of pkg/ui, parsed, by base
// name.
func editorElementSources(t *testing.T) map[string]*ast.File {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	names, err := filepath.Glob(filepath.Join(root, "pkg", "ui", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*ast.File{}
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		out[filepath.Base(name)] = f
	}
	if len(out) == 0 {
		t.Fatal("no pkg/ui sources")
	}
	return out
}

// The editor's drop-down, tip, bar and status or legend each have one
// builder; a second one returns as a name, a call or a literal elsewhere.
func TestEditorFrameworkElementsHaveOneBuilder(t *testing.T) {
	files := editorElementSources(t)
	dropdownType := regexp.MustCompile(`(?i)(dropdown|combo)`)
	hitParts := map[string]bool{"dropdownMiss": true, "dropdownToggle": true, "dropdownRow": true, "dropdownBar": true}
	bars := map[string]bool{"topBarLayout": true, "toolbarBar": true, "searchLayout": true}
	for name, f := range files {
		editor := strings.HasPrefix(name, "mapeditor") || strings.HasPrefix(name, "editor")
		for _, decl := range f.Decls {
			if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.TYPE {
				for _, spec := range g.Specs {
					ts := spec.(*ast.TypeSpec)
					if dropdownType.MatchString(ts.Name.Name) && name != "widgetdropdown.go" {
						t.Errorf("%s declares drop-down type %s; extend dropdown in widgetdropdown.go", name, ts.Name.Name)
					}
				}
			}
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.Ident:
					if hitParts[n.Name] && name != "widgetdropdown.go" {
						t.Errorf("%s.%s reads drop-down part %s; press it through dropdown.press", name, fn.Name.Name, n.Name)
					}
				case *ast.CompositeLit:
					switch id, _ := n.Type.(*ast.Ident); {
					case id == nil:
					case id.Name == "popupBar" && !bars[fn.Name.Name]:
						t.Errorf("%s.%s builds a bar; the menu bar, toolbar and search field are the editor's bars", name, fn.Name.Name)
					case id.Name == "popupTip" && len(n.Elts) > 0 && name != "editortip.go" && name != "widgethover.go":
						t.Errorf("%s.%s builds a tip; lay it out with layoutTip", name, fn.Name.Name)
					}
				case *ast.CallExpr:
					callee := ""
					switch c := n.Fun.(type) {
					case *ast.Ident:
						callee = c.Name
					case *ast.SelectorExpr:
						callee = c.Sel.Name
					}
					switch {
					case callee == "drawIconButton" && name != "editorpopupbar.go" && !strings.HasPrefix(name, "widget"):
						t.Errorf("%s.%s paints an icon button; put it in a bar", name, fn.Name.Name)
					case callee == "recordWidget" && len(n.Args) > 0 && exprName(n.Args[0]) == "widgetTip" && name != "widgethover.go":
						t.Errorf("%s.%s paints a tip; paint it with drawPopupTip", name, fn.Name.Name)
					case callee == "setDelay" && editor && (len(n.Args) != 1 || exprName(n.Args[0]) != "editorTipDelay"):
						t.Errorf("%s.%s sets a tip delay other than editorTipDelay", name, fn.Name.Name)
					case callee == "statusStripSpec" && fn.Name.Name != "statusSpec" && name != "popupchrome.go":
						t.Errorf("%s.%s builds a status line; read statusSpec", name, fn.Name.Name)
					case callee == "playerLegendSpec" && fn.Name.Name != "legendSpec" && name != "popupchrome.go":
						t.Errorf("%s.%s builds a player legend; read legendSpec", name, fn.Name.Name)
					}
				}
				return true
			})
		}
	}
	delay := false
	for _, decl := range files["mapeditorhover.go"].Decls {
		if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.CONST {
			for _, spec := range g.Specs {
				v := spec.(*ast.ValueSpec)
				if len(v.Names) == 1 && v.Names[0].Name == "editorTipDelay" && len(v.Values) == 1 {
					lit, ok := v.Values[0].(*ast.BasicLit)
					delay = ok && lit.Value == "300"
				}
			}
		}
	}
	if !delay {
		t.Error("editorTipDelay must stay 300 ms")
	}
}
