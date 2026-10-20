package sim

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func callsIn(t *testing.T, file, fn string) map[string]bool {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	calls := map[string]bool{}
	found := false
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name.Name != fn {
			continue
		}
		found = true
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				switch f := c.Fun.(type) {
				case *ast.SelectorExpr:
					calls[f.Sel.Name] = true
				case *ast.Ident:
					calls[f.Name] = true
				}
			}
			return true
		})
	}
	if !found {
		t.Fatalf("%s not found in %s", fn, file)
	}
	return calls
}

func TestStructureCallersShareOneApproach(t *testing.T) {
	for _, c := range []struct{ file, fn string }{
		{"structurecombat.go", "approachStructure"},
		{"structureuse.go", "advanceStructureUse"},
	} {
		calls := callsIn(t, c.file, c.fn)
		if !calls["approachCell"] {
			t.Errorf("%s bypasses approachCell", c.fn)
		}
		for _, banned := range []string{"searchRoute", "SliceStable", "restFree", "chebyshevTo"} {
			if banned == "chebyshevTo" && c.fn == "advanceStructureUse" {
				continue
			}
			if calls[banned] {
				t.Errorf("%s keeps its own scan: calls %s", c.fn, banned)
			}
		}
	}
	calls := callsIn(t, "structureapproach.go", "approachCell")
	for _, need := range []string{"searchRoute", "walkTo", "SliceStable", "restFree"} {
		if !calls[need] {
			t.Errorf("approachCell no longer calls %s", need)
		}
	}
}

func TestStructureCallersWalkThroughApproach(t *testing.T) {
	a, s := structureCombatActor(), structureCombatTarget()
	a.X, a.Y = 9, 3
	w := structureCombatWorld(t, 0, a, s)
	Step(w, []Command{AttackStructure(0, 0)})
	if e := w.entities[0]; !e.HasTarget || !InStructureReach(withCell(e, e.TargetX, e.TargetY), s) {
		t.Fatalf("attack walk goal not in reach: %+v", e)
	}

	u := structureUseWorld(t, 15)
	u.structures[0].Col, u.structures[0].Row = 12, 12
	Step(u, []Command{UseStructure(0, 0)})
	goal, radius := structureUseGoal(u.structures[0])
	if e := u.entities[0]; !e.HasTarget || (cell{e.TargetX, e.TargetY}).chebyshevTo(goal) > radius {
		t.Fatalf("use walk goal outside its square: %+v", e)
	}
}

func withCell(e Entity, x, y int32) Entity { e.X, e.Y = x, y; return e }
