package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"testing"
)

// displayScaleFile holds the display presenter; the arithmetic lives in
// widgetkit.
const (
	displayScaleFile = "pkg/ui/devicescale.go"
	scaleLeafFile    = "pkg/ui/widgetkit/devicescale.go"
	starterWindow    = "cmd/starter/window.go"
)

// displayScaleHomes names, for each part of the device-scale arithmetic, the
// declarations (file:declaration) that may read it.
var displayScaleHomes = map[string][]string{
	"WindowPixels":       {displayScaleFile + ":displayRow.layout", starterWindow + ":window.LayoutF"},
	"MonitorDeviceScale": {displayScaleFile + ":displayRow.layout", starterWindow + ":window.LayoutF"},
	"PointOfDevice":      {displayScaleFile + ":displayRow.toLayout", starterWindow + ":window.cursor"},
	"DeviceScaleFactor":  {scaleLeafFile + ":MonitorDeviceScale"},
}

func displayScaleHome(name, file, decl string) bool {
	for _, h := range displayScaleHomes[name] {
		if h == file+":"+decl {
			return true
		}
	}
	return false
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
				if _, ok := displayScaleHomes[id.Name]; ok && !displayScaleHome(id.Name, name, home) {
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
			"func (p displayRow) layout() { widgetkit.WindowPixels(1, 1, widgetkit.MonitorDeviceScale()) }\n" +
			"func (a *App) LayoutF(w, h float64) { widgetkit.WindowPixels(w, h, widgetkit.MonitorDeviceScale()) }\n",
		"pkg/ui/x.go": "package ui\nfunc f() { _ = ebiten.Monitor().DeviceScaleFactor() }\n",
		starterWindow: "package main\nfunc (w *window) Layout() { widgetkit.WindowPixels(1, 1, 1) }\n",
	})
	want := []string{
		"cmd/starter/window.go:window.Layout:WindowPixels",
		displayScaleFile + ":App.LayoutF:MonitorDeviceScale",
		displayScaleFile + ":App.LayoutF:WindowPixels",
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
