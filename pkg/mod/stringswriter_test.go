package mod

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

// The strings kind's writer in the kind table writes pairs that read back
// equal; a changed value is rewritten in place, a renamed key keeps its
// line, a new one is appended, and every byte outside them stays.
func TestStringsWriterKeepsTheSource(t *testing.T) {
	rel, bom := StringsFile("en"), string(utf8BOM)
	k, ok := WrittenKind(rel)
	if !ok || k.Name != "strings" {
		t.Fatalf("the editor writes no strings file: %+v", k)
	}
	src := bom + "# mod texts\n\"m1.title\" = \"First\" # shown in the list\nm1_brief = 'Go north'\n\n# the end\n\"gone\" = \"x\"\r\n\"last\" = \"y\""
	before, err := ParseStrings(rel, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	same, err := k.Write(rel, []byte(src), Data{Strings: map[string]Strings{"en": before}})
	if err != nil || string(same) != src {
		t.Fatalf("an unchanged write: %v\n%q", err, same)
	}
	after := maps.Clone(before)
	after["m1.title"], after["m1_brief"] = "Первый \"q\"", "Go\nsouth"
	delete(after, "gone")
	after["renamed key"], after["z.new"] = "x", "new"
	out, err := k.Write(rel, []byte(src), Data{Strings: map[string]Strings{"en": after}})
	if err != nil {
		t.Fatal(err)
	}
	want := bom + "# mod texts\n\"m1.title\" = \"Первый \\\"q\\\"\" # shown in the list\nm1_brief = \"Go\\nsouth\"\n\n# the end\n\"renamed key\" = \"x\"\r\n\"last\" = \"y\"\n\"z.new\" = \"new\"\n"
	if string(out) != want {
		t.Fatalf("written\n%q\nwant\n%q", out, want)
	}
	back, err := ParseStrings(rel, out)
	if err != nil || !maps.Equal(back, after) {
		t.Fatalf("read back %v: %v", back, err)
	}
	gone, err := k.Write(rel, []byte(src), Data{})
	if err != nil || string(gone) != bom+"# mod texts\n\n# the end\n" {
		t.Fatalf("every pair removed: %v\n%q", err, gone)
	}
	ru := StringsFile("ru")
	fresh, err := k.Write(ru, nil, Data{Strings: map[string]Strings{"ru": {"a": "б"}}})
	if err != nil || string(fresh) != "\"a\" = \"б\"\n" {
		t.Fatalf("a new file: %v\n%q", err, fresh)
	}
	if _, err := k.Write("text/xx/strings.toml", nil, Data{}); err == nil {
		t.Fatal("a file outside the strings layout was written")
	}
}

// A mission row holds the text keys its title and briefing cite, and the row
// writer writes the keys, so a row's key picked anew is the one saved.
func TestMissionRowsCiteTheirTextKeys(t *testing.T) {
	d, err := ParseMissions("m", MissionsFile, []byte(goodMission), missionText, "en")
	if err != nil {
		t.Fatal(err)
	}
	m := d.Missions[0]
	if m.TitleKey != "t" || m.BriefKey != "b" || m.Title != "Title" || m.Brief != "Briefing" {
		t.Fatalf("row %+v", m)
	}
	same, err := WriteMissions([]byte(goodMission), d.Missions, d.Missions)
	if err != nil || string(same) != goodMission {
		t.Fatalf("an unchanged write: %v\n%s", err, same)
	}
	after := CloneMission(m)
	after.TitleKey, after.Title = "b", "Briefing"
	out, err := WriteMissions([]byte(goodMission), d.Missions, []Mission{after})
	if err != nil || string(out) != strings.Replace(goodMission, "title    = \"t\"", "title    = \"b\"", 1) {
		t.Fatalf("a new title key: %v\n%s", err, out)
	}
}

// Cites is the text keys a kind's file cites through its own reader, held
// or not; a kind that cites no text answers none.
func TestKindsCiteTheTextKeysTheirReadersAsk(t *testing.T) {
	r := DataReader{Mod: "m", Lang: "en", Text: textOf(map[string]string{})}
	for _, c := range []struct {
		name, rel, src string
		want           []string
	}{
		{"items", ItemsFile, goodItems, []string{"item.gambeson"}},
		{"terrain", TerrainFile, "[[group]]\ngroup = 13\nkind = \"grass\"\nname = \"Snow\"\n", nil},
	} {
		k, ok := ScriptDataKind(c.rel)
		if !ok || k.Name != c.name {
			t.Fatalf("%s: kind %q", c.rel, k.Name)
		}
		got, err := k.Cites(r, c.rel, []byte(c.src))
		if err != nil || !slices.Equal(got, c.want) {
			t.Fatalf("%s cites %q: %v", c.name, got, err)
		}
	}
}

// A key the strings file cannot hold is refused by the writer in the words
// the parser refuses it with.
func TestStringsWriterRefusesTheKeyTheParserRefuses(t *testing.T) {
	rel := StringsFile("en")
	_, parsed := ParseStrings(rel, []byte("\"\" = \"x\"\n"))
	k, _ := WrittenKind(rel)
	_, written := k.Write(rel, nil, Data{Strings: map[string]Strings{"en": {"": "x"}}})
	if parsed == nil || written == nil || CheckTextKey("") == nil || !strings.Contains(parsed.Error(), CheckTextKey("").Error()) || !strings.Contains(written.Error(), CheckTextKey("").Error()) {
		t.Fatalf("parser %v, writer %v", parsed, written)
	}
}
