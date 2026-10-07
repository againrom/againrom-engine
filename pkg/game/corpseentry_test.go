package game

import (
	"fmt"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestAStageOneCorpseVoicesOnceOnMissionEntry(t *testing.T) {
	body := sim.Entity{ID: 0, X: 4, Y: 4, Owner: sim.SelfSlot,
		TypeID: sim.HeroTypeID(false, false), Humanoid: true, Class: 14,
		HP: -15, MaxHP: 100, Decay: sim.DecayFallen, Dwell: 128, DyingTime: 128}
	mw, app, heard := hurtVoiceFight(t, body, figureID{Dir: data.FigureDirManFighter, Hero: true}, 0, 128)
	if got, ok := mw.world.Entity(body.ID); !ok || got.Decay != sim.DecayFallen {
		t.Fatal("fixture lost the independently represented fallen stage", got, ok)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if len(heard.plays) != 1 || heard.plays[0].src != "mf_hero/die.wav" {
		t.Fatalf("entry heard %v, want exactly mf_hero/die.wav for stage 1 at HP -15", heard.plays)
	}
	for frame := 0; frame < 3; frame++ {
		mw.push()
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if len(heard.plays) != 1 {
		t.Fatal("repeated snapshots replayed the entry cue", heard.plays)
	}
}

func TestSessionEntryWorldStagesOtherThanOneStaySilent(t *testing.T) {
	for _, stage := range []sim.DecayStage{0, 2, 3, 4} {
		t.Run(fmt.Sprint(stage), func(t *testing.T) {
			body := sim.Entity{ID: 0, X: 4, Y: 4, Owner: sim.SelfSlot,
				TypeID: sim.HeroTypeID(false, false), Humanoid: true, Class: 14,
				HP: -15, MaxHP: 100, Decay: stage, Dwell: 128, DyingTime: 128}
			if stage == 0 {
				body.HP = 100
			}
			_, app, heard := hurtVoiceFight(t, body, figureID{Dir: data.FigureDirManFighter, Hero: true}, 0, 128)
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
			if len(heard.plays) != 0 {
				t.Fatal("entry voiced a stage other than 1", stage, heard.plays)
			}
		})
	}
}

func TestOriginalStageOneEntryUsesSavedStageBelowHealthThreshold(t *testing.T) {
	a := &poolFixtureActor{mapID: 91, cell: 0x100f, hp: 65521, maxHP: 31, stage: 1, timer: 7, human: true}
	raw := savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a}}}}, nil))
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := source.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, record := range graph.Actors {
		if record.Stage == 1 {
			if record.HP != -15 {
				t.Fatal("literal source Stage/Health bytes changed", record.Stage, record.HP)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("literal source has no stage 1 actor")
	}
	f := currentRetainedRuntimeFixture(t, nil, a)
	v := f.live.view
	body := poolEntity(t, f.live.world, 91)
	if body.Decay != sim.DecayFallen || body.HP != -15 || body.Dwell != 7 {
		t.Fatal("cold LOAD inferred stage from health", body)
	}
	f.live.sounds = hurtVoiceClasses
	f.live.figures[body.ID] = figureID{Dir: data.FigureDirManFighter, Hero: true}
	f.live.push()
	heard := &hurtVoiceRecorder{}
	v.SetAudio(heard, hurtVoiceBank())
	v.SetSpeechAudio(heard)
	app := f.App("original stage entry")
	if err := app.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		return v, f.live.deterministicFrame, f.live.enqueue, f.live.setCadenceMode, f.live.affect, nil, f.live.attackOrCast, f.live.grab, f.live.stance, f.live.march, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if len(heard.plays) != 1 || heard.plays[0].src != "m_peasant/die.wav" {
		t.Fatal("original stage 1 entry cue", heard.plays)
	}
}
