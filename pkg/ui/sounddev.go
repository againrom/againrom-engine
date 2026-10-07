package ui

import (
	"fmt"
	"sync"

	ebitenaudio "github.com/hajimehoshi/ebiten/v2/audio"

	"againrom/pkg/audio"
)

// Ebitengine permits one context per process. All independently adjustable
// devices share that context, but retain their own settings and voices.
var (
	audioContextOnce sync.Once
	audioContext     *ebitenaudio.Context
	audioContextErr  error
)

type soundVoicePlayer interface {
	Play()
	IsPlaying() bool
	SetVolume(float64)
	Volume() float64
	Close() error
}

type device struct {
	newPlayer func([]byte) soundVoicePlayer
	mu        sync.Mutex
	st        audio.Settings
	voices    map[*deviceVoice]struct{}
}

// SetSettings updates current voices without restarting or seeking them.
func (d *device) SetSettings(st audio.Settings) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.st = st
	d.pruneLocked()
	for voice := range d.voices {
		voice.player.SetVolume(musicVolume(st))
	}
}

func OpenAudio(st audio.Settings) (audio.Player, error) {
	ctx, err := openAudioContext()
	if err != nil {
		return nil, err
	}
	return &device{st: st, newPlayer: func(buf []byte) soundVoicePlayer {
		return newDevicePlayer(ctx.NewPlayerFromBytes(buf))
	}}, nil
}

func openAudioContext() (*ebitenaudio.Context, error) {
	audioContextOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				audioContextErr = fmt.Errorf("audio: opening the device: %v", r)
			}
		}()
		audioContext = ebitenaudio.NewContext(audio.DeviceRate)
	})
	if audioContextErr != nil {
		return nil, audioContextErr
	}
	if audioContext == nil {
		return nil, fmt.Errorf("audio: opening the device returned no context")
	}
	return audioContext, nil
}

func (d *device) Play(sample audio.Sample, placement audio.Placement) {
	_ = d.StartVoice(sample, placement)
}

// Placement is mixed once at full gain. Device gain remains adjustable on the
// retained player. Muted voices keep their playback position, as music does.
func (d *device) StartVoice(sample audio.Sample, placement audio.Placement) audio.Voice {
	if d == nil || d.newPlayer == nil {
		return nil
	}
	buf := audio.Stereo(sample, placement, audio.DefaultSettings)
	if len(buf) == 0 {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pruneLocked()
	player := d.newPlayer(buf)
	if player == nil {
		return nil
	}
	player.SetVolume(musicVolume(d.st))
	player.Play()
	voice := &deviceVoice{device: d, player: player}
	if d.voices == nil {
		d.voices = make(map[*deviceVoice]struct{})
	}
	d.voices[voice] = struct{}{}
	return voice
}

func (d *device) pruneLocked() {
	for voice := range d.voices {
		if voice.player == nil || !voice.player.IsPlaying() {
			voice.stopLocked()
		}
	}
}

type deviceVoice struct {
	device *device
	player soundVoicePlayer
}

func (v *deviceVoice) Playing() bool {
	v.device.mu.Lock()
	defer v.device.mu.Unlock()
	return v.player != nil && v.player.IsPlaying()
}

func (v *deviceVoice) Stop() {
	v.device.mu.Lock()
	defer v.device.mu.Unlock()
	v.stopLocked()
}

func (v *deviceVoice) stopLocked() {
	if v.player != nil {
		_ = v.player.Close()
		v.player = nil
	}
	delete(v.device.voices, v)
}

// SoundDeviceState reads actual retained player gains for executable witnesses.
// It never creates a player, changes playback or estimates an audible result.
func SoundDeviceState(player audio.Player) (audio.Settings, []float64, bool) {
	if p, ok := player.(*deliveryPlayer); ok && p != nil {
		return sharedDeviceState(p.scope.shared, p.group)
	}
	d, ok := player.(*device)
	if !ok || d == nil {
		return audio.Settings{}, nil, false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	var gains []float64
	for voice := range d.voices {
		if voice.player != nil && voice.player.IsPlaying() {
			gains = append(gains, voice.player.Volume())
		}
	}
	return d.st, gains, true
}
