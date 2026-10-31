package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ObjectRegistryAllowed are the production files that may name the object
// registry entry, each with its reason.
var ObjectRegistryAllowed = map[string]string{
	"pkg/game/objectcatalog.go": "the object catalog, the one reader of the install's object registry",
	"cmd/classdump/main.go":     "a dump tool that reads all three registries from a bare archive path, not from an install",

	"internal/archtest/objectregistry.go": "this scan names the entry it looks for",
}

// ObjectRegistrySites lists the production files not allowed that hold a
// string literal naming objects.reg. Tests name it as fixture data.
func ObjectRegistrySites(root string) ([]string, error) {
	var sites []string
	for _, top := range []string{"pkg", "cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			rel, rerr := filepath.Rel(root, path)
			if rerr != nil {
				return rerr
			}
			rel = filepath.ToSlash(rel)
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
				return nil
			}
			if _, ok := ObjectRegistryAllowed[rel]; ok {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			sites = append(sites, objectRegistryInSource(rel, src)...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(sites)
	return sites, nil
}

func objectRegistryInSource(name string, src []byte) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	var hits []string
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if s, err := strconv.Unquote(lit.Value); err == nil && strings.Contains(strings.ToLower(s), "objects.reg") {
			hits = append(hits, name+":"+strconv.Itoa(fset.Position(lit.Pos()).Line))
		}
		return true
	})
	return hits
}
