package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"
)

func TestRectangleBrushHasOneCoverAndRecordRoute(t *testing.T) {
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
	covers, routes, policies := 0, 0, 0
	for name, source := range files {
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Body != nil {
				brush := name == "pkg/mapedit/rectanglebrush.go" || name == "pkg/mapedit/brush.go" || rectangleBrushFunction(fn)
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					if brush && rectangleRecordWrite(call) && (name != "pkg/mapedit/rectanglebrush.go" || fn.Name.Name != "rectangleStroke") {
						t.Errorf("%s:%s writes a structure outside rectangleStroke", name, fn.Name.Name)
					}
					if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "rectangleCover" {
						covers++
						if name != "pkg/mapedit/rectanglebrush.go" || fn.Name.Name != "rectangleStroke" {
							t.Errorf("%s:%s calls the cover outside the structure brush", name, fn.Name.Name)
						}
					}
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "rectangleStroke" {
						routes++
						if name != "pkg/mapedit/brush.go" || fn.Name.Name != "Paint" {
							t.Errorf("%s:%s bypasses Paint", name, fn.Name.Name)
						}
					}
					return true
				})
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if rectangleBrushType(lit.Type) {
				policies++
				if name != "pkg/game/mapinspectionbrush.go" {
					t.Errorf("%s declares a second rectangular structure policy", name)
				}
			}
			return true
		})
	}
	if covers != 1 || routes != 1 || policies != 1 {
		t.Fatalf("cover calls=%d, Paint routes=%d, policies=%d; want one each", covers, routes, policies)
	}
}

func rectangleBrushType(expr ast.Expr) bool {
	switch typ := expr.(type) {
	case *ast.Ident:
		return typ.Name == "RectangleBrush"
	case *ast.SelectorExpr:
		return typ.Sel.Name == "RectangleBrush"
	}
	return false
}

func rectangleBrushFunction(fn *ast.FuncDecl) bool {
	brush := false
	ast.Inspect(fn, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			brush = brush || n.Name == "Brush" || n.Name == "RectangleBrush"
		case *ast.SelectorExpr:
			brush = brush || n.Sel.Name == "Rectangles"
		}
		return true
	})
	return brush
}

func rectangleRecordWrite(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name == "NewStructureRecord" {
		return true
	}
	switch sel.Sel.Name {
	case "Place", "PlaceAll", "Move", "SetField", "SetFields", "Delete":
		for _, arg := range call.Args {
			switch arg := arg.(type) {
			case *ast.Ident:
				if arg.Name == "KindStructure" {
					return true
				}
			case *ast.SelectorExpr:
				if arg.Sel.Name == "KindStructure" {
					return true
				}
			}
		}
	}
	return false
}

func TestRectangleBrushRecognizesBypassWrites(t *testing.T) {
	for _, statement := range []string{
		"e.NewStructureRecord(0, 0, 33, 1)",
		"e.Place(KindStructure, record)",
		"e.PlaceAll(mapedit.KindStructure, records)",
		"e.Move(KindStructure, 0, 0, 0)",
		"e.SetField(KindStructure, 0, key, 3)",
		"e.SetFields(KindStructure, 0, field)",
		"e.Delete(KindStructure, 0)",
	} {
		f, err := parser.ParseFile(token.NewFileSet(), "brush.go", "package mapedit; func bypass(b Brush) { "+statement+" }", parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		fn := f.Decls[0].(*ast.FuncDecl)
		call := fn.Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr)
		if !rectangleBrushFunction(fn) || !rectangleRecordWrite(call) {
			t.Errorf("missed brush structure write: %s", statement)
		}
	}
	if !rectangleBrushType(ast.NewIdent("RectangleBrush")) {
		t.Fatal("missed an unqualified policy literal")
	}
}
