package mod

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/town"
)

// decoderTownKeys are the keys the towns decoder reads, as its own refusal
// of an unknown key lists them.
func decoderTownKeys(t *testing.T) []string {
	t.Helper()
	_, _, err := ParseTowns("probe", TownsFile, []byte("[[town]]\nid = 1\nno-such-key = 1\n"))
	m := regexp.MustCompile(`\(keys: ([^)]*)\)`).FindStringSubmatch(errString(err))
	if m == nil {
		t.Fatalf("the decoder's unknown-key refusal lists no keys: %v", err)
	}
	return strings.Split(m[1], ", ")
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// missingTownKeys are the decoder's keys a written towns file leaves out of
// some table, each named once.
func missingTownKeys(t *testing.T, data []byte) []string {
	t.Helper()
	records, _, err := ParseTowns("dump", TownsFile, data)
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, r := range records {
		have := []string{"id"}
		for _, f := range r.Fields {
			have = append(have, f.Name)
		}
		for _, k := range decoderTownKeys(t) {
			if !slices.Contains(have, k) && !slices.Contains(missing, k) {
				missing = append(missing, k)
			}
		}
	}
	return missing
}

// eastEntry sets every field a [[town]] table can set.
var eastEntry = town.Entry{ID: 2, Description: "towns/east.json", Hooks: "chapters", Offers: "chapter-keys",
	MapPoint: town.MapPoint{Point: []int{430, 230}, Rect: []int{410, 205, 40, 30}}, Name: `Восток "East" \ 1`,
	OpensAfter: 20, Missions: []int{31, 32}, Cite: []string{"TOWN-489", "owner"}}

// A written towns file holds every key the decoder reads, in each table, and
// reads back to the entries it was written from. Its loss control: the same
// file without one key's line is reported as missing that key.
func TestWriteTownsHoldsEveryKeyTheDecoderReads(t *testing.T) {
	base := town.Entry{ID: 1, Description: "rom1", Hooks: "chapters", Offers: "chapter-keys",
		MapPoint: town.MapPoint{Object: 1}, Name: "TOWN", Cite: []string{"TOWN-489"}}
	data, err := WriteTowns([]string{"head line", ""}, []town.Entry{base, eastEntry})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "# head line\n#\n\n[[town]]\nid          = 1\n") {
		t.Fatalf("file head:\n%s", data)
	}
	if missing := missingTownKeys(t, data); len(missing) != 0 {
		t.Fatalf("the file leaves out %v:\n%s", missing, data)
	}
	records, _, err := ParseTowns("dump", TownsFile, data)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []town.Entry{base, eastEntry} {
		var got town.Entry
		records[i].Apply(&got)
		if !sameTownEntry(got, want) || records[i].Off {
			t.Errorf("town %d reads back as %+v", want.ID, got)
		}
	}
	for _, key := range decoderTownKeys(t) {
		if key == "id" {
			continue
		}
		cut := regexp.MustCompile(`(?m)^`+regexp.QuoteMeta(key)+` +=.*\n`).ReplaceAllString(string(data), "")
		if missing := missingTownKeys(t, []byte(cut)); !slices.Equal(missing, []string{key}) {
			t.Errorf("without its %s lines the file is missing %v", key, missing)
		}
	}
}

// An entry with no name writes no name key; every other key stays, and the
// file still reads back to the entry. The map point's other forms and empty
// lists are written in the decoder's format.
func TestWriteTownsForms(t *testing.T) {
	entries := []town.Entry{
		{ID: 1, Description: "rom2-town1", Hooks: "destinations", Offers: "scenario-bank", MapPoint: town.MapPoint{Record: true}},
	}
	data, err := WriteTowns(nil, entries)
	if err != nil {
		t.Fatal(err)
	}
	want := "[[town]]\nid          = 1\ndescription = \"rom2-town1\"\nhooks       = \"destinations\"\noffers      = \"scenario-bank\"\n" +
		"map-point   = { record = true }\nopens-after = 0\nmissions    = []\ncite        = []\noff         = false\n"
	if string(data) != want {
		t.Fatalf("got\n%s\nwant\n%s", data, want)
	}
	if missing := missingTownKeys(t, data); !slices.Equal(missing, []string{"name"}) {
		t.Fatalf("missing %v, want only name", missing)
	}
}

// An entry the file cannot give back is refused: a name with a line break,
// a name that is not UTF-8, and a comment line with a line break.
func TestWriteTownsRefusesWhatDoesNotReadBack(t *testing.T) {
	for _, name := range []string{"a\nb", "\x87\xa0"} {
		e := eastEntry
		e.Name = name
		if _, err := WriteTowns(nil, []town.Entry{e}); err == nil || !strings.Contains(err.Error(), "does not read back") {
			t.Errorf("name %q: %v", name, err)
		}
	}
	if _, err := WriteTowns([]string{"a\nb"}, []town.Entry{eastEntry}); err == nil {
		t.Error("a comment line with a line break was written")
	}
}
