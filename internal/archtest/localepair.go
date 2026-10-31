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

// LocalePairSites lists the production files outside pkg/locale that map one
// language code to the other with literals; locale.Other answers that.
func LocalePairSites(root string) ([]string, error) {
	codes := map[string]bool{"en": true, "ru": true}
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
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") || strings.HasPrefix(rel, "pkg/locale/") {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			sites = append(sites, localePairInSource(rel, src, codes)...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(sites)
	return sites, nil
}

func localePairInSource(name string, src []byte, codes map[string]bool) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	var hits []string
	ast.Inspect(file, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		k, kok := stringLiteral(kv.Key)
		v, vok := stringLiteral(kv.Value)
		if kok && vok && k != v && codes[k] && codes[v] {
			hits = append(hits, name+":"+strconv.Itoa(fset.Position(kv.Pos()).Line))
		}
		return true
	})
	return hits
}

func stringLiteral(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil
}
