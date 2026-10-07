package game

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func profileJunction(t *testing.T, link, target string) {
	t.Helper()
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Fatalf("create owned temporary junction: %v %s", err, out)
	}
	t.Cleanup(func() { _ = os.Remove(link) })
}

func TestEffectRimOutputJunctionCannotHideAnInstall(t *testing.T) {
	install := profileInstall(t)
	link := filepath.Join(t.TempDir(), "redirect")
	profileJunction(t, link, install)
	if path, err := effectRimOutputPath(install, link); err == nil {
		t.Fatalf("install junction admitted as witness output: %s", path)
	}
}

func TestRuntimeProfileJunctionCannotGrantOrRetargetInstallWrites(t *testing.T) {
	t.Run("initial profile junction falls back", func(t *testing.T) {
		install, other, config := profileInstall(t), profileInstall(t), t.TempDir()
		profileJunction(t, filepath.Join(install, "Againrom"), other)
		profile, err := ResolveRuntimeProfile("", filepath.Join(install, "againrom.exe"), config, install)
		if err != nil {
			t.Fatal(err)
		}
		if profile.Directory != filepath.Join(config, "Againrom") {
			t.Fatal("profile junction was granted", profile.Directory)
		}
		if _, err := os.Stat(filepath.Join(other, "saves")); !os.IsNotExist(err) {
			t.Fatal("created saves through profile junction", err)
		}
	})
	t.Run("saves junction after confirmation", func(t *testing.T) {
		install, other := profileInstall(t), profileInstall(t)
		profile, err := ResolveRuntimeProfile("", filepath.Join(install, "againrom.exe"), t.TempDir(), install)
		if err != nil {
			t.Fatal(err)
		}
		name, err := profile.Saves.WriteOriginal(install, []byte("keep"))
		if err != nil {
			t.Fatal(err)
		}
		remove, err := prepareLoadDelete(profile.Saves.Dir, localOriginalSaveToken(name), []string{install}, profile.Saves.profile)
		if err != nil {
			t.Fatal(err)
		}
		prepared, err := prepareNamedSave(profile.Saves.Dir, "new", []namedSavePayload{{".sav", []byte("new")}}, []string{install}, nil, profile.Saves.profile)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(other, name), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(profile.Saves.Dir, profile.Saves.Dir+"-held"); err != nil {
			t.Fatal(err)
		}
		profileJunction(t, profile.Saves.Dir, other)
		if _, err := profile.Saves.WriteOriginal(install, []byte("no")); err == nil {
			t.Fatal("quick SAVE followed junction")
		}
		if _, err := prepared.Commit(false); err == nil {
			t.Fatal("confirmed SAVE followed junction")
		}
		if err := remove(); err == nil {
			t.Fatal("confirmed delete followed junction")
		}
		if got, err := os.ReadFile(filepath.Join(other, name)); err != nil || string(got) != "keep" {
			t.Fatal("original namesake changed", err)
		}
		if _, err := os.Stat(filepath.Join(other, "new.sav")); !os.IsNotExist(err) {
			t.Fatal("published through junction", err)
		}
	})
}
