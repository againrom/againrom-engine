package town

import (
	"image"
	"time"
)

// Loop cycles the first Loop frames of a complete art entry, one frame each
// time its step runs. Shown is the frame published for paint.
type Loop struct {
	still
	spec         *ActorSpec
	Index, Shown int
}

func (l *Loop) reset(s *Scene)                { l.Index, l.Shown = 0, 0 }
func (l *Loop) enter(s *Scene, now time.Time) { l.reset(s) }

func (l *Loop) phase(s *Scene, phase, name string, now time.Time) {
	switch phase {
	case "publish":
		l.Shown = l.Index
	case "advance":
		if s.Complete(l.spec.Art, l.spec.Frames) {
			l.Index = (l.Index + 1) % l.spec.Loop
		}
	}
}

func (l *Loop) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	if s.ready {
		place(dst, frameAt(s.frames(l.spec.Art), l.Shown), layer)
	}
}

// Alternating rests for a drawn wait, then starts the state whose parity
// matches the wait's millisecond count when that state's art is complete;
// otherwise it draws a new wait. A started state steps on its clock's period:
// "forward" to its last frame, after which the index returns to zero while
// the shown frame stays; "ping-pong" forward then back to the first frame.
// Each state keeps its own index and the last frame it showed.
type Alternating struct {
	still
	spec       *ActorSpec
	State      int // 0 resting, otherwise the 1-based running state
	Index      []int
	Cached     []int
	Direction  int
	Delay      time.Duration
	ShownState int
	ShownFrame int
}

func (a *Alternating) reset(s *Scene) {
	n := len(a.spec.States)
	a.State, a.Direction, a.Delay, a.ShownState, a.ShownFrame = 0, 0, 0, 0, 0
	a.Index, a.Cached = make([]int, n), make([]int, n)
}

func (a *Alternating) enter(s *Scene, now time.Time) {
	a.reset(s)
	a.Delay = s.pickWait(*a.spec.Delay)
}

func (a *Alternating) finish(s *Scene, now time.Time, clearDirection bool) {
	a.State = 0
	if clearDirection {
		a.Direction = 0
	}
	a.Delay = s.pickWait(*a.spec.Delay)
	s.SetClock(a.spec.Clock, now)
}

func (a *Alternating) phase(s *Scene, phase, name string, now time.Time) {
	switch phase {
	case "arm":
		if now.Sub(s.Clock(a.spec.Clock)) <= a.Delay {
			return
		}
		parity := "even"
		if a.Delay/time.Millisecond%2 != 0 {
			parity = "odd"
		}
		for i, st := range a.spec.States {
			if st.When != parity {
				continue
			}
			if s.Complete(st.Art, st.Complete) {
				a.State = i + 1
				if st.Motion == "ping-pong" {
					a.Direction = 1
				}
				for _, slot := range st.StartSounds {
					s.sound(slot, "")
				}
				return
			}
			break
		}
		// A chosen state whose art is missing does not trap the actor: it
		// draws a new wait and stays as it is.
		s.SetClock(a.spec.Clock, now)
		a.Delay = s.pickWait(*a.spec.Delay)
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
				s.sound(c.Slot, "")
			}
		}
	case "advance":
		a.advance(s, now)
	}
}

