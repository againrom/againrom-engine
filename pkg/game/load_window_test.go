package game

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type failTimedDelete struct{ osNamedSaveFiles }

func (f failTimedDelete) Remove(path string) error {
	if filepath.Ext(path) == timedOwnerExtension {
		return errors.New("injected companion deletion failure")
	}
	return f.osNamedSaveFiles.Remove(path)
}

func TestLoadDeleteTimedPairAuthorityAndRollback(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("delete ownership")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err := f.playerMissionSave(snapshot, "timed autosave 1")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"changed SAV", "changed owner", "foreign owner", "ordinary name", "second remove failure"} {
		t.Run(kind, func(t *testing.T) {
			store := SaveStore{Dir: t.TempDir()}
			if err := writeTimedSave(store, nil, 0, 1, raw, nil); err != nil {
				t.Fatal(err)
			}
			base := filepath.Join(store.Dir, string(timedSaveBase(0)))
			owner, err := os.ReadFile(base + timedOwnerExtension)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "ordinary name" {
				base = filepath.Join(store.Dir, "manual")
				if err := os.WriteFile(base+".sav", raw, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(base+timedOwnerExtension, owner, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "foreign owner" {
				owner = []byte("foreign marker")
				if err := os.WriteFile(base+timedOwnerExtension, owner, 0600); err != nil {
					t.Fatal(err)
				}
			}
			var files namedSaveFileOps = osNamedSaveFiles{}
			if kind == "second remove failure" {
				files = failTimedDelete{}
			}
			remove, err := prepareLoadDeleteFiles(store.Dir, saveDialogDeleteToken(filepath.Base(base)+".sav"), nil, files)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "changed owner" {
				owner = append(owner, ' ')
				if err := os.WriteFile(base+timedOwnerExtension, owner, 0600); err != nil {
					t.Fatal(err)
				}
			}
			want := raw
			if kind == "changed SAV" {
				want = append(bytes.Clone(raw), 0)
				if err := os.WriteFile(base+".sav", want, 0600); err != nil {
					t.Fatal(err)
				}
			}
			err = remove()
			if kind == "foreign owner" || kind == "ordinary name" {
				if err != nil {
					t.Fatal("ordinary file delete", err)
				}
				if _, err := os.Stat(base + ".sav"); !os.IsNotExist(err) {
					t.Fatal("SAV retained", err)
				}
			} else {
				if err == nil {
					t.Fatal("changed or failed pair deletion succeeded")
				}
				if got, err := os.ReadFile(base + ".sav"); err != nil || !bytes.Equal(got, want) {
					t.Fatal("SAV lost during refused deletion", err)
				}
			}
			if got, err := os.ReadFile(base + timedOwnerExtension); err != nil || !bytes.Equal(got, owner) {
				t.Fatal("companion lost or altered", err)
			}
			entries, err := os.ReadDir(store.Dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if len(entry.Name()) > 0 && entry.Name()[0] == '.' {
					t.Fatal("delete leaked backup", entry.Name())
				}
			}
		})
	}
}

func TestLoadDeletePreservesOriginalAndChangedSave(t *testing.T) {
	dir := t.TempDir()
	fence := t.TempDir()
	for _, root := range []string{dir, fence} {
		if err := os.WriteFile(filepath.Join(root, "game0001.sav"), []byte("snapshot"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := prepareLoadDelete(dir, "game0001.sav", []string{fence}); err == nil {
		t.Fatal("an install token was allowed to delete a local namesake")
	}
	if _, err := prepareLoadDelete(fence, localOriginalSaveToken("game0001.sav"), []string{fence}); err == nil {
		t.Fatal("the install directory was writable")
	}
	remove, err := prepareLoadDelete(dir, localOriginalSaveToken("game0001.sav"), []string{fence})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "game0001.sav")
	if err := os.WriteFile(path, []byte("changed snapshot"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := remove(); err == nil {
		t.Fatal("changed save was deleted after confirmation")
	}
	remove, err = prepareLoadDelete(dir, localOriginalSaveToken("game0001.sav"), []string{fence})
	if err != nil {
		t.Fatal(err)
	}
	if err := remove(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("confirmed save still exists", err)
	}
	if got, err := os.ReadFile(filepath.Join(fence, "game0001.sav")); err != nil || string(got) != "snapshot" {
		t.Fatal("preserved original changed")
	}
}
