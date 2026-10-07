package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestReleaseFunctionKeysSaveLoadAndDiplomacy(t *testing.T) {
	for _, inTown := range []bool{false, true} {
		t.Run(map[bool]string{false: "campaign map", true: "town"}[inTown], func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			store := SaveStore{Dir: t.TempDir()}
			app := f.App("1080-function-keys")
			save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
			app.SetSaveSeams(save, list, load)
			seedCount := 0
			wantMission := 10
			if inTown {
				// Enter the real town via the existing saved-town route. The
				// seed lives in a test directory, never in the install.
				f.Carried = f.NextParty()
				f.arriveInTown()
				if _, err := save(false); err != nil {
					t.Fatal(err)
				}
				if err := app.HeadlessKey("load"); err != nil {
					t.Fatal(err)
				}
				if err := app.HeadlessKey("enter"); err != nil || app.Screen() != ui.ScreenTown {
					t.Fatalf("town entry: screen=%v err=%v", app.Screen(), err)
				}
				seedCount, wantMission = 1, 0
			} else {
				if err := app.OpenMission(f.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
				if err := app.HeadlessKey("f3"); err != nil || app.Screen() != ui.ScreenLoad || app.HeadlessMessage() != "no saved games" {
					t.Fatalf("empty F3: screen=%v message=%q err=%v", app.Screen(), app.HeadlessMessage(), err)
				}
				for i := 0; i < 2; i++ {
					if err := app.HeadlessKey("escape"); err != nil {
						t.Fatal(err)
					}
				}
			}
			origin := app.Screen()
			if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenGameMenu {
				t.Fatalf("F2: screen=%v err=%v", app.Screen(), err)
			}
			if got := app.HeadlessMessage(); got != f.Words.SaveAcknowledgement {
				t.Fatalf("F2 acknowledgement=%q want installed %q", got, f.Words.SaveAcknowledgement)
			}
			entries, err := store.List()
			if err != nil || len(entries) != seedCount+1 {
				t.Fatalf("F2 native store: count=%d want=%d error=%v", len(entries), seedCount+1, err)
			}
			raw, err := store.Read(entries[0].Name)
			if err != nil {
				t.Fatal(err)
			}
			savedMission := 0
			if IsOriginal(entries[0].Name) {
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				savedMission = int(doc.Head.Mission)
				if (doc.World != nil) != !inTown {
					t.Fatal("F2 changed the mission or town save point")
				}
			} else {
				if !inTown {
					t.Fatal("ordinary mission F2 did not write SAV")
				}
				snapshot, _, err := DecodeSave(raw)
				if err != nil {
					t.Fatal(err)
				}
				savedMission = snapshot.Mission
			}
			if savedMission != wantMission {
				t.Fatalf("F2 saved mission=%d want=%d", savedMission, wantMission)
			}
			if err := app.HeadlessKey("escape"); err != nil || app.Screen() != origin {
				t.Fatalf("Save return: %v error=%v", app.Screen(), err)
			}
			if err := app.HeadlessKey("f3"); err != nil || app.Screen() != ui.ScreenLoad {
				t.Fatalf("F3: %v error=%v", app.Screen(), err)
			}
			if err := app.HeadlessKey("enter"); err != nil || app.Screen() != origin {
				t.Fatalf("F3 restored screen=%v want=%v error=%v message=%q", app.Screen(), origin, err, app.HeadlessMessage())
			}
			t.Logf("F2 wrote one generated save (mission %d); F3 restored %s through production SaveSeams", savedMission, origin)
		})
	}
	t.Run("standalone", func(t *testing.T) {
		f := releaseFront(t)
		app := f.App("1080-standalone")
		store := SaveStore{Dir: t.TempDir()}
		app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
		row := -1
		for i, entry := range f.Maps {
			if entry.Mission == 0 && entry.Choosable() {
				row = i
				break
			}
		}
		if row < 0 {
			t.Fatal("no installed standalone map")
		}
		if err := app.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence,
			ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
			return f.loadMap(row)
		}); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenMap {
			t.Fatalf("standalone F2: screen=%v error=%v", app.Screen(), err)
		}
		if entries, err := store.List(); err != nil || len(entries) != 0 {
			t.Fatalf("standalone F2 wrote a save: entries=%v error=%v", entries, err)
		}
		if err := app.HeadlessKey("f3"); err != nil || app.Screen() != ui.ScreenGameMenu {
			t.Fatalf("standalone F3: screen=%v error=%v", app.Screen(), err)
		}
		rows := app.HeadlessRows()
		if len(rows) < 2 || rows[0].Text != "DIPLOMACY" {
			t.Fatalf("F3 did not show existing Diplomacy: %+v", rows)
		}
		t.Logf("standalone F2 wrote no save; F3 shows Diplomacy with %d rows", len(rows))
	})
}
