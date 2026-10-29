package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// The town room popup has one lifecycle in both games: townScreen holds one
// tip field, tip of type roomTip; roomTip is declared once, in
// pkg/game/tips.go; only that file writes the field or its parts; and its
// functions read no host rules, so what a game adds is tip table data.
func TestTownRoomTipHasOneLifecycle(t *testing.T) {
	const home = "pkg/game/tips.go"
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
	declared, tipFields := 0, 0
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/game/") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		at := func(n ast.Node) int { return fset.Position(n.Pos()).Line }
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if ts.Name.Name == "roomTip" {
						declared++
						if name != home {
							t.Errorf("%s:%d declares roomTip outside %s", name, at(ts), home)
						}
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok || ts.Name.Name != "townScreen" {
						continue
					}
					for _, field := range st.Fields.List {
						for _, id := range field.Names {
							if !strings.Contains(strings.ToLower(id.Name), "tip") {
								continue
							}
							tipFields++
							if typ, ok := field.Type.(*ast.Ident); id.Name != "tip" || !ok || typ.Name != "roomTip" {
								t.Errorf("%s:%d townScreen field %s holds room tip state beside tip roomTip", name, at(id), id.Name)
							}
						}
					}
				}
			case *ast.FuncDecl:
				if d.Body == nil {
					continue
				}
				ast.Inspect(d.Body, func(n ast.Node) bool {
					var targets []ast.Expr
					switch s := n.(type) {
					case *ast.AssignStmt:
						targets = s.Lhs
					case *ast.IncDecStmt:
						targets = []ast.Expr{s.X}
					case *ast.CallExpr:
						if sel, ok := s.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "hostRules" && name == home {
							t.Errorf("%s:%d %s reads host rules; the room tip lifecycle takes its rules from the tip table", name, at(s), d.Name.Name)
						}
					}
					for _, e := range targets {
						if writesRoomTip(e) && name != home {
							t.Errorf("%s:%d %s writes the room tip outside %s", name, at(e), d.Name.Name, home)
						}
					}
					return true
				})
			}
		}
	}
	if declared != 1 || tipFields != 1 {
		t.Fatalf("pkg/game declares roomTip %d times and townScreen %d tip fields, want 1 each", declared, tipFields)
	}
}

// writesRoomTip reports that an assignment target is a .tip selector or a
// part of one.
func writesRoomTip(e ast.Expr) bool {
	for {
		switch x := e.(type) {
		case *ast.SelectorExpr:
			if x.Sel.Name == "tip" {
				return true
			}
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.StarExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		default:
			return false
		}
	}
}
