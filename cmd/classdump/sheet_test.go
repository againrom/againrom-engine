package main

// Tests for the whole character sheet. Every fixture is a synthetic archive
// written to a temp dir with the wire primitives databin_test.go's binWriter
// already streams; no game install is read (golden rule 2).

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/databin"
	"againrom/pkg/mapload"
)

// sheetDataBin writes a data.bin whose Units and Humans collections each
// carry one written row with ten distinct statistic, skill and family
// cells — chosen so a column this file's printer puts in the wrong place, or
// fails to print at all, is a value a reader can see is wrong rather than a
// coincidence that happens to read right.
func sheetDataBin() []byte {
	w := &binWriter{}
	w.emptyGroup(2) // A: Shapes, Materials
	w.emptyGroup(1) // B: Magic
	w.emptyGroup(3) // C: Armors, Shields, Weapons
	w.emptyGroup(1) // D: MagicItems

	// E: Units — one written creature row, keyed by TypeID 210 / Face 0, the
	// pair data.FindUnit searches on.
	w.strArray(nil)
	w.u32(2)
	w.str("Creature")
	w.params(binRow(map[int]int32{
		0: 41, 1: 42, 2: 43, 3: 44, // Body, Agility, Mind, Spirit
		4: 100, 6: 30, // health maximum, mana maximum
		8: 12, 10: 9, // speed, scan range
		11: 5, 12: 9, // damage 5..9 (base 5, spread 4)
		14: 40, 15: 12, 16: 3, // to-hit, defence, absorption
		19: 51, 20: 52, 21: 53, 22: 54, 23: 55, // elemental Fire..Astral
		24: 61, 25: 62, 26: 63, 27: 64, 28: 65, // weapon-kind Blade..Shooting
		29: 210, 30: 0, // TypeID, Face -- data.FindUnit's own search key
		37: 77, //
	}))
	w.strs(make([]string, 2))

	// F: Humans — one written person row, keyed by TypeID 9, the column
	// data.FindHumanByType searches on. Its ten trailing strings are all
	// empty, so this person is BARE: no weapon, and no worn or carried item
	// for the campaign table's own new Armors and Shields collections to
	// resolve (verification that adding them moves nothing, below).
	w.strArray(nil)
	w.u32(2)
	w.str("Person")
	w.params(binRow(map[int]int32{
		0: 21, 1: 22, 2: 23, 3: 24, // Body, Agility, Mind, Spirit -- all under
		// StatCap (50), so the derived-stat graph's cap changes none of them
		6: 11, 8: 6, // speed, scan range
		10: 31, 11: 32, 12: 33, 13: 34, 14: 35, 15: 36, // Skill[0..5]
		16: 9,        // TypeID -- data.FindHumanByType's own search key
		19: 9, 20: 4, // attack charge/relax
	}))
	w.strs(make([]string, 10))

	w.emptyGroup(1) // G: Buildings
	w.emptyGroup(1) // H: Spells
	return w.b
}

func sheetMapBytes() []byte {
	return synth.ALM(synth.ALMOptions{
		Width: 32, Height: 32,
		Units: []synth.ALMUnit{
			{X: 0x0A80, Y: 0x0A80, ClassID: 210}, // units arm: resolves (TypeID 210, Face 0)
			{X: 0x0B80, Y: 0x0A80, ClassID: 9},   // humans-by-type arm: resolves (TypeID 9)
			{X: 0x0C80, Y: 0x0A80, ClassID: 220}, // units arm: matches no row at all
		},
	})
}

// sheetBlock returns placement index's own block from a -databin or
// -campaign run: the header line plus the six lines printSheetBlock always
// prints after it. Locating it by the header's own "sheet <index>" fields,
// rather than by fieldsOfLine's single-line search, is what lets a test
// address (say) placement 2's "Body" line without finding placement 0's.
func sheetBlock(t *testing.T, out string, index int) []string {
	t.Helper()
	lines := strings.Split(out, "\n")
	want := fmt.Sprintf("%d", index)
	for i, line := range lines {
		f := strings.Fields(line)
		if len(f) >= 2 && f[0] == "sheet" && f[1] == want {
			end := i + 7 // the header and the six lines below it
			if end > len(lines) {
				end = len(lines)
			}
			return lines[i:end]
		}
	}
	t.Fatalf("no sheet block for placement %d in:\n%s", index, out)
	return nil
}

