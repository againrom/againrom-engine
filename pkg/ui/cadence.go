package ui

import "time"

// cadenceGate says how an elapsed time meets a cadence's period.
type cadenceGate uint8

const (
	// gateAfter admits when the elapsed time is strictly more than the period.
	gateAfter cadenceGate = iota
	// gateAtLeast admits when the elapsed time reaches the period.
	gateAtLeast
)

// cadence is the one descriptor of a presentation cadence: how long after a
// stamp the next step, flip or show is admitted. phase is time already counted
// toward the period when the stamp is taken, gate how the period is met. Every
// animated presentation asks its clock through due or dueMillis and nowhere
// else; elapsed-time measurements (durations, timeouts, windows) are not
// cadences and stay where they are.
type cadence struct {
	period time.Duration
	phase  time.Duration
	gate   cadenceGate
}

// met reports whether elapsed, counted with the phase, meets the period.
func (c cadence) met(elapsed, period int64) bool {
	if c.gate == gateAtLeast {
		return elapsed >= period
	}
	return elapsed > period
}

// due reports whether the cadence admits at now, given the stamp last. A
// stamp in the future is not yet due.
func (c cadence) due(last, now time.Time) bool {
	return c.met(int64(now.Sub(last)+c.phase), int64(c.period))
}

// dueMillis is due for a clock read in whole milliseconds.
func (c cadence) dueMillis(last, now int64) bool {
	return c.met(now-last+c.phase.Milliseconds(), c.period.Milliseconds())
}

// reached is due for a deadline: it reports whether now is past the deadline
// (gateAfter) or at or past it (gateAtLeast). The period is zero.
func (g cadenceGate) reached(deadline, now time.Time) bool {
	return cadence{gate: g}.due(deadline, now)
}
