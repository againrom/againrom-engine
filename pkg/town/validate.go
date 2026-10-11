package town

import (
	"fmt"
	"strings"
)

// Validate checks that every name a description uses resolves: art, actors,
// slots, hotspots, rooms, clocks, draw sources, programs, formats, phases,
// actions and modes. The square's scene and every room's scene pass the one
// scene check. The first failure is the error, naming the town and the
// offending value.
func (d *Description) Validate() error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("town %s: %s", d.Town, fmt.Sprintf(format, args...))
	}
	if d.Town == "" {
		return fmt.Errorf("town description: no town name")
	}
	if d.View.Size[0] <= 0 || d.View.Size[1] <= 0 {
		return fail("view size %v", d.View.Size)
	}
	if d.Clock.Compare != "greater" && d.Clock.Compare != "at-least" {
		return fail("clock compare %q", d.Clock.Compare)
	}
	if d.Clock.PeriodMS <= 0 {
		return fail("clock period %d ms", d.Clock.PeriodMS)
	}
	rooms := map[string]bool{}
	for _, r := range d.Rooms {
		rooms[r.Name] = true
	}
	if err := d.validateScene(d.SquareScene(), rooms, fail); err != nil {
		return err
	}
	for _, r := range d.Rooms {
		for _, s := range append(append([]Step{}, r.Enter...), r.Exit...) {
			if err := validateStep(s, "room "+r.Name, fail); err != nil {
				return err
			}
		}
		if err := d.validateRoom(r, rooms); err != nil {
			return err
		}
	}
	for _, s := range d.Square.Enter {
		if err := validateStep(s, "square", fail); err != nil {
			return err
		}
	}
	return nil
}

func validateStep(s Step, where string, fail func(string, ...any) error) error {
	if (s.Composer == "") == (s.Hook == "") {
		return fail("%s has a step with neither or both of composer and hook", where)
	}
	if s.Composer != "" && s.Composer != "reset" {
		return fail("%s has unknown composer step %q", where, s.Composer)
	}
	return nil
}

// validateRoom checks a room's music, tip and scene.
func (d *Description) validateRoom(r RoomSpec, rooms map[string]bool) error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("town %s: room %s: %s", d.Town, r.Name, fmt.Sprintf(format, args...))
	}
	if m := r.Music; m != nil {
		if (m.Track == "") == (len(m.Tracks) == 0) {
			return fail("music names neither or both of track and tracks")
		}
		if len(m.Tracks) > 0 && m.By == "" {
			return fail("music tracks without the value that chooses them")
		}
	}
	if t := r.Tip; t != nil && t.Rect.Rectangle().Empty() {
		return fail("tip rectangle %v is empty", t.Rect)
	}
	if r.Scene == nil {
		return nil
	}
	return d.validateScene(r.Scene, rooms, fail)
}

// phases are the steps each program runs.
var phases = map[string][]string{
	"episode": {"trigger", "advance"}, "pendulum": {"advance"}, "stepper": {"advance"},
	"driven": {"advance"}, "flock": {"settle", "arm", "advance"}, "families": {"advance", "schedule"},
	"loop": {"publish", "advance"}, "alternating": {"arm", "publish", "cue", "advance"},
	"selector": {"advance"}, "priority": {"arm", "advance"}, "bounce": {"advance"},
	"cycle": {"advance"}, "target": {}, "training": {"advance"},
}

// actions are the event actions each program answers; a priority actor
// answers the name of each of its states.
var actions = map[string][]string{
	"selector": {"select"}, "bounce": {"arm"}, "training": {"change"},
}

func listed(list []string, name string) bool {
	for _, n := range list {
		if n == name {
			return true
		}
	}
	return false
}

type sceneNames struct {
	random   map[string]bool
	art      map[string]bool
	slots    map[string]bool
	keyed    map[string]bool
	spots    map[string]bool
	clocks   map[string]*SceneClock
	programs map[string]string
	rooms    map[string]bool
	fail     func(string, ...any) error
}

func (n *sceneNames) needDraw(where, source string) error {
	if !n.random[source] {
		return n.fail("%s draws from unknown source %q", where, source)
	}
	return nil
}

