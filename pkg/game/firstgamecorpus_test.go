package game

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
)

// walkFirstGameSaves is the one walk of a save corpus for the first game's
// instruments: every .sav file under corpus in walk order, past corpusDirSkip,
// except a file whose envelope is another game's. A file opening with the
// second game's magic (Bsg&) is left out; a file no envelope opens stays in,
// for its reader to refuse. fn gets the path and the slash path under corpus.
func walkFirstGameSaves(corpus string, fn func(path, rel string) error) error {
	return filepath.WalkDir(corpus, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip := corpusDirSkip(d); skip != nil {
			return skip
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".sav" {
			return nil
		}
		other, err := otherGameSave(path)
		if err != nil {
			return err
		}
		if other {
			return nil
		}
		rel, relErr := filepath.Rel(corpus, path)
		if relErr != nil {
			rel = path
		}
		return fn(path, filepath.ToSlash(rel))
	})
}

// otherGameSave reports whether the file's envelope magic is a game's other
// than the first.
func otherGameSave(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	head := make([]byte, 4)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false, err
	}
	p, ok := sav.ProfileOf(head[:n])
	return ok && p != sav.FirstGame, nil
}

// The corpus walk gives a first-game file and a file no envelope opens, and
// leaves out a second-game file, another extension and an exp-* directory.
func TestFirstGameSaveWalkLeavesOutTheSecondGame(t *testing.T) {
	corpus := t.TempDir()
	for name, body := range map[string]string{
		"a.sav":         "Asg&first",
		"b.sav":         "Bsg&second",
		"c.sav":         "no",
		"d.ags":         "Asg&first",
		"exp-run/e.sav": "Asg&first",
		"sub/f.sav":     "Asg&first",
		"sub/g.sav":     "Bsg&second",
	} {
		path := filepath.Join(corpus, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var got []string
	if err := walkFirstGameSaves(corpus, func(path, rel string) error {
		got = append(got, rel)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"a.sav", "c.sav", "sub/f.sav"}; !slices.Equal(got, want) {
		t.Fatalf("the walk gave %q, want %q", got, want)
	}
}
