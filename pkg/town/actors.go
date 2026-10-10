package town

import (
	"image"
	"strconv"
	"time"
)

// actor is one program's runtime. reset returns it to a scene never entered
// and prepare to a scene just made ready; enter runs on each entry and leave
// on each hiding of a shown scene; resume runs when a paused scene runs
// again; phase runs one named step; arm and drive are hover reactions; event
// runs one bound action and answers whether it changed state; activeNow is a
// layer condition; draw paints the layer's actor or member.
type actor interface {
	members() []string
	reset(s *Scene)
	prepare(s *Scene)
	enter(s *Scene, now time.Time)
	leave(s *Scene)
	resume(s *Scene, now time.Time)
	phase(s *Scene, phase, name string, now time.Time)
	arm(s *Scene, name string)
	drive(s *Scene, dir int)
	event(s *Scene, do string, args []int) bool
	activeNow() bool
	draw(s *Scene, dst *image.RGBA, layer LayerSpec)
}

// still is a program's answer to every call it takes no part in.
type still struct{}

func (still) members() []string                          { return nil }
func (still) reset(s *Scene)                             {}
func (still) prepare(s *Scene)                           {}
func (still) enter(s *Scene, now time.Time)              {}
func (still) leave(s *Scene)                             {}
func (still) resume(s *Scene, now time.Time)             {}
func (still) arm(s *Scene, name string)                  {}
func (still) drive(s *Scene, dir int)                    {}
func (still) event(s *Scene, do string, args []int) bool { return false }
func (still) activeNow() bool                            { return false }

// programs are the program names an actor may run, in either scene.
var programs = []string{"episode", "pendulum", "stepper", "driven", "flock", "families",
	"loop", "alternating", "selector", "priority", "bounce", "cycle", "target", "training"}

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
	case "loop":
		return &Loop{spec: spec}
	case "alternating":
		return &Alternating{spec: spec}
	case "selector":
		return &Selector{spec: spec}
	case "priority":
		return &Priority{spec: spec}
	case "bounce":
		return &Bounce{spec: spec}
	case "cycle":
		return &Cycle{spec: spec}
	case "target":
		return &Target{spec: spec}
	case "training":
		return &Training{spec: spec}
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
// count that outlives the scene.
type Episode struct {
	still
	spec    *ActorSpec
	Frame   int
	Enabled bool
	Visible bool
}

func (e *Episode) reset(s *Scene) { e.Frame, e.Enabled, e.Visible = 0, false, true }

func (e *Episode) enter(s *Scene, now time.Time) {
	if e.spec.ResetOnEntry {
		e.Enabled, e.Frame, e.Visible = false, 0, true
	}
}

func (e *Episode) leave(s *Scene) {
	if e.spec.ResetOnEntry {
		e.Enabled, e.Visible = false, false
	}
}

func (e *Episode) arm(s *Scene, name string) { e.Enabled = true }
func (e *Episode) activeNow() bool           { return e.Enabled }

func (e *Episode) cue(s *Scene) {
	c := e.spec.StartSound
	if c == nil {
		return
	}
	for _, slot := range c.Stop {
		s.stop(slot)
	}
	s.sound(c.Slot, c.Key)
}

