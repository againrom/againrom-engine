package game

import (
	"io"
	"path/filepath"
	"testing"

	"againrom/pkg/audio"
)

type playbackRecorder1189 struct {
	menuSettingsPlayer
	starts  int
	invalid bool
}

func (p *playbackRecorder1189) Start(track audio.Track) {
	p.starts++
	p.invalid = p.invalid || track.Rate != audio.DeviceRate || len(track.StereoPCM) < 4
}

func TestReleaseSoundPlayback1189InstalledTracksAndColdStop(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	device := &playbackRecorder1189{}
	f.MusicPlayer = device
	if err := f.witnessMusicPlayback("", io.Discard); err != nil {
		t.Fatal(err)
	}
	if device.starts < 3 || device.invalid {
		t.Fatal("installed tracks did not reach the device as decoded PCM", device)
	}
	stored, err := f.Options.MusicPreferences()
	if err != nil || stored.Enabled || stored.RandomOrder {
		t.Fatal("Stop and sequential order did not persist", stored, err)
	}
	cold := releaseFront(t)
	cold.Options = f.Options
	freshDevice := &playbackRecorder1189{}
	cold.MusicPlayer = freshDevice
	a := cold.App("cold stopped music")
	defer a.StopAudio()
	if err := a.OpenMission(cold.MissionOpener(111)); err != nil {
		t.Fatal(err)
	}
	if _, playing := a.MusicPlaybackState(); playing != "" || freshDevice.starts != 0 {
		t.Fatal("stopped profile started music before or after mission load", playing, freshDevice.starts)
	}
}
