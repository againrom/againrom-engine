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
					if fn.Name.Name != "drawGameMenuOverMap" {
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

const menuPlacementFile = "pkg/ui/missionmenu.go"

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
