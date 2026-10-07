package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/fame"
	"againrom/pkg/ui"
)

func profileInstall(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{MainArchive, GraphicsArchive, ScenarioArchive, WorldArchive, MoviesArchive} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRuntimeProfileInstallLaunchWritesOnlyPrivateFiles(t *testing.T) {
	for _, subdir := range []string{"", "bin"} {
		t.Run("executable_"+subdir, func(t *testing.T) {
			install := profileInstall(t)
			exeDir := filepath.Join(install, subdir)
			if err := os.MkdirAll(exeDir, 0755); err != nil {
				t.Fatal(err)
			}
			seed, err := fame.Marshal([]fame.Record{{Name: "original", Score: 5}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(install, fameFileName), seed, 0600); err != nil {
				t.Fatal(err)
			}
			profile, err := ResolveRuntimeProfile("", filepath.Join(exeDir, "againrom.exe"), t.TempDir(), install)
			if err != nil {
				t.Fatal(err)
			}
			want := filepath.Join(exeDir, "Againrom")
			if profile.Directory != want || profile.Saves.Dir != filepath.Join(want, "saves") || profile.Options.Path != filepath.Join(want, optionsFileName) {
				t.Fatalf("launch profile = %+v, want private profile %s", profile, want)
			}
			raw := originalWriterLabeled(t, "profile checkpoint")
			name, err := profile.Saves.WriteOriginal(install, raw)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := profile.Saves.Read(name); err != nil || !bytes.Equal(got, raw) {
				t.Fatal("cold save read", err)
			}
			f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Root: install}}}
			f.ConfigureSaveSeams(ui.NewApp("profile", nil, nil, nil), profile.Saves, OriginalStore{Dir: install}, nil)
			if _, err := f.hallStore.add(fame.Record{Name: "player", Score: 10}); err != nil {
				t.Fatal(err)
			}
			if rows, err := f.hallStore.read(); err != nil || len(rows) != 2 || rows[1].Name != "original" {
				t.Fatal("profile hall seed", rows, err)
			}
			if err := profile.Options.SetTipsMode(false); err != nil {
				t.Fatal(err)
			}
			if on, err := (OptionsStore{Path: profile.Options.Path}).TipsMode(); err != nil || on {
				t.Fatal("cold profile options", on, err)
			}
			seams := f.SaveDialogSeams(profile.Saves, OriginalStore{Dir: install})
			if listed, err := seams.List(profile.Saves.Dir); err != nil || len(listed.Entries) != 1 {
				t.Fatal("profile dialog list", listed, err)
			}
			remove, err := seams.PrepareDelete(profile.Saves.Dir, name)
			if err != nil {
				t.Fatal(err)
			}
			if err := remove(); err != nil {
				t.Fatal(err)
			}
			if got, err := os.ReadFile(filepath.Join(install, fameFileName)); err != nil || !bytes.Equal(got, seed) {
				t.Fatal("installed hall changed", err)
			}
			if _, err := os.Stat(filepath.Join(install, optionsFileName)); !os.IsNotExist(err) {
				t.Fatal("installed options were created", err)
			}
		})
	}
}

func TestRuntimeProfileFallsBackWhenPrivateDirectoryCannotBeCreated(t *testing.T) {
	install, config := profileInstall(t), t.TempDir()
	if err := os.WriteFile(filepath.Join(install, "Againrom"), []byte("occupied"), 0600); err != nil {
		t.Fatal(err)
	}
	profile, err := ResolveRuntimeProfile("", filepath.Join(install, "againrom.exe"), config, install)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Directory != filepath.Join(config, "Againrom") {
		t.Fatalf("fallback profile = %s", profile.Directory)
	}
	if _, err := profile.Saves.WriteOriginal(install, []byte("fallback")); err != nil {
		t.Fatal(err)
	}
	if err := profile.Options.SetTipsMode(false); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(install, "Againrom")); string(got) != "occupied" {
		t.Fatal("replaced blocker")
	}
}

func TestRuntimeProfileExplicitInstallTargetsKeepAllWriteFences(t *testing.T) {
	install := profileInstall(t)
	for _, dir := range []string{install, filepath.Join(install, "new-saves"), filepath.Join(install, "Againrom", "saves")} {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			profile, err := ResolveRuntimeProfile(dir, filepath.Join(install, "againrom.exe"), t.TempDir(), install)
			if err != nil {
				t.Fatal(err)
			}
			if profile.Saves.Dir != dir {
				t.Fatal("explicit override moved", profile.Saves.Dir)
			}
			if _, err := profile.Saves.WriteOriginal(install, []byte("no")); err == nil {
				t.Fatal("explicit SAV escaped install fence")
			}
			if dir != install {
				if err := profile.Options.SetTipsMode(false); err == nil {
					t.Fatal("explicit options escaped install fence")
				}
			}
		})
	}
}

