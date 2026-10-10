package town

import (
	"image"
	"strings"
	"time"
)

// SceneSpec is one scene as data: the square, or a room page's centre, whose
// page kind draws its own widgets between the layer groups. SquareScene
// answers the square's.
type SceneSpec struct {
	View     ViewSpec      `json:"view"`
	Random   []RandomSpec  `json:"random"`
	Art      []ArtSpec     `json:"art"`
	Mask     MaskSpec      `json:"mask"`
	Hotspots []HotspotSpec `json:"hotspots"`
	Pointer  PointerSpec   `json:"pointer"`
	Sounds   SoundSpec     `json:"sounds"`
	Clocks   []SceneClock  `json:"clocks"`
	Actors   []ActorSpec   `json:"actors"`
	// Lifecycle is how the scene is entered: "" when the game enters it, or
	// on its first activation, and a pause leaves it entered; "shown" when it
	// is entered each time it is shown and left each time it is hidden.
	Lifecycle string `json:"lifecycle"`
	// AdvanceWhen "active" runs no paint step while the scene is inactive.
	AdvanceWhen string `json:"advance-when"`
	// EnterActive makes an entered scene run at once: the first activation
	// after an entry restamps nothing.
	EnterActive bool `json:"enter-active"`
	// Enter runs once each time the scene is entered, after every actor's own
	// entry: "sound <slot>" requests a slot's sound.
	Enter  []string    `json:"enter"`
	Steps  []StepGroup `json:"steps"`
	Layers []LayerSpec `json:"layers"`
	Cite   []string    `json:"cite"`
}

// LifecycleShown is the lifecycle of a scene entered each time it is shown.
const LifecycleShown = "shown"

// SceneClock is one named clock of a scene. Entry stamps it at now plus
// EntryMS. A group bound to it runs when the time since its stamp exceeds the
// period ("greater") or reaches it ("at-least"), and restamps it. A clock with
// no period is a stamp an actor reads. BackStamps restamps a clock that reads
// a time before its stamp; the paint then runs no further group. Rebase
// restamps it at now when the scene resumes: on every resume ("every"), or on
// every resume after the first ("after-first"). A Process clock's stamp
// outlives every reset and entry, kept in the Process; while it holds no
// stamp a paint only stamps it and runs no group.
type SceneClock struct {
	Name       string   `json:"name"`
	PeriodMS   *int     `json:"period-ms"`
	Compare    string   `json:"compare"`
	EntryMS    int      `json:"entry-ms"`
	BackStamps bool     `json:"back-stamps"`
	Rebase     string   `json:"rebase"`
	Process    bool     `json:"process"`
	Cite       []string `json:"cite"`
}

// StepGroup is a run of step entries, each "<phase> <actor>" or "sound
// <slot>". A group bound to a clock runs only when the clock admits it; a
// group "when" "active" runs only while the scene is entered and active.
type StepGroup struct {
	Clock string   `json:"clock"`
	When  string   `json:"when"`
	Run   []string `json:"run"`
	Cite  []string `json:"cite"`
}

// EventSpec binds an event the game raises to an actor's action.
type EventSpec struct {
	Event string   `json:"event"`
	Do    string   `json:"do"`
	Cite  []string `json:"cite"`
}

// StateSpec is one state of an alternating or priority actor: its art, the
// frame count that makes the art complete, and how it moves and sounds.
type StateSpec struct {
	Name     string `json:"name"`
	Art      string `json:"art"`
	Complete int    `json:"complete"`
	// When is the wait parity that chooses an alternating state: "odd" or
	// "even".
	When string `json:"when"`
	// Motion is "ping-pong" (forward to the last frame, then back to the
	// first) or "forward" (to the last frame, then the index returns to zero
	// and the shown frame stays).
	Motion string `json:"motion"`
	// Steps is the index at which a priority state ends.
	Steps       int        `json:"steps"`
	StartSounds []string   `json:"start-sounds"`
	FrameSounds []FrameCue `json:"frame-sounds"`
	EndSounds   []string   `json:"end-sounds"`
	Cite        []string   `json:"cite"`
}

// VariantSpec is one variant of a training actor: its transition and idle
// art and counts, the idle index its reverse restarts from, and the
// transition index at which a variant change starts the column.
type VariantSpec struct {
	Name            string   `json:"name"`
	Transition      string   `json:"transition"`
	TransitionCount int      `json:"transition-count"`
	Idle            string   `json:"idle"`
	IdleCount       int      `json:"idle-count"`
	ReverseFrom     *int     `json:"reverse-from"`
	ColumnAt        int      `json:"column-at"`
	Cite            []string `json:"cite"`
}

