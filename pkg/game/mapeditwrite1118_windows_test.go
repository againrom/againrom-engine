package game

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMapEditor1118WindowsJunctionFence(t *testing.T) {
	base := t.TempDir()
	en, ru := filepath.Join(base, "gameversions", "en"), filepath.Join(base, "gameversions", "ru")
	for _, path := range []string{en, ru} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(base, "alias")
	// Only creates a junction between two owned temporary fixture paths.
	// No real install, ACL, profile or registry is touched.
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, ru).CombinedOutput(); err != nil {
		t.Fatalf("create temporary junction: %v %s", err, out)
	}
	defer os.Remove(link)
	resolved, err := editorPhysicalDirectory(link)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := os.Stat(resolved)
	b, _ := os.Stat(ru)
	if !os.SameFile(a, b) {
		t.Fatalf("junction resolution %q does not name RU", resolved)
	}
	if _, err := writeEditedMap(en, "scenario/test.alm", filepath.Join(link, "new.alm"), []byte("not written")); err == nil {
		t.Fatal("junction hid the other preserved install")
	}
	if _, err := os.Stat(filepath.Join(ru, "new.alm")); !os.IsNotExist(err) {
		t.Fatal("junction refusal created a file")
	}
}
