package game

import (
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestGameMenuContextReadsLiveOrderedRelations(t *testing.T) {
	rel := sim.Relations{}
	rel.Set(sim.SelfSlot, 7, 1)
	rel.Set(sim.SelfSlot, 3, 2)
	w, err := sim.NewRelatedWorld(1, scenarioBounds, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{
		scenarioFighter(1, sim.SelfSlot, 1, 1),
		scenarioFighter(2, 7, 2, 1),
		scenarioFighter(3, 3, 3, 1),
		scenarioFighter(4, 5, 4, 1),
	}, nil, rel)
	if err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	ctx := gameMenuContext(&mapWorld{world: w}, false, "Reach the gate")
	if after := w.Hash(); after != before {
		t.Fatalf("reading the menu context moved hashed state: %#x -> %#x", before, after)
	}
	want := []struct {
		slot  uint32
		state string
	}{{3, "ALLIED"}, {5, "NEUTRAL"}, {7, "HOSTILE"}}
	if ctx.Campaign || ctx.Objective != "Reach the gate" || len(ctx.Relations) != len(want) {
		t.Fatalf("context = %+v", ctx)
	}
	for i := range want {
		if ctx.Relations[i].Slot != want[i].slot || ctx.Relations[i].State != want[i].state {
			t.Errorf("relation %d = %+v, want %+v", i, ctx.Relations[i], want[i])
		}
	}
}

type menuSettingsPlayer struct {
	settings []audio.Settings
}

func (*menuSettingsPlayer) Play(audio.Sample, audio.Placement) {}
func (*menuSettingsPlayer) RequestSample(audio.Sample, audio.Request) audio.Voice {
	return nil
}
func (*menuSettingsPlayer) RequestLoop(ui.AmbientLoop, audio.Sample, audio.Request) audio.Voice {
	return nil
}
func (*menuSettingsPlayer) Start(audio.Track)                                       {}
func (*menuSettingsPlayer) Stop()                                                   {}
func (*menuSettingsPlayer) Ended() bool                                             { return false }
func (*menuSettingsPlayer) StartLoop(ui.AmbientLoop, audio.Sample, audio.Placement) {}
func (*menuSettingsPlayer) MoveLoop(ui.AmbientLoop, audio.Placement)                {}
func (*menuSettingsPlayer) StopLoop(ui.AmbientLoop)                                 {}
func (p *menuSettingsPlayer) SetSettings(s audio.Settings) {
	p.settings = append(p.settings, s)
}

func TestSoundOptionsChangeSessionStateAndAllOpenedDevices(t *testing.T) {
	sound, music, ambient := &menuSettingsPlayer{}, &menuSettingsPlayer{}, &menuSettingsPlayer{}
	f := &FrontEnd{RuntimeServices: RuntimeServices{Sound: SoundOptions{Enabled: true, Volume: 75}, SoundChannels: audio.FullChannelVolumes(), SoundPlayer: sound, MusicPlayer: music, AmbientPlayer: ambient}}
	enabled, volume, available := f.gameMenuSound()
	if !enabled || volume != 75 || !available {
		t.Fatalf("gameMenuSound = (%v,%d,%v)", enabled, volume, available)
	}
	f.setGameMenuSound(false, 125)
	if f.Sound != (SoundOptions{Enabled: false, Volume: audio.MasterUnit}) {
		t.Fatalf("front-end sound = %+v", f.Sound)
	}
	want := audio.Settings{Master: audio.MasterUnit, Muted: true}
	if len(sound.settings) != 1 || sound.settings[0] != want {
		t.Fatalf("sound device settings = %v, want [%+v]", sound.settings, want)
	}
	if len(music.settings) != 1 || music.settings[0] != want {
		t.Fatalf("music device settings = %v, want [%+v]", music.settings, want)
	}
	if len(ambient.settings) != 1 || ambient.settings[0] != want {
		t.Fatalf("ambient device settings = %v, want [%+v]", ambient.settings, want)
	}
}