// Voice is one retained sound the Host started.
type Voice interface {
	Playing() bool
	Stop()
}

// Host is what a game gives a scene: its art, clock, draws, conditions,
// values, sound and campaign hooks. Condition, Value and Hook answer the
// names a description uses; the composer gives the names no meaning of its
// own.
type Host interface {
	// Art answers the scene's art, nil while it has none.
	Art() *Art
	Now() time.Time
	// Draw is a bounded draw in [0,n) from the named host source.
	Draw(source string, n int) int
	// Seed seeds the process-scoped generators the description keeps.
	Seed() int64
	Condition(name string) bool
	// Value answers a named number the game holds.
	Value(name string) int
	// PlaySound starts key for a slot's source and answers its voice, or nil
	// when the key cannot play.
	PlaySound(source, key string, loop bool) Voice
	StopSound(Voice)
	// StartLoop starts the scene's entry loop and reports whether it started.
	StartLoop(key string) bool
	StopLoop(key string)
	// Leave releases the named room's own audio scope when a shown scene is
	// hidden.
	Leave(room string)
	// Hook runs one named campaign hook for the named room.
	Hook(name, room string)
}

// Process is the state that outlives every reset of a scene: the process
// clocks' stamps, the description's generators, the flocks' wait latches and
// the held episodes' end counters. A game keeps one for the life of the
// process; its zero value is ready.
type Process struct {
	clocks map[string]time.Time
	lcg    map[string]*lcg
	waits  map[string]*Wait
	ends   map[string]int
}

func (p *Process) init() {
	if p.lcg == nil {
		p.lcg, p.waits, p.ends = map[string]*lcg{}, map[string]*Wait{}, map[string]int{}
	}
	if p.clocks == nil {
		p.clocks = map[string]time.Time{}
	}
}

// Scene is one scene built from a description: the square or a room page.
// Reset returns it to a scene never entered; the process state survives.
type Scene struct {
	desc *Description
	spec *SceneSpec
	name string
	host Host
	proc *Process

	ready, present, active, started bool
	hover                           *HotspotSpec
	latches                         map[string]bool
	voices                          []Voice
	loopOn                          bool

	clocks map[string]*sceneClock
	order  []*sceneClock
	actors []actor
	byName map[string]actor
	slots  map[string]int
	spots  map[string]*HotspotSpec
	rooms  map[string]*RoomSpec
}

type sceneClock struct {
	spec *SceneClock
	last time.Time
}

// NewScene builds the scene of the named room of d over host, keeping its
// process-scoped state in proc; a nil proc gives the scene its own. The
// square's name builds the square. It answers nil for a room with no scene.
// d must have passed Validate.
func NewScene(d *Description, room string, host Host, proc *Process) *Scene {
	spec := d.Scene(room)
	if spec == nil {
		return nil
	}
	if proc == nil {
		proc = &Process{}
	}
	proc.init()
	s := &Scene{
		desc: d, spec: spec, name: room, host: host, proc: proc,
		clocks: map[string]*sceneClock{}, byName: map[string]actor{}, slots: map[string]int{},
		spots: map[string]*HotspotSpec{}, rooms: map[string]*RoomSpec{},
	}
	for i := range spec.Clocks {
		c := &sceneClock{spec: &spec.Clocks[i]}
		s.clocks[c.spec.Name] = c
		s.order = append(s.order, c)
	}
	for i, sl := range spec.Sounds.Slots {
		s.slots[sl.Name] = i
	}
	s.voices = make([]Voice, len(spec.Sounds.Slots))
	for i := range spec.Hotspots {
		s.spots[spec.Hotspots[i].Name] = &spec.Hotspots[i]
	}
	for i := range d.Rooms {
		s.rooms[d.Rooms[i].Name] = &d.Rooms[i]
	}
	for i := range spec.Actors {
		a := newActor(&spec.Actors[i])
		s.actors = append(s.actors, a)
		s.byName[spec.Actors[i].Name] = a
		for _, m := range a.members() {
			s.byName[m] = a
		}
	}
	s.Reset()
	return s
}

// Actor answers the runtime of the named actor or actor member: an *Episode,
// *Pendulum, *Stepper, *Driven, *Flock, *Families, *Loop, *Alternating,
// *Selector, *Priority, *Bounce, *Cycle, *Target or *Training; nil for an
// unknown name.
func (s *Scene) Actor(name string) any {
	if a, ok := s.byName[name]; ok {
		return a
	}
	return nil
}

