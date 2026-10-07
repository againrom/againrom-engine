package main

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/game"
)

func TestStartupSoundPreferencesAndExplicitFlags(t *testing.T) {
	root := defaultInstall(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "options.txt")
	const preferences = "SoundEnabled=0\nMasterVolume=25\nMusicVolume=40\nEffectsVolume=0\nSpeechVolume=80\nGameSpeed=6\nOther=retained\n"
	if err := os.WriteFile(path, []byte(preferences), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		want game.SoundOptions
	}{
		{"stored", nil, game.SoundOptions{Enabled: false, Volume: 25}},
		{"explicit enable", []string{"-sound=true"}, game.SoundOptions{Enabled: true, Volume: 25}},
		{"explicit full volume", []string{"-volume=100"}, game.SoundOptions{Enabled: false, Volume: 100}},
		{"explicit both", []string{"-sound=true", "-volume=75"}, game.SoundOptions{Enabled: true, Volume: 75}},
		{"explicit silence", []string{"-sound=false", "-volume=0"}, game.SoundOptions{Enabled: false, Volume: 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"-saves", filepath.Join(dir, "saves")}, tc.args...)
			o, err := parse(args)
			if err != nil {
				t.Fatal(err)
			}
			profile, _, err := loadSources(root, o, "", "")
			if err != nil {
				t.Fatal(err)
			}
			front, err := frontEnd(root, o, profile.Options)
			if err != nil {
				t.Fatal(err)
			}
			if front.Sound != tc.want {
				t.Fatalf("startup sound=%+v want %+v", front.Sound, tc.want)
			}
			if front.SoundChannels != (audio.ChannelVolumes{40, 0, 80}) {
				t.Fatal("master flag changed independent channel preferences", front.SoundChannels)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != preferences {
				t.Fatalf("startup rewrote preferences: %q: %v", got, err)
			}
		})
	}
}

func TestSoundPreferenceHeadlessMuteAndVolumeBounds(t *testing.T) {
	dir := t.TempDir()
	store := game.OptionsStore{Path: filepath.Join(dir, "options.txt")}
	if err := store.SetSoundOptions(game.SoundOptions{Enabled: true, Volume: 25}); err != nil {
		t.Fatal(err)
	}
	o, err := parse([]string{"-headless", "scenario.json", "-sound=true"})
	if err != nil {
		t.Fatal(err)
	}
	if got := startupSoundOptions(store, o); got.Enabled || got.Volume != 25 {
		t.Fatal("scenario must stay silent even with stored or explicit sound", got)
	}
	for _, value := range []string{"-1", "101", "150"} {
		if _, err := parse([]string{"-volume", value}); err == nil {
			t.Fatal("out-of-range volume accepted", value)
		}
	}
	missing := game.OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	o, _ = parse(nil)
	if got := startupSoundOptions(missing, o); !got.Enabled || got.Volume != 100 {
		t.Fatal("missing preferences changed original defaults", got)
	}
	if _, err := os.Stat(missing.Path); !os.IsNotExist(err) {
		t.Fatal("startup created a preference file", err)
	}
}