func (n *sceneNames) needPick(where string, p *PickSpec) error {
	if p == nil {
		return n.fail("%s has no draw", where)
	}
	if err := n.needDraw(where, p.Draw); err != nil {
		return err
	}
	if p.Form == "scaled" && p.Raw > 0 && (p.N <= 0 || p.Raw <= 1) {
		return n.fail("%s scales a draw without n and raw", where)
	}
	return nil
}

func (n *sceneNames) needArt(where, name string) error {
	if !n.art[name] {
		return n.fail("%s names unknown art %q", where, name)
	}
	return nil
}

func (n *sceneNames) needSlot(where, slot string) error {
	if !n.slots[slot] {
		return n.fail("%s names unknown slot %q", where, slot)
	}
	return nil
}

func (n *sceneNames) needKeyedSlot(where, slot string) error {
	if err := n.needSlot(where, slot); err != nil {
		return err
	}
	if !n.keyed[slot] {
		return n.fail("%s requests slot %q, which has no key", where, slot)
	}
	return nil
}

func (d *Description) validateScene(s *SceneSpec, rooms map[string]bool, fail func(string, ...any) error) error {
	n := &sceneNames{
		random: map[string]bool{}, art: map[string]bool{}, slots: map[string]bool{}, keyed: map[string]bool{},
		spots: map[string]bool{}, clocks: map[string]*SceneClock{}, programs: map[string]string{},
		rooms: rooms, fail: fail,
	}
	switch s.Lifecycle {
	case "", LifecycleShown:
	default:
		return fail("unknown lifecycle %q", s.Lifecycle)
	}
	if s.AdvanceWhen != "" && s.AdvanceWhen != "active" {
		return fail("unknown advance-when %q", s.AdvanceWhen)
	}
	for _, r := range s.Random {
		switch r.Source {
		case "host":
		case "lcg":
			if r.Mask == 0 {
				return fail("generator %q has no mask", r.Name)
			}
		default:
			return fail("draw source %q has unknown kind %q", r.Name, r.Source)
		}
		n.random[r.Name] = true
	}
	for _, a := range s.Art {
		switch a.Format {
		case "picture", "series", "sprites":
		case "mask":
			if s.View.Size[0] <= 0 || s.View.Size[1] <= 0 {
				return fail("mask art %q in a scene without a view size", a.Name)
			}
		default:
			return fail("art %q has unknown format %q", a.Name, a.Format)
		}
		if a.Name == "" || a.Key == "" {
			return fail("art entry without name or key")
		}
		if len(a.Index) > 2 {
			return fail("art %q has %d indices", a.Name, len(a.Index))
		}
		n.art[a.Name] = true
	}
	for _, sl := range s.Sounds.Slots {
		n.slots[sl.Name] = true
		n.keyed[sl.Name] = sl.Key != ""
	}
	for _, h := range s.Hotspots {
		n.spots[h.Name] = true
		if h.Tip < 0 {
			return fail("hotspot %q has negative tip slot %d", h.Name, h.Tip)
		}
	}
	if s.Mask.Art != "" && !n.art[s.Mask.Art] {
		return fail("mask art %q", s.Mask.Art)
	}
	maskBytes := map[int]bool{}
	for _, m := range s.Mask.Bytes {
		if !n.spots[m.Hotspot] {
			return fail("mask byte %d names no hotspot %q", m.Byte, m.Hotspot)
		}
		if maskBytes[m.Byte] {
			return fail("mask byte %d is mapped twice", m.Byte)
		}
		maskBytes[m.Byte] = true
	}
	for i := range s.Clocks {
		c := &s.Clocks[i]
		if c.PeriodMS != nil && *c.PeriodMS <= 0 {
			return fail("clock %q period %d ms", c.Name, *c.PeriodMS)
		}
		if c.Compare != "greater" && c.Compare != "at-least" {
			return fail("clock %q compare %q", c.Name, c.Compare)
		}
		switch c.Rebase {
		case "", "every", "after-first":
		default:
			return fail("clock %q rebase %q", c.Name, c.Rebase)
		}
		n.clocks[c.Name] = c
	}
	for _, a := range s.Actors {
		if !listed(programs, a.Program) {
			return fail("actor %q has unknown program %q", a.Name, a.Program)
		}
		n.programs[a.Name] = a.Program
		for _, m := range a.Members {
			n.programs[m.Name] = a.Program
		}
		for _, v := range a.Variants {
			n.programs[v.Name] = a.Program
		}
	}
	for _, a := range s.Actors {
		if err := n.validateActor(a); err != nil {
			return err
		}
	}
	for _, h := range s.Hotspots {
		if err := n.checkOps("hotspot "+h.Name, h.Click); err != nil {
			return err
		}
		if err := n.checkOps("hotspot "+h.Name, h.Hover); err != nil {
			return err
		}
	}
	if err := n.checkOps("pointer", s.Pointer.Every); err != nil {
		return err
	}
	if err := n.checkOps("pointer", s.Pointer.Off); err != nil {
		return err
	}
	if err := n.checkEntries("enter", s.Enter); err != nil {
		return err
	}
	for i, g := range s.Steps {
		where := fmt.Sprintf("step group %d", i)
		if g.Clock != "" {
			c := n.clocks[g.Clock]
			if c == nil || c.PeriodMS == nil {
				return fail("%s is bound to clock %q, which has no period", where, g.Clock)
			}
		}
		if g.When != "" && g.When != "active" {
			return fail("%s has unknown when %q", where, g.When)
		}
		if err := n.checkEntries(where, g.Run); err != nil {
			return err
		}
	}
	for i, l := range s.Layers {
		where := fmt.Sprintf("layer %d", i)
		if (l.Art == "") == (l.Actor == "") {
			return fail("%s names neither or both of art and actor", where)
		}
		if l.Art != "" {
			if err := n.needArt(where, l.Art); err != nil {
				return err
			}
		}
		if _, ok := n.programs[l.Actor]; l.Actor != "" && !ok {
			return fail("%s names unknown actor %q", where, l.Actor)
		}
		if l.Mode != "copy" && l.Mode != "over" {
			return fail("%s has unknown mode %q", where, l.Mode)
		}
		if l.When != nil {
			if l.When.Hover != "" && !n.spots[l.When.Hover] {
				return fail("%s waits on unknown hotspot %q", where, l.When.Hover)
			}
			if _, ok := n.programs[l.When.Active]; l.When.Active != "" && !ok {
				return fail("%s waits on unknown actor %q", where, l.When.Active)
			}
		}
	}
	return nil
}

