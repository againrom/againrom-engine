package sim

import "encoding/binary"

const orderRoam uint8 = 17

// AI-GROUPCMD-020 and AI-PATROL-018: command17 runs the stop helper,
// then writes the Group order and first member's cell. It neither builds
// Patrol's ring nor copies the new Group cell to a member destination.
func (w *World) cmdGroupRoam(group uint32) {
	if w.savedGroups != nil {
		groups, err := w.resolveSavedGroups(group)
		if err != nil {
			return
		}
		for _, g := range groups {
			if len(g.Members) == 0 {
				continue
			}
			first := indexOfEntity(w.entities, g.Members[0].Entity)
			members, complete := w.savedLiving(g)
			if first < 0 || !complete {
				continue
			}
			g.AI[0x20] = 0
			for _, i := range members {
				w.stopRoamMember(i)
				o := w.ensureSavedOrder(i)
				o.Raw[0x14] = uint8(w.entities[i].Reach)
				for _, at := range []int{0x38, 0x50, 0x60} {
					clear(o.Raw[at : at+4])
				}
				if m := w.motionFor(w.entities[i].ID); m != nil {
					clear(m.Mover[0x7c:0x80])
					m.ActorAction = 0
				}
			}
			g.AI[0x20] = orderRoam
			e := w.entities[first]
			binary.LittleEndian.PutUint16(g.AI[10:], uint16(e.X)|uint16(e.Y)<<8)
		}
		return
	}
	for _, gi := range w.groupsNamed(group) {
		g := &w.groups[gi]
		members := w.groupLivingMembers(g.owner, g.group)
		if len(members) == 0 {
			continue
		}
		g.order = orderNone
		for _, i := range members {
			w.stopRoamMember(i)
		}
		g.order = orderRoam
		g.commandedX, g.commandedY = w.entities[members[0]].X, w.entities[members[0]].Y
	}
}

func (w *World) stopRoamMember(i int) {
	w.clearOrder(i)
	w.retainCycleForState(i)
	w.entities[i].clearGroupSpeed()
}

// AI-ROAM-025: update a Group cell/counter, then evaluate Swarm2. Each roll
// is the AI range idiom over the eight directions. DIV-1022 also bounds reroll
// work: after 32 rejected draws choose the first valid direction; if none
// exists, withhold this primary evaluation without mutating the state.
func (w *World) rollRoamCell(dst cell, members []int, counter uint8) (cell, uint8, bool) {
	var far int64
	for _, i := range members {
		far = max(far, cellOf(w.entities[i]).chebyshevTo(dst))
	}
	if far >= 10 && counter <= 50 {
		return dst, counter, true
	}
	valid := func(c cell) bool {
		return c.x >= 8 && c.y >= 8 && c.x <= w.bounds.Width-9 && c.y <= w.bounds.Height-9
	}
	var fallback cell
	found := false
	for _, d := range savedRoamDirections {
		c := cell{dst.x + 20*d.x, dst.y + 20*d.y}
		if valid(c) {
			fallback, found = c, true
			break
		}
	}
	if !found {
		return dst, counter, false
	}
	for range 32 {
		d := savedRoamDirections[w.rng.aiRange(8)]
		c := cell{dst.x + 20*d.x, dst.y + 20*d.y}
		if valid(c) {
			return c, 0, true
		}
	}
	return fallback, 0, true
}

func (w *World) nativeRoam(g aiGroup) (int, bool) {
	for i := range w.groups {
		stored := &w.groups[i]
		if stored.owner != g.owner || stored.group != g.group {
			continue
		}
		cell, counter, ok := w.rollRoamCell(cell{stored.commandedX, stored.commandedY}, g.members, stored.roamCounter)
		if ok {
			stored.commandedX, stored.commandedY, stored.roamCounter = cell.x, cell.y, counter
		}
		return i, ok
	}
	return 0, false
}
