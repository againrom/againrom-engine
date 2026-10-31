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

	"againrom/pkg/locale"
)

// LocaleValues are the table's names: Code, Entry, BaseID and "-"+Code; Code
// alone; the nonzero code pages.
type LocaleValues struct {
	Words map[string]bool
	Codes map[string]bool
	Pages map[int]bool
}

// TableLocaleValues reads them from locale.All().
func TableLocaleValues() LocaleValues {
	v := LocaleValues{Words: map[string]bool{}, Codes: map[string]bool{}, Pages: map[int]bool{}}
	for _, l := range locale.All() {
		for _, w := range []string{l.Code, l.Entry, l.BaseID, "-" + l.Code} {
			if w != "" && w != "-" {
				v.Words[w] = true
			}
		}
		v.Codes[l.Code] = true
		for _, p := range []int{l.CodePage, l.WindowsCodePage} {
			if p != 0 {
				v.Pages[p] = true
			}
		}
	}
	return v
}

// localePolicyExempt hold damage selectors, and import no pkg/locale (tested).
var localePolicyExempt = []string{"pkg/sim/", "pkg/data/"}

// LocalePolicySites names where pkg/ and cmd/ code outside pkg/locale and
// pkg/base/base.go spells a table name (also as a path segment), tests a value
// against a table code page, or tests a Selector or code page field, or a
// variable selector, against an integer.
func LocalePolicySites(root string) ([]string, error) {
	vals := TableLocaleValues()
	var sites []string
	for _, top := range []string{"pkg", "cmd"} {
		err := walkProductionGo(root, top, func(rel string, src []byte) {
			if strings.HasPrefix(rel, "pkg/locale/") || rel == "pkg/base/base.go" {
				return
			}
			sites = append(sites, localePolicyInSource(rel, src, vals)...)
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(sites)
	return sites, nil
}

// LocalePairSites names where code outside pkg/locale maps one code to the other.
func LocalePairSites(root string) ([]string, error) {
	codes := TableLocaleValues().Codes
	var sites []string
	for _, top := range []string{"pkg", "cmd", "internal"} {
		err := walkProductionGo(root, top, func(rel string, src []byte) {
			if strings.HasPrefix(rel, "pkg/locale/") {
				return
			}
			sites = append(sites, localePairInSource(rel, src, codes)...)
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(sites)
	return sites, nil
}

// walkProductionGo visits the non-test Go files under root/top.
func walkProductionGo(root, top string, visit func(rel string, src []byte)) error {
	return filepath.WalkDir(filepath.Join(root, top), func(path string, d fs.DirEntry, err error) error {
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
		src, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		visit(rel, src)
		return nil
	})
}

func localePolicyInSource(name string, src []byte, vals LocaleValues) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	var hits []string
	add := func(n ast.Node, what string) {
		hits = append(hits, name+":"+strconv.Itoa(fset.Position(n.Pos()).Line)+": "+what)
	}
	fieldTests := !localePolicyExemptFile(name)
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BasicLit:
			if s, ok := stringLiteral(n); ok {
				if vals.Words[s] {
					add(n, "string literal "+strconv.Quote(s)+" names a language")
				} else if hasCodeSegment(s, vals.Codes) {
					add(n, "string literal "+strconv.Quote(s)+" holds a language code as a path segment")
				}
			}
		case *ast.BinaryExpr:
			if !isComparison(n.Op) {
				return true
			}
			x, y := n.X, n.Y
			if _, ok := localeIntLiteral(x); ok {
				x, y = y, x
			}
			v, ok := localeIntLiteral(y)
			if !ok {
				return true
			}
			if vals.Pages[v] && (n.Op == token.EQL || n.Op == token.NEQ) {
				add(n, "a value is tested against the code page "+strconv.Itoa(v))
			}
			if fieldTests {
				switch langFieldKind(x) {
				case fieldSelectorOrPage:
					add(n, "a language field is tested against "+strconv.Itoa(v))
				case bareSelector:
					if n.Op == token.EQL || n.Op == token.NEQ {
						add(n, "a selector is tested against "+strconv.Itoa(v))
					}
				}
			}
		case *ast.SwitchStmt:
			if fieldTests && n.Tag != nil && langFieldKind(n.Tag) != notLanguageField {
				for _, c := range n.Body.List {
					cc, ok := c.(*ast.CaseClause)
					if !ok {
						continue
					}
					for _, e := range cc.List {
						if _, ok := localeIntLiteral(e); ok {
							add(e, "a switch on a language field has an integer case")
						}
					}
				}
			}
		}
		return true
	})
	return hits
}

func localePolicyExemptFile(name string) bool {
	for _, d := range localePolicyExempt {
		if strings.HasPrefix(name, d) {
			return true
		}
	}
	return false
}

type languageFieldKind int

const (
	notLanguageField languageFieldKind = iota
	fieldSelectorOrPage
	bareSelector
)

func langFieldKind(e ast.Expr) languageFieldKind {
	switch e := e.(type) {
	case *ast.ParenExpr:
		return langFieldKind(e.X)
	case *ast.SelectorExpr:
		switch e.Sel.Name {
		case "Selector", "CodePage", "WindowsCodePage":
			return fieldSelectorOrPage
		}
	case *ast.Ident:
		if e.Name == "selector" {
			return bareSelector
		}
	}
	return notLanguageField
}

func isComparison(op token.Token) bool {
	switch op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	}
	return false
}

func localeIntLiteral(e ast.Expr) (int, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, false
	}
	v, err := strconv.ParseInt(lit.Value, 0, 64)
	return int(v), err == nil
}

func hasCodeSegment(s string, codes map[string]bool) bool {
	if !strings.Contains(s, "/") {
		return false
	}
	for _, seg := range strings.Split(s, "/") {
		if codes[seg] {
			return true
		}
	}
	return false
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
