package pathdialog

import "testing"

// TestQuestionRequestCarriesEveryField pins the one route from a question to
// a Request: the kind, title, filters and default name pass through, and the
// start is the one the host gives.
func TestQuestionRequestCarriesEveryField(t *testing.T) {
	filters := []Filter{{Name: "Maps", Extensions: []string{"alm"}}}
	q := Question{Kind: SaveFile, Title: "Save map as", Filters: filters, DefaultName: "new-map.alm"}
	got := q.Request("/maps")
	want := NewRequest(SaveFile, "Save map as", "/maps", filters...)
	want.DefaultName = "new-map.alm"
	if got.Kind != want.Kind || got.Title != want.Title || got.StartDir != want.StartDir || got.DefaultName != want.DefaultName {
		t.Fatalf("Request = %+v, want %+v", got, want)
	}
	if len(got.Filters) != 1 || got.Filters[0].Name != "Maps" {
		t.Fatalf("Request filters = %+v, want the question's filters", got.Filters)
	}
	if empty := (Question{Kind: ChooseFolder}).Request(""); empty.StartDir != "" || empty.Filters != nil {
		t.Fatalf("empty question Request = %+v, want no start and no filters", empty)
	}
}
