package ui

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/video"
)

type eventMovies struct {
	present map[string]bool
	played  []string
}

func (s *eventMovies) Open(name string) (*video.Player, error) {
	if !s.present[name] {
		return nil, video.ErrAbsent
	}
	s.played = append(s.played, name)
	return video.NewPlayer(io.NopCloser(bytes.NewReader(syntheticMovie()))), nil
}

func completeEventMovies(t *testing.T, a *App) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for a.Screen() == ScreenCutscene && time.Now().Before(deadline) {
		a.step(appInput{}, time.Now())
		if a.Screen() == ScreenCutscene {
			if _, _, err := a.HeadlessFrame(); err != nil {
				t.Fatal(err)
			}
		}
		time.Sleep(time.Millisecond)
	}
	if a.Screen() == ScreenCutscene || a.CutsceneError() != nil {
		t.Fatalf("movie did not finish: %v", a.CutsceneError())
	}
}

func TestCutsceneStartupThenNewGameButton(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	source := &eventMovies{present: map[string]bool{"logos/buka.smk": true, "intro/04.smk": true, "intro/05.smk": true, "newgame/01.smk": true}}
	a.SetCutscenes(source)
	if !a.PlayStartupCutscenes() {
		t.Fatal("startup did not play")
	}
	completeEventMovies(t, a)
	if a.Screen() != ScreenMenu || !reflect.DeepEqual(source.played, []string{"logos/buka.smk", "intro/04.smk", "intro/05.smk"}) {
		t.Fatalf("startup %v %v", a.Screen(), source.played)
	}
	a.step(appInput{}, time.Now())
	if err := a.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenCutscene || a.CutsceneName() != "newgame/01.smk" {
		t.Fatalf("new game did not play: %v %s", a.Screen(), a.CutsceneName())
	}
	a.step(appInput{AnyKey: true}, time.Now())
	if a.Screen() != ScreenPicker {
		t.Fatalf("skip lost the new-game destination: %s", a.Screen())
	}
}

func TestCutsceneCampaignStartAndLaterMissionAreDistinct(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	source := &eventMovies{present: map[string]bool{"start/01.smk": true, "m10/01.smk": true, "m20/01.smk": true}}
	a.SetCutscenes(source)
	v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := okOpener(t)()
	if err != nil {
		t.Fatal(err)
	}
	v.SetMissionCutscene(10)
	a.flow.setScreen(ScreenChargen)
	a.flow.enter(v, tick, order, cadence, affect, advance, attack, grab, stance, march)
	a.startPendingCutscene()
	completeEventMovies(t, a)
	if !reflect.DeepEqual(source.played, []string{"start/01.smk"}) {
		t.Fatalf("first mission: %v", source.played)
	}
	// Accepted Victory enters a fresh successor. The movie must name the
	// mission that just ended, even after that viewer has been replaced.
	next := func() (*Viewer, MapTick, MapOrder, MapCadence, MapAffect, MapAdvance, MapAttack, MapGrab, MapStance, MapMarch, error) {
		v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := okOpener(t)()
		v.SetMissionCutscene(20)
		return v, tick, order, cadence, affect, advance, attack, grab, stance, march, err
	}
	a.flow.advance = func(actions ...NoticeAction) (NoticeDest, string, MapOpener) {
		if actions[0] == NoticeContinue {
			return NoticeStay, "", nil
		}
		return NoticeToMission, "", next
	}
	a.flow.takeNoticeAction(NoticeContinue)
	a.startPendingCutscene()
	if len(source.played) != 1 {
		t.Fatal("Continue played a completion movie")
	}
	a.flow.takeNoticeAction(NoticeVictory)
	a.startPendingCutscene()
	if a.CutsceneName() != "m10/01.smk" {
		t.Fatal("completion selected successor's movie", a.CutsceneName())
	}
	completeEventMovies(t, a)
	if !reflect.DeepEqual(source.played, []string{"start/01.smk", "m10/01.smk"}) || a.Screen() != ScreenMap {
		t.Fatalf("later mission played early: %v", source.played)
	}
	a.startPendingCutscene()
	if len(source.played) != 2 {
		t.Fatal("same completion replayed")
	}
}

func TestCutsceneCompletionSurvivesTownAndEndingViewerTeardown(t *testing.T) {
	for _, dest := range []NoticeDest{NoticeToTown, NoticeToEnding, NoticeToMapList} {
		t.Run(fmt.Sprint(dest), func(t *testing.T) {
			a := newTestApp(t, appRows(1), okLoader(t))
			source := &eventMovies{present: map[string]bool{"m20/01.smk": true}}
			a.SetCutscenes(source)
			if err := a.OpenMission(okOpener(t)); err != nil {
				t.Fatal(err)
			}
			a.flow.viewer.SetMissionCutscene(20)
			a.flow.advance = func(...NoticeAction) (NoticeDest, string, MapOpener) { return dest, "", nil }
			a.flow.takeNoticeAction(NoticeVictory)
			a.startPendingCutscene()
			if a.CutsceneName() != "m20/01.smk" {
				t.Fatal("teardown lost completed mission", dest, a.CutsceneName())
			}
			completeEventMovies(t, a)
			a.startPendingCutscene()
			if len(source.played) != 1 {
				t.Fatal("completion replayed")
			}
		})
	}
}
