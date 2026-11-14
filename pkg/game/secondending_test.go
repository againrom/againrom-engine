package game

import (
	"errors"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func secondEndStage(t *testing.T) int32 {
	t.Helper()
	profile, ok := base.Find(base.ROM2EN)
	if !ok {
		t.Fatal("missing ROM2 profile")
	}
	return profile.Edition().CampaignEndStage
}

// secondVictory opens mission n through the acknowledged-advance seam on a
// campaign at stage with the listed destinations, and answers the campaign.
func secondVictory(t *testing.T, n int, stage int32, available []partyPlace) (*FrontEnd, *ui.App, *secondCampaign) {
	t.Helper()
	f, app, _ := secondCampaignFixture(t, true)
	f.Campaign = resolved(Campaign{EndStage: secondEndStage(t)}, nil)
	c := newSecondCampaign()
	c.bank[768] = stage
	c.current, c.available = partyPlace{1, n}, available
	f.Town.second = c
	w := secondGameScriptWorld(t, []sim.ScriptInstant{{Op: sim.ScriptInstantWin}}, []sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	w.SetROM2ScenarioState(c.bank)
	mw, view := missionDriverFor(t, w, nil, nil)
	mw.mission.table = readUnder(base.GameROM2, &mapload.Table{})
	ms := &Mission{Number: n, World: w}
	advance := continueMission(frontTransitions{f}, n, ms, mw.advanceNotice)
	f.live, f.liveMission = mw, n
	if err := app.OpenMission(openPrepared(preparedMap{viewer: view, tick: mw.deterministicFrame, cadence: mw.setCadenceMode, advance: advance}, func() {})); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 32 && !mw.mission.open; tick++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if !mw.mission.open || mw.mission.kind != ui.NoticeSuccess {
		t.Fatal("script victory did not reach production acknowledgement")
	}
	return f, app, c
}

func TestSecondCampaignLastVictoryRunsTheEnding(t *testing.T) {
	f, app, c := secondVictory(t, 110, 110, []partyPlace{{2, 2}, {2, 3}, {1, 96}, {1, 110}})
	if f.completedCampaign() {
		t.Fatal("the campaign ended before the acknowledgement")
	}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	if c.bank[768] != 120 || c.bank[896+110] != 1 || f.live != nil || !f.completedCampaign() {
		t.Fatalf("stage=%d flag=%d live=%v completed=%v", c.bank[768], c.bank[896+110], f.live != nil, f.completedCampaign())
	}
	if screen := app.Screen(); screen != ui.ScreenCredits && screen != ui.ScreenEnding {
		t.Fatalf("Victory in the last mission opened screen %v, want the ending", screen)
	}
	for i := 0; i < 8 && app.Screen() == ui.ScreenCredits; i++ {
		if err := app.HeadlessKey("ctrl-f2"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenEnding {
		t.Fatalf("the credits did not reach the hall: screen %v", app.Screen())
	}
	for i := 0; i < 4 && app.Screen() == ui.ScreenEnding; i++ {
		rows := app.HeadlessRows()
		if len(rows) == 0 {
			t.Fatal("the ending has no button")
		}
		if err := app.HeadlessActivate(rows[0].Text); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMenu || f.completedCampaign() || f.Town.second != nil {
		t.Fatalf("the hall did not reset the campaign: screen %v completed %v", app.Screen(), f.completedCampaign())
	}
}

func TestSecondCampaignEarlierVictoryKeepsTheMap(t *testing.T) {
	f, app, c := secondVictory(t, 100, 100, []partyPlace{{2, 2}, {2, 3}, {1, 100}, {1, 110}})
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	if c.bank[768] != 110 || f.completedCampaign() || app.Screen() != ui.ScreenTown {
		t.Fatalf("stage=%d completed=%v screen=%v", c.bank[768], f.completedCampaign(), app.Screen())
	}
}

// Only the decade mission that brings the stage to the end ends the campaign.
func TestSecondCampaignEveryDecadeMissionVerdict(t *testing.T) {
	end := Campaign{EndStage: secondEndStage(t)}
	c := newSecondCampaign()
	c.bank[768] = 10
	town := NewTown(Campaign{})
	town.second = c
	for n := 10; n <= 110; n += 10 {
		c.current, c.available = partyPlace{1, n}, []partyPlace{{2, 2}, {1, n}}
		before := c.bank[768]
		c.completeBank(c.bank)
		ended := n == 110
		if c.bank[768] != before+10 || end.completedBy(town) != ended || end.TerminalMission(n, c.bank[768]) != ended {
			t.Fatalf("mission %d: stage %d -> %d completed %v", n, before, c.bank[768], end.completedBy(town))
		}
	}
}

func TestSecondCampaignRefusesATownSaveAtTheEnd(t *testing.T) {
	f := &FrontEnd{}
	f.Campaign = resolved(Campaign{EndStage: secondEndStage(t)}, nil)
	var c currentSecondCampaign
	c.Bank[768] = 120
	if err := (secondCampaignRules{}).refuseCompleted(f, Snapshot{second: &c}); !errors.Is(err, refusedCompleted) {
		t.Fatalf("town save at the end stage: %v", err)
	}
	if err := (secondCampaignRules{}).refuseCompleted(f, Snapshot{Mission: 96, second: &c}); err != nil {
		t.Fatalf("mission save at the end stage: %v", err)
	}
	c.Bank[768] = 110
	if err := (secondCampaignRules{}).refuseCompleted(f, Snapshot{second: &c}); err != nil {
		t.Fatalf("town save before the end stage: %v", err)
	}
}
