package game

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// labelledSave is a synthetic whole save whose label region holds label.
func labelledSave(mission uint32, label string) []byte {
	b := savedFile(mission, nil)
	end := binary.LittleEndian.Uint32(b[4:])
	copy(b[end:], label)
	return b
}

func writeScanSave(t *testing.T, dir, name string, data []byte, age time.Duration) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
}

// referenceLocalRows lists dir as the first game's load window did before the
// label reader was shared: every file read and opened on every call.
func referenceLocalRows(dir string, selector int) []ui.SaveEntry {
	ents, _ := os.ReadDir(dir)
	type row struct {
		name, label string
		mod         time.Time
	}
	var rows []row
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), originalSaveExt) {
			continue
		}
		fi, err := e.Info()
		if err != nil || fi.Size() > maxSaveBytes {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		label, err := OriginalSaveLabel(b, selector)
		if err != nil {
			continue
		}
		rows = append(rows, row{e.Name(), label, fi.ModTime()})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].mod.Equal(rows[j].mod) {
			return rows[i].name > rows[j].name
		}
		return rows[i].mod.After(rows[j].mod)
	})
	var out []ui.SaveEntry
	for _, r := range rows {
		out = append(out, ui.SaveEntry{Name: localOriginalSaveToken(r.name), Label: drawableLabel(r.label)})
	}
	return out
}

func TestMenuLoadGateReadsOnlyUntilTheFirstOpeningSave(t *testing.T) {
	dir := t.TempDir()
	writeScanSave(t, dir, "a.sav", []byte("not a save at all"), 5*time.Hour)
	for i, name := range []string{"b.sav", "c.sav", "d.sav", "e.sav"} {
		writeScanSave(t, dir, name, labelledSave(uint32(i), name), time.Duration(i)*time.Hour)
	}
	f := &FrontEnd{}
	store := SaveStore{Dir: dir}
	before := saveFactsReads()
	for i := 0; i < 3; i++ {
		if !f.anyLoadableSaveRow(store, OriginalStore{}) {
			t.Fatal("the gate reported no save")
		}
	}
	if got := saveFactsReads() - before; got != 2 {
		t.Fatalf("three gate checks read %d files in full; want 2 (the corrupt one and the first save)", got)
	}
	empty := t.TempDir()
	writeScanSave(t, empty, "x.sav", []byte("junk"), 0)
	if f.anyLoadableSaveRow(SaveStore{Dir: empty}, OriginalStore{}) {
		t.Fatal("a directory with only a corrupt file offered Load")
	}
	if f.anyLoadableSaveRow(SaveStore{Dir: filepath.Join(dir, "missing")}, OriginalStore{}) {
		t.Fatal("a missing directory offered Load")
	}
}

func TestLoadPageRowsMatchTheUncachedListing(t *testing.T) {
	dir := t.TempDir()
	writeScanSave(t, dir, "game0001.sav", labelledSave(3, "First"), 3*time.Hour)
	writeScanSave(t, dir, "game0002.sav", labelledSave(0, "Between \xe9"), 2*time.Hour)
	writeScanSave(t, dir, "broken.sav", []byte("corrupt"), time.Hour)
	writeScanSave(t, dir, "notes.txt", []byte("not a save"), time.Hour)
	if err := os.Mkdir(filepath.Join(dir, "sub.sav"), 0o700); err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{}
	store := SaveStore{Dir: dir}
	check := func(stage string) {
		t.Helper()
		want := referenceLocalRows(dir, store.Selector)
		for pass := 0; pass < 2; pass++ {
			if got := f.saveRows(store, OriginalStore{}, true); !reflect.DeepEqual(got, want) {
				t.Fatalf("%s pass %d: rows %v, uncached listing %v", stage, pass, got, want)
			}
		}
	}
	check("first listing")
	reads := saveFactsReads()
	f.saveRows(store, OriginalStore{}, true)
	if saveFactsReads() != reads {
		t.Fatalf("a repeat listing read %d files", saveFactsReads()-reads)
	}
	// A save written after the first listing appears with its label, at the top.
	writeScanSave(t, dir, "game0003.sav", labelledSave(7, "Fresh"), 0)
	check("new save")
	rows := f.saveRows(store, OriginalStore{}, true)
	if len(rows) == 0 || rows[0].Name != localOriginalSaveToken("game0003.sav") || !strings.Contains(rows[0].Label, "Fresh - mission 7") {
		t.Fatalf("new save row = %v", rows)
	}
	// The same file rewritten with the same size and a new time is read again.
	writeScanSave(t, dir, "game0003.sav", labelledSave(8, "Fresh"), time.Minute)
	check("rewritten save")
	// A broken file repaired in place joins the list.
	writeScanSave(t, dir, "broken.sav", labelledSave(1, "Mended"), 30*time.Minute)
	check("mended save")
	if err := os.Remove(filepath.Join(dir, "game0001.sav")); err != nil {
		t.Fatal(err)
	}
	check("deleted save")
}

func TestLoadPageKeepsStoredLabelsWithoutAMangleSecondListing(t *testing.T) {
	dir := t.TempDir()
	writeScanSave(t, dir, "game0001.sav", labelledSave(2, "Plain"), time.Hour)
	f := &FrontEnd{}
	store := SaveStore{Dir: dir}
	before := saveFactsReads()
	raw := f.saveRows(store, OriginalStore{}, false)
	f.saveRows(store, OriginalStore{}, true)
	if got := saveFactsReads() - before; got != 1 {
		t.Fatalf("two listings of one file read %d files in full", got)
	}
	if len(raw) != 1 || raw[0].Label != "Plain - mission 2" {
		t.Fatalf("rows = %v", raw)
	}
}

// TestSharedLabelReaderMatchesOpen pins that a cached label is the label
// OriginalSaveLabel gives for the same bytes.
func TestSharedLabelReaderMatchesOpen(t *testing.T) {
	for _, mission := range []uint32{0, 1, 29} {
		b := labelledSave(mission, fmt.Sprintf("Row %d", mission))
		want, err := OriginalSaveLabel(b, 0)
		if err != nil {
			t.Fatal(err)
		}
		sf, err := sav.Open(b)
		if err != nil {
			t.Fatal(err)
		}
		if got := originalSaveLabelOf(sf.Label, sf.Head.Mission, 0); got != want {
			t.Fatalf("mission %d: %q, want %q", mission, got, want)
		}
	}
}