func TestRuntimeProfileFallbackCannotAuthorizeAnotherInstall(t *testing.T) {
	install, other := profileInstall(t), profileInstall(t)
	if err := os.WriteFile(filepath.Join(install, "Againrom"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveRuntimeProfile("", filepath.Join(install, "againrom.exe"), other, install); err == nil {
		t.Fatal("configuration path authorized another install")
	}
	if _, err := os.Stat(filepath.Join(other, "Againrom")); !os.IsNotExist(err) {
		t.Fatal("created fallback inside another install", err)
	}
}

func TestRuntimeProfileNamedSaveAndLoadKeepTheLaunchGrant(t *testing.T) {
	install := profileInstall(t)
	profile, err := ResolveRuntimeProfile("", filepath.Join(install, "againrom.exe"), t.TempDir(), install)
	if err != nil {
		t.Fatal(err)
	}
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("profile save")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	f.ConfigureSaveSeams(app, profile.Saves, OriginalStore{Dir: install}, nil)
	seams := f.SaveDialogSeams(profile.Saves, OriginalStore{Dir: install})
	request := ui.SaveRequest{Directory: profile.Saves.Dir, Name: "checkpoint", Format: ui.SaveSAV, OnMap: true}
	for cycle := 0; cycle < 2; cycle++ {
		prepared, err := seams.Prepare(request)
		if err != nil {
			t.Fatal(err)
		}
		if len(prepared.Existing) != cycle {
			t.Fatal("replacement confirmation", prepared.Existing)
		}
		if _, err := prepared.Commit(cycle != 0); err != nil {
			t.Fatal(err)
		}
	}
	_, list, load := f.SaveSeams(profile.Saves, OriginalStore{Dir: install}, nil)
	if rows := list(); len(rows) != 1 || rows[0].Name != localOriginalSaveToken("checkpoint.sav") {
		t.Fatal("LOAD profile rows", rows)
	}
	if open, town, err := load(localOriginalSaveToken("checkpoint.sav")); err != nil || town || open == nil {
		t.Fatal("LOAD profile mission", town, err)
	}
	remove, err := prepareLoadDelete(profile.Saves.Dir, localOriginalSaveToken("checkpoint.sav"), []string{install}, profile.Saves.profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := remove(); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{install, filepath.Join(install, "new-saves"), filepath.Join(profile.Directory, "elsewhere"), profileInstall(t)} {
		if _, err := seams.List(dir); err == nil {
			t.Fatalf("browser granted another install directory: %s", dir)
		}
		moved := profile.Saves
		moved.Dir = dir
		if _, err := moved.WriteOriginal(install, []byte("no")); err == nil {
			t.Fatalf("copied store widened grant: %s", dir)
		}
	}
	if _, err := (SaveStore{Dir: profile.Saves.Dir}).WriteOriginal(install, []byte("no")); err == nil {
		t.Fatal("directory name granted private access")
	}
}

func TestRuntimeProfileGrantRejectsReplacementAndNestedInstall(t *testing.T) {
	for _, change := range []string{"replace saves", "replace root", "nested install", "profile becomes install"} {
		t.Run(change, func(t *testing.T) {
			install := profileInstall(t)
			profile, err := ResolveRuntimeProfile("", filepath.Join(install, "againrom.exe"), t.TempDir(), install)
			if err != nil {
				t.Fatal(err)
			}
			name, err := profile.Saves.WriteOriginal(install, []byte("keep"))
			if err != nil {
				t.Fatal(err)
			}
			seams := (&FrontEnd{}).SaveDialogSeams(profile.Saves, OriginalStore{Dir: install})
			remove, err := seams.PrepareDelete(profile.Saves.Dir, name)
			if err != nil {
				t.Fatal(err)
			}
			prepared, err := prepareNamedSave(profile.Saves.Dir, "new", []namedSavePayload{{".sav", []byte("new")}}, []string{install}, nil, profile.Saves.profile)
			if err != nil {
				t.Fatal(err)
			}
			target := profile.Saves.Dir
			switch change {
			case "replace saves", "replace root":
				changed := profile.Saves.Dir
				if change == "replace root" {
					changed = profile.Directory
				}
				if err := os.Rename(changed, changed+"-held"); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(profile.Saves.Dir, 0700); err != nil {
					t.Fatal(err)
				}
			case "nested install", "profile becomes install":
				if change == "profile becomes install" {
					target = profile.Directory
				}
				for _, archive := range []string{MainArchive, GraphicsArchive, ScenarioArchive, WorldArchive, MoviesArchive} {
					if err := os.WriteFile(filepath.Join(target, archive), nil, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			if _, err := profile.Saves.WriteOriginal(install, []byte("no")); err == nil {
				t.Fatal("changed profile accepted quick SAVE")
			}
			if _, err := prepared.Commit(false); err == nil {
				t.Fatal("changed profile accepted confirmed SAVE")
			}
			if err := remove(); err == nil {
				t.Fatal("changed profile accepted confirmed delete")
			}
		})
	}
}

func TestRuntimeProfileOptionsAtomicReplacementPreservesLinkedOriginal(t *testing.T) {
	for _, link := range []string{"hardlink", "symlink"} {
		t.Run(link, func(t *testing.T) {
			install := profileInstall(t)
			profile, err := ResolveRuntimeProfile("", filepath.Join(install, "againrom.exe"), t.TempDir(), install)
			if err != nil {
				t.Fatal(err)
			}
			original := filepath.Join(install, optionsFileName)
			seed := []byte("TipsMode=1\nUnrelated=keep\n")
			if err := os.WriteFile(original, seed, 0600); err != nil {
				t.Fatal(err)
			}
			if link == "hardlink" {
				err = os.Link(original, profile.Options.Path)
			} else {
				err = os.Symlink(original, profile.Options.Path)
			}
			if err != nil {
				t.Skipf("fixture %s unavailable: %v", link, err)
			}
			err = profile.Options.SetTipsMode(false)
			if link == "symlink" && err == nil {
				t.Fatal("options followed symbolic link")
			}
			if link == "hardlink" && err != nil {
				t.Fatal("atomic options replacement", err)
			}
			if got, err := os.ReadFile(original); err != nil || !bytes.Equal(got, seed) {
				t.Fatal("options changed original through link", err)
			}
		})
	}
}
