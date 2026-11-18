package game

import "testing"

// The mission-start autosave of the second game holds the Diary the previous
// mission ended with, and a cold LOAD of it keeps the count.
func TestReleaseSecondMissionSaveAtStartKeepsKnowledge(t *testing.T) {
	f := secondGameFront(t)
	app := f.App("autosave knowledge")
	app.Layout(1024, 768)
	out := t.TempDir()
	f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, nil)
	enterSecondCampaignMission(t, app)
	const row = 109
	secondCardSetCount(f.live.world, row, 5)
	enterSecondCampaignNextMission(t, f, app, false)
	if f.liveMission != 20 || enemyCardCount(f.live.world, row) != 5 {
		t.Fatalf("mission %d opened with count %d, want mission 20 with 5", f.liveMission, enemyCardCount(f.live.world, row))
	}
	saves, _ := secondSaveFiles(out)
	for _, s := range saves {
		if s.Label != "autosave start mission 20" {
			continue
		}
		cold, _ := secondMissionColdAt(t, out, s.Name, 20)
		if got := enemyCardCount(cold.live.world, row); got != 5 {
			t.Fatalf("cold LOAD of the mission-start autosave counts %d, want 5", got)
		}
		return
	}
	t.Fatalf("no mission-start autosave of mission 20 in %v", saves)
}
