package town

import "time"

func (s *Scene) random(name string) *RandomSpec {
	for i := range s.spec.Random {
		if s.spec.Random[i].Name == name {
			return &s.spec.Random[i]
		}
	}
	return nil
}

// draw is one value in the named arithmetic form from the named source:
// a host source answers its own bounded draw; a generator answers
// "scaled" (r*n/mask) % n or "masked" (r*n/mask) & (n-1).
func (s *Scene) draw(source, form string, n int) int {
	r := s.random(source)
	if r == nil || n <= 0 {
		return 0
	}
	if r.Source == "host" {
		return s.host.Draw(source, n)
	}
	g := s.generator(r)
	raw := g.next(s.host.Seed())
	scaled := raw * n / int(r.Mask)
	if form == "masked" {
		return scaled & (n - 1)
	}
	return scaled % n
}

// Pick answers one draw of the named source in the named arithmetic form.
func (s *Scene) Pick(source, form string, n int) int { return s.draw(source, form, n) }

func (s *Scene) generator(r *RandomSpec) *lcg {
	g := s.proc.lcg[r.Name]
	if g == nil {
		g = &lcg{spec: r}
		s.proc.lcg[r.Name] = g
	}
	return g
}

// RawDraw answers a generator's replacement raw values, nil when it runs.
func (s *Scene) RawDraw(source string) func() int {
	if r := s.random(source); r != nil && r.Source != "host" {
		return s.generator(r).raw
	}
	return nil
}

// SetRawDraw replaces a generator's raw values; tests pin a sequence with it.
func (s *Scene) SetRawDraw(source string, raw func() int) {
	if r := s.random(source); r != nil && r.Source != "host" {
		s.generator(r).raw = raw
	}
}

func (s *Scene) chance(c ChanceSpec) bool {
	return s.draw(c.Draw, "", c.N) > c.Above
}

// amount evaluates a pick: base plus a draw of n in the pick's form, times
// Times; or, with Raw, base plus a raw draw r of [0,Raw) divided by Divide or
// scaled in the "scaled" form.
func (s *Scene) amount(p PickSpec) int {
	if p.Raw > 0 {
		r := s.draw(p.Draw, "", p.Raw)
		switch {
		case p.Form == "scaled":
			return p.Base + (r*p.N/(p.Raw-1))%p.N
		case p.Divide > 0:
			return p.Base + r/p.Divide
		}
		return p.Base + r
	}
	v := s.draw(p.Draw, p.Form, p.N)
	if p.Times > 0 {
		v *= p.Times
	}
	return p.Base + v
}

// pickWait is a pick read as milliseconds.
func (s *Scene) pickWait(p PickSpec) time.Duration {
	return time.Duration(s.amount(p)) * time.Millisecond
}

// wait is a wait spec's base plus its drawn span.
func (s *Scene) wait(w WaitSpec) time.Duration {
	return time.Duration(w.BaseMS+s.draw(w.Draw, w.Form, w.SpanMS)) * time.Millisecond
}

// lcg is a linear congruential generator seeded once from the Host's seed.
type lcg struct {
	spec   *RandomSpec
	state  uint32
	seeded bool
	raw    func() int
}

func (g *lcg) next(seed int64) int {
	if g.raw != nil {
		return g.raw()
	}
	if !g.seeded {
		g.state, g.seeded = uint32(seed), true
	}
	g.state = g.state*g.spec.Multiplier + g.spec.Increment
	return int(g.state>>g.spec.Shift) & int(g.spec.Mask)
}

// Wait is a flock's process-scoped wait latch.
type Wait struct {
	Ready bool
	Wait  time.Duration
}

// WaitLatch answers a flock's wait latch.
func (s *Scene) WaitLatch(name string) *Wait { return s.proc.WaitLatch(name) }

// WaitLatch answers a flock's process-scoped wait latch.
func (p *Process) WaitLatch(name string) *Wait {
	p.init()
	w := p.waits[name]
	if w == nil {
		w = &Wait{}
		p.waits[name] = w
	}
	return w
}

// EndCount answers a held episode's process-scoped end counter.
func (s *Scene) EndCount(name string) int { return s.proc.ends[name] }

// SetEndCount sets a held episode's end counter.
func (s *Scene) SetEndCount(name string, n int) { s.proc.ends[name] = n }

// ResetDraw returns a generator to unseeded; its next draw seeds it again.
func (s *Scene) ResetDraw(source string) { delete(s.proc.lcg, source) }

// RestartGenerators returns every generator the description keeps to
// unseeded, in place: a new session's next draw seeds each again from the
// host's seed. A replacement raw source stays.
func (p *Process) RestartGenerators() {
	for _, g := range p.lcg {
		g.seeded = false
	}
}
