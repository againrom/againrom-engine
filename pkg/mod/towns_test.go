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
	rs, err := ParseTowns("east", TownsFile, []byte(src))
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
	} {
		if _, err := ParseTowns("m", TownsFile, []byte(c.src)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %v, want %q", c.src, err, c.want)
		}
	}
}
