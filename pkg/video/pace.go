package video

import "time"

// timerPacer is the decoder's timer wait (VIDEO-073): a deadline in units of
// 10 microseconds, held in 32-bit arithmetic exactly as the installed decoder
// holds it. It is the pacing a movie uses when no sound position is
// available. The first poll arms the deadline and reports not ready; a later
// poll is ready once the clock has reached the deadline, and re-arms it
// according to how late the poll came.
type timerPacer struct {
	interval uint32 // header interval in units of 10 microseconds
	deadline uint32
	armed    bool
}

func newTimerPacer(interval uint32) *timerPacer {
	return &timerPacer{interval: interval}
}

// poll reports whether the next frame is ready at clock value now.
func (p *timerPacer) poll(now uint32) bool {
	if !p.armed {
		p.armed = true
		p.deadline = now + p.interval
		return false
	}
	if now < p.deadline {
		return false
	}
	late := now - p.deadline
	switch {
	case late < p.interval:
		p.deadline = now + p.interval
	case late-p.interval < p.interval:
		p.deadline += p.interval
	default:
		p.deadline = now + p.interval
	}
	return true
}

// remaining is how many units of the clock the pacer still waits at now.
// It is zero when the next poll is ready.
func (p *timerPacer) remaining(now uint32) uint32 {
	if !p.armed || now >= p.deadline {
		return 0
	}
	return p.deadline - now
}

// pacerClock supplies the decoder clock in units of 10 microseconds, in
// 32-bit arithmetic, and a sleep that a stop signal interrupts.
type pacerClock struct {
	start time.Time
	stop  <-chan struct{}
}

// clockBase keeps the clock far from its 32-bit origin, as a clock counting
// from machine start is: a deadline that wraps below zero is then in the past
// rather than a wait of the whole 32-bit range.
const clockBase = 1 << 30

func (c pacerClock) now() uint32 {
	return clockBase + uint32(time.Since(c.start)/(10*time.Microsecond))
}

// sleep waits for the given units or until stop closes; false means stopped.
func (c pacerClock) sleep(units uint32) bool {
	t := time.NewTimer(time.Duration(units) * 10 * time.Microsecond)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-c.stop:
		return false
	}
}

// waitFrame blocks until the pacer is ready for the next frame. False means
// the stop signal fired first.
func waitFrame(p *timerPacer, c pacerClock) bool {
	return waitFrameSynced(p, c, nil, nil, 0)
}
