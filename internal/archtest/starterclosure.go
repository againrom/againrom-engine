package archtest

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// starterRoot is the launcher; starterForbidden are the packages it must not
// link, directly or through any import of its own.
const starterRoot = "cmd/starter"

var starterForbidden = []string{"pkg/game", "pkg/ui", "pkg/sim"}

// checkClosure walks the module-relative import graph from start and reports
// every forbidden package it reaches, with one import chain that reaches it.
func checkClosure(start string, forbidden []string, imports func(pkg string) ([]string, error)) []Violation {
	bad := map[string]bool{}
	for _, f := range forbidden {
		bad[f] = true
	}
	chain := map[string]string{start: start}
	var vs []Violation
	queue := []string{start}
	for len(queue) > 0 {
		pkg := queue[0]
		queue = queue[1:]
		list, err := imports(pkg)
		if err != nil {
			vs = append(vs, Violation{From: pkg, Reason: err.Error()})
			continue
		}
		for _, imp := range list {
			if _, seen := chain[imp]; seen {
				continue
			}
			chain[imp] = chain[pkg] + " -> " + imp
			if bad[imp] {
				vs = append(vs, Violation{From: start, Reason: "links " + imp + " through " + chain[imp]})
				continue
			}
			queue = append(queue, imp)
		}
	}
	sort.Slice(vs, func(i, j int) bool { return vs[i].Reason < vs[j].Reason })
	return vs
}

// packageImports reads the production files of one package of the module, on
// every platform, and returns the module-relative packages they import.
func packageImports(root, pkg string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(pkg)))
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, filepath.FromSlash(pkg), e.Name()), nil, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return nil, err
			}
			if rel, ok := strings.CutPrefix(p, ModulePath+"/"); ok && !seen[rel] {
				seen[rel] = true
				out = append(out, rel)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// CheckStarterClosure reports every game package the launcher's production
// imports reach. The launcher holds the widgets it shares with the game in
// small packages precisely so that it does not link the game.
func CheckStarterClosure(root string) []Violation {
	return checkClosure(starterRoot, starterForbidden, func(pkg string) ([]string, error) {
		return packageImports(root, pkg)
	})
}