func (s *Scene) Description() *Description { return s.desc }

func (s *Scene) Spec() *SceneSpec { return s.spec }

// Reset stops the entry loop and every slot, then returns every actor, latch,
// hover and scene-scoped clock to a scene never entered. It reads no art: an
// actor whose rest frame depends on its art takes it when the scene is next
// made ready.
func (s *Scene) Reset() {
	s.stopLoop()
	s.silence()
	s.ready, s.present, s.active, s.started = false, false, false, false
	s.hover = nil
	s.latches = map[string]bool{}
	for _, c := range s.order {
		if !c.spec.Process {
			c.last = time.Time{}
		}
	}
	for _, a := range s.actors {
		a.reset(s)
	}
}

// Ready reports whether the scene has been made ready since its last reset:
// entered, or for a shown scene activated.
func (s *Scene) Ready() bool { return s.ready }

// Present reports whether the scene is entered and, for a shown scene,
// visible.
func (s *Scene) Present() bool { return s.present }

// Active reports whether the scene runs.
func (s *Scene) Active() bool { return s.active }

// Enter returns the scene to one just entered: reset, made ready, every
// scene-scoped clock stamped, every actor entered in order, then the entry
// steps.
func (s *Scene) Enter() {
	s.Reset()
	now := s.host.Now()
	s.ready = true
	s.active = s.spec.EnterActive
	s.prepare()
	s.enter(now)
}

func (s *Scene) prepare() {
	for _, a := range s.actors {
		a.prepare(s)
	}
}

func (s *Scene) enter(now time.Time) {
	s.present = true
	for _, c := range s.order {
		if !c.spec.Process {
			c.last = now.Add(time.Duration(c.spec.EntryMS) * time.Millisecond)
		}
	}
	for _, a := range s.actors {
		a.enter(s, now)
	}
	s.runEntries(s.spec.Enter, now)
}

// SetActive is the scene's lifecycle. shown says whether the scene is the
// screen shown; active is ignored when it is not. A scene asked to run that
// is not ready is made ready: a shown scene prepares its actors, any other
// is entered. A shown scene that is hidden is left and its room's audio
// released; shown again it is entered. A pause clears the hover and stops
// every slot and the entry loop; a resume restamps the clocks the scene
// rebases, lets each actor rebase its own and starts the entry loop.
func (s *Scene) SetActive(active, shown bool) {
	active = active && shown
	followsShow := s.spec.Lifecycle == LifecycleShown
	if !s.ready {
		if !active {
			s.pause()
			return
		}
		if followsShow {
			s.ready = true
			s.prepare()
		} else {
			s.Enter()
		}
	}
	if followsShow && !shown {
		s.host.Leave(s.name)
		s.present = false
		for _, a := range s.actors {
			a.leave(s)
		}
	}
	if followsShow && shown && !s.present {
		s.enter(s.host.Now())
	}
	if !active {
		s.pause()
		return
	}
	if s.active {
		return
	}
	now := s.host.Now()
	for _, c := range s.order {
		if c.spec.Rebase == "every" || c.spec.Rebase == "after-first" && s.started {
			s.setStamp(c, now)
		}
	}
	for _, a := range s.actors {
		a.resume(s, now)
	}
	s.started, s.active = true, true
	s.startLoop()
}

func (s *Scene) pause() {
	s.active, s.hover = false, nil
	s.silence()
	s.stopLoop()
}

// Advance is one paint of the scene: each step group in order. A scene that
// advances only while active, or whose art is absent, does not advance. While
// a process clock holds no stamp the paint only stamps it.
func (s *Scene) Advance() {
	if s.spec.AdvanceWhen == "active" && !s.active {
		return
	}
	if s.host.Art() == nil {
		return
	}
	now := s.host.Now()
	first := false
	for _, c := range s.order {
		if c.spec.Process && s.stamp(c).IsZero() {
			s.setStamp(c, now)
			first = true
		}
	}
	if first {
		return
	}
	for _, g := range s.spec.Steps {
		if g.When == "active" && (!s.ready || !s.active) {
			continue
		}
		if c := s.clocks[g.Clock]; c != nil {
			last := s.stamp(c)
			if c.spec.BackStamps && now.Before(last) {
				s.setStamp(c, now)
				return
			}
			if !elapsed(now.Sub(last), time.Duration(*c.spec.PeriodMS)*time.Millisecond, c.spec.Compare) {
				continue
			}
			s.setStamp(c, now)
		}
		s.runEntries(g.Run, now)
	}
}

