package game

import (
	"strings"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/video"
)

type completionMovieRequests struct{ names []string }

func (m *completionMovieRequests) Open(name string) (*video.Player, error) {
	m.names = append(m.names, name)
	return nil, video.ErrAbsent
}

func TestMissionMoviesFollowAcceptedVictoryAndNotContinueOrDefeat(t *testing.T) {
	for _, mode := range []string{"victory", "continue-victory", "defeat"} {
		t.Run(mode, func(t *testing.T) {
			f := advanceFront(t)
			f.Town = NewTown(f.Campaign.Value())
			f.Font = resolved(missionFont(), nil)
			op := int32(sim.ScriptInstantWin)
			if mode == "defeat" {
				op = sim.ScriptInstantLose
			}
			ms := continuityMission(t, 10, op)
			ms.Map = worldFixtureMap()
			v := worldFixtureViewer(t, ms.Map)
			v.SetFont(f.Font.Value())
			v.SetMissionCutscene(10)
			mw := openMission(ms, nil, nil, v, missionSource{}, nil, nil)
			missionSteps(mw, 1)
			v.SetGameMenuContext(func() ui.GameMenuContext { return gameMenuContext(mw, true, "") })
			a := f.App("completion movies")
			movies := &completionMovieRequests{}
			a.SetCutscenes(movies)
			defer a.StopAudio()
			if err := a.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
				return v, mw.paced, mw.enqueue, mw.setCadenceMode, mw.affect, f.continuity(10, ms, mw.advanceNotice), mw.attackOrCast, mw.grab, mw.stance, mw.march, nil
			}); err != nil {
				t.Fatal(err)
			}
			if len(movies.names) != 0 {
				t.Fatal("entry requested a mission movie")
			}
			if mode == "continue-victory" {
				if err := a.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				if len(movies.names) != 0 || f.Town.Done(10) {
					t.Fatal("Continue completed the mission")
				}
				if err := a.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				for _, action := range []string{"end", "victory"} {
					if err := a.HeadlessGameMenuAction(action); err != nil {
						t.Fatal(err)
					}
				}
			} else if err := a.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			if mode == "defeat" {
				if len(movies.names) != 0 || f.Town.Done(10) {
					t.Fatal("defeat requested a victory movie")
				}
				return
			}
			if !f.Town.Done(10) || len(movies.names) != 99 || a.Screen() != ui.ScreenMap || f.live.mission.number != 20 {
				t.Fatalf("completion: done=%v requests=%v screen=%s", f.Town.Done(10), movies.names, a.Screen())
			}
			for _, name := range movies.names {
				if !strings.HasPrefix(name, "m10/") {
					t.Fatal("successor movie played early", name)
				}
			}
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
			if len(movies.names) != 99 {
				t.Fatal("completion replayed")
			}
		})
	}
}
