package town

import (
	"image"
	"strings"
	"time"
)

// Voice is one retained sound the Host started.
type Voice interface {
	Playing() bool
	Stop()
}

// Host is what a game gives the composer: its art, clock, draws, conditions,
// sound and campaign hooks. Condition and Hook answer the names a description
// uses; the composer gives the names no meaning of its own.
type Host interface {
	Art() *Art
	Now() time.Time
	// Draw is a bounded draw in [0,n) from the named host source.
	Draw(source string, n int) int
	// Seed seeds the process-scoped generators the description keeps.
	Seed() int64
	Condition(name string) bool
	// PlaySound starts key for a slot's source and answers its voice, or nil
	// when the key cannot play.
	PlaySound(source, key string) Voice
	StopSound(Voice)
	// StartLoop starts the view's entry loop and reports whether it started.
	StartLoop(key string) bool
	StopLoop(key string)
	// LeaveSquare releases the square's own audio scope.
	LeaveSquare()
	// Hook runs one named campaign hook for the named room.
	Hook(name, room string)
}

// Process is the state that outlives every reset of a view: the clock's last
// admitted step, the description's generators, the flocks' wait latches and
// the held episodes' end counters. A game keeps one for the life of the
// process; its zero value is ready.
type Process struct {
	last  time.Time
	lcg   map[string]*lcg
	waits map[string]*Wait
	ends  map[string]int
}

func (p *Process) init() {
	if p.lcg == nil {
		p.lcg, p.waits, p.ends = map[string]*lcg{}, map[string]*Wait{}, map[string]int{}
	}
}

// View is one town view built from a description. Reset returns it to the
// state of a view just built; the process state survives.
type View struct {
	desc *Description
	host Host
	proc *Process

	ready, present, active bool
	hover                  *HotspotSpec
	latches                map[string]bool
	voices                 []Voice
	loopOn                 bool

	actors []actor
	byName map[string]actor
	slots  map[string]int
	spots  map[string]*HotspotSpec
	rooms  map[string]*RoomSpec
}

// NewView builds a view of d over host, keeping its process-scoped state in
// proc; a nil proc gives the view its own. d must have passed Validate.
func NewView(d *Description, host Host, proc *Process) *View {
	if proc == nil {
		proc = &Process{}
	}
	proc.init()
	v := &View{
		desc: d, host: host, proc: proc,
		byName: map[string]actor{}, slots: map[string]int{},
		spots: map[string]*HotspotSpec{}, rooms: map[string]*RoomSpec{},
	}
	for i, s := range d.Sounds.Slots {
		v.slots[s.Name] = i
	}
	for i := range d.Hotspots {
		v.spots[d.Hotspots[i].Name] = &d.Hotspots[i]
	}
	for i := range d.Rooms {
		v.rooms[d.Rooms[i].Name] = &d.Rooms[i]
	}
	for i := range d.Actors {
		a := newActor(&d.Actors[i])
		v.actors = append(v.actors, a)
		v.byName[d.Actors[i].Name] = a
		for _, m := range a.members() {
			v.byName[m] = a
		}
	}
	v.voices = make([]Voice, len(d.Sounds.Slots))
	v.Reset()
	return v
}

// Actor answers the runtime of the named actor or actor member: an
// *Episode, *Pendulum, *Stepper, *Driven, *Flock or *Families; nil for an
// unknown name.
func (v *View) Actor(name string) any {
	if a, ok := v.byName[name]; ok {
		return a
	}
	return nil
}

// Description is the description the view was built from.
func (v *View) Description() *Description { return v.desc }

// Reset stops the entry loop and every slot, then returns every actor, latch
// and hover to the state of a view just built. It reads no art: an actor
// whose rest frame depends on its art takes it when the view is next made
// ready.
func (v *View) Reset() {
	v.stopLoop()
	v.silence()
	v.ready, v.present, v.active = false, false, false
	v.hover = nil
	v.latches = map[string]bool{}
	for _, a := range v.actors {
		a.reset(v)
	}
}

// Ready reports whether the view has been activated since its last reset.
func (v *View) Ready() bool { return v.ready }

// Present reports whether the square is entered and visible.
func (v *View) Present() bool { return v.present }

// Active reports whether the view runs: on the square, focused and unpaused.
func (v *View) Active() bool { return v.active }

// SetActive is the view's lifecycle. onSquare says whether the square is the
// screen shown; active is ignored off the square. Leaving the square hides
// entry-scoped actors; entering it runs every actor's entry; a change of
// activity starts or stops the entry loop and, on pause, clears the hover and
// every slot.
func (v *View) SetActive(active, onSquare bool) {
	active = active && onSquare
	if !active && !v.ready {
		return
	}
	if !v.ready {
		v.ready = true
		for _, a := range v.actors {
			a.prepare(v)
		}
	}
	if !onSquare {
		v.host.LeaveSquare()
		v.present = false
		for _, a := range v.actors {
			a.leave(v)
		}
	}
	if onSquare && !v.present {
		now := v.host.Now()
		v.present = true
		for _, a := range v.actors {
			a.enter(v, now)
		}
	}
	if v.active != active {
		v.active = active
		if !active {
			v.hover = nil
			v.silence()
			v.stopLoop()
		} else {
			v.startLoop()
		}
	}
}

