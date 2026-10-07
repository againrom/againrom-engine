package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The original's mission-10 SAV labelled "112": the two guards are dead and
// the witch, unit 21, walks with the player's party a few cells short of
// (56,21). MISSION-M10-009's win chain remains: she reaches (56,21), which
// sets the escort counter, then the hero reaches (66,16). The original
// mission-20 SAV is the campaign record the original holds after mission 10's
// automatic route.
const (
	autoGetTenSAV     = "2026-08-02/game0004.sav"
	autoGetTenHash    = "7f33c3f05bf574e73f4020f8241ba5e7f848c05ac5fca60ba79f1cb65325b604"
	autoGetTwentySAV  = "2026-08-02/game0008.sav"
	autoGetTwentyHash = "ebe713078281f3a4fef57fbd5e71be7899b39d98846fa6260f7ba845fab0d466"
	autoGetOwnerHash  = "987482f272dd2d765b99daf05dfbf4a34c2983824174784389e3d65356d5b467"
)

// autoGetSession cold-LOADs the one file in dir through the main menu.
func autoGetSession(t *testing.T, dir string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("AutoGetMission SAV")
	t.Cleanup(app.StopAudio)
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatal("main-menu LOAD:", err)
	}
	rows := app.HeadlessRows()
	if app.Screen() != ui.ScreenLoad || len(rows) != 1 {
		t.Fatalf("LOAD lists %v on %s, want one file", rows, app.Screen())
	}
	if err := app.HeadlessActivate(rows[0].Text); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatalf("LOAD refused the file: %v %s", err, app.HeadlessMessage())
	}
	return f, app
}

