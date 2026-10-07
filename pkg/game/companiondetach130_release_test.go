package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// requireNoCompanion22 fails when NPC22 is still in the party the town holds.
func requireNoCompanion22(t *testing.T, f *FrontEnd, stage string) {
	t.Helper()
	if len(f.Carried) == 0 {
		t.Fatalf("%s: town holds no party", stage)
	}
	for _, p := range f.Carried {
		if p.CompanionNPC == 22 {
			t.Fatalf("%s: killed traitor NPC22 is still carried as %s worn=%v", stage, p.ID, p.Worn)
		}
	}
}

// requireMintedSack fails unless w holds a registry-bound ground Sack with
// contents, such as the killed companion's death sack.
func requireMintedSack(t *testing.T, w *sim.World) {
	t.Helper()
	r := w.SavedObjects()
	if r == nil {
		t.Fatal("world has no object registry")
	}
	for _, s := range w.Sacks() {
		for _, row := range r.Sacks {
			if row.ID == s.ObjectID && s.ObjectID != 0 && (len(s.ItemInstances) != 0 || s.Gold != 0) {
				return
			}
		}
	}
	t.Fatalf("no registry-bound death sack on the ground: %d sacks", len(w.Sacks()))
}

// returnKilledCompanion130 wins mission 130 by the installed "We defeat
// Veglud" trigger, acknowledges Victory through the App, and checks the town
// party live and across a town SAVE and cold LOAD.
func returnKilledCompanion130(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	if f.live.world.Outcome() == sim.OutcomeUndecided {
		if n, err := f.live.world.HeadlessKillPlayer(8); err != nil || n == 0 {
			t.Fatal("Veglud's side", n, err)
		}
	}
	for i := 0; i < 300; i++ {
		if _, kind, up := f.LiveNotice(); up {
			if kind == ui.NoticeSuccess {
				break
			}
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if _, kind, up := f.LiveNotice(); !up || kind != ui.NoticeSuccess || f.live.world.Outcome() != sim.OutcomeWon {
		t.Fatal("mission 130 did not reach script Victory", up, kind)
	}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal("Victory acknowledgment", err)
	}
	for i := 0; i < 4000 && (app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare()); i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal("return frame", err)
		}
	}
	if app.Screen() != ui.ScreenTown {
		t.Fatal("Victory did not return to town", app.Screen())
	}
	requireNoCompanion22(t, f, "live town")
	store := SaveStore{Dir: t.TempDir()}
	prepared, err := f.SaveDialogSeams(store, OriginalStore{}).Prepare(ui.SaveRequest{
		Directory: store.Dir, Name: "Town after 130", Format: ui.SaveSAV})
	if err != nil {
		t.Fatal("prepare town SAVE", err)
	}
	if paths, err := prepared.Commit(false); err != nil || len(paths) != 1 {
		t.Fatal("publish town SAVE", paths, err)
	}
	cold, _ := coldLoadCompanion130(t, store)
	requireNoCompanion22(t, cold, "cold town SAV")
}

// coldLoadCompanion130 opens the only SAV in store through a fresh App's
// main-menu LOAD.
func coldLoadCompanion130(t *testing.T, store SaveStore) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("companion detach cold LOAD")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatal("cold main-menu LOAD", err)
	}
	rows := app.HeadlessRows()
	if app.Screen() != ui.ScreenLoad || len(rows) != 1 {
		t.Fatal("cold LOAD must see exactly the written file", app.Screen(), rows)
	}
	if err := app.HeadlessActivate(rows[0].Text); err != nil || app.Screen() == ui.ScreenLoad {
		t.Fatal("cold LOAD refused the written SAV", err, app.HeadlessMessage())
	}
	return f, app
}

func TestReleaseMission130KilledCompanionLeavesParty(t *testing.T) {
	for _, cold := range []bool{false, true} {
		name := "live"
		if cold {
			name = "cold-before-victory"
		}
		t.Run(name, func(t *testing.T) {
			x := startBetrayal130(t, false)
			x.prerequisite(t, false)
			x.transfer(t, false)
			x.hostile(t, false)
			if err := x.f.live.world.HeadlessKill(x.companion); err != nil {
				t.Fatal(err)
			}
			x.drive(t, 32)
			x.running(t, "killed traitor")
			if cold {
				// The session's first SAVE: a never-loaded world holding the
				// killed companion's death sack. It must write, not panic.
				requireMintedSack(t, x.f.live.world)
				x.cold(t, "killed traitor", nil)
				requireMintedSack(t, x.f.live.world)
			}
			returnKilledCompanion130(t, x.f, x.app)
		})
	}
	// The owner's own Victory save: a mission-130 SAV whose NPC22 is departed.
	t.Run("owner-save", func(t *testing.T) {
		path := os.Getenv("AGAINROM_SAVE_COMPANION130")
		if path == "" {
			t.Skip("AGAINROM_SAVE_COMPANION130 is not set")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		store := SaveStore{Dir: t.TempDir()}
		if err := os.WriteFile(filepath.Join(store.Dir, "owner.sav"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		f, app := coldLoadCompanion130(t, store)
		if f.live == nil || f.liveMission != 130 {
			t.Fatal("owner save is not a mission-130 SAV")
		}
		departed := false
		for i, p := range f.live.mission.party {
			departed = departed || (p.CompanionNPC == 22 && f.live.mission.departed[f.live.mission.ids[i]])
		}
		if !departed {
			t.Fatal("owner save does not hold NPC22 as departed")
		}
		returnKilledCompanion130(t, f, app)
	})
}
