package game

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
)

func charRow(name string, typ, face, gender int32, cells ...string) dbEntry {
	p := make([]int32, data.HumanSlotsRequired+7)
	p[data.HumanSlotTypeID], p[data.HumanSlotFace], p[data.HumanSlotGender] = typ, face, gender
	return dbEntry{name: name, params: p, strings: slices.Clone(cells)}
}

// characterFront is a front end whose Humans collection holds a four-tier
// family to edit, a peasant kind of the same tiers, a one-row kind and a
// bystander.
func characterFront() *FrontEnd {
	rows := dbCollection{
		{},
		charRow("NPC06_1", 14, 3, 1, "Bow", "Shield", "Helm", "Mail", "Boots", "", "", "", "", ""),
		charRow("NPC06_2", 14, 3, 1, "Bow", "Shield", "Helm", "Mail", "Boots", "", "", "", "", ""),
		charRow("NPC07_1", 15, 4, 1, "Crossbow", "", "Helm", "Mail"),
		charRow("A_Guard1", 14, 9, 1, "Bow"),
		charRow("A_Guard2", 14, 9, 1, "Bow"),
		charRow("U_Woman", 1, 12, 1),
		{name: "Short", params: []int32{1, 2, 3}},
	}
	f := &FrontEnd{}
	f.Table = &mapload.Table{Humans: rows}
	f.Humans = rows
	return f
}

func charRows(rows ...mod.CharacterRow) mod.CharacterData {
	for i := range rows {
		rows[i].Mod, rows[i].File, rows[i].Line = "m", mod.CharactersFile, 3+i
	}
	return mod.CharacterData{Rows: rows}
}

func TestSetModCharactersEditsTheNamedFamilyOnly(t *testing.T) {
	f := characterFront()
	before := f.Table.Humans
	err := f.SetModCharacters(charRows(mod.CharacterRow{Target: "NPC06", Name: "Archer girl", Kind: "A_Guard", Strip: []string{mod.StripArmour}}))
	if err != nil {
		t.Fatal(err)
	}
	after := f.Table.Humans
	if f.Humans != after {
		t.Fatal("the install's Humans collection is not the edited one")
	}
	if after.Len() != before.Len() {
		t.Fatal("the length changed")
	}
	for i := 0; i < before.Len(); i++ {
		edited := i == 1 || i == 2
		same := slices.Equal(before.EntryParams(i), after.EntryParams(i)) && slices.Equal(before.EntryStrings(i), after.EntryStrings(i))
		if edited == same || before.EntryName(i) != after.EntryName(i) {
			t.Fatalf("row %d %q: edited=%v unchanged=%v", i, before.EntryName(i), edited, same)
		}
	}
	for _, i := range []int{1, 2} {
		p := after.EntryParams(i)
		if p[data.HumanSlotTypeID] != 14 || p[data.HumanSlotFace] != 9 || p[data.HumanSlotGender] != 1 {
			t.Fatalf("row %d class and figure columns %v", i, p[data.HumanSlotTypeID:data.HumanSlotGender+1])
		}
		if got := after.EntryStrings(i); got[0] != "Bow" || got[1] != "Shield" || slices.ContainsFunc(got[2:], func(s string) bool { return s != "" }) {
			t.Fatalf("row %d cells %q", i, got)
		}
	}
	for row, want := range map[string]string{"NPC06_1": "Archer girl", "NPC06_2": "Archer girl"} {
		if got, ok := f.Table.Mods.CharacterName(row); !ok || got != want {
			t.Fatalf("name of %s is %q, %v", row, got, ok)
		}
	}
	for _, row := range []string{"NPC07_1", "A_Guard1", "NPC06", ""} {
		if got, ok := f.Table.Mods.CharacterName(row); ok {
			t.Fatalf("%q is named %q", row, got)
		}
	}
}

func TestSetModCharactersTargetsKindsFacesAndStripGroups(t *testing.T) {
	f := characterFront()
	err := f.SetModCharacters(charRows(
		mod.CharacterRow{Target: "NPC07_1", Kind: "U_Woman", Face: 20, Strip: []string{mod.StripWeapon}},
		mod.CharacterRow{Target: "NPC06_2", Kind: "A_Guard1", Strip: []string{mod.StripShield}},
	))
	if err != nil {
		t.Fatal(err)
	}
	h := f.Table.Humans
	if p := h.EntryParams(3); p[data.HumanSlotTypeID] != 1 || p[data.HumanSlotFace] != 20 || p[data.HumanSlotGender] != 1 {
		t.Fatalf("a one-row kind and an explicit face: %v", p[data.HumanSlotTypeID:data.HumanSlotGender+1])
	}
	if got := h.EntryStrings(3); got[0] != "" || got[2] != "Helm" || got[3] != "Mail" {
		t.Fatalf("weapon strip: %q", got)
	}
	if p := h.EntryParams(2); p[data.HumanSlotFace] != 9 {
		t.Fatalf("a kind family takes the row of the target's tier: face %d", p[data.HumanSlotFace])
	}
	if got := h.EntryStrings(2); got[1] != "" || got[0] != "Bow" || got[2] != "Helm" {
		t.Fatalf("shield strip: %q", got)
	}
	if !slices.Equal(h.EntryParams(1), characterFront().Table.Humans.EntryParams(1)) {
		t.Fatal("NPC06_1 is not named by any edit")
	}
}

