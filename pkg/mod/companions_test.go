package mod

import (
	"strings"
	"testing"
)

const goodJoin = `# one condition
[[join]]
key       = "mage"
companion = 22
chapter   = 30
building  = "tavern"
talk      = 22
`

func TestParseCompanionsReadsAJoinCondition(t *testing.T) {
	d, err := ParseCompanions("m", CompanionsFile, []byte(goodJoin))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Joins) != 1 || d.Empty() {
		t.Fatalf("%+v", d)
	}
	j := d.Joins[0]
	if j.Mod != "m" || j.Key != "mage" || j.Companion != 22 || j.Chapter != 30 || j.Building != BuildingTavern || j.Talk != 22 || j.Line != 2 {
		t.Fatalf("%+v", j)
	}
	if got, ok := d.Find(30, 22); !ok || got.Key != "mage" {
		t.Fatalf("Find: %+v %v", got, ok)
	}
	if _, ok := d.Find(40, 22); ok {
		t.Fatal("Find answered for a chapter the condition does not name")
	}
	if got := d.ForTalk(30, BuildingTavern, 22); len(got) != 1 {
		t.Fatalf("ForTalk: %+v", got)
	}
	for _, miss := range [][3]int{{30, 0, 21}, {40, 0, 22}} {
		if got := d.ForTalk(miss[0], BuildingTavern, miss[2]); len(got) != 0 {
			t.Fatalf("ForTalk%v: %+v", miss, got)
		}
	}
	if got := d.ForTalk(30, "shop", 22); len(got) != 0 {
		t.Fatalf("ForTalk answered for another building: %+v", got)
	}
}

func TestParseCompanionsRefusalsNameFileAndLine(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"outside a table", "key = \"a\"\n", "data/companions.toml:1: key is outside a [[join]] table"},
		{"unknown table", "[[perk]]\n", `data/companions.toml:1: unknown table "perk"`},
		{"single table", "[join]\n", `data/companions.toml:1: unknown table "join"`},
		{"unknown key", strings.Replace(goodJoin, "talk      = 22", "talk      = 22\nwhen = 1", 1), `data/companions.toml:8: unknown key "when"`},
		{"string number", strings.Replace(goodJoin, "companion = 22", `companion = "22"`, 1), "data/companions.toml:4: companion must be an integer, not a string"},
		{"zero", strings.Replace(goodJoin, "chapter   = 30", "chapter   = 0", 1), "data/companions.toml:5: chapter is 0; use 1 to 65535"},
		{"too large", strings.Replace(goodJoin, "talk      = 22", "talk      = 70000", 1), "data/companions.toml:7: talk is 70000; use 1 to 65535"},
		{"building", strings.Replace(goodJoin, `"tavern"`, `"shop"`, 1), `data/companions.toml:6: building is "shop"; only "tavern" conversations can start a join`},
		{"bad key", strings.Replace(goodJoin, `"mage"`, `"Mage"`, 1), `data/companions.toml:3: key "Mage" must be 1 to 32 of a-z`},
		{"missing talk", strings.Replace(goodJoin, "talk      = 22\n", "", 1), "data/companions.toml:2: [[join]] has no talk"},
		{"duplicate key", goodJoin + strings.Replace(goodJoin, "companion = 22", "companion = 25", 1), `data/companions.toml:9: join key "mage" is already used at line 2`},
		{"duplicate companion", goodJoin + strings.Replace(goodJoin, `"mage"`, `"other"`, 1), "data/companions.toml:9: companion 22 of chapter 30 already has a join condition at line 2"},
	}
	for _, c := range cases {
		_, err := ParseCompanions("m", CompanionsFile, []byte(c.src))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err %v, want %q", c.name, err, c.want)
		}
	}
}
