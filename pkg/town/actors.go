package town

import (
	"image"
	"strconv"
	"time"
)

// actor is one program's runtime. reset returns it to a view just built and
// prepare to a view just made ready; enter runs on each entry to the visible
// square and leave on leaving it; phase runs one named clock phase; arm and
// drive are hover reactions; draw paints the named member of the actor for
// one layer.
type actor interface {
	members() []string
	reset(v *View)
	prepare(v *View)
	enter(v *View, now time.Time)
	leave(v *View)
	phase(v *View, phase, name string, now time.Time)
	arm(v *View, name string)
	drive(v *View, dir int)
	activeNow() bool
	draw(v *View, dst *image.RGBA, name string, layer LayerSpec)
}

func newActor(spec *ActorSpec) actor {
	switch spec.Program {
	case "episode":
		return &Episode{spec: spec}
	case "pendulum":
		return &Pendulum{spec: spec, Members: make([]Swing, len(spec.Members))}
	case "stepper":
		return &Stepper{spec: spec}
	case "driven":
		return &Driven{spec: spec}
	case "flock":
		return &Flock{spec: spec, Progress: make([]int, spec.GroupSize)}
	case "families":
		return &Families{spec: spec, Members: make([]FamilyMember, len(spec.Members))}
	}
	return nil
}

// frameAt answers frame i of a sequence, nil out of range.
func frameAt(frames []image.Image, i int) image.Image {
	if i < 0 || i >= len(frames) {
		return nil
	}
	return frames[i]
}

// restFrame is "first" or "last" of a loaded sequence; -1 with none loaded.
func restFrame(rule string, n int) int {
	if rule == "first" {
		if n == 0 {
			return -1
		}
		return 0
	}
	return n - 1
}

// Episode is a one-shot sequence. An armed episode advances one frame per
// admitted step. Its start cue plays at frame 0. A "rewind" episode returns to
// frame 0 and disarms after its last frame, and is frozen with no frames
// loaded. A "hold" episode runs a fixed number of frames whatever is loaded,
// stays on its last count hidden, and rewinds only every RewindEvery ends, a
// count that outlives the view.
type Episode struct {
	spec    *ActorSpec
	Frame   int
	Enabled bool
	Visible bool
}

func (e *Episode) members() []string { return nil }

func (e *Episode) prepare(v *View) {}

func (e *Episode) reset(v *View) { e.Frame, e.Enabled, e.Visible = 0, false, true }

func (e *Episode) enter(v *View, now time.Time) {
	if e.spec.ResetOnEntry {
		e.Enabled, e.Frame, e.Visible = false, 0, true
	}
}

func (e *Episode) leave(v *View) {
	if e.spec.ResetOnEntry {
		e.Enabled, e.Visible = false, false
	}
}

func (e *Episode) arm(v *View, name string) { e.Enabled = true }
func (e *Episode) drive(v *View, dir int)   {}
func (e *Episode) activeNow() bool          { return e.Enabled }

func (e *Episode) cue(v *View) {
	c := e.spec.StartSound
	if c == nil {
		return
	}
	for _, s := range c.Stop {
		v.stop(s)
	}
	v.sound(c.Slot, c.Key)
}

func (e *Episode) phase(v *View, phase, name string, now time.Time) {
	switch phase {
	case "trigger":
		if e.spec.Trigger != nil && v.chance(*e.spec.Trigger) {
			e.Enabled = true
		}
	case "advance":
		if e.spec.End == "hold" {
			e.advanceHeld(v)
			return
		}
		count := len(v.frames(e.spec.Art))
		if e.Enabled && count > 0 && e.Frame == 0 {
			e.cue(v)
		}
		if !e.Enabled || count == 0 {
			return
		}
		e.Frame++
		if e.Frame >= count {
			e.Frame, e.Enabled = 0, false
		}
	}
}

func (e *Episode) loaded(v *View) bool { return len(v.frames(e.spec.Art)) == e.spec.Frames }

