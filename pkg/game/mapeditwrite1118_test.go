package game

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMapEditor1118SaveAsGuardsBothInstallsExistingAndSource(t *testing.T) {
	base := t.TempDir()
	en, ru := filepath.Join(base, "gameversions", "en"), filepath.Join(base, "gameversions", "ru")
	for _, dir := range []string{en, ru, filepath.Join(ru, "maps")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	source := filepath.Join(base, "source.alm")
	input := []byte("source sentinel")
	if err := os.WriteFile(source, input, 0600); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(base, "existing.alm")
	if err := os.WriteFile(existing, []byte("existing sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(base, "source-alias.alm")
	if err := os.Link(source, alias); err != nil {
		t.Fatal(err)
	}
	paths := []string{
		"", filepath.Join(base, "wrong.sav"), source, alias, existing,
		filepath.Join(en, "new.alm"), filepath.Join(ru, "new.alm"), filepath.Join(ru, "maps", "new.alm"),
		ru + string(os.PathSeparator) + "maps" + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "dots.alm",
		filepath.Join(base, "missing-dir", "new.alm"),
	}
	if runtime.GOOS == "windows" {
		paths = append(paths, strings.ToUpper(source), strings.ToUpper(filepath.Join(ru, "case.alm")), source+":stream.alm")
		for _, device := range []string{"NUL", "con", "AUX", "COM1", "LPT9", "COM¹"} {
			paths = append(paths, filepath.Join(base, device+".alm"))
		}
	}
	for _, path := range paths {
		if _, err := writeEditedMap(en, source, path, []byte("changed")); err == nil {
			t.Fatalf("unsafe output accepted: %q", path)
		}
	}
	for _, name := range []string{source, alias} {
		b, err := os.ReadFile(name)
		if err != nil || !bytes.Equal(b, input) {
			t.Fatalf("source alias changed: %s %v", name, err)
		}
	}
	if b, _ := os.ReadFile(existing); string(b) != "existing sentinel" {
		t.Fatal("existing destination changed")
	}
	for _, path := range paths[5:10] {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("refusal created file %s: %v", path, err)
		}
	}
	// Archive detection protects other roots, regardless of explicit context.
	other := filepath.Join(base, "another-install")
	if err := os.Mkdir(other, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range RequiredArchives() {
		if err := os.WriteFile(filepath.Join(other, strings.ToUpper(name)), []byte("synthetic archive marker"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := writeEditedMap(en, "scenario/test.alm", filepath.Join(other, "new.alm"), input); err == nil {
		t.Fatal("archive-identified install accepted")
	}
	for i, src := range []string{source, "scenario/test.alm", "loose/test.alm"} {
		target := filepath.Join(base, string(rune('a'+i))+".alm")
		if _, err := writeEditedMap(en, src, target, input); err != nil {
			t.Fatalf("explicit external target refused: %v", err)
		}
		b, _ := os.ReadFile(target)
		if !bytes.Equal(b, input) {
			t.Fatal("writer changed map bytes")
		}
	}
}

func TestMapEditor1118SaveAsSymlinkFence(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "install")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(root, link); err != nil {
		t.Skipf("symlink privilege unavailable: %v; Windows junction tested separately", err)
	}
	defer os.Remove(link)
	if _, err := writeEditedMap(root, "scenario/test.alm", filepath.Join(link, "new.alm"), []byte("not written")); err == nil {
		t.Fatal("symlink hid install")
	}
	if _, err := os.Stat(filepath.Join(root, "new.alm")); !os.IsNotExist(err) {
		t.Fatal("symlink refusal wrote a file")
	}
}
