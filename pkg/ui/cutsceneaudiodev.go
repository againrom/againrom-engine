package ui

import (
	"io"
	"runtime"
	"sync"
	"time"

	"againrom/pkg/audio"
)

// OpenCutsceneAudio opens the retained cutscene-audio device on the
// process-wide context, exactly like OpenMusic and OpenAmbient. Failure is
// optional and callers degrade it to silence (following sounddev.go's own
// OpenAudio contract).
func OpenCutsceneAudio(st audio.Settings) (CutsceneAudioDevice, error) {
	ctx, err := openAudioContext()
	if err != nil {
		return nil, err
	}
	return &cutsceneAudioDevice{st: st, newPlayer: func(src io.Reader) (cutscenePlayer, error) {
		raw, err := ctx.NewPlayer(src)
		if err != nil {
			return nil, err
		}
		raw.SetBufferSize(cutsceneAudioBufferSize(runtime.GOOS))
		return newDevicePlayer(raw), nil
	}}, nil
}

func cutsceneAudioBufferSize(goos string) time.Duration {
	if goos == "darwin" {
		// The device requests about 70ms at 22050Hz; keep a full callback
		// buffered with a scheduling margin.
		return 100 * time.Millisecond
	}
	return 40 * time.Millisecond
}

type cutscenePlayer interface {
	soundVoicePlayer
	Pause()
}

// cutsceneAudioDevice streams pushed PCM through an unbounded in-process
// queue: Push appends once per delivered movie frame. An empty read supplies
// silence so the mixer's serial refill cannot block another player. Nothing
// here resamples -- Start refuses any rate but audio.DeviceRate -- and a mono
// track is duplicated into interleaved stereo before it ever reaches the
// queue, matching every other device in this package (musicVolume, Track).
type cutsceneAudioDevice struct {
	newPlayer func(io.Reader) (cutscenePlayer, error)
	mu        sync.Mutex
	st        audio.Settings
	s         *cutsceneSession
}

// cutsceneSession owns one stream's player. A gain arriving while Play holds
// the player's lock waits in pending; Stop closes channels and run makes the
// calls on the player.
type cutsceneSession struct {
	q       *cutsceneAudioQueue
	pl      cutscenePlayer
	stop    chan struct{}
	mu      sync.Mutex
	played  chan struct{}
	pending float64
	queued  bool
}

func (s *cutsceneSession) run() {
	s.pl.Play()
	s.mu.Lock()
	if s.queued {
		s.pl.SetVolume(s.pending)
	}
	close(s.played)
	s.mu.Unlock()
	<-s.stop
	s.pl.Pause()
	_ = s.pl.Close()
}

func (s *cutsceneSession) hasPlayed() bool {
	select {
	case <-s.played:
		return true
	default:
		return false
	}
}

func (s *cutsceneSession) setGain(gain float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hasPlayed() {
		s.pl.SetVolume(gain)
		return
	}
	s.pending, s.queued = gain, true
}

func (d *cutsceneAudioDevice) Start(rate, channels int) {
	if d == nil || rate != audio.DeviceRate || (channels != 1 && channels != 2) {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopLocked()
	q := newCutsceneAudioQueue(channels)
	pl, err := d.newPlayer(q)
	if err != nil {
		return
	}
	pl.SetVolume(musicVolume(d.st))
	d.s = &cutsceneSession{q: q, pl: pl, stop: make(chan struct{}), played: make(chan struct{})}
	go d.s.run()
}

func (d *cutsceneAudioDevice) Push(pcm []byte) {
	if d == nil || len(pcm) == 0 {
		return
	}
	d.mu.Lock()
	var q *cutsceneAudioQueue
	if d.s != nil {
		q = d.s.q
	}
	d.mu.Unlock()
	if q != nil {
		q.push(pcm)
	}
}

func (d *cutsceneAudioDevice) Stop() {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.stopLocked()
	d.mu.Unlock()
}

// stopLocked closes the queue first, which ends a Play still reading it, and
// leaves Pause and Close to run.
func (d *cutsceneAudioDevice) stopLocked() {
	if d.s != nil {
		d.s.q.closeQueue()
		close(d.s.stop)
		d.s = nil
	}
}

func (d *cutsceneAudioDevice) SetSettings(st audio.Settings) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.st = st
	if d.s != nil {
		d.s.setGain(musicVolume(d.st))
	}
	d.mu.Unlock()
}

type cutsceneAudioQueue struct {
	mono   bool
	played int64 // queued bytes handed to the player, silence excluded
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
}

func newCutsceneAudioQueue(channels int) *cutsceneAudioQueue {
	q := &cutsceneAudioQueue{mono: channels == 1}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// push duplicates a mono chunk into interleaved stereo before it is queued:
// ebiten's own context always mixes interleaved stereo 16-bit PCM (every
// other device in this package feeds it exactly that -- Track.StereoPCM,
// audio.Stereo's own return). A trailing odd half-sample, which never
// occurs for a chunk this project's own decoder emits (WriteFrame/ReadFrame
// both refuse one), is dropped rather than guessed at.
func (q *cutsceneAudioQueue) push(pcm []byte) {
	if len(pcm) == 0 {
		return
	}
	if q.mono {
		frames := len(pcm) / 2
		out := make([]byte, frames*4)
		for i := 0; i < frames; i++ {
			copy(out[i*4:i*4+2], pcm[i*2:i*2+2])
			copy(out[i*4+2:i*4+4], pcm[i*2:i*2+2])
		}
		pcm = out
	}
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	q.buf = append(q.buf, pcm...)
	q.cond.Broadcast()
	q.mu.Unlock()
}

func (q *cutsceneAudioQueue) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return 0, io.EOF
	}
	n := copy(p, q.buf)
	q.buf = q.buf[n:]
	q.played += int64(n)
	q.mu.Unlock()
	clear(p[n:])
	return len(p), nil
}

func (q *cutsceneAudioQueue) closeQueue() {
	q.mu.Lock()
	q.closed = true
	q.buf = nil
	q.cond.Broadcast()
	q.mu.Unlock()
}

// consumed is how many queued bytes the player has taken so far; silence
// supplied for an empty queue is not counted.
func (q *cutsceneAudioQueue) consumed() int64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.played
}

// PlayedBytes is the movie's own audio the device has played, in the movie's
// own sample format (VIDEO-081's position, from the player's read count less
// what the player still buffers). Ok is false while no session runs.
func (d *cutsceneAudioDevice) PlayedBytes() (int64, bool) {
	if d == nil {
		return 0, false
	}
	d.mu.Lock()
	s := d.s
	d.mu.Unlock()
	if s == nil {
		return 0, false
	}
	n := s.q.consumed()
	if b, ok := s.pl.(interface{ BufferedSize() int }); ok {
		n -= min(int64(b.BufferedSize()), n)
	}
	if s.q.mono {
		n /= 2
	}
	return n, true
}