func (e *Episode) advanceHeld(v *View) {
	if !e.Enabled {
		return
	}
	if e.Frame == 0 {
		e.cue(v)
	}
	e.Frame++
	if e.Frame < e.spec.Frames {
		e.Visible = e.loaded(v)
		return
	}
	n := v.proc.ends[e.spec.Name] + 1
	if n >= e.spec.RewindEvery {
		e.Frame, n = 0, 0
	}
	v.proc.ends[e.spec.Name] = n
	e.Visible, e.Enabled = false, false
}

// Shown reports whether the episode's current frame is drawn.
func (e *Episode) Shown(v *View) bool {
	if e.spec.End == "hold" {
		return e.Visible && e.loaded(v)
	}
	return true
}

func (e *Episode) draw(v *View, dst *image.RGBA, name string, layer LayerSpec) {
	if !e.Shown(v) {
		return
	}
	put(dst, frameAt(v.frames(e.spec.Art), e.Frame), layer)
}

// Pendulum is a group of members swinging between their first and last frames
// under one shared enable. At either end a member rests until a chance holds;
// a member back at its first frame from a backward swing clears the shared
// enable, and the members after it still take their step that phase.
type Pendulum struct {
	spec    *ActorSpec
	Enabled bool
	Members []Swing
}

// Swing is one pendulum member's frame and step.
type Swing struct{ Frame, Step int }

func (p *Pendulum) members() []string {
	var out []string
	for _, m := range p.spec.Members {
		out = append(out, m.Name)
	}
	return out
}

func (p *Pendulum) prepare(v *View) {}

func (p *Pendulum) reset(v *View) {
	p.Enabled = false
	for i := range p.Members {
		p.Members[i] = Swing{}
	}
}

func (p *Pendulum) enter(v *View, now time.Time) {}
func (p *Pendulum) leave(v *View)                {}
func (p *Pendulum) arm(v *View, name string)     { p.Enabled = true }
func (p *Pendulum) drive(v *View, dir int)       {}
func (p *Pendulum) activeNow() bool              { return p.Enabled }

func (p *Pendulum) phase(v *View, phase, name string, now time.Time) {
	if phase != "advance" || !p.Enabled {
		return
	}
	for i, m := range p.spec.Members {
		p.swing(v, &p.Members[i], len(v.frames(m.Art)))
	}
}

func (p *Pendulum) swing(v *View, s *Swing, count int) {
	if count == 0 {
		return
	}
	if s.Frame == 0 && s.Step == -1 {
		s.Step = 0
		p.Enabled = false
		return
	}
	if s.Frame == 0 && s.Step == 0 {
		if v.chance(*p.spec.Chance) {
			s.Step = 1
		}
	} else if s.Frame == count-1 {
		s.Step = 0
		if v.chance(*p.spec.Chance) {
			s.Step = -1
		}
	}
	s.Frame += s.Step
}

// Member answers the named member's swing.
func (p *Pendulum) Member(name string) *Swing {
	for i, m := range p.spec.Members {
		if m.Name == name {
			return &p.Members[i]
		}
	}
	return nil
}

func (p *Pendulum) draw(v *View, dst *image.RGBA, name string, layer LayerSpec) {
	for i, m := range p.spec.Members {
		if m.Name == name {
			put(dst, frameAt(v.frames(m.Art), p.Members[i].Frame), layer)
		}
	}
}

// Stepper moves one frame per admitted step toward its first frame while its
// hotspot is hovered and toward its last frame otherwise. A change of target
// stops its slot and requests the target's sound. While its hold condition is
// false it shows its hold frame and keeps its target latch.
type Stepper struct {
	spec        *ActorSpec
	Frame       int
	TowardFirst bool
}

func (s *Stepper) members() []string { return nil }

func (s *Stepper) reset(v *View) { s.Frame, s.TowardFirst = 0, false }

func (s *Stepper) prepare(v *View) { s.Frame = restFrame(s.spec.Rest, len(v.frames(s.spec.Art))) }

func (s *Stepper) enter(v *View, now time.Time) {}
func (s *Stepper) leave(v *View)                {}
func (s *Stepper) arm(v *View, name string)     {}
func (s *Stepper) drive(v *View, dir int)       {}
func (s *Stepper) activeNow() bool              { return false }

