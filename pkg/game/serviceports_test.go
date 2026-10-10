package game

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/mapload"
	"againrom/pkg/random"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type fakeTransitions struct {
	calls     []string
	successor int
	decide    winRoute
	leaveDest ui.NoticeDest
}

func (f *fakeTransitions) leave(n int, _ *Mission, action ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) {
	f.calls = append(f.calls, "leave")
	return f.leaveDest, "left", nil
}

func (f *fakeTransitions) finishWon(n int, _ *Mission) (int, string) {
	f.calls = append(f.calls, "finish")
	return f.successor, "finished"
}

func (f *fakeTransitions) route(n, successor int) winRoute {
	f.calls = append(f.calls, "route")
	return f.decide
}

func (f *fakeTransitions) opener(n int) ui.MapOpener {
	f.calls = append(f.calls, "opener")
	return func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, errors.New("fake opener")
	}
}

func (f *fakeTransitions) returnToTown(n int) { f.calls = append(f.calls, "town") }

func TestContinueMissionRunsOverAFakeTransitionRule(t *testing.T) {
	won := continuityMission(t, 10, sim.ScriptInstantWin)
	lost := continuityMission(t, 10, sim.ScriptInstantLose)
	cases := []struct {
		name  string
		ms    *Mission
		fake  *fakeTransitions
		acts  []ui.NoticeAction
		want  ui.NoticeDest
		calls []string
	}{
		{"win into a mission", won, &fakeTransitions{successor: 20, decide: winMission}, nil, ui.NoticeToMission, []string{"finish", "route", "opener"}},
		{"win into the town", won, &fakeTransitions{decide: winTown}, nil, ui.NoticeToTown, []string{"finish", "route", "town"}},
		{"win into the ending", won, &fakeTransitions{decide: winEnding}, nil, ui.NoticeToEnding, []string{"finish", "route"}},
		{"unfinished win stays", won, &fakeTransitions{successor: -1, decide: winStay}, nil, ui.NoticeStay, []string{"finish", "route"}},
		{"loss never asks the rule", lost, &fakeTransitions{decide: winMission}, nil, ui.NoticeToMapList, nil},
		{"abandon is the rule's", won, &fakeTransitions{leaveDest: ui.NoticeToTown}, []ui.NoticeAction{ui.NoticeAbandon}, ui.NoticeToTown, []string{"leave"}},
		{"restart is the rule's", won, &fakeTransitions{leaveDest: ui.NoticeToMission}, []ui.NoticeAction{ui.NoticeRestart}, ui.NoticeToMission, []string{"leave"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dest, _, _ := continueMission(tc.fake, 10, tc.ms, listAdvance)(tc.acts...)
			if dest != tc.want {
				t.Errorf("destination %v, want %v", dest, tc.want)
			}
			if !reflect.DeepEqual(tc.fake.calls, tc.calls) {
				t.Errorf("calls %v, want %v", tc.fake.calls, tc.calls)
			}
		})
	}
	t.Run("a mission with no world passes the driver's answer", func(t *testing.T) {
		fake := &fakeTransitions{decide: winMission}
		dest, msg, _ := continueMission(fake, 10, &Mission{}, listAdvance)()
		if dest != ui.NoticeToMapList || msg != TownNotBuiltMessage || len(fake.calls) != 0 {
			t.Errorf("dest %v msg %q calls %v", dest, msg, fake.calls)
		}
	})
}

type fakeSnapshotSource struct {
	snapshot Snapshot
	err      error
	onMap    []bool
}

func (s *fakeSnapshotSource) Snapshot(onMap bool) (Snapshot, string, error) {
	s.onMap = append(s.onMap, onMap)
	return s.snapshot, "", s.err
}

func TestOpenCityBaseRunsOverAFakeSnapshotSource(t *testing.T) {
	boom := errors.New("no game")
	src := &fakeSnapshotSource{err: boom}
	if _, _, err := openCityBase(src, nil, nil); !errors.Is(err, boom) {
		t.Fatalf("error %v, want the source's", err)
	}
	if !reflect.DeepEqual(src.onMap, []bool{false}) {
		t.Fatalf("the base asks the town snapshot, got %v", src.onMap)
	}
	src = &fakeSnapshotSource{snapshot: Snapshot{Gold: -1}}
	if _, _, err := cityBaseFrom(src, nil)([]mapload.PartyMember{{ID: "a"}}); err == nil {
		t.Fatal("a negative purse is refused through the port")
	}
}

