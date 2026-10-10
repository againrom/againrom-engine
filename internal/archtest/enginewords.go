package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The engine-words scan. The engine's own words are one table per language in
// pkg/words, chosen by the install's language; the menu font's selector only
// converts bytes. In the production files of pkg/ui and pkg/game it finds two
// shapes:
//   - a //go:embed directive: a file built into a screen package, the form
//     every per-screen word table took;
//   - an == or != between a font selector (a value or call whose name ends in
//     Selector) and a selector value (a number or text.SelectorConverting):
//     a branch on the font's alphabet, the form every language choice took.
var engineWordsPackages = []string{"pkg/ui/", "pkg/game/"}

// EngineWordsAllowed are the files that may hold a finding, each with its
// reason.
var EngineWordsAllowed = map[string]string{
	"pkg/game/townsquare.go": "embeds the first game's town description, not a word table",
}

// EngineWordsDebt is the selector branches that remain, per file. Each converts
// bytes for the font's alphabet rather than choosing words. A count may only
// fall; a file whose count reaches zero leaves the map in the same commit.
var EngineWordsDebt = map[string]int{
	"pkg/game/secondgametext.go": 1, // the second game's RU mission text code page
	"pkg/ui/gamemenu.go":         1, // the accelerator's CP866 lowercase fold
	"pkg/ui/load_draw.go":        1, // the EN font's ASCII fallback for a save name
}

// EngineWordsFinding is one site and the shape found there.
type EngineWordsFinding struct {
	File  string
	Line  int
	Shape string
}

func (f EngineWordsFinding) String() string {
	return fmt.Sprintf("%s:%d %s", f.File, f.Line, f.Shape)
}

// CheckEngineWords turns the findings into violations: a finding in a file
// that is neither allowed nor in debt, an embed in a debt file, a debt file
// holding more branches than its count, and a debt entry whose count no
// longer matches.
func CheckEngineWords(found []EngineWordsFinding, debt map[string]int) []string {
	var out []string
	held := map[string]int{}
	for _, f := range found {
		if EngineWordsAllowed[f.File] != "" {
			continue
		}
		if _, ok := debt[f.File]; ok && f.Shape == shapeSelectorBranch {
			held[f.File]++
			continue
		}
		out = append(out, f.String()+"; put engine words in pkg/words and read them through the install's language")
	}
	names := make([]string, 0, len(debt))
	for name := range debt {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		switch got := held[name]; {
		case got > debt[name]:
			out = append(out, fmt.Sprintf("%s holds %d font-selector branches, above its debt of %d", name, got, debt[name]))
		case got < debt[name]:
			out = append(out, fmt.Sprintf("%s holds %d font-selector branches, below its debt of %d; lower EngineWordsDebt", name, got, debt[name]))
		}
	}
	return out
}

const (
	shapeEmbed          = "embeds a file"
	shapeSelectorBranch = "branches on the font selector"
)

// LoadEngineWordsFindings reads every production file of pkg/ui and pkg/game.
func LoadEngineWordsFindings(root string) ([]EngineWordsFinding, error) {
	var out []EngineWordsFinding
	for _, pkg := range engineWordsPackages {
		names, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pkg), "*.go"))
		if err != nil {
			return nil, err
		}
		if len(names) == 0 {
			return nil, fmt.Errorf("%s holds no Go file; the scan went blind", pkg)
		}
		for _, name := range names {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			src, err := os.ReadFile(name)
			if err != nil {
				return nil, err
			}
			found, err := EngineWordsFindings(pkg+filepath.Base(name), string(src))
			if err != nil {
				return nil, err
			}
			out = append(out, found...)
		}
	}
	return out, nil
}

// EngineWordsFindings scans one source file named rel.
func EngineWordsFindings(rel, src string) ([]EngineWordsFinding, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var out []EngineWordsFinding
	add := func(pos token.Pos, shape string) {
		out = append(out, EngineWordsFinding{File: rel, Line: fset.Position(pos).Line, Shape: shape})
	}
	for _, g := range f.Comments {
		for _, c := range g.List {
			if strings.HasPrefix(c.Text, "//go:embed") {
				add(c.Pos(), shapeEmbed)
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		b, ok := n.(*ast.BinaryExpr)
		if !ok || b.Op != token.EQL && b.Op != token.NEQ {
			return true
		}
		if isFontSelector(b.X) && isSelectorValue(b.Y) || isFontSelector(b.Y) && isSelectorValue(b.X) {
			add(b.Pos(), shapeSelectorBranch)
		}
		return true
	})
	return out, nil
}

// isFontSelector reports whether e names a font selector: the converting
// selector constant, or an identifier, field or call whose name ends in
// Selector.
func isFontSelector(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.ParenExpr:
		return isFontSelector(e.X)
	case *ast.CallExpr:
		return isFontSelector(e.Fun)
	case *ast.Ident:
		return strings.HasSuffix(strings.ToLower(e.Name), "selector")
	case *ast.SelectorExpr:
		return e.Sel.Name == "SelectorConverting" || strings.HasSuffix(e.Sel.Name, "Selector")
	}
	return false
}

// isSelectorValue reports whether e is a selector value: a number or the
// converting selector constant.
func isSelectorValue(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.BasicLit:
		return e.Kind == token.INT
	case *ast.SelectorExpr:
		return e.Sel.Name == "SelectorConverting"
	}
	return false
}
