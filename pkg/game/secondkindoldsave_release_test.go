package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

// withoutKindSection is a World byte form as an older build wrote it: the
// wrappers that follow the kind section (structure owners, wide registers) and
// the kind section itself are cut off.
func withoutKindSection(t *testing.T, world []byte) []byte {
	t.Helper()
	cut := func(tag string) bool {
		end := len(world)
		if end < 9 || string(world[end-4:]) != tag {
			return false
		}
		span := int(binary.LittleEndian.Uint32(world[end-9:]))
		out := append([]byte(nil), world[:end-9-span]...)
		out[0] = world[end-5]
		world = out
		return true
	}
	for cut("SOW1") || cut("WRG1") {
	}
	if !cut("AKW1") {
		t.Fatalf("the World form does not end in a kind section")
	}
	return world
}

// A second-game mission SAVE an older build wrote holds no kind section: its
// LOAD takes every actor's kind word from the actor's own record, as the
// original's file does.
func TestReleaseSecondOldMissionSaveTakesKindWordsFromRecords(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(secondCompanionCorpus, "rom2-gog-original-2", "game0012.sav"))
	if err != nil {
		t.Skipf("no corpus file: %v", err)
	}
	load := func(name string, data []byte) *FrontEnd {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
		f := secondGameFront(t)
		app := f.App("second-old-kinds")
		app.Layout(1024, 768)
		f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
		_, list, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
		groundAppLoad(t, app, list, name)
		return f
	}
	f := load("game0012.sav", raw)
	s, label, err := f.Snapshot(f.live != nil)
	if err != nil {
		t.Fatal(err)
	}
	s.World = withoutKindSection(t, s.World)
	old, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	type key struct{ class, index uint16 }
	id := func(e sim.Entity) key {
		return key{uint16(e.SourceBinding.Class), uint16(e.SourceBinding.ArchiveIndex)}
	}
	want := map[key]uint16{}
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.Class != 0 {
			want[id(e)] = e.KindWord
		}
	}
	cold := load("game0013.sav", old)
	checked := 0
	for _, e := range cold.live.world.Entities() {
		if k, ok := want[id(e)]; ok && e.SourceBinding.Class != 0 {
			checked++
			if e.KindWord != k {
				t.Errorf("actor %d (record %d) holds kind word %d, its record holds %d", e.ID, e.SourceBinding.ArchiveIndex, e.KindWord, k)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no actor compared")
	}
}
