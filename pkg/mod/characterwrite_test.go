package mod

import (
	"strings"
	"testing"
)

func TestCharactersWriterRoundTripsAndPreservesSource(t *testing.T) {
	source := []byte("# kept\n[[character]]\ntarget = 'NPC06' # family\nface = 3 # appearance\n\n[[character]]\nkey = 'guest'\nrow = 'NPC06'\nname = 'guest.name'\n")
	look := func(string) (string, bool) { return "Guest", true }
	d, err := ParseCharacters("m", CharactersFile, source, look, "en")
	if err != nil {
		t.Fatal(err)
	}
	k, ok := WrittenKind(CharactersFile)
	if !ok || !HeldDraft(CharactersFile) || !RebuildsContext(CharactersFile, func(string) bool { return true }) {
		t.Fatal("characters have no held writer or draft context")
	}
	same, err := k.Write(CharactersFile, source, Data{Characters: d})
	if err != nil || string(same) != string(source) {
		t.Fatalf("unchanged: %s, %v", same, err)
	}
	d.Rows[0].Face = 9
	d.Rows = append(d.Rows, CharacterRow{Key: "visitor", Row: "NPC07", NameKey: "visitor.name", Face: 4},
		CharacterRow{Target: "NPC06_1", Kind: "A_Guard", Strip: []string{StripArmour}}, CharacterRow{Key: "hidden", Off: true})
	out, err := k.Write(CharactersFile, source, Data{Characters: d})
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseCharacters("m", CharactersFile, out, look, "ru")
	if err != nil || len(back.Rows) != 5 || back.Rows[0].Face != 9 || back.Rows[2].Name != "Guest" || !back.Rows[4].Off {
		t.Fatalf("round trip: %+v, %v", back, err)
	}
	if !strings.Contains(string(out), "target = 'NPC06' # family\nface = 9 # appearance") {
		t.Fatalf("source lost: %s", out)
	}
	d.Rows = d.Rows[1:]
	if _, err := k.Write(CharactersFile, out, Data{Characters: d}); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Write(CharactersFile, nil, Data{Characters: CharacterData{Rows: []CharacterRow{{Target: "NPC06"}}}}); err == nil {
		t.Fatal("empty edit accepted")
	}
}

func TestCharactersWriterDeleteOfTheFirstRepeatKeepsTheOthersLines(t *testing.T) {
	source := "# head\n[[character]]\ntarget = \"Guard_2\" # A first\nface = 2 # A face\n\n[[character]]\ntarget = \"Guard_2\" # B second\nface = 7 # B face\n"
	look := func(string) (string, bool) { return "-", true }
	d, err := ParseCharacters("m", CharactersFile, []byte(source), look, "en")
	if err != nil || len(d.Rows) != 2 {
		t.Fatalf("parse: %+v, %v", d, err)
	}
	d.Rows = d.Rows[1:]
	out, err := writeCharacters("", []byte(source), Data{Characters: d})
	want := "# head\n\n[[character]]\ntarget = \"Guard_2\" # B second\nface = 7 # B face\n"
	if err != nil || string(out) != want {
		t.Fatalf("delete first: %q, %v; want %q", out, err, want)
	}
}

func TestOneRecordKeyRuleMakesTwoDistinctKeysForOneStem(t *testing.T) {
	held := map[string]bool{}
	a := RecordKey("character", func(k string) bool { return held[k] })
	held[a] = true
	if b := RecordKey("character", func(k string) bool { return held[k] }); a == b {
		t.Fatalf("one key %q twice", a)
	}
}
