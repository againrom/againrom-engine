package video

import (
	"context"
	"fmt"
	"image"
	"io"
	"sync"
	"time"
)

// StallLimit bounds a decoder that opens but never delivers another frame or
// exit. The process owner must make Close interrupt its reader.
const StallLimit = 10 * time.Second

type event struct {
	frame         *image.RGBA
	audio         []byte
	audioRate     uint32
	audioChannels uint8
	finalFrame    bool
	err           error
}

// Player never blocks the UI. A one-event queue bounds read-ahead; the producer
// owns at most one additional frame. All public methods run on the UI goroutine.
type Player struct {
	input       io.ReadCloser
	cancel      context.CancelFunc
	events      chan event
	closed      bool
	frame       *image.RGBA
	frameNumber uint32
	finalFrame  bool
	// audioChunk is the PCM delivered alongside the most recently advanced
	// frame, or nil for a video-only stream or a silent frame. audioRate and
	// audioChannels are the stream's own constant audio format, known from the
	// first delivered frame onward and zero before that and for a stream with
	// no audio track.
	audioChunk    []byte
	audioRate     uint32
	audioChannels uint8
	err           error
	progress      time.Time
	done          chan struct{}
	once          sync.Once
	// sound carries the sound clock to the decoder goroutine of a Smacker
	// stream; nil for any other stream.
	sound *soundSource
}

func NewPlayer(input io.ReadCloser) *Player {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Player{input: input, cancel: cancel, events: make(chan event, 1), done: make(chan struct{})}
	go p.read(ctx)
	return p
}

func (p *Player) read(ctx context.Context) {
	defer close(p.done)
	defer close(p.events)
	send := func(e event) bool {
		select {
		case p.events <- e:
			return true
		case <-ctx.Done():
			return false
		}
	}
	if p.input == nil {
		send(event{err: fmt.Errorf("video: no decoder stream")})
		return
	}
	info, err := ReadHeader(p.input)
	if err != nil {
		send(event{err: err})
		return
	}
	for n := uint32(0); n < info.Frames; n++ {
		frame, audio, err := ReadFrame(p.input, info)
		if !send(event{frame: frame, audio: audio, audioRate: info.AudioRate, audioChannels: info.AudioChannels, finalFrame: n+1 == info.Frames, err: err}) || err != nil {
			return
		}
	}
	// The final frame and stream disposal are separate events.
	var extra [1]byte
	n, err := p.input.Read(extra[:])
	if n != 0 {
		err = fmt.Errorf("video: data after declared last frame")
	} else if err == io.EOF {
		err = nil
	} else if err == nil {
		err = io.ErrNoProgress
	}
	send(event{err: err})
}

// Advance retains each delivered frame for at least one UI tick. False means
// completion, cancellation or failure; callers return to their prior screen.
func (p *Player) Advance(now time.Time) bool {
	if p == nil || p.closed {
		return false
	}
	if p.progress.IsZero() {
		p.progress = now
	}
	select {
	case e, ok := <-p.events:
		if !ok || e.frame == nil || e.err != nil {
			p.err = e.err
			p.Close()
			return false
		}
		p.frame, p.audioChunk, p.progress = e.frame, e.audio, now
		p.audioRate, p.audioChannels = e.audioRate, e.audioChannels
		p.finalFrame = e.finalFrame
		p.frameNumber++
	default:
		if now.Sub(p.progress) >= StallLimit {
			p.err = fmt.Errorf("video: decoder stalled")
			p.Close()
			return false
		}
	}
	return true
}

// FrameNumber is one-based, or zero before the first delivered frame.
func (p *Player) FrameNumber() uint32 {
	if p == nil {
		return 0
	}
	return p.frameNumber
}

// FinalFrame reports the declared last frame, independently of stream EOF.
func (p *Player) FinalFrame() bool { return p != nil && p.finalFrame }

func (p *Player) Frame() *image.RGBA {
	if p == nil {
		return nil
	}
	return p.frame
}

// AudioChunk is the PCM delivered with the frame Frame() currently returns,
// or nil for a video-only stream or a frame the decoder produced no audio
// for. It is consumed once by design: a caller that wants to play it forward
// reads it exactly once per FrameNumber change, exactly as Frame() itself is
// read once per change.
func (p *Player) AudioChunk() []byte {
	if p == nil {
		return nil
	}
	return p.audioChunk
}

// AudioFormat is the stream's own constant rate and channel count, valid
// from the first delivered frame onward; both are zero before that and for a
// stream with no audio track at all.
func (p *Player) AudioFormat() (rate uint32, channels uint8) {
	if p == nil {
		return 0, 0
	}
	return p.audioRate, p.audioChannels
}

func (p *Player) Err() error {
	if p == nil {
		return nil
	}
	return p.err
}

func (p *Player) Close() {
	if p == nil {
		return
	}
	p.closed = true
	p.once.Do(func() {
		p.cancel()
		if p.input != nil {
			_ = p.input.Close()
		}
	})
}
