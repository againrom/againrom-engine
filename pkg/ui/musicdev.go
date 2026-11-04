package ui

import (
	"math"
	"sync"
	"time"

	"againrom/pkg/audio"
)

// musicDevice retains exactly one player. Replacing or stopping closes the
// previous player before another stream can start.
type musicDevice struct {
	ctx deviceOutput
	mu  sync.Mutex
	st  audio.Settings
	pl  *devicePlayer
	// paused: pl is held at its position and has not ended.
	paused     bool
	duration   time.Duration
	volume     int
	fadeVolume bool
}

// OpenMusic opens a retained player on the process-wide audio context. Failure
// is optional and callers degrade it to a nil device.
func OpenMusic(st audio.Settings) (MusicDevice, error) {
	ctx, err := openAudioContext()
	if err != nil {
		return nil, err
	}
	return &musicDevice{ctx: ctx, st: st}, nil
}

func (d *musicDevice) Start(track audio.Track) {
	d.SelectMusic(track, true)
}

func (d *musicDevice) SelectMusic(track audio.Track, playing bool) {
	if d == nil || track.Rate != audio.DeviceRate || len(track.StereoPCM) < 4 || len(track.StereoPCM)%4 != 0 {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopLocked()
	d.pl = newDevicePlayer(d.ctx.NewPlayerFromBytes(track.StereoPCM))
	d.duration = time.Duration(track.Frames()) * time.Second / time.Duration(track.Rate)
	d.applyVolumeLocked()
	d.paused = !playing
	if playing {
		d.pl.Play()
	}
}

func (d *musicDevice) Stop() {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.stopLocked()
	d.mu.Unlock()
}

// Pause holds the stream at its position.
func (d *musicDevice) Pause() bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.pl == nil || !d.pl.IsPlaying() {
		return false
	}
	d.pl.Pause()
	d.paused = true
	return true
}

// Resume continues a held stream from its position.
func (d *musicDevice) Resume() bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.pl == nil || !d.paused {
		return false
	}
	d.paused = false
	d.pl.Play()
	return true
}

func (d *musicDevice) stopLocked() {
	d.paused = false
	if d.pl == nil {
		return
	}
	d.pl.Pause()
	_ = d.pl.Close()
	d.pl = nil
}

func (d *musicDevice) Ended() bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.pl != nil && !d.paused && !d.pl.IsPlaying()
}

func (d *musicDevice) SetSettings(st audio.Settings) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.st = st
	d.applyVolumeLocked()
	d.mu.Unlock()
}

func (d *musicDevice) MusicVolume() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fadeVolume {
		return d.volume
	}
	gain := musicVolume(d.st)
	if gain <= 0 {
		return -10000
	}
	return max(-10000, int(math.Round(2000*math.Log10(gain))))
}

func (d *musicDevice) SetMusicVolume(volume int) {
	d.mu.Lock()
	d.volume, d.fadeVolume = volume, true
	d.applyVolumeLocked()
	d.mu.Unlock()
}

func (d *musicDevice) ResetMusicVolume() {
	d.mu.Lock()
	d.fadeVolume = false
	d.applyVolumeLocked()
	d.mu.Unlock()
}

func (d *musicDevice) applyVolumeLocked() {
	if d.pl == nil {
		return
	}
	gain := musicVolume(d.st)
	if d.fadeVolume {
		gain = float64(audio.SpatialPlacement(audio.SpatialTerms{Attenuation: d.volume}).Left) / audio.GainUnit
		if d.st.Muted || d.st.Master <= 0 {
			gain = 0
		}
	}
	d.pl.SetVolume(gain)
}

func (d *musicDevice) MusicTimes() (time.Duration, time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.pl == nil {
		return -1, -1
	}
	return max(0, d.duration-d.pl.Position()), -1
}

func musicVolume(st audio.Settings) float64 {
	if st.Muted || st.Master <= 0 {
		return 0
	}
	if st.Master >= audio.MasterUnit {
		return 1
	}
	return float64(st.Master) / float64(audio.MasterUnit)
}
