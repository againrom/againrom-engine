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

// HandFoldAllowed are the production files outside pkg/namefold that fold
// ASCII case by hand, each with its reason: none of them folds a resource
// name.
var HandFoldAllowed = map[string]string{
	"pkg/formats/sav/stateleaf.go": "keys the record names of a save's city-state tree",
	"pkg/ui/app.go":                "compares a typed hero name with the reserved names",
	"pkg/ui/gamemenu.go":           "matches a typed letter with a menu label's accelerator",
}

// HandFoldSites lists the functions of production code, outside pkg/namefold
// and the allowed files, that fold ASCII case by hand: they test a byte
// against 'A' and 'Z' and add the distance between the cases ('a' or 'A' in
// arithmetic, or 32). Resource names fold through pkg/namefold. Tests are not
// scanned.
func HandFoldSites(root string) ([]string, error) {
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
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") || strings.HasPrefix(rel, "pkg/namefold/") {
				return nil
			}
			if _, ok := HandFoldAllowed[rel]; ok {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			sites = append(sites, handFoldInSource(rel, src)...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(sites)
	return sites, nil
}

func handFoldInSource(name string, src []byte) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	var hits []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		var low, high, shift bool
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			b, ok := n.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			switch b.Op {
			case token.GEQ:
				low = low || isLit(b.Y, token.CHAR, "'A'")
			case token.LEQ:
				high = high || isLit(b.Y, token.CHAR, "'Z'")
			case token.ADD, token.SUB:
				for _, e := range []ast.Expr{b.X, b.Y} {
					e = unparen(e)
					shift = shift || isLit(e, token.CHAR, "'a'") || isLit(e, token.CHAR, "'A'") || isLit(e, token.INT, "32") || isLit(e, token.INT, "0x20")
					if inner, ok := e.(*ast.BinaryExpr); ok && inner.Op == token.SUB {
						shift = shift || isLit(inner.X, token.CHAR, "'a'") && isLit(inner.Y, token.CHAR, "'A'")
					}
				}
			}
			return true
		})
		if low && high && shift {
			hits = append(hits, name+":"+strconv.Itoa(fset.Position(fn.Pos()).Line)+" "+fn.Name.Name)
		}
	}
	return hits
}

func isLit(e ast.Expr, kind token.Token, value string) bool {
	lit, ok := unparen(e).(*ast.BasicLit)
	return ok && lit.Kind == kind && lit.Value == value
}

func unparen(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}
