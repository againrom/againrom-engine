package game

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// secondContinuedWin wins the live mission by a script instant and steps the
// application until the victory panel stands.
func secondContinuedWin(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	mw := f.live
	trigger := sim.ScriptTrigger{Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true}
	program, err := sim.NewROM2Script(nil, []sim.ScriptInstant{{Op: sim.ScriptInstantWin, Args: [10]int32{1}}}, []sim.ScriptTrigger{trigger})
	if err != nil {
		t.Fatal(err)
	}
	won, err := sim.NewControlledScriptWorld(mw.world, program)
	if err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(won)
	*mw.world = *won
	for range 64 {
		if _, kind, up := f.LiveNotice(); up && kind == ui.NoticeSuccess {
			break
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if _, kind, up := f.LiveNotice(); mw.world.Outcome() != sim.OutcomeWon || !up || kind != ui.NoticeSuccess || !mw.mission.open {
		t.Fatalf("no victory panel: outcome %v up %v kind %v open %v", mw.world.Outcome(), up, kind, mw.mission.open)
	}
}

// secondContinuedFront is secondBagFront with no bag and a font, so the
// mission's panels can stand.
func secondContinuedFront(t *testing.T) (*FrontEnd, *ui.App, *townScreen, string) {
	t.Helper()
	f, app, screen, dir := secondBagFront(t, nil)
	f.Font = resolved(missionFont(), nil)
	return f, app, screen, dir
}

// secondContinuedUnits names every entity of w by ID, cell, health and owner.
func secondContinuedUnits(w *sim.World) []string {
	var out []string
	for _, e := range w.Entities() {
		out = append(out, fmt.Sprintf("%d %d,%d hp%d owner%d", e.ID, e.X, e.Y, e.HP, e.Owner))
	}
	return out
}

// A won mission 10 whose victory panel the player answered with Continue
// admits SAVE (owner direction, DIV-4298). LOAD returns to mission 10 with
// the same units and the won outcome; the panel stands again, and taking
// the victory leaves the mission. Fails while a won mission refuses SAVE.
func TestSecondMissionSaveAfterContinueLoadsBackToTheMission(t *testing.T) {
	f, app, screen, dir := secondContinuedFront(t)
	secondBagEnter(t, f, app, screen)
	secondContinuedWin(t, f, app)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if f.live.mission.open || !f.live.mission.delayedVictory {
		t.Fatalf("Continue left the panel open %v, continued %v, screen %v", f.live.mission.open, f.live.mission.delayedVictory, app.Screen())
	}
	units := secondContinuedUnits(f.live.world)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := SaveStore{Dir: dir}.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("SAVE after Continue: %+v %v: %s", entries, err, app.HeadlessMessage())
	}
	cold, coldApp := secondMissionLoadOf(t, secondMissionContent{side: 128, font: true}, dir, entries[0].Name)
	if cold.live == nil || cold.liveMission != 10 || coldApp.Screen() != ui.ScreenMap {
		t.Fatalf("LOAD opened mission %d on screen %v", cold.liveMission, coldApp.Screen())
	}
	if cold.live.world.Outcome() != sim.OutcomeWon {
		t.Fatalf("LOAD outcome %v, want won", cold.live.world.Outcome())
	}
	if got := secondContinuedUnits(cold.live.world); !slices.Equal(got, units) {
		t.Fatalf("LOAD units %v, want %v", got, units)
	}
	if m := cold.live.mission; !m.open || m.kind != ui.NoticeSuccess || m.delayedVictory {
		t.Fatalf("LOAD panel open %v kind %v continued %v, want the victory panel", m.open, m.kind, m.delayedVictory)
	}
	if err := coldApp.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	if c := cold.Town.second; c.current != (partyPlace{}) || coldApp.Screen() == ui.ScreenMap {
		t.Fatalf("Victory after LOAD left the party at %v on screen %v: %s", c.current, coldApp.Screen(), coldApp.HeadlessMessage())
	}
}

// Each refused second-game save point names its reason in the install's
// language through the save dialog's admission, and the named SAVE refuses
// with the same words: an open talk, the world map, a standing victory
// panel and a lost mission. A quiet town and a continued victory admit.
func TestSecondSaveRefusalNamesTheReason(t *testing.T) {
	ru := words.For("russian")
	if ru.Text("save.refused.talk") == words.For("").Text("save.refused.talk") {
		t.Fatal("the Russian table holds no refusal words")
	}
	f, app, screen, dir := secondContinuedFront(t)
	f.Words.Engine = ru
	seams := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{})
	admit := func(onMap bool) string {
		t.Helper()
		if _, err := seams.Prepare(ui.SaveRequest{OnMap: onMap, Admission: true}); err != nil {
			return err.Error()
		}
		return ""
	}
	if got := admit(false); got != "" {
		t.Fatalf("a quiet town refused: %q", got)
	}
	c := f.Town.second
	c.talk.open([]byte{1})
	if got, want := admit(false), ru.Text("save.refused.talk"); got != want {
		t.Fatalf("open talk: %q, want %q", got, want)
	}
	c.talk = talkCursor{}

	secondBagEnter(t, f, app, screen)
	secondContinuedWin(t, f, app)
	want := ru.Text("save.refused.notice")
	if got := admit(true); got != want {
		t.Fatalf("victory panel: %q, want %q", got, want)
	}
	if _, err := seams.Prepare(ui.SaveRequest{OnMap: true, Directory: dir, Name: "panel", Format: ui.SaveSAV}); err == nil || err.Error() != want {
		t.Fatalf("named SAVE at the victory panel: %v, want %q", err, want)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if err := (secondCampaignRules{}).exportable(s); !errors.Is(err, refusedNotice) || !errors.Is(err, errSavingUnavailable) {
		t.Fatalf("victory panel exportable: %v", err)
	}
	s.Residue.MissionLost = true
	if err := (secondCampaignRules{}).exportable(s); !errors.Is(err, refusedLost) {
		t.Fatalf("lost mission exportable: %v", err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if got := admit(true); got != "" {
		t.Fatalf("continued victory refused: %q", got)
	}

	g, _, gscreen, gdir := secondBagFront(t, nil)
	g.Words.Engine = ru
	destinationTownCampaign{gscreen}.leave()
	g.Town.second.current = partyPlace{}
	gseams := g.SaveDialogSeams(SaveStore{Dir: gdir}, OriginalStore{})
	if _, err := gseams.Prepare(ui.SaveRequest{Admission: true}); err == nil || err.Error() != ru.Text("save.refused.world_map") {
		t.Fatalf("world map: %v", err)
	}
}
