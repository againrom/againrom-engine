package mod

import (
	"testing"

	"againrom/pkg/town"
)

const offerWriteSource = `# The towns of the mod.
[[town]]
id = 2
name = "Ford"   # kept

# The talk of 22.
[[tavern]]
town = 1
chapter = 30
npc = 22
mission = 31   # the first talk

[[shop]]
town = 1
from = 30
mission = 33
won = [31,32]
`

// The one row writer keeps the comments, the order and the untouched
// tables of a towns file: an edit rewrites the changed values only, an off
// switch keeps the key, a removed record leaves with its lines and a new
// record is appended as a hand-made one would read.
func TestOfferRowsKeepTheSourceAndChangeOnlyTheEditedValues(t *testing.T) {
	_, before, err := ParseTowns("m", TownsFile, []byte(offerWriteSource))
	if err != nil {
		t.Fatal(err)
	}
	edit := func(f func([]OfferRecord) []OfferRecord) string {
		t.Helper()
		after := f(append([]OfferRecord(nil), before...))
		b, err := WriteOfferRows([]byte(offerWriteSource), before, after)
		if err != nil {
			t.Fatal(err)
		}
		if _, back, err := ParseTowns("m", TownsFile, b); err != nil || len(back) != len(after) {
			t.Fatalf("written rows do not read back: %v\n%s", err, b)
		}
		return string(b)
	}
	if got := edit(func(r []OfferRecord) []OfferRecord { return r }); got != offerWriteSource {
		t.Fatalf("an unchanged draft rewrote the file:\n%s", got)
	}
	for _, c := range []struct {
		name string
		f    func([]OfferRecord) []OfferRecord
		want string
	}{
		{"mission", func(r []OfferRecord) []OfferRecord {
			row := r[0].Row
			row.Topic = 35
			r[0] = OfferRecordOf(town.TavernTable, row, false)
			r[0].Source = before[0].Source
			return r
		}, "mission = 35   # the first talk\n"},
		{"won", func(r []OfferRecord) []OfferRecord {
			row := r[1].Row
			row.Won = []int{31}
			r[1] = OfferRecordOf(town.ShopTable, row, false)
			r[1].Source = before[1].Source
			return r
		}, "won = [31]\n"},
		{"off", func(r []OfferRecord) []OfferRecord {
			r[0] = OfferRecordOf(town.TavernTable, r[0].Row, true)
			r[0].Source = before[0].Source
			return r
		}, "npc = 22\n\noff = true\n"},
		{"remove", func(r []OfferRecord) []OfferRecord { return r[:1] }, "# the first talk\n\n"},
		{"add", func(r []OfferRecord) []OfferRecord {
			row := town.ChapterOffer(1, 40, 0, 41)
			row.Won = []int{31}
			return append(r, OfferRecordOf(town.SchoolTable, row, false))
		}, "won = [31,32]\n[[school]]\ntown    = 1\nchapter = 40\nmission = 41\nwon     = [31]\n"},
	} {
		got := edit(c.f)
		if !hasLines(got, "# The towns of the mod.\n[[town]]\nid = 2\nname = \"Ford\"   # kept\n\n# The talk of 22.\n") || !hasLines(got, c.want) {
			t.Fatalf("%s:\n%s", c.name, got)
		}
	}
}

func hasLines(s, part string) bool {
	for i := 0; i+len(part) <= len(s); i++ {
		if s[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
