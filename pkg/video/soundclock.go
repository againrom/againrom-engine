package video

import (
	"sync/atomic"
	"time"
)

// SoundClock reports how many bytes of a movie's own audio (in the movie's
// own sample format, before any channel duplication) the output device has
// played. Ok is false while no device session runs for the movie.
type SoundClock interface {
	PlayedBytes() (played int64, ok bool)
}

// soundSource carries the clock from the UI, which owns the output device, to
// the decoder goroutine that paces the frames. Until a clock is attached, and
// whenever the clock reports no session, frames are paced by the timer wait.
type soundSource struct {
	clock atomic.Pointer[SoundClock]
}

func (s *soundSource) set(c SoundClock) {
	if c == nil {
		s.clock.Store(nil)
		return
	}
	s.clock.Store(&c)
}

func (s *soundSource) played() (int64, bool) {
	if s == nil {
		return 0, false
	}
	c := s.clock.Load()
	if c == nil {
		return 0, false
	}
	return (*c).PlayedBytes()
}

// SetSoundClock attaches the sound position that paces this movie's frames
// while its audio plays (VIDEO-081). A nil clock, or a clock with no session,
// leaves the timer wait in charge. A stream not made by StartSmackerDecoder
// ignores it.
func (p *Player) SetSoundClock(c SoundClock) {
	if p == nil || p.sound == nil {
		return
	}
	p.sound.set(c)
}

const (
	// soundSlack is the margin the decoder wait allows between the byte target
	// and the played position (VIDEO-073: ready once target <= position + 8).
	soundSlack = 8
	// soundPoll is how often a sound wait re-reads the position, in 10
	// microsecond units (2 ms).
	soundPoll = 200
)

// soundStallLimit ends sound pacing for the rest of a movie when the played
// position has not moved for this long while a frame waits. The original holds
// without bound (VIDEO-081); this keeps a stopped output device from freezing
// the movie.
var soundStallLimit = 2 * time.Second

// soundPacing derives the byte target a frame waits for.
type soundPacing struct {
	// target(k) = k * bytesPerSecond * interval / 100000, the interval being in
	// units of 10 microseconds.
	bytesPerSecond int64
	interval       int64
	lost           bool
}

func newSoundPacing(info Info, interval uint32) *soundPacing {
	if !info.HasAudio() {
		return nil
	}
	return &soundPacing{
		bytesPerSecond: int64(info.AudioRate) * int64(info.audioFrameBytes()),
		interval:       int64(interval),
	}
}

// target is the played-byte count that frame k waits for: k frame intervals of
// audio, so the frame shown is the one whose audio has just played.
func (s *soundPacing) target(k int) int64 {
	return int64(k) * s.bytesPerSecond * s.interval / 100000
}

// waitFrameSynced blocks until frame k may be delivered. While a sound clock
// reports a session, the frame waits until the played bytes reach its target;
// otherwise the timer wait paces it. False means the stop signal fired first.
func waitFrameSynced(p *timerPacer, c pacerClock, snd *soundSource, sp *soundPacing, k int) bool {
	var lastPos int64 = -1
	var lastMove time.Time
	for {
		if sp != nil && !sp.lost {
			if pos, ok := snd.played(); ok {
				if pos+soundSlack >= sp.target(k) {
					return true
				}
				now := time.Now()
				if pos != lastPos {
					lastPos, lastMove = pos, now
				} else if now.Sub(lastMove) >= soundStallLimit {
					sp.lost = true
					continue
				}
				if !c.sleep(soundPoll) {
					return false
				}
				continue
			}
		}
		if p.poll(c.now()) {
			return true
		}
		select {
		case <-c.stop:
			return false
		default:
		}
		r := p.remaining(c.now())
		if r == 0 {
			continue
		}
		if sp != nil && r > soundPoll {
			// Re-check for a sound session that starts during the wait.
			r = soundPoll
		}
		if !c.sleep(r) {
			return false
		}
	}
}
