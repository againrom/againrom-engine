package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// The reference listers are the three scanners as they stood before the shared
// scanner: each reads and opens every file on every call.
func refSaveStoreList(s SaveStore) []SaveEntry {
	ents, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil
	}
	var out []SaveEntry
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), originalSaveExt) {
			continue
		}
		fi, err := e.Info()
		if err != nil || fi.Size() < 0 || fi.Size() > maxSaveBytes {
			continue
		}
		b, err := s.Read(e.Name())
		if err != nil {
			continue
		}
		label, err := OriginalSaveLabel(b, s.Selector)
		if err != nil {
			continue
		}
		out = append(out, SaveEntry{Name: e.Name(), Label: label, Mod: fi.ModTime()})
	}
	refSort(out)
	return out
}

func refOriginalList(s OriginalStore) []SaveEntry {
	ents, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil
	}
	var out []SaveEntry
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), originalSaveExt) {
			continue
		}
		b, err := s.Read(e.Name())
		if err != nil {
			continue
		}
		label, err := OriginalSaveLabel(b, s.Selector)
		if err != nil {
			continue
		}
		var mod time.Time
		if fi, err := e.Info(); err == nil {
			mod = fi.ModTime()
		}
		out = append(out, SaveEntry{Name: e.Name(), Label: label, Mod: mod})
	}
	refSort(out)
	return out
}

func refOwnGameSaves(dir string, selector int, g base.Game) []SaveEntry {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []SaveEntry
	for _, e := range ents {
		if e.IsDir() || !IsOriginal(e.Name()) {
			continue
		}
		fi, err := e.Info()
		if err != nil || !fi.Mode().IsRegular() || fi.Size() > maxSaveBytes {
			continue
		}
		b, err := (SaveStore{Dir: dir}).Read(e.Name())
		if err != nil {
			continue
		}
		sf, err := sav.Open(b)
		if err != nil || validateOriginalGame(b, g) != nil {
			continue
		}
		out = append(out, SaveEntry{Name: e.Name(), Label: asciiLabel(sf.Label, selector), Mod: fi.ModTime()})
	}
	refSort(out)
	return out
}

func refSort(out []SaveEntry) {
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Mod.Equal(out[j].Mod) {
			return out[i].Name > out[j].Name
		}
		return out[i].Mod.After(out[j].Mod)
	})
}

