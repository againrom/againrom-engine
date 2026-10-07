package game

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/fame"
)

func TestFameStoreKeepsInstallAndColdLocalOrder(t *testing.T) {
	install := t.TempDir()
	seed, _ := fame.Marshal([]fame.Record{{Name: "old", Score: 50, Tail: [2]uint32{8, 9}}})
	if err := os.WriteFile(filepath.Join(install, fameFileName), seed, 0600); err != nil {
		t.Fatal(err)
	}
	store := fameStore{Dir: filepath.Join(t.TempDir(), "profile"), OriginalDir: install}
	for _, score := range []int32{40, 50} {
		if _, err := store.add(fame.Record{Name: "hero", Score: score}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := (fameStore{Dir: store.Dir, OriginalDir: install}).read()
	if err != nil || len(got) != 3 || got[0].Name != "hero" || got[0].Score != 50 || got[1].Name != "old" || got[1].Tail != [2]uint32{8, 9} || got[2].Score != 40 {
		t.Fatal(got, err)
	}
	after, err := os.ReadFile(filepath.Join(install, fameFileName))
	if err != nil || !bytes.Equal(after, seed) {
		t.Fatal("changed installed hall", err)
	}
}

type failFameReplace struct{ osNamedSaveFiles }

func (failFameReplace) Replace(string, string) error { return errors.New("injected replace failure") }

func TestFameStoreFailedWriteAndMalformedDataPreservePreviousBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fameFileName)
	valid, _ := fame.Marshal([]fame.Record{{Name: "old", Score: 50}})
	for _, old := range [][]byte{valid, []byte("broken")} {
		if err := os.WriteFile(path, old, 0600); err != nil {
			t.Fatal(err)
		}
		store := fameStore{Dir: dir, files: failFameReplace{}}
		if _, err := store.add(fame.Record{Name: "new", Score: 60}); err == nil {
			t.Fatal("expected write refusal")
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(old, after) {
			t.Fatal("lost previous bytes", err)
		}
		files, err := filepath.Glob(filepath.Join(dir, ".againrom-fame-*"))
		if err != nil || len(files) != 0 {
			t.Fatal("left temporary files", files, err)
		}
	}
}

func TestFameStoreRefusesInstalledOutput(t *testing.T) {
	install := t.TempDir()
	for _, dir := range []string{install, filepath.Join(install, "child")} {
		if _, err := (fameStore{Dir: dir, OriginalDir: install}).add(fame.Record{Name: "hero"}); err == nil {
			t.Fatal("accepted installed output", dir)
		}
	}
}

func TestFameStoreStartsWithoutAnInstalledSeed(t *testing.T) {
	store := fameStore{Dir: filepath.Join(t.TempDir(), "profile")}
	rows, err := store.add(fame.Record{Name: "first", Score: 7})
	if err != nil || len(rows) != 1 || rows[0].Score != 7 {
		t.Fatal(rows, err)
	}
}
