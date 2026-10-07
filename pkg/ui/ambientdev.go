package ui

import (
	"bytes"
	"sync"
	"time"

	ebitenaudio "github.com/hajimehoshi/ebiten/v2/audio"

	"againrom/pkg/audio"
)

type ambientStream struct {
	sample audio.Sample
	player *devicePlayer
}

// ambientDevice owns one retained stream for each recovered loop selector.
// Placement changes rebuild only that stream and seek to its prior phase.
type ambientDevice struct {
	ctx   *ebitenaudio.Context
	mu    sync.Mutex
	st    audio.Settings
	loops [ambientLoopCount]ambientStream
}

// OpenAmbient opens the retained ambience device on the process-wide context.
// Failure is optional and callers degrade it to silence.
func OpenAmbient(st audio.Settings) (AmbientDevice, error) {
	ctx, err := openAudioContext()
	if err != nil {
		return nil, err
	}
	return &ambientDevice{ctx: ctx, st: st}, nil
}

func (d *ambientDevice) StartLoop(kind AmbientLoop, sample audio.Sample, placement audio.Placement) {
	if d == nil || !validAmbientLoop(kind) {
		return
	}
	d.mu.Lock()
	d.stopLoopLocked(kind)
	d.startLoopLocked(kind, sample, placement, 0)
	d.mu.Unlock()
}

func (d *ambientDevice) MoveLoop(kind AmbientLoop, placement audio.Placement) {
	if d == nil || !validAmbientLoop(kind) {
		return
	}
	d.mu.Lock()
	stream := d.loops[kind]
	if stream.player == nil {
		d.mu.Unlock()
		return
	}
	position := stream.player.Position()
	d.stopLoopLocked(kind)
	d.startLoopLocked(kind, stream.sample, placement, position)
	d.mu.Unlock()
}

func (d *ambientDevice) StopLoop(kind AmbientLoop) {
	if d == nil || !validAmbientLoop(kind) {
		return
	}
	d.mu.Lock()
	d.stopLoopLocked(kind)
	d.mu.Unlock()
}

func (d *ambientDevice) Stop() {
	if d == nil {
		return
	}
	d.mu.Lock()
	for kind := AmbientLoop(0); kind < ambientLoopCount; kind++ {
		d.stopLoopLocked(kind)
	}
	d.mu.Unlock()
}

func (d *ambientDevice) SetSettings(st audio.Settings) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.st = st
	for i := range d.loops {
		if d.loops[i].player != nil {
			d.loops[i].player.SetVolume(musicVolume(st))
		}
	}
	d.mu.Unlock()
}

func (d *ambientDevice) startLoopLocked(kind AmbientLoop, sample audio.Sample, placement audio.Placement, position time.Duration) {
	if sample.Rate != audio.DeviceRate || len(sample.PCM) == 0 {
		return
	}
	buf := audio.Stereo(sample, placement, audio.DefaultSettings)
	if len(buf) < 4 || len(buf)%4 != 0 {
		return
	}
	loop := ebitenaudio.NewInfiniteLoop(bytes.NewReader(buf), int64(len(buf)))
	raw, err := d.ctx.NewPlayer(loop)
	if err != nil {
		return
	}
	player := newDevicePlayer(raw)
	duration := time.Duration(len(sample.PCM)) * time.Second / time.Duration(sample.Rate)
	if position > 0 && duration > 0 {
		_ = player.Seek(position % duration)
	}
	player.SetVolume(musicVolume(d.st))
	player.Play()
	d.loops[kind] = ambientStream{sample: sample, player: player}
}

func (d *ambientDevice) stopLoopLocked(kind AmbientLoop) {
	stream := &d.loops[kind]
	if stream.player != nil {
		stream.player.Pause()
		_ = stream.player.Close()
	}
	*stream = ambientStream{}
}

func validAmbientLoop(kind AmbientLoop) bool { return kind < ambientLoopCount }
