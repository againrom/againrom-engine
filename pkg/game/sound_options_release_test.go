package game

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

type speechRecorder struct {
	menuSettingsPlayer
	samples []audio.Sample
}

func (p *speechRecorder) RequestSample(s audio.Sample, _ audio.Request) audio.Voice {
	p.samples = append(p.samples, s)
	return nil
}

func TestReleaseSoundOptions1187InstalledControls(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.Sound, f.SoundChannels = SoundOptions{Enabled: true, Volume: 100}, audio.FullChannelVolumes()
	speechRec := &speechRecorder{}
	effects, speech, music, ambient := &menuSettingsPlayer{}, &speechRec.menuSettingsPlayer, &menuSettingsPlayer{}, &menuSettingsPlayer{}
	f.SoundPlayer, f.SpeechPlayer, f.MusicPlayer, f.AmbientPlayer = effects, speechRec, music, ambient
	a := f.App("installed sound controls")
	defer a.StopAudio()
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	a.Layout(640, 480)
	openMissionGameMenu(t, a)
	if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
		t.Fatal(err)
	}
	rows := a.HeadlessRows()
	if len(rows) != 13 {
		t.Fatal("channel panel absent", rows)
	}
	table := LoadTextTable(f.Archives.Containers, DialogsTextPath)
	for i := 0; i < 3; i++ {
		word, ok := table.At(16 + i)
		if !ok || !strings.HasPrefix(rows[i].Text, drawnMenuLabel(word)) {
			t.Fatal("installed channel caption not used", i, rows[i].Text)
		}
	}
	for i, width := range map[int]int{4: 182, 5: 100, 6: 102, 7: 182, 8: 128, 11: 144, 12: 212} {
		if f.Font.Value().Advance(rows[i].Text) > width-6 {
			t.Fatal("localized button text clipped", i, f.Font.Value().Advance(rows[i].Text), width)
		}
	}
	hash, tick := f.live.world.Hash(), f.live.world.Tick()
	want := audio.ChannelVolumes{25, 50, 75}
	for channel, value := range want {
		for _, edge := range []string{"press", "release"} {
			at := ui.SoundSliderPoint(audio.Channel(channel), value)
			if err := a.HeadlessPointer(edge, at.X, at.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	voice, ok := f.SoundBank.NamedSample("mf_merc/select2.wav")
	if !ok || len(voice.PCM) == 0 || len(speechRec.samples) != 1 ||
		len(speechRec.samples[0].PCM) != len(voice.PCM) || speechRec.samples[0].PCM[len(voice.PCM)/2] != voice.PCM[len(voice.PCM)/2] {
		t.Fatal("speech release did not play the installed voice test once", ok, len(speechRec.samples))
	}
	if f.SoundChannels != want || f.live.world.Hash() != hash || f.live.world.Tick() != tick {
		t.Fatal("control changed wrong channel or stepped world", f.SoundChannels)
	}
	for player, volume := range map[*menuSettingsPlayer]int{music: 25, effects: 50, speech: 75, ambient: 50} {
		if got := player.settings[len(player.settings)-1]; got != (audio.Settings{Master: volume}) {
			t.Fatal("wrong production routing", got, volume)
		}
	}
	if loaded, err := f.Options.SoundChannelVolumes(); err != nil || loaded != want {
		t.Fatal("cold profile", loaded, err)
	}
	picture := a.GameMenuPanel()
	if picture == nil {
		t.Fatal("installed sound panel absent")
	}
	if dir := os.Getenv("AGAINROM_SOUND_ARTIFACTS"); dir != "" {
		dir = filepath.Join(dir, filepath.Base(os.Getenv("AGAINROM_ASSETS")))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(filepath.Join(dir, "sound-options.png"))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(file, picture)
		if closeErr := file.Close(); err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
	}
	if err := a.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("return"); err != nil || a.Screen() != ui.ScreenMap {
		t.Fatal("return to map", err)
	}
	if f.SoundChannels != want || f.live.world.Hash() != hash {
		t.Fatal("menu close altered process settings or world")
	}
	t.Log("installed labels and frame; three pointer controls; paused mission; independent routing; persisted profile; modal return")
}
