package town

import (
	"image"
	"image/draw"
	"strings"
	"time"
)

// SceneSpec is a room page's centre as data: its art, draw sources, sound
// slots, clocks, actors, the steps one paint runs and the layers each paint
// group draws. The page kind draws its own widgets between the groups.
type SceneSpec struct {
	Random []RandomSpec `json:"random"`
	Art    []ArtSpec    `json:"art"`
	Sounds SoundSpec    `json:"sounds"`
	Clocks []PageClock  `json:"clocks"`
	Actors []ActorSpec  `json:"actors"`
	// EnterActive makes an entered page run at once: the first activation
	// after an entry restamps nothing.
	EnterActive bool `json:"enter-active"`
	// Enter runs once each time the page is entered, after every actor's own
	// entry: "sound <slot>" requests a slot's sound.
	Enter  []string    `json:"enter"`
	Steps  []StepGroup `json:"steps"`
	Layers []LayerSpec `json:"layers"`
	Cite   []string    `json:"cite"`
}

// PageClock is one named clock of a page. Entry stamps it at now plus
// EntryMS. A group bound to it runs when the time since its stamp exceeds the
// period ("greater") or reaches it ("at-least"), and restamps it. A clock with
// no period is a stamp an actor reads. BackStamps restamps a clock that reads
// a time before its stamp; the paint then runs no further group. Rebase
// restamps it at now when the page resumes: on every resume ("every"), or on
// every resume after the first ("after-first").
type PageClock struct {
	Name       string   `json:"name"`
	PeriodMS   *int     `json:"period-ms"`
	Compare    string   `json:"compare"`
	EntryMS    int      `json:"entry-ms"`
	BackStamps bool     `json:"back-stamps"`
	Rebase     string   `json:"rebase"`
	Cite       []string `json:"cite"`
}

// StepGroup is a run of step entries, each "<phase> <actor>" or "sound
// <slot>". A group bound to a clock runs only when the clock admits it; a
// group "when" "active" runs only while the page is entered and active.
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

// PageHost is what a game gives a room page: the page's art, the clock, the
// draws, sound, and the values the description names.
type PageHost interface {
	// Art answers the page's art, nil while the room has none.
	Art() *Art
	Now() time.Time
	// Draw is a bounded draw in [0,n) from the named host source.
	Draw(source string, n int) int
	// Reseed restarts a host source on page entry.
	Reseed(source string)
	// PlaySound starts key for a slot's source and answers its voice, or nil.
	PlaySound(source, key string, loop bool) Voice
	StopSound(Voice)
	// Value answers a named number the game holds.
	Value(name string) int
}

// Page is one room page's centre built from a scene: its actors, clocks and
// sound slots. Enter returns it to a page just entered and Reset to a page
// never entered; the actors' process-scoped state in Process outlives both.
type Page struct {
	room  *RoomSpec
	scene *SceneSpec
	host  PageHost
	proc  *Process

	ready, active, started bool

	clocks map[string]*pageClock
	order  []*pageClock
	actors []pageActor
	byName map[string]pageActor
	slots  map[string]int
	voices []Voice
}

type pageClock struct {
	spec *PageClock
	last time.Time
}

// NewPage builds the page of the named room, whose scene a validated
// description carries, keeping process-scoped state in proc. It answers nil
// for a room with no scene.
func NewPage(d *Description, room string, host PageHost, proc *Process) *Page {
	if proc == nil {
		proc = &Process{}
	}
	proc.init()
	var r *RoomSpec
	for i := range d.Rooms {
		if d.Rooms[i].Name == room {
			r = &d.Rooms[i]
		}
	}
	if r == nil || r.Scene == nil {
		return nil
	}
	p := &Page{room: r, scene: r.Scene, host: host, proc: proc,
		clocks: map[string]*pageClock{}, byName: map[string]pageActor{}, slots: map[string]int{}}
	for i := range r.Scene.Clocks {
		c := &pageClock{spec: &r.Scene.Clocks[i]}
		p.clocks[c.spec.Name] = c
		p.order = append(p.order, c)
	}
	for i, s := range r.Scene.Sounds.Slots {
		p.slots[s.Name] = i
	}
	p.voices = make([]Voice, len(r.Scene.Sounds.Slots))
	for i := range r.Scene.Actors {
		a := newPageActor(&r.Scene.Actors[i])
		p.actors = append(p.actors, a)
		p.byName[r.Scene.Actors[i].Name] = a
		for _, m := range a.members() {
			p.byName[m] = a
		}
	}
	p.Reset()
	return p
}

