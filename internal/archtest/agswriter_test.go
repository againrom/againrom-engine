package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"testing"
)

func TestNoProductionCallerWritesAGS(t *testing.T) {
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
	for _, finding := range CheckNoAGSWriter(module) {
		t.Error(finding)
	}
}

// fixedImporter resolves the synthetic module's only import.
type fixedImporter struct {
	path string
	pkg  *types.Package
}

func (f fixedImporter) Import(path string) (*types.Package, error) {
	if path == f.path {
		return f.pkg, nil
	}
	return nil, fmt.Errorf("import not available in this synthetic module: %s", path)
}

func TestCheckNoAGSWriterFindsOnlyAQualifiedProductionCall(t *testing.T) {
	fset := token.NewFileSet()
	gameFile, err := parser.ParseFile(fset, "game.go", `package game; func EncodeSave() {}`, 0)
	if err != nil {
		t.Fatal(err)
	}
	gameInfo := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	gamePkg, err := (&types.Config{}).Check("againrom/pkg/game", fset, []*ast.File{gameFile}, gameInfo)
	if err != nil {
		t.Fatal(err)
	}
	gameCheckedPackage := &CheckedPackage{ImportPath: "againrom/pkg/game", Files: []*ast.File{gameFile}, RelPaths: []string{"pkg/game/game.go"}, NumProd: 1, Info: gameInfo}

	for _, tc := range []struct {
		name, source string
		bad          bool
	}{
		{"qualified production call", `package caller

import "againrom/pkg/game"

func Bad() { game.EncodeSave() }
`, true},
		{"unrelated package, same method name", `package caller

type Other struct{}

func (o Other) EncodeSave() {}

func Fine() { Other{}.EncodeSave() }
`, false},
		{"no call at all", `package caller

func Fine() {}
`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			callerFile, err := parser.ParseFile(fset, "caller.go", tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			callerInfo := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
			importer := fixedImporter{path: "againrom/pkg/game", pkg: gamePkg}
			if _, err := (&types.Config{Importer: importer}).Check("caller", fset, []*ast.File{callerFile}, callerInfo); err != nil {
				t.Fatal(err)
			}
			module := &TypeCheckedModule{Fset: fset, Packages: map[string]*CheckedPackage{
				"againrom/pkg/game": gameCheckedPackage,
				"caller":            {ImportPath: "caller", Files: []*ast.File{callerFile}, RelPaths: []string{"cmd/caller/main.go"}, NumProd: 1, Info: callerInfo},
			}}
			if got := CheckNoAGSWriter(module); (len(got) != 0) != tc.bad {
				t.Fatalf("findings %v, want bad=%v", got, tc.bad)
			}
		})
	}

	t.Run("call reachable only through a test file is not production", func(t *testing.T) {
		callerFile, err := parser.ParseFile(fset, "caller_test.go", `package caller

import "againrom/pkg/game"

func testOnlyBad() { game.EncodeSave() }
`, 0)
		if err != nil {
			t.Fatal(err)
		}
		callerInfo := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
		importer := fixedImporter{path: "againrom/pkg/game", pkg: gamePkg}
		if _, err := (&types.Config{Importer: importer}).Check("caller", fset, []*ast.File{callerFile}, callerInfo); err != nil {
			t.Fatal(err)
		}
		module := &TypeCheckedModule{Fset: fset, Packages: map[string]*CheckedPackage{
			"againrom/pkg/game": gameCheckedPackage,
			// NumProd 0: the only file is a _test.go augmentation, exactly the
			// slice CheckNoAGSWriter must never look past.
			"caller": {ImportPath: "caller", Files: []*ast.File{callerFile}, RelPaths: []string{"cmd/caller/main_test.go"}, NumProd: 0, Info: callerInfo},
		}}
		if got := CheckNoAGSWriter(module); len(got) != 0 {
			t.Fatalf("findings %v, want none: a test-only file must not count as a production caller", got)
		}
	})
}
