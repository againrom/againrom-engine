package ui

import (
	"runtime"
	"testing"
	"time"

	"againrom/pkg/audio"
)

func TestDeviceOutputLevelReadsTheTestVolume(t *testing.T) {
	for _, c := range []struct {
		test    bool
		percent string
		want    float64
	}{
		{false, "", 1}, {false, "5", 1}, {true, "", 0}, {true, "x", 0},
		{true, "-1", 0}, {true, "101", 0}, {true, "5", 0.05}, {true, " 100 ", 1},
	} {
		if got := outputLevel(c.test, c.percent); got != c.want {
			t.Errorf("outputLevel(%v, %q) = %v, want %v", c.test, c.percent, got, c.want)
		}
	}
	if got := outputLevel(testing.Testing(), ""); got != 0 {
		t.Errorf("this test process plays at %v, want 0", got)
	}
}

// Every real player hands the device the engine gain times the output level,
// while Volume and AudioDeviceState keep reporting the engine gain.
func TestDevicePlayersScaleOnlyTheDeviceGain(t *testing.T) {
	if _, err := openAudioContext(); err != nil {
		t.Skipf("no audio device: %v", err)
	}
	saved := deviceOutputLevel
	t.Cleanup(func() { deviceOutputLevel = saved })
	withinDeviceBound(t, func() { devicePlayersScaleOnlyTheDeviceGain(t) })
}

func devicePlayersScaleOnlyTheDeviceGain(t *testing.T) {
	st := audio.Settings{Master: audio.MasterUnit / 2}
	engine := musicVolume(st)
	pcm := make([]int16, audio.DeviceRate)
	sample := audio.Sample{Rate: audio.DeviceRate, PCM: pcm}
	centre := audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}

	for _, level := range []float64{0, 0.05} {
		deviceOutputLevel = level
		sound, _ := OpenAudio(st)
		voice := sound.(*device).StartVoice(sample, centre)
		music, _ := OpenMusic(st)
		music.Start(audio.Track{Rate: audio.DeviceRate, StereoPCM: make([]byte, 4*len(pcm))})
		ambient, _ := OpenAmbient(st)
		ambient.StartLoop(AmbientRiver, sample, centre)
		cutscene, _ := OpenCutsceneAudio(st)
		cutscene.Start(audio.DeviceRate, 2)
		// One second of stereo fills Play's buffer, so Play returns.
		cutscene.Push(make([]byte, 4*len(pcm)))
		session := cutscene.(*cutsceneAudioDevice).s
		<-session.played

		players := map[string]*devicePlayer{
			"sound":    voice.(*deviceVoice).player.(*devicePlayer),
			"music":    music.(*musicDevice).pl,
			"ambient":  ambient.(*ambientDevice).loops[AmbientRiver].player,
			"cutscene": session.pl.(*devicePlayer),
		}
		for name, p := range players {
			if p.Volume() != engine {
				t.Errorf("level %v %s: Volume() = %v, want engine gain %v", level, name, p.Volume(), engine)
			}
			if got := p.Player.Volume(); got != engine*level {
				t.Errorf("level %v %s: device gain = %v, want %v", level, name, got, engine*level)
			}
		}
		if _, gains, ok := AudioDeviceState(music); !ok || len(gains) != 1 || gains[0] != engine {
			t.Errorf("level %v: AudioDeviceState(music) = %v, %v, want [%v]", level, gains, ok, engine)
		}
		voice.Stop()
		music.Stop()
		ambient.Stop()
		cutscene.Stop()
	}
}

// deviceBound is how long a test waits on the audio device before it fails.
const deviceBound = 30 * time.Second

// withinDeviceBound runs body and fails with every goroutine's stack if body
// has not returned after deviceBound. body may call t.Error but not t.Fatal.
func withinDeviceBound(t *testing.T, body func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		body()
	}()
	select {
	case <-done:
	case <-time.After(deviceBound):
		stacks := make([]byte, 1<<20)
		t.Fatalf("audio device did not return within %v; goroutines:\n%s", deviceBound, stacks[:runtime.Stack(stacks, true)])
	}
}