func elapsed(since, wait time.Duration, compare string) bool {
	if compare == "at-least" {
		return since >= wait
	}
	return since > wait
}

func (s *Scene) stamp(c *sceneClock) time.Time {
	if c.spec.Process {
		return s.proc.clocks[c.spec.Name]
	}
	return c.last
}

func (s *Scene) setStamp(c *sceneClock, t time.Time) {
	if c.spec.Process {
		s.proc.clocks[c.spec.Name] = t
		return
	}
	c.last = t
}

// Clock answers a clock's stamp.
func (s *Scene) Clock(name string) time.Time {
	if c := s.clocks[name]; c != nil {
		return s.stamp(c)
	}
	return time.Time{}
}

// SetClock stamps a clock; the zero time makes a process clock's next paint
// only stamp it.
func (s *Scene) SetClock(name string, t time.Time) {
	if c := s.clocks[name]; c != nil {
		s.setStamp(c, t)
	}
}

func (s *Scene) period(name string) time.Duration {
	if c := s.clocks[name]; c != nil && c.spec.PeriodMS != nil {
		return time.Duration(*c.spec.PeriodMS) * time.Millisecond
	}
	return 0
}

func (s *Scene) compare(name string) string {
	if c := s.clocks[name]; c != nil {
		return c.spec.Compare
	}
	return ""
}

func (s *Scene) runEntries(list []string, now time.Time) {
	for _, entry := range list {
		phase, name, _ := strings.Cut(entry, " ")
		if phase == "sound" {
			s.sound(name, "")
			continue
		}
		if a := s.byName[name]; a != nil {
			a.phase(s, phase, name, now)
		}
	}
}

// Event raises a named game event with its arguments; every actor bound to
// it runs the bound action. It answers whether any action changed state.
func (s *Scene) Event(name string, args ...int) bool {
	changed := false
	for i, a := range s.actors {
		for _, on := range s.spec.Actors[i].On {
			if on.Event == name && a.event(s, on.Do, args) {
				changed = true
			}
		}
	}
	return changed
}

// HotspotAt answers the hotspot under p, a scene-relative point, read from
// the mask; nil off the mask or over an unmapped byte.
func (s *Scene) HotspotAt(p image.Point) *HotspotSpec {
	art := s.host.Art()
	if art == nil || art.Mask == nil || !p.In(art.Mask.Bounds()) {
		return nil
	}
	b := int(art.Mask.ColorIndexAt(p.X, p.Y))
	for _, m := range s.spec.Mask.Bytes {
		if m.Byte == b {
			return s.spots[m.Hotspot]
		}
	}
	return nil
}

// Hotspot answers the named hotspot.
func (s *Scene) Hotspot(name string) *HotspotSpec { return s.spots[name] }

// HotspotForRow answers the hotspot the list fallback shows at row i.
func (s *Scene) HotspotForRow(i int) *HotspotSpec {
	for k := range s.spec.Hotspots {
		if r := s.spec.Hotspots[k].Row; r != nil && *r == i {
			return &s.spec.Hotspots[k]
		}
	}
	return nil
}

// Hover answers the hovered hotspot, nil for none.
func (s *Scene) Hover() *HotspotSpec { return s.hover }

// Pointer is one delivered pointer update at p. The every-update reactions run
// first, then the hovered hotspot's or the off-hotspot reactions.
func (s *Scene) Pointer(p image.Point) {
	s.hover = s.HotspotAt(p)
	s.run(s.spec.Pointer.Every)
	if s.hover == nil {
		s.run(s.spec.Pointer.Off)
		return
	}
	s.run(s.hover.Hover)
}

// Click runs a hotspot's click program and answers the menu it names, if any.
func (s *Scene) Click(h *HotspotSpec) string {
	if h == nil {
		return ""
	}
	return s.run(h.Click)
}

// Menu answers the menu a hotspot's click program names without running it.
func Menu(ops []Op) string {
	for _, op := range ops {
		if op.Menu != "" {
			return op.Menu
		}
	}
	return ""
}