// HotspotAt answers the hotspot under p, a view-relative point, read from the
// mask; nil off the mask or over an unmapped byte.
func (v *View) HotspotAt(p image.Point) *HotspotSpec {
	art := v.host.Art()
	if art == nil || art.Mask == nil || !p.In(art.Mask.Bounds()) {
		return nil
	}
	b := int(art.Mask.ColorIndexAt(p.X, p.Y))
	for _, m := range v.desc.Mask.Bytes {
		if m.Byte == b {
			return v.spots[m.Hotspot]
		}
	}
	return nil
}

// Hotspot answers the named hotspot.
func (v *View) Hotspot(name string) *HotspotSpec { return v.spots[name] }

// HotspotForRow answers the hotspot the list fallback shows at row i.
func (v *View) HotspotForRow(i int) *HotspotSpec {
	for k := range v.desc.Hotspots {
		if r := v.desc.Hotspots[k].Row; r != nil && *r == i {
			return &v.desc.Hotspots[k]
		}
	}
	return nil
}

// Hover answers the hovered hotspot, nil for none.
func (v *View) Hover() *HotspotSpec { return v.hover }

// Pointer is one delivered pointer update at p. The every-update reactions run
// first, then the hovered hotspot's or the off-hotspot reactions.
func (v *View) Pointer(p image.Point) {
	v.hover = v.HotspotAt(p)
	v.run(v.desc.Pointer.Every)
	if v.hover == nil {
		v.run(v.desc.Pointer.Off)
		return
	}
	v.run(v.hover.Hover)
}

