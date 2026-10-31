package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestMissionMenuLayersUseOneHost(t *testing.T) {
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
	calls := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				name, ok := call.Fun.(*ast.Ident)
				if ok && name.Name == "blitMenuOverMap" {
					calls++
					if fn.Name.Name != "drawMenuLayerOverMap" {
						t.Errorf("%s adds a mission-menu host in %s", fn.Name.Name, entry.Name())
					}
				}
				return true
			})
		}
	}
	if calls != 1 {
		t.Fatalf("mission-menu blit calls %d, want one shared host", calls)
	}
}

func menuPlacementReaders(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, source := range files {
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok && (sel.Sel.Name == "NativeMissionWindows" || sel.Sel.Name == "nativeMissionMenu") {
					out = append(out, name+":"+fn.Name.Name+":"+sel.Sel.Name)
				}
				return true
			})
		}
	}
	sort.Strings(out)
	return out
}

func TestMenuPlacementScanNamesEachReader(t *testing.T) {
	got := menuPlacementReaders(t, map[string]string{
		menuPlacementFile: "package ui\nfunc (p P) placement() bool { return p.NativeMissionWindows }\n",
		"pkg/ui/x.go":     "package ui\nfunc f(a A) bool { return a.flow.nativeMissionMenu() || a.p.NativeMissionWindows }\n",
	})
	want := []string{"pkg/ui/missionmenu.go:placement:NativeMissionWindows", "pkg/ui/x.go:f:NativeMissionWindows", "pkg/ui/x.go:f:nativeMissionMenu"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("menuPlacementReaders = %v", got)
	}
}

func TestNativeMissionWindowsIsReadByThePlacementBuilderOnly(t *testing.T) {
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
	if _, ok := files[menuPlacementFile]; !ok {
		t.Fatalf("%s is missing; the scan guards nothing", menuPlacementFile)
	}
	got := menuPlacementReaders(t, files)
	if want := menuPlacementFile + ":placement:NativeMissionWindows"; len(got) != 1 || got[0] != want {
		t.Fatalf("NativeMissionWindows readers %v, want only %s", got, want)
	}
}

func TestMenuPlacementIsAskedThroughItsOperations(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	m, err := loadTypeCheckedModule(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range CheckMenuPlacement(m) {
		t.Error(f)
	}
}

// placementFixture is a pkg/ui in small: the placement, its host row and one
// operation type in missionmenu.go, one planted shape per line of x.go, and a
// map editor type with its own anchor and centred fields.
var placementFixture = map[string]string{
	menuPlacementFile: `package ui

type menuWindowAnchor int

const menuAnchorMissionView menuWindowAnchor = 1

type menuUnderlay int

const mapUnderlay menuUnderlay = 1

func (u menuUnderlay) hold()      {}
func (u menuUnderlay) held() bool { return u == mapUnderlay }

type menuWindowHost struct {
	centred  bool
	underlay menuUnderlay
}

var missionViewHost = menuWindowHost{centred: true, underlay: mapUnderlay}

type menuWindowPlacement struct {
	menuWindowHost
	anchor menuWindowAnchor
}

func (pl menuWindowPlacement) overMission() bool { return pl.anchor == menuAnchorMissionView }

type GameMenuProfile struct{ NativeMissionWindows bool }

type flow struct{ p GameMenuProfile }

func (f *flow) menuPlacement() menuWindowPlacement { return menuWindowPlacement{menuWindowHost: missionViewHost} }
func (f *flow) menuWindowShown() bool             { return f.p.NativeMissionWindows }
`,
	"pkg/ui/x.go": `package ui

func boolMethod(f *flow) bool { return f.menuPlacement().overMission() }
func opBool(f *flow) bool { return f.menuPlacement().underlay.held() }
func anchorRead(f *flow) bool { pl := f.menuPlacement(); return pl.anchor != 0 }
func centredRead(f *flow) bool { return f.menuPlacement().centred }
func constRef() menuWindowAnchor { return menuAnchorMissionView }
func opValue(f *flow) menuUnderlay { return f.menuPlacement().underlay }
func hostField(f *flow) menuWindowHost { return f.menuPlacement().menuWindowHost }
func hostTable() menuWindowHost { return missionViewHost }
func native(f *flow) bool { return f.p.NativeMissionWindows }
func shown(f *flow) bool { return f.menuWindowShown() }
func ifCond(f *flow, g func(menuWindowPlacement) bool) { if g(f.menuPlacement()) { f.menuPlacement().underlay.hold() } }
func ifInit(f *flow) { if pl := f.menuPlacement(); pl.menuWindowHost == (menuWindowHost{}) { } }
func fine(f *flow) { f.menuPlacement().underlay.hold() }
`,
	"pkg/ui/editor.go": `package ui

type caret struct {
	anchor  int
	centred bool
}

func editorAnchor(c caret) int   { return c.anchor }
func editorCentred(c caret) bool { return c.centred }
`,
}

func TestMenuPlacementScanNamesEachShape(t *testing.T) {
	fset := token.NewFileSet()
	cp := &CheckedPackage{ImportPath: uiPackage, Info: &types.Info{Types: map[ast.Expr]types.TypeAndValue{},
		Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}}
	for _, name := range []string{menuPlacementFile, "pkg/ui/x.go", "pkg/ui/editor.go"} {
		f, err := parser.ParseFile(fset, name, placementFixture[name], 0)
		if err != nil {
			t.Fatal(err)
		}
		cp.Files, cp.RelPaths = append(cp.Files, f), append(cp.RelPaths, name)
	}
	cp.NumProd = len(cp.Files)
	if _, err := (&types.Config{}).Check(uiPackage, fset, cp.Files, cp.Info); err != nil {
		t.Fatal(err)
	}
	got := CheckMenuPlacement(&TypeCheckedModule{Fset: fset, Packages: map[string]*CheckedPackage{uiPackage: cp}})
	want := map[string]string{
		"3":  "calls menuWindowPlacement.overMission",
		"4":  "calls menuUnderlay.held",
		"5":  "reads the placement's anchor",
		"6":  "reads the placement's centred",
		"7":  "names the placement constant menuAnchorMissionView",
		"8":  "uses the placement's underlay other than to call its operation",
		"9":  "reads the placement's host row",
		"10": "names the host row missionViewHost",
		"11": "reads NativeMissionWindows",
		"12": "calls menuWindowShown",
		"13": "an if statement asks a menu placement",
		"14": "an if statement asks a menu placement|reads the placement's host row",
	}
	seen := map[string][]string{}
	for _, f := range got {
		rest, ok := strings.CutPrefix(f, "pkg/ui/x.go:")
		if !ok {
			t.Errorf("finding outside the planted file: %s", f)
			continue
		}
		line, msg, _ := strings.Cut(rest, ": ")
		seen[line] = append(seen[line], msg)
	}
	for line, shapes := range want {
		for _, shape := range strings.Split(shapes, "|") {
			found := false
			for _, msg := range seen[line] {
				found = found || strings.HasPrefix(msg, shape)
			}
			if !found {
				t.Errorf("x.go:%s: no finding %q in %v", line, shape, seen[line])
			}
		}
	}
	for line, msgs := range seen {
		if _, ok := want[line]; !ok {
			t.Errorf("x.go:%s: unplanted findings %v", line, msgs)
		}
	}
}
