package launch

import "testing"

// The selected base is the one named last, else the first, and last stays as
// it was when there is no base.
func TestSelectedKeyAndBase(t *testing.T) {
	bases := []NamedBase{{Key: "en", Path: "/g/en"}, {Key: "ru", Path: "/g/ru"}}
	for _, c := range []struct {
		name      string
		bases     []NamedBase
		last      string
		wantKey   string
		wantPath  string
		wantFound bool
	}{
		{"named base", bases, "ru", "ru", "/g/ru", true},
		{"named base in another case", bases, "RU", "RU", "/g/ru", true},
		{"unknown name falls to the first", bases, "de", "en", "/g/en", true},
		{"empty name falls to the first", bases, "", "en", "/g/en", true},
		{"no base keeps last", nil, "ru", "ru", "", false},
		{"no base and no name", nil, "", "", "", false},
	} {
		if got := SelectedKey(c.bases, c.last); got != c.wantKey {
			t.Errorf("%s: SelectedKey = %q, want %q", c.name, got, c.wantKey)
		}
		path, ok := SelectedBase(c.bases, c.last)
		if path != c.wantPath || ok != c.wantFound {
			t.Errorf("%s: SelectedBase = %q, %v, want %q, %v", c.name, path, ok, c.wantPath, c.wantFound)
		}
	}
}
