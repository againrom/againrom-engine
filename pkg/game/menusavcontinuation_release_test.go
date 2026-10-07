package game

import (
	"bytes"
	"fmt"
	"math/bits"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func menuSAVE(t *testing.T, f *FrontEnd, app *ui.App, orig OriginalStore) (SaveStore, string, []byte) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, orig, func() time.Time { return time.Unix(1, 0) })
	app.SetSaveSeams(save, list, load)
	if _, _, open := f.LiveNotice(); open {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(app.Screen(), err)
	}
	if app.Screen() != ui.ScreenGameMenu {
		t.Fatal("not the SAVE menu", app.Screen())
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
		t.Fatal("menu SAVE wrote", entries, err, app.HeadlessMessage())
	}
	raw, err := store.Read(entries[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	return store, entries[0].Name, raw
}

func loadAlteredSAV(t *testing.T, raw []byte, alter func(*sav.DocumentData) bool) *FrontEnd {
	t.Helper()
	path := filepath.Join(t.TempDir(), "altered.sav")
	if err := os.WriteFile(path, alterSAV(t, raw, alter), 0o600); err != nil {
		t.Fatal(err)
	}
	return loadAreaContinuation(t, path)
}

func alterSAV(t *testing.T, raw []byte, alter func(*sav.DocumentData) bool) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !alter(&doc) {
		t.Fatal("loss control found no field to alter")
	}
	if doc, _, err = sav.ReindexDocumentData(doc); err != nil {
		t.Fatal(err)
	}
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func setSavedSpellRange(doc *sav.DocumentData, key uint32, value uint8) bool {
	for i := range doc.Objects {
		if doc.Objects[i].Class != "Spell" {
			continue
		}
		if this, err := savedStructureValue(&doc.Objects[i], "This"); err == nil && this == key {
			return savedStructureSetValue(&doc.Objects[i], "S09", uint32(value)) == nil
		}
	}
	return false
}

func requireSavedBook(t *testing.T, known uint32, spells []sav.SavedSpell, want uint32, book sim.Spellbook) {
	t.Helper()
	if known != want || len(spells) != bits.OnesCount32(want) {
		t.Fatalf("saved membership %#x with %d spells, live %#x", known, len(spells), want)
	}
	for _, spell := range spells {
		if got := (sim.BookSpell{Range: spell.Range, Defensive: spell.Defensive, ManaCost: spell.ManaCost}); got != book.Slots[spell.ID-1] {
			t.Fatalf("saved spell %d %+v, live %+v", spell.ID, got, book.Slots[spell.ID-1])
		}
	}
}

func savedCityMage(t *testing.T, raw []byte, name string) sav.CityCharacter {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	city, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range city.Roster() {
		if c.Name == name && c.HasSpellbook {
			return c
		}
	}
	t.Fatalf("saved city has no book for %q", name)
	return sav.CityCharacter{}
}

func openLocalTownSAV(t *testing.T, f *FrontEnd, dir, name string) *ui.App {
	t.Helper()
	f.SetDeterministicFrames(true)
	app := f.App("local town SAV")
	app.Layout(1024, 768)
	save, list, load := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	label := ""
	for _, row := range list() {
		if row.Name == localOriginalSaveToken(name) {
			label = row.Label
		}
	}
	if label == "" {
		t.Fatalf("save %s is absent from the load window", name)
	}
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(label); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatalf("town LOAD at %s: %v %s", app.Screen(), err, app.HeadlessMessage())
	}
	return app
}

func mageMissionTrace(t *testing.T, f *FrontEnd, memberID string) []byte {
	t.Helper()
	if err := f.App("trained mage mission").OpenMission(f.MissionOpener(30)); err != nil {
		t.Fatal(err)
	}
	var id sim.EntityID
	for i, p := range f.live.mission.party {
		if p.ID == memberID {
			id = f.live.mission.ids[i]
		}
	}
	if id == 0 {
		t.Fatal("mission member absent")
	}
	var trace bytes.Buffer
	for tick := range 64 {
		e, _ := f.live.entity(id)
		fmt.Fprintf(&trace, "%d %#x %v %d %d\n", tick, e.KnownSpells, e.Book, e.Mana, e.HP)
		f.live.tick()
	}
	return trace.Bytes()
}

func openOriginalSAVApp(t *testing.T, f *FrontEnd, source []byte, name string) (*ui.App, string) {
	t.Helper()
	return openOriginalSAVAppAt(t, f, source, name, 1024, 768)
}

// openOriginalSAVAppAt is openOriginalSAVApp in a window of width by height.
func openOriginalSAVAppAt(t *testing.T, f *FrontEnd, source []byte, name string, width, height int) (*ui.App, string) {
	t.Helper()
	f.SetDeterministicFrames(true)
	app := f.App("original SAV")
	app.Layout(width, height)
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, source, 0600); err != nil {
		t.Fatal(err)
	}
	save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, name)
	return app, path
}

func townSAVRoundTrip(t *testing.T, f *FrontEnd) ([]byte, *sav.CityProvenance, *FrontEnd) {
	t.Helper()
	dir := t.TempDir()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatal("ordinary town SAVE", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != name || filepath.Ext(name) != ".sav" {
		t.Fatal("ordinary town SAVE wrote", entries, err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	city, err := file.CityProvenance()
	if err != nil || city == nil {
		t.Fatal("town SAV has no city", err)
	}
	loaded := releaseFront(t)
	openLocalTownSAV(t, loaded, dir, name)
	return raw, city, loaded
}

func schoolTrain(f *FrontEnd, id string) string {
	s := f.TownScreen().(*townScreen)
	s.room, s.schoolCell = roomSchool, 0
	for i, m := range f.Carried {
		if m.ID == id {
			s.shopMember = i
		}
	}
	return s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}, false).Msg
}
