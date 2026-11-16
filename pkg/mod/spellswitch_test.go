package mod

import (
	"strings"
	"testing"

	"againrom/pkg/rules"
)

func TestParseSpellsReadsTheSwitchPositionAndBookSize(t *testing.T) {
	src := "[global]\nbook_size = 23\n\n[[spell]]\ntarget = \"Fire_Ball\"\noff = true\n\n[[spell]]\ntarget = \"Wall_of_Fire\"\nbook_position = 1\noff = false\n"
	d, err := ParseSpells("m", SpellsFile, []byte(src), nil)
	if err != nil {
		t.Fatal(err)
	}
	g := d.Global
	if !g.BookSize.Set || g.BookSize.Val != 23 || g.BookSize.Line != 2 || g.BookSizeMod != "m" || g.BookSizeFile != SpellsFile {
		t.Errorf("book size %+v", g)
	}
	if keys := g.Globals(); len(keys) != 1 || keys[0].Key != "book_size" || d.Empty() {
		t.Errorf("globals %+v", keys)
	}
	if e := d.Rows[0].SpellEdit(); e != (rules.SpellEdit{Name: "Fire_Ball", Off: true}) || d.Rows[0].Off.Line != 6 {
		t.Errorf("off row %+v", d.Rows[0])
	}
	if e := d.Rows[1].SpellEdit(); e != (rules.SpellEdit{Name: "Wall_of_Fire", On: true, Position: 1}) || d.Rows[1].BookPosition.Line != 10 {
		t.Errorf("position row %+v", d.Rows[1])
	}
	m, _, err := MergeSpells(d, []string{"Fire Arrow", "Fire Ball", "Wall of Fire"})
	if err != nil || len(m.Rows) != 2 || !m.Rows[0].Off.Val || m.Rows[1].BookPosition.Val != 1 || !m.Global.BookSize.Set {
		t.Errorf("merged %+v %v", m, err)
	}
}

func TestSpellSwitchRefusalsNameTheLineAndTheLimit(t *testing.T) {
	for name, c := range map[string]struct{ src, want string }{
		"size high":         {"[global]\nbook_size = 25\n", "data/spells.toml:2: book_size is 25, outside 1..24"},
		"size zero":         {"[global]\nbook_size = 0\n", "data/spells.toml:2: book_size is 0, outside 1..24"},
		"position high":     {"[[spell]]\ntarget = \"Heal\"\nbook_position = 29\n", "data/spells.toml:3: book_position is 29, outside 1..24"},
		"off with position": {"[[spell]]\ntarget = \"Heal\"\noff = true\nbook_position = 2\n", "data/spells.toml:4: Heal is off, so book_position does not apply to it"},
		"off not a bool":    {"[[spell]]\ntarget = \"Heal\"\noff = 1\n", "data/spells.toml:3: off"},
	} {
		_, err := ParseSpells("m", SpellsFile, []byte(c.src), nil)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}
