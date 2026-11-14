package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/base"
	"againrom/pkg/ui"
)

func quickSlotsPerGameFiles(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = raw
	}
	return out
}

func quickSlotsPerGameNames(rows []SaveEntry) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Name)
	}
	slices.Sort(out)
	return out
}

func TestQuickAndTimedSlotsAreEachGamesOwn(t *testing.T) {
	dir := t.TempDir()
	store := SaveStore{Dir: dir}
	now := time.Unix(100, 0)
	clock := func() time.Time { return now }

	first := currentPoolFixtureFront(t, 91, 92)
	first.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "first-options.txt")}
	firstApp := first.App("first game slots")
	first.ConfigureSaveSeams(firstApp, store, OriginalStore{}, clock)
	if err := firstApp.OpenMission(first.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := firstApp.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	if got := first.savedGame(); got.Edition().SlotWord != "rom1" {
		t.Fatalf("the first front runs %q", got)
	}

	legacyRaw := quickFixtureRaw(t)
	if err := quickSlots.write(store, nil, 0, 50, legacyRaw, nil); err != nil {
		t.Fatal(err)
	}
	if err := timedSlots.write(store, nil, 1, 60, legacyRaw, nil); err != nil {
		t.Fatal(err)
	}
	legacy := map[string][]byte{}
	for name, raw := range quickSlotsPerGameFiles(t, dir) {
		if strings.HasPrefix(name, "quick-save-") || strings.HasPrefix(name, "timed-autosave-") {
			legacy[name] = raw
		}
	}
	if len(legacy) != 4 {
		t.Fatalf("legacy files %d", len(legacy))
	}

	f2 := func(f *FrontEnd, app *ui.App, n int) {
		for range n {
			if err := app.HeadlessKey("f2"); err != nil {
				t.Fatal(err)
			}
		}
	}
	timed := func(app *ui.App, n int) {
		for range n {
			now = now.Add(5 * time.Minute)
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
	}
	f2(first, firstApp, 3)
	timed(firstApp, 3)

	second, secondApp, screen, _ := secondBagFront(t, secondBagSample())
	second.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "second-options.txt")}
	second.ConfigureSaveSeams(secondApp, store, OriginalStore{}, clock)
	secondBagEnter(t, second, secondApp, screen)
	if got := second.savedGame(); got.Edition().SlotWord != "rom2" {
		t.Fatalf("the second front runs %q", got)
	}
	f2(second, secondApp, 3)
	timed(secondApp, 3)

	files := quickSlotsPerGameFiles(t, dir)
	for _, game := range []string{"rom1", "rom2"} {
		for _, kind := range []string{"quick-save", "timed-autosave"} {
			for n := 1; n <= 3; n++ {
				for _, ext := range []string{".sav", ".quick-owner", ".timed-owner"} {
					name := fmt.Sprintf("%s-%s-%d%s", game, kind, n, ext)
					want := strings.HasPrefix(kind, "quick") == (ext == ".quick-owner") || ext == ".sav"
					if _, ok := files[name]; ok != want {
						t.Errorf("%s present %v, want %v", name, ok, want)
					}
				}
			}
		}
	}
	for name, raw := range legacy {
		if !bytes.Equal(files[name], raw) {
			t.Errorf("the older file %s changed", name)
		}
	}
	slotFiles := 0
	for name := range files {
		if !strings.HasPrefix(name, "game") {
			slotFiles++
		}
	}
	if slotFiles != len(legacy)+24 {
		t.Errorf("%d slot files, want %d", slotFiles, len(legacy)+24)
	}
	if bytes.Equal(files["rom1-quick-save-1.sav"], files["rom2-quick-save-1.sav"]) {
		t.Error("the two games' first quick saves are one file")
	}

	rom1Slots := readQuickSaveSlots(quickSlots.forGame(base.GameROM1), dir)
	raw, err := newestQuickSave(rom1Slots)
	if err != nil || !bytes.Equal(raw, files["rom1-quick-save-3.sav"]) {
		t.Fatalf("the first game's newest quick save is not its own third: %v", err)
	}
	live := first.live
	if err := firstApp.HeadlessKey("f3"); err != nil {
		t.Fatal(err)
	}
	if first.live == live {
		t.Fatal("F3 of the first game loaded nothing")
	}

	wantRows := func(game string) []string {
		var out []string
		for n := 1; n <= 3; n++ {
			out = append(out, fmt.Sprintf("%s-quick-save-%d.sav", game, n), fmt.Sprintf("%s-timed-autosave-%d.sav", game, n))
		}
		return out
	}
	rows1 := quickSlotsPerGameNames(ownGameSaves(dir, 0, base.GameROM1))
	rows2 := quickSlotsPerGameNames(ownGameSaves(dir, 0, base.GameROM2))
	for _, name := range wantRows("rom1") {
		if !slices.Contains(rows1, name) || slices.Contains(rows2, name) {
			t.Errorf("%s: first list %v, second list %v", name, slices.Contains(rows1, name), slices.Contains(rows2, name))
		}
	}
	for _, name := range wantRows("rom2") {
		if !slices.Contains(rows2, name) || slices.Contains(rows1, name) {
			t.Errorf("%s: first list %v, second list %v", name, slices.Contains(rows1, name), slices.Contains(rows2, name))
		}
	}
	for _, name := range []string{"quick-save-1.sav", "timed-autosave-2.sav"} {
		if !slices.Contains(rows1, name) || slices.Contains(rows2, name) {
			t.Errorf("older %s belongs to the first game: first list %v, second list %v", name, slices.Contains(rows1, name), slices.Contains(rows2, name))
		}
	}
	for _, hidden := range secondSaveHidden() {
		if strings.Contains(hidden, "quick-save-1") && !slices.Contains([]string{"quick-save-1.sav", "rom1-quick-save-1.sav", "rom2-quick-save-1.sav"}, hidden) {
			t.Errorf("hidden name %s", hidden)
		}
	}
	for _, name := range []string{"quick-save-1.sav", "rom1-quick-save-1.sav", "rom2-timed-autosave-3.sav"} {
		if !slices.Contains(secondSaveHidden(), name) {
			t.Errorf("the save page does not hide %s", name)
		}
	}

	for _, name := range []string{"quick-save-1", "timed-autosave-2", "rom1-quick-save-2", "rom2-timed-autosave-1"} {
		remove, err := prepareLoadDelete(dir, saveDialogDeleteToken(name+".sav"), nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := remove(); err != nil {
			t.Fatal(err)
		}
		ext := ".quick-owner"
		if strings.Contains(name, "timed") {
			ext = ".timed-owner"
		}
		for _, gone := range []string{name + ".sav", name + ext} {
			if _, err := os.Lstat(filepath.Join(dir, gone)); !os.IsNotExist(err) {
				t.Errorf("%s remains after the delete: %v", gone, err)
			}
		}
	}
}
