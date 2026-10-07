package mod

import (
	"errors"
	"strings"
	"testing"
)

const goodCharacters = `# one character
[[character]]
target = "NPC06"
name   = "character.girl"
kind   = "A_PeasantGuard"
face   = 9
strip  = ["armour", "shield"]

[[character]]
target = "PC_Naira"
name   = "character.naira"
`

func TestParseCharactersReadsRows(t *testing.T) {
	text := textOf(map[string]string{"character.girl": "Archer girl", "character.naira": "Naira"})
	d, err := ParseCharacters("archer-girl", CharactersFile, []byte(goodCharacters), text, "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Rows) != 2 || d.Empty() {
		t.Fatalf("rows %d", len(d.Rows))
	}
	r := d.Rows[0]
	if r.Mod != "archer-girl" || r.Target != "NPC06" || r.Name != "Archer girl" || r.NameKey != "character.girl" ||
		r.NameLine != 4 || r.Kind != "A_PeasantGuard" || r.KindLine != 5 || r.Face != 9 || len(r.Strip) != 2 || r.Strip[0] != StripArmour || r.Strip[1] != StripShield || r.Line != 2 {
		t.Fatalf("row %+v", r)
	}
	if n := d.Rows[1]; n.Target != "PC_Naira" || n.Name != "Naira" || n.Kind != "" || n.Face != 0 || len(n.Strip) != 0 {
		t.Fatalf("row %+v", n)
	}
	if !(CharacterData{}).Empty() {
		t.Fatal("zero data is not empty")
	}
}

func TestParseCharactersRefusalsNameFileAndLine(t *testing.T) {
	row := func(extra string) string { return "[[character]]\ntarget = \"NPC06\"\n" + extra }
	cases := []struct {
		name, src string
		line      int
		want      string
	}{
		{"no target", "[[character]]\nname = \"n\"\n", 1, "has no target"},
		{"changes nothing", row(""), 1, `of "NPC06" changes nothing`},
		{"unknown key", row("hair = 2\n"), 3, `unknown key "hair"`},
		{"target type", "[[character]]\ntarget = 3\n", 2, "target must be a string"},
		{"empty target", "[[character]]\ntarget = \"  \"\nname = \"n\"\n", 2, "target is empty"},
		{"missing text", row("name = \"zzz\"\n"), 3, `text key "zzz" is not in text/en/strings.toml`},
		{"long name", row("name = \"long\"\n"), 3, "has 45 characters; a character name has 1 to 40"},
		{"face type", row("face = \"nine\"\n"), 3, "face must be an integer"},
		{"face range", row("face = 200\n"), 3, "outside 1..127"},
		{"face zero", row("face = 0\n"), 3, "outside 1..127"},
		{"strip type", row("strip = \"armour\"\n"), 3, "strip must be an array"},
		{"strip element", row("strip = [3]\n"), 3, "use group names as strings"},
		{"strip group", row("strip = [\"boots\"]\n"), 3, `unknown strip group "boots" (groups: weapon, shield, armour)`},
		{"strip twice", row("strip = [\"armour\", \"armour\"]\n"), 3, `strip names "armour" twice`},
		{"strip empty", row("strip = []\n"), 3, "strip is empty"},
		{"stray key", "target = \"a\"\n", 1, "outside a [[character]] table"},
		{"unknown table", "[[item]]\nkey = \"a\"\n", 1, `unknown table "item" (this file holds [[character]])`},
		{"plain table", "[character]\ntarget = \"a\"\n", 1, `unknown table "character"`},
		{"syntax", "[[character]]\ntarget\n", 2, "expected key = value"},
	}
	text := textOf(map[string]string{"long": strings.Repeat("x", 45)})
	for _, c := range cases {
		_, err := ParseCharacters("m", CharactersFile, []byte(c.src), text, "en")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		var fe *ItemFileError
		if !errors.As(err, &fe) || fe.File != CharactersFile || fe.Line != c.line {
			t.Errorf("%s: %v is not %s:%d", c.name, err, CharactersFile, c.line)
		}
	}
}

func TestParseCharactersNamesTheLanguageOfTheText(t *testing.T) {
	src := "[[character]]\ntarget = \"a\"\nname = \"k\"\n"
	_, err := ParseCharacters("m", CharactersFile, []byte(src), textOf(nil), "ru")
	if err == nil || !strings.Contains(err.Error(), "text/ru/strings.toml (nor in text/en/strings.toml)") {
		t.Fatalf("%v", err)
	}
}
