package town

import (
	"image"
	"time"
)

// pageActor is one room-scene program's runtime. reset returns it to a page
// never entered and enter to a page just entered; resume runs when a paused
// page runs again; phase runs one named step; event runs one bound action and
// answers whether it changed state; draw paints the layer's member.
type pageActor interface {
	members() []string
	reset(p *Page)
	enter(p *Page, now time.Time)
	resume(p *Page, now time.Time)
	phase(p *Page, phase string, now time.Time)
	event(p *Page, do string, args []int) bool
	draw(p *Page, dst *image.RGBA, layer LayerSpec)
}

func newPageActor(spec *ActorSpec) pageActor {
	switch spec.Program {
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

// pagePrograms are the program names a room scene may use.
var pagePrograms = []string{"loop", "alternating", "selector", "priority", "bounce", "cycle", "target", "training"}

// Loop cycles the first Loop frames of a complete art entry, one frame each
// time its step runs. Shown is the frame published for paint.
type Loop struct {
	spec         *ActorSpec
	Index, Shown int
}

func (l *Loop) members() []string                      { return nil }
func (l *Loop) reset(p *Page)                          { l.Index, l.Shown = 0, 0 }
func (l *Loop) enter(p *Page, now time.Time)           { l.reset(p) }
func (l *Loop) resume(p *Page, now time.Time)          {}
func (l *Loop) event(p *Page, do string, a []int) bool { return false }

func (l *Loop) phase(p *Page, phase string, now time.Time) {
	switch phase {
	case "publish":
		l.Shown = l.Index
	case "advance":
		if p.Complete(l.spec.Art, l.spec.Frames) {
			l.Index = (l.Index + 1) % l.spec.Loop
		}
	}
}

func (l *Loop) draw(p *Page, dst *image.RGBA, layer LayerSpec) {
	if p.ready {
		place(dst, frameAt(p.frames(l.spec.Art), l.Shown), layer)
	}
}

// Alternating rests for a drawn wait, then starts the state whose parity
// matches the wait's millisecond count when that state's art is complete;
// otherwise it draws a new wait. A started state steps on its clock's period:
// "forward" to its last frame, after which the index returns to zero while
// the shown frame stays; "ping-pong" forward then back to the first frame.
// Each state keeps its own index and the last frame it showed.
type Alternating struct {
	spec       *ActorSpec
	State      int // 0 resting, otherwise the 1-based running state
	Index      []int
	Cached     []int
	Direction  int
	Delay      time.Duration
	ShownState int
	ShownFrame int
}

func (a *Alternating) members() []string { return nil }

func (a *Alternating) reset(p *Page) {
	n := len(a.spec.States)
	a.State, a.Direction, a.Delay, a.ShownState, a.ShownFrame = 0, 0, 0, 0, 0
	a.Index, a.Cached = make([]int, n), make([]int, n)
}

func (a *Alternating) enter(p *Page, now time.Time) {
	a.reset(p)
	a.Delay = p.wait(*a.spec.Delay)
}

func (a *Alternating) resume(p *Page, now time.Time)          {}
func (a *Alternating) event(p *Page, do string, x []int) bool { return false }

func (a *Alternating) finish(p *Page, now time.Time, clearDirection bool) {
	a.State = 0
	if clearDirection {
		a.Direction = 0
	}
	a.Delay = p.wait(*a.spec.Delay)
	p.SetClock(a.spec.Clock, now)
}

func (a *Alternating) phase(p *Page, phase string, now time.Time) {
	switch phase {
	case "arm":
		if now.Sub(p.Clock(a.spec.Clock)) <= a.Delay {
			return
		}
		parity := "even"
		if a.Delay/time.Millisecond%2 != 0 {
			parity = "odd"
		}
		for i, s := range a.spec.States {
			if s.When != parity {
				continue
			}
			if p.Complete(s.Art, s.Complete) {
				a.State = i + 1
				if s.Motion == "ping-pong" {
					a.Direction = 1
				}
				for _, slot := range s.StartSounds {
					p.sound(slot)
				}
				return
			}
			break
		}
		// A chosen state whose art is missing does not trap the actor: it
		// draws a new wait and stays as it is.
		p.SetClock(a.spec.Clock, now)
		a.Delay = p.wait(*a.spec.Delay)
	case "publish":
		a.ShownState = a.State
		if a.State > 0 {
			a.ShownFrame = a.Cached[a.State-1]
		}
	case "cue":
		if a.State == 0 {
			return
		}
		for _, c := range a.spec.States[a.State-1].FrameSounds {
			if a.Index[a.State-1] == c.Frame {
				p.sound(c.Slot)
			}
		}
	case "advance":
		a.advance(p, now)
	}
}

func (a *Alternating) advance(p *Page, now time.Time) {
	if a.State == 0 || now.Sub(p.Clock(a.spec.Clock)) <= p.period(a.spec.Clock) {
		return
	}
	p.SetClock(a.spec.Clock, now)
	i := a.State - 1
	s := a.spec.States[i]
	frames := p.frames(s.Art)
	if s.Motion == "forward" {
		if !stepFrame(&a.Index[i], &a.Cached[i], frames, 1) {
			a.Index[i] = 0
			a.end(p, now, s, false)
		}
		return
	}
	if a.Direction > 0 {
		if stepFrame(&a.Index[i], &a.Cached[i], frames, 1) {
			return
		}
		a.Direction = -1
		if stepFrame(&a.Index[i], &a.Cached[i], frames, -1) {
			return
		}
	} else if stepFrame(&a.Index[i], &a.Cached[i], frames, -1) {
		return
	}
	a.end(p, now, s, true)
}

func (a *Alternating) end(p *Page, now time.Time, s StateSpec, clearDirection bool) {
	for _, slot := range s.EndSounds {
		p.sound(slot)
	}
	a.finish(p, now, clearDirection)
}

// stepFrame moves index one frame in dir when the frame there exists and
// loaded, keeping cached on the frame shown.
func stepFrame(index, cached *int, frames []image.Image, dir int) bool {
	next := *index + dir
	if *index < 0 || *index >= len(frames) || next < 0 || next >= len(frames) || frames[next] == nil {
		return false
	}
	*index, *cached = next, next
	return true
}

func (a *Alternating) draw(p *Page, dst *image.RGBA, layer LayerSpec) {
	if p.ready && a.ShownState > 0 {
		place(dst, frameAt(p.frames(a.spec.States[a.ShownState-1].Art), a.ShownFrame), layer)
	}
}

// Selector is a group of members of which one is selected. Entry selects the
// member a host value names. Selecting another member starts it from its
// first frame and sends the previously selected running member to its release
// frame; a change raises the actor's event. Each step moves every running
// member one frame: LoopAt returns to LoopTo and StopAt stops it. A member
// whose art is incomplete shows its first frame always; before entry every
// member shows its first frame.
type Selector struct {
	spec     *ActorSpec
	Selected int
	Enabled  []bool
	Index    []int
}

func (s *Selector) members() []string {
	var out []string
	for _, m := range s.spec.Members {
		out = append(out, m.Name)
	}
	return out
}

func (s *Selector) reset(p *Page) {
	s.Selected = 0
	s.Enabled, s.Index = make([]bool, len(s.spec.Members)), make([]int, len(s.spec.Members))
}

func (s *Selector) enter(p *Page, now time.Time) {
	s.reset(p)
	s.Selected = p.host.Value(s.spec.Selected)
	if s.Selected >= 0 && s.Selected < len(s.Enabled) {
		s.Enabled[s.Selected] = true
	}
}

func (s *Selector) resume(p *Page, now time.Time) {}

func (s *Selector) event(p *Page, do string, args []int) bool {
	if do != "select" || !p.ready || len(args) == 0 {
		return false
	}
	m := args[0]
	if m < 0 || m >= len(s.Enabled) || m == s.Selected {
		return false
	}
	if old := s.Selected; old >= 0 && old < len(s.Enabled) && s.Enabled[old] {
		s.Index[old] = s.spec.Release
	}
	s.Selected = m
	s.Enabled[m], s.Index[m] = true, 0
	if s.spec.Raise != "" {
		p.Event(s.spec.Raise)
	}
	return true
}

func (s *Selector) phase(p *Page, phase string, now time.Time) {
	if phase != "advance" {
		return
	}
	for i := range s.Enabled {
		if !s.Enabled[i] {
			continue
		}
		switch next := s.Index[i] + 1; next {
		case s.spec.LoopAt:
			s.Index[i] = s.spec.LoopTo
		case s.spec.StopAt:
			s.Enabled[i], s.Index[i] = false, 0
		default:
			s.Index[i] = next
		}
	}
}

func (s *Selector) draw(p *Page, dst *image.RGBA, layer LayerSpec) {
	for i, m := range s.spec.Members {
		if m.Name != layer.Actor {
			continue
		}
		frames := p.frames(m.Art)
		if !p.ready || !p.Complete(m.Art, s.spec.Frames) {
			place(dst, frameAt(frames, 0), layer)
			continue
		}
		if s.Enabled[i] {
			place(dst, frameAt(frames, s.Index[i]), layer)
		}
	}
}

// Priority runs at most one of its states at a time, the first raised in
// list order. Its first state is raised after a wait drawn on every step;
// the others by events. A running state advances one index per step and ends
// at its Steps, restamping the actor's clock. Index i above zero shows frame
// i-1 of a complete state; otherwise the actor shows its base art.
type Priority struct {
	spec   *ActorSpec
	Raised []bool
	Index  int
}

func (q *Priority) members() []string { return nil }

func (q *Priority) reset(p *Page) {
	q.Raised, q.Index = make([]bool, len(q.spec.States)), 0
}

func (q *Priority) enter(p *Page, now time.Time)  { q.reset(p) }
func (q *Priority) resume(p *Page, now time.Time) {}

func (q *Priority) event(p *Page, do string, args []int) bool {
	if !p.ready {
		return false
	}
	for i, s := range q.spec.States {
		if s.Name == do {
			q.Raised[i] = true
			return true
		}
	}
	return false
}

// Running answers the index of the running state, -1 for none.
func (q *Priority) Running() int {
	for i, r := range q.Raised {
		if r {
			return i
		}
	}
	return -1
}

func (q *Priority) phase(p *Page, phase string, now time.Time) {
	switch phase {
	case "arm":
		wait := p.wait(*q.spec.Delay)
		if q.Running() < 0 && elapsed(now.Sub(p.Clock(q.spec.Clock)), wait, p.compare(q.spec.Clock)) {
			q.Raised[0] = true
		}
	case "advance":
		i := q.Running()
		if i < 0 {
			return
		}
		if next := q.Index + 1; next < q.spec.States[i].Steps {
			q.Index = next
			return
		}
		q.Raised[i], q.Index = false, 0
		p.SetClock(q.spec.Clock, now)
	}
}

func (q *Priority) draw(p *Page, dst *image.RGBA, layer LayerSpec) {
	pic := frameAt(p.frames(q.spec.Art), 0)
	if i := q.Running(); p.ready && i >= 0 {
		s := q.spec.States[i]
		if q.Index > 0 && q.Index < s.Steps && p.Complete(s.Art, s.Complete) {
			if f := frameAt(p.frames(s.Art), q.Index-1); f != nil {
				pic = f
			}
		}
	}
	place(dst, pic, layer)
}

// Bounce runs from its first frame to its last and back once each time it is
// armed, one frame per step. It is shown while running, from its first step.
type Bounce struct {
	spec        *ActorSpec
	Frame, Step int
	Stepped     bool
}

func (b *Bounce) members() []string             { return nil }
func (b *Bounce) reset(p *Page)                 { b.Frame, b.Step, b.Stepped = 0, 0, false }
func (b *Bounce) enter(p *Page, now time.Time)  { b.reset(p) }
func (b *Bounce) resume(p *Page, now time.Time) {}

func (b *Bounce) event(p *Page, do string, args []int) bool {
	if do != "arm" {
		return false
	}
	b.Step = 1
	return true
}

func (b *Bounce) phase(p *Page, phase string, now time.Time) {
	if phase != "advance" || b.Step == 0 {
		return
	}
	last := b.spec.Frames - 1
	b.Frame += b.Step
	if b.Frame >= last {
		b.Frame, b.Step = last, -1
	} else if b.Frame == 0 {
		b.Step = 0
	}
	b.Stepped = true
}

// Shown reports whether the bounce is drawn.
func (b *Bounce) Shown() bool { return b.Stepped && b.Step != 0 }

func (b *Bounce) draw(p *Page, dst *image.RGBA, layer LayerSpec) {
	if b.Shown() {
		place(dst, frameAt(p.frames(b.spec.Art), b.Frame), layer)
	}
}

// Cycle counts modulo Frames, one count each time its wait has elapsed since
// its last count. Its first step, and a step that reads a time before its
// stamp, only stamps it. Its state outlives every page entry. Order maps the
// count to a slot per variant.
type Cycle struct {
	spec    *ActorSpec
	Index   int
	Last    time.Time
	Stamped bool
}

func (c *Cycle) members() []string                          { return nil }
func (c *Cycle) reset(p *Page)                              {}
func (c *Cycle) enter(p *Page, now time.Time)               {}
func (c *Cycle) resume(p *Page, now time.Time)              { c.Last = now }
func (c *Cycle) event(p *Page, do string, args []int) bool  { return false }
func (c *Cycle) draw(p *Page, dst *image.RGBA, l LayerSpec) {}

func (c *Cycle) phase(p *Page, phase string, now time.Time) {
	if phase != "advance" {
		return
	}
	if !c.Stamped || now.Before(c.Last) {
		c.Stamped, c.Last = true, now
		return
	}
	if elapsed(now.Sub(c.Last), time.Duration(c.spec.Wait.BaseMS)*time.Millisecond, c.spec.Wait.Compare) {
		c.Index = (c.Index + 1) % c.spec.Frames
		c.Last = now
	}
}

// Slot answers the count's slot for a variant; a variant outside Order
// answers the count itself.
func (c *Cycle) Slot(variant int) int {
	if variant < 0 || variant >= len(c.spec.Order) || c.Index >= len(c.spec.Order[variant]) {
		return c.Index
	}
	return c.spec.Order[variant][c.Index]
}

// Target walks one frame per step toward the endpoint of its variant. Entry
// rests it on the endpoint of the variant a host value names. On its own it
// steps only when its wait has elapsed since its last retarget or step; a
// training actor may step it directly.
type Target struct {
	spec        *ActorSpec
	Frame, Goal int
	Ready       bool
	Last        time.Time
}

func (t *Target) members() []string                          { return nil }
func (t *Target) reset(p *Page)                              { t.Frame, t.Goal, t.Ready, t.Last = 0, 0, false, time.Time{} }
func (t *Target) resume(p *Page, now time.Time)              {}
func (t *Target) event(p *Page, do string, args []int) bool  { return false }
func (t *Target) phase(p *Page, phase string, now time.Time) {}

func (t *Target) enter(p *Page, now time.Time) {
	t.reset(p)
	if v := p.host.Value(t.spec.Value); v >= 0 {
		t.Rest(v)
	}
}

// Endpoint answers a variant's endpoint frame.
func (t *Target) Endpoint(variant int) int {
	if variant >= 0 && variant < len(t.spec.Endpoints) {
		return t.spec.Endpoints[variant]
	}
	return 0
}

// Rest places the target on a variant's endpoint.
func (t *Target) Rest(variant int) {
	e := t.Endpoint(variant)
	t.Frame, t.Goal, t.Ready, t.Last = e, e, true, time.Time{}
}

// Retarget aims at a variant's endpoint and answers whether the target must
// move to reach it.
func (t *Target) Retarget(variant int, now time.Time) bool {
	goal := t.Endpoint(variant)
	if t.Goal == goal {
		return false
	}
	t.Goal, t.Last = goal, now
	return t.Frame != goal
}

// Step moves one frame toward the goal.
func (t *Target) Step() {
	if !t.Ready || t.Frame == t.Goal {
		return
	}
	if t.Frame < t.Goal {
		t.Frame++
	} else {
		t.Frame--
	}
}

func (t *Target) advance(now time.Time) {
	if !t.Ready || t.Frame == t.Goal || !elapsed(now.Sub(t.Last), time.Duration(t.spec.Wait.BaseMS)*time.Millisecond, t.spec.Wait.Compare) {
		return
	}
	t.Step()
	t.Last = now
}

func (t *Target) draw(p *Page, dst *image.RGBA, layer LayerSpec) {
	if t.Ready {
		place(dst, frameAt(p.frames(t.spec.Art), t.Frame), layer)
	}
}

// Training is a pair of variant sides, each with a transition sequence and
// an idle sequence that has priority over it. Entry arms the transition of
// the variant a host value names. A variant change arms that variant's
// transition, waits for it, and walks the column actor to the new endpoint
// from the transition's ColumnAt index; without a complete transition the
// column walks on its own. Sides step together on the clock's period. An idle
// sequence starts after its wait, plays forward, holds for a drawn count,
// then plays back. Its timers, waits and holds outlive every page entry.
type Training struct {
	spec          *ActorSpec
	WaitVariant   int
	ColumnStarted bool
	Sides         []TrainingSide
	Static        []TrainingStatic
}

// TrainingSide is one variant's page-scoped sequence state.
type TrainingSide struct {
	TransitionActive bool
	TransitionIndex  int
	IdleActive       bool
	IdleIndex        int
	IdleCached       int
	IdleDirection    int
}

// TrainingStatic is one variant's idle timer, wait and hold, which outlive
// the page.
type TrainingStatic struct {
	Initialized bool
	IdleLast    time.Time
	IdleExtra   time.Duration
	Hold        int
	HoldLimit   int
}

// NoVariant is the wait variant of a training actor no change waits for.
const NoVariant = -1

func (t *Training) members() []string {
	var out []string
	for _, v := range t.spec.Variants {
		out = append(out, v.Name)
	}
	return out
}

func (t *Training) reset(p *Page) {
	t.WaitVariant, t.ColumnStarted = 0, false
	t.Sides = make([]TrainingSide, len(t.spec.Variants))
	if len(t.Static) != len(t.spec.Variants) {
		t.Static = make([]TrainingStatic, len(t.spec.Variants))
	}
}

func (t *Training) column(p *Page) *Target {
	c, _ := p.byName[t.spec.Column].(*Target)
	return c
}

func (t *Training) enter(p *Page, now time.Time) {
	t.reset(p)
	t.WaitVariant = NoVariant
	if v := p.host.Value(t.spec.Value); v >= 0 && v < len(t.Sides) {
		t.arm(p, v, now, false)
	}
}

func (t *Training) resume(p *Page, now time.Time) {
	for i := range t.Static {
		if t.Static[i].Initialized {
			t.Static[i].IdleLast = now
		}
	}
}

// Busy reports whether the page waits for a variant change to finish.
func (t *Training) Busy(p *Page) bool { return p.ready && t.WaitVariant != NoVariant }

func (t *Training) transitionComplete(p *Page, v int) bool {
	s := t.spec.Variants[v]
	return p.Complete(s.Transition, s.TransitionCount)
}

func (t *Training) idleComplete(p *Page, v int) bool {
	s := t.spec.Variants[v]
	return p.Complete(s.Idle, s.IdleCount)
}

func (t *Training) arm(p *Page, v int, now time.Time, changed bool) bool {
	if v < 0 || v >= len(t.Sides) || !t.transitionComplete(p, v) {
		return false
	}
	side := &t.Sides[v]
	side.TransitionActive, side.TransitionIndex = true, 0
	if changed {
		t.WaitVariant = v
	}
	col := t.column(p)
	if !col.Ready {
		col.Rest(v)
	}
	if changed {
		col.Retarget(v, now)
	}
	if !p.Complete(col.spec.Art, col.spec.Frames) {
		col.Rest(v)
	}
	t.ColumnStarted = col.Frame == col.Endpoint(v)
	return true
}

func (t *Training) event(p *Page, do string, args []int) bool {
	if do != "change" || len(args) < 2 {
		return false
	}
	from, to := args[0], args[1]
	if from == to {
		return false
	}
	col := t.column(p)
	if !col.Ready {
		col.Rest(from)
	}
	now := p.host.Now()
	if t.arm(p, to, now, true) {
		return true
	}
	if !p.Complete(col.spec.Art, col.spec.Frames) {
		col.Rest(to)
		return true
	}
	if col.Retarget(to, now) {
		p.sound(t.spec.Slot)
	}
	return true
}

func (t *Training) phase(p *Page, phase string, now time.Time) {
	if phase != "advance" {
		return
	}
	owned := false
	if p.ready && p.active {
		owned = t.advance(p, now)
	}
	if !owned && !(p.ready && !p.active) {
		t.column(p).advance(now)
	}
}

// advance runs the shared clock and answers whether a variant change owned
// the column's step on this paint.
func (t *Training) advance(p *Page, now time.Time) bool {
	owned := t.WaitVariant != NoVariant
	if now.Before(p.Clock(t.spec.Clock)) {
		p.SetClock(t.spec.Clock, now)
		for i := range t.Static {
			if t.Static[i].Initialized && now.Before(t.Static[i].IdleLast) {
				t.Static[i].IdleLast = now
			}
		}
		return owned
	}
	for v := range t.Sides {
		t.armIdle(p, v, now)
	}
	if now.Sub(p.Clock(t.spec.Clock)) <= p.period(t.spec.Clock) {
		return owned
	}
	p.SetClock(t.spec.Clock, now)
	for v := range t.Sides {
		side := &t.Sides[v]
		if side.IdleActive {
			t.advanceIdle(p, v, now)
		} else if side.TransitionActive {
			side.TransitionIndex = (side.TransitionIndex + 1) % t.spec.Variants[v].TransitionCount
			if side.TransitionIndex == 0 {
				side.TransitionActive = false
			}
		}
	}
	if v := t.WaitVariant; v != NoVariant {
		side, col := &t.Sides[v], t.column(p)
		if !t.ColumnStarted && side.TransitionIndex >= t.spec.Variants[v].ColumnAt {
			t.ColumnStarted = true
			p.sound(t.spec.Slot)
		}
		if t.ColumnStarted && col.Frame != col.Endpoint(v) {
			col.Step()
		}
		if !side.TransitionActive && col.Frame == col.Endpoint(v) {
			t.WaitVariant = NoVariant
		}
	}
	return owned
}

func (t *Training) armIdle(p *Page, v int, now time.Time) {
	side, st := &t.Sides[v], &t.Static[v]
	busy := side.TransitionActive || side.IdleActive
	if !st.Initialized {
		st.Initialized, st.IdleLast = true, now
		if t.idleComplete(p, v) {
			st.IdleExtra = p.wait(*t.spec.Delay)
		}
		return
	}
	if now.Before(st.IdleLast) || busy {
		st.IdleLast = now
		return
	}
	wait := time.Duration(t.spec.DelayMS)*time.Millisecond + st.IdleExtra
	if !t.idleComplete(p, v) || now.Sub(st.IdleLast) <= wait {
		return
	}
	*side = TrainingSide{TransitionActive: side.TransitionActive, TransitionIndex: side.TransitionIndex,
		IdleActive: true, IdleIndex: -1, IdleDirection: 1}
	st.IdleLast = now
}

func (t *Training) advanceIdle(p *Page, v int, now time.Time) {
	side, st := &t.Sides[v], &t.Static[v]
	last := t.spec.Variants[v].IdleCount - 1
	if side.TransitionActive && side.IdleDirection > 0 {
		side.IdleDirection = -1
		st.Hold = st.HoldLimit
	}
	if side.IdleDirection > 0 {
		if side.IdleIndex < last {
			side.IdleIndex++
			side.IdleCached = side.IdleIndex
			return
		}
		side.IdleDirection = -1
		st.Hold, st.HoldLimit = 0, p.amount(*t.spec.HoldPick)
		// The reverse may restart from another index while the shown frame
		// stays.
		if from := t.spec.Variants[v].ReverseFrom; from != nil {
			side.IdleIndex = *from
		}
		return
	}
	if st.Hold < st.HoldLimit {
		st.Hold++
		if st.Hold < st.HoldLimit {
			return
		}
	}
	if side.IdleIndex <= 0 {
		side.IdleActive, side.IdleIndex, side.IdleCached, side.IdleDirection = false, 0, 0, 0
		st.IdleLast, st.IdleExtra = now, p.wait(*t.spec.Delay)
		return
	}
	side.IdleIndex--
	side.IdleCached = side.IdleIndex
}

func (t *Training) draw(p *Page, dst *image.RGBA, layer LayerSpec) {
	if !p.ready {
		return
	}
	for v, s := range t.spec.Variants {
		if s.Name != layer.Actor {
			continue
		}
		side := t.Sides[v]
		switch {
		case side.IdleActive:
			place(dst, frameAt(p.frames(s.Idle), side.IdleCached), layer)
		case side.TransitionActive:
			place(dst, frameAt(p.frames(s.Transition), side.TransitionIndex), layer)
		}
	}
}
