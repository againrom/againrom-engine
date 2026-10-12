package archtest

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

func liveProductionSources(t *testing.T) map[string]string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// TestLiveAreaArmSitesMatchTheirBaseline measures the real tree.
func TestLiveAreaArmSitesMatchTheirBaseline(t *testing.T) {
	files := liveProductionSources(t)
	arms, err := AreaArmIDs(files[AreaArmTableFile])
	if err != nil {
		t.Fatal(err)
	}
	for _, arm := range []int64{3, 4, 7, 8, 9, 12, 17, 19, 21} {
		if !arms[arm] {
			t.Fatalf("the table names no arm %d; the scan reads the wrong file", arm)
		}
	}
	got, vs := AreaArmSites(files, arms)
	for _, v := range vs {
		t.Error(v)
	}
	if bad := CheckAreaArmSites(got, CommittedAreaArmSites); len(bad) != 0 {
		for _, v := range bad {
			t.Error(v)
		}
		names := make([]string, 0, len(got))
		for name := range got {
			names = append(names, name)
		}
		sort.Strings(names)
		var b strings.Builder
		for _, name := range names {
			fmt.Fprintf(&b, "\t%q: %d,\n", name, got[name])
		}
		t.Logf("measured:\n%s", b.String())
	}
}

// TestLiveSpellValuesLiveInOneFile measures the real tree.
func TestLiveSpellValuesLiveInOneFile(t *testing.T) {
	for _, v := range CheckSpellValues(liveProductionSources(t)) {
		t.Error(v)
	}
}

// TestAreaArmSitesAreFound: each shape of an arm site counts outside the
// table; an arm the table does not name does not.
func TestAreaArmSitesAreFound(t *testing.T) {
	table := `package sim
var areaArms = [...]areaArm{{arm: 3, layer: 0}, {arm: 21, stages: 32}}
`
	arms, err := AreaArmIDs(table)
	if err != nil {
		t.Fatal(err)
	}
	if len(arms) != 2 || !arms[3] || !arms[21] {
		t.Fatalf("arms %v, want 3 and 21", arms)
	}
	files := map[string]string{
		AreaArmTableFile: table + `func f(arm uint16) bool { return arm == 3 }`,
		"pkg/sim/a.go": `package sim
func a(w *World, rule SpellRule, e cellEffect, id uint16) bool {
	if rule.arm() == 3 || w.spellArm(e.Spell) != 21 || 21 == uint16(id) {
		return true
	}
	switch w.SpellArm(e.Spell) {
	case 3, 21, 5:
		return true
	}
	return w.spellArmIs(int64(id), 3) || rule.arm() == 5 || e.Spell == 3
}
`,
		"pkg/game/b.go": `package game
func b(spell uint16, n int) bool { return spell == 21 || n == 3 }
`,
	}
	got, vs := AreaArmSites(files, arms)
	if len(vs) != 0 {
		t.Fatal(vs)
	}
	want := map[string]int{"pkg/sim/a.go": 7, "pkg/game/b.go": 1}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("sites %v, want %v", got, want)
	}
	vs = CheckAreaArmSites(got, map[string]int{"pkg/sim/a.go": 9, "pkg/sim/c.go": 1})
	if len(vs) != 3 {
		t.Fatalf("violations %v, want a fall, a new file and a gone file", vs)
	}
}

// TestSpellValueFormulasAreFound: a leaf name, column arithmetic and a table
// read are findings; a column comparison and the two entries are not.
func TestSpellValueFormulasAreFound(t *testing.T) {
	files := map[string]string{
		SpellValuesFile: `package sim
func spellRangeOf() {}
func spellDurationOf() {}
func spellRange() {}
func areaLife() {}
`,
		"pkg/sim/a.go": `package sim
func a(r Rules, rule SpellRule) int64 {
	spellRange()
	f := areaLife
	_ = f
	spellRangeOf()
	_, _ = r.SpellFormulaAt(rules.FormulaRange, rule.ID, 3)
	_, _ = r.SpellFormulaAt(rules.FormulaDamage, rule.ID, 3)
	if int64(rule.MaxRange) > 4 {
		return int64(rule.AreaDuration) << 4
	}
	return int64(rule.MaxRange) + 1
}
`,
		"pkg/game/b.go": `package game
func b(rule sim.SpellRule) int32 { spellRange(); return rule.SpellDuration * 2 }
`,
	}
	var got []string
	for _, v := range CheckSpellValues(files) {
		got = append(got, v.From)
	}
	want := []string{"pkg/game/b.go:2", "pkg/sim/a.go:3", "pkg/sim/a.go:4", "pkg/sim/a.go:7", "pkg/sim/a.go:10", "pkg/sim/a.go:12"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("findings %v, want %v", got, want)
	}
}