// fieldsOfBlockLine finds the one line of block whose first field is head,
// scoped to a single placement's own lines rather than the whole report —
// every block repeats "Body", "Health" and the rest once per placement, so
// fieldsOfLine's whole-report search cannot tell them apart.
func fieldsOfBlockLine(t *testing.T, block []string, head string) []string {
	t.Helper()
	for _, line := range block {
		f := strings.Fields(line)
		if len(f) > 0 && f[0] == head {
			return f
		}
	}
	t.Fatalf("no %q line in block:\n%s", head, strings.Join(block, "\n"))
	return nil
}

func TestDataBinVerbPrintsTheWholeCharacterSheet(t *testing.T) {
	dir := t.TempDir()
	res := filepath.Join(dir, "world.res")
	if err := os.WriteFile(res, synth.Archive([]synth.File{{Path: "data/data.bin", Data: sheetDataBin()}}), 0o644); err != nil {
		t.Fatal(err)
	}
	mapPath := filepath.Join(dir, "scn.alm")
	if err := os.WriteFile(mapPath, sheetMapBytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := run([]string{"-databin", res, mapPath}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := out.String()

	if !strings.Contains(got, "character sheets: 3 placement(s) (FR-9)") {
		t.Fatalf("no sheet section header:\n%s", got)
	}
	for _, phrase := range []string{
		"'-' marks UNSTATED", "Weight has no column", "PERSON's XP is the units collection's own",
	} {
		if !strings.Contains(got, phrase) {
			t.Errorf("the legend is missing %q:\n%s", phrase, got)
		}
	}

	// Placement 0: the resolved CREATURE (FR-4a). Every sheet-sourced field is
	// its row's own column, unchanged, and the five weapon-keyed skill
	// positions repeat the weapon-kind family's own five numbers.
	b0 := sheetBlock(t, got, 0)
	if f := fieldsOfBlockLine(t, b0, "sheet"); !sameFields(f, []string{"sheet", "0", "band", "creature", "entry", `"Creature"`}) {
		t.Errorf("placement 0 header reads %q", f)
	}
	if f := fieldsOfBlockLine(t, b0, "Body"); !sameFields(f, []string{"Body", "41", "Agility", "42", "Mind", "43", "Spirit", "44"}) {
		t.Errorf("placement 0 statistics read %q", f)
	}
	if f := fieldsOfBlockLine(t, b0, "Health"); !sameFields(f, []string{"Health", "100/100", "Mana", "30/30"}) {
		t.Errorf("placement 0 pools read %q", f)
	}
	if f := fieldsOfBlockLine(t, b0, "Dmg"); !sameFields(f, []string{"Dmg", "5-9", "Absorb", "3", "Attack", "40", "Defense", "12"}) {
		t.Errorf("placement 0 combat numbers read %q", f)
	}
	if f := fieldsOfBlockLine(t, b0, "Skill"); !sameFields(f, []string{
		"Skill", "Blade", "61", "Axe", "62", "Bludgen", "63", "Pike", "64", "Shooting", "65",
	}) {
		t.Errorf("placement 0 skill positions read %q, want the weapon-kind columns (FR-4a)", f)
	}
	if f := fieldsOfBlockLine(t, b0, "Elemental"); !sameFields(f, []string{
		"Elemental", "Fire", "51", "Water", "52", "Air", "53", "Earth", "54", "Astral", "55",
	}) {
		t.Errorf("placement 0 elemental family reads %q", f)
	}
	if f := fieldsOfBlockLine(t, b0, "Weight"); !sameFields(f, []string{"Weight", "-", "XP", "77", "Sight", "9", "Speed", "12"}) {
		t.Errorf("placement 0 closing line reads %q, want its own units-collection XP stated, not dashed", f)
	}

	// Placement 2: a units-arm placement that resolves to NOTHING.
	b2 := sheetBlock(t, got, 2)
	if f := fieldsOfBlockLine(t, b2, "sheet"); f[3] != "-" {
		t.Errorf("placement 2's band reads %q, want unknown (-)", f[3])
	}
	if f := fieldsOfBlockLine(t, b2, "Body"); !sameFields(f, []string{"Body", "-", "Agility", "-", "Mind", "-", "Spirit", "-"}) {
		t.Errorf("placement 2 statistics read %q, want every sheet-sourced field dashed", f)
	}
	if f := fieldsOfBlockLine(t, b2, "Health"); !sameFields(f, []string{"Health", "100/100", "Mana", "0/0"}) {
		t.Errorf("placement 2 pools read %q, want the constructor's own provisional pair", f)
	}
	if f := fieldsOfBlockLine(t, b2, "Dmg"); !sameFields(f, []string{"Dmg", "0-0", "Absorb", "0", "Attack", "0", "Defense", "0"}) {
		t.Errorf("placement 2 combat numbers read %q", f)
	}
	if f := fieldsOfBlockLine(t, b2, "Skill"); !sameFields(f, []string{
		"Skill", "Blade", "-", "Axe", "-", "Bludgen", "-", "Pike", "-", "Shooting", "-",
	}) {
		t.Errorf("placement 2 skill positions read %q, want every position dashed", f)
	}
	if f := fieldsOfBlockLine(t, b2, "Elemental"); !sameFields(f, []string{
		"Elemental", "Fire", "-", "Water", "-", "Air", "-", "Earth", "-", "Astral", "-",
	}) {
		t.Errorf("placement 2 elemental family reads %q, want every position dashed", f)
	}
	if f := fieldsOfBlockLine(t, b2, "Weight"); !sameFields(f, []string{"Weight", "-", "XP", "0", "Sight", "5", "Speed", "10"}) {
		t.Errorf("placement 2 closing line reads %q, want XP off the entity's own zero, not dashed", f)
	}

	// Placement 1: the resolved PERSON. His four statistics and his six skill
	// levels are checked exactly, chosen under StatCap (50) and SkillCap (100)
	// so a capped or restored number and a raw one cannot coincide by
	// construction. His elemental family and his entity-carried numbers are
	// cross-checked against the SAME two functions this report itself reads —
	// pkg/mapload.PlacedSheets and pkg/mapload.FromALMWith — which is this
	// test's own way of asking "does the report say what those functions say"
	// without re-deriving either; re-deriving the graph is pkg/mapload's own
	// test's job, not this one's.
	b1 := sheetBlock(t, got, 1)
	if f := fieldsOfBlockLine(t, b1, "sheet"); !sameFields(f, []string{"sheet", "1", "band", "person", "entry", `"Person"`}) {
		t.Errorf("placement 1 header reads %q", f)
	}
	if f := fieldsOfBlockLine(t, b1, "Body"); !sameFields(f, []string{"Body", "21", "Agility", "22", "Mind", "23", "Spirit", "24"}) {
		t.Errorf("placement 1 statistics read %q, want its row's own, under StatCap", f)
	}
	if f := fieldsOfBlockLine(t, b1, "Skill"); !sameFields(f, []string{
		"Skill", "Blade", "32", "Axe", "33", "Bludgen", "34", "Pike", "35", "Shooting", "36",
	}) {
		t.Errorf("placement 1 skill positions read %q, want its row's own levels (FR-5, plan DD-7)", f)
	}
	if f := fieldsOfBlockLine(t, b1, "Weight"); f[1] != "-" || f[3] != "-" {
		t.Errorf("placement 1 closing line reads %q, want Weight AND XP both dashed (FR-9's own PERSON clause)", f)
	}

	m, err := alm.Open(sheetMapBytes())
	if err != nil {
		t.Fatal(err)
	}
	file, err := databin.Parse(sheetDataBin())
	if err != nil {
		t.Fatal(err)
	}
	tbl := &mapload.Table{Units: file.Collection(databin.Units), Humans: file.Collection(databin.Humans)}

	sheets := mapload.PlacedSheets(m, tbl)
	sheet1, ok := sheets[1]
	if !ok {
		t.Fatal("mapload.PlacedSheets carries no entry for placement 1")
	}
	wantElemental := []string{"Elemental",
		"Fire", fmt.Sprintf("%d", sheet1.Elemental[0]), "Water", fmt.Sprintf("%d", sheet1.Elemental[1]),
		"Air", fmt.Sprintf("%d", sheet1.Elemental[2]), "Earth", fmt.Sprintf("%d", sheet1.Elemental[3]),
		"Astral", fmt.Sprintf("%d", sheet1.Elemental[4])}
	if f := fieldsOfBlockLine(t, b1, "Elemental"); !sameFields(f, wantElemental) {
		t.Errorf("placement 1 elemental family reads %q, want PlacedSheets' own %q", f, wantElemental)
	}

	world, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	ents := world.Entities()
	if len(ents) != 3 {
		t.Fatalf("the world holds %d entities for 3 placements", len(ents))
	}
	e := ents[1]
	wantHealth := []string{"Health", fmt.Sprintf("%d/%d", e.HP, e.MaxHP), "Mana", fmt.Sprintf("%d/%d", e.Mana, e.MaxMana)}
	if f := fieldsOfBlockLine(t, b1, "Health"); !sameFields(f, wantHealth) {
		t.Errorf("placement 1 pools read %q, want the built world's own %q", f, wantHealth)
	}
	wantCombat := []string{"Dmg", fmt.Sprintf("%d-%d", e.DamageBase, e.DamageBase+e.DamageSpread),
		"Absorb", fmt.Sprintf("%d", e.Absorption), "Attack", fmt.Sprintf("%d", e.ToHit), "Defense", fmt.Sprintf("%d", e.Defence)}
	if f := fieldsOfBlockLine(t, b1, "Dmg"); !sameFields(f, wantCombat) {
		t.Errorf("placement 1 combat numbers read %q, want the built world's own %q", f, wantCombat)
	}
	wantClose := []string{"Weight", "-", "XP", "-", "Sight", fmt.Sprintf("%d", e.ScanRange), "Speed", fmt.Sprintf("%d", e.Speed)}
	if f := fieldsOfBlockLine(t, b1, "Weight"); !sameFields(f, wantClose) {
		t.Errorf("placement 1 closing line reads %q, want %q", f, wantClose)
	}
}

// sheetCampaignRoot lays a synthetic asset root shaped like campaign_test.go's
// own campaignRoot, over sheetDataBin() so the same known creature and person
// rows are reachable through the campaign container's own map.
func sheetCampaignRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, b []byte) {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("scenario.res", synth.Archive([]synth.File{
		{Path: "10.alm", Data: sheetMapBytes()},
		{Path: "npc.reg", Data: synth.NPCReg(nil)},
	}))
	write("world.res", synth.Archive([]synth.File{{Path: "data/data.bin", Data: sheetDataBin()}}))
	return dir
}

func TestCampaignVerbPrintsTheSameSheetBlockAsDataBin(t *testing.T) {
	dbDir := t.TempDir()
	res := filepath.Join(dbDir, "world.res")
	if err := os.WriteFile(res, synth.Archive([]synth.File{{Path: "data/data.bin", Data: sheetDataBin()}}), 0o644); err != nil {
		t.Fatal(err)
	}
	mapPath := filepath.Join(dbDir, "scn.alm")
	if err := os.WriteFile(mapPath, sheetMapBytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	var dbOut bytes.Buffer
	if err := run([]string{"-databin", res, mapPath}, &dbOut); err != nil {
		t.Fatalf("-databin run: %v", err)
	}

	var campOut bytes.Buffer
	if err := run([]string{"-campaign", sheetCampaignRoot(t), "10.alm"}, &campOut); err != nil {
		t.Fatalf("-campaign run: %v", err)
	}
	got := campOut.String()
	if !strings.Contains(got, "character sheets: 3 placement(s) (FR-9)") {
		t.Fatalf("-campaign printed no sheet section:\n%s", got)
	}

	for _, idx := range []int{0, 1, 2} {
		dbBlock := strings.Join(sheetBlock(t, dbOut.String(), idx), "\n")
		campBlock := strings.Join(sheetBlock(t, got, idx), "\n")
		if dbBlock != campBlock {
			t.Errorf("placement %d: -databin printed:\n%s\n-campaign printed:\n%s", idx, dbBlock, campBlock)
		}
	}
}

// TestCampaignVerbDocNoLongerClaimsCountsAndCellsOnly pins tasks.md's own
// "Done when" clause: the named-map path prints an entry name and a
// definition row's own values now, so the verb's doc must no longer claim it
// prints counts and cells only across the whole verb — only its own count
// census still does.
func TestCampaignVerbDocNoLongerClaimsCountsAndCellsOnly(t *testing.T) {
	src, err := os.ReadFile("campaign.go")
	if err != nil {
		t.Fatal(err)
	}
	got := string(src)
	if !strings.Contains(got, "ITS OWN CENSUS PRINTS COUNTS AND CELLS ONLY") {
		t.Error(`campaign.go no longer scopes the "counts and cells only" claim to the census alone`)
	}
	if strings.Contains(got, "It prints COUNTS AND CELLS ONLY") {
		t.Error(`campaign.go still claims the WHOLE verb prints counts and cells only`)
	}
}
