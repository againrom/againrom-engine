package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pickupBindingBypass(src string) bool {
	f, err := parser.ParseFile(token.NewFileSet(), "binding.go", src, 0)
	if err != nil {
		return true
	}
	bypass := false
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.KeyValueExpr:
			if field, ok := n.Key.(*ast.Ident); ok && missionBindingField(field.Name) {
				bypass = true
			}
		case *ast.CaseClause:
			physical := false
			for _, label := range n.List {
				if lit, ok := label.(*ast.BasicLit); ok {
					name := strings.Trim(lit.Value, "\"'")
					physical = physical || strings.Contains(" p f g d c s t r m l f4 f5 f6 f7 f8 f9 f10 f11 f12 ", " "+name+" ")
				}
			}
			if physical {
				for _, stmt := range n.Body {
					ast.Inspect(stmt, func(node ast.Node) bool {
						if assign, ok := node.(*ast.AssignStmt); ok {
							for _, lhs := range assign.Lhs {
								if field, ok := lhs.(*ast.SelectorExpr); ok && missionBindingField(field.Sel.Name) {
									bypass = true
								}
							}
						}
						return true
					})
				}
			}
		}
		return true
	})
	return bypass
}

func missionBindingField(name string) bool {
	return strings.Contains(" Grab Patrol Guard Defend Cast March StandGround PlayerRetreat Move Support QuickSpell QuickSave QuickLoad Grid Readout FPS Reveal ", " "+name+" ")
}

func TestPickupAndPatrolKeysUseOneBindingTable(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for file, call := range map[string]string{"app.go": "in.readMissionKeys(keys,", "headless.go": "in.missionKeyNamed(a.activeMissionKeys(),"} {
		src, err := os.ReadFile(filepath.Join(root, "pkg", "ui", file))
		if err != nil {
			t.Fatal(err)
		}
		if pickupBindingBypass(string(src)) || !strings.Contains(string(src), call) {
			t.Errorf("%s bypasses the shared pickup and patrol binding reader", file)
		}
	}
}

func TestPickupBindingGuardRejectsDirectAssignments(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want bool
	}{
		{`package ui; var in = appInput{Grab: pressed()}`, true},
		{`package ui; var in = appInput{Patrol: pressed()}`, true},
		{`package ui; var in = appInput{QuickSpell: pressed()}`, true},
		{`package ui; func f() { switch key { case "d": in.Defend = true } }`, true},
		{`package ui; func f() { switch key { case "p", "patrol": in.Patrol = true } }`, true},
		{`package ui; func f() { switch key { case 'f': in.Grab = true } }`, true},
		{`package ui; func f() { switch key { case "p", "f": in.missionKeyNamed(keys, key) } }`, false},
		{`package ui; func f() { switch key { case "patrol": in.Patrol = true } }`, false},
	} {
		if got := pickupBindingBypass(tc.src); got != tc.want {
			t.Errorf("guard %v, want %v: %s", got, tc.want, tc.src)
		}
	}
}
