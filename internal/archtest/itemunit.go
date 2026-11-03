package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"
)

// The item identity scan: every crossing of an item between a shop, a bag,
// a party pack, the city holdings and a saved form asks sim.SameUnit with a
// named row. In production pkg/game and pkg/secondshop it refuses any use of
// the comparisons the rule's rows are made of, called or passed as a value.

const itemUnitRuleFile = "pkg/sim/itemunit.go"

const simImportPath = "againrom/pkg/sim"

var itemUnitRoots = []string{"pkg/game/", "pkg/secondshop/"}

var itemUnitComparisons = map[string]bool{"StackStateEqual": true, "ItemEqual": true, "CanMergeItemValues": true}

// CheckItemUnit reports every use of an item comparison outside the item
// identity rule. files is keyed by module-relative slash path.
func CheckItemUnit(files map[string]string) []Violation {
	if !strings.Contains(files[itemUnitRuleFile], "func SameUnit[") {
		return []Violation{{From: itemUnitRuleFile, Reason: "the item identity rule is gone: no SameUnit is declared"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !itemUnitScanned(name) {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		local := ""
		for _, spec := range f.Imports {
			if p, err := strconv.Unquote(spec.Path.Value); err == nil && p == simImportPath {
				local = path.Base(p)
				if spec.Name != nil {
					local = spec.Name.Name
				}
			}
		}
		if local == "" {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok || !itemUnitComparisons[sel.Sel.Name] {
				return true
			}
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == local {
				vs = append(vs, Violation{From: name + ":" + strconv.Itoa(fset.Position(sel.Pos()).Line),
					Reason: "sim." + sel.Sel.Name + " compares items here; ask sim.SameUnit with the crossing's row"})
			}
			return true
		})
	}
	return vs
}

func itemUnitScanned(name string) bool {
	if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
		return false
	}
	for _, root := range itemUnitRoots {
		if strings.HasPrefix(name, root) {
			return true
		}
	}
	return false
}
