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

// TableReadersAllowed are the production files that may name the terrain cost
// table's file or parse a definition table directly, each with its reason.
var TableReadersAllowed = map[string]string{
	"pkg/game/table.go":         "the one terrain cost reader and the one definition table parse",
	"pkg/game/gamefiles.go":     "the game rows that hold the terrain cost table's address",
	"cmd/classdump/campaign.go": "a dump tool that parses a bare archive's definition table",
	"cmd/classdump/databin.go":  "a dump tool that parses a bare archive's definition table",
	"cmd/mapview/blocked.go":    "the viewer builds a buildings-only table from a bare world archive",

	"internal/archtest/tablereaders.go": "this scan names what it looks for",
}

// TableReaderSites lists the production files not allowed that hold a string
// literal naming the terrain cost table's file or call databin.Parse. Tests
// name both as fixture data.
func TableReaderSites(root string) ([]string, error) {
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
			if _, ok := TableReadersAllowed[rel]; ok {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			sites = append(sites, tableReadersInSource(rel, src)...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(sites)
	return sites, nil
}

func tableReadersInSource(name string, src []byte) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	var hits []string
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BasicLit:
			if s, err := strconv.Unquote(n.Value); n.Kind == token.STRING && err == nil && strings.Contains(strings.ToLower(s), "data/map.reg") {
				hits = append(hits, name+":"+strconv.Itoa(fset.Position(n.Pos()).Line))
			}
		case *ast.CallExpr:
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Parse" {
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "databin" {
					hits = append(hits, name+":"+strconv.Itoa(fset.Position(n.Pos()).Line))
				}
			}
		}
		return true
	})
	return hits
}