func (n *sceneNames) checkEntries(where string, list []string) error {
	for _, entry := range list {
		phase, name, _ := strings.Cut(entry, " ")
		if phase == "sound" {
			if err := n.needKeyedSlot(where, name); err != nil {
				return err
			}
			continue
		}
		program, ok := n.programs[name]
		if !ok {
			return n.fail("%s step %q names unknown actor", where, entry)
		}
		if !listed(phases[program], phase) {
			return n.fail("%s step %q: a %s has no phase %q", where, entry, program, phase)
		}
	}
	return nil
}

func (n *sceneNames) checkOps(where string, ops []Op) error {
	for _, op := range ops {
		switch {
		case op.Arm != "":
			if _, ok := n.programs[op.Arm]; !ok {
				return n.fail("%s arms unknown actor %q", where, op.Arm)
			}
			if op.Chance != nil {
				if err := n.needDraw(where, op.Chance.Draw); err != nil {
					return err
				}
			}
		case op.Latch != "":
			if err := n.needSlot(where, op.Slot); err != nil {
				return err
			}
			for _, s := range op.Stop {
				if err := n.needSlot(where, s); err != nil {
					return err
				}
			}
		case op.Drive != "":
			if n.programs[op.Drive] != "driven" {
				return n.fail("%s drives %q, which is no driven actor", where, op.Drive)
			}
		case len(op.ClearLatch) > 0, op.Hook != "", op.Menu != "":
		case op.Room != "":
			if !n.rooms[op.Room] {
				return n.fail("%s enters unknown room %q", where, op.Room)
			}
		case op.If != "":
			if err := n.checkOps(where, op.Then); err != nil {
				return err
			}
			if err := n.checkOps(where, op.Else); err != nil {
				return err
			}
		default:
			return n.fail("%s has an operation with no verb", where)
		}
	}
	return nil
}