// autoGetF2Save closes an open notice, which holds F2, the way a player does:
// a message with its button, the Victory panel with Continue. It then writes
// the current mission through F2 under name into a new directory and returns
// that directory and the file's bytes.
func autoGetF2Save(t *testing.T, f *FrontEnd, app *ui.App, name string) (string, []byte) {
	t.Helper()
	for i := 0; i < 16; i++ {
		text, kind, up := f.LiveNotice()
		if !up {
			break
		}
		t.Logf("%s: closing notice kind %d %q", name, kind, text)
		var err error
		if kind == ui.NoticeSuccess {
			err = app.HeadlessKey("escape")
		} else {
			err = app.HeadlessActivate("notice")
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	before := f.live.world.Hash()
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("F2 SAVE", err, app.Screen())
	}
	dir := t.TempDir()
	if err := app.HeadlessSaveEdit(dir, name, ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if f.live.world.Hash() != before {
		t.Fatal("F2 SAVE changed the current World")
	}
	for i := 0; i < 4 && app.Screen() != ui.ScreenMap; i++ {
		t.Logf("%s: leaving %s", name, app.Screen())
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatalf("%s: SAVE left the screen at %s", name, app.Screen())
	}
	raw, err := os.ReadFile(filepath.Join(dir, name+".sav"))
	if err != nil {
		t.Fatal(err)
	}
	return dir, raw
}

// autoGetCampaign decodes a SAV's campaign record through the format reader.
func autoGetCampaign(t *testing.T, raw []byte) sav.CampaignProjection {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	c, present, err := file.Campaign()
	if err != nil || !present {
		t.Fatal("campaign record", present, err)
	}
	return c
}

// autoGetStep closes a message the mission raises, fails on a lost mission and
// runs one frame.
func autoGetStep(t *testing.T, f *FrontEnd, app *ui.App, phase string) {
	t.Helper()
	if _, kind, up := f.LiveNotice(); up && kind != ui.NoticeSuccess {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.mission.announced && f.live.mission.outcome != sim.OutcomeWon {
		t.Fatalf("%s: mission %d announced %v", phase, f.liveMission, f.live.mission.outcome)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
}

// autoGetVictory waits for the Victory panel and acknowledges it.
func autoGetVictory(t *testing.T, f *FrontEnd, app *ui.App, frames int, phase string) {
	t.Helper()
	for i := 0; i < frames; i++ {
		if _, kind, up := f.LiveNotice(); up && kind == ui.NoticeSuccess {
			break
		}
		autoGetStep(t, f, app, phase)
	}
	if _, kind, up := f.LiveNotice(); !up || kind != ui.NoticeSuccess || f.live.world.Outcome() != sim.OutcomeWon {
		t.Fatalf("%s: no Victory: notice up=%v kind %v, outcome %v", phase, up, kind, f.live.world.Outcome())
	}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal("Victory acknowledgment:", err)
	}
}

// TestReleaseMissionTenWinSAVRoutesMissionTwentyHome cold-LOADs the original
// mission-10 SAV. Its F2 SAVE holds mission 10's own AutoGetMission 20. The
// hero walks to the escort's end, the script wins, and Victory opens mission
// 20. There F2 SAVE writes mission 20's own -1 and its main record announced,
// as the original's mission-20 SAV holds them (REG-SCN-063); a cold LOAD of
// that file and a second SAVE write the same values.
func TestReleaseMissionTenWinSAVRoutesMissionTwentyHome(t *testing.T) {
	_, raw := groundCorpusFile(t, autoGetTenSAV, autoGetTenHash)
	_, reference := groundCorpusFile(t, autoGetTwentySAV, autoGetTwentyHash)
	original := autoGetCampaign(t, reference)
	if original.Main.Mission != 20 || original.AutoGetMission != 0xFFFFFFFF || !original.Main.Announced {
		t.Fatalf("original mission-20 SAV holds main %+v, AutoGetMission %#x", original.Main, original.AutoGetMission)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, filepath.Base(autoGetTenSAV)), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := autoGetSession(t, dir)
	if f.liveMission != 10 || f.live == nil {
		t.Fatalf("cold LOAD opened mission %d, want 10", f.liveMission)
	}
	_, ten := autoGetF2Save(t, f, app, "mission ten")
	if c := autoGetCampaign(t, ten); c.AutoGetMission != 20 || c.Main.Mission != 10 || c.Main.Announced {
		t.Fatalf("mission-10 SAV holds AutoGetMission %#x, main %+v; want 0x14 at mission 10 unannounced", c.AutoGetMission, c.Main)
	}

	var hero sim.EntityID
	for i, p := range f.live.mission.party {
		if p.StartingHero {
			hero = f.live.mission.ids[i]
		}
	}
	witch := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)[21]
	h, _ := f.live.entity(hero)
	w, _ := f.live.entity(witch)
	if hero == 0 || !w.Alive() || w.Owner != h.Owner || f.live.world.ScriptRegister(50) != 0 || f.live.world.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("mission-10 SAV: hero %d, witch alive=%v owner %d (hero's %d), escort counter %d, outcome %v; want the witch in the party short of the escort's end",
			hero, w.Alive(), w.Owner, h.Owner, f.live.world.ScriptRegister(50), f.live.world.Outcome())
	}
	f.live.enqueue(uint32(witch), 56, 21)
	for i := 0; i < 2000 && f.live.world.ScriptRegister(50) == 0; i++ {
		autoGetStep(t, f, app, "the witch walks to (56,21)")
	}
	if f.live.world.ScriptRegister(50) == 0 {
		t.Fatal("the witch did not reach the escort's end")
	}
	f.live.enqueue(uint32(hero), 66, 16)
	autoGetVictory(t, f, app, 4000, "the hero walks to (66,16)")
	if f.liveMission != 20 || f.live == nil {
		t.Fatalf("mission-10 Victory opened mission %d, want 20", f.liveMission)
	}

	saved, twenty := autoGetF2Save(t, f, app, "mission twenty")
	first := autoGetCampaign(t, twenty)
	if first.AutoGetMission != 0xFFFFFFFF || first.Main.Mission != 20 || !first.Main.Announced {
		t.Fatalf("mission-20 SAV holds AutoGetMission %#x, main %+v; want 0xffffffff at mission 20 announced", first.AutoGetMission, first.Main)
	}
	if !reflect.DeepEqual(first.Main, original.Main) {
		t.Fatalf("mission-20 SAV main record %+v, original mission-20 SAV %+v", first.Main, original.Main)
	}
	g, gapp := autoGetSession(t, saved)
	if g.liveMission != 20 || g.live == nil {
		t.Fatalf("cold LOAD of the mission-20 SAV opened mission %d, want 20", g.liveMission)
	}
	_, again := autoGetF2Save(t, g, gapp, "mission twenty again")
	second := autoGetCampaign(t, again)
	if second.AutoGetMission != first.AutoGetMission || !reflect.DeepEqual(second.Main, first.Main) {
		t.Fatalf("second mission-20 SAV holds AutoGetMission %#x, main %+v; first %#x, %+v",
			second.AutoGetMission, second.Main, first.AutoGetMission, first.Main)
	}
	t.Logf("hero from (%d,%d), witch from (%d,%d); mission-20 main record %+v, AutoGetMission %#x after SAVE and after cold LOAD",
		h.X, h.Y, w.X, w.Y, first.Main, first.AutoGetMission)
}

// TestReleaseOwnerMissionTwentyEndSAVRoutesHome loads the owner's engine SAV
// of won mission 20, which an earlier build wrote with mission 10's
// AutoGetMission 20: the original then sends the party back to mission 20's
// map object. F2 SAVE writes mission 20's own -1, and a cold LOAD of the new
// file still reaches the town on the engine's Victory.
func TestReleaseOwnerMissionTwentyEndSAVRoutesHome(t *testing.T) {
	path := os.Getenv("AGAINROM_MISSION20_END_SAV")
	if path == "" {
		t.Skip("no AGAINROM_MISSION20_END_SAV: the owner's mission20end save is an owner input")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != autoGetOwnerHash {
		t.Fatalf("owner save changed: %s", got)
	}
	if c := autoGetCampaign(t, raw); c.AutoGetMission != 20 || c.Main.Mission != 20 {
		t.Fatalf("owner save holds AutoGetMission %#x at main %d, want the stale 0x14 at 20", c.AutoGetMission, c.Main.Mission)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0007.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := autoGetSession(t, dir)
	if f.liveMission != 20 || f.live == nil {
		t.Fatalf("LOAD opened mission %d, want 20", f.liveMission)
	}
	saved, out := autoGetF2Save(t, f, app, "9402 autoget")
	if kit := os.Getenv("AGAINROM_AUTOGET_KIT"); kit != "" {
		if err := os.WriteFile(filepath.Join(kit, "game9402.sav"), out, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if label, err := OriginalSaveLabel(out, f.textSelector()); err != nil || label != "9402 autoget - mission 20" {
		t.Fatalf("SAV label %q %v", label, err)
	}
	c := autoGetCampaign(t, out)
	if c.AutoGetMission != 0xFFFFFFFF || c.Main.Mission != 20 || c.SelectedMission != 20 {
		t.Fatalf("resaved SAV holds AutoGetMission %#x, main %+v, selected %d; want 0xffffffff at mission 20",
			c.AutoGetMission, c.Main, c.SelectedMission)
	}

	g, gapp := autoGetSession(t, saved)
	if g.liveMission != 20 || g.live == nil {
		t.Fatalf("cold LOAD of the resaved SAV opened mission %d, want 20", g.liveMission)
	}
	autoGetVictory(t, g, gapp, 256, "owner mission 20")
	for i := 0; i < 4000 && (gapp.Screen() != ui.ScreenTown || !g.townUI.AtTownSquare()); i++ {
		if err := gapp.HeadlessStep(); err != nil {
			t.Fatal("return frame:", err)
		}
	}
	if gapp.Screen() != ui.ScreenTown || !g.townUI.AtTownSquare() || !g.Town.Open() || g.Town.Chapter() != 30 {
		t.Fatalf("Victory returned to screen %s, town open=%v chapter %d: want the town square at chapter 30",
			gapp.Screen(), g.Town.Open(), g.Town.Chapter())
	}
	t.Logf("resaved main record %+v, AutoGetMission %#x, selected %d", c.Main, c.AutoGetMission, c.SelectedMission)
}
