package game

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A folder the test play guard admitted once does not admit a fenced target:
// the install, the other preserved install, an archive-identified install,
// and a junction below an admitted folder that leads into the install are each
// refused after the guard admitted the profile, and nothing is written there.
func TestTestPlayGuardAdmittedOnceStillRefusesEveryFencedTarget(t *testing.T) {
	base := t.TempDir()
	en, ru := filepath.Join(base, "gameversions", "en"), filepath.Join(base, "gameversions", "ru")
	other := filepath.Join(base, "another-install")
	profile := filepath.Join(base, "profile")
	for _, dir := range []string{en, ru, other, profile} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range RequiredArchives() {
		if err := os.WriteFile(filepath.Join(other, strings.ToUpper(name)), []byte("synthetic archive marker"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	checks := 0
	guard, forget := admittedOnce(func(dir string) error {
		checks++
		return fenceEditorWrite(en, dir, "Test play")
	})
	folder := filepath.Join(profile, TestPlayModID)
	for _, rel := range []string{"a.txt", "b.txt", "texts/c.txt", "texts/d.txt"} {
		if err := writeTestModFile(guard, profile, folder, rel, []byte("x"), false); err != nil {
			t.Fatalf("admitted profile write %s: %v", rel, err)
		}
	}
	// profile, profile/testplay and profile/testplay/texts: one check each.
	if checks != 3 {
		t.Fatalf("the guard ran its fence %d times for 3 folders", checks)
	}
	for _, top := range []string{en, ru, other} {
		for range 2 {
			if err := writeTestModFile(guard, top, filepath.Join(top, TestPlayModID), "a.txt", []byte("x"), true); err == nil {
				t.Fatalf("a write into fenced %s was admitted", top)
			}
		}
		if _, err := os.Stat(filepath.Join(top, TestPlayModID)); !os.IsNotExist(err) {
			t.Fatalf("a refused write created a folder in %s: %v", top, err)
		}
	}
	if runtime.GOOS != "windows" {
		return
	}
	link := filepath.Join(folder, "link")
	// Only creates a junction between two owned temporary fixture paths.
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, ru).CombinedOutput(); err != nil {
		t.Fatalf("create temporary junction: %v %s", err, out)
	}
	defer os.Remove(link)
	forget()
	if err := guard(folder); err != nil {
		t.Fatalf("admit the test mod folder: %v", err)
	}
	if err := writeTestModFile(guard, profile, folder, "link/e.txt", []byte("x"), false); err == nil {
		t.Fatal("a junction below an admitted folder led a write into the install")
	}
	if _, err := os.Stat(filepath.Join(ru, "e.txt")); !os.IsNotExist(err) {
		t.Fatalf("the junction refusal wrote into the install: %v", err)
	}
}
