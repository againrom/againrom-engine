package modrt

import (
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

func TestSpellSwitchFixtureGivesTheSwitchPositionsAndSize(t *testing.T) {
	entry := mod.Load(filepath.Join("testdata", "mods", "spell-switch"))
	if entry.Err != nil {
		t.Fatal(entry.Err)
	}
	for _, base := range []string{"rom1-en", "rom1-ru", "rom2-en", "rom2-ru"} {
		res, err := Load([]mod.Entry{entry}, base, nil, Options{})
		if err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		d := res.Spells
		if g := d.Global; g.BookSize.Val != 23 || g.BookSizeMod != "spell-switch" || g.BookSizeFile != mod.SpellsFile {
			t.Errorf("%s book size %+v", base, g)
		}
		var got []string
		for _, r := range d.Rows {
			e := r.SpellEdit()
			got = append(got, strings.Join([]string{e.Name, map[bool]string{true: "off", false: "on"}[e.Off], string(rune('0' + e.Position))}, " "))
		}
		if want := "Fire_Ball off 0|Wall_of_Fire on 1|Fire_Arrow on 3"; strings.Join(got, "|") != want {
			t.Errorf("%s rows %q, want %q", base, got, want)
		}
	}
}

func TestSpellSwitchFieldsAreOneModsEach(t *testing.T) {
	src := "[global]\nbook_size = 20\n\n[[spell]]\ntarget = \"Heal\"\noff = true\n"
	entries := makeMods(t, map[string]map[string]string{
		"a": {"main.star": spellsMain, "data/spells.toml": src},
		"b": {"main.star": spellsMain, "data/spells.toml": src},
	})
	_, err := Load(entries, "rom1-en", nil, Options{})
	if err == nil || !strings.Contains(err.Error(), `data/spells.toml:2: book_size is already set by mod "a"`) {
		t.Errorf("%v", err)
	}
	src = "[[spell]]\ntarget = \"Heal\"\nbook_position = 2\n"
	entries = makeMods(t, map[string]map[string]string{
		"a": {"main.star": spellsMain, "data/spells.toml": src},
		"b": {"main.star": spellsMain, "data/spells.toml": src},
	})
	if _, err := Load(entries, "rom1-en", nil, Options{}); err == nil || !strings.Contains(err.Error(), `data/spells.toml:3: book_position of "Heal" is already set by mod "a"`) {
		t.Errorf("%v", err)
	}
}