// scanWitnessDirs are the directories every caller is compared over: a
// synthetic one, a missing one, and each directory in AGAINROM_SCAN_DIRS.
func scanWitnessDirs(t *testing.T) []string {
	t.Helper()
	dir := t.TempDir()
	writeScanSave(t, dir, "game0001.sav", labelledSave(3, "First"), 3*time.Hour)
	writeScanSave(t, dir, "game0002.sav", labelledSave(0, "Between \xe9"), 2*time.Hour)
	writeScanSave(t, dir, "tie-a.sav", labelledSave(1, "Tie"), time.Hour)
	writeScanSave(t, dir, "tie-b.sav", labelledSave(1, "Tie"), time.Hour)
	writeScanSave(t, dir, "broken.sav", []byte("corrupt"), 30*time.Minute)
	writeScanSave(t, dir, "UPPER.SAV", labelledSave(5, "Upper"), 20*time.Minute)
	writeScanSave(t, dir, "notes.txt", []byte("not a save"), time.Hour)
	if err := os.Mkdir(filepath.Join(dir, "sub.sav"), 0o700); err != nil {
		t.Fatal(err)
	}
	dirs := []string{dir, filepath.Join(dir, "missing")}
	for _, d := range filepath.SplitList(os.Getenv("AGAINROM_SCAN_DIRS")) {
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

func scanCallers(dir string, selector int) map[string]any {
	out := map[string]any{}
	out["SaveStore.List"], _ = (SaveStore{Dir: dir, Selector: selector}).List()
	out["OriginalStore.List"] = (OriginalStore{Dir: dir, Selector: selector}).List()
	for _, g := range []base.Game{base.GameROM1, base.GameROM2} {
		out["ownGameSaves "+string(g)] = ownGameSaves(dir, selector, g)
		listed, err := listOwnSaveDirectory(dir, nil, selector, g)
		out["listOwnSaveDirectory "+string(g)] = fmt.Sprint(listed, err)
	}
	listed, err := listSaveDirectory(dir, nil, selector)
	out["listSaveDirectory"] = fmt.Sprint(listed, err)
	return out
}

func refCallers(dir string, selector int) map[string]any {
	out := map[string]any{}
	out["SaveStore.List"] = refSaveStoreList(SaveStore{Dir: dir, Selector: selector})
	out["OriginalStore.List"] = refOriginalList(OriginalStore{Dir: dir, Selector: selector})
	resolved, refusal := namedSaveDirectory(dir, nil)
	if refusal != nil {
		// The named-directory refusal (an install directory) is the caller's
		// answer on main, before any file is listed.
		for _, g := range []base.Game{base.GameROM1, base.GameROM2} {
			out["listOwnSaveDirectory "+string(g)] = fmt.Sprint(ui.SaveDirectory{}, refusal)
		}
		out["listSaveDirectory"] = fmt.Sprint(ui.SaveDirectory{}, refusal)
		return out
	}
	for _, g := range []base.Game{base.GameROM1, base.GameROM2} {
		own := refOwnGameSaves(dir, selector, g)
		out["ownGameSaves "+string(g)] = own
		var entries []ui.SaveEntry
		for _, e := range own {
			entries = append(entries, ui.SaveEntry{Name: e.Name, Label: e.Label, Detail: saveRowDetail(e.Label, e.Mod)})
		}
		out["listOwnSaveDirectory "+string(g)] = fmt.Sprint(ui.SaveDirectory{Path: resolved, Entries: entries}, error(nil))
	}
	browser := ui.SaveDirectory{Path: resolved}
	if ents, err := os.ReadDir(dir); err == nil {
		for _, e := range ents {
			if e.IsDir() {
				browser.Directories = append(browser.Directories, e.Name())
			}
		}
	}
	for _, e := range refSaveStoreList(SaveStore{Dir: dir, Selector: selector}) {
		browser.Entries = append(browser.Entries, ui.SaveEntry{Name: e.Name, Label: e.Label})
	}
	out["listSaveDirectory"] = fmt.Sprint(browser, error(nil))
	return out
}

func nilToEmpty(v any) any {
	if s, ok := v.([]SaveEntry); ok && len(s) == 0 {
		return []SaveEntry(nil)
	}
	return v
}

// TestSharedScannerKeepsEveryCallersRows compares each former scanner's output
// with the shared scanner's, per caller, cold and cached, over every witness
// directory; the existence check is compared with the list for each game.
func TestSharedScannerKeepsEveryCallersRows(t *testing.T) {
	for _, dir := range scanWitnessDirs(t) {
		if _, err := os.ReadDir(dir); err != nil {
			continue // a missing directory is an error path, not a list
		}
		for _, selector := range []int{0, 1} {
			want := refCallers(dir, selector)
			for pass := 0; pass < 2; pass++ {
				got := scanCallers(dir, selector)
				for name, w := range want {
					if !reflect.DeepEqual(nilToEmpty(got[name]), nilToEmpty(w)) {
						t.Errorf("%s selector %d pass %d caller %s differs:\n got  %v\n want %v", dir, selector, pass, name, got[name], w)
					}
				}
				for _, g := range []base.Game{base.GameROM1, base.GameROM2} {
					first := len(scanSaves(saveScan{Dir: dir, Bounded: true, Regular: true, OfGame: true, Game: g, First: true})) != 0
					if first != (len(refOwnGameSaves(dir, selector, g)) != 0) {
						t.Errorf("%s game %s: existence check %v disagrees with the list", dir, g, first)
					}
				}
				if first := len(scanSaves(saveScan{Dir: dir, Bounded: true, First: true})) != 0; first != (len(refSaveStoreList(SaveStore{Dir: dir})) != 0) {
					t.Errorf("%s: first-game existence check %v disagrees with the list", dir, first)
				}
				if first := len(scanSaves(saveScan{Dir: dir, First: true})) != 0; first != (len(refOriginalList(OriginalStore{Dir: dir})) != 0) {
					t.Errorf("%s: install existence check %v disagrees with the list", dir, first)
				}
			}
		}
	}
}

// TestSharedScannerLossControls shows the comparison can fail: a scan with the
// game filter dropped, or with the order reversed, differs from the reference.
func TestSharedScannerLossControls(t *testing.T) {
	var noGameFilter, reversed bool
	for _, dir := range scanWitnessDirs(t) {
		want := refOwnGameSaves(dir, 0, base.GameROM2)
		if all := scanSaves(saveScan{Dir: dir, Bounded: true, Regular: true}); len(all) != len(want) {
			noGameFilter = true
		}
		ordered := refSaveStoreList(SaveStore{Dir: dir})
		scanned := scanSaves(saveScan{Dir: dir, Bounded: true})
		for i := range scanned {
			if i < len(ordered) && scanned[len(scanned)-1-i].Name != ordered[i].Name {
				reversed = true
			}
		}
	}
	if !noGameFilter {
		t.Error("dropping the game filter changed no list: the witness cannot see a wrong filter")
	}
	if !reversed {
		t.Error("reversing the order changed no list: the witness cannot see a wrong order")
	}
}
