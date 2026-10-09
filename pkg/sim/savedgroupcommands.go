package sim

import (
	"encoding/binary"
	"fmt"
)

// resolveSavedGroups answers the one saved Group a script selector names. A
// selector carried by Groups of different owners resolves to the later one in
// the player-then-group walk, the original's id-only map with the last write
// winning (AI-366); the earlier is unreachable by selector. Two Groups of one
// owner, or an owner no Player row names, stay ambiguous and refuse. Whether a
// SAV restore rebuilds the walk order is open, so the order here is the order
// the document stores its Players and Groups in.
func (w *World) resolveSavedGroups(selector uint32) ([]*SavedGroup, error) {
	var found []*SavedGroup
	var at []int
	for i := range w.savedGroups.Groups {
		g := &w.savedGroups.Groups[i]
		if g.Selector != selector {
			continue
		}
		found = append(found, g)
		at = append(at, i)
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("saved Group selector %d is absent", selector)
	}
	if len(found) > 1 {
		owners := map[uint32]bool{}
		for _, g := range found {
			owner, known := g.Owner.Owner, g.Owner.Class == 1
			if g.OwnerID != 0 {
				p, ok := w.savedPlayerByID(g.OwnerID)
				owner, known = p.Slot, ok
			}
			if !known || owners[owner] {
				return nil, fmt.Errorf("saved Group selector %d is ambiguous", selector)
			}
			owners[owner] = true
		}
		last := 0
		for k := 1; k < len(found); k++ {
			if w.savedWalkRank(found[k], at[k]) > w.savedWalkRank(found[last], at[last]) {
				last = k
			}
		}
		found = found[last : last+1]
	}
	if _, ok := w.savedLiving(found[0]); !ok {
		return nil, fmt.Errorf("saved Group selector %d has unmaterialized members", selector)
	}
	return found, nil
}

// savedWalkRank orders saved Groups as the original's list walk visits them:
// by the position of the containing Player when the document names Players,
// then by the Group's position in the document.
func (w *World) savedWalkRank(g *SavedGroup, index int) int {
	if w.savedGroups.PlayersPresent {
		for k, p := range w.savedGroups.Players {
			if p.ID == g.ContainerID {
				return (k+1)*(len(w.savedGroups.Groups)+1) + index
			}
		}
	}
	return index
}

// A refused current count cannot let a stale register fire a trigger. The
// refusal does not modify either its register or any dependent trigger latch.
func (w *World) savedTriggerBlocked(t ScriptTrigger) bool {
	if w.savedGroups == nil || w.script == nil {
		return false
	}
	for _, c := range w.script.checks {
		if c.Op != ScriptCheckGroupCount || !c.HasGroup {
			continue
		}
		if _, err := w.resolveSavedGroups(c.Group); err == nil {
			continue
		}
		for _, p := range t.Pairs {
			if p.Used && (p.Left == c.Register || p.Right == c.Register) {
				return true
			}
		}
	}
	for _, index := range t.Instants {
		if index < 0 || int(index) >= len(w.script.instants) {
			continue
		}
		in := w.script.instants[index]
		if !in.HasGroup {
			continue
		}
		if _, err := w.resolveSavedGroups(in.Group); err != nil {
			return true
		}
		if in.Op == ScriptInstantGroupOrder && !savedGroupCommandSupported(in.Args[0]) {
			return true
		}
	}
	return false
}

func savedGroupCommandSupported(order int32) bool {
	return groupOrderSupported(order)
}

func (w *World) ensureSavedOrder(i int) *SavedActorOrder {
	if o := w.savedOrder(w.entities[i].ID); o != nil {
		return o
	}
	o := SavedActorOrder{Entity: w.entities[i].ID, Authored: true}
	n := w.entities[i].attackNotice
	binary.LittleEndian.PutUint16(o.Raw[0x58:], n.Cell)
	o.Raw[0x5a] = n.Scans
	w.entities[i].attackNotice = attackNotice{}
	w.savedGroups.Orders = append(w.savedGroups.Orders, o)
	return &w.savedGroups.Orders[len(w.savedGroups.Orders)-1]
}

// A tactical setter changes the actor after constructing its order-none
// Group. Keep that current native order in the SAV dispatcher too; retained
// source keys never become native escort identities. Action progress remains
// owned by the entity and is not cancelled by this synchronization.
func (w *World) syncSavedActorCommand(i int) *SavedActorOrder {
	if w.savedGroups == nil {
		return nil
	}
	e := w.entities[i]
	o := w.ensureSavedOrder(i)
	o.State, o.Patrol = uint32(e.ActorState), nil
	o.authorNative()
	o.Raw[0x70] = e.EscortRange
	return o
}

