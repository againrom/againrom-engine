package mod

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/rules"
)

func TestModsDocSpellNamesAreTheCatalog(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "MODS.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, section, ok := strings.Cut(string(doc), "\n## Spell names\n")
	if !ok {
		t.Fatal("docs/MODS.md has no Spell names section")
	}
	section, _, _ = strings.Cut(section, "\n## ")
	var got []string
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "| `") {
			got = append(got, line)
		}
	}
	var want []string
	number := func(g rules.SpellGame, name string) string {
		if n, ok := rules.SpellNumber(g, name); ok {
			return fmt.Sprint(n)
		}
		return "-"
	}
	for _, name := range rules.SpellCatalog() {
		want = append(want, fmt.Sprintf("| `%s` | %s | %s |", name, number(rules.SpellGameFirst, name), number(rules.SpellGameSecond, name)))
	}
	if !slices.Equal(got, want) {
		t.Errorf("docs/MODS.md Spell names:\n%s\nwant the catalog:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestModsDocStatesEachFormulaKeyWithItsLimitsAndRefusal(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "MODS.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, section, ok := strings.Cut(string(doc), "\n## Spell formula tables\n")
	if !ok {
		t.Fatal("docs/MODS.md has no Spell formula tables section")
	}
	section, _, _ = strings.Cut(section, "\n## ")
	section = strings.Join(strings.Fields(section), " ")
	for f := rules.FormulaPower; f <= rules.FormulaMagnitude; f++ {
		lo, hi, entries := rules.FormulaBounds(f)
		_, row, _ := strings.Cut(section, fmt.Sprintf("| `%s` |", f))
		row, _, _ = strings.Cut(row, " | `")
		if want := fmt.Sprintf("| 1..%d | %d..%d |", entries, lo, hi); !strings.Contains(row, want) {
			t.Errorf("%s row %q lacks %q", f, row, want)
		}
		if refusal := fmt.Sprintf("so %s does not apply to it", f); f != rules.FormulaPower && !strings.Contains(section, refusal) {
			t.Errorf("the section lacks the refusal %q", refusal)
		}
	}
}

func TestMergeSpellsKeysRowsByTheCatalogName(t *testing.T) {
	parse := func(id, src string) SpellData {
		d, err := ParseSpells(id, SpellsFile, []byte(src), nil)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	byName := parse("a", "[[spell]]\ntarget = \"Fire_Ball\"\nmana = 20\n")
	byTarget := parse("b", "[[spell]]\ntarget = \"Fire Ball\"\nrange = 14\n\n[[spell]]\ntarget = \"Heal\"\nmana = 3\n")
	d := SpellData{Rows: append(byName.Rows, byTarget.Rows...)}
	got, overrides, err := MergeSpells(d, []string{"", "Fire Arrow", "Fire Ball", "Heal"})
	if err != nil || len(overrides) != 0 {
		t.Fatalf("merge: %v, overrides %v", err, overrides)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("merged %d rows, want Fire_Ball and Heal: %+v", len(got.Rows), got.Rows)
	}
	fb, heal := got.Rows[0], got.Rows[1]
	if fb.Target != "Fire_Ball" || fb.Mod != "a" || fb.Mana.Val != 20 || fb.Range.Val != 14 || !fb.Mana.Set || !fb.Range.Set {
		t.Errorf("Fire_Ball row %+v, want mod a's mana 20 and mod b's range 14", fb)
	}
	if heal.Target != "Heal" || heal.Mod != "b" || heal.Mana.Val != 3 || heal.Range.Set {
		t.Errorf("Heal row %+v", heal)
	}

	twice := SpellData{Rows: append(parse("a", "[[spell]]\ntarget = \"Fire_Ball\"\nmana = 20\n").Rows,
		parse("b", "[[spell]]\ntarget = \"Fire Ball\"\nmana = 9\n").Rows...)}
	if _, _, err := MergeSpells(twice, []string{"Fire Ball"}); err == nil || !strings.Contains(err.Error(), "neither loads after the other") {
		t.Errorf("two mods on one field without order: %v", err)
	}
	twice.Order = Precedence{"b": {"a"}}
	got, overrides, err = MergeSpells(twice, []string{"Fire Ball"})
	if err != nil || len(overrides) != 1 || got.Rows[0].Mana.Val != 9 {
		t.Errorf("b after a: %v, overrides %v, rows %+v", err, overrides, got.Rows)
	}
}
