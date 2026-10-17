package pathdialog

import (
	"errors"
	"testing"
)

func TestUnavailable(t *testing.T) {
	result, err := unavailable(Request{Kind: ChooseFolder})
	if result != (Result{}) || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unavailable = %+v, %v", result, err)
	}
}

func TestValidate(t *testing.T) {
	valid := Request{Kind: OpenFile, Title: "Открыть карту", Filters: []Filter{{Name: "Maps", Extensions: []string{"alm"}}, {Name: "All files", Extensions: []string{"*"}}}, StartDir: "folder", DefaultName: "map.alm"}
	if err := validate(valid); err != nil {
		t.Fatal(err)
	}
	for _, request := range []Request{
		{Kind: Kind(99)}, {Title: "bad\x00title"}, {StartDir: "bad\x00folder"}, {DefaultName: "bad\x00name"},
		{Filters: []Filter{{Name: "Maps"}}},
		{Filters: []Filter{{Name: "Maps", Extensions: []string{".alm"}}}},
		{Filters: []Filter{{Name: "Maps", Extensions: []string{"alm;exe"}}}},
		{Filters: []Filter{{Name: "Maps", Extensions: []string{"*", "alm"}}}},
	} {
		if err := validate(request); err == nil {
			t.Errorf("accepted invalid request %+v", request)
		}
	}
}

func TestSelection(t *testing.T) {
	for _, test := range []struct {
		path   string
		cancel bool
		want   Result
		fail   bool
	}{
		{"карта.alm", false, Result{Path: "карта.alm"}, false},
		{"ignored.alm", true, Result{Cancelled: true}, false},
		{"", false, Result{}, true},
	} {
		got, err := selected(test.path, test.cancel)
		if got != test.want || (err != nil) != test.fail {
			t.Errorf("selected(%q,%t) = %+v, %v", test.path, test.cancel, got, err)
		}
	}
}
