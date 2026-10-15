package game

import (
	"os"
	"path/filepath"
	"testing"
)

// These tests are SYNTHETIC and read no install (golden rule 2). What they can
// check is the store's own behaviour: which reader a name belongs to, what a
// missing directory answers, what an unreadable file does to a list, and which
// names are refused outright. That a real `game####.sav` resumes a real mission
// is checked by cmd/savecheck against a lawful install and recorded as evidence,
// which is the same division every other format in this tree uses.

func TestIsOriginalReadsTheExtensionAndOpensNothing(t *testing.T) {
	for _, c := range []struct {
		name string
		want bool
	}{
		{"game0000.sav", true},
		{"GAME0000.SAV", true},
		{"save-20260812-120000.ags", false},
		{"game0000.sav.ags", false},
		{"sav", false},
		{"", false},
	} {
		if got := IsOriginal(c.name); got != c.want {
			t.Errorf("IsOriginal(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestOriginalStoreAnswersNothingRatherThanFailing(t *testing.T) {
	// An unset directory is a build pointed at no install; a missing one is an
	// install with no saves beside it. Both are "no rows" to a player, and
	// neither is an error the load window could act on.
	if got := (OriginalStore{}).List(); got != nil {
		t.Errorf("an unset directory listed %d row(s), want none", len(got))
	}
	if got := (OriginalStore{Dir: filepath.Join(t.TempDir(), "absent")}).List(); got != nil {
		t.Errorf("a missing directory listed %d row(s), want none", len(got))
	}
	if _, err := (OriginalStore{}).Read("game0000.sav"); err == nil {
		t.Error("reading from an unset directory succeeded")
	}
}

func TestOriginalStoreSkipsWhatItCannotRead(t *testing.T) {
	dir := t.TempDir()
	// Right extension, wrong bytes: this is the file the original game never
	// wrote, and one of them must not hide the others.
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), []byte("not a save at all"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Wrong extension entirely, so not even a candidate.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := (OriginalStore{Dir: dir}).List(); len(got) != 0 {
		t.Errorf("listed %d row(s) from a directory holding nothing readable: %v", len(got), got)
	}
}

func TestOriginalStoreRefusesANameThatIsNotBare(t *testing.T) {
	// This directory is the player's own game install, so a name carrying a
	// separator is the one way a chosen row could reach a file that is not a
	// save. The names come from List, which produces bare ones.
	s := OriginalStore{Dir: t.TempDir()}
	for _, name := range []string{
		"../game0000.sav",
		"sub/game0000.sav",
		`sub\game0000.sav`,
		"..",
		".",
	} {
		if _, err := s.Read(name); err == nil {
			t.Errorf("Read(%q) was allowed", name)
		}
	}
}

func TestTheLoadListPutsOursFirstAndTheOriginalsAfter(t *testing.T) {
	ours, originals := t.TempDir(), t.TempDir()
	for _, row := range []struct{ dir, name, label string }{
		{ours, "a.sav", "ours one"}, {ours, "b.sav", "ours two"}, {originals, "original.sav", "original"},
	} {
		if err := os.WriteFile(filepath.Join(row.dir, row.name), encodeForTest(t, row.label), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, list, _ := (&FrontEnd{}).SaveSeams(SaveStore{Dir: ours}, OriginalStore{Dir: originals}, nil)
	rows := list()
	if len(rows) != 3 {
		t.Fatalf("listed %d row(s), want two local then one original: %v", len(rows), rows)
	}
	local := map[string]bool{localOriginalSaveToken("a.sav"): true, localOriginalSaveToken("b.sav"): true}
	for _, row := range rows[:2] {
		if !local[row.Name] {
			t.Fatal("local SAV row lost its namespace", row)
		}
		delete(local, row.Name)
	}
	if len(local) != 0 || rows[2].Name != "original.sav" {
		t.Fatal("original SAV interleaved with local rows", rows)
	}
}

func encodeForTest(t *testing.T, label string) []byte {
	t.Helper()
	f, _ := city1095Front(t)
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestALabelIsMadeDrawableThroughTheInstallsOwnPage(t *testing.T) {
	for _, c := range []struct {
		name     string
		raw      []byte
		selector int
		want     string
	}{
		{"plain ascii is untouched", []byte("finished"), 0, "finished"},
		{"a high byte decodes at selector 0", []byte{0xe0, 'f', 'i', 'n'}, 0, "аfin"},
		{"the same byte decodes differently at selector 1", []byte{0xe0, 'f', 'i', 'n'}, 1, "рfin"},
		{"a NUL ends the label", []byte{'1', '2', 0, 'j', 'u', 'n', 'k'}, 0, "12"},
		{"control bytes still cannot decode", []byte{'a', 0x07, 'b'}, 0, "a?b"},
		{"an all-Cyrillic name keeps its length and its letters", []byte{0xe0, 0xe1, 0xe2}, 0, "абв"},
		{"the one Windows-1251 gap still marks", []byte{'a', 0x98, 'b'}, 0, "a?b"},
		{"empty is empty", nil, 0, ""},
	} {
		if got := asciiLabel(c.raw, c.selector); got != c.want {
			t.Errorf("%s: asciiLabel(%v, %d) = %q, want %q", c.name, c.raw, c.selector, got, c.want)
		}
	}
}