func (e *Episode) phase(s *Scene, phase, name string, now time.Time) {
	switch phase {
	case "trigger":
		if e.spec.Trigger != nil && s.chance(*e.spec.Trigger) {
			e.Enabled = true
		}
	case "advance":
		if e.spec.End == "hold" {
			e.advanceHeld(s)
			return
		}
		count := len(s.frames(e.spec.Art))
		if e.Enabled && count > 0 && e.Frame == 0 {
			e.cue(s)
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

func (e *Episode) loaded(s *Scene) bool { return len(s.frames(e.spec.Art)) == e.spec.Frames }

func (e *Episode) advanceHeld(s *Scene) {
	if !e.Enabled {
		return
	}
	if e.Frame == 0 {
		e.cue(s)
	}
	e.Frame++
	if e.Frame < e.spec.Frames {
		e.Visible = e.loaded(s)
		return
	}
	n := s.proc.ends[e.spec.Name] + 1
	if n >= e.spec.RewindEvery {
		e.Frame, n = 0, 0
	}
	s.proc.ends[e.spec.Name] = n
	e.Visible, e.Enabled = false, false
}

// Shown reports whether the episode's current frame is drawn.
func (e *Episode) Shown(s *Scene) bool {
	if e.spec.End == "hold" {
		return e.Visible && e.loaded(s)
	}
	return true
}

func (e *Episode) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	if !e.Shown(s) {
		return
	}
	place(dst, frameAt(s.frames(e.spec.Art), e.Frame), layer)
}

// Pendulum is a group of members swinging between their first and last frames
// under one shared enable. At either end a member rests until a chance holds;
// a member back at its first frame from a backward swing clears the shared
// enable, and the members after it still take their step that phase.
type Pendulum struct {
	still
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

func (p *Pendulum) reset(s *Scene) {
	p.Enabled = false
	for i := range p.Members {
		p.Members[i] = Swing{}
	}
}

func (p *Pendulum) arm(s *Scene, name string) { p.Enabled = true }
func (p *Pendulum) activeNow() bool           { return p.Enabled }

func (p *Pendulum) phase(s *Scene, phase, name string, now time.Time) {
	if phase != "advance" || !p.Enabled {
		return
	}
	for i, m := range p.spec.Members {
		p.swing(s, &p.Members[i], len(s.frames(m.Art)))
	}
}

func (p *Pendulum) swing(s *Scene, w *Swing, count int) {
	if count == 0 {
		return
	}
	if w.Frame == 0 && w.Step == -1 {
		w.Step = 0
		p.Enabled = false
		return
	}
	if w.Frame == 0 && w.Step == 0 {
		if s.chance(*p.spec.Chance) {
			w.Step = 1
		}
	} else if w.Frame == count-1 {
		w.Step = 0
		if s.chance(*p.spec.Chance) {
			w.Step = -1
		}
	}
	w.Frame += w.Step
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

func (p *Pendulum) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	for i, m := range p.spec.Members {
		if m.Name == layer.Actor {
			place(dst, frameAt(s.frames(m.Art), p.Members[i].Frame), layer)
		}
	}
}

// Stepper moves one frame per admitted step toward its first frame while its
// hotspot is hovered and toward its last frame otherwise. A change of target
// stops its slot and requests the target's sound. While its hold condition is
// false it shows its hold frame and keeps its target latch.
type Stepper struct {
	still
	spec        *ActorSpec
	Frame       int
	TowardFirst bool
}

func (st *Stepper) reset(s *Scene) { st.Frame, st.TowardFirst = 0, false }

func (st *Stepper) prepare(s *Scene) { st.Frame = restFrame(st.spec.Rest, len(s.frames(st.spec.Art))) }

func (st *Stepper) phase(s *Scene, phase, name string, now time.Time) {
	count := len(s.frames(st.spec.Art))
	if phase != "advance" || count == 0 {
		return
	}
	if st.spec.HoldUnless != "" && !s.host.Condition(st.spec.HoldUnless) {
		st.Frame = restFrame(st.spec.Hold, count)
		return
	}
	first := s.hover != nil && s.hover.Name == st.spec.TowardFirstOnHover
	if first != st.TowardFirst {
		s.stop(st.spec.Slot)
		key := st.spec.LastSound
		if first {
			key = st.spec.FirstSound
		}
		s.sound(st.spec.Slot, key)
		st.TowardFirst = first
	}
	if first && st.Frame > 0 {
		st.Frame--
	}
	if !first && st.Frame < count-1 {
		st.Frame++
	}
}

// Shown is the frame the stepper draws: its rest frame until the scene is
// made ready.
func (st *Stepper) Shown(s *Scene) int {
	if !s.ready {
		return restFrame(st.spec.Rest, len(s.frames(st.spec.Art)))
	}
	return st.Frame
}

func (st *Stepper) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	place(dst, frameAt(s.frames(st.spec.Art), st.Shown(s)), layer)
}

// Driven moves one frame per admitted step in the direction a hover reaction
// set, until it passes an end: there it clamps, stops and stops its slot. A
// change of direction stops its slot and requests the direction's sound.
type Driven struct {
	still
	spec    *ActorSpec
	Frame   int
	Dir     int
	Forward bool
}

func (d *Driven) reset(s *Scene) { d.Frame, d.Dir, d.Forward = 0, 0, false }

func (d *Driven) prepare(s *Scene) { d.Frame = restFrame(d.spec.Rest, len(s.frames(d.spec.Art))) }

func (d *Driven) drive(s *Scene, dir int) { d.Dir = dir }
func (d *Driven) activeNow() bool         { return d.Dir != 0 }

func (d *Driven) phase(s *Scene, phase, name string, now time.Time) {
	count := len(s.frames(d.spec.Art))
	if phase != "advance" || d.Dir == 0 || count == 0 {
		return
	}
	d.Frame += d.Dir
	forward := d.Dir > 0
	if forward != d.Forward {
		s.stop(d.spec.Slot)
		key := d.spec.FirstSound
		if forward {
			key = d.spec.LastSound
		}
		s.sound(d.spec.Slot, key)
		d.Forward = forward
	}
	if d.Frame < 0 || d.Frame >= count {
		if d.Frame < 0 {
			d.Frame = 0
		} else {
			d.Frame = count - 1
		}
		d.Dir = 0
		s.stop(d.spec.Slot)
	}
}

// Shown is the frame the driven actor draws: its rest frame until the scene
// is made ready.
func (d *Driven) Shown(s *Scene) int {
	if !s.ready {
		return restFrame(d.spec.Rest, len(s.frames(d.spec.Art)))
	}
	return d.Frame
}

func (d *Driven) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	place(dst, frameAt(s.frames(d.spec.Art), d.Shown(s)), layer)
}

// Flock is a scheduled group-and-prefix episode: after its wait it picks one
// group and a prefix of that group's members, requests the cue for that
// count, and advances every member one frame per admitted step. When every
// shown member has passed its last frame the flock stays shown for that one
// paint and its next wait starts. Its wait latch outlives the scene.
type Flock struct {
	still
	spec       *ActorSpec
	Active     bool
	Terminal   bool
	Group      int
	Count      int
	Progress   []int
	Clock      time.Time
	TerminalAt time.Time
}

func (f *Flock) reset(s *Scene) {
	f.Active, f.Terminal, f.Group, f.Count = false, false, 0, 0
	for i := range f.Progress {
		f.Progress[i] = 0
	}
	f.Clock, f.TerminalAt = time.Time{}, time.Time{}
}

func (f *Flock) enter(s *Scene, now time.Time) {
	f.reset(s)
	f.Clock = now
}

func (f *Flock) leave(s *Scene)  { f.Active, f.Terminal = false, false }
func (f *Flock) activeNow() bool { return f.Active }

func (f *Flock) family(i int) string { return f.spec.Art + "/" + strconv.Itoa(i) }

func (f *Flock) loaded(s *Scene) bool {
	for i := 0; i < f.spec.Groups*f.spec.GroupSize; i++ {
		if len(s.frames(f.family(i))) == f.spec.Frames {
			return true
		}
	}
	return false
}

func (f *Flock) phase(s *Scene, phase, name string, now time.Time) {
	switch phase {
	case "settle":
		if f.Terminal && !now.Equal(f.TerminalAt) {
			f.Terminal, f.Active = false, false
		}
	case "arm":
		if !f.loaded(s) || f.Active {
			return
		}
		w := s.WaitLatch(f.spec.Name)
		if !w.Ready {
			w.Wait, w.Ready = s.wait(*f.spec.Wait), true
		}
		if !elapsed(now.Sub(f.Clock), w.Wait, f.spec.Wait.Compare) {
			return
		}
		f.Group = s.amount(*f.spec.Group)
		f.Count = s.amount(*f.spec.Count)
		for i := range f.Progress {
			f.Progress[i] = 0
		}
		f.Clock, f.Active = now, true
		w.Wait = s.wait(*f.spec.Wait)
		for _, c := range f.spec.CountCues {
			if c.Count == 0 || c.Count == f.Count {
				s.sound(f.spec.Slot, c.Key)
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
func (f *Flock) Shown(s *Scene) []FlockFrame {
	if !f.Active {
		return nil
	}
	var out []FlockFrame
	for i := 0; i < f.Count && i < len(f.Progress); i++ {
		family := f.Group*f.spec.GroupSize + i
		frame := f.Progress[i]
		frames := s.frames(f.family(family))
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

func (f *Flock) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	for _, sh := range f.Shown(s) {
		if sh.Visible {
			place(dst, frameAt(s.frames(f.family(sh.Family)), sh.Frame), layer)
		}
	}
}

// Families is a group of scheduled members drawn at positions picked on
// entry. An "episode" member starts on a picked sheet when its wait has
// elapsed, advances one frame per admitted step and ends after its sheet's
// last frame. A "loop" member cycles its sheet for as long as the scene
// lives. The group's draws come from its own source in the order the
// description lists them.
type Families struct {
	still
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

func (g *Families) reset(s *Scene) {
	g.Entered = false
	for i := range g.Members {
		g.Members[i] = FamilyMember{}
	}
}

func (g *Families) enter(s *Scene, now time.Time) {
	g.reset(s)
	g.Entered = true
	for _, e := range g.spec.Entry {
		switch {
		case e.Position != "":
			m := g.Member(e.Position)
			other := g.Member(e.Unequal)
			for {
				m.Position = s.amount(*e.Pick)
				if other == nil || m.Position != other.Position {
					break
				}
			}
		case e.Wait != "":
			m := g.Member(e.Wait)
			m.Current, m.Clock = -1, now
			m.Wait = s.wait(*e.After)
		}
	}
	for i, m := range g.spec.Members {
		if m.Mode == "loop" {
			g.Members[i].Current, g.Members[i].Active = 0, true
		}
	}
}

func (g *Families) activeNow() bool { return g.Entered }

func (g *Families) sheetName(i int) string {
	spec, m := g.spec.Members[i], g.Members[i]
	if spec.Sheets > 0 {
		return spec.Art + "/" + strconv.Itoa(m.Position) + "/" + strconv.Itoa(m.Sheet)
	}
	return spec.Art + "/" + strconv.Itoa(m.Position)
}

// sheetLen is the loaded length of a member's sheet, or its declared length
// when the sheet did not load.
func (g *Families) sheetLen(s *Scene, i int) int {
	if n := len(s.frames(g.sheetName(i))); n > 0 {
		return n
	}
	spec := g.spec.Members[i]
	for _, a := range s.spec.Art {
		if a.Name == spec.Art {
			return a.countAt(g.Members[i].Sheet)
		}
	}
	return 0
}

func (g *Families) phase(s *Scene, phase, name string, now time.Time) {
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
				if frames := s.frames(g.sheetName(i)); len(frames) == 0 {
					m.Current = -1
				} else {
					m.Current = (m.Current + 1) % len(frames)
				}
				continue
			}
			m.Current++
			m.Clock = now
			if m.Current >= g.sheetLen(s, i) {
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
			m.Wait = s.wait(*spec.Later)
			m.Sheet = s.amount(*spec.SheetPick)
		}
		for i, spec := range g.spec.Members {
			if len(spec.FrameCues) == 0 || len(s.frames(g.sheetName(i))) == 0 {
				continue
			}
			m := g.Members[i]
			for _, c := range spec.FrameCues {
				if (c.Sheet < 0 || c.Sheet == m.Sheet) && c.Frame == m.Current {
					s.sound(c.Slot, c.Key)
					break
				}
			}
		}
	}
}

func (g *Families) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	i := g.index(layer.Actor)
	if i < 0 || !g.Entered || !s.present {
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
	place(dst, frameAt(s.frames(g.sheetName(i)), max(m.Current, 0)), at)
}