func (n *sceneNames) validateActor(a ActorSpec) error {
	where := "actor " + a.Name
	fail := func(format string, args ...any) error {
		return n.fail("%s: %s", where, fmt.Sprintf(format, args...))
	}
	needClock := func(period bool) error {
		c := n.clocks[a.Clock]
		if c == nil || period && c.PeriodMS == nil {
			return fail("needs a clock with a period; %q is none", a.Clock)
		}
		return nil
	}
	for _, on := range a.On {
		known := listed(actions[a.Program], on.Do)
		if a.Program == "priority" {
			for _, s := range a.States {
				known = known || s.Name == on.Do
			}
		}
		if !known {
			return fail("a %s has no action %q", a.Program, on.Do)
		}
	}
	switch a.Program {
	case "episode":
		if err := n.needArt(where, a.Art); err != nil {
			return err
		}
		if a.End != "" && a.End != "rewind" && a.End != "hold" {
			return fail("unknown end %q", a.End)
		}
		if a.End == "hold" && (a.Frames <= 0 || a.RewindEvery <= 0) {
			return fail("a held episode needs frames and rewind-every")
		}
		if a.Trigger != nil {
			if err := n.needDraw(where, a.Trigger.Draw); err != nil {
				return err
			}
		}
		if c := a.StartSound; c != nil {
			if err := n.needSlot(where, c.Slot); err != nil {
				return err
			}
			for _, s := range c.Stop {
				if err := n.needSlot(where, s); err != nil {
					return err
				}
			}
		}
	case "pendulum":
		if a.Chance == nil || len(a.Members) == 0 {
			return fail("a pendulum needs members and a chance")
		}
		for _, m := range a.Members {
			if err := n.needArt(where, m.Art); err != nil {
				return err
			}
		}
		return n.needDraw(where, a.Chance.Draw)
	case "stepper", "driven":
		if err := n.needArt(where, a.Art); err != nil {
			return err
		}
		if a.Rest != "first" && a.Rest != "last" {
			return fail("unknown rest frame %q", a.Rest)
		}
		if a.HoldUnless != "" && a.Hold != "first" && a.Hold != "last" {
			return fail("unknown hold frame %q", a.Hold)
		}
		return n.needSlot(where, a.Slot)
	case "flock":
		if err := n.needArt(where, a.Art); err != nil {
			return err
		}
		if a.Wait == nil || a.Group == nil || a.Count == nil || a.GroupSize <= 0 || a.Groups <= 0 || a.Frames <= 0 {
			return fail("a flock needs groups, group-size, frames, wait, group and count")
		}
		for _, s := range []string{a.Wait.Draw, a.Group.Draw, a.Count.Draw} {
			if err := n.needDraw(where, s); err != nil {
				return err
			}
		}
		return n.needSlot(where, a.Slot)
	case "families":
		return n.validateFamilies(a, where, fail)
	case "loop":
		if a.Frames <= 0 || a.Loop <= 0 || a.Loop > a.Frames {
			return fail("needs frames and a loop no longer than them")
		}
		return n.needArt(where, a.Art)
	case "alternating":
		if err := needClock(true); err != nil {
			return err
		}
		if err := n.needPick(where, a.Delay); err != nil {
			return err
		}
		if len(a.States) == 0 {
			return fail("has no states")
		}
		for _, s := range a.States {
			if s.When != "odd" && s.When != "even" {
				return fail("state %q has unknown parity %q", s.Name, s.When)
			}
			if s.Motion != "forward" && s.Motion != "ping-pong" {
				return fail("state %q has unknown motion %q", s.Name, s.Motion)
			}
			if err := n.checkState(where, s); err != nil {
				return err
			}
		}
	case "selector":
		if len(a.Members) == 0 || a.Frames <= 0 {
			return fail("needs members and frames")
		}
		for _, m := range a.Members {
			if err := n.needArt(where, m.Art); err != nil {
				return err
			}
		}
		for _, i := range []int{a.LoopAt, a.LoopTo, a.StopAt, a.Release} {
			if i < 0 || i > a.Frames {
				return fail("index %d is outside its %d frames", i, a.Frames)
			}
		}
	case "priority":
		if err := needClock(false); err != nil {
			return err
		}
		if err := n.needArt(where, a.Art); err != nil {
			return err
		}
		if err := n.needPick(where, a.Delay); err != nil {
			return err
		}
		if len(a.States) == 0 {
			return fail("has no states")
		}
		for _, s := range a.States {
			if s.Steps <= 0 {
				return fail("state %q has no steps", s.Name)
			}
			if err := n.checkState(where, s); err != nil {
				return err
			}
		}
	case "bounce":
		if a.Frames < 2 {
			return fail("needs at least two frames")
		}
		return n.needArt(where, a.Art)
	case "cycle":
		if a.Frames <= 0 || a.Wait == nil || a.Wait.BaseMS <= 0 {
			return fail("needs frames and a wait")
		}
		for _, order := range a.Order {
			if len(order) != a.Frames {
				return fail("order %v does not map its %d counts", order, a.Frames)
			}
		}
	case "target":
		if a.Wait == nil || a.Wait.BaseMS <= 0 || len(a.Endpoints) == 0 || a.Frames <= 0 {
			return fail("needs frames, endpoints and a wait")
		}
		for _, e := range a.Endpoints {
			if e < 0 || e >= a.Frames {
				return fail("endpoint %d is outside its %d frames", e, a.Frames)
			}
		}
		return n.needArt(where, a.Art)
	case "training":
		if err := needClock(true); err != nil {
			return err
		}
		if n.programs[a.Column] != "target" {
			return fail("walks %q, which is no target actor", a.Column)
		}
		if err := n.needKeyedSlot(where, a.Slot); err != nil {
			return err
		}
		if err := n.needPick(where, a.Delay); err != nil {
			return err
		}
		if err := n.needPick(where, a.HoldPick); err != nil {
			return err
		}
		if len(a.Variants) == 0 {
			return fail("has no variants")
		}
		for _, v := range a.Variants {
			if v.TransitionCount <= 0 || v.IdleCount <= 0 {
				return fail("variant %q needs transition and idle counts", v.Name)
			}
			if f := v.ReverseFrom; f != nil && (*f < 0 || *f >= v.IdleCount) {
				return fail("variant %q reverses from %d of %d frames", v.Name, *f, v.IdleCount)
			}
			if err := n.needArt(where, v.Transition); err != nil {
				return err
			}
			if err := n.needArt(where, v.Idle); err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *sceneNames) validateFamilies(a ActorSpec, where string, fail func(string, ...any) error) error {
	names := map[string]bool{}
	for _, m := range a.Members {
		if err := n.needArt(where, m.Art); err != nil {
			return err
		}
		if m.Mode != "episode" && m.Mode != "loop" {
			return fail("member %q has unknown mode %q", m.Name, m.Mode)
		}
		if m.Mode == "episode" && (m.Later == nil || m.SheetPick == nil) {
			return fail("member %q needs later and sheet-pick", m.Name)
		}
		for _, c := range m.FrameCues {
			if err := n.needSlot(where, c.Slot); err != nil {
				return err
			}
		}
		names[m.Name] = true
	}
	for _, s := range a.Entry {
		for _, name := range []string{s.Position, s.Wait, s.Unequal} {
			if name != "" && !names[name] {
				return fail("entry names unknown member %q", name)
			}
		}
		if s.Position != "" && s.Pick == nil || s.Wait != "" && s.After == nil {
			return fail("entry step without its draw")
		}
	}
	for _, name := range append(append([]string{}, a.StepOrder...), a.PaintOrder...) {
		if !names[name] {
			return fail("order names unknown member %q", name)
		}
	}
	for _, name := range a.PaintOrder {
		for _, m := range a.Members {
			if m.Name == name && m.Mode != "episode" {
				return fail("paint-order names %s member %q; only an episode member is scheduled", m.Mode, name)
			}
		}
	}
	picks := map[string]*PickSpec{}
	for _, s := range a.Entry {
		if s.Unequal != "" {
			other := picks[s.Unequal]
			if s.Position == "" || other == nil {
				return fail("entry %q is drawn unequal to %q, which no earlier entry positions", s.Position, s.Unequal)
			}
			if onlyEqual(*s.Pick, *other) {
				return fail("entry %q can only draw the value %q holds", s.Position, s.Unequal)
			}
		}
		if s.Position != "" {
			picks[s.Position] = s.Pick
		}
	}
	return nil
}

func (n *sceneNames) checkState(where string, s StateSpec) error {
	if err := n.needArt(where, s.Art); err != nil {
		return err
	}
	for _, list := range [][]string{s.StartSounds, s.EndSounds} {
		for _, slot := range list {
			if err := n.needKeyedSlot(where, slot); err != nil {
				return err
			}
		}
	}
	for _, c := range s.FrameSounds {
		if err := n.needKeyedSlot(where, c.Slot); err != nil {
			return err
		}
	}
	return nil
}

// pickRange is the closed range of values a pick can answer.
func pickRange(p PickSpec) (lo, hi int) {
	if p.N <= 1 {
		return p.Base, p.Base
	}
	return p.Base, p.Base + p.N - 1
}

// onlyEqual reports whether a redraw of p until it differs from a value o
// answered can never end: p has one value and o can answer it.
func onlyEqual(p, o PickSpec) bool {
	plo, phi := pickRange(p)
	olo, ohi := pickRange(o)
	return plo == phi && plo >= olo && plo <= ohi
}

// checkVocabulary refuses a hook, condition, value or event name the game
// does not answer.
func (d *Description) checkVocabulary(vocab Vocabulary) error {
	lists := map[string][]string{"hook": vocab.Hooks, "condition": vocab.Conditions, "value": vocab.Values, "event": vocab.Events}
	for _, u := range d.uses() {
		if !vocab.has(lists[u.kind], u.name) {
			return fmt.Errorf("town %s: %s names unknown %s %q", d.Town, u.where, u.kind, u.name)
		}
	}
	return nil
}

type nameUse struct {
	where, name, kind string
}

// uses lists every hook, condition, value and event name the description
// uses. An event a scene's own actor raises is the scene's, not the game's.
func (d *Description) uses() []nameUse {
	var out []nameUse
	use := func(kind, where, name string) {
		if name != "" {
			out = append(out, nameUse{where: where, name: name, kind: kind})
		}
	}
	var ops func(where string, list []Op)
	ops = func(where string, list []Op) {
		for _, op := range list {
			use("hook", where, op.Hook)
			use("condition", where, op.If)
			use("condition", where, op.Unless)
			ops(where, op.Then)
			ops(where, op.Else)
		}
	}
	scene := func(prefix string, s *SceneSpec) {
		for _, h := range s.Hotspots {
			ops(prefix+"hotspot "+h.Name, h.Click)
			ops(prefix+"hotspot "+h.Name, h.Hover)
		}
		ops(prefix+"pointer", s.Pointer.Every)
		ops(prefix+"pointer", s.Pointer.Off)
		raised := map[string]bool{}
		for _, a := range s.Actors {
			raised[a.Raise] = true
		}
		for _, a := range s.Actors {
			where := prefix + "actor " + a.Name
			use("condition", where, a.HoldUnless)
			use("value", where, a.Value)
			use("value", where, a.Selected)
			for _, on := range a.On {
				if !raised[on.Event] {
					use("event", where, on.Event)
				}
			}
		}
	}
	scene("", d.SquareScene())
	for _, s := range d.Square.Enter {
		use("hook", "square", s.Hook)
	}
	for _, r := range d.Rooms {
		for _, s := range append(append([]Step{}, r.Enter...), r.Exit...) {
			use("hook", "room "+r.Name, s.Hook)
		}
		if r.Music != nil {
			use("value", "room "+r.Name+" music", r.Music.By)
		}
		if r.Scene != nil {
			scene("room "+r.Name+" ", r.Scene)
		}
	}
	use("condition", "save", d.Save.AdmittedWhen)
	return out
}
