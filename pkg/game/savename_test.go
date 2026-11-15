package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/savename"
)

func TestSaveFolderListsAndLoadsBothExtensions(t *testing.T) {
	dir := t.TempDir()
	raw := labelledSave(3, "both")
	when := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, name := range []string{"a.sav", "b.savx"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, when, when); err != nil {
			t.Fatal(err)
		}
	}
	store := SaveStore{Dir: dir}
	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range list {
		names = append(names, e.Name)
	}
	if want := []string{"b.savx", "a.sav"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("list %v, want %v", names, want)
	}
	a, err := store.Read("a.sav")
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Read("b.savx")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("b.savx does not read equal to a.sav")
	}
	if _, err := loadDeletePath(dir, saveDialogDeleteToken("b.savx"), nil); err != nil {
		t.Fatalf("a listed .savx file is not deletable: %v", err)
	}
}

func TestSaveExtensionRule(t *testing.T) {
	if got := saveExtOf(Snapshot{}); got != ".sav" {
		t.Fatalf("state without a mod: %q", got)
	}
	if got := saveExtOf(Snapshot{guest: &guestMission{}}); got != ".sav" {
		t.Fatalf("state under a mod: %q", got)
	}
	if got := savename.For(true); got != ".savx" {
		t.Fatalf("extended state: %q", got)
	}
}

func TestWrittenSaveNameKeepsItsExtension(t *testing.T) {
	name, err := (SaveStore{Dir: t.TempDir()}).WriteOriginal("", labelledSave(3, "x"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "game0000.sav" {
		t.Fatalf("written name %q", name)
	}
}