func (s *Stepper) phase(v *View, phase, name string, now time.Time) {
	count := len(v.frames(s.spec.Art))
	if phase != "advance" || count == 0 {
		return
	}
	if s.spec.HoldUnless != "" && !v.host.Condition(s.spec.HoldUnless) {
		s.Frame = restFrame(s.spec.Hold, count)
		return
	}
	first := v.hover != nil && v.hover.Name == s.spec.TowardFirstOnHover
	if first != s.TowardFirst {
		v.stop(s.spec.Slot)
		key := s.spec.LastSound
		if first {
			key = s.spec.FirstSound
		}
		v.sound(s.spec.Slot, key)
		s.TowardFirst = first
	}
	if first && s.Frame > 0 {
		s.Frame--
	}
	if !first && s.Frame < count-1 {
		s.Frame++
	}
}

// Shown is the frame the stepper draws: its rest frame until the view is
// made ready.
func (s *Stepper) Shown(v *View) int {
	if !v.ready {
		return restFrame(s.spec.Rest, len(v.frames(s.spec.Art)))
	}
	return s.Frame
}

func (s *Stepper) draw(v *View, dst *image.RGBA, name string, layer LayerSpec) {
	put(dst, frameAt(v.frames(s.spec.Art), s.Shown(v)), layer)
}

// Driven moves one frame per admitted step in the direction a hover reaction
// set, until it passes an end: there it clamps, stops and stops its slot. A
// change of direction stops its slot and requests the direction's sound.
type Driven struct {
	spec    *ActorSpec
	Frame   int
	Dir     int
	Forward bool
}

func (d *Driven) members() []string { return nil }

func (d *Driven) reset(v *View) { d.Frame, d.Dir, d.Forward = 0, 0, false }

func (d *Driven) prepare(v *View) { d.Frame = restFrame(d.spec.Rest, len(v.frames(d.spec.Art))) }

func (d *Driven) enter(v *View, now time.Time) {}
func (d *Driven) leave(v *View)                {}
func (d *Driven) arm(v *View, name string)     {}
func (d *Driven) drive(v *View, dir int)       { d.Dir = dir }
func (d *Driven) activeNow() bool              { return d.Dir != 0 }

func (d *Driven) phase(v *View, phase, name string, now time.Time) {
	count := len(v.frames(d.spec.Art))
	if phase != "advance" || d.Dir == 0 || count == 0 {
		return
	}
	d.Frame += d.Dir
	forward := d.Dir > 0
	if forward != d.Forward {
		v.stop(d.spec.Slot)
		key := d.spec.FirstSound
		if forward {
			key = d.spec.LastSound
		}
		v.sound(d.spec.Slot, key)
		d.Forward = forward
	}
	if d.Frame < 0 || d.Frame >= count {
		if d.Frame < 0 {
			d.Frame = 0
		} else {
			d.Frame = count - 1
		}
		d.Dir = 0
		v.stop(d.spec.Slot)
	}
}

// Shown is the frame the driven actor draws: its rest frame until the view
// is made ready.
func (d *Driven) Shown(v *View) int {
	if !v.ready {
		return restFrame(d.spec.Rest, len(v.frames(d.spec.Art)))
	}
	return d.Frame
}

func (d *Driven) draw(v *View, dst *image.RGBA, name string, layer LayerSpec) {
	put(dst, frameAt(v.frames(d.spec.Art), d.Shown(v)), layer)
}

// Flock is a scheduled group-and-prefix episode: after its wait it picks one
// group and a prefix of that group's members, requests the cue for that
// count, and advances every member one frame per admitted step. When every
// shown member has passed its last frame the flock stays shown for that one
// paint and its next wait starts. Its wait latch outlives the view.
type Flock struct {
	spec       *ActorSpec
	Active     bool
	Terminal   bool
	Group      int
	Count      int
	Progress   []int
	Clock      time.Time
	TerminalAt time.Time
}

func (f *Flock) members() []string { return nil }

func (f *Flock) prepare(v *View) {}

func (f *Flock) reset(v *View) {
	f.Active, f.Terminal, f.Group, f.Count = false, false, 0, 0
	for i := range f.Progress {
		f.Progress[i] = 0
	}
	f.Clock, f.TerminalAt = time.Time{}, time.Time{}
}

func (f *Flock) enter(v *View, now time.Time) {
	f.reset(v)
	f.Clock = now
}

