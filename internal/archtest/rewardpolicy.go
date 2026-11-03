package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

const rewardPolicyFile = "pkg/sim/gameprofile.go"

// CheckRewardPolicy confines the reward switch to World.paysNoReward.
func CheckRewardPolicy(files map[string]string) []Violation {
	names := make([]string, 0, len(files))
	for name := range files {
		if strings.HasPrefix(name, "pkg/sim/") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var vs []Violation
	reads := 0
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		for _, unit := range declUnits(f) {
			ast.Inspect(unit.node, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "suppressedRewards" {
					return true
				}
				if name == rewardPolicyFile && unit.context == "World.paysNoReward" {
					reads++
				} else {
					vs = append(vs, Violation{
						From:   name + ":" + strconv.Itoa(fset.Position(sel.Pos()).Line),
						Reason: "reward switch read outside World.paysNoReward; call the predicate",
					})
				}
				return true
			})
		}
	}
	if reads != 1 {
		vs = append(vs, Violation{From: rewardPolicyFile, Reason: "World.paysNoReward must contain exactly one reward switch read; found " + strconv.Itoa(reads)})
	}
	return vs
}
