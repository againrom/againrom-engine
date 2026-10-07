package ui

import (
	"io"
	"sync"
	"testing"
	"time"

	"againrom/pkg/audio"
)

// heldPlayer holds its lock through Play while it reads fill bytes from src,
// as ebiten's player does, and records every call made on it.
type heldPlayer struct {
	src     io.Reader
	fill    int
	lock    sync.Mutex
	playing chan struct{}

	logMu sync.Mutex
	log   []string
	gain  float64
}

// heldFillSource keeps the player-lock contention test adversarial even when
// the production queue supplies silence on an empty read. Push or Stop ends
// this test-only wait through the queue's existing condition signal.
type heldFillSource struct {
	q    *cutsceneAudioQueue
	fill int
}

func (s *heldFillSource) Read(p []byte) (int, error) {
	s.q.mu.Lock()
	for len(s.q.buf) < s.fill && !s.q.closed {
		s.q.cond.Wait()
	}
	s.q.mu.Unlock()
	return s.q.Read(p)
}

func (p *heldPlayer) record(call string) {
	p.logMu.Lock()
	p.log = append(p.log, call)
	p.logMu.Unlock()
}

func (p *heldPlayer) calls() []string {
	p.logMu.Lock()
	defer p.logMu.Unlock()
	return append([]string(nil), p.log...)
}

func (p *heldPlayer) Play() {
	p.lock.Lock()
	defer p.lock.Unlock()
	close(p.playing)
	buf := make([]byte, p.fill)
	for got := 0; got < p.fill; {
		n, err := p.src.Read(buf[got:])
		got += n
		if err != nil {
			break
		}
	}
	p.record("play")
}

func (p *heldPlayer) held(call string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.record(call)
}

func (p *heldPlayer) Pause()          { p.held("pause") }
func (p *heldPlayer) Close() error    { p.held("close"); return nil }
func (p *heldPlayer) IsPlaying() bool { p.held("isplaying"); return true }
func (p *heldPlayer) SetVolume(gain float64) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.logMu.Lock()
	p.log, p.gain = append(p.log, "setvolume"), gain
	p.logMu.Unlock()
}
func (p *heldPlayer) Volume() float64 {
	p.logMu.Lock()
	defer p.logMu.Unlock()
	return p.gain
}

func heldCutsceneDevice(st audio.Settings, fill int) (*cutsceneAudioDevice, *heldPlayer) {
	player := &heldPlayer{fill: fill, playing: make(chan struct{})}
	return &cutsceneAudioDevice{st: st, newPlayer: func(src io.Reader) (cutscenePlayer, error) {
		player.src = &heldFillSource{q: src.(*cutsceneAudioQueue), fill: fill}
		return player, nil
	}}, player
}

func waitFor(t *testing.T, what string, done func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); !done(); time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("%s did not happen within 5s", what)
		}
	}
}

// While Play waits for audio that has not been pushed, no device call waits
// on it: the next gain is applied after Play, and Stop ends Play before
// Pause and Close run.
func TestCutsceneAudioCallsDoNotWaitOnAFillingPlay(t *testing.T) {
	st := audio.Settings{Master: audio.MasterUnit / 2}
	d, player := heldCutsceneDevice(st, 4*audio.DeviceRate)
	d.Start(audio.DeviceRate, 2)
	<-player.playing
	louder := audio.Settings{Master: audio.MasterUnit}
	withinDeviceBound(t, func() {
		d.SetSettings(louder)
		d.Push(make([]byte, 4))
		if _, gains, ok := AudioDeviceState(d); !ok || gains != nil {
			t.Errorf("AudioDeviceState while Play fills = %v, %v, want no gain", gains, ok)
		}
	})
	d.Push(make([]byte, 4*audio.DeviceRate))
	waitFor(t, "the pushed gain", func() bool { return player.Volume() == musicVolume(louder) })
	withinDeviceBound(t, func() {
		if _, gains, ok := AudioDeviceState(d); !ok || len(gains) != 1 || gains[0] != musicVolume(louder) {
			t.Errorf("AudioDeviceState after Play = %v, %v, want [%v]", gains, ok, musicVolume(louder))
		}
	})

	d.Stop()
}

func TestCutsceneAudioStopEndsAFillingPlayBeforeClosing(t *testing.T) {
	d, player := heldCutsceneDevice(audio.Settings{Master: audio.MasterUnit}, 4*audio.DeviceRate)
	d.Start(audio.DeviceRate, 2)
	<-player.playing
	withinDeviceBound(t, d.Stop)
	waitFor(t, "close", func() bool { calls := player.calls(); return len(calls) > 0 && calls[len(calls)-1] == "close" })
	calls := player.calls()
	want := []string{"setvolume", "play", "pause", "close"}
	if len(calls) != len(want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls = %v, want %v", calls, want)
		}
	}
}

type bufferedPlayer struct {
	heldPlayer
	buffered int
}

func (p *bufferedPlayer) BufferedSize() int { return p.buffered }

// The device position counts queued bytes the player took, less what the
// player still buffers, in the movie's own sample format; silence for an empty
// queue adds nothing.
func TestCutsceneDevicePlayedBytes(t *testing.T) {
	bp := &bufferedPlayer{}
	d := &cutsceneAudioDevice{newPlayer: func(src io.Reader) (cutscenePlayer, error) {
		bp.heldPlayer = heldPlayer{src: src, playing: make(chan struct{})}
		return bp, nil
	}}
	if _, ok := d.PlayedBytes(); ok {
		t.Fatal("position reported with no session")
	}
	d.Start(audio.DeviceRate, 1)
	defer d.Stop()
	q := d.s.q
	buf := make([]byte, 64)
	q.Read(buf) // empty queue: silence only
	if n, ok := d.PlayedBytes(); !ok || n != 0 {
		t.Fatalf("after silence: %d, %v; want 0, true", n, ok)
	}
	q.push(make([]byte, 20)) // 20 mono bytes queue as 40 stereo bytes
	q.Read(buf)
	if n, _ := d.PlayedBytes(); n != 20 {
		t.Fatalf("after 40 stereo bytes: %d, want 20 mono bytes", n)
	}
	bp.buffered = 10
	if n, _ := d.PlayedBytes(); n != 15 {
		t.Fatalf("with 10 stereo bytes buffered: %d, want 15", n)
	}
	d.Stop()
	if _, ok := d.PlayedBytes(); ok {
		t.Fatal("position reported after Stop")
	}
}
