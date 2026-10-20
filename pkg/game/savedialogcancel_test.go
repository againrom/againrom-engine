package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/ui"
)

func loadedGroupFrontForDialog(t *testing.T, words ...uint32) *FrontEnd {
	t.Helper()
	f := newGroupFront(t, -1)
	raw := newGroupLiteral(t, f, false, words...)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("LOAD", town, err)
	}
	if err := f.App("dialog").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f
}

func groupListsForDialog(f *FrontEnd) string {
	groups, _, _ := f.live.world.SavedGroups()
	s := ""
	for _, g := range groups {
		s += fmt.Sprintf("[id%d sel%d n%d]", g.ID, g.Selector, len(g.Members))
	}
	return s
}

// A SAVE prunes where it writes (SAV-1251): a dialog SAVE whose overwrite is
// cancelled writes nothing and leaves the live World and its hash alone; a
// confirmed one prunes and writes the file a LOAD reads back as that World.
func TestSaveDialogPrunesOnlyWhenTheFileIsWritten(t *testing.T) {
	for _, words := range [][]uint32{{0, 1}, {1, 0}} {
		t.Run(fmt.Sprint(words), func(t *testing.T) {
			f := loadedGroupFrontForDialog(t, words...)
			dir := t.TempDir()
			slot := filepath.Join(dir, "slot.sav")
			if err := os.WriteFile(slot, []byte("old"), 0o600); err != nil {
				t.Fatal(err)
			}
			request := ui.SaveRequest{OnMap: true, Directory: dir, Name: "slot", Label: "slot", Format: ui.SaveSAV}
			beforeLists, beforeHash := groupListsForDialog(f), f.live.world.Hash()
			seams := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{})

			cancelled, err := seams.Prepare(request)
			if err != nil {
				t.Fatal(err)
			}
			if len(cancelled.Existing) != 1 {
				t.Fatalf("existing %v", cancelled.Existing)
			}
			if _, err := cancelled.Commit(false); err == nil {
				t.Fatal("an unconfirmed overwrite committed")
			}
			if groupListsForDialog(f) != beforeLists || f.live.world.Hash() != beforeHash {
				t.Fatalf("cancelled overwrite changed the World: %s -> %s", beforeLists, groupListsForDialog(f))
			}
			if raw, _ := os.ReadFile(slot); string(raw) != "old" {
				t.Fatalf("cancelled overwrite wrote %q", raw)
			}

			confirmed, err := seams.Prepare(request)
			if err != nil {
				t.Fatal(err)
			}
			if groupListsForDialog(f) != beforeLists || f.live.world.Hash() != beforeHash {
				t.Fatalf("Prepare changed the World: %s -> %s", beforeLists, groupListsForDialog(f))
			}
			if _, err := confirmed.Commit(true); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(slot)
			if err != nil || string(raw) == "old" {
				t.Fatalf("confirmed save wrote %q %v", raw, err)
			}
			afterLists, afterHash := groupListsForDialog(f), f.live.world.Hash()
			g := loadedGroupFrontForDialog(t, words...)
			g.live.world.PruneGroupsForSave()
			if afterLists != groupListsForDialog(g) || afterHash != g.live.world.Hash() {
				t.Fatalf("written SAVE left %s, the pruning entry point gives %s", afterLists, groupListsForDialog(g))
			}
			if words[0] == 0 && afterLists == beforeLists {
				t.Fatalf("a confirmed SAVE pruned nothing: %s", afterLists)
			}
			open, town, err := f.RestoreOriginal(raw)
			if err != nil || town {
				t.Fatal("LOAD of the written file", town, err)
			}
			if err := f.App("dialog").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			if groupListsForDialog(f) != afterLists || f.live.world.Hash() != afterHash {
				t.Fatalf("LOAD of the file %s, World after SAVE %s", groupListsForDialog(f), afterLists)
			}
		})
	}
}
