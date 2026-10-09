package town

import (
	"fmt"
	"strings"
)

// validateRoom checks a room's scene, music and tip.
func (d *Description) validateRoom(r RoomSpec) error {
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
	return validateScene(r.Scene, fail)
}

// pagePhases are the steps each scene program runs.
var pagePhases = map[string][]string{
	"loop": {"publish", "advance"}, "alternating": {"arm", "publish", "cue", "advance"},
	"selector": {"advance"}, "priority": {"arm", "advance"}, "bounce": {"advance"},
	"cycle": {"advance"}, "target": {}, "training": {"advance"},
}

// pageActions are the event actions each scene program answers; a priority
// actor answers the name of each of its states.
var pageActions = map[string][]string{
	"selector": {"select"}, "bounce": {"arm"}, "training": {"change"},
}

func validateScene(s *SceneSpec, fail func(string, ...any) error) error {
	random := map[string]bool{}
	for _, r := range s.Random {
		if r.Source != "host" {
			return fail("scene draw source %q must be a host source", r.Name)
		}
		random[r.Name] = true
	}
	art := map[string]bool{}
	for _, a := range s.Art {
		if a.Format != "picture" && a.Format != "series" {
			return fail("scene art %q has unknown format %q", a.Name, a.Format)
		}
		if a.Name == "" || a.Key == "" || len(a.Index) > 0 {
			return fail("scene art entry %q needs a name and a key and takes no index", a.Name)
		}
		art[a.Name] = true
	}
	needArt := func(where, name string) error {
		if !art[name] {
			return fail("%s names unknown art %q", where, name)
		}
		return nil
	}
	slots := map[string]bool{}
	for _, sl := range s.Sounds.Slots {
		if sl.Key == "" {
			return fail("scene slot %q has no key", sl.Name)
		}
		slots[sl.Name] = true
	}
	needSlot := func(where, name string) error {
		if !slots[name] {
			return fail("%s names unknown slot %q", where, name)
		}
		return nil
	}
	clocks := map[string]*PageClock{}
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
		clocks[c.Name] = c
	}
	needDraw := func(where string, p *PickSpec) error {
		if p == nil {
			return fail("%s has no draw", where)
		}
		if !random[p.Draw] {
			return fail("%s draws from unknown source %q", where, p.Draw)
		}
		if p.Form == "scaled" && (p.N <= 0 || p.Raw <= 1) {
			return fail("%s scales a draw without n and raw", where)
		}
		return nil
	}
	programs := map[string]string{}
	for _, a := range s.Actors {
		where := "actor " + a.Name
		known := false
		for _, p := range pagePrograms {
			known = known || p == a.Program
		}
		if !known {
			return fail("%s has unknown program %q", where, a.Program)
		}
		programs[a.Name] = a.Program
		for _, m := range a.Members {
			programs[m.Name] = a.Program
		}
		for _, v := range a.Variants {
			programs[v.Name] = a.Program
		}
	}
	for _, a := range s.Actors {
		if err := validatePageActor(a, programs, clocks, needArt, needSlot, needDraw, fail); err != nil {
			return err
		}
	}
	checkEntries := func(where string, list []string) error {
		for _, entry := range list {
			phase, name, _ := strings.Cut(entry, " ")
			if phase == "sound" {
				if err := needSlot(where, name); err != nil {
					return err
				}
				continue
			}
			program, ok := programs[name]
			if !ok {
				return fail("%s step %q names unknown actor", where, entry)
			}
			found := false
			for _, p := range pagePhases[program] {
				found = found || p == phase
			}
			if !found {
				return fail("%s step %q: a %s has no phase %q", where, entry, program, phase)
			}
		}
		return nil
	}
	if err := checkEntries("enter", s.Enter); err != nil {
		return err
	}
	for i, g := range s.Steps {
		where := fmt.Sprintf("step group %d", i)
		if g.Clock != "" {
			c := clocks[g.Clock]
			if c == nil || c.PeriodMS == nil {
				return fail("%s is bound to clock %q, which has no period", where, g.Clock)
			}
		}
		if g.When != "" && g.When != "active" {
			return fail("%s has unknown when %q", where, g.When)
		}
		if err := checkEntries(where, g.Run); err != nil {
			return err
		}
	}
	for i, l := range s.Layers {
		where := fmt.Sprintf("layer %d", i)
		if l.Group == "" {
			return fail("%s has no group", where)
		}
		if (l.Art == "") == (l.Actor == "") {
			return fail("%s names neither or both of art and actor", where)
		}
		if l.Art != "" {
			if err := needArt(where, l.Art); err != nil {
				return err
			}
		}
		if _, ok := programs[l.Actor]; l.Actor != "" && !ok {
			return fail("%s names unknown actor %q", where, l.Actor)
		}
		if l.Mode != "copy" && l.Mode != "over" {
			return fail("%s has unknown mode %q", where, l.Mode)
		}
		if l.When != nil {
			return fail("%s: a scene layer takes no condition", where)
		}
	}
	return nil
}

