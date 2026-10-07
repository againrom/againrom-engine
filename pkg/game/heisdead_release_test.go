package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// AGAINROM_HEISDEAD_SAV overrides the owner's checkout location.
func heisdeadPath(t *testing.T) string {
	t.Helper()
	path := os.Getenv("AGAINROM_HEISDEAD_SAV")
	if path == "" {
		path = seatPath("engine/saves/heisdead.sav")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skip("owner campaign-end save not present: ", path)
	}
	return path
}

// The owner's mission-150 SAV has the demon dead and Victory pending (DIV-1264).
func TestReleaseHeisdeadCampaignEnd(t *testing.T) {
	raw, err := os.ReadFile(heisdeadPath(t))
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, "heisdead.sav"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("campaign end cold LOAD")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatal("cold main-menu LOAD", err)
	}
	rows := app.HeadlessRows()
	if app.Screen() != ui.ScreenLoad || len(rows) != 1 {
		t.Fatal("cold LOAD must see exactly the owner save", app.Screen(), rows)
	}
	if err := app.HeadlessActivate(rows[0].Text); err != nil || app.Screen() == ui.ScreenLoad {
		t.Fatal("cold LOAD refused the owner save", err, app.HeadlessMessage())
	}
	if app.Screen() != ui.ScreenMap || f.liveMission != 150 || f.live == nil || f.completedCampaign() {
		t.Fatal("cold LOAD did not restore the live mission 150", app.Screen(), f.liveMission)
	}
	w := f.live.world
	refs := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)
	id, ok := refs[188]
	if !ok {
		t.Fatal("installed final mission lacks script unit 188")
	}
	demon, found := w.Entity(id)
	if !found || demon.OrdinaryTargetable() || demon.HP > 0 {
		t.Fatal("saved demon is not dead", found, demon.HP, demon.MaxHP)
	}
	if w.Outcome() != sim.OutcomeWon {
		t.Fatal("saved mission outcome is not Victory", w.Outcome())
	}
	if _, kind, up := f.LiveNotice(); !up || kind != ui.NoticeSuccess {
		t.Fatal("saved Victory notice is not pending", up, kind)
	}
	if f.Town.Done(150) {
		t.Fatal("mission 150 was paid before the Victory acknowledgment")
	}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal("Victory acknowledgment", err)
	}
	if app.Screen() != ui.ScreenCredits {
		t.Fatal("terminal Victory did not show the ending credits", app.Screen())
	}
	if !f.Town.Done(150) || !f.completedCampaign() {
		t.Fatal("Victory acknowledgment did not complete the campaign")
	}
	saves := SaveStore{Dir: filepath.Join(t.TempDir(), "ending-saves")}
	f.ConfigureSaveSeams(app, saves, OriginalStore{}, nil)
	for i := 0; i < 5; i++ {
		if err := app.HeadlessKey("f2"); err != nil && app.Screen() == ui.ScreenCredits {
			t.Fatal(err)
		}
		if app.Screen() == ui.ScreenSave {
			t.Fatal("the ending offered SAVE")
		}
		if i == 0 && app.Screen() != ui.ScreenEnding {
			t.Fatal("a key did not end the credits into the hall", app.Screen())
		}
	}
	if app.Screen() != ui.ScreenEnding || len(app.HeadlessRows()) != 1 {
		t.Fatal("the hall stays until its button", app.Screen(), app.HeadlessRows())
	}
	if err := app.HeadlessActivate(app.HeadlessRows()[0].Text); err != nil || app.Screen() != ui.ScreenMenu {
		t.Fatal("hall button", err, app.Screen())
	}
	if f.completedCampaign() {
		t.Fatal("the terminal route did not reset the campaign")
	}
	if entries, err := os.ReadDir(saves.Dir); !os.IsNotExist(err) && (err != nil || len(entries) != 0) {
		t.Fatal("ending route wrote a save", entries, err)
	}
}
