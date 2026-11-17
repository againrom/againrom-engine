package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func headlessRequestFixture(named int) map[string]string {
	var b strings.Builder
	b.WriteString("package ui\nfunc (a *App) HeadlessPicture(req HeadlessRequest) (*image.RGBA, HeadlessExtras, error) { return nil, HeadlessExtras{}, nil }\nfunc ParseWindowSize(value string, empty image.Point) (image.Point, error) { return empty, nil }\n")
	for i := range named {
		fmt.Fprintf(&b, "func (e *MapEditor) Named%cRect() (image.Rectangle, bool) { return image.Rectangle{}, false }\n", rune('A'+i))
	}
	return map[string]string{"pkg/ui/a.go": b.String()}
}

func TestCheckHeadlessRequestNamesEachFault(t *testing.T) {
	if v := CheckHeadlessRequest(headlessRequestFixture(namedControlAccessorsRemaining)); len(v) != 0 {
		t.Fatalf("clean fixture: %v", v)
	}
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"second picture", func() map[string]string {
			f := headlessRequestFixture(namedControlAccessorsRemaining)
			f["pkg/ui/a.go"] += "func (a *App) HeadlessExtraFrame() (*image.RGBA, error) { return nil, nil }\n"
			return f
		}(), "HeadlessPicture only"},
		{"second parser", func() map[string]string {
			f := headlessRequestFixture(namedControlAccessorsRemaining)
			f["cmd/x/main.go"] = "package main\nvar _, _ = fmt.Sscanf(v, \"%dx%d\", &a, &b)\n"
			return f
		}(), "window size parser"},
		{"second split", func() map[string]string {
			f := headlessRequestFixture(namedControlAccessorsRemaining)
			f["pkg/editorapp/run.go"] = "package editorapp\nvar parts = strings.Split(*size, \"x\")\n"
			return f
		}(), "window size parser"},
		{"rose", headlessRequestFixture(namedControlAccessorsRemaining + 1), "rose to"},
		{"fell", headlessRequestFixture(namedControlAccessorsRemaining - 1), "fell to"},
	}
	for _, tc := range cases {
		found := false
		for _, v := range CheckHeadlessRequest(tc.files) {
			found = found || strings.Contains(v.Reason, tc.want)
		}
		if !found {
			t.Errorf("%s: no violation naming %q", tc.name, tc.want)
		}
	}
}

// TestHeadlessRequestHasOneBuilder is the live-tree half.
func TestHeadlessRequestHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadHeadlessRequestSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckHeadlessRequest(files) {
		t.Error(v)
	}
}

func TestHeadlessControlCentresUseOneFinder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		file  string
		names []string
	}{
		{"headless.go", []string{"HeadlessActivate"}},
		{"headlesscommand.go", []string{"HeadlessCommandPoint"}},
		{"headlesspointer.go", []string{"HeadlessWornCellPoint", "HeadlessDollBoxPoint", "HeadlessSpellPoint", "HeadlessShopPoint",
			"HeadlessWorldMapMissionPoint", "HeadlessWorldMapTownPoint", "HeadlessDocumentPoint", "HeadlessHelpBarPoint", "HeadlessTipClosePoint"}},
	} {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "pkg/ui", c.file), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range c.names {
			found := false
			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name.Name != name {
					continue
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "headlessControlPoint" {
						found = true
					}
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "FrameToWindow" || sel.Sel.Name == "frameToWindow" || sel.Sel.Name == "nativeFrameToWindow") {
						if name != "HeadlessShopPoint" {
							t.Errorf("%s.%s converts a control point beside headlessControlPoint", c.file, name)
						}
					}
					return true
				})
			}
			if !found {
				t.Errorf("%s.%s must use headlessControlPoint", c.file, name)
			}
		}
	}
}
