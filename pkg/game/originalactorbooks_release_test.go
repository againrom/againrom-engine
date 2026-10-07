package game

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Both lawful roots replay one unchanged owner save, not two ROM1 recordings.
// The seat's independent published-master archive census identified Brigands'
// Human@27305, mapID32, and these nine-byte Spell body offsets. Assertions read
// literal decoded bytes, never construct expected values with the importer.
func TestReleaseOriginalNonPartySpellbooks1105(t *testing.T) {
	f := releaseFront(t)
	path, payload := groundCorpusFile(t, "2026-08-15/game0017.sav", "eafce5d6575d54fdddc7a35f57531cd3df9317006c80f7c4085866c1b02b4fe0")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	if source.Head.Mission != 40 || len(source.Body) < 28104 {
		t.Fatal("source envelope changed")
	}
	for off, want := range map[int][]byte{28076: {1, 7, 0, 3, 0, 0x70, 0x33, 0xc2, 2}, 28095: {6, 6, 1, 5, 0, 0xe0, 0x34, 0xc2, 2}} {
		if !bytes.Equal(source.Body[off:off+9], want) {
			t.Fatalf("literal Spell body at%d: %x", off, source.Body[off:off+9])
		}
	}
	assert := func(w *sim.World) {
		t.Helper()
		e := poolEntity(t, w, 32)
		if e.X != 123 || e.Y != 23 || e.HP != 35 || e.KnownSpells != 0x42 || e.Book.State != sim.BookPresent || e.Book.Slots[0] != (sim.BookSpell{Range: 7, Defensive: 0, ManaCost: 3}) || e.Book.Slots[5] != (sim.BookSpell{Range: 6, Defensive: 1, ManaCost: 5}) {
			t.Fatalf("natural non-party book: position%d,%d HP%d known%x book%+v", e.X, e.Y, e.HP, e.KnownSpells, e.Book)
		}
	}
	ms, report, err := ResumeOriginalSave(f.Archives.Containers, payload, f.Table, mapload.DifficultyNormal, nil, f.Bodies)
	if err != nil || report.Books.Spells < 2 {
		t.Fatalf("diagnostic natural book %+v %v", report.Books, err)
	}
	assert(ms.World)
	f.SetDeterministicFrames(true)
	app := f.App("1105-natural-books")
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, filepath.Base(path))
	assert(f.live.world)
	e := poolEntity(t, f.live.world, 32)
	for _, id := range f.live.mission.ids {
		if e.ID == id {
			t.Fatal("natural witness is persistent party")
		}
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
		t.Fatalf("ordinary SAVE %+v %v", entries, err)
	}
	fresh := releaseFront(t)
	fresh.SetDeterministicFrames(true)
	freshApp := fresh.App("1105-fresh-natural")
	fsave, flist, fload := nativeContinuationSeams1170(t, fresh, store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(fsave, flist, fload)
	groundAppLoad(t, freshApp, flist, entries[0].Name)
	assert(fresh.live.world)
	if f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatal("fresh native LOAD changed natural book world")
	}
	for range 32 {
		a, b := sim.StepObserved(f.live.world, nil), sim.StepObserved(fresh.live.world, nil)
		if !reflect.DeepEqual(a, b) || f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("natural continuation hash/events diverged")
		}
	}
	t.Logf("game0017 mission40 non-party mapID32: BookLegacy -> BookPresent; literal slots1=(7,0,3),6=(6,1,5); both LOAD doors; ordinary .ags SAVE/fresh App LOAD and32ticks identical")
}