// New player-command Groups follow this build's explicit native authoring
// policy. They do not pretend to reconstruct unnamed original ctor fields.
func (w *World) commandSavedGroup(members []int, order uint8, ordered cell) {
	for _, i := range members {
		w.noteActorMotionOrder(w.entities[i].ID)
	}
	w.newSavedCommandGroup(members, order, ordered, true, w.savedCommandContainer(members), true)
}

func (w *World) newSavedCommandGroup(members []int, order uint8, ordered cell, resetOrder bool, container uint32, cleanup bool) bool {
	if len(members) == 0 || w.savedGroups == nil {
		return false
	}
	high := max(w.savedGroups.HighWater, maxSavedGroupID(w.savedGroups.Groups))
	if high == ^uint32(0) {
		return false
	}
	id := high + 1
	selector := w.freeSavedSelector()
	if selector == 0 {
		return false
	}
	if cleanup {
		w.removeFirstEmptySavedGroup(container)
	}
	w.savedGroups.HighWater = id
	g := SavedGroup{ID: id, ContainerID: container, Selector: selector, Authored: true}
	if w.hasSavedFormations() {
		g.OwnerID = container
	}
	g.AI[0x20], g.AI[0x45] = order, 1
	binary.LittleEndian.PutUint16(g.AI[10:], uint16(ordered.x)|uint16(ordered.y)<<8)
	cx, cy := groupCentroid(w.entities, members)
	g.AI[0x38] = noticeBase(w.entities, members, cx, cy)
	for _, i := range members {
		e := &w.entities[i]
		w.detachSavedMember(e.ID)
		g.Members = append(g.Members, SavedGroupMember{Archive: e.SourceBinding.ArchiveIndex, Entity: e.ID, Bound: true})
		g.Owner = SavedGroupReference{Class: 1, Owner: e.Owner}
		currentSelector := e.CommandGroup != 0 && e.SourceBinding.GroupSelector == e.CommandGroup
		e.CommandGroup = selector
		if e.SourceBinding.GroupIndex != 0 && currentSelector {
			e.SourceBinding.GroupSelector = selector
		}
		if !resetOrder {
			continue
		}
		e.clearPatrol()
		w.clearEscort(i)
		o := w.ensureSavedOrder(i)
		o.State, o.Patrol = 0xb, nil
		o.authorNative()
		binary.LittleEndian.PutUint32(o.Raw[0x50:], 0)
		binary.LittleEndian.PutUint16(o.Raw[:], uint16(e.X)|uint16(e.Y)<<8)
		binary.LittleEndian.PutUint16(o.Raw[10:], uint16(ordered.x)|uint16(ordered.y)<<8)
	}
	w.appendSavedCommandGroup(g)
	return true
}

func (w *World) handOverSaved(i int, player uint32) {
	// A handover is a membership/ownership mutation, not a fresh movement or
	// patrol setter. The current actor order and raw operands survive.
	// PARTY-JOIN-025 constructs directly in the destination Player; it does
	// not call the ordinary command wrapper's first-rejected cleanup.
	if !w.newSavedCommandGroup([]int{i}, orderNone, cell{}, false, w.uniqueSavedPlayerSlot(player), false) {
		return
	}
	w.entities[i].Owner = player
	w.inheritAutoHealing(i)
	w.savedGroupFor(w.entities[i].ID).Owner = SavedGroupReference{Class: 1, Owner: player}
}

func (w *World) freeSavedSelector() uint32 {
	for id := w.commandFloor(); id != 0; id++ {
		taken := false
		for _, g := range w.savedGroups.Groups {
			if g.Selector == id {
				taken = true
				break
			}
		}
		if !taken {
			return id
		}
	}
	return 0
}

// savedMoveGroup reports whether id stands in a saved group whose order is
// Move. AI-CMD-054's attack order writes the group order to zero; left at
// Move, the group's arrival arm would end the attack on arrival.
func (w *World) savedMoveGroup(id EntityID) bool {
	g := w.savedGroupFor(id)
	return g != nil && g.AI[0x20] == orderMove
}

func (w *World) syncSavedDestination(i int) {
	if w.savedGroups == nil {
		return
	}
	e := w.entities[i]
	o := w.ensureSavedOrder(i)
	if e.HasTarget {
		binary.LittleEndian.PutUint16(o.Raw[10:], uint16(e.TargetX)|uint16(e.TargetY)<<8)
		o.Raw[8] = 1
	}
	binary.LittleEndian.PutUint32(o.Raw[0x50:], 0)
	if g := w.savedGroupFor(e.ID); g != nil && g.Authored && g.AI[0x44] == 0 && e.GroupSpeed != 0 {
		g.AI[0x44] = e.GroupSpeed
	}
}

