package game

import (
	"testing"

	"againrom/pkg/ui"
)

func TestReleaseGeneratedWorld1171Mission20SecondNewGameOpensAfterEmptyUnlockWin(t *testing.T) {
	f := releaseFront(t)
	app := f.App("mission20 reopen after an empty-unlock win")
	res := ui.ChargenResult{Name: "First mission 20", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}}
	if err := app.OpenMission(f.NewGameOpener(20, res)); err != nil {
		t.Fatalf("first NEW GAME of mission 20: %v", err)
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatalf("winning mission 20: %v", err)
	}
	if !f.Town.Open() {
		t.Fatal("winning mission 20 must open the campaign's own town (Campaign.NextMission names a town offer)")
	}
	if got := impliedMercenaryUnlocks(f.Campaign.Value(), 20); len(got) != 0 {
		t.Fatalf("this fixture assumes chapter 20 implies no mercenary unlock of its own, got %v", got)
	}
	for typ, enabled := range f.Town.mercEnabled {
		if enabled {
			t.Fatalf("this fixture assumes an empty permanent mercenary list after mission 20 alone; type %d is already enabled", typ)
		}
	}
	res2 := ui.ChargenResult{Name: "Second mission 20", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}}
	if err := app.OpenMission(f.NewGameOpener(20, res2)); err != nil {
		t.Fatalf("second NEW GAME of mission 20 must open, not refuse: %v", err)
	}
	if f.liveMission != 20 || f.live == nil || f.live.world == nil {
		t.Fatalf("second NEW GAME of mission 20 did not actually open the map: liveMission=%d live=%v", f.liveMission, f.live)
	}
}
