package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
	"testing"
)

// fixturePackage is one synthetic package of a planted module: its import
// path and its production sources by module-relative path.
type fixturePackage struct {
	path  string
	files map[string]string
}

type fixtureImporter map[string]*types.Package

func (f fixtureImporter) Import(path string) (*types.Package, error) {
	if p, ok := f[path]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("import not available in this synthetic module: %s", path)
}

// checkedFixture type-checks the packages in the order given, each importing
// only earlier ones, and returns them as the module the guards read.
func checkedFixture(t *testing.T, pkgs ...fixturePackage) *TypeCheckedModule {
	t.Helper()
	m := &TypeCheckedModule{Fset: token.NewFileSet(), Packages: map[string]*CheckedPackage{}}
	imp := fixtureImporter{}
	for _, p := range pkgs {
		rels := make([]string, 0, len(p.files))
		for rel := range p.files {
			rels = append(rels, rel)
		}
		sort.Strings(rels)
		var files []*ast.File
		for _, rel := range rels {
			f, err := parser.ParseFile(m.Fset, rel, p.files[rel], 0)
			if err != nil {
				t.Fatal(err)
			}
			files = append(files, f)
		}
		info := &types.Info{
			Types: map[ast.Expr]types.TypeAndValue{},
			Defs:  map[*ast.Ident]types.Object{},
			Uses:  map[*ast.Ident]types.Object{},
		}
		pkg, err := (&types.Config{Importer: imp}).Check(p.path, m.Fset, files, info)
		if err != nil {
			t.Fatal(err)
		}
		imp[p.path] = pkg
		m.Packages[p.path] = &CheckedPackage{ImportPath: p.path, Files: files, RelPaths: rels, NumProd: len(files), Info: info}
	}
	return m
}
