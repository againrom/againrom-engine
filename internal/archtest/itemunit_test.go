package archtest

import (
	"os"
	"strings"
	"testing"
)

const itemUnitRuleSource = "package sim\nfunc SameUnit[T any](a, b T, rule UnitRule) bool { return StackStateEqual(x, y) || ItemEqual(i, j) }\n"

// TestCheckItemUnitNamesEachUse plants each shape an item comparison outside
// the identity rule takes, and the uses the rule keeps.
func TestCheckItemUnitNamesEachUse(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a crossing through the rule", "pkg/game/a.go",
			"package game\nimport \"againrom/pkg/sim\"\nfunc f(a, b sim.ItemStack) bool { return sim.SameUnit(a, b, sim.UnitStack) || a.Empty() }\n", ""},
		{"a stack state call", "pkg/game/a.go",
			"package game\nimport \"againrom/pkg/sim\"\nfunc f(a, b sim.ItemStack) bool { return sim.StackStateEqual(a, b) }\n", "sim.StackStateEqual"},
		{"an item equality call", "pkg/secondshop/a.go",
			"package secondshop\nimport \"againrom/pkg/sim\"\nfunc f(a, b sim.ItemInstance) bool { return a.Price == b.Price && sim.ItemEqual(a, b) }\n", "sim.ItemEqual"},
		{"a merge call", "pkg/game/a.go",
			"package game\nimport \"againrom/pkg/sim\"\nfunc f(a, b sim.ItemInstance) bool { return sim.CanMergeItemValues(a, b) }\n", "sim.CanMergeItemValues"},
		{"a comparison passed as a value", "pkg/game/a.go",
			"package game\nimport \"againrom/pkg/sim\"\nvar same = sim.ItemEqual\n", "sim.ItemEqual"},
		{"a renamed import", "pkg/game/a.go",
			"package game\nimport s \"againrom/pkg/sim\"\nfunc f(a, b s.ItemStack) bool { return s.StackStateEqual(a, b) }\n", "sim.StackStateEqual"},
		{"another package's ItemEqual", "pkg/game/a.go",
			"package game\nimport \"againrom/pkg/sim\"\nfunc f(a, b sim.ItemStack, o other) bool { return o.ItemEqual(a, b) }\n", ""},
		{"a test file", "pkg/game/a_test.go",
			"package game\nimport \"againrom/pkg/sim\"\nfunc f(a, b sim.ItemStack) bool { return sim.StackStateEqual(a, b) }\n", ""},
		{"outside the roots", "pkg/sim/a.go",
			"package sim\nfunc f(a, b ItemStack) bool { return StackStateEqual(a, b) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckItemUnit(map[string]string{itemUnitRuleFile: itemUnitRuleSource, tc.file: tc.src})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("CheckItemUnit = %v, want none", vs)
				}
				return
			}
			if len(vs) != 1 || !strings.Contains(vs[0].Reason, tc.want) || !strings.HasPrefix(vs[0].From, tc.file+":") {
				t.Fatalf("CheckItemUnit = %v, want one in %s naming %q", vs, tc.file, tc.want)
			}
		})
	}
	if vs := CheckItemUnit(map[string]string{"pkg/game/a.go": "package game\n"}); len(vs) != 1 || !strings.Contains(vs[0].Reason, "rule is gone") {
		t.Fatalf("CheckItemUnit without the rule = %v, want one violation", vs)
	}
}

// TestItemUnitHasOneRule is the live-tree half.
func TestItemUnitHasOneRule(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["pkg/game/shop.go"]; !ok {
		t.Fatal("pkg/game/shop.go is not among the scanned sources")
	}
	for _, v := range CheckItemUnit(files) {
		t.Errorf("%s", v)
	}
}