type fakeTownAudio struct {
	asked   []string
	sound   fakePlayer
	speech  fakePlayer
	ambient *recordingAmbient
}

type recordingAmbient struct {
	fakePlayer
	stopped []ui.AmbientLoop
}

func (a *recordingAmbient) StopLoop(l ui.AmbientLoop) { a.stopped = append(a.stopped, l) }

func (a *fakeTownAudio) soundDevice() audio.Player {
	a.asked = append(a.asked, "sound")
	return a.sound
}

func (a *fakeTownAudio) speechDevice() audio.Player {
	a.asked = append(a.asked, "speech")
	return a.speech
}

func (a *fakeTownAudio) ambientDevice() ui.AmbientDevice {
	a.asked = append(a.asked, "ambient")
	return a.ambient
}

func (a *fakeTownAudio) wildlifeSeed() int64 { a.asked = append(a.asked, "seed"); return 99 }

func TestTownScreenPlaysThroughASubstitutedAudioService(t *testing.T) {
	fake := &fakeTownAudio{ambient: &recordingAmbient{}}
	s := bareTownScreen(t)
	s.sound = fake
	s.squareLoop = &tavernInteriorVoice{playing: true}

	townSquareHost{s}.StopLoop(ROM1TownDescription().Sounds.Loop)
	if !reflect.DeepEqual(fake.ambient.stopped, []ui.AmbientLoop{ui.AmbientTownCrowd}) {
		t.Fatalf("the crowd loop was stopped on %v", fake.ambient.stopped)
	}
	if s.squareLoop != nil {
		t.Fatal("the crowd stays active")
	}
	if got := s.roomSoundPlayer(audio.EffectsChannel); got != audio.Player(fake.sound) {
		t.Fatalf("effects device %T, want the service's sound device", got)
	}
	if got := s.roomSoundPlayer(audio.SpeechChannel); got != audio.Player(fake.speech) {
		t.Fatalf("speech device %T, want the service's speech device", got)
	}
	if len(fake.asked) == 0 {
		t.Fatal("the screen asked the service for nothing")
	}

	rule := &fakeTransitions{successor: 20, decide: winMission}
	dest, _, _ := continueMission(rule, 10, continuityMission(t, 10, sim.ScriptInstantWin), listAdvance)()
	if dest != ui.NoticeToMission || !reflect.DeepEqual(rule.calls, []string{"finish", "route", "opener"}) {
		t.Fatalf("transition under a substituted audio service: %v %v", dest, rule.calls)
	}
}

type fakeTownDraws struct{ fixed func(int) int }

func (d fakeTownDraws) animationClock() func() time.Time {
	return func() time.Time { return time.Unix(7, 0) }
}
func (d fakeTownDraws) animationDraw() func(int) int { return d.fixed }
func (d fakeTownDraws) ambientDraw() func(int) int   { return nil }
func (d fakeTownDraws) tavernDraw() func(int) int    { return nil }
func (d fakeTownDraws) shopDraw() func(int) int      { return nil }
func (d fakeTownDraws) schoolDraw() func(int) int    { return nil }
func (d fakeTownDraws) stream(random.Name) *random.Stream {
	return nil
}

func TestTownScreenReadsASubstitutedDrawService(t *testing.T) {
	s := bareTownScreen(t)
	s.draws = fakeTownDraws{fixed: func(n int) int { return n - 1 }}
	if got := (townSquareHost{s}).Draw("animation", 5); got != 4 {
		t.Fatalf("exterior roll %d, want the service's draw", got)
	}
	if got := s.townAnimationNow(); !got.Equal(time.Unix(7, 0)) {
		t.Fatalf("animation clock %v, want the service's", got)
	}
}

type fakeTownArt struct {
	townArt
	frame *ui.MenuPanelArt
}

func (a fakeTownArt) menuFrame() *ui.MenuPanelArt { return a.frame }

func TestTownScreenDrawsWithASubstitutedArtService(t *testing.T) {
	s := bareTownScreen(t)
	frame := &ui.MenuPanelArt{}
	s.art = fakeTownArt{frame: frame}
	s.room = roomTalk
	layout, _, ok := s.townDialogueLayout()
	if !ok || layout.Frame != frame {
		t.Fatalf("dialogue frame %p, want the service's %p (ok %v)", layout.Frame, frame, ok)
	}
}