func validatePageActor(a ActorSpec, programs map[string]string, clocks map[string]*PageClock,
	needArt, needSlot func(string, string) error, needDraw func(string, *PickSpec) error,
	fail func(string, ...any) error) error {
	where := "actor " + a.Name
	needClock := func(period bool) error {
		c := clocks[a.Clock]
		if c == nil || period && c.PeriodMS == nil {
			return fail("%s needs a clock with a period; %q is none", where, a.Clock)
		}
		return nil
	}
	for _, on := range a.On {
		known := false
		for _, do := range pageActions[a.Program] {
			known = known || do == on.Do
		}
		if a.Program == "priority" {
			for _, s := range a.States {
				known = known || s.Name == on.Do
			}
		}
		if !known {
			return fail("%s: a %s has no action %q", where, a.Program, on.Do)
		}
	}
	switch a.Program {
	case "loop":
		if a.Frames <= 0 || a.Loop <= 0 || a.Loop > a.Frames {
			return fail("%s needs frames and a loop no longer than them", where)
		}
		return needArt(where, a.Art)
	case "alternating":
		if err := needClock(true); err != nil {
			return err
		}
		if err := needDraw(where, a.Delay); err != nil {
			return err
		}
		if len(a.States) == 0 {
			return fail("%s has no states", where)
		}
		for _, s := range a.States {
			if s.When != "odd" && s.When != "even" {
				return fail("%s state %q has unknown parity %q", where, s.Name, s.When)
			}
			if s.Motion != "forward" && s.Motion != "ping-pong" {
				return fail("%s state %q has unknown motion %q", where, s.Name, s.Motion)
			}
			if err := checkState(where, s, needArt, needSlot); err != nil {
				return err
			}
		}
	case "selector":
		if len(a.Members) == 0 || a.Frames <= 0 {
			return fail("%s needs members and frames", where)
		}
		for _, m := range a.Members {
			if err := needArt(where, m.Art); err != nil {
				return err
			}
		}
		for _, n := range []int{a.LoopAt, a.LoopTo, a.StopAt, a.Release} {
			if n < 0 || n > a.Frames {
				return fail("%s index %d is outside its %d frames", where, n, a.Frames)
			}
		}
	case "priority":
		if err := needClock(false); err != nil {
			return err
		}
		if err := needArt(where, a.Art); err != nil {
			return err
		}
		if err := needDraw(where, a.Delay); err != nil {
			return err
		}
		if len(a.States) == 0 {
			return fail("%s has no states", where)
		}
		for _, s := range a.States {
			if s.Steps <= 0 {
				return fail("%s state %q has no steps", where, s.Name)
			}
			if err := checkState(where, s, needArt, needSlot); err != nil {
				return err
			}
		}
	case "bounce":
		if a.Frames < 2 {
			return fail("%s needs at least two frames", where)
		}
		return needArt(where, a.Art)
	case "cycle":
		if a.Frames <= 0 || a.Wait == nil || a.Wait.BaseMS <= 0 {
			return fail("%s needs frames and a wait", where)
		}
		for _, order := range a.Order {
			if len(order) != a.Frames {
				return fail("%s order %v does not map its %d counts", where, order, a.Frames)
			}
		}
	case "target":
		if a.Wait == nil || a.Wait.BaseMS <= 0 || len(a.Endpoints) == 0 || a.Frames <= 0 {
			return fail("%s needs frames, endpoints and a wait", where)
		}
		for _, e := range a.Endpoints {
			if e < 0 || e >= a.Frames {
				return fail("%s endpoint %d is outside its %d frames", where, e, a.Frames)
			}
		}
		return needArt(where, a.Art)
	case "training":
		if err := needClock(true); err != nil {
			return err
		}
		if programs[a.Column] != "target" {
			return fail("%s walks %q, which is no target actor", where, a.Column)
		}
		if err := needSlot(where, a.Slot); err != nil {
			return err
		}
		if err := needDraw(where, a.Delay); err != nil {
			return err
		}
		if err := needDraw(where, a.HoldPick); err != nil {
			return err
		}
		if len(a.Variants) == 0 {
			return fail("%s has no variants", where)
		}
		for _, v := range a.Variants {
			if v.TransitionCount <= 0 || v.IdleCount <= 0 {
				return fail("%s variant %q needs transition and idle counts", where, v.Name)
			}
			if f := v.ReverseFrom; f != nil && (*f < 0 || *f >= v.IdleCount) {
				return fail("%s variant %q reverses from %d of %d frames", where, v.Name, *f, v.IdleCount)
			}
			if err := needArt(where, v.Transition); err != nil {
				return err
			}
			if err := needArt(where, v.Idle); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkState(where string, s StateSpec, needArt, needSlot func(string, string) error) error {
	if err := needArt(where, s.Art); err != nil {
		return err
	}
	for _, list := range [][]string{s.StartSounds, s.EndSounds} {
		for _, slot := range list {
			if err := needSlot(where, slot); err != nil {
				return err
			}
		}
	}
	for _, c := range s.FrameSounds {
		if err := needSlot(where, c.Slot); err != nil {
			return err
		}
	}
	return nil
}
