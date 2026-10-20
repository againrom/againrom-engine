package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"testing"
)

// Every SAVE route of the game captures through the one capture function,
// and the Group drop rule has one applier. Each function the rule names must
// exist, so a rename cannot leave the rule holding nothing.
func TestSaveRoutesCaptureThroughOneFunction(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	module, err := loadTypeCheckedModule(root)
	if err != nil {
		t.Fatal(err)
	}
	var packages []*CheckedPackage
	defined := map[string]bool{}
	for _, p := range module.Packages {
		packages = append(packages, p)
		for _, file := range p.Files[:p.NumProd] {
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					if obj, _ := p.Info.Defs[fn.Name].(*types.Func); obj != nil {
						defined[saveCaptureKey(obj)] = true
					}
				}
			}
		}
	}
	for key, owners := range saveCaptureOwners {
		if !defined[key] {
			t.Errorf("%s is not a production function", key)
		}
		for _, owner := range owners {
			if !defined[owner] {
				t.Errorf("owner %s of %s is not a production function", owner, key)
			}
		}
	}
	for _, finding := range CheckSaveCapture(packages) {
		t.Error(finding)
	}
}

func TestSaveCaptureRuleRejectsAFourthVariant(t *testing.T) {
	const simSource = `package sim; type World struct{}; func SaveKeepsGroup(p uint32, n int) bool { return n != 0 || p != 0 }
func KeepGroupsForSave(n int) bool { return SaveKeepsGroup(0, n) }
func (w *World) PruneGroupsForSaveUndo() (bool, func()) { return false, func() {} }
func (w *World) PruneGroupsForSave() bool { p, _ := w.PruneGroupsForSaveUndo(); return p }`
	const gameHead = `package game; import "againrom/pkg/sim"; type Snapshot struct{}; type Town struct{}; func (t *Town) pruneCityGroupsForSave() {}
type FrontEnd struct{ w *sim.World; t *Town }
func (f *FrontEnd) Snapshot(bool) (Snapshot, string, error) { return Snapshot{}, "", nil }
func pruneGroupsForSave(f *FrontEnd) func() { _, undo := f.w.PruneGroupsForSaveUndo(); f.t.pruneCityGroupsForSave(); return undo }
func (f *FrontEnd) captureSave(build func(Snapshot) error) error { defer pruneGroupsForSave(f)(); s, _, _ := f.Snapshot(true); return build(s) }
`
	for _, tc := range []struct {
		name, sim, game string
		bad             bool
	}{
		{"route through the capture", "", `func (f *FrontEnd) save() error { return f.captureSave(func(Snapshot) error { return nil }) }`, false},
		{"route that captures by itself", "", `func (f *FrontEnd) save() { f.Snapshot(true) }`, true},
		{"route that holds the capture as a value", "", `func (f *FrontEnd) save() { capture := f.Snapshot; capture(false) }`, true},
		{"route with its own prune", "", `func (f *FrontEnd) save() { pruneGroupsForSave(f); f.captureSave(nil) }`, true},
		{"route that prunes the World", "", `func (f *FrontEnd) save() { f.w.PruneGroupsForSave() }`, true},
		{"route that prunes the town", "", `func (f *FrontEnd) save() { f.t.pruneCityGroupsForSave() }`, true},
		{"game applier of the rule", "", `func keep(n int) bool { return sim.SaveKeepsGroup(0, n) }`, true},
		{"second sim applier", `func (w *World) drops(n int) bool { return !SaveKeepsGroup(0, n) }`, ``, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			simFile, err := parser.ParseFile(fset, "sim.go", simSource+"\n"+tc.sim, 0)
			if err != nil {
				t.Fatal(err)
			}
			simInfo := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
			simPkg, err := (&types.Config{}).Check("againrom/pkg/sim", fset, []*ast.File{simFile}, simInfo)
			if err != nil {
				t.Fatal(err)
			}
			gameFile, err := parser.ParseFile(fset, "game.go", gameHead+tc.game, 0)
			if err != nil {
				t.Fatal(err)
			}
			gameInfo := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
			config := types.Config{Importer: fixedImporter{path: "againrom/pkg/sim", pkg: simPkg}}
			if _, err := config.Check("againrom/pkg/game", fset, []*ast.File{gameFile}, gameInfo); err != nil {
				t.Fatal(err)
			}
			packages := []*CheckedPackage{
				{ImportPath: "againrom/pkg/sim", Files: []*ast.File{simFile}, NumProd: 1, Info: simInfo},
				{ImportPath: "againrom/pkg/game", Files: []*ast.File{gameFile}, NumProd: 1, Info: gameInfo},
			}
			if got := CheckSaveCapture(packages); (len(got) != 0) != tc.bad {
				t.Fatalf("findings %v, want bad=%v", got, tc.bad)
			}
		})
	}
}
