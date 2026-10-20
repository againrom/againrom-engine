package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// WornChangeFile holds the one wear and take-off transaction; WornTownHolderFile
// holds the town's holder. A Spell deletion elsewhere in production pkg/sim or
// pkg/game is a second implementation of the take-off rule unless its file's
// count is committed in CommittedSpellDeletionSites.
const (
	WornChangeFile     = "pkg/sim/wornchange.go"
	WornTownHolderFile = "pkg/game/savweaponspell.go"
)

// SpellDeletionSites counts each production file's Spell deletions outside
// the transaction and its holders: an assignment of the empty SourceItemSpell
// or of 0 to a .Spell selector, and a replaceObjectSpell call with the empty
// SourceItemSpell.
func SpellDeletionSites(files map[string]string) (map[string]int, []Violation) {
	out := map[string]int{}
	var vs []Violation
	for _, name := range sortedNames(files) {
		if !strings.HasPrefix(name, "pkg/sim/") && !strings.HasPrefix(name, "pkg/game/") {
			continue
		}
		if name == WornChangeFile || name == WornTownHolderFile || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		n := 0
		ast.Inspect(f, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.AssignStmt:
				if len(x.Lhs) != len(x.Rhs) {
					return true
				}
				for k, lhs := range x.Lhs {
					if sel, ok := lhs.(*ast.SelectorExpr); ok && sel.Sel.Name == "Spell" && spellDeletionValue(x.Rhs[k]) {
						n++
					}
				}
			case *ast.CallExpr:
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "replaceObjectSpell" && len(x.Args) == 2 && emptySpellLiteral(x.Args[1]) {
					n++
				}
			}
			return true
		})
		if n != 0 {
			out[name] = n
		}
	}
	return out, vs
}

func spellDeletionValue(e ast.Expr) bool {
	if lit, ok := e.(*ast.BasicLit); ok {
		return lit.Kind == token.INT && lit.Value == "0"
	}
	return emptySpellLiteral(e)
}

func emptySpellLiteral(e ast.Expr) bool {
	lit, ok := e.(*ast.CompositeLit)
	if !ok || len(lit.Elts) != 0 {
		return false
	}
	switch t := lit.Type.(type) {
	case *ast.Ident:
		return t.Name == "SourceItemSpell"
	case *ast.SelectorExpr:
		return t.Sel.Name == "SourceItemSpell"
	}
	return false
}

// CheckSpellDeletionSites reports a count above or below its committed one.
func CheckSpellDeletionSites(got, committed map[string]int) []Violation {
	var vs []Violation
	for _, name := range sortedNames(got) {
		switch want := committed[name]; {
		case got[name] > want:
			vs = append(vs, Violation{From: name, Reason: fmt.Sprintf("%d Spell deletions, committed %d: a worn item's Spell changes through sim.TakeOffWorn in %s", got[name], want, WornChangeFile)})
		case got[name] < want:
			vs = append(vs, Violation{From: name, Reason: fmt.Sprintf("%d Spell deletions, committed %d: lower CommittedSpellDeletionSites", got[name], want)})
		}
	}
	for _, name := range sortedNames(committed) {
		if _, ok := got[name]; !ok {
			vs = append(vs, Violation{From: name, Reason: fmt.Sprintf("0 Spell deletions, committed %d: drop the row from CommittedSpellDeletionSites", committed[name])})
		}
	}
	return vs
}
