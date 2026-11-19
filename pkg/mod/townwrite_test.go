package mod

import (
	"strings"
	"testing"

	"againrom/pkg/town"
)

const townWriteSource = `# The towns of the mod.
[[town]]
id = 2
name = "Ford"   # kept
opens-after = 20

[[tavern]]
town = 1
chapter = 30
npc = 22
mission = 31

[[inn]]
topic = 7
npc = 520
town = 2
stages = [30]
`

// The towns kind's one writer changes a town, a tavern row and an inn row of
// a towns file and a record it appends reads back as the record, with the
// comments and untouched tables of the source kept.
func TestTownsWriterRoundTripsATownATavernRowAndAnInnRow(t *testing.T) {
	records, offers, err := ParseTowns("m", TownsFile, []byte(townWriteSource))
	if err != nil {
		t.Fatal(err)
	}
	before := TownData{Records: records, Offers: offers}
	write := func(after TownData) string {
		t.Helper()
		out, err := WriteTownData([]byte(townWriteSource), before, after)
		if err != nil {
			t.Fatal(err)
		}
		return string(out)
	}
	if got := write(TownData{Records: append([]TownRecord(nil), records...), Offers: append([]OfferRecord(nil), offers...)}); got != townWriteSource {
		t.Fatalf("an unchanged draft rewrote the file:\n%s", got)
	}
	changed := append([]TownRecord(nil), records...)
	changed[0].Entry.Name = "Harbor"
	changed[0].Entry.OpensAfter = 31
	row := offers[0].Row
	row.Topic, row.Kind = 35, town.OfferMission
	inn := offers[1].Row
	inn.NPC = 521
	added := TownRecord{Entry: town.Entry{ID: 1, Name: "Home", MapPoint: town.MapPoint{Point: []int{10, 20}, Rect: []int{0, 5, 40, 30}}},
		Fields: []PatchField{{Name: "name"}, {Name: "map-point"}}}
	after := TownData{Records: append(changed, added), Offers: []OfferRecord{OfferRecordOf(town.TavernTable, row, false), OfferRecordOf(town.InnTable, inn, false)}}
	got := write(after)
	for _, want := range []string{"name = \"Harbor\"   # kept", "opens-after = 31", "mission = 35", "npc     = 521", "# The towns of the mod.",
		"id      = 1", "name    = \"Home\"", "map-point = { point = [10, 20], rect = [0, 5, 40, 30] }"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the written file lacks %q:\n%s", want, got)
		}
	}
	back, rows, err := ParseTowns("m", TownsFile, []byte(got))
	if err != nil || len(back) != 2 || len(rows) != 2 {
		t.Fatalf("the written file reads back %d towns, %d rows: %v", len(back), len(rows), err)
	}
	var e town.Entry
	back[1].Apply(&e)
	if e.ID != 1 || e.Name != "Home" || len(e.MapPoint.Point) != 2 {
		t.Fatalf("the appended town reads back as %+v", e)
	}
	if rows[0].Row.Topic != 35 || rows[1].Table != town.InnTable || rows[1].Row.NPC != 521 {
		t.Fatalf("the rows read back as %+v", rows)
	}
	gone := write(TownData{Records: records[:0], Offers: offers[:1]})
	if strings.Contains(gone, "[[town]]") || strings.Contains(gone, "[[inn]]") || !strings.Contains(gone, "[[tavern]]") {
		t.Fatalf("a record after lacks stayed:\n%s", gone)
	}
	off := append([]TownRecord(nil), records...)
	off[0] = TownRecord{Source: records[0].Source, Entry: town.Entry{ID: 2}, Off: true, Fields: []PatchField{{Name: OffField}}}
	if got := write(TownData{Records: off, Offers: offers}); !strings.Contains(got, "off = true") || strings.Contains(got, "Ford") {
		t.Fatalf("a switched-off town keeps its keys:\n%s", got)
	}
}

// A talk text reads back as the words it was written from.
func TestTalkTextReadsBackAsItsWords(t *testing.T) {
	b := TalkText(41, "One line.\nTwo lines.")
	if got := TalkBody(b); got != "One line.\nTwo lines." || !strings.HasPrefix(string(b), "<part=1 npc=41>\r\n") {
		t.Fatalf("talk text %q reads back %q", b, got)
	}
}