// Room is the page's room.
func (p *Page) Room() *RoomSpec { return p.room }

// Actor answers the named actor's runtime: a *Loop, *Alternating, *Selector,
// *Priority, *Bounce, *Cycle, *Target or *Training; nil for none.
func (p *Page) Actor(name string) any {
	if a, ok := p.byName[name]; ok {
		return a
	}
	return nil
}

// Ready reports whether the page has been entered since its last reset.
func (p *Page) Ready() bool { return p.ready }

// Active reports whether the page runs.
func (p *Page) Active() bool { return p.active }

// Reset silences every slot and returns the page to one never entered.
func (p *Page) Reset() {
	p.Silence()
	p.ready, p.active, p.started = false, false, false
	for _, c := range p.order {
		c.last = time.Time{}
	}
	for _, a := range p.actors {
		a.reset(p)
	}
}

// Enter returns the page to one just entered: every clock stamped, every
// host source reseeded, every actor entered in order, then the entry sounds.
func (p *Page) Enter() {
	p.Reset()
	now := p.host.Now()
	p.ready = true
	p.active = p.scene.EnterActive
	for _, c := range p.order {
		c.last = now.Add(time.Duration(c.spec.EntryMS) * time.Millisecond)
	}
	for _, r := range p.scene.Random {
		if r.Source == "host" {
			p.host.Reseed(r.Name)
		}
	}
	for _, a := range p.actors {
		a.enter(p, now)
	}
	p.runEntries(p.scene.Enter, now)
}

// SetActive is the page's lifecycle. A page asked to run that was never
// entered is entered first. A pause clears the active flag and silences the
// slots; a resume restamps the clocks the scene rebases and lets each actor
// rebase its own.
func (p *Page) SetActive(active bool) {
	if active && !p.ready {
		p.Enter()
	}
	if !active {
		p.active = false
		p.Silence()
		return
	}
	if p.active {
		return
	}
	now := p.host.Now()
	for _, c := range p.order {
		if c.spec.Rebase == "every" || c.spec.Rebase == "after-first" && p.started {
			c.last = now
		}
	}
	for _, a := range p.actors {
		a.resume(p, now)
	}
	p.started, p.active = true, true
}

// Advance is one paint of the page: each step group in order. A page whose
// art is absent does not advance.
func (p *Page) Advance() {
	if p.host.Art() == nil {
		return
	}
	now := p.host.Now()
	for _, g := range p.scene.Steps {
		if g.When == "active" && (!p.ready || !p.active) {
			continue
		}
		if c := p.clocks[g.Clock]; c != nil {
			if c.spec.BackStamps && now.Before(c.last) {
				c.last = now
				return
			}
			if !elapsed(now.Sub(c.last), time.Duration(*c.spec.PeriodMS)*time.Millisecond, c.spec.Compare) {
				continue
			}
			c.last = now
		}
		p.runEntries(g.Run, now)
	}
}

// Clock answers a clock's stamp.
func (p *Page) Clock(name string) time.Time {
	if c := p.clocks[name]; c != nil {
		return c.last
	}
	return time.Time{}
}

// SetClock stamps a clock.
func (p *Page) SetClock(name string, t time.Time) {
	if c := p.clocks[name]; c != nil {
		c.last = t
	}
}

func (p *Page) period(name string) time.Duration {
	if c := p.clocks[name]; c != nil && c.spec.PeriodMS != nil {
		return time.Duration(*c.spec.PeriodMS) * time.Millisecond
	}
	return 0
}

func (p *Page) compare(name string) string {
	if c := p.clocks[name]; c != nil {
		return c.spec.Compare
	}
	return ""
}

func (p *Page) runEntries(list []string, now time.Time) {
	for _, entry := range list {
		phase, name, _ := strings.Cut(entry, " ")
		if phase == "sound" {
			p.sound(name)
			continue
		}
		if a := p.byName[name]; a != nil {
			a.phase(p, phase, now)
		}
	}
}

