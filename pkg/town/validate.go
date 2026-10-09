package town

import (
	"fmt"
	"strings"
)

// Validate checks that every name a description uses resolves: art, actors,
// slots, hotspots, rooms, draw sources, programs, formats, phases and modes.
// The first failure is the error, naming the town and the offending value.
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
	random := map[string]bool{}
	for _, r := range d.Random {
		switch r.Source {
		case "host":
		case "lcg":
			if r.Mask == 0 {
				return fail("generator %q has no mask", r.Name)
			}
		default:
			return fail("draw source %q has unknown kind %q", r.Name, r.Source)
		}
		random[r.Name] = true
	}
	art := map[string]bool{}
	for _, a := range d.Art {
		switch a.Format {
		case "picture", "series", "sprites", "mask":
		default:
			return fail("art %q has unknown format %q", a.Name, a.Format)
		}
		if a.Name == "" || a.Key == "" {
			return fail("art entry without name or key")
		}
		if len(a.Index) > 2 {
			return fail("art %q has %d indices", a.Name, len(a.Index))
		}
		art[a.Name] = true
	}
	slots := map[string]bool{}
	for _, s := range d.Sounds.Slots {
		slots[s.Name] = true
	}
	spots := map[string]bool{}
	for _, h := range d.Hotspots {
		spots[h.Name] = true
	}
	if d.Mask.Art != "" && !art[d.Mask.Art] {
		return fail("mask art %q", d.Mask.Art)
	}
	for _, m := range d.Mask.Bytes {
		if !spots[m.Hotspot] {
			return fail("mask byte %d names no hotspot %q", m.Byte, m.Hotspot)
		}
	}
	actors := map[string]string{}
	for _, a := range d.Actors {
		switch a.Program {
		case "episode", "pendulum", "stepper", "driven", "flock", "families":
		default:
			return fail("actor %q has unknown program %q", a.Name, a.Program)
		}
		actors[a.Name] = a.Program
		for _, m := range a.Members {
			actors[m.Name] = a.Program
		}
	}
	checkDraw := func(where, source string) error {
		if !random[source] {
			return fail("%s draws from unknown source %q", where, source)
		}
		return nil
	}
	checkSlot := func(where, slot string) error {
		if !slots[slot] {
			return fail("%s names unknown slot %q", where, slot)
		}
		return nil
	}
	for _, a := range d.Actors {
		where := "actor " + a.Name
		if err := d.validateActor(a, where, art, checkDraw, checkSlot); err != nil {
			return err
		}
	}
	rooms := map[string]bool{}
	for _, r := range d.Rooms {
		rooms[r.Name] = true
		for _, s := range append(append([]Step{}, r.Enter...), r.Exit...) {
			if err := validateStep(s, "room "+r.Name, fail); err != nil {
				return err
			}
		}
	}
	for _, s := range d.Square.Enter {
		if err := validateStep(s, "square", fail); err != nil {
			return err
		}
	}
	var checkOps func(where string, ops []Op) error
	checkOps = func(where string, ops []Op) error {
		for _, op := range ops {
			switch {
			case op.Arm != "":
				if _, ok := actors[op.Arm]; !ok {
					return fail("%s arms unknown actor %q", where, op.Arm)
				}
				if op.Chance != nil {
					if err := checkDraw(where, op.Chance.Draw); err != nil {
						return err
					}
				}
			case op.Latch != "":
				if err := checkSlot(where, op.Slot); err != nil {
					return err
				}
				for _, s := range op.Stop {
					if err := checkSlot(where, s); err != nil {
						return err
					}
				}
			case op.Drive != "":
				if actors[op.Drive] != "driven" {
					return fail("%s drives %q, which is no driven actor", where, op.Drive)
				}
			case len(op.ClearLatch) > 0, op.Hook != "", op.Menu != "":
			case op.Room != "":
				if !rooms[op.Room] {
					return fail("%s enters unknown room %q", where, op.Room)
				}
			case op.If != "":
				if err := checkOps(where, op.Then); err != nil {
					return err
				}
				if err := checkOps(where, op.Else); err != nil {
					return err
				}
			default:
				return fail("%s has an operation with no verb", where)
			}
		}
		return nil
	}
	for _, h := range d.Hotspots {
		if err := checkOps("hotspot "+h.Name, h.Click); err != nil {
			return err
		}
		if err := checkOps("hotspot "+h.Name, h.Hover); err != nil {
			return err
		}
	}
	if err := checkOps("pointer", d.Pointer.Every); err != nil {
		return err
	}
	if err := checkOps("pointer", d.Pointer.Off); err != nil {
		return err
	}
	phases := map[string][]string{
		"episode": {"trigger", "advance"}, "pendulum": {"advance"}, "stepper": {"advance"},
		"driven": {"advance"}, "flock": {"settle", "arm", "advance"}, "families": {"advance", "schedule"},
	}
	for _, list := range [][]string{d.Step.Before, d.Step.Admitted, d.Step.After} {
		for _, entry := range list {
			phase, name, _ := strings.Cut(entry, " ")
			program, ok := actors[name]
			if !ok {
				return fail("step %q names unknown actor", entry)
			}
			known := false
			for _, p := range phases[program] {
				known = known || p == phase
			}
			if !known {
				return fail("step %q: a %s has no phase %q", entry, program, phase)
			}
		}
	}
	for i, l := range d.Layers {
		if (l.Art == "") == (l.Actor == "") {
			return fail("layer %d names neither or both of art and actor", i)
		}
		if l.Art != "" && !art[l.Art] {
			return fail("layer %d names unknown art %q", i, l.Art)
		}
		if _, ok := actors[l.Actor]; l.Actor != "" && !ok {
			return fail("layer %d names unknown actor %q", i, l.Actor)
		}
		if l.Mode != "copy" && l.Mode != "over" {
			return fail("layer %d has unknown mode %q", i, l.Mode)
		}
		if l.When != nil {
			if l.When.Hover != "" && !spots[l.When.Hover] {
				return fail("layer %d waits on unknown hotspot %q", i, l.When.Hover)
			}
			if _, ok := actors[l.When.Active]; l.When.Active != "" && !ok {
				return fail("layer %d waits on unknown actor %q", i, l.When.Active)
			}
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

func (d *Description) validateActor(a ActorSpec, where string, art map[string]bool,
	checkDraw, checkSlot func(string, string) error) error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("town %s: %s: %s", d.Town, where, fmt.Sprintf(format, args...))
	}
	needArt := func(name string) error {
		if !art[name] {
			return fail("unknown art %q", name)
		}
		return nil
	}
	switch a.Program {
	case "episode":
		if err := needArt(a.Art); err != nil {
			return err
		}
		if a.End != "" && a.End != "rewind" && a.End != "hold" {
			return fail("unknown end %q", a.End)
		}
		if a.End == "hold" && (a.Frames <= 0 || a.RewindEvery <= 0) {
			return fail("a held episode needs frames and rewind-every")
		}
		if a.Trigger != nil {
			if err := checkDraw(where, a.Trigger.Draw); err != nil {
				return err
			}
		}
		if c := a.StartSound; c != nil {
			if err := checkSlot(where, c.Slot); err != nil {
				return err
			}
			for _, s := range c.Stop {
				if err := checkSlot(where, s); err != nil {
					return err
				}
			}
		}
	case "pendulum":
		if a.Chance == nil || len(a.Members) == 0 {
			return fail("a pendulum needs members and a chance")
		}
		for _, m := range a.Members {
			if err := needArt(m.Art); err != nil {
				return err
			}
		}
		return checkDraw(where, a.Chance.Draw)
	case "stepper", "driven":
		if err := needArt(a.Art); err != nil {
			return err
		}
		if a.Rest != "first" && a.Rest != "last" {
			return fail("unknown rest frame %q", a.Rest)
		}
		if a.HoldUnless != "" && a.Hold != "first" && a.Hold != "last" {
			return fail("unknown hold frame %q", a.Hold)
		}
		return checkSlot(where, a.Slot)
	case "flock":
		if err := needArt(a.Art); err != nil {
			return err
		}
		if a.Wait == nil || a.Group == nil || a.Count == nil || a.GroupSize <= 0 || a.Groups <= 0 || a.Frames <= 0 {
			return fail("a flock needs groups, group-size, frames, wait, group and count")
		}
		for _, s := range []string{a.Wait.Draw, a.Group.Draw, a.Count.Draw} {
			if err := checkDraw(where, s); err != nil {
				return err
			}
		}
		return checkSlot(where, a.Slot)
	case "families":
		names := map[string]bool{}
		for _, m := range a.Members {
			if err := needArt(m.Art); err != nil {
				return err
			}
			if m.Mode != "episode" && m.Mode != "loop" {
				return fail("member %q has unknown mode %q", m.Name, m.Mode)
			}
			if m.Mode == "episode" && (m.Later == nil || m.SheetPick == nil) {
				return fail("member %q needs later and sheet-pick", m.Name)
			}
			for _, c := range m.FrameCues {
				if err := checkSlot(where, c.Slot); err != nil {
					return err
				}
			}
			names[m.Name] = true
		}
		for _, s := range a.Entry {
			for _, n := range []string{s.Position, s.Wait, s.Unequal} {
				if n != "" && !names[n] {
					return fail("entry names unknown member %q", n)
				}
			}
			if s.Position != "" && s.Pick == nil || s.Wait != "" && s.After == nil {
				return fail("entry step without its draw")
			}
		}
		for _, n := range append(append([]string{}, a.StepOrder...), a.PaintOrder...) {
			if !names[n] {
				return fail("order names unknown member %q", n)
			}
		}
	}
	return nil
}
