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
	"testing"
)

// listPageFiles hold the list page and its role rows (listRole).
var listPageFiles = []string{"pkg/ui/save.go", "pkg/ui/savepage.go", "pkg/ui/load_window.go", "pkg/ui/load_draw.go",
	"pkg/ui/save_dialog.go", "pkg/ui/mapchoice.go", "pkg/ui/listrole.go"}

// listPageBodyFiles hold the list page's one step, compose and draw bodies:
// none may read the open save screen, which only the save role's own rows
// reach.
var listPageBodyFiles = map[string]bool{"pkg/ui/save.go": true, "pkg/ui/savepage.go": true, "pkg/ui/load_window.go": true,
	"pkg/ui/load_draw.go": true, "pkg/ui/save_dialog.go": true, "pkg/ui/mapchoice.go": true}

// profileListingFiles hold the saves' and the characters' rows of the one
// profile file lister, listProfileFiles in pkg/game/profilefiles.go.
var profileListingFiles = []string{"pkg/game/savescan.go", "pkg/game/characterstore.go"}

// listPageRoleBranches are the places in files where the page could tell its
// roles apart: a field named after a role, a read of a row's roleName outside
// CharacterList, and a listing file that reads a folder itself.
func listPageRoleBranches(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		at := func(n ast.Node, what string) {
			out = append(out, name+":"+strconv.Itoa(fset.Position(n.Pos()).Line)+":"+what)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if field, ok := n.(*ast.Field); ok {
				for _, id := range field.Names {
					if id.Name == "characters" || id.Name == "maps" {
						at(id, "field "+id.Name)
					}
				}
			}
			if be, ok := n.(*ast.BinaryExpr); ok && (be.Op == token.EQL || be.Op == token.NEQ) {
				if sel, ok := be.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "save" {
					if id, ok := be.Y.(*ast.Ident); ok && id.Name == "nil" {
						at(sel, "save state tested")
					}
				}
			}
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "saveScreen" && listPageBodyFiles[name] {
				at(sel, "save screen read")
			}
			if fd, ok := n.(*ast.FuncDecl); ok && fd.Body != nil && fd.Name.Name != "CharacterList" {
				ast.Inspect(fd.Body, func(m ast.Node) bool {
					if sel, ok := m.(*ast.SelectorExpr); ok && sel.Sel.Name == "roleName" {
						at(sel, "roleName read in "+fd.Name.Name)
					}
					return true
				})
			}
			return true
		})
		if strings.HasPrefix(name, "pkg/game/") && strings.Contains(source, "os.ReadDir(") {
			out = append(out, name+": os.ReadDir")
		}
	}
	sort.Strings(out)
	return out
}

func TestListPageRoleScanNamesEachBranch(t *testing.T) {
	got := listPageRoleBranches(t, map[string]string{
		"pkg/ui/a.go": "package ui\ntype w struct {\n\tcharacters bool\n}\n" +
			"func (a *App) CharacterList() bool { return r.roleName == \"x\" }\n" +
			"func (f *flow) loadPage() { if r.roleName == \"x\" {} }\n" +
			"func (f *flow) cancel(v pageView) { if v.save != nil {} }\n",
		"pkg/game/b.go":         "package game\nfunc l() { os.ReadDir(\"d\") }\n",
		"pkg/ui/load_window.go": "package ui\nfunc (f *flow) step() { _ = f.saveScreen }\n",
		"pkg/ui/listrole.go":    "package ui\nfunc (f *flow) rows() { _ = f.saveScreen }\n",
	})
	want := []string{"pkg/game/b.go: os.ReadDir", "pkg/ui/a.go:3:field characters", "pkg/ui/a.go:6:roleName read in loadPage", "pkg/ui/a.go:7:save state tested",
		"pkg/ui/load_window.go:2:save screen read"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("listPageRoleBranches = %q, want %q", got, want)
	}
}

// TestListPageRolesAreRows holds the list page to its role rows and the save
// and character lists to the one profile file lister.
func TestListPageRolesAreRows(t *testing.T) {
	root := moduleRootForTest(t)
	files := map[string]string{}
	for _, name := range append(append([]string(nil), listPageFiles...), profileListingFiles...) {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("%s: %v; the scan guards nothing", name, err)
		}
		files[name] = string(b)
	}
	for _, s := range listPageRoleBranches(t, files) {
		t.Errorf("the list page tells a role apart outside its row: %s", s)
	}
}

// TestSaveScreenHasNoBodyOfItsOwn states that the save screen owns no step,
// compose or draw function: both list screens are built by listPageHandler.
func TestSaveScreenHasNoBodyOfItsOwn(t *testing.T) {
	root := moduleRootForTest(t)
	banned := map[string]bool{"stepSaveScreen": true, "composeSaveScreen": true, "drawSaveScreen": true, "savePagePaint": true, "composeLoadScreen": true}
	entries, err := os.ReadDir(filepath.Join(root, "pkg", "ui"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(root, "pkg", "ui", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(token.NewFileSet(), e.Name(), src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && banned[fd.Name.Name] {
				t.Errorf("%s declares %s: the save screen has a body of its own", e.Name(), fd.Name.Name)
			}
		}
	}
	b, err := os.ReadFile(filepath.Join(root, "pkg", "ui", "screenhandler.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"screenHandlers[ScreenLoad] = listPageHandler(", "screenHandlers[ScreenSave] = listPageHandler("} {
		if !strings.Contains(string(b), s) {
			t.Errorf("screenhandler.go lacks %q", s)
		}
	}
}
