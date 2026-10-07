package game

import (
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseCityNewGameSessionBoundary(t *testing.T) {
	t.Run("explicit profile", func(t *testing.T) { cityNewGameSessionBoundary(t, true) })
	t.Run("fresh default", func(t *testing.T) { cityNewGameSessionBoundary(t, false) })
}

func cityNewGameSessionBoundary(t *testing.T, explicitProfile bool) {
	for _, origin := range []string{"current arrival", "original city"} {
		for _, pending := range []bool{false, true} {
			state := "settled"
			if pending {
				state = "pending return"
			}
			t.Run(origin+"/"+state, func(t *testing.T) {
				var f *FrontEnd
				if origin == "current arrival" {
					var app *ui.App
					f, app = saveDialogArrivedTown(t, t.TempDir())
					t.Cleanup(app.StopAudio)
				} else {
					_, raw := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
					f, _, _ = cityPotionLoadApp(t, raw)
				}
				f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
				if explicitProfile {
					if err := f.Options.setGameOption(ui.GameOptionAutoHealing, 1); err != nil {
						t.Fatal(err)
					}
				}
				if pending {
					screen := f.TownScreen().(*townScreen)
					screen.beginWorldMapReturn(20)
					if screen.worldMap == nil || screen.worldMap.returnMission != 20 {
						t.Fatal("fixture did not establish homeward travel")
					}
				}
				oldTown, oldLive := f.Town, f.live
				party, graph, source := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), f.originalCity.snapshot()
				gold := f.Town.Gold()
				var oldHash uint64
				if oldLive != nil {
					oldHash = oldLive.world.Hash()
				}
				result := ui.ChargenResult{Name: "New campaign", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}}
				open, err := f.prepareNewGame(10, result)
				if f.Town != oldTown || f.live != oldLive || !reflect.DeepEqual(f.Carried, party) || f.Town.Gold() != gold ||
					!reflect.DeepEqual(f.Town.cityObjects, graph) || !reflect.DeepEqual(f.originalCity.snapshot(), source) ||
					oldLive != nil && oldLive.world.Hash() != oldHash {
					t.Fatal("NEW GAME preparation changed the previous session")
				}
				if err != nil {
					t.Fatalf("NEW GAME must not save the previous city: %v", err)
				}
				if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
					t.Fatal("new campaign adoption", err)
				}
				if f.Town == oldTown || f.live == oldLive || f.liveMission != 10 || f.originalCity != nil {
					t.Fatal("new campaign did not replace the previous session")
				}
				clean := releaseFront(t)
				clean.Options = f.Options
				fresh, err := clean.prepareNewGame(10, result)
				if err != nil {
					t.Fatal(err)
				}
				if _, _, _, _, _, _, _, _, _, _, err := fresh(); err != nil {
					t.Fatal(err)
				}
				for _, world := range []*sim.World{f.live.world, clean.live.world} {
					if percent, present := world.AutoHealing(sim.SelfSlot); !present || percent != 50 {
						t.Fatal("NEW GAME did not apply Standard healing", percent, present)
					}
				}
				for tick := 0; tick <= 3; tick++ {
					cityPotionMissionEqual(t, f.live.world, clean.live.world, tick)
					if tick < 3 {
						f.LiveAdvance(1)
						clean.LiveAdvance(1)
					}
				}
			})
		}
	}
}