// Click runs a hotspot's click program and answers the menu it names, if any.
func (v *View) Click(h *HotspotSpec) string {
	if h == nil {
		return ""
	}
	return v.run(h.Click)
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

func (v *View) run(ops []Op) string {
	menu := ""
	for _, op := range ops {
		switch {
		case op.Arm != "":
			if op.Chance == nil || v.chance(*op.Chance) {
				if a := v.byName[op.Arm]; a != nil {
					a.arm(v, op.Arm)
				}
			}
		case op.Latch != "":
			if !v.latches[op.Latch] {
				for _, s := range op.Stop {
					v.stop(s)
				}
				v.sound(op.Slot, op.Sound)
				v.latches[op.Latch] = true
				for _, c := range op.Clear {
					v.latches[c] = false
				}
			}
		case op.Drive != "":
			if op.Unless == "" || !v.host.Condition(op.Unless) {
				if a := v.byName[op.Drive]; a != nil {
					a.drive(v, op.Dir)
				}
			}
		case len(op.ClearLatch) > 0:
			for _, c := range op.ClearLatch {
				v.latches[c] = false
			}
		case op.If != "":
			branch := op.Else
			if v.host.Condition(op.If) {
				branch = op.Then
			}
			if m := v.run(branch); m != "" {
				menu = m
			}
		case op.Room != "":
			v.EnterRoom(op.Room)
		case op.Hook != "":
			v.host.Hook(op.Hook, "")
		case op.Menu != "":
			menu = op.Menu
		}
	}
	return menu
}

// Latch reports one hover latch.
func (v *View) Latch(name string) bool { return v.latches[name] }

// Advance is the paint clock: called once per paint of the square. The first
// paint stamps the clock. Every later paint runs the before phases, then the
// admitted phases when the clock admits a step, then the after phases.
func (v *View) Advance() {
	if !v.active || v.host.Art() == nil {
		return
	}
	now := v.host.Now()
	if v.proc.last.IsZero() {
		v.proc.last = now
		return
	}
	v.phases(v.desc.Step.Before, now)
	if !elapsed(now.Sub(v.proc.last), time.Duration(v.desc.Clock.PeriodMS)*time.Millisecond, v.desc.Clock.Compare) {
		v.phases(v.desc.Step.After, now)
		return
	}
	v.proc.last = now
	v.phases(v.desc.Step.Admitted, now)
	v.phases(v.desc.Step.After, now)
}

func elapsed(since, wait time.Duration, compare string) bool {
	if compare == "at-least" {
		return since >= wait
	}
	return since > wait
}

func (v *View) phases(list []string, now time.Time) {
	for _, entry := range list {
		phase, name, _ := strings.Cut(entry, " ")
		if a := v.byName[name]; a != nil {
			a.phase(v, phase, name, now)
		}
	}
}

// EnterRoom runs a room's entry steps in order.
func (v *View) EnterRoom(name string) {
	room := v.rooms[name]
	if room == nil {
		return
	}
	v.steps(room.Enter, room.Name)
}

// LeaveRoom runs a room's exit steps, then the steps that return to the square.
func (v *View) LeaveRoom(name string) {
	if room := v.rooms[name]; room != nil {
		v.steps(room.Exit, room.Name)
	}
	v.EnterSquare()
}

// EnterSquare runs the steps that return to the square.
func (v *View) EnterSquare() {
	v.steps(v.desc.Square.Enter, v.desc.Square.Name)
}

// Room answers the named room.
func (v *View) Room(name string) *RoomSpec { return v.rooms[name] }

func (v *View) steps(list []Step, room string) {
	for _, s := range list {
		switch {
		case s.Composer == "reset":
			v.Reset()
		case s.Hook != "":
			v.host.Hook(s.Hook, room)
		}
	}
}

// SaveAdmitted answers the description's save admission condition.
func (v *View) SaveAdmitted() bool {
	return v.desc.Save.AdmittedWhen == "" || v.host.Condition(v.desc.Save.AdmittedWhen)
}

// sound requests key on a slot. A slot whose sound still plays takes no new
// request; a finished one is stopped first.
func (v *View) sound(slot, key string) {
	i, ok := v.slots[slot]
	if !ok {
		return
	}
	if vc := v.voices[i]; vc != nil {
		if vc.Playing() {
			return
		}
		v.stopIndex(i)
	}
	v.voices[i] = v.host.PlaySound(v.desc.Sounds.Slots[i].Source, key)
}

func (v *View) stop(slot string) {
	if i, ok := v.slots[slot]; ok {
		v.stopIndex(i)
	}
}

func (v *View) stopIndex(i int) {
	if v.voices[i] != nil {
		v.host.StopSound(v.voices[i])
		v.voices[i] = nil
	}
}

func (v *View) silence() {
	for i := range v.voices {
		v.stopIndex(i)
	}
}

// Voice answers a slot's retained voice.
func (v *View) Voice(slot string) Voice {
	if i, ok := v.slots[slot]; ok {
		return v.voices[i]
	}
	return nil
}

func (v *View) startLoop() {
	if v.loopOn || v.desc.Sounds.Loop == "" {
		return
	}
	v.loopOn = v.host.StartLoop(v.desc.Sounds.Loop)
}

func (v *View) stopLoop() {
	if !v.loopOn {
		return
	}
	v.host.StopLoop(v.desc.Sounds.Loop)
	v.loopOn = false
}

// LoopOn reports whether the entry loop runs.
func (v *View) LoopOn() bool { return v.loopOn }

func (v *View) random(name string) *RandomSpec {
	for i := range v.desc.Random {
		if v.desc.Random[i].Name == name {
			return &v.desc.Random[i]
		}
	}
	return nil
}

// draw is one value in the named arithmetic form from the named source:
// a host source answers its own bounded draw; a generator answers
// "scaled" (r*n/mask) % n or "masked" (r*n/mask) & (n-1).
func (v *View) draw(source, form string, n int) int {
	r := v.random(source)
	if r == nil || n <= 0 {
		return 0
	}
	if r.Source == "host" {
		return v.host.Draw(source, n)
	}
	g := v.generator(r)
	raw := g.next(v.host.Seed())
	scaled := raw * n / int(r.Mask)
	if form == "masked" {
		return scaled & (n - 1)
	}
	return scaled % n
}

// Pick answers one draw of the named source in the named arithmetic form.
func (v *View) Pick(source, form string, n int) int { return v.draw(source, form, n) }

func (v *View) generator(r *RandomSpec) *lcg {
	g := v.proc.lcg[r.Name]
	if g == nil {
		g = &lcg{spec: r}
		v.proc.lcg[r.Name] = g
	}
	return g
}

// RawDraw answers a generator's replacement raw values, nil when it runs.
func (v *View) RawDraw(source string) func() int {
	if r := v.random(source); r != nil && r.Source != "host" {
		return v.generator(r).raw
	}
	return nil
}

// SetRawDraw replaces a generator's raw values; tests pin a sequence with it.
func (v *View) SetRawDraw(source string, raw func() int) {
	if r := v.random(source); r != nil && r.Source != "host" {
		v.generator(r).raw = raw
	}
}

func (v *View) chance(c ChanceSpec) bool {
	return v.draw(c.Draw, "", c.N) > c.Above
}

func (v *View) pick(p PickSpec) int {
	return p.Base + v.draw(p.Draw, p.Form, p.N)
}

func (v *View) wait(w WaitSpec) time.Duration {
	return time.Duration(w.BaseMS+v.draw(w.Draw, w.Form, w.SpanMS)) * time.Millisecond
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
func (v *View) WaitLatch(name string) *Wait { return v.proc.WaitLatch(name) }

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
func (v *View) EndCount(name string) int { return v.proc.ends[name] }

// SetEndCount sets a held episode's end counter.
func (v *View) SetEndCount(name string, n int) { v.proc.ends[name] = n }

// ResetDraw returns a generator to unseeded; its next draw seeds it again.
func (v *View) ResetDraw(source string) { delete(v.proc.lcg, source) }

// ClockLast is the clock's last admitted step.
func (v *View) ClockLast() time.Time { return v.proc.last }

// SetClockLast sets the clock's last admitted step; the zero time makes the
// next paint only stamp the clock.
func (v *View) SetClockLast(t time.Time) { v.proc.last = t }

func (v *View) frames(name string) []image.Image {
	return v.host.Art().Pictures(name)
}