func (s *Scene) run(ops []Op) string {
	menu := ""
	for _, op := range ops {
		switch {
		case op.Arm != "":
			if op.Chance == nil || s.chance(*op.Chance) {
				if a := s.byName[op.Arm]; a != nil {
					a.arm(s, op.Arm)
				}
			}
		case op.Latch != "":
			if !s.latches[op.Latch] {
				for _, sl := range op.Stop {
					s.stop(sl)
				}
				s.sound(op.Slot, op.Sound)
				s.latches[op.Latch] = true
				for _, c := range op.Clear {
					s.latches[c] = false
				}
			}
		case op.Drive != "":
			if op.Unless == "" || !s.host.Condition(op.Unless) {
				if a := s.byName[op.Drive]; a != nil {
					a.drive(s, op.Dir)
				}
			}
		case len(op.ClearLatch) > 0:
			for _, c := range op.ClearLatch {
				s.latches[c] = false
			}
		case op.If != "":
			branch := op.Else
			if s.host.Condition(op.If) {
				branch = op.Then
			}
			if m := s.run(branch); m != "" {
				menu = m
			}
		case op.Room != "":
			s.EnterRoom(op.Room)
		case op.Hook != "":
			s.host.Hook(op.Hook, "")
		case op.Menu != "":
			menu = op.Menu
		}
	}
	return menu
}

// Latch reports one hover latch.
func (s *Scene) Latch(name string) bool { return s.latches[name] }

// EnterRoom runs a room's entry steps in order.
func (s *Scene) EnterRoom(name string) {
	room := s.rooms[name]
	if room == nil {
		return
	}
	s.steps(room.Enter, room.Name)
}

// LeaveRoom runs a room's exit steps, then the steps that return to the square.
func (s *Scene) LeaveRoom(name string) {
	if room := s.rooms[name]; room != nil {
		s.steps(room.Exit, room.Name)
	}
	s.EnterSquare()
}

// EnterSquare runs the steps that return to the square.
func (s *Scene) EnterSquare() {
	s.steps(s.desc.Square.Enter, s.desc.Square.Name)
}

// Room answers the named room.
func (s *Scene) Room(name string) *RoomSpec { return s.rooms[name] }

func (s *Scene) steps(list []Step, room string) {
	for _, st := range list {
		switch {
		case st.Composer == "reset":
			s.Reset()
		case st.Hook != "":
			s.host.Hook(st.Hook, room)
		}
	}
}

// SaveAdmitted answers the description's save admission condition.
func (s *Scene) SaveAdmitted() bool {
	return s.desc.Save.AdmittedWhen == "" || s.host.Condition(s.desc.Save.AdmittedWhen)
}

// sound requests key on a slot, or the slot's own key when key is empty. An
// untracked slot plays every request. A tracked slot whose sound still plays
// takes no new request; a finished one is stopped first.
func (s *Scene) sound(slot, key string) {
	i, ok := s.slots[slot]
	if !ok {
		return
	}
	spec := s.spec.Sounds.Slots[i]
	if key == "" {
		key = spec.Key
	}
	if spec.Untracked {
		s.host.PlaySound(spec.Source, key, spec.Loop)
		return
	}
	if vc := s.voices[i]; vc != nil {
		if vc.Playing() {
			return
		}
		s.stopIndex(i)
	}
	s.voices[i] = s.host.PlaySound(spec.Source, key, spec.Loop)
}

func (s *Scene) stop(slot string) {
	if i, ok := s.slots[slot]; ok {
		s.stopIndex(i)
	}
}

func (s *Scene) stopIndex(i int) {
	if s.voices[i] != nil {
		s.host.StopSound(s.voices[i])
		s.voices[i] = nil
	}
}

func (s *Scene) silence() {
	for i := range s.voices {
		s.stopIndex(i)
	}
}

// Silence stops every tracked slot's voice.
func (s *Scene) Silence() { s.silence() }

// Voice answers a slot's retained voice.
func (s *Scene) Voice(slot string) Voice {
	if i, ok := s.slots[slot]; ok {
		return s.voices[i]
	}
	return nil
}

func (s *Scene) startLoop() {
	if s.loopOn || s.spec.Sounds.Loop == "" {
		return
	}
	s.loopOn = s.host.StartLoop(s.spec.Sounds.Loop)
}

func (s *Scene) stopLoop() {
	if !s.loopOn {
		return
	}
	s.host.StopLoop(s.spec.Sounds.Loop)
	s.loopOn = false
}

// LoopOn reports whether the entry loop runs.
func (s *Scene) LoopOn() bool { return s.loopOn }

func (s *Scene) frames(name string) []image.Image {
	return s.host.Art().Pictures(name)
}

// Complete reports whether an art entry holds exactly n frames, none nil.
func (s *Scene) Complete(name string, n int) bool {
	frames := s.frames(name)
	if len(frames) != n {
		return false
	}
	for _, f := range frames {
		if f == nil {
			return false
		}
	}
	return true
}
