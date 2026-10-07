package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseSavedEnemyHumanPortraitsAndBooks(t *testing.T) {
	path, _ := groundCorpusFile(t, "2026-09-14/mission111-enemy-dolls.ags", "548c9c2eb6d0e05e86215adf6cb47f5197896446eecc99ba288c5fd50e5dff6f")
	store := SaveStore{Dir: filepath.Dir(path)}
	name := filepath.Base(path)
	for generation := 0; generation < 2; generation++ {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		a := f.App("enemy portraits")
		_, list, load := agsSaveSeams(f, store, OriginalStore{}, nil)
		a.SetSaveSeams(nil, list, load)
		groundAppLoad(t, a, list, name)
		a.Layout(1024, 768)
		releasePauseMission(t, f, a)
		mw := f.live
		for i := range mw.fog.visible {
			mw.fog.visible[i], mw.fog.explored[i] = 1, 1
		}
		mw.push()
		before := mw.world.Hash()
		for _, tc := range []struct {
			id     sim.EntityID
			dir    data.FigureDir
			face   int
			packed uint8
		}{
			{44, data.FigureDirManFighter, 24, 24},
			{30, data.FigureDirWomanFighter, 10, 138},
			{49, data.FigureDirWomanMage, 3, 131},
		} {
			e := releaseEntityByID(t, mw.world, tc.id)
			if e.SourceBinding.Class != 2 || e.SourceBinding.Face != tc.packed {
				t.Fatalf("actor%d source changed: %+v", tc.id, e.SourceBinding)
			}
			eq := mw.equipmentOf(tc.id)
			if eq == (data.Equipment{}) {
				t.Fatalf("actor%d fixture lost worn equipment", tc.id)
			}
			want, _ := composeUnitFigure(mw.archive(), eq, figureID{Dir: tc.dir, Face: tc.face})
			bare, _ := composeUnitFigure(mw.archive(), data.Equipment{}, figureID{Dir: tc.dir, Face: tc.face})
			if want == nil || imagesEqual(want, bare) {
				t.Fatalf("actor%d installed worn layers are absent", tc.id)
			}
			if !imagesEqual(mw.inspectionUnitPicture(uint32(tc.id)), want) || !imagesEqual(mw.unitPicture(tc.id, e.Class), want) {
				t.Fatalf("generation%d actor%d hover/selection lost its equipped figure", generation, tc.id)
			}
			inspectionCentre(mw, int(e.X), int(e.Y))
			mw.push()
			releaseHoverInspection(t, a, mw, ui.InspectionSubject{Kind: ui.InspectionUnit, ID: uint32(tc.id)})
			pic, _, err := a.HeadlessCharacterPane()
			if err != nil {
				t.Fatal(err)
			}
			checkInspectionFigure(t, pic, want, "saved enemy equipped figure")
			x, y, err := mw.view.InspectionPoint(ui.InspectionSubject{Kind: ui.InspectionUnit, ID: uint32(tc.id)})
			if err != nil {
				t.Fatal(err)
			}
			// Cancel an inherited spell first, then clear the owned selection;
			// with an owned actor selected, left click would order an attack.
			for range 2 {
				if err := a.HeadlessPointer("right-press", x, y); err != nil {
					t.Fatal(err)
				}
				if err := a.HeadlessPointer("right-release", x, y); err != nil {
					t.Fatal(err)
				}
			}
			if err := a.HeadlessSelectEntity(uint32(tc.id)); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessPointer("hover", 0, 0); err != nil {
				t.Fatal(err)
			}
			pic, _, err = a.HeadlessCharacterPane()
			if err != nil {
				t.Fatal(err)
			}
			checkInspectionFigure(t, pic, want, "selected enemy equipped figure")
		}
		mage := releaseEntityByID(t, mw.world, 49)
		book := spellbookOf(sim.Rules{}, mage, mw.world.Spells(), mw.spellNames, mw.view.Words(), nil)
		if mage.KnownSpells != 0x2042 || mage.Book.State != sim.BookPresent || len(book) != 3 {
			t.Fatalf("mage book: %x %+v %v", mage.KnownSpells, mage.Book, book)
		}
		for i, id := range []uint32{1, 6, 13} {
			if book[i].ID != id || mage.Book.Slots[id-1].Range < 2 {
				t.Fatalf("enemy mage spell%d not initialized: %+v", id, mage.Book)
			}
		}
		if mw.world.Hash() != before {
			t.Fatal("portrait inspection changed simulation")
		}
		// Preserve the untouched owner input; re-save only into a new directory.
		if generation == 0 {
			store = SaveStore{Dir: t.TempDir()}
			var err error
			name, err = nativeCheckpoint1170(f, store)(true)
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := mw.world.HeadlessPlace(81, mage.X-3, mage.Y); err != nil {
			t.Fatal(err)
		}
		cast := false
		for tick := 0; tick < 200 && !cast; tick++ {
			for _, event := range sim.StepObserved(mw.world, nil) {
				if event.Caster == mage.ID && !event.Weapon {
					if event.Spell != 1 && event.Spell != 6 && event.Spell != 13 {
						t.Fatalf("enemy cast unknown spell: %+v", event)
					}
					t.Logf("generation%d enemy mage cast spell%d at actor%d after%d ticks", generation, event.Spell, event.Target, tick+1)
					cast = true
				}
			}
		}
		if !cast {
			t.Fatal("enemy mage never used its initialized book")
		}
	}
}