func (w *World) syncSavedPost(i int) {
	if w.savedGroups == nil {
		return
	}
	e := w.entities[i]
	o := w.ensureSavedOrder(i)
	binary.LittleEndian.PutUint16(o.Raw[:], uint16(e.PostX)|uint16(e.PostY)<<8)
}

func (w *World) setSavedPatrol(members []int, point cell) {
	x, y := w.bounds.clamp(point.x, point.y)
	dst := uint16(x) | uint16(y)<<8
	for _, i := range members {
		w.clearOrder(i)
		e := &w.entities[i]
		w.retainCycleForState(i)
		e.clearPatrol()
		w.clearEscort(i)
		e.clearGroupSpeed()
		e.ActorState = actorStatePatrol
		e.PatrolHeadX, e.PatrolHeadY, e.PatrolTailX, e.PatrolTailY = e.X, e.Y, x, y
		e.PatrolLeg = patrolLegTail
		e.PostX, e.PostY = e.X, e.Y
		o := w.ensureSavedOrder(i)
		o.State = 0xa
		o.authorNative()
		o.Patrol = []uint16{uint16(e.X) | uint16(e.Y)<<8, dst}
		binary.LittleEndian.PutUint16(o.Raw[:], o.Patrol[0])
		binary.LittleEndian.PutUint16(o.Raw[2:], dst)
		binary.LittleEndian.PutUint32(o.Raw[4:], 0)
		o.Raw[8] = 0
	}
}

func (w *World) cmdSavedGroupOrder(in ScriptInstant) {
	if !in.HasGroup {
		return
	}
	groups, err := w.resolveSavedGroups(in.Group)
	if err != nil {
		return
	}
	order := in.Args[0]
	if order == subCommandDefend || order == subCommandFollow || order == subCommandAttack {
		if !in.HasUnit || indexOfEntity(w.entities, in.Unit) < 0 {
			return
		}
		if order == subCommandAttack && !w.entities[indexOfEntity(w.entities, in.Unit)].OrdinaryTargetable() {
			return
		}
		if order == subCommandDefend {
			w.cmdGroupEscort(in, actorStateDefend)
		} else if order == subCommandFollow {
			w.cmdGroupEscort(in, actorStateFollow)
		} else {
			w.cmdGroupAttack(in)
		}
		for _, g := range groups {
			members, _ := w.savedLiving(g)
			for _, i := range members {
				e := w.entities[i]
				o := w.ensureSavedOrder(i)
				o.authorNative()
				o.State = uint32(e.ActorState)
				o.Patrol = nil
				o.Raw[8] = 0
				o.Raw[0x70] = e.EscortRange
				if order == subCommandAttack && e.HasAttackTarget {
					o.Raw[0x14] = uint8(e.Reach)
				}
				binary.LittleEndian.PutUint16(o.Raw[:], uint16(e.PostX)|uint16(e.PostY)<<8)
			}
		}
		return
	}
	for _, g := range groups {
		members, _ := w.savedLiving(g)
		point := cell{x: in.Args[1], y: in.Args[2]}
		switch order {
		case 1, 3:
			g.AI[0x20] = uint8(order)
			if order == 1 {
				var cx, cy int32
				if len(members) > 0 {
					cx, cy = groupCentroid(w.entities, members)
				}
				g.AI[0x38] = noticeBase(w.entities, members, cx, cy)
			}
			for _, i := range members {
				w.entities[i].PostX, w.entities[i].PostY = w.entities[i].X, w.entities[i].Y
				w.syncSavedPost(i)
			}
			if order == 3 {
				w.standScriptedMembers(members)
			}
		case 2, 4, 5:
			g.AI[0x20] = uint8(order)
			x, y := w.bounds.clamp(point.x, point.y)
			binary.LittleEndian.PutUint16(g.AI[10:], uint16(x)|uint16(y)<<8)
			if order == 4 || order == 5 {
				w.issueSavedGroupDestination(g, members, cell{x, y})
			}
		case subCommandPatrol:
			g.AI[0x20] = 0
			w.setSavedPatrol(members, point)
		}
	}
}

func (w *World) stopSavedGroupMembers(group uint32) []int {
	groups, err := w.resolveSavedGroups(group)
	if err != nil {
		return nil
	}
	var stopped []int
	for _, g := range groups {
		members, _ := w.savedLiving(g)
		g.AI[0x20] = 0
		for _, i := range members {
			w.clearOrder(i)
			w.retainCycleForState(i)
			w.entities[i].clearGroupSpeed()
			w.entities[i].clearPatrol()
			w.clearEscort(i)
			stopped = append(stopped, i)
		}
	}
	return stopped
}
