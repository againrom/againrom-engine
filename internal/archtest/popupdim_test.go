package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// popupDimColourNames are the identifiers a popup's backdrop colour is read
// from. A fill of a rectangle that reads one of them belongs in popupdim.go.
var popupDimColourNames = map[string]bool{
	"AuthoredNoticeBackdrop": true,
	"noticeBackdrop":         true,
	"menuBackdrop":           true,
	"MissionBackdrop":        true,
}

// popupDimFillSites lists the calls in src that fill a rectangle
// (DrawFilledRect, draw.Draw) with an argument naming a popup backdrop colour.
func popupDimFillSites(fset *token.FileSet, f *ast.File) []string {
	var sites []string
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "DrawFilledRect" && sel.Sel.Name != "Draw") {
			return true
		}
		named := false
		for _, arg := range call.Args {
			ast.Inspect(arg, func(m ast.Node) bool {
				switch x := m.(type) {
				case *ast.Ident:
					named = named || popupDimColourNames[x.Name]
				case *ast.SelectorExpr:
					named = named || popupDimColourNames[x.Sel.Name]
				}
				return true
			})
		}
		if named {
			sites = append(sites, fset.Position(call.Pos()).String())
		}
		return true
	})
	return sites
}

func TestNoSecondPopupDimFillInUI(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "pkg", "ui")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "popupdim.go" {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		found = append(found, popupDimFillSites(fset, f)...)
	}
	sort.Strings(found)
	for _, s := range found {
		t.Errorf("pkg/ui fills a rectangle with a popup backdrop colour outside popupdim.go: %s", s)
	}
}

func TestPopupDimFillSitesFindsOnlyABackdropFill(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		bad        bool
	}{
		{"backdrop fill", `vector.DrawFilledRect(dst, 0, 0, 1, 1, AuthoredNoticeBackdrop(), false)`, true},
		{"field fill", `draw.Draw(dst, r, &image.Uniform{C: v.menuBackdrop}, p, draw.Over)`, true},
		{"other fill", `vector.DrawFilledRect(dst, 0, 0, 1, 1, wash, false)`, false},
	} {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "x.go", "package x\nfunc f() {\n"+tc.body+"\n}\n", 0)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(popupDimFillSites(fset, f)) > 0; got != tc.bad {
			t.Errorf("%s: flagged %v, want %v", tc.name, got, tc.bad)
		}
	}
}