func (f *Flock) leave(v *View)            { f.Active, f.Terminal = false, false }
func (f *Flock) arm(v *View, name string) {}
func (f *Flock) drive(v *View, dir int)   {}
func (f *Flock) activeNow() bool          { return f.Active }

func (f *Flock) family(i int) string { return f.spec.Art + "/" + strconv.Itoa(i) }

func (f *Flock) loaded(v *View) bool {
	for i := 0; i < f.spec.Groups*f.spec.GroupSize; i++ {
		if len(v.frames(f.family(i))) == f.spec.Frames {
			return true
		}
	}
	return false
}

func (f *Flock) phase(v *View, phase, name string, now time.Time) {
	switch phase {
	case "settle":
		if f.Terminal && !now.Equal(f.TerminalAt) {
			f.Terminal, f.Active = false, false
		}
	case "arm":
		if !f.loaded(v) || f.Active {
			return
		}
		w := v.WaitLatch(f.spec.Name)
		if !w.Ready {
			w.Wait, w.Ready = v.wait(*f.spec.Wait), true
		}
		if !elapsed(now.Sub(f.Clock), w.Wait, f.spec.Wait.Compare) {
			return
		}
		f.Group = v.pick(*f.spec.Group)
		f.Count = v.pick(*f.spec.Count)
		for i := range f.Progress {
			f.Progress[i] = 0
		}
		f.Clock, f.Active = now, true
		w.Wait = v.wait(*f.spec.Wait)
		for _, c := range f.spec.CountCues {
			if c.Count == 0 || c.Count == f.Count {
				v.sound(f.spec.Slot, c.Key)
				break
			}
		}
	case "advance":
		if !f.Active {
			return
		}
		for i := range f.Progress {
			f.Progress[i]++
		}
		terminal := f.Count > 0
		for i := 0; i < f.Count && i < len(f.Progress); i++ {
			terminal = terminal && f.Progress[i] >= f.spec.Frames
		}
		if terminal {
			f.Terminal, f.TerminalAt, f.Clock = true, now, now
		}
	}
}

// Shown answers, for each member slot, the family index and frame it shows
// and whether it is drawn.
func (f *Flock) Shown(v *View) []FlockFrame {
	if !f.Active {
		return nil
	}
	var out []FlockFrame
	for i := 0; i < f.Count && i < len(f.Progress); i++ {
		family := f.Group*f.spec.GroupSize + i
		frame := f.Progress[i]
		frames := v.frames(f.family(family))
		visible := frame >= 0 && frame < f.spec.Frames && family >= 0 &&
			family < f.spec.Groups*f.spec.GroupSize && frame < len(frames)
		out = append(out, FlockFrame{Family: family, Frame: frame, Visible: visible})
	}
	return out
}

// FlockFrame is one shown flock member.
type FlockFrame struct {
	Family, Frame int
	Visible       bool
}

func (f *Flock) draw(v *View, dst *image.RGBA, name string, layer LayerSpec) {
	for _, s := range f.Shown(v) {
		if s.Visible {
			put(dst, frameAt(v.frames(f.family(s.Family)), s.Frame), layer)
		}
	}
}

// Families is a group of scheduled members drawn at positions picked on
// entry. An "episode" member starts on a picked sheet when its wait has
// elapsed, advances one frame per admitted step and ends after its sheet's
// last frame. A "loop" member cycles its sheet for as long as the view lives.
// The group's draws come from its own source in the order the description
// lists them.
type Families struct {
	spec    *ActorSpec
	Entered bool
	Members []FamilyMember
}

// FamilyMember is one member's position, sheet, current frame and wait.
type FamilyMember struct {
	Position, Sheet int
	Current         int
	Active          bool
	Clock           time.Time
	Wait            time.Duration
}

func (g *Families) members() []string {
	var out []string
	for _, m := range g.spec.Members {
		out = append(out, m.Name)
	}
	return out
}

func (g *Families) index(name string) int {
	for i, m := range g.spec.Members {
		if m.Name == name {
			return i
		}
	}
	return -1
}

// Member answers the named member.
func (g *Families) Member(name string) *FamilyMember {
	if i := g.index(name); i >= 0 {
		return &g.Members[i]
	}
	return nil
}

