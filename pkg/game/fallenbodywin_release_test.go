package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The owner's original SAV of mission 30: the hero, the companion the town
// added before the mission and two hired men stand together forty cells south
// of the goal. The mission is won when the hero stands within three cells of
// unit 56 at (65,15) (MISSION-M30-024).
const (
	bodyWinSAV  = "2027-09-07/game0032.sav"
	bodyWinHash = "7375f0c08c8361b2fa32d20564802acba688d5e5a445ab5ca635c609632499ed"
)

// bodyWinHumans reads a SAV's party through the format reader: how many
// characters the Player names as its hero, and the undecoded names of the
// other named human characters. A hired man's name is empty.
func bodyWinHumans(t *testing.T, raw []byte) (heroes int, others []string) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	party, err := file.Party()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range party {
		switch {
		case c.Hero:
			heroes++
		case c.Class == "Human" && c.Name != "":
			others = append(others, c.Name)
		}
	}
	return heroes, others
}

// bodyWinCompanions lists the party members the town's AddHero grant made.
func bodyWinCompanions(party []mapload.PartyMember) []string {
	var ids []string
	for _, p := range party {
		if p.CompanionNPC == 22 || p.ID == "npc:22" {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

// bodyWinAttempt cold-LOADs the mission-30 SAV and plays one attempt. The hero
// walks to the goal with the companion at his side, the script's message and
// win follow, and Continue closes the success panel. After wait frames the
// hero strikes her until she falls; ok reports that the blow that felled her
// left her at -10 or below inside her dying time.
func bodyWinAttempt(t *testing.T, raw []byte, wait int) (f *FrontEnd, app *ui.App, companion sim.EntityID, ok bool) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, filepath.Base(bodyWinSAV)), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app = fallenHeroWinColdLoad(t, store)
	if f.liveMission != 30 || f.live == nil {
		t.Fatalf("cold LOAD opened mission %d, want 30", f.liveMission)
	}
	var hero sim.EntityID
	found := 0
	for i, p := range f.live.mission.party {
		if p.StartingHero {
			hero, found = f.live.mission.ids[i], found+1
		}
		if p.CompanionNPC == 22 {
			companion, found = f.live.mission.ids[i], found+1
		}
	}
	if found != 2 {
		t.Fatalf("mission 30 party %v lacks the hero or the companion", fallenBodyWinIDs(f.live.mission.party))
	}
	step := func(phase string) {
		t.Helper()
		if _, kind, up := f.LiveNotice(); up && kind != ui.NoticeSuccess {
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		if f.live.mission.announced && f.live.mission.outcome != sim.OutcomeWon {
			t.Fatalf("%s: mission announced %v", phase, f.live.mission.outcome)
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	at := func(id sim.EntityID, x, y int32) bool { e, _ := f.live.entity(id); return e.X == x && e.Y == y }
	f.live.enqueue(uint32(hero), 65, 18)
	f.live.enqueue(uint32(companion), 65, 19)
	for i := 0; i < 4000; i++ {
		if _, kind, up := f.LiveNotice(); up && kind == ui.NoticeSuccess {
			break
		}
		step("walk to the goal")
	}
	if _, kind, up := f.LiveNotice(); !up || kind != ui.NoticeSuccess || f.live.world.Outcome() != sim.OutcomeWon {
		t.Fatalf("no success panel: notice up=%v kind %v, outcome %v", up, kind, f.live.world.Outcome())
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal("Continue:", err)
	}
	if _, _, up := f.LiveNotice(); up || !f.live.mission.delayedVictory {
		t.Fatalf("Continue left the notice up=%v, delayed victory %v", up, f.live.mission.delayedVictory)
	}
	for i := 0; i < 400 && !(at(hero, 65, 18) && at(companion, 65, 19)); i++ {
		step("close up")
	}
	for i := 0; i < wait; i++ {
		step("wait")
	}
	f.live.strike(uint32(hero), uint32(companion))
	for i := 0; i < 512; i++ {
		if e, _ := f.live.entity(companion); !e.Alive() {
			t.Logf("attempt after %d frames: the companion fell at health %d, dying %v", wait, e.HP, e.Dying())
			return f, app, companion, e.HP <= -10 && e.Dying()
		}
		step("strike")
	}
	e, _ := f.live.entity(companion)
	t.Fatalf("the companion still stands at health %d after 512 frames of the hero's blows", e.HP)
	return f, app, companion, false
}

// bodyWinVictory takes End Quest's Victory from the map and waits for the town
// square of chapter 40.
func bodyWinVictory(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenGameMenu {
		t.Fatalf("game menu: %v on %s", err, app.Screen())
	}
	if err := app.HeadlessGameMenuAction("end"); err != nil {
		t.Fatal("End Quest:", err)
	}
	if err := app.HeadlessGameMenuAction("victory"); err != nil {
		t.Fatal("Victory:", err)
	}
	for i := 0; i < 4000 && (app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare()); i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal("return frame:", err)
		}
	}
	if app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare() || !f.Town.Open() || f.Town.Chapter() != 40 {
		t.Fatalf("Victory returned to screen %s, town open=%v chapter %d: want the town square at chapter 40",
			app.Screen(), f.Town.Open(), f.Town.Chapter())
	}
}

// TestReleaseCompanionBodyAtTheWinStaysBehind cold-LOADs the original mission-30
// SAV. The hero walks to the goal with the companion at his side and the script
// wins; Continue keeps the party on the map, and the hero's blows fell her.
// End Quest's Victory ends the mission while her body lies inside its dying
// time. Attempts differ only in how long the party waits before the blows.
// Where she lies at -1 through -9 the mission-end cull raises her and she
// crosses with the hero. Where the blow left her at -10 or below the cull
// leaves her lying (PARTY-CULL-004), so the town, the city SAVE, its cold
// LOAD, mission 40's entry, its SAVE and that SAVE's cold LOAD hold the hero
// and no companion, and mission 40 runs without the objective that protects
// her.
func TestReleaseCompanionBodyAtTheWinStaysBehind(t *testing.T) {
	_, raw := groundCorpusFile(t, bodyWinSAV, bodyWinHash)
	heroes, others := bodyWinHumans(t, raw)
	if heroes != 1 || len(others) != 1 {
		t.Fatalf("mission 30 SAV holds %d heroes and other humans %q, want the hero and the companion", heroes, others)
	}
	companionName := others[0]
	var (
		f               *FrontEnd
		app             *ui.App
		companion       sim.EntityID
		felled, raised  bool
		attempts, lying int
	)
	for wait := 0; wait < 256 && (!felled || !raised); wait += 16 {
		attempts++
		af, aapp, acompanion, afelled := bodyWinAttempt(t, raw, wait)
		switch {
		case afelled && !felled:
			f, app, companion, felled = af, aapp, acompanion, true
		case !afelled && !raised:
			e, _ := af.live.entity(acompanion)
			lying = int(e.HP)
			bodyWinVictory(t, af, aapp)
			carried := bodyWinCompanions(af.Carried)
			if len(carried) != 1 {
				t.Fatalf("town party %v after the companion ended the mission at health %d, want her carried",
					fallenBodyWinIDs(af.Carried), e.HP)
			}
			for _, p := range af.Carried {
				if p.ID == carried[0] && p.Carry == nil {
					t.Fatalf("the companion who ended the mission at health %d crossed without her mission", e.HP)
				}
			}
			raised = true
		}
	}
	if !felled || !raised {
		t.Fatalf("in %d attempts the blows left her at -10 or below %v and at -1 through -9 %v, want both", attempts, felled, raised)
	}
	body, _ := f.live.entity(companion)
	if f.live.mission.outcome != sim.OutcomeWon || f.live.world.Outcome() != sim.OutcomeWon {
		t.Fatalf("outcome %v, world %v with the companion at health %d: want the win to stand while her body dies",
			f.live.mission.outcome, f.live.world.Outcome(), body.HP)
	}
	carriedIn := fallenBodyWinIDs(f.live.mission.party)
	finished := f.live.world
	bodyWinVictory(t, f, app)
	town := fallenBodyWinIDs(f.Carried)
	if len(f.Carried) == 0 || !f.Carried[0].StartingHero || len(bodyWinCompanions(f.Carried)) != 0 {
		t.Fatalf("town party %v, want the hero without the companion who ended the mission at health %d", town, body.HP)
	}
	for _, e := range finished.Entities() {
		if e.ID == companion && (e.Alive() || e.HP > -10) {
			t.Fatalf("the cull moved the companion's body to health %d alive=%v, want it left lying", e.HP, e.Alive())
		}
	}
	if f.originalCity != nil && f.originalCity.unavailable != nil {
		t.Fatalf("city SAVE state unavailable: %v", f.originalCity.unavailable)
	}

	cityStore := SaveStore{Dir: t.TempDir()}
	prepared, err := f.SaveDialogSeams(cityStore, OriginalStore{}).Prepare(ui.SaveRequest{Directory: cityStore.Dir, Name: "Town after the goal", Format: ui.SaveSAV})
	if err != nil {
		t.Fatal("city SAVE prepare:", err)
	}
	paths, err := prepared.Commit(false)
	if err != nil || len(paths) != 1 {
		t.Fatal("city SAVE commit:", paths, err)
	}
	written, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if heroes, others := bodyWinHumans(t, written); heroes != 1 || slices.Contains(others, companionName) {
		t.Fatalf("city SAV holds %d heroes and other humans %q, want the hero without the companion", heroes, others)
	}

	cold, coldApp := fallenHeroWinColdLoad(t, cityStore)
	if !cold.Town.Open() || cold.Town.Chapter() != 40 || len(cold.Carried) == 0 || len(bodyWinCompanions(cold.Carried)) != 0 {
		t.Fatalf("cold LOAD of the city SAV: town open=%v chapter %d, party %v: want chapter 40 without the companion",
			cold.Town.Open(), cold.Town.Chapter(), fallenBodyWinIDs(cold.Carried))
	}
	takeCampaignOffer(t, cold, 40)
	if err := coldApp.OpenMission(cold.MissionOpenerWith(40, cold.NextParty())); err != nil {
		t.Fatal("mission 40 entry:", err)
	}
	if len(bodyWinCompanions(cold.live.mission.party)) != 0 {
		t.Fatalf("mission 40 party %v, want no companion", fallenBodyWinIDs(cold.live.mission.party))
	}
	for _, c := range cold.live.world.Script().Checks() {
		if c.Op == sim.ScriptCheckVIP {
			t.Fatalf("mission 40 protects unit %d with no companion in the party", c.Unit)
		}
	}

	missionStore := SaveStore{Dir: t.TempDir()}
	prepared, err = cold.SaveDialogSeams(missionStore, OriginalStore{}).Prepare(ui.SaveRequest{OnMap: true, Directory: missionStore.Dir, Name: "Mission 40 without her", Format: ui.SaveSAV})
	if err != nil {
		t.Fatal("mission 40 SAVE prepare:", err)
	}
	paths, err = prepared.Commit(false)
	if err != nil || len(paths) != 1 {
		t.Fatal("mission 40 SAVE commit:", paths, err)
	}
	written, err = os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if heroes, others := bodyWinHumans(t, written); heroes != 1 || slices.Contains(others, companionName) {
		t.Fatalf("mission 40 SAV holds %d heroes and other humans %q, want the hero without the companion", heroes, others)
	}

	resumed, resumedApp := fallenHeroWinColdLoad(t, missionStore)
	if resumed.liveMission != 40 || resumed.live == nil || len(bodyWinCompanions(resumed.live.mission.party)) != 0 {
		t.Fatalf("cold LOAD of the mission 40 SAV opened mission %d: want 40 without the companion", resumed.liveMission)
	}
	for i := 0; i < 256; i++ {
		if _, kind, up := resumed.LiveNotice(); up && kind != ui.NoticeSuccess {
			if err := resumedApp.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		if resumed.live.mission.announced {
			t.Fatalf("resumed mission 40 announced %v after %d frames", resumed.live.mission.outcome, i)
		}
		if err := resumedApp.HeadlessStep(); err != nil {
			t.Fatal("mission 40 frame:", err)
		}
	}
	t.Logf("attempts %d; raised from health %d; left lying at health %d; mission 30 party %v, town %v, mission 40 %v",
		attempts, lying, body.HP, carriedIn, town, fallenBodyWinIDs(resumed.live.mission.party))
}
