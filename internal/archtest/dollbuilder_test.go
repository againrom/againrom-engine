package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestDollLayersHaveOneBuilder(t *testing.T) {
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
	count := 0
	for name, source := range files {
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		if f.Name.Name != "game" {
			continue
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "FigureDrawSteps" {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != "data" {
					return true
				}
				count++
				if name != "pkg/game/inventory.go" || fn.Name.Name != "composeDoll" {
					t.Errorf("%s:%s builds doll layers outside composeDoll", name, fn.Name.Name)
				}
				return true
			})
		}
	}
	if count != 2 {
		t.Fatalf("FigureDrawSteps references = %d, want the source and nil-source arms", count)
	}
}

func TestEditorDraftPortraitUsesSpeakerBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ file, function, builder, forbidden string }{
		{"pkg/game/mapinspectionunitsettings.go", "UnitSettings", "SpeakerFace", "composeUnitFigure"},
		{"pkg/ui/mapeditorwindow.go", "drawObjectPreview", "drawNoticePortrait", "blitNoticeWindow"},
	} {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, tc.file), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		calls := 0
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != tc.function {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := ""
				switch x := call.Fun.(type) {
				case *ast.Ident:
					name = x.Name
				case *ast.SelectorExpr:
					name = x.Sel.Name
				}
				if name == tc.builder {
					calls++
				}
				if name == tc.forbidden {
					t.Errorf("%s bypasses %s through %s", tc.function, tc.builder, name)
				}
				return true
			})
		}
		if calls != 1 {
			t.Errorf("%s calls %s %d times, want one", tc.function, tc.builder, calls)
		}
	}
}