func (a *Alternating) advance(s *Scene, now time.Time) {
	if a.State == 0 || now.Sub(s.Clock(a.spec.Clock)) <= s.period(a.spec.Clock) {
		return
	}
	s.SetClock(a.spec.Clock, now)
	i := a.State - 1
	st := a.spec.States[i]
	frames := s.frames(st.Art)
	if st.Motion == "forward" {
		if !stepFrame(&a.Index[i], &a.Cached[i], frames, 1) {
			a.Index[i] = 0
			a.end(s, now, st, false)
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
	a.end(s, now, st, true)
}

func (a *Alternating) end(s *Scene, now time.Time, st StateSpec, clearDirection bool) {
	for _, slot := range st.EndSounds {
		s.sound(slot, "")
	}
	a.finish(s, now, clearDirection)
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

func (a *Alternating) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	if s.ready && a.ShownState > 0 {
		place(dst, frameAt(s.frames(a.spec.States[a.ShownState-1].Art), a.ShownFrame), layer)
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
	still
	spec     *ActorSpec
	Selected int
	Enabled  []bool
	Index    []int
}

func (sel *Selector) members() []string {
	var out []string
	for _, m := range sel.spec.Members {
		out = append(out, m.Name)
	}
	return out
}

func (sel *Selector) reset(s *Scene) {
	sel.Selected = 0
	sel.Enabled, sel.Index = make([]bool, len(sel.spec.Members)), make([]int, len(sel.spec.Members))
}

func (sel *Selector) enter(s *Scene, now time.Time) {
	sel.reset(s)
	sel.Selected = s.host.Value(sel.spec.Selected)
	if sel.Selected >= 0 && sel.Selected < len(sel.Enabled) {
		sel.Enabled[sel.Selected] = true
	}
}

func (sel *Selector) event(s *Scene, do string, args []int) bool {
	if do != "select" || !s.ready || len(args) == 0 {
		return false
	}
	m := args[0]
	if m < 0 || m >= len(sel.Enabled) || m == sel.Selected {
		return false
	}
	if old := sel.Selected; old >= 0 && old < len(sel.Enabled) && sel.Enabled[old] {
		sel.Index[old] = sel.spec.Release
	}
	sel.Selected = m
	sel.Enabled[m], sel.Index[m] = true, 0
	if sel.spec.Raise != "" {
		s.Event(sel.spec.Raise)
	}
	return true
}

func (sel *Selector) phase(s *Scene, phase, name string, now time.Time) {
	if phase != "advance" {
		return
	}
	for i := range sel.Enabled {
		if !sel.Enabled[i] {
			continue
		}
		switch next := sel.Index[i] + 1; next {
		case sel.spec.LoopAt:
			sel.Index[i] = sel.spec.LoopTo
		case sel.spec.StopAt:
			sel.Enabled[i], sel.Index[i] = false, 0
		default:
			sel.Index[i] = next
		}
	}
}

func (sel *Selector) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	for i, m := range sel.spec.Members {
		if m.Name != layer.Actor {
			continue
		}
		frames := s.frames(m.Art)
		if !s.ready || !s.Complete(m.Art, sel.spec.Frames) {
			place(dst, frameAt(frames, 0), layer)
			continue
		}
		if sel.Enabled[i] {
			place(dst, frameAt(frames, sel.Index[i]), layer)
		}
	}
}

// Priority runs at most one of its states at a time, the first raised in
// list order. Its first state is raised after a wait drawn on every step;
// the others by events. A running state advances one index per step and ends
// at its Steps, restamping the actor's clock. Index i above zero shows frame
// i-1 of a complete state; otherwise the actor shows its base art.
type Priority struct {
	still
	spec   *ActorSpec
	Raised []bool
	Index  int
}

func (q *Priority) reset(s *Scene) {
	q.Raised, q.Index = make([]bool, len(q.spec.States)), 0
}

func (q *Priority) enter(s *Scene, now time.Time) { q.reset(s) }

func (q *Priority) event(s *Scene, do string, args []int) bool {
	if !s.ready {
		return false
	}
	for i, st := range q.spec.States {
		if st.Name == do {
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

func (q *Priority) phase(s *Scene, phase, name string, now time.Time) {
	switch phase {
	case "arm":
		wait := s.pickWait(*q.spec.Delay)
		if q.Running() < 0 && elapsed(now.Sub(s.Clock(q.spec.Clock)), wait, s.compare(q.spec.Clock)) {
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
		s.SetClock(q.spec.Clock, now)
	}
}

func (q *Priority) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	pic := frameAt(s.frames(q.spec.Art), 0)
	if i := q.Running(); s.ready && i >= 0 {
		st := q.spec.States[i]
		if q.Index > 0 && q.Index < st.Steps && s.Complete(st.Art, st.Complete) {
			if f := frameAt(s.frames(st.Art), q.Index-1); f != nil {
				pic = f
			}
		}
	}
	place(dst, pic, layer)
}

// Bounce runs from its first frame to its last and back once each time it is
// armed, one frame per step. It is shown while running, from its first step.
type Bounce struct {
	still
	spec        *ActorSpec
	Frame, Step int
	Stepped     bool
}

func (b *Bounce) reset(s *Scene)                { b.Frame, b.Step, b.Stepped = 0, 0, false }
func (b *Bounce) enter(s *Scene, now time.Time) { b.reset(s) }

func (b *Bounce) event(s *Scene, do string, args []int) bool {
	if do != "arm" {
		return false
	}
	b.Step = 1
	return true
}

func (b *Bounce) phase(s *Scene, phase, name string, now time.Time) {
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

func (b *Bounce) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	if b.Shown() {
		place(dst, frameAt(s.frames(b.spec.Art), b.Frame), layer)
	}
}

// Cycle counts modulo Frames, one count each time its wait has elapsed since
// its last count. Its first step, and a step that reads a time before its
// stamp, only stamps it. Its state outlives every entry. Order maps the
// count to a slot per variant.
type Cycle struct {
	still
	spec    *ActorSpec
	Index   int
	Last    time.Time
	Stamped bool
}

func (c *Cycle) resume(s *Scene, now time.Time)              { c.Last = now }
func (c *Cycle) draw(s *Scene, dst *image.RGBA, l LayerSpec) {}

func (c *Cycle) phase(s *Scene, phase, name string, now time.Time) {
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
	still
	spec        *ActorSpec
	Frame, Goal int
	Ready       bool
	Last        time.Time
}

func (t *Target) reset(s *Scene)                                    { t.Frame, t.Goal, t.Ready, t.Last = 0, 0, false, time.Time{} }
func (t *Target) phase(s *Scene, phase, name string, now time.Time) {}

func (t *Target) enter(s *Scene, now time.Time) {
	t.reset(s)
	if v := s.host.Value(t.spec.Value); v >= 0 {
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

func (t *Target) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	if t.Ready {
		place(dst, frameAt(s.frames(t.spec.Art), t.Frame), layer)
	}
}

// Training is a pair of variant sides, each with a transition sequence and
// an idle sequence that has priority over it. Entry arms the transition of
// the variant a host value names. A variant change arms that variant's
// transition, waits for it, and walks the column actor to the new endpoint
// from the transition's ColumnAt index; without a complete transition the
// column walks on its own. Sides step together on the clock's period. An idle
// sequence starts after its wait, plays forward, holds for a drawn count,
// then plays back. Its timers, waits and holds outlive every entry.
type Training struct {
	still
	spec          *ActorSpec
	WaitVariant   int
	ColumnStarted bool
	Sides         []TrainingSide
	Static        []TrainingStatic
}

// TrainingSide is one variant's entry-scoped sequence state.
type TrainingSide struct {
	TransitionActive bool
	TransitionIndex  int
	IdleActive       bool
	IdleIndex        int
	IdleCached       int
	IdleDirection    int
}

// TrainingStatic is one variant's idle timer, wait and hold, which outlive
// every entry.
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

func (t *Training) reset(s *Scene) {
	t.WaitVariant, t.ColumnStarted = 0, false
	t.Sides = make([]TrainingSide, len(t.spec.Variants))
	if len(t.Static) != len(t.spec.Variants) {
		t.Static = make([]TrainingStatic, len(t.spec.Variants))
	}
}

func (t *Training) column(s *Scene) *Target {
	c, _ := s.byName[t.spec.Column].(*Target)
	return c
}

func (t *Training) enter(s *Scene, now time.Time) {
	t.reset(s)
	t.WaitVariant = NoVariant
	if v := s.host.Value(t.spec.Value); v >= 0 && v < len(t.Sides) {
		t.armVariant(s, v, now, false)
	}
}

func (t *Training) resume(s *Scene, now time.Time) {
	for i := range t.Static {
		if t.Static[i].Initialized {
			t.Static[i].IdleLast = now
		}
	}
}

// Busy reports whether the scene waits for a variant change to finish.
func (t *Training) Busy(s *Scene) bool { return s.ready && t.WaitVariant != NoVariant }

func (t *Training) transitionComplete(s *Scene, v int) bool {
	vs := t.spec.Variants[v]
	return s.Complete(vs.Transition, vs.TransitionCount)
}

func (t *Training) idleComplete(s *Scene, v int) bool {
	vs := t.spec.Variants[v]
	return s.Complete(vs.Idle, vs.IdleCount)
}

func (t *Training) armVariant(s *Scene, v int, now time.Time, changed bool) bool {
	if v < 0 || v >= len(t.Sides) || !t.transitionComplete(s, v) {
		return false
	}
	side := &t.Sides[v]
	side.TransitionActive, side.TransitionIndex = true, 0
	if changed {
		t.WaitVariant = v
	}
	col := t.column(s)
	if !col.Ready {
		col.Rest(v)
	}
	if changed {
		col.Retarget(v, now)
	}
	if !s.Complete(col.spec.Art, col.spec.Frames) {
		col.Rest(v)
	}
	t.ColumnStarted = col.Frame == col.Endpoint(v)
	return true
}

func (t *Training) event(s *Scene, do string, args []int) bool {
	if do != "change" || len(args) < 2 {
		return false
	}
	from, to := args[0], args[1]
	if from == to {
		return false
	}
	col := t.column(s)
	if !col.Ready {
		col.Rest(from)
	}
	now := s.host.Now()
	if t.armVariant(s, to, now, true) {
		return true
	}
	if !s.Complete(col.spec.Art, col.spec.Frames) {
		col.Rest(to)
		return true
	}
	if col.Retarget(to, now) {
		s.sound(t.spec.Slot, "")
	}
	return true
}

func (t *Training) phase(s *Scene, phase, name string, now time.Time) {
	if phase != "advance" {
		return
	}
	owned := false
	if s.ready && s.active {
		owned = t.advance(s, now)
	}
	if !owned && !(s.ready && !s.active) {
		t.column(s).advance(now)
	}
}

// advance runs the shared clock and answers whether a variant change owned
// the column's step on this paint.
func (t *Training) advance(s *Scene, now time.Time) bool {
	owned := t.WaitVariant != NoVariant
	if now.Before(s.Clock(t.spec.Clock)) {
		s.SetClock(t.spec.Clock, now)
		for i := range t.Static {
			if t.Static[i].Initialized && now.Before(t.Static[i].IdleLast) {
				t.Static[i].IdleLast = now
			}
		}
		return owned
	}
	for v := range t.Sides {
		t.armIdle(s, v, now)
	}
	if now.Sub(s.Clock(t.spec.Clock)) <= s.period(t.spec.Clock) {
		return owned
	}
	s.SetClock(t.spec.Clock, now)
	for v := range t.Sides {
		side := &t.Sides[v]
		if side.IdleActive {
			t.advanceIdle(s, v, now)
		} else if side.TransitionActive {
			side.TransitionIndex = (side.TransitionIndex + 1) % t.spec.Variants[v].TransitionCount
			if side.TransitionIndex == 0 {
				side.TransitionActive = false
			}
		}
	}
	if v := t.WaitVariant; v != NoVariant {
		side, col := &t.Sides[v], t.column(s)
		if !t.ColumnStarted && side.TransitionIndex >= t.spec.Variants[v].ColumnAt {
			t.ColumnStarted = true
			s.sound(t.spec.Slot, "")
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

func (t *Training) armIdle(s *Scene, v int, now time.Time) {
	side, st := &t.Sides[v], &t.Static[v]
	busy := side.TransitionActive || side.IdleActive
	if !st.Initialized {
		st.Initialized, st.IdleLast = true, now
		if t.idleComplete(s, v) {
			st.IdleExtra = s.pickWait(*t.spec.Delay)
		}
		return
	}
	if now.Before(st.IdleLast) || busy {
		st.IdleLast = now
		return
	}
	wait := time.Duration(t.spec.DelayMS)*time.Millisecond + st.IdleExtra
	if !t.idleComplete(s, v) || now.Sub(st.IdleLast) <= wait {
		return
	}
	*side = TrainingSide{TransitionActive: side.TransitionActive, TransitionIndex: side.TransitionIndex,
		IdleActive: true, IdleIndex: -1, IdleDirection: 1}
	st.IdleLast = now
}

func (t *Training) advanceIdle(s *Scene, v int, now time.Time) {
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
		st.Hold, st.HoldLimit = 0, s.amount(*t.spec.HoldPick)
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
		st.IdleLast, st.IdleExtra = now, s.pickWait(*t.spec.Delay)
		return
	}
	side.IdleIndex--
	side.IdleCached = side.IdleIndex
}

func (t *Training) draw(s *Scene, dst *image.RGBA, layer LayerSpec) {
	if !s.ready {
		return
	}
	for v, vs := range t.spec.Variants {
		if vs.Name != layer.Actor {
			continue
		}
		side := t.Sides[v]
		switch {
		case side.IdleActive:
			place(dst, frameAt(s.frames(vs.Idle), side.IdleCached), layer)
		case side.TransitionActive:
			place(dst, frameAt(s.frames(vs.Transition), side.TransitionIndex), layer)
		}
	}
}
