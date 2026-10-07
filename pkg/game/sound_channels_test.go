package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/audio"
)

type channelMoviePlayer struct{ settings []audio.Settings }

func (*channelMoviePlayer) Start(int, int)                 {}
func (*channelMoviePlayer) Push([]byte)                    {}
func (*channelMoviePlayer) Stop()                          {}
func (p *channelMoviePlayer) SetSettings(s audio.Settings) { p.settings = append(p.settings, s) }

func TestSoundChannelsPersistRouteAndRejectFailedWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	if err := os.WriteFile(path, []byte("Other=kept\nMasterVolume=80\nGameSpeed=6\n"), 0600); err != nil {
		t.Fatal(err)
	}
	effects, speech, music, ambient := &menuSettingsPlayer{}, &menuSettingsPlayer{}, &menuSettingsPlayer{}, &menuSettingsPlayer{}
	movie := &channelMoviePlayer{}
	f := &FrontEnd{RuntimeServices: RuntimeServices{Sound: SoundOptions{Enabled: true, Volume: 80}, SoundChannels: audio.FullChannelVolumes(), SoundPlayer: effects, SpeechPlayer: speech, MusicPlayer: music, AmbientPlayer: ambient, CutsceneAudioPlayer: movie}, PersistenceContext: PersistenceContext{Options: OptionsStore{Path: path}}}
	for channel, value := range []int{25, 50, 75} {
		if err := f.setSoundChannel(audio.Channel(channel), value); err != nil {
			t.Fatal(err)
		}
	}
	for player, want := range map[*menuSettingsPlayer]int{music: 20, effects: 40, speech: 60, ambient: 40} {
		if got := player.settings[len(player.settings)-1]; got != (audio.Settings{Master: want}) {
			t.Fatal("wrong channel projection", got, want)
		}
	}
	if movie.settings[len(movie.settings)-1] != (audio.Settings{Master: 80}) {
		t.Fatal("channel volume changed unsplit movie audio")
	}
	if got, err := (OptionsStore{Path: path}).SoundChannelVolumes(); err != nil || got != (audio.ChannelVolumes{25, 50, 75}) {
		t.Fatal("cold channel preferences", got, err)
	}
	if err := f.setGameMenuSound(false, 50); err != nil {
		t.Fatal(err)
	}
	if f.SoundChannels != (audio.ChannelVolumes{25, 50, 75}) || !speech.settings[len(speech.settings)-1].Muted {
		t.Fatal("master mute reset channels or missed speech")
	}
	data, _ := os.ReadFile(path)
	for _, line := range []string{"Other=kept\n", "GameSpeed=6\n", "MusicVolume=25\n", "EffectsVolume=50\n", "SpeechVolume=75\n", "MasterVolume=50\n"} {
		if !strings.Contains(string(data), line) {
			t.Fatal("unrelated preference lost", line, string(data))
		}
	}
	f.Options.Path = t.TempDir()
	priorSound, priorLevels, priorCalls := f.Sound, f.SoundChannels, len(speech.settings)
	if err := f.setSoundChannel(audio.SpeechChannel, 0); err == nil || f.SoundChannels != priorLevels {
		t.Fatal("failed channel write changed live values")
	}
	if err := f.setGameMenuSound(true, 100); err == nil || f.Sound != priorSound || len(speech.settings) != priorCalls {
		t.Fatal("failed master write reached a device")
	}
}

func TestSoundChannelPreferencesUseFieldDefaultsWithoutRepair(t *testing.T) {
	path := filepath.Join(t.TempDir(), "options.txt")
	store := OptionsStore{Path: path}
	if levels, err := store.SoundChannelVolumes(); err != nil || levels != audio.FullChannelVolumes() {
		t.Fatal("old profile did not preserve full channels", levels, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("reading missing profile wrote a file")
	}
	const payload = "MusicVolume=40\r\nEffectsVolume=-1\r\nSpeechVolume=150\r\nMusicVolume=0\r\n"
	if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
		t.Fatal(err)
	}
	if levels, err := store.SoundChannelVolumes(); err != nil || levels != (audio.ChannelVolumes{0, 100, 100}) {
		t.Fatal("invalid field contaminated another channel", levels, err)
	}
	if err := store.SetSoundChannelVolumes(audio.ChannelVolumes{1, 2, 101}); err == nil {
		t.Fatal("invalid channel write accepted")
	}
	got, _ := os.ReadFile(path)
	if string(got) != payload {
		t.Fatal("read or refused write modified profile")
	}
}