// Event raises a named game event with its arguments; every actor bound to
// it runs the bound action. It answers whether any action changed state.
func (p *Page) Event(name string, args ...int) bool {
	changed := false
	for i, a := range p.actors {
		for _, on := range p.scene.Actors[i].On {
			if on.Event == name && a.event(p, on.Do, args) {
				changed = true
			}
		}
	}
	return changed
}

// Silence stops every tracked slot's voice.
func (p *Page) Silence() {
	for i := range p.voices {
		if p.voices[i] != nil {
			p.host.StopSound(p.voices[i])
			p.voices[i] = nil
		}
	}
}

// sound requests a slot's sound. A tracked slot holds one voice: a request
// while it plays is no request.
func (p *Page) sound(slot string) {
	i, ok := p.slots[slot]
	if !ok {
		return
	}
	spec := p.scene.Sounds.Slots[i]
	if spec.Untracked {
		p.host.PlaySound(spec.Source, spec.Key, spec.Loop)
		return
	}
	if v := p.voices[i]; v != nil {
		if v.Playing() {
			return
		}
		p.host.StopSound(v)
		p.voices[i] = nil
	}
	p.voices[i] = p.host.PlaySound(spec.Source, spec.Key, spec.Loop)
}

// Voice answers a slot's retained voice.
func (p *Page) Voice(slot string) Voice {
	if i, ok := p.slots[slot]; ok {
		return p.voices[i]
	}
	return nil
}

func (p *Page) frames(name string) []image.Image {
	return p.host.Art().Pictures(name)
}

// Complete reports whether an art entry holds exactly n frames, none nil.
func (p *Page) Complete(name string, n int) bool {
	frames := p.frames(name)
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

// amount evaluates a pick over the page's host sources: base plus a bounded
// draw of n, times Times; or, with Raw, base plus a raw draw r of [0,Raw)
// divided by Divide or scaled in the "scaled" form.
func (p *Page) amount(s PickSpec) int {
	if s.Raw > 0 {
		r := p.host.Draw(s.Draw, s.Raw)
		switch {
		case s.Form == "scaled":
			return s.Base + (r*s.N/(s.Raw-1))%s.N
		case s.Divide > 0:
			return s.Base + r/s.Divide
		}
		return s.Base + r
	}
	v := p.host.Draw(s.Draw, s.N)
	if s.Times > 0 {
		v *= s.Times
	}
	return s.Base + v
}

func (p *Page) wait(s PickSpec) time.Duration {
	return time.Duration(p.amount(s)) * time.Millisecond
}

// Paint draws the layers of one group in order onto dst, whose origin is the
// screen's. Paint changes no state.
func (p *Page) Paint(dst *image.RGBA, group string) {
	if p.host.Art() == nil {
		return
	}
	for _, layer := range p.scene.Layers {
		if layer.Group != group {
			continue
		}
		if layer.Actor != "" {
			if a := p.byName[layer.Actor]; a != nil {
				a.draw(p, dst, layer)
			}
			continue
		}
		place(dst, frameAt(p.frames(layer.Art), 0), layer)
	}
}

// place draws pic with its top-left at the layer's point, copied or
// composited over, inside the layer's clip when it has one.
func place(dst *image.RGBA, pic image.Image, layer LayerSpec) {
	if pic == nil {
		return
	}
	if layer.Clip == nil {
		put(dst, pic, layer)
		return
	}
	op := draw.Over
	if layer.Mode == "copy" {
		op = draw.Src
	}
	b := pic.Bounds()
	placed := b.Add(layer.At.Pt().Sub(b.Min))
	clip := placed.Intersect(layer.Clip.Rectangle())
	if clip.Empty() {
		return
	}
	draw.Draw(dst, clip, pic, b.Min.Add(clip.Min.Sub(placed.Min)), op)
}

// LoadSceneArt resolves a scene's art entries in order, as LoadArt does for a
// town: a required entry that fails fails the load, an optional one is left
// absent and named in Problems.
func LoadSceneArt(s *SceneSpec, src Loader) (*Art, error) {
	a := &Art{Frames: map[string][]image.Image{}}
	for _, spec := range s.Art {
		for _, entry := range expandArt(spec) {
			frames, err := loadEntry(spec, entry, src)
			if err != nil {
				if spec.Required {
					return nil, err
				}
				a.Problems = append(a.Problems, err.Error())
				continue
			}
			a.Frames[entry.name] = frames
		}
	}
	return a, nil
}
