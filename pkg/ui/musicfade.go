package ui

import "time"

type musicFadeDevice interface {
	MusicVolume() int
	SetMusicVolume(int)
	ResetMusicVolume()
	MusicTimes() (remaining, buffered time.Duration)
}

type musicFade struct {
	volume, threshold int
	scale             float64
	due               time.Time
	interval          time.Duration
	terminal          bool
}

func (m *MusicController) Tune(index int) bool {
	if m == nil || m.desc == nil || m.desc.Fade == nil || index < 0 || index >= len(m.Candidates()) {
		return false
	}
	m.hold, m.heldIndex = true, index
	m.beginFade()
	return true
}

func (m *MusicController) resetVolume() {
	if d, ok := m.device.(musicFadeDevice); ok {
		d.ResetMusicVolume()
	}
}

func (m *MusicController) beginFade() {
	spec := m.desc.Fade
	t := spec.DurationMS
	v := 0
	if d, ok := m.device.(musicFadeDevice); ok {
		v = d.MusicVolume()
		r, q := d.MusicTimes()
		for _, span := range []time.Duration{r, q} {
			if span >= 0 {
				t = min(t, int(span/time.Millisecond))
			}
		}
	}
	f := &musicFade{volume: v, threshold: spec.Threshold, interval: time.Duration(spec.IntervalMS) * time.Millisecond,
		terminal: t <= 0 || v <= -spec.Threshold}
	f.due = m.now().Add(f.interval)
	if !f.terminal {
		q := spec.Threshold * spec.IntervalMS / t
		f.scale = float64(q) * float64(spec.Threshold+v) / float64(spec.Threshold)
	}
	m.fade = f
}

func (m *MusicController) stepFade(now time.Time) {
	f := m.fade
	if f == nil || !m.active || !gateAtLeast.reached(f.due, now) {
		return
	}
	f.due = f.due.Add(f.interval)
	if f.terminal || f.volume < -f.threshold {
		m.pause()
		return
	}
	// R2-ENGINE-839. The stop test precedes the step; no threshold is forced.
	x := (float64(f.volume) + float64(f.threshold)/4) / (float64(f.threshold) / 2)
	x *= x
	f.volume -= int(f.scale * (0.3*x*x + 0.9))
	if d, ok := m.device.(musicFadeDevice); ok {
		d.SetMusicVolume(f.volume)
	}
}

type MusicCue struct {
	index   int
	pending bool
}

func (c *MusicCue) QueueMusicTune(index int) {
	if index >= 0 {
		c.index, c.pending = index, true
	}
}

func (c *MusicCue) TakeMusicTune() (int, bool) {
	index, pending := c.index, c.pending
	c.pending = false
	return index, pending
}
