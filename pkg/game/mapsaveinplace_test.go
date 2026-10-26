package game

import (
	"os"
	"path/filepath"
	"testing"
)

func inPlaceEdits(t *testing.T, source string) *mapInspectionEdits {
	t.Helper()
	m := built(t, cleanOptions(), nil, nil)
	return &mapInspectionEdits{inspector: checkInspector(), model: m, source: source, saved: m.Bytes()}
}

func TestSaveInPlaceWritesTheOwnFileAndLeavesNoTemporary(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "map.alm")
	if err := os.WriteFile(source, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := inPlaceEdits(t, source)
	if p, ok := s.OwnFile(); !ok || p != source {
		t.Fatalf("OwnFile %q %v", p, ok)
	}
	if err := s.model.SetUnitField(0, "owner", 1); err != nil {
		t.Fatal(err)
	}
	got, err := s.SaveInPlace()
	if err != nil || got != source {
		t.Fatalf("SaveInPlace %q %v", got, err)
	}
	if b, err := os.ReadFile(source); err != nil || string(b) != string(mustSaveBytes(t, s)) {
		t.Fatalf("written map: %v", err)
	}
	if s.Dirty() {
		t.Fatal("dirty after Save")
	}
	if names, _ := filepath.Glob(filepath.Join(dir, ".map-*")); len(names) != 0 {
		t.Fatalf("temporary left: %v", names)
	}
}

func TestSaveInPlaceIsRefusedWhereThereIsNoOwnFile(t *testing.T) {
	dir := t.TempDir()
	for _, source := range []string{"new/80x80", "scenario/20.alm", "loose/a.alm", filepath.Join(dir, "missing.alm")} {
		s := inPlaceEdits(t, source)
		if _, ok := s.OwnFile(); ok {
			t.Fatalf("%q has an own file", source)
		}
		if _, err := s.SaveInPlace(); err == nil {
			t.Fatalf("%q saved in place", source)
		}
	}
	// A first Save As makes the chosen file the own file.
	s := inPlaceEdits(t, "scenario/20.alm")
	target := filepath.Join(dir, "out.alm")
	if _, err := s.SaveAs(target); err != nil {
		t.Fatal(err)
	}
	if p, ok := s.OwnFile(); !ok || p != target {
		t.Fatalf("after Save As: %q %v", p, ok)
	}
	// Save As onto the existing file is still refused.
	if _, err := s.SaveAs(target); err == nil {
		t.Fatal("Save As replaced an existing file")
	}
}

func TestSaveInPlaceAsksAboutErrorsFirst(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "map.alm")
	if err := os.WriteFile(source, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := errorEdits(t, source)
	if _, err := s.SaveInPlace(); !asksToSave(err) {
		t.Fatalf("Save: %v", err)
	}
	if b, _ := os.ReadFile(source); string(b) != "old" {
		t.Fatal("the question wrote the file")
	}
	s.AllowErrors()
	if _, err := s.SaveInPlace(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveInPlaceNeverWritesIntoAnInstall(t *testing.T) {
	base := t.TempDir()
	en, ru := filepath.Join(base, "gameversions", "en"), filepath.Join(base, "gameversions", "ru")
	for _, dir := range []string{en, ru} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, dir := range []string{en, ru} {
		path := filepath.Join(dir, "own.alm")
		if err := os.WriteFile(path, []byte("sentinel"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := writeEditedMap(en, path, path, []byte("changed"), true); err == nil {
			t.Fatalf("overwrite into %s accepted", dir)
		}
		if b, _ := os.ReadFile(path); string(b) != "sentinel" {
			t.Fatalf("%s changed", path)
		}
	}
	if _, err := writeEditedMap(en, "x", filepath.Join(base, "none.txt"), nil, true); err == nil {
		t.Fatal("non-.alm accepted")
	}
	dirTarget := filepath.Join(base, "d.alm")
	if err := os.Mkdir(dirTarget, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := writeEditedMap(en, dirTarget, dirTarget, []byte("x"), true); err == nil {
		t.Fatal("directory replaced")
	}
}

func TestSaveInPlaceRewritesOwnTextsFolderOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "map.alm")
	folder := filepath.Join(dir, "map.text")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "stale.txt"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	pages := map[string][]byte{"title.txt": []byte("fresh")}
	if _, err := writeEditedMapPages("", path, path, []byte("new"), pages, "en", true); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(folder, "title.txt")); string(b) != "fresh" {
		t.Fatalf("pages %q", b)
	}
	if _, err := os.Stat(filepath.Join(folder, "stale.txt")); err == nil {
		t.Fatal("stale page kept")
	}
	if _, err := os.Stat(folder + ".old"); err == nil {
		t.Fatal("backup folder left")
	}
	if b, _ := os.ReadFile(path); string(b) != "new" {
		t.Fatalf("map %q", b)
	}
	// A texts path that is a file is not the map's own folder: not replaced,
	// the map still saves and the note says what was not written.
	other := filepath.Join(dir, "two.alm")
	if err := os.WriteFile(other, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "two.text"), []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := writeEditedMapPages("", other, other, []byte("new"), pages, "en", true); err == nil {
		t.Fatal("a texts path that is a file was replaced")
	}
	s := inPlaceEdits(t, other)
	if _, err := s.SaveInPlace(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "two.text")); string(b) != "file" {
		t.Fatal("texts path changed")
	}
}

// detachSource removes the host file an editor opened, so its map has no file
// of its own and Save asks for a name.
func detachSource(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}
