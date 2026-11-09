package installinfo

import (
	"errors"
	"testing"

	"againrom/pkg/locale"
)

type fakeSource struct {
	want    string
	payload []byte
	err     error
}

func (f fakeSource) ReadFile(name string) ([]byte, error) {
	if name != f.want {
		return nil, errors.New("wrong address " + name)
	}
	return f.payload, f.err
}

func TestLanguageEntryReadsTheLocaleRecord(t *testing.T) {
	for _, l := range locale.All() {
		if got := languageEntry(l.Selector); got != l.Entry {
			t.Errorf("languageEntry(%d) = %q, want %q", l.Selector, got, l.Entry)
		}
	}
	if got := languageEntry(7); got != "language selector 7" {
		t.Errorf("unknown selector entry = %q", got)
	}
}

func TestLanguageSelectorReadsTheLastDigit(t *testing.T) {
	const addr = "main/id"
	if LanguagePath != addr {
		t.Fatalf("LanguagePath = %q, want %q", LanguagePath, addr)
	}
	for _, tc := range []struct {
		name string
		src  EntrySource
		want int
	}{
		{"no source", nil, 0},
		{"english", fakeSource{want: addr, payload: []byte("english 0")}, 0},
		{"russian", fakeSource{want: addr, payload: []byte("russian 1")}, 1},
		{"no digit", fakeSource{want: addr, payload: []byte("russian")}, 0},
		{"empty", fakeSource{want: addr}, 0},
		{"unreadable", fakeSource{want: addr, err: errors.New("no entry")}, 0},
	} {
		if got := LanguageSelector(tc.src); got != tc.want {
			t.Errorf("%s: selector = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestInspectOfAnEmptyDirectoryListsTheArchives(t *testing.T) {
	info := Inspect(t.TempDir())
	if info.Valid() || len(info.Missing) == 0 || info.Language != "" {
		t.Fatalf("Inspect of an empty directory = %+v", info)
	}
}
