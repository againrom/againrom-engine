package game_test

// Tests for the install's language selector.
//
// The selector decides whether every drawn byte is converted, so the thing worth
// pinning is not that it parses a digit — it is that EVERY way of failing to
// find one answers 0, the identity rule. A selector guessed wrong would silently
// redraw the whole game; 0 can only ever leave drawing as it was.

import (
	"errors"
	"io/fs"
	"testing"

	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

// selSource is the one thing LanguageSelector needs of a filesystem, with a
// single entry in it. A name it does not hold reports the same error a real
// container reports for a missing entry.
type selSource struct {
	name string
	data []byte
}

func (s selSource) ReadFile(name string) ([]byte, error) {
	if s.name != "" && name == s.name {
		return s.data, nil
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

func TestLanguageSelectorReadsTheTrailingDigit(t *testing.T) {
	// The two shipped roots' own payloads, written as bytes so no non-ASCII
	// literal enters a fixture and so the trailing byte is unmistakable.
	for _, tc := range []struct {
		name string
		data string
		want int
	}{
		{"english", "english 0", 0},
		{"russian", "russian 1", 1},
		{"bare digit", "7", 7},
		{"trailing newline is not a digit", "russian 1\n", 0},
		{"digit run takes the last", "russian 21", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := game.LanguageSelector(selSource{name: game.LanguagePath, data: []byte(tc.data)})
			if got != tc.want {
				t.Fatalf("LanguageSelector(%q) = %d, want %d", tc.data, got, tc.want)
			}
		})
	}
}

func TestLanguageSelectorIsZeroOnEveryFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  terrain.EntrySource
	}{
		{"no source", nil},
		{"no entry", selSource{}},
		{"empty payload", selSource{name: game.LanguagePath}},
		{"ends in a letter", selSource{name: game.LanguagePath, data: []byte("english")}},
		{"ends in a space", selSource{name: game.LanguagePath, data: []byte("russian ")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := game.LanguageSelector(tc.src); got != 0 {
				t.Fatalf("LanguageSelector = %d, want 0 — a selector that cannot be read must leave drawing alone", got)
			}
		})
	}
}

// The address is composed from the container's own identity segment, so a reader
// can tell which archive it comes out of.
func TestLanguagePathNamesTheMainContainer(t *testing.T) {
	if got, want := game.LanguagePath, "main/id"; got != want {
		t.Fatalf("LanguagePath = %q, want %q", got, want)
	}
	// And it is genuinely the entry the selector asks for.
	src := selSource{name: game.LanguagePath, data: []byte("russian 1")}
	if game.LanguageSelector(src) != 1 {
		t.Fatal("the selector did not read LanguagePath")
	}
	if _, err := src.ReadFile("main/other"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("the stub should refuse every other name")
	}
}
