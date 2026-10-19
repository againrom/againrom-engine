package town

import (
	"image"
	"time"
)

// SoundCycle opens once per visit, then cycles endpoint sounds and a drawn
// hold before stopping them and requesting a close. Delays count from Enter
// or the previous close, not from the opening cue.
type SoundCycle struct {
	still
	spec            *ActorSpec
	last            time.Time
	delay, hold     time.Duration
	opened, running bool
}

func (c *SoundCycle) reset(s *Scene) {
	c.last, c.delay, c.hold = time.Time{}, 0, 0
	c.opened, c.running = false, false
}

func (c *SoundCycle) enter(s *Scene, now time.Time) {
	c.reset(s)
	c.last = now
	c.delay, c.hold = s.pickWait(*c.spec.Delay), s.pickWait(*c.spec.HoldPick)
}

func (c *SoundCycle) resume(s *Scene, now time.Time) {
	c.last, c.running = now, false
}

func (c *SoundCycle) phase(s *Scene, phase, name string, now time.Time) {
	if phase != "advance" || !s.ready || !s.active {
		return
	}
	column := s.byName[c.spec.Column].(*Target)
	endpoint := -1
	for i, frame := range column.spec.Endpoints {
		if column.Ready && column.Frame == frame {
			endpoint = i
			break
		}
	}
	if endpoint < 0 {
		return
	}
	elapsed := now.Sub(c.last)
	if c.running {
		if elapsed <= c.hold {
			return
		}
		for _, state := range c.spec.States {
			for _, slot := range state.EndSounds {
				s.stop(slot)
			}
		}
		s.sound(c.spec.LastSound, "")
		c.last, c.running = now, false
		c.hold = s.pickWait(*c.spec.HoldPick)
		return
	}
	if !c.opened {
		if elapsed > time.Duration(c.spec.DelayMS)*time.Millisecond {
			s.sound(c.spec.FirstSound, "")
			c.opened = true
		}
		return
	}
	if elapsed > c.delay {
		for _, slot := range c.spec.States[endpoint].StartSounds {
			s.sound(slot, "")
		}
		c.last, c.running = now, true
		c.delay = s.pickWait(*c.spec.Delay)
	}
}

func (c *SoundCycle) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {}
