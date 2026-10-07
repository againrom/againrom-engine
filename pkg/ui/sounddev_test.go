package ui

import (
	"bytes"
	"sync"
	"testing"

	"againrom/pkg/audio"
)

type testSoundVoice struct {
	playing bool
	gain    float64
	plays   int
	closed  int
}

func (v *testSoundVoice) Play()                  { v.playing = true; v.plays++ }
func (v *testSoundVoice) IsPlaying() bool        { return v.playing }
func (v *testSoundVoice) SetVolume(gain float64) { v.gain = gain }
func (v *testSoundVoice) Volume() float64        { return v.gain }
func (v *testSoundVoice) Close() error           { v.playing = false; v.closed++; return nil }

func TestSoundDeviceUpdatesActiveVoicesWithoutRestart(t *testing.T) {
	sample := audio.Sample{Rate: audio.DeviceRate, PCM: []int16{8192, -4096}}
	placement := audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit / 2}
	var players []*testSoundVoice
	d := &device{st: audio.Settings{Master: 50}, newPlayer: func(pcm []byte) soundVoicePlayer {
		if !bytes.Equal(pcm, audio.Stereo(sample, placement, audio.DefaultSettings)) {
			t.Fatal("master gain was baked into PCM; later volume changes cannot recover it")
		}
		v := &testSoundVoice{}
		players = append(players, v)
		return v
	}}
	voice := d.StartVoice(sample, placement)
	d.Play(sample, placement)
	for _, setting := range []audio.Settings{{Master: 25}, {Master: 75, Muted: true}, {Master: 100}} {
		d.SetSettings(setting)
		_, gains, ok := SoundDeviceState(d)
		if !ok || len(gains) != 2 || !voice.Playing() {
			t.Fatal("volume change lost an active voice", gains)
		}
		for _, player := range players {
			if player.gain != musicVolume(setting) || player.plays != 1 || player.closed != 0 {
				t.Fatal("active voice was restarted, stopped or retained old gain", player)
			}
		}
	}
	voice.Stop()
	voice.Stop()
	if players[0].closed != 1 || len(d.voices) != 1 || voice.Playing() {
		t.Fatal("Stop must release exactly once")
	}
	players[1].playing = false
	d.SetSettings(audio.DefaultSettings)
	if len(d.voices) != 0 || players[1].closed != 1 {
		t.Fatal("completed one-shot retained its player")
	}
	d.SetSettings(audio.Settings{Muted: true})
	muted := d.StartVoice(sample, placement)
	if muted == nil || !muted.Playing() || players[2].gain != 0 {
		t.Fatal("muted voice must advance silently so re-enabling resumes its current position")
	}
	d.SetSettings(audio.DefaultSettings)
	if players[2].gain != 1 || players[2].plays != 1 {
		t.Fatal("re-enabling replayed a voice")
	}
	muted.Stop()
	if d.StartVoice(audio.Sample{}, placement) != nil || d.StartVoice(sample, audio.Placement{}) != nil {
		t.Fatal("empty or unplaced sample started")
	}
}

func TestSoundDeviceConcurrentStopAndVolume(t *testing.T) {
	d := &device{st: audio.DefaultSettings, newPlayer: func([]byte) soundVoicePlayer { return &testSoundVoice{} }}
	sample := audio.Sample{Rate: audio.DeviceRate, PCM: []int16{1, -1}}
	placement := audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 30; n++ {
				voice := d.StartVoice(sample, placement)
				d.SetSettings(audio.Settings{Master: n, Muted: n%2 == 0})
				_ = voice.Playing()
				voice.Stop()
			}
		}()
	}
	wg.Wait()
	if len(d.voices) != 0 {
		t.Fatal("stopped voices remain registered")
	}
}
