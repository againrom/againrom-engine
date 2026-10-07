package main

import (
	"bytes"
	"os"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

type withdrawalRow struct {
	name    string
	params  []int32
	strings []string
}

type withdrawalRows []withdrawalRow

func (r withdrawalRows) Len() int                    { return len(r) }
func (r withdrawalRows) EntryName(i int) string      { return r[i].name }
func (r withdrawalRows) EntryParams(i int) []int32   { return r[i].params }
func (r withdrawalRows) EntryStrings(i int) []string { return r[i].strings }

type withdrawalScales []struct {
	name string
	data []float64
}

func (s withdrawalScales) Len() int                     { return len(s) }
func (s withdrawalScales) EntryName(i int) string       { return s[i].name }
func (s withdrawalScales) EntryDoubles(i int) []float64 { return s[i].data }

func TestWithdrawalVerbArgumentShape(t *testing.T) {
	if err := run([]string{withdrawFlag}, &bytes.Buffer{}); err == nil {
		t.Fatal("a bare withdrawal verb was accepted")
	}
	if err := run([]string{withdrawFlag, "a", "b"}, &bytes.Buffer{}); err == nil {
		t.Fatal("a withdrawal verb with a map argument was accepted")
	}
}

func TestWithdrawalRangedClassificationReadsTheResolvedAttackNotReach(t *testing.T) {
	identity := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}
	scales := withdrawalScales{{data: identity}}
	weapon := func(attack, reach int32) []int32 {
		row := make([]int32, 14)
		row[5], row[9], row[12], row[13] = attack, reach, -1, -1
		return row
	}
	weapons := withdrawalRows{
		{},
		{name: "Close Shoot", params: weapon(data.SkillShoot, 1)},
		{name: "Long Blade", params: weapon(data.SkillBlade, 9)},
	}
	for _, tc := range []struct {
		name       string
		literal    string
		wantAttack int32
		wantRanged bool
	}{
		{"shoot at reach one", "Close Shoot", data.SkillShoot, true},
		{"blade at reach nine", "Long Blade", data.SkillBlade, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table := &mapload.Table{
				Units:   withdrawalRows{{}, {strings: []string{tc.literal}}},
				Weapons: weapons, Shapes: scales, Materials: scales,
			}
			attack, ranged := resolvedUnitAttack(table, 1)
			if attack != tc.wantAttack || ranged != tc.wantRanged {
				t.Fatalf("resolved attack = %d ranged=%t, want %d/%t", attack, ranged, tc.wantAttack, tc.wantRanged)
			}
		})
	}
}

// The release gate runs this once for each lawful root. Exact placement totals
// differ between EN and RU; the definition table and resolved SkillShoot
// classification of every shipped positive-Withdraw placement do not.
func TestReleaseWithdrawalPopulationAndMissionWitness(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: ranged-withdrawal census needs a lawful install")
	}
	c, err := openCampaign(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := censusWithdrawal(c, root)
	if err != nil {
		t.Fatal(err)
	}
	if got.parameterised != 56 || got.withdrawDefs != 12 || got.wimpyDefs != 24 {
		t.Errorf("definition census = %d parameterised, %d Withdraw, %d Wimpy; want 56, 12, 24",
			got.parameterised, got.withdrawDefs, got.wimpyDefs)
	}
	wantPopulation := map[[2]int][2]int{
		{38, 8094}: {1551, 3464}, // EN: campaign plus ten loose maps
		{34, 3991}: {817, 1754},  // RU: campaign plus six loose maps
	}
	want, ok := wantPopulation[[2]int{got.maps, got.placements}]
	if !ok {
		t.Errorf("walked %d maps / %d placements, want a complete EN or RU corpus", got.maps, got.placements)
	} else if got.withdrawPlaced != want[0] || got.wimpyPlaced != want[1] {
		t.Errorf("threshold placements = %d Withdraw / %d Wimpy, want %d / %d for this root",
			got.withdrawPlaced, got.wimpyPlaced, want[0], want[1])
	}
	if got.withdrawRangedDefs != got.withdrawDefs {
		t.Errorf("resolved SkillShoot EquipItem definitions = %d of %d", got.withdrawRangedDefs, got.withdrawDefs)
	}
	if got.withdrawPlaced == 0 || got.withdrawRangedPlaced != got.withdrawPlaced {
		t.Errorf("resolved SkillShoot EquipItem placements = %d of %d", got.withdrawRangedPlaced, got.withdrawPlaced)
	}
	if got.maxHP != 2000 || got.maxThresholdHP != 1024 {
		t.Errorf("maximum HP = %d overall / %d threshold-active, want 2000 / 1024",
			got.maxHP, got.maxThresholdHP)
	}
	t.Logf("maps=%d placements=%d Withdraw=%d ranged-defs=%d ranged-placements=%d Wimpy=%d maxHP=%d thresholdMaxHP=%d",
		got.maps, got.placements, got.withdrawPlaced, got.withdrawRangedDefs,
		got.withdrawRangedPlaced, got.wimpyPlaced, got.maxHP, got.maxThresholdHP)
}
