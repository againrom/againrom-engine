package mod

import (
	"reflect"
	"testing"

	"againrom/pkg/town"
)

// WriteTowns writes its offer rows through the one row writer, so the rows it
// writes are the rows WriteOfferRows appends to an empty towns file.
func TestWriteTownsOfferRowsAreTheRowWriterRows(t *testing.T) {
	shop := town.ChapterOffer(1, 30, 0, 31)
	shop.Place = town.PlaceShop
	rows := []town.OfferRow{town.ChapterOffer(1, 30, 22, 30), shop}
	var records []OfferRecord
	for _, r := range rows {
		records = append(records, OfferRecordOf(town.TavernTable.Of(r), r, false))
	}
	want, err := WriteOfferRows(nil, nil, records)
	if err != nil {
		t.Fatal(err)
	}
	got, err := WriteTowns(nil, nil, rows...)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("WriteTowns writes\n%s\nthe row writer writes\n%s", got, want)
	}
}

// A towns file written with the older layout (eleven-wide keys, blank lines,
// off = false) reads back as the same offer rows as the layout written now.
func TestTownsFileOldOfferRowLayoutReadsTheSame(t *testing.T) {
	old := "\n[[tavern]]\ntown        = 1\nchapter     = 30\nnpc         = 22\nmission     = 30\noff         = false\n" +
		"\n[[shop]]\ntown        = 1\nchapter     = 30\nmission     = 31\noff         = false\n"
	now := "[[tavern]]\ntown    = 1\nchapter = 30\nnpc     = 22\nmission = 30\n" +
		"[[shop]]\ntown    = 1\nchapter = 30\nmission = 31\n"
	_, a, err := ParseTowns("", TownsFile, []byte(old))
	if err != nil {
		t.Fatal(err)
	}
	_, b, err := ParseTowns("", TownsFile, []byte(now))
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 2 || len(b) != 2 {
		t.Fatalf("old layout %d rows, new layout %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Table != b[i].Table || a[i].Off != b[i].Off || !reflect.DeepEqual(a[i].Row, b[i].Row) {
			t.Fatalf("row %d: old layout %v, new layout %v", i, a[i], b[i])
		}
	}
}
