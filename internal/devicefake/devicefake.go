// Package devicefake is the one controlled delivery device player for tests.
// It satisfies ui.DeliveryDevicePlayer structurally and imports no game
// package; only test files import it.
//
// With a nil Clock the test drives the device by hand: it sets Playing and At
// itself and Seek records the position it was given. With a Clock the device
// consumes its reader at Rate frames per second of that clock whenever Pump,
// IsPlaying or Pause observe it, so a voice finishes as it would on a sound
// card, and Position is the consumed byte count.
package devicefake

import (
	"io"
	"time"

	"againrom/pkg/audio"
)

const bytesPerFrame = 4

// Player is one controlled delivery device player.
type Player struct {
	Reader  io.ReadSeeker
	Playing bool
	// At is the reported position of a hand-driven player.
	At            time.Duration
	Gain          float64
	Closes, Seeks int
	// Clock, when set, makes the player consume Reader against it.
	Clock func() time.Time
	// Rate is the consumed frames per second; zero means audio.DeviceRate.
	Rate int

	last time.Time
	pos  int64
}

func (p *Player) rate() int64 {
	if p.Rate > 0 {
		return int64(p.Rate)
	}
	return audio.DeviceRate
}

func (p *Player) closed() bool { return p.Closes > 0 }

// Pump consumes the reader up to the clock. It does nothing without a Clock.
func (p *Player) Pump() {
	if p.Clock == nil || !p.Playing || p.closed() {
		return
	}
	now := p.Clock()
	if now.Before(p.last) {
		p.last = now
	}
	want := int64(now.Sub(p.last)) * p.rate() / int64(time.Second) * bytesPerFrame
	p.last = now
	buf := make([]byte, 4096)
	for want > 0 {
		n := min(want, int64(len(buf)))
		k, err := p.Reader.Read(buf[:n])
		p.pos += int64(k)
		want -= int64(k)
		if err == io.EOF || k == 0 {
			p.Playing = false
			return
		}
	}
}

func (p *Player) Play() {
	if p.Clock == nil {
		p.Playing = true
		return
	}
	if !p.closed() && !p.Playing {
		p.Playing, p.last = true, p.Clock()
	}
}

func (p *Player) IsPlaying() bool {
	p.Pump()
	return p.Playing && (p.Clock == nil || !p.closed())
}

func (p *Player) Pause() { p.Pump(); p.Playing = false }

func (p *Player) Seek(at time.Duration) error {
	p.Seeks++
	var b int64
	if p.Clock != nil {
		b = int64(at) * p.rate() / int64(time.Second) * bytesPerFrame
		p.pos = b
	} else {
		b = int64(at) * p.rate() * bytesPerFrame / int64(time.Second)
		p.At = at
	}
	_, err := p.Reader.Seek(b, io.SeekStart)
	return err
}

func (p *Player) Position() time.Duration {
	if p.Clock == nil {
		return p.At
	}
	return time.Duration(p.pos / bytesPerFrame * int64(time.Second) / p.rate())
}

func (p *Player) SetVolume(gain float64) { p.Gain = gain }
func (p *Player) Volume() float64        { return p.Gain }

func (p *Player) Close() error {
	p.Closes++
	p.Playing = false
	return nil
}