func TestSetModCharactersMergesEditsOfOneRowInLoadOrder(t *testing.T) {
	f := characterFront()
	err := f.SetModCharacters(charRows(
		mod.CharacterRow{Target: "NPC06", Face: 5, Name: "First"},
		mod.CharacterRow{Target: "NPC06_1", Face: 6, Name: "Second", Strip: []string{mod.StripArmour}},
	))
	if err != nil {
		t.Fatal(err)
	}
	h := f.Table.Humans
	if h.EntryParams(1)[data.HumanSlotFace] != 6 || h.EntryParams(2)[data.HumanSlotFace] != 5 {
		t.Fatalf("faces %d %d", h.EntryParams(1)[data.HumanSlotFace], h.EntryParams(2)[data.HumanSlotFace])
	}
	if got, _ := f.Table.Mods.CharacterName("NPC06_1"); got != "Second" {
		t.Fatalf("the later edit names the row %q", got)
	}
	if got, _ := f.Table.Mods.CharacterName("NPC06_2"); got != "First" {
		t.Fatalf("the earlier edit names the other row %q", got)
	}
}

func TestSetModCharactersRefusalsNameModFileAndLine(t *testing.T) {
	cases := []struct {
		name string
		row  mod.CharacterRow
		want string
	}{
		{"no row", mod.CharacterRow{Target: "NPC99", Face: 5}, `mod "m": data/characters.toml:3: no definition row is named "NPC99"`},
		{"a family needs its tier", mod.CharacterRow{Target: "NPC0", Face: 5}, `no definition row is named "NPC0"`},
		{"no kind", mod.CharacterRow{Target: "NPC06", Kind: "Nobody", KindLine: 7}, `data/characters.toml:7: no definition row is named "Nobody"`},
		{"short row", mod.CharacterRow{Target: "Short", Face: 5}, `the row "Short" carries no class and figure columns to edit`},
		{"short kind", mod.CharacterRow{Target: "NPC06", Kind: "Short", KindLine: 7}, `the row "Short" carries no class and figure columns to take`},
	}
	for _, c := range cases {
		f := characterFront()
		err := f.SetModCharacters(charRows(c.row))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	var none *FrontEnd = &FrontEnd{}
	if err := none.SetModCharacters(charRows(mod.CharacterRow{Target: "a", Face: 1})); err == nil {
		t.Error("a front end without a table accepted an edit")
	}
}

func TestSetModCharactersDrawsANameInTheInstallAlphabet(t *testing.T) {
	f := characterFront()
	f.Font = resolved(&text.Font{Selector: text.SelectorConverting}, nil)
	row := mod.CharacterRow{Target: "NPC06", Name: "Лучница", NameLine: 4}
	if err := f.SetModCharacters(charRows(row)); err != nil {
		t.Fatal(err)
	}
	want, err := encodeSaveLabel("Лучница", text.SelectorConverting)
	if err != nil || want == "Лучница" {
		t.Fatalf("the install alphabet did not change the text: %q %v", want, err)
	}
	if got, _ := f.Table.Mods.CharacterName("NPC06_1"); got != want {
		t.Fatalf("name %q, want %q", got, want)
	}
	g := characterFront()
	g.Font = resolved(&text.Font{Selector: text.SelectorConverting}, nil)
	row.Name = "日本"
	err = g.SetModCharacters(charRows(row))
	if err == nil || !strings.Contains(err.Error(), `mod "m": data/characters.toml:4: the name "日本" cannot be drawn by this install`) {
		t.Fatalf("an undrawable name: %v", err)
	}
}

func TestSetModCharactersWithNothingLeavesTheTableAlone(t *testing.T) {
	f := characterFront()
	if err := f.SetModCharacters(mod.CharacterData{}); err != nil {
		t.Fatal(err)
	}
	if _, edited := f.Table.Humans.(*data.EditedRows); edited || len(f.Table.Mods.Characters) != 0 {
		t.Fatal("an empty edit changed the table")
	}
}

func TestSetModsKeepsTheCharactersTheOtherCallAdded(t *testing.T) {
	f := characterFront()
	if err := f.SetModCharacters(charRows(mod.CharacterRow{Target: "NPC06", Name: "Archer girl"})); err != nil {
		t.Fatal(err)
	}
	if err := f.SetMods(sim.Rules{}, mod.Set{Base: "rom1-en"}, false); err != nil {
		t.Fatal(err)
	}
	if got, ok := f.Table.Mods.CharacterName("NPC06_1"); !ok || got != "Archer girl" {
		t.Fatalf("SetMods dropped the names: %q %v", got, ok)
	}
}

func TestModCharacterNameWithoutAnInstall(t *testing.T) {
	var in *InstallResources
	if _, ok := in.modCharacterName("NPC06_1"); ok {
		t.Fatal("a nil install names a row")
	}
	if _, ok := (&InstallResources{}).modCharacterName("NPC06_1"); ok {
		t.Fatal("an install without a table names a row")
	}
}
