package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"slices"
	"testing"
)

// ownedProductionFunctions is the module's checked packages and the key of
// each production function and interface method they define.
func ownedProductionFunctions(t *testing.T) ([]*CheckedPackage, map[string]bool) {
	t.Helper()
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
				ast.Inspect(decl, func(node ast.Node) bool {
					if it, ok := node.(*ast.InterfaceType); ok {
						for _, m := range it.Methods.List {
							for _, name := range m.Names {
								if obj, _ := p.Info.Defs[name].(*types.Func); obj != nil {
									defined[saveCaptureKey(obj)] = true
								}
							}
						}
					}
					return true
				})
			}
		}
	}
	return packages, defined
}

// checkOwnersDefined reports each function and owner of owners that is not
// a production function, so a rename cannot leave a rule holding nothing.
func checkOwnersDefined(t *testing.T, owners map[string][]string, defined map[string]bool) {
	t.Helper()
	for key, names := range owners {
		if !defined[key] {
			t.Errorf("%s is not a production function", key)
		}
		for _, owner := range names {
			if !defined[owner] {
				t.Errorf("owner %s of %s is not a production function", owner, key)
			}
		}
	}
}

// Every World carrier hook is run through the one carrier set, and the
// loaded-record carrier has one writer. Each function the rule names must
// exist, so a rename cannot leave the rule holding nothing.
func TestWorldCarriersRunThroughOneSet(t *testing.T) {
	packages, defined := ownedProductionFunctions(t)
	checkOwnersDefined(t, worldCarrierOwners, defined)
	for _, finding := range CheckWorldCarriers(packages) {
		t.Error(finding)
	}
}

// The carrier rule names a planted bypass: a LOAD that decodes the Diary by
// itself, a LOAD that writes it into the World, a headless mission start
// that seats the town purse itself, and a row whose hook the rule does not
// name.
func TestWorldCarrierRuleNamesAPlantedBypass(t *testing.T) {
	const simSource = `package sim; type World struct{}; func (w *World) ImportOriginalDiaries() error { return nil }
func (w *World) SetPurse(slot, gold uint32) bool { return true }`
	const gameHead = `package game; import "againrom/pkg/sim"
type worldCarrier struct{ decode func(*sim.World) error }
func importSavedDiaries(w *sim.World) error { return w.ImportOriginalDiaries() }
func seatTownPurse(w *sim.World) error { w.SetPurse(1, 100); return nil }
func plantedDecode(w *sim.World) error { return nil }
`
	const set = `func worldCarrierSet() []worldCarrier { return []worldCarrier{{decode: importSavedDiaries}, {decode: seatTownPurse}} }`
	for _, tc := range []struct {
		name, game, want string
	}{
		{"every carrier through the set", set, ""},
		{"a LOAD that decodes the Diary by itself", set + "\nfunc importSavedDocument(w *sim.World) error { return importSavedDiaries(w) }",
			"againrom/pkg/game.importSavedDocument references againrom/pkg/game.importSavedDiaries"},
		{"a LOAD that writes the Diary into the World", set + "\nfunc load(w *sim.World) { w.ImportOriginalDiaries() }",
			"againrom/pkg/game.load references againrom/pkg/sim.World.ImportOriginalDiaries"},
		{"a mission start that seats the town purse by itself", set + "\ntype FrontEnd struct{}\nfunc (f *FrontEnd) AdvanceLine(w *sim.World) { w.SetPurse(1, 100) }",
			"againrom/pkg/game.FrontEnd.AdvanceLine references againrom/pkg/sim.World.SetPurse"},
		{"a row whose hook the rule does not name", `func worldCarrierSet() []worldCarrier { return []worldCarrier{{decode: importSavedDiaries}, {decode: seatTownPurse}, {decode: plantedDecode}} }`,
			worldCarrierSet + " runs againrom/pkg/game.plantedDecode, which the carrier rule does not name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			simFile, err := parser.ParseFile(fset, "sim.go", simSource, 0)
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
			got := CheckWorldCarriers([]*CheckedPackage{
				{ImportPath: "againrom/pkg/sim", Files: []*ast.File{simFile}, NumProd: 1, Info: simInfo},
				{ImportPath: "againrom/pkg/game", Files: []*ast.File{gameFile}, NumProd: 1, Info: gameInfo},
			})
			if tc.want == "" && len(got) != 0 || tc.want != "" && !slices.Contains(got, tc.want) {
				t.Fatalf("findings %q, want %q", got, tc.want)
			}
		})
	}
}
