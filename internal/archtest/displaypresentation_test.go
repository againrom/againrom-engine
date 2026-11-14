package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"testing"
)

// displayScaleFile holds the one display presenter.
const displayScaleFile = "pkg/ui/devicescale.go"

// displayScaleHomes names, for each part of the device-scale arithmetic, the
// one declaration in displayScaleFile that may read it.
var displayScaleHomes = map[string]string{
	"windowPixels":       "displayRow.layout",
	"monitorDeviceScale": "displayRow.layout",
	"pointOfDevice":      "displayRow.toLayout",
	"DeviceScaleFactor":  "monitorDeviceScale",
}

// displayScaleReaders are the reads of a displayScaleHomes name outside its
// home, as file:declaration:name.
func displayScaleReaders(t *testing.T, files map[string]string) []string {
	t.Helper()
	var out []string
	for name, src := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, d := range f.Decls {
			home, own := declHome(d)
			ast.Inspect(d, func(n ast.Node) bool {
				id, ok := n.(*ast.Ident)
				if !ok || own[id] {
					return true
				}
				if want, ok := displayScaleHomes[id.Name]; ok && (name != displayScaleFile || home != want) {
					out = append(out, name+":"+home+":"+id.Name)
				}
				return true
			})
		}
	}
	sort.Strings(out)
	return out
}

// declHome is a declaration's name (Type.Method for a method) and the
// identifiers that declare it.
func declHome(d ast.Decl) (string, map[*ast.Ident]bool) {
	own := map[*ast.Ident]bool{}
	switch d := d.(type) {
	case *ast.FuncDecl:
		own[d.Name] = true
		if d.Recv != nil && len(d.Recv.List) == 1 {
			t := d.Recv.List[0].Type
			if s, ok := t.(*ast.StarExpr); ok {
				t = s.X
			}
			if id, ok := t.(*ast.Ident); ok {
				return id.Name + "." + d.Name.Name, own
			}
		}
		return d.Name.Name, own
	case *ast.GenDecl:
		home := ""
		for _, s := range d.Specs {
			if v, ok := s.(*ast.ValueSpec); ok {
				for _, n := range v.Names {
					own[n] = true
					if home == "" {
						home = n.Name
					}
				}
			}
		}
		return home, own
	}
	return "", own
}

// TestDisplayScaleReaderNamesASecondRoute shows the scan names a LayoutF that
// does its own scale arithmetic beside the presenter.
func TestDisplayScaleReaderNamesASecondRoute(t *testing.T) {
	got := displayScaleReaders(t, map[string]string{
		displayScaleFile: "package ui\n" +
			"var monitorDeviceScale = func() float64 { return ebiten.Monitor().DeviceScaleFactor() }\n" +
			"func windowPixels(w, h, s float64) {}\n" +
			"func (p displayRow) layout() { windowPixels(1, 1, monitorDeviceScale()) }\n" +
			"func (a *App) LayoutF(w, h float64) { windowPixels(w, h, monitorDeviceScale()) }\n",
		"pkg/ui/x.go": "package ui\nfunc f() { _ = ebiten.Monitor().DeviceScaleFactor() }\n",
	})
	want := []string{
		displayScaleFile + ":App.LayoutF:monitorDeviceScale",
		displayScaleFile + ":App.LayoutF:windowPixels",
		"pkg/ui/x.go:f:DeviceScaleFactor",
	}
	if len(got) != len(want) {
		t.Fatalf("readers %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("readers %v, want %v", got, want)
		}
	}
}

// TestDisplayScaleArithmeticHasOneHome holds the device-scale arithmetic to
// the presenter in pkg/ui/devicescale.go: each LayoutF, final stage and
// pointer mapping goes through its row.
func TestDisplayScaleArithmeticHasOneHome(t *testing.T) {
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
	if _, ok := files[displayScaleFile]; !ok {
		t.Fatalf("%s is missing; the scan guards nothing", displayScaleFile)
	}
	for _, r := range displayScaleReaders(t, files) {
		t.Errorf("device-scale arithmetic outside its presenter home: %s", r)
	}
}
