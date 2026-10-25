package game

import (
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// missionEntryPanels reads the two panel leaves of a written SAV.
func missionEntryPanels(t *testing.T, raw []byte) (inventory, book int32) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, leaf := range []struct {
		path string
		dst  *int32
	}{{"/Inventory/IsOpen", &inventory}, {"/SpellBook/IsOpen", &book}} {
		value, present, err := doc.State.ReadLeaf(leaf.path)
		if err != nil || !present || value.Kind != 2 {
			t.Fatalf("%s: present %v kind %d err %v", leaf.path, present, value.Kind, err)
		}
		*leaf.dst = value.Int32
	}
	return inventory, book
}

// missionEntrySave takes the save the owner kits take: F2, a chooser label,
// SAVE, the file as written.
func missionEntrySave(t *testing.T, f *FrontEnd, app *ui.App, label string) []byte {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	raw := cityRosterF2Save(t, app, store, label)
	if app.Screen() == ui.ScreenGameMenu {
		if err := app.HeadlessGameMenuAction("return"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatalf("SAVE left screen %s, want the map", app.Screen())
	}
	return raw
}

// missionEntryScenario is the kits' New Game route: the menu, the mission
// list entry and a created character, nothing after the mission opens.
func missionEntryScenario(t *testing.T, mission int, class string) HeadlessScenario {
	t.Helper()
	entry, err := ReadHeadlessScenario(filepath.Join("..", "..", "scenarios", "1173-save-dialog.json"))
	if err != nil {
		t.Fatal(err)
	}
	entry.Steps = entry.Steps[:3]
	entry.Steps[1].Target = fmt.Sprintf("Mission %d: %d.alm", mission, mission)
	entry.Steps[2].Character = &HeadlessCharacter{Name: "Warrior", Sex: "Male", Class: "Fighter", Skill: "Blade",
		Stats: map[string]int{"Body": 31, "Reaction": 27, "Mind": 24, "Spirit": 29}}
	if class == "Mage" {
		entry.Steps[2].Character = &HeadlessCharacter{Name: "Kit Mage", Sex: "Male", Class: "Mage", Skill: "Air",
			Stats: map[string]int{"Body": 27, "Reaction": 24, "Mind": 31, "Spirit": 29}}
	}
	return entry
}

func missionEntryFront(t *testing.T, start int, class string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("mission entry book")
	t.Cleanup(app.StopAudio)
	f.ConfigureSaveSeams(app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	if err := RunHeadlessScenario(f, app, missionEntryScenario(t, start, class), io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if f.live == nil || f.liveMission != start {
		t.Fatalf("not in mission %d", start)
	}
	return f, app
}

// missionEntryNext wins the open mission by an imported won session, follows
// the ordinary Victory route and stops in the next campaign mission, either
// straight from the Victory or through the town.
func missionEntryNext(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	start := f.liveMission
	importWonSession(t, f.live.world)
	for frame := 0; frame < 180; frame++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		_, kind, open := f.LiveNotice()
		if !open {
			continue
		}
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
		if kind != ui.NoticeSuccess {
			continue
		}
		for i := 0; i < 4000; i++ {
			if f.live != nil && f.liveMission != start {
				return
			}
			if app.Screen() == ui.ScreenTown && f.townUI.AtTownSquare() {
				if err := app.OpenMission(f.MissionOpener(f.Town.Chapter())); err != nil {
					t.Fatal(err)
				}
				if f.live == nil || f.liveMission == start {
					t.Fatal("the town opened no next mission")
				}
				return
			}
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		t.Fatal("the Victory reached neither the town nor the next mission")
	}
	t.Fatal("the won session displayed no Victory")
}

// TestReleaseMissionEntrySavesTheSpellBookClosed takes the four owner-kit
// mission-start saves (mission 10 warrior and mage, 20, 30). The original's
// own start saves of missions 10, 20 and 30 store SpellBook/IsOpen 0, so an
// engine-started mission's save at that point stores 0 as well. A save after
// the player toggles the book stores the toggled value.
func TestReleaseMissionEntrySavesTheSpellBookClosed(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, tc := range []struct {
		name         string
		start, saved int
		class        string
	}{
		{"m10 warrior", 10, 10, "Fighter"}, {"m10 mage", 10, 10, "Mage"},
		{"m20 warrior", 10, 20, "Fighter"}, {"m30 warrior", 20, 30, "Fighter"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, app := missionEntryFront(t, tc.start, tc.class)
			if tc.saved != tc.start {
				missionEntryNext(t, f, app)
			}
			if f.liveMission != tc.saved {
				t.Fatalf("live mission %d, want %d", f.liveMission, tc.saved)
			}
			inventory, book := missionEntryPanels(t, missionEntrySave(t, f, app, "mission entry"))
			t.Logf("mission %d at entry: Inventory/IsOpen %d SpellBook/IsOpen %d", f.liveMission, inventory, book)
			if book != 0 {
				t.Errorf("mission %d entry save stores SpellBook/IsOpen %d, want 0 as the original's start saves", f.liveMission, book)
			}
			for _, want := range []int32{1, 0} {
				if err := app.HeadlessKey("b"); err != nil {
					t.Fatal(err)
				}
				if got := f.live.view.SaveApplication().SpellBookOpen; got != (want == 1) {
					t.Fatalf("live book shown %v after the toggle, want %v", got, want == 1)
				}
				if _, got := missionEntryPanels(t, missionEntrySave(t, f, app, fmt.Sprintf("toggled %d", want))); got != want {
					t.Errorf("save after the toggle stores SpellBook/IsOpen %d, want %d", got, want)
				}
			}
		})
	}
}

// TestReleaseOriginalStartSaveLoadsItsStoredSpellBook loads the original's own
// mission-start saves of missions 10, 20 and 30. Each stores SpellBook/IsOpen
// 0 and opens with the book closed; the same file with the leaf set to 1 opens
// with it shown, so LOAD applies the stored value and not the mission default.
func TestReleaseOriginalStartSaveLoadsItsStoredSpellBook(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, tc := range []struct{ relative, hash string }{
		{"2026-08-12/game9999.sav", "f7bc275c0eca91751ba9a9d691ffc55ea93c465bb36c3f8a83404245e2e980d7"},
		{"2026-08-02/game9999.sav", "5822c37e8fa531e0b6d9b2348977c31f78d33f5035dd5c780e8b73459840b78e"},
		{"2026-08-24/game9999.sav", "d954bd394473d0311e934b5358519eb73ca45773732b8070e6948346f6475b27"},
	} {
		t.Run(tc.relative, func(t *testing.T) {
			_, input := groundCorpusFile(t, tc.relative, tc.hash)
			if _, book := missionEntryPanels(t, input); book != 0 {
				t.Fatalf("the original start save stores SpellBook/IsOpen %d, want 0", book)
			}
			for _, stored := range []int32{0, 1} {
				file, err := sav.Open(input)
				if err != nil {
					t.Fatal(err)
				}
				if err := file.SetStoreInt("SpellBook", "IsOpen", stored); err != nil {
					t.Fatal(err)
				}
				f := releaseFront(t)
				f.SetDeterministicFrames(true)
				open, town, err := f.RestoreOriginal(file.Marshal())
				if err != nil || town {
					t.Fatalf("LOAD: town %v err %v", town, err)
				}
				app := f.App("original start save book")
				t.Cleanup(app.StopAudio)
				if err := app.OpenMission(open); err != nil {
					t.Fatal(err)
				}
				if got := f.live.view.SaveApplication().SpellBookOpen; got != (stored == 1) {
					t.Errorf("stored SpellBook/IsOpen %d loads with the book shown %v", stored, got)
				}
			}
		})
	}
}

// TestReleaseMissionStartsWithTheGamesOwnSpellBook opens mission 10 through
// the production new-game route on the root under test. Missions of both
// games start with the book closed.
func TestReleaseMissionStartsWithTheGamesOwnSpellBook(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("mission start book")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.DirectNewGame(10)); err != nil {
		t.Fatal(err)
	}
	if got := f.live.view.SaveApplication().SpellBookOpen; got {
		t.Errorf("game %q: mission 10 starts with the book shown", f.Archives.Game())
	}
	if got := f.live.view.SaveApplication().InventoryOpen; !got {
		t.Error("the pack switch is off at the mission start")
	}
}
