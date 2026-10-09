package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/ui"
)

func originalWriterLabeled(t *testing.T, label string) []byte {
	t.Helper()
	b := savedFile(0, tenActors())
	bodyEnd := int(binary.LittleEndian.Uint32(b[4:8]))
	if bodyEnd+0x100 > len(b) || len(label) >= 0x100 {
		t.Fatalf("synthetic original label does not fit: body end %d, file %d, label %d", bodyEnd, len(b), len(label))
	}
	copy(b[bodyEnd:bodyEnd+0x100], make([]byte, 0x100))
	copy(b[bodyEnd:], label)
	return b
}

func TestWriteOriginalClaimsTheFirstFreeROMNameWithoutOverwrite(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "game0000.sav")
	if err := os.WriteFile(firstPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: dir}
	payload := []byte("deterministic authored bytes")

	name, err := store.WriteOriginal("", payload)
	if err != nil {
		t.Fatalf("WriteOriginal: %v", err)
	}
	if name != "game0001.sav" {
		t.Fatalf("WriteOriginal name = %q, want game0001.sav", name)
	}
	if got, err := os.ReadFile(firstPath); err != nil || string(got) != "existing" {
		t.Fatalf("pre-existing slot = (%q, %v), want unchanged", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, name)); err != nil || string(got) != string(payload) {
		t.Fatalf("authored slot = (%q, %v), want %q", got, err, payload)
	}

	second, err := store.WriteOriginal("", payload)
	if err != nil {
		t.Fatalf("second WriteOriginal: %v", err)
	}
	if second != "game0002.sav" {
		t.Fatalf("second WriteOriginal name = %q, want game0002.sav", second)
	}
	if got, err := os.ReadFile(filepath.Join(dir, second)); err != nil || string(got) != string(payload) {
		t.Fatalf("second authored slot = (%q, %v), want the same deterministic bytes", got, err)
	}
}

func TestWriteOriginalRefusesPreservedInstallTargetsAtRuntime(t *testing.T) {
	t.Run("archive census without OriginalStore context", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"MAIN.RES", "graphics.res", "Scenario.Res", "WORLD.RES", "movies.res"} {
			if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		name, err := (SaveStore{Dir: dir}).WriteOriginal("", []byte("must not land"))
		if err == nil || name != "" || !strings.Contains(err.Error(), "game install") {
			t.Fatalf("WriteOriginal into synthetic install = (%q, %v), want install refusal", name, err)
		}
		if matches, globErr := filepath.Glob(filepath.Join(dir, "game????.sav")); globErr != nil || len(matches) != 0 {
			t.Fatalf("published saves after refusal = (%v, %v), want none", matches, globErr)
		}
	})

	t.Run("archive census in target ancestor without OriginalStore context", func(t *testing.T) {
		install := t.TempDir()
		for _, name := range []string{"main.res", "GRAPHICS.RES", "scenario.res", "World.Res", "movies.res"} {
			if err := os.WriteFile(filepath.Join(install, name), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		target := filepath.Join(install, "player-saves", "nested")
		name, err := (SaveStore{Dir: target}).WriteOriginal("", []byte("must not land"))
		if err == nil || name != "" || !strings.Contains(err.Error(), "game install") {
			t.Fatalf("WriteOriginal under synthetic install = (%q, %v), want ancestor install refusal", name, err)
		}
		if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
			t.Fatalf("refused descendant stat = %v, want no directory created", statErr)
		}
	})

	t.Run("configured read-only install descendant", func(t *testing.T) {
		install := t.TempDir()
		target := filepath.Join(install, "new-saves")
		name, err := (SaveStore{Dir: target}).WriteOriginal(install, []byte("must not land"))
		if err == nil || name != "" || !strings.Contains(err.Error(), "read-only game install") {
			t.Fatalf("WriteOriginal under configured install = (%q, %v), want read-only refusal", name, err)
		}
		if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
			t.Fatalf("refused target stat = %v, want no directory created", statErr)
		}
	})
}

func TestSaveStoreListsLocalSAVAndIgnoresAGS(t *testing.T) {
	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	if err := os.WriteFile(filepath.Join(dir, "game0000.sav"), savedFile(0, tenActors()), 0o644); err != nil {
		t.Fatal(err)
	}
	native, err := EncodeSave(Snapshot{}, "native")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Write(time.Date(2026, 8, 27, 1, 2, 3, 0, time.UTC), native); err != nil {
		t.Fatal(err)
	}

	entries, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Name] = true
	}
	if !seen["game0000.sav"] || len(seen) != 1 {
		t.Fatalf("listed names = %v, want the one local .sav and no .ags", seen)
	}
}

func TestSaveSeamsKeepsCollidingLocalAndInstallOriginalRowsDistinct(t *testing.T) {
	localDir, installDir := t.TempDir(), t.TempDir()
	localPath := filepath.Join(localDir, "game0000.sav")
	installPath := filepath.Join(installDir, "game0000.sav")
	localBytes := originalWriterLabeled(t, "LOCAL")
	installBytes := originalWriterLabeled(t, "INSTALL")
	if err := os.WriteFile(localPath, localBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installPath, installBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(Campaign{}, nil)}, CampaignSession: CampaignSession{Town: NewTown(Campaign{})}}
	_, list, load := agsSaveSeams(f, SaveStore{Dir: localDir}, OriginalStore{Dir: installDir}, nil)
	rows := list()
	byName := make(map[string]ui.SaveEntry, len(rows))
	for _, row := range rows {
		byName[row.Name] = row
	}
	localToken := localOriginalSaveToken("game0000.sav")
	if got := byName[localToken]; !strings.HasPrefix(got.Label, "LOCAL ") {
		t.Fatalf("local collision row = %+v, want LOCAL label", got)
	}
	if got := byName["game0000.sav"]; !strings.HasPrefix(got.Label, "INSTALL ") {
		t.Fatalf("install collision row = %+v, want INSTALL label", got)
	}

	if err := os.WriteFile(installPath, []byte("broken install row"), 0o644); err != nil {
		t.Fatal(err)
	}
	if open, town, err := load(localToken); err != nil || open != nil || !town {
		t.Fatalf("load(local token) = (open=%v, town=%v, err=%v), want local bytes loaded into town", open != nil, town, err)
	}
	if err := os.WriteFile(installPath, installBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localPath, []byte("broken local row"), 0o644); err != nil {
		t.Fatal(err)
	}
	if open, town, err := load("game0000.sav"); err != nil || open != nil || !town {
		t.Fatalf("load(install name) = (open=%v, town=%v, err=%v), want install bytes loaded into town", open != nil, town, err)
	}
}

func TestProductionLoadRefusesAnAGSName(t *testing.T) {
	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	native, err := EncodeSave(Snapshot{}, "native")
	if err != nil {
		t.Fatal(err)
	}
	name, err := store.Write(time.Date(2026, 8, 27, 1, 2, 3, 0, time.UTC), native)
	if err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{}
	_, _, load := f.SaveSeams(store, OriginalStore{}, time.Now)
	if _, _, err := load(name); err == nil || !strings.Contains(err.Error(), "not a SAV file") {
		t.Fatalf("production load of %q = %v, want a not-a-SAV refusal", name, err)
	}
}
