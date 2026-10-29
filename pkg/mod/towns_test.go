package mod

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/town"
)

// Every key of a [[town]] table reads into the catalog's entry format, and
// the record names exactly the fields it sets, each at its line.
func TestTownsReadTheEntryFormat(t *testing.T) {
	src := `# East
[[town]]
id          = 2
description = "towns/east.json"
hooks       = "chapters"
offers      = "chapter-keys"
map-point   = { point = [430, 230], rect = [410, 205, 40, 30] }
name        = "East"
opens-after = 20
missions    = [31]
cite        = ["owner"]

[[town]]
id  = 1
off = true

[[town]]
id        = 3
map-point = { object = 3 }
`
	rs, _, err := ParseTowns("east", TownsFile, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 3 {
		t.Fatalf("%d records", len(rs))
	}
	want := town.Entry{ID: 2, Description: "towns/east.json", Hooks: "chapters", Offers: "chapter-keys",
		MapPoint: town.MapPoint{Point: []int{430, 230}, Rect: []int{410, 205, 40, 30}}, Name: "East", OpensAfter: 20,
		Missions: []int{31}, Cite: []string{"owner"}}
	if !reflect.DeepEqual(rs[0].Entry, want) || rs[0].Line != 2 || rs[0].Mod != "east" || rs[0].File != TownsFile {
		t.Fatalf("record %+v, want %+v", rs[0], want)
	}
	var names []string
	for _, f := range rs[0].Fields {
		names = append(names, f.Name)
	}
	if !reflect.DeepEqual(names, []string{"description", "hooks", "offers", "map-point", "name", "opens-after", "missions", "cite"}) ||
		rs[0].LineOf("missions") != 10 || rs[0].LineOf("id") != 2 {
		t.Fatalf("fields %v", names)
	}
	if !rs[1].Off || len(rs[1].Fields) != 1 || rs[1].Fields[0].Name != OffField {
		t.Fatalf("switch-off %+v", rs[1])
	}
	if rs[2].Entry.MapPoint.Object != 3 {
		t.Fatalf("object point %+v", rs[2].Entry.MapPoint)
	}
	var e town.Entry
	rs[2].Apply(&e)
	if e.ID != 3 || e.MapPoint.Object != 3 || e.Name != "" {
		t.Fatalf("applied %+v", e)
	}
	if !IsTownDescriptionFile("towns/East.JSON") || IsTownDescriptionFile("rom1") || ModDescriptionName("east", "towns/east.json") != "east:towns/east.json" {
		t.Fatal("description file names")
	}
}

// A [[tavern]] table names its row by town, chapter and npc and sets
// mission or off; Apply writes the key and only the fields it sets.
func TestTownsReadTavernRows(t *testing.T) {
	src := `[[town]]
id   = 1
name = "West"

[[tavern]]
town    = 1
chapter = 50
npc     = 41
mission = 0

[[tavern]]
town    = 1
chapter = 50
npc     = 25
off     = true
`
	towns, rows, err := ParseTowns("m", TownsFile, []byte(src))
	if err != nil || len(towns) != 1 || len(rows) != 2 {
		t.Fatalf("%d towns, %d rows: %v", len(towns), len(rows), err)
	}
	if !rows[0].Row.Equal(town.ChapterOffer(1, 50, 41, 0)) || rows[0].Table != town.TavernTable || rows[0].Line != 5 || len(rows[0].Fields) != 1 ||
		rows[0].Fields[0] != (PatchField{Name: "mission", Line: 9}) {
		t.Fatalf("talk %+v", rows[0])
	}
	if !rows[1].Off || len(rows[1].Fields) != 1 || rows[1].Fields[0].Name != OffField {
		t.Fatalf("switch-off %+v", rows[1])
	}
	r := town.OfferRow{Kind: town.OfferMission, Topic: 51}
	rows[1].Apply(&r)
	if !r.Equal(town.ChapterOffer(1, 50, 25, 51)) {
		t.Fatalf("an off record changed the mission: %+v", r)
	}
	if d := (TownData{Offers: rows}); d.Empty() || d.First() != rows[0].Source {
		t.Fatal("a file of tavern rows alone reads as empty")
	}
}

// Every key of an [[inn]] table reads into the inn row format; a record names
// the fields it sets, and Apply sets those alone.
func TestTownsReadInnRows(t *testing.T) {
	src := `[[inn]]
topic   = 38
npc     = 2110
kind    = 3
town    = 2
stages  = [30, 40]
after   = 20
zero    = [927]
nonzero = [533]
one     = [774]
any     = [970, 971]

[[inn]]
topic   = 31
npc     = 2108
nonzero = [533]

[[inn]]
topic = 39
npc   = 2110
off   = true
`
	_, inns, err := ParseTowns("m", TownsFile, []byte(src))
	if err != nil || len(inns) != 3 {
		t.Fatalf("%d inn rows: %v", len(inns), err)
	}
	want := town.OfferRow{Kind: 3, Topic: 38, NPC: 2110, Town: 2, Stages: []int{30, 40}, After: 20,
		Zero: []int{927}, Nonzero: []int{533}, One: []int{774}, Any: []int{970, 971}}
	if !reflect.DeepEqual(inns[0].Row, want) || inns[0].Line != 1 || len(inns[0].Fields) != 8 {
		t.Fatalf("full row %+v", inns[0])
	}
	shipped := town.OfferRow{Kind: 3, Topic: 31, NPC: 2108, Stages: []int{30}, Zero: []int{927}}
	edited := shipped
	inns[1].Apply(&edited)
	if !reflect.DeepEqual(edited, town.OfferRow{Kind: 3, Topic: 31, NPC: 2108, Stages: []int{30}, Zero: []int{927}, Nonzero: []int{533}}) {
		t.Fatalf("one-field edit %+v", edited)
	}
	if edited.Nonzero[0] = 1; inns[1].Row.Nonzero[0] != 533 {
		t.Fatal("the applied list shares its record's array")
	}
	if !inns[2].Off || len(inns[2].Fields) != 1 || inns[2].Fields[0].Name != OffField {
		t.Fatalf("switch-off %+v", inns[2])
	}
	if d := (TownData{Offers: inns}); d.Empty() || d.First() != inns[0].Source {
		t.Fatal("a file of inn rows alone reads as empty")
	}
	for _, bad := range []struct{ src, want string }{
		{"[[inn]]\ntopic = 1\n", "no npc"},
		{"[[inn]]\ntopic = 1\nnpc = 0\nkind = 3\n", "npc"},
		{"[[inn]]\ntopic = 1\nnpc = 2\n", "changes nothing"},
		{"[[inn]]\ntopic = 1\nnpc = 2\noff = true\nkind = 3\n", "stands alone"},
		{"[[inn]]\ntopic = 1\nnpc = 2\nstages = [0]\n", "stages"},
		{"[[inn]]\ntopic = 1\nnpc = 2\nbank = 3\n", "unknown key"},
		{"[[inn]]\ntopic = 1\nnpc = 2\nkind = 3\n[[inn]]\ntopic = 1\nnpc = 2\nkind = 0\n", "topic 1 npc 2"},
	} {
		if _, _, err := ParseTowns("m", TownsFile, []byte(bad.src)); err == nil || !strings.Contains(err.Error(), bad.want) {
			t.Errorf("%q: %v", bad.src, err)
		}
	}
}

// One parser reads both tables into offer records, each in its table's key.
func TestTownsReadOfferRowsOfBothTables(t *testing.T) {
	src := "[[inn]]\ntopic = 9\nnpc = 207\nkind = 3\n\n[[tavern]]\ntown = 1\nchapter = 30\nnpc = 207\nmission = 31\n"
	_, rows, err := ParseTowns("m", TownsFile, []byte(src))
	if err != nil || len(rows) != 2 {
		t.Fatalf("%d rows: %v", len(rows), err)
	}
	if rows[0].Table != town.InnTable || rows[0].Key() != (town.OfferKey{Table: town.InnTable, Topic: 9, NPC: 207}) ||
		rows[0].Key().String() != "topic 9 npc 207" {
		t.Fatalf("inn record %+v", rows[0])
	}
	var tavern town.OfferRow
	rows[1].Apply(&tavern)
	if rows[1].Table != town.TavernTable || !tavern.Equal(town.ChapterOffer(1, 30, 207, 31)) || tavern.Kind != town.OfferMission ||
		rows[1].Key().String() != "town 1 chapter 30 npc 207" {
		t.Fatalf("tavern record %+v as %+v", rows[1], tavern)
	}
	if d := (TownData{Offers: rows}); d.OfferRows(town.InnTable) != 1 || d.OfferRows(town.TavernTable) != 1 {
		t.Fatal("the rows of each table are not counted apart")
	}
}

// A refusal names the file and the line.
func TestTownsRefuseBrokenTables(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"id = 1\n", "data/towns.toml:1: id is outside a [[town]] table"},
		{"[[city]]\nid = 1\n", `data/towns.toml:1: unknown table "city"`},
		{"[[town]]\nname = \"A\"\n", "data/towns.toml:1: [[town]] has no id"},
		{"[[town]]\nid = 1\n", "data/towns.toml:1: [[town]] 1 changes nothing"},
		{"[[town]]\nid = 0\nname = \"A\"\n", "data/towns.toml:2: id is 0, outside 1.."},
		{"[[town]]\nid = 1\ncolour = 3\n", `data/towns.toml:3: unknown key "colour" in [[town]]`},
		{"[[town]]\nid = 1\nname = 3\n", "data/towns.toml:3: name must be a string"},
		{"[[town]]\nid = 1\noff = true\nname = \"A\"\n", "data/towns.toml:1: [[town]] 1 switches the town off and sets other keys"},
		{"[[town]]\nid = 1\noff = 1\n", "data/towns.toml:3: off must be true or false"},
		{"[[town]]\nid = 1\nmissions = [0]\n", "data/towns.toml:3: missions holds 0"},
		{"[[town]]\nid = 1\nmap-point = 3\n", "data/towns.toml:3: map-point must be an inline table"},
		{"[[town]]\nid = 1\nmap-point = { at = 3 }\n", `data/towns.toml:3: unknown key "at" in map-point`},
		{"[[town]]\nid = 1\nmap-point = { record = false }\n", "data/towns.toml:3: map-point record must be true"},
		{"[[town]]\nid = 1\nname = \"A\"\n[[town]]\nid = 1\nhooks = \"x\"\n", "data/towns.toml:4: town 1 is named again; its first table is at line 1"},
		{"[[tavern]]\nchapter = 50\nnpc = 41\nmission = 0\n", "data/towns.toml:1: [[tavern]] has no town"},
		{"[[tavern]]\ntown = 1\nchapter = 50\nnpc = 41\n", "data/towns.toml:1: [[tavern]] town 1 chapter 50 npc 41 changes nothing"},
		{"[[tavern]]\ntown = 1\nchapter = 50\nnpc = 41\ntext = \"x.txt\"\n", `data/towns.toml:5: unknown key "text" in [[tavern]]`},
		{"[[tavern]]\ntown = 1\nchapter = 50\nnpc = 41\noff = true\nmission = 3\n", "switches the row off and sets other keys"},
		{"[[tavern]]\ntown = 1\nchapter = 50\nnpc = 41\nmission = -1\n", "data/towns.toml:5: mission is -1, outside 0.."},
		{"[[tavern]]\ntown = 1\nchapter = 50\nnpc = 41\nmission = 0\n[[tavern]]\ntown = 1\nchapter = 50\nnpc = 41\noff = true\n",
			"data/towns.toml:6: tavern row town 1 chapter 50 npc 41 is named again; its first table is at line 1"},
	} {
		if _, _, err := ParseTowns("m", TownsFile, []byte(c.src)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %v, want %q", c.src, err, c.want)
		}
	}
}
