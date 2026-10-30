package archtest

import (
	"go/ast"
	"go/types"
	"os"
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
