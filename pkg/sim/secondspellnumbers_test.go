package sim

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateSecondSpellNumbers = flag.Bool("update-second-spell-numbers", false,
	"rewrite testdata/secondspellnumbers.txt from the current code")

// secondCatalogSpells is the second game's catalog profile length, which
// TestSpellCatalogProfilesReproduceTheReplacedTables pins in pkg/rules; a
// sim test imports no project package.
const secondCatalogSpells = 29

// secondCatalogRow is the second game's spell number id with synthetic
// columns that vary by row, its arm set as the loader sets it.
func secondCatalogRow(id int) SpellRule {
	i := int32(id - 1)
	distribution := uint8(1)
	if i%3 == 1 {
		distribution = 3
	}
	return secondRow(SpellRule{ID: uint16(id), School: uint8(i%5 + 1), MaxRange: uint8(i % 4 * 3),
		DamageMin: 2 + i, DamageMax: 4 + 2*i, TargetsUnit: true, Area: distribution != 1, Distribution: distribution,
		Radius: 2, AreaDuration: i % 3 * 2, SpellDuration: i % 5, EffectKind: EffectKind(i % 4), EffectMode: EffectDuration,
		EffectMagnitude: i%7 - 3, EffectDuration: uint16(8 + i)})
}

// secondSpellNumbers states, for every spell of the second game's catalog
// profile, each number a cast or the book computes at a run of powers under r.
func secondSpellNumbers(t *testing.T, r Rules) string {
	t.Helper()
	var b strings.Builder
	for id := 1; id <= secondCatalogSpells; id++ {
		rule := secondCatalogRow(id)
		name := fmt.Sprintf("spell %d arm %d", id, rule.arm())
		w := spWorld(t, 1, []SpellRule{rule}, spEnt(1, 2, 2))
		w.SetRules(r)
		for _, sum := range []int32{0, 31, 60, 130, 200, 285, 400, 510} {
			fmt.Fprintf(&b, "%s power %d: %d\n", name, sum, spellPowerUnder(r, rule, sum/2, sum-sum/2))
		}
		for _, p := range []int32{0, 1, 2, 29, 30, 45, 99, 100, 101, 170, 200, 255} {
			lo, hi := SpellDamagePair(r, rule, p)
			kind, mag, duration, mode := w.pointEffect(0, rule, p)
			fmt.Fprintf(&b, "%s at %d: damage %d-%d range %d/%d duration %d/%d/%d effect %d %d %d %d rays %d\n", name, p, lo, hi,
				spellRangeOf(r, rule, p, rangeCast), spellRangeOf(r, rule, p, rangeRecord),
				spellDurationOf(r, rule, p, durationEffect), spellDurationOf(r, rule, p, durationRecord),
				spellDurationOf(r, rule, p, durationArea), kind, mag, duration, mode, rule.RayLimit(p))
		}
	}
	return b.String()
}

func TestSecondGameRowsReadTheFormulaSetUnderTheirOwnNumbers(t *testing.T) {
	var set SpellFormulaSet
	set.Spell[FormulaDamage][5] = []int32{90}
	set.Spell[FormulaRange][5] = []int32{7}
	set.Spell[FormulaPower][5] = []int32{40}
	for _, id := range []int{6, 12, 18} {
		set.Spell[FormulaDuration][id] = []int32{2000}
	}
	set.Spell[FormulaMagnitude][29] = []int32{-9}
	r := formulaRules(t, set)
	ice := secondCatalogRow(5)
	ice.MaxRange = 6
	if lo, hi := SpellDamagePair(r, ice, 100); lo != int64(ice.DamageMin)*3 || hi != int64(ice.DamageMax)*3 {
		t.Errorf("Ice Missile damage %d-%d, want three times its pair", lo, hi)
	}
	if got := spellRangeOf(r, ice, 100, rangeCast); got != int64(ice.MaxRange)+7 {
		t.Errorf("Ice Missile range %d, want its column plus 7", got)
	}
	if got := spellPowerUnder(r, ice, 10, 10); got != 40 {
		t.Errorf("Ice Missile power %d, want 40", got)
	}
	w := spWorld(t, 1, nil, spEnt(1, 2, 2))
	w.SetRules(r)
	poison := secondRow(SpellRule{ID: 6, Area: true, Distribution: 3, SpellDuration: 5, EffectKind: EffectHealth,
		EffectMode: EffectContinuous, EffectMagnitude: -2, EffectDuration: 128})
	invisible := secondRow(SpellRule{ID: 12, TargetsUnit: true, SpellDuration: 10})
	stone := secondRow(SpellRule{ID: 18, TargetsUnit: true, SpellDuration: 2})
	for _, c := range []struct {
		name string
		rule SpellRule
		want uint16
	}{{"Poison Cloud", poison, 128}, {"Invisibility", invisible, 3 * 16 * 2}} {
		if _, _, got, _ := w.pointEffect(0, c.rule, 70); got != c.want {
			t.Errorf("%s lasts %d at a factor of 2000, want %d", c.name, got, c.want)
		}
	}
	if _, _, got, _ := w.pointEffect(0, stone, 70); got != secondCurseCeiling(70, 0, false) {
		t.Errorf("Stone Curse's ceiling %d moved under a duration table", got)
	}
	if got := spellDurationOf(r, stone, 70, durationRecord); got != spellDurationOf(Rules{}, stone, 70, durationRecord) {
		t.Errorf("Stone Curse's record duration %d moved under a duration table", got)
	}
	slow := secondRow(SpellRule{ID: 29, TargetsUnit: true, SpellDuration: 1})
	if _, mag, _, _ := w.pointEffect(0, slow, 70); mag != -9 {
		t.Errorf("Slow, the second game's spell 29, has magnitude %d, want the table's -9", mag)
	}
	if DurationTableRule(poison) || !DurationTableRule(invisible) || DurationTableRule(stone) || !DurationTableRule(SpellRule{ID: 20}) {
		t.Error("the rows a duration table reaches")
	}
}

func TestSecondGameSpellNumbersWithoutAModAreTheCommittedOnes(t *testing.T) {
	path := filepath.Join("testdata", "secondspellnumbers.txt")
	got := secondSpellNumbers(t, Rules{})
	if *updateSecondSpellNumbers {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := range min(len(gl), len(wl)) {
			if gl[i] != wl[i] {
				t.Fatalf("line %d is %q, want %q", i+1, gl[i], wl[i])
			}
		}
		t.Fatalf("%d lines, want %d", len(gl), len(wl))
	}
}