func (g *Families) prepare(v *View) {}

func (g *Families) reset(v *View) {
	g.Entered = false
	for i := range g.Members {
		g.Members[i] = FamilyMember{}
	}
}

func (g *Families) enter(v *View, now time.Time) {
	g.reset(v)
	g.Entered = true
	for _, s := range g.spec.Entry {
		switch {
		case s.Position != "":
			m := g.Member(s.Position)
			other := g.Member(s.Unequal)
			for {
				m.Position = v.pick(*s.Pick)
				if other == nil || m.Position != other.Position {
					break
				}
			}
		case s.Wait != "":
			m := g.Member(s.Wait)
			m.Current, m.Clock = -1, now
			m.Wait = v.wait(*s.After)
		}
	}
	for i, m := range g.spec.Members {
		if m.Mode == "loop" {
			g.Members[i].Current, g.Members[i].Active = 0, true
		}
	}
}

func (g *Families) leave(v *View)            {}
func (g *Families) arm(v *View, name string) {}
func (g *Families) drive(v *View, dir int)   {}
func (g *Families) activeNow() bool          { return g.Entered }

func (g *Families) sheetName(i int) string {
	spec, m := g.spec.Members[i], g.Members[i]
	if spec.Sheets > 0 {
		return spec.Art + "/" + strconv.Itoa(m.Position) + "/" + strconv.Itoa(m.Sheet)
	}
	return spec.Art + "/" + strconv.Itoa(m.Position)
}

// sheetLen is the loaded length of a member's sheet, or its declared length
// when the sheet did not load.
func (g *Families) sheetLen(v *View, i int) int {
	if n := len(v.frames(g.sheetName(i))); n > 0 {
		return n
	}
	spec := g.spec.Members[i]
	for _, a := range v.desc.Art {
		if a.Name == spec.Art {
			return a.countAt(g.Members[i].Sheet)
		}
	}
	return 0
}

func (g *Families) phase(v *View, phase, name string, now time.Time) {
	if !g.Entered {
		return
	}
	switch phase {
	case "advance":
		for _, n := range g.spec.StepOrder {
			i := g.index(n)
			if i < 0 || !g.Members[i].Active {
				continue
			}
			m := &g.Members[i]
			if g.spec.Members[i].Mode == "loop" {
				if frames := v.frames(g.sheetName(i)); len(frames) == 0 {
					m.Current = -1
				} else {
					m.Current = (m.Current + 1) % len(frames)
				}
				continue
			}
			m.Current++
			m.Clock = now
			if m.Current >= g.sheetLen(v, i) {
				m.Current, m.Active = -1, false
			}
		}
	case "schedule":
		for _, n := range g.spec.PaintOrder {
			i := g.index(n)
			if i < 0 {
				continue
			}
			spec, m := g.spec.Members[i], &g.Members[i]
			if !elapsed(now.Sub(m.Clock), m.Wait, spec.Later.Compare) {
				continue
			}
			m.Clock, m.Current, m.Active = now, 0, true
			m.Wait = v.wait(*spec.Later)
			m.Sheet = v.pick(*spec.SheetPick)
		}
		for i, spec := range g.spec.Members {
			if len(spec.FrameCues) == 0 || len(v.frames(g.sheetName(i))) == 0 {
				continue
			}
			m := g.Members[i]
			for _, c := range spec.FrameCues {
				if (c.Sheet < 0 || c.Sheet == m.Sheet) && c.Frame == m.Current {
					v.sound(c.Slot, c.Key)
					break
				}
			}
		}
	}
}

func (g *Families) draw(v *View, dst *image.RGBA, name string, layer LayerSpec) {
	i := g.index(name)
	if i < 0 || !g.Entered || !v.present {
		return
	}
	spec, m := g.spec.Members[i], g.Members[i]
	if m.Position < 0 || m.Position >= len(spec.Positions) {
		return
	}
	if spec.Sheets > 0 && (m.Sheet < 0 || m.Sheet >= spec.Sheets) {
		return
	}
	at := layer
	at.At = spec.Positions[m.Position]
	put(dst, frameAt(v.frames(g.sheetName(i)), max(m.Current, 0)), at)
}
