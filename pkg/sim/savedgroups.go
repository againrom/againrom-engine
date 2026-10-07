package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
)

// SavedGroupReference is a source lookup result, not a native identity. Class
// is 0 for unresolved, 1 for Player and 2 for another archive object.
type SavedGroupReference struct {
	Key     uint32
	Archive uint16
	Class   uint8
	Owner   uint32
}

// SavedGroupMember keeps unresolved archive membership visible. Entity is only
// meaningful when Bound; zero is itself a valid native entity ID.
type SavedGroupMember struct {
	Archive uint16
	Entity  EntityID
	Bound   bool
}

type SavedGroup struct {
	ID               uint32
	ContainerID      uint32 // exact native Player container; zero means unknown
	OwnerID          uint32 // exact formation owner, independent of containment
	Selector         uint32
	Authored         bool // native command construction, not a restored original AI block
	Reference, Owner SavedGroupReference
	// AI excludes the replaced +4c list pointer. Its remaining raw numeric
	// fields remain intact; only named supported consumers interpret them.
	AI          [76]byte
	Words, Path []uint16
	Members     []SavedGroupMember
}

type SavedActorOrder struct {
	Entity   EntityID
	State    uint32
	Authored bool // typed native command target; never an imported raw pointer
	// Stage gates original order-key repair. EscortTarget is a separately
	// validated source-key binding, never a cast of Raw+10 to EntityID.
	RepairStage  uint8
	EscortTarget EntityID
	EscortBound  bool
	// Raw excludes the replaced +90 list pointer.
	Raw    [144]byte
	Patrol []uint16
}

type savedGroupState struct {
	Groups            []SavedGroup
	Orders            []SavedActorOrder
	Players           []SavedGroupPlayer
	PlayersPresent    bool
	Formations        []SavedPlayerFormation
	FormationsPresent bool
	HighWater         uint32
}

func (o *SavedActorOrder) authorNative() {
	o.Authored = true
	o.RepairStage, o.EscortTarget, o.EscortBound = 0, 0, false
}

func cloneSavedGroups(s *savedGroupState) *savedGroupState {
	if s == nil {
		return nil
	}
	c := *s
	c.Groups, c.Orders, c.Players = slices.Clone(s.Groups), slices.Clone(s.Orders), slices.Clone(s.Players)
	c.Formations = slices.Clone(s.Formations)
	for i := range c.Groups {
		g := &c.Groups[i]
		g.Words, g.Path, g.Members = cloneNonEmpty(g.Words), cloneNonEmpty(g.Path), cloneNonEmpty(g.Members)
	}
	for i := range c.Orders {
		c.Orders[i].Patrol = cloneNonEmpty(c.Orders[i].Patrol)
	}
	return &c
}

// cloneNonEmpty copies s and gives every zero-length sequence the nil shape
// that the binary decoder produces, so an empty sequence has one
// representation however the registry was built.
func cloneNonEmpty[S ~[]E, E any](s S) S {
	if len(s) == 0 {
		return nil
	}
	return slices.Clone(s)
}

// SavedGroups returns detached current state. False is the explicit legacy
// native mode, including old AGS; an empty imported registry is still present.
func (w *World) SavedGroups() ([]SavedGroup, []SavedActorOrder, bool) {
	s := cloneSavedGroups(w.savedGroups)
	if s == nil {
		return nil, nil, false
	}
	return s.Groups, s.Orders, true
}

func (w *World) ImportSavedGroups(groups []SavedGroup, orders []SavedActorOrder) error {
	s := cloneSavedGroups(&savedGroupState{Groups: groups, Orders: orders})
	if prior := w.savedGroups; prior != nil {
		s.Players, s.PlayersPresent, s.HighWater = slices.Clone(prior.Players), prior.PlayersPresent, prior.HighWater
		s.Formations, s.FormationsPresent = slices.Clone(prior.Formations), prior.FormationsPresent
		old := make(map[uint32]bool, len(prior.Groups))
		for _, g := range prior.Groups {
			old[g.ID] = true
		}
		for _, g := range s.Groups {
			if !old[g.ID] && g.ID <= prior.HighWater {
				return fmt.Errorf("sim: saved Group update reuses identity")
			}
		}
	}
	s.HighWater = max(s.HighWater, maxSavedGroupID(s.Groups))
	if err := savedGroupsFault(s, w.entities, w.originalDead); err != nil {
		return err
	}
	w.savedGroups = s
	for _, o := range s.Orders {
		w.entities[indexOfEntity(w.entities, o.Entity)].attackNotice = attackNotice{}
	}
	return nil
}

func savedGroupsFault(s *savedGroupState, entities []Entity, dead []originalDeadRecord) error {
	if s == nil {
		return nil
	}
	if err := savedGroupPlayersFault(s); err != nil {
		return err
	}
	if err := savedFormationsFault(s); err != nil {
		return err
	}
	ids, members, archives, orders := map[uint32]bool{}, map[EntityID]bool{}, map[uint16]bool{}, map[EntityID]bool{}
	for _, g := range s.Groups {
		if g.ID == 0 || ids[g.ID] {
			return fmt.Errorf("sim: duplicate or zero saved Group identity %d", g.ID)
		}
		ids[g.ID] = true
		for _, ref := range []SavedGroupReference{g.Reference, g.Owner} {
			if ref.Class > 2 || ref.Class == 0 && (ref.Archive != 0 || ref.Owner != 0) || ref.Class == 2 && ref.Owner != 0 {
				return fmt.Errorf("sim: invalid saved Group reference")
			}
		}
		for _, m := range g.Members {
			if m.Archive != 0 && archives[m.Archive] {
				return fmt.Errorf("sim: repeated saved Group archive member %d", m.Archive)
			}
			if m.Archive != 0 {
				archives[m.Archive] = true
			}
			if !m.Bound {
				if m.Entity != 0 {
					return fmt.Errorf("sim: unbound Group member has native identity")
				}
				continue
			}
			if members[m.Entity] || indexOfEntity(entities, m.Entity) < 0 {
				return fmt.Errorf("sim: invalid saved Group member %d", m.Entity)
			}
			members[m.Entity] = true
		}
	}
	for _, o := range s.Orders {
		if orders[o.Entity] || indexOfEntity(entities, o.Entity) < 0 {
			return fmt.Errorf("sim: invalid saved actor order %d", o.Entity)
		}
		orders[o.Entity] = true
		if o.EscortBound {
			if o.Authored || o.RepairStage != 0 || o.State != 8 && o.State != 0x11 || !savedEscortBindingValid(o, entities, dead) {
				return fmt.Errorf("sim: invalid saved escort binding for actor %d", o.Entity)
			}
		} else if o.EscortTarget != 0 {
			return fmt.Errorf("sim: unbound saved escort has native identity")
		}
	}
	return nil
}

func (w *World) savedGroupFor(id EntityID) *SavedGroup {
	return savedGroupForIn(w.savedGroups, id)
}

// savedGroupForIn is savedGroupFor's own membership scan, pulled out as a free
// function so a reader that has decoded state but no *World yet — the decode
// path's own pickup-completion check (pickupcompletion.go) among them — can
// run the identical lookup rather than a second copy of it that could drift.
func savedGroupForIn(s *savedGroupState, id EntityID) *SavedGroup {
	if s != nil {
		for i := range s.Groups {
			g := &s.Groups[i]
			for _, m := range g.Members {
				if m.Bound && m.Entity == id {
					return g
				}
			}
		}
	}
	return nil
}

func (w *World) savedGroupByID(id uint32) *SavedGroup {
	for i := range w.savedGroups.Groups {
		if w.savedGroups.Groups[i].ID == id {
			return &w.savedGroups.Groups[i]
		}
	}
	return nil
}

func (w *World) groupRateEntity(e Entity) Entity {
	if w.savedGroups != nil {
		e.GroupSpeed = 0
		if g := w.savedGroupFor(e.ID); g != nil {
			e.GroupSpeed = g.AI[0x44]
		}
	}
	return e
}

func (w *World) savedOrder(id EntityID) *SavedActorOrder {
	if w.savedGroups != nil {
		for i := range w.savedGroups.Orders {
			if w.savedGroups.Orders[i].Entity == id {
				return &w.savedGroups.Orders[i]
			}
		}
	}
	return nil
}

// SavedGroupIssues names operations that cannot currently be executed. Raw
// values and membership remain saveable; a new supported player order can
// replace an unsupported continuation without discarding the source Group.
func (w *World) SavedGroupIssues() []string {
	if w.savedGroups == nil {
		return nil
	}
	var out []string
	selectors := map[uint32]int{}
	for _, g := range w.savedGroups.Groups {
		selectors[g.Selector]++
	}
	for _, g := range w.savedGroups.Groups {
		if selectors[g.Selector] > 1 {
			if _, err := w.resolveSavedGroups(g.Selector); err != nil {
				out = append(out, fmt.Sprintf("Group %d: %v", g.ID, err))
			}
		}
		if w.hasSavedFormations() && g.OwnerID == 0 && (g.AI[0x20] == orderMove || g.AI[0x20] == orderSwarm2) {
			out = append(out, fmt.Sprintf("Group %d: formation movement lacks an exact Player owner", g.ID))
		}
		for _, m := range g.Members {
			if !m.Bound {
				out = append(out, fmt.Sprintf("Group %d: member archive %d is unmaterialized", g.ID, m.Archive))
			}
		}
		if issue := w.savedPrimaryIssue(&g); issue != "" {
			out = append(out, fmt.Sprintf("Group %d: %s", g.ID, issue))
		}
	}
	for _, o := range w.savedGroups.Orders {
		if !savedActorOrderSupported(o) {
			out = append(out, fmt.Sprintf("actor %d: saved state %d continuation unsupported", o.Entity, o.State))
		}
		if o.State == 0xa && !slices.Contains(o.Patrol, binary.LittleEndian.Uint16(o.Raw[2:])) {
			out = append(out, fmt.Sprintf("actor %d: patrol cursor is absent from ring", o.Entity))
		}
	}
	if w.script != nil {
		for _, c := range w.script.checks {
			if c.Op == ScriptCheckGroupCount && c.HasGroup {
				if _, err := w.resolveSavedGroups(c.Group); err != nil {
					out = append(out, "count: "+err.Error())
				}
			}
		}
		for _, in := range w.script.instants {
			if in.Op == ScriptInstantFormation && in.HasPlayer && w.hasSavedFormations() && w.triggerFormation(in.Player) == nil {
				out = append(out, fmt.Sprintf("formation trigger: Player+08 %d is absent or ambiguous", in.Player))
			}
			if !in.HasGroup {
				continue
			}
			if _, err := w.resolveSavedGroups(in.Group); err != nil {
				out = append(out, "command: "+err.Error())
			}
			if in.Op == ScriptInstantGroupOrder && !savedGroupCommandSupported(in.Args[0]) {
				out = append(out, fmt.Sprintf("Group command %d: saved continuation unsupported", in.Args[0]))
			}
		}
	}
	var unique []string
	for _, issue := range out {
		if !slices.Contains(unique, issue) {
			unique = append(unique, issue)
		}
	}
	return unique
}

func (w *World) detachSavedMember(id EntityID) {
	w.detachTerminalRegistry(id, false)
	if w.savedGroups == nil {
		return
	}
	for i := range w.savedGroups.Groups {
		g := &w.savedGroups.Groups[i]
		g.Members = slices.DeleteFunc(g.Members, func(m SavedGroupMember) bool { return m.Bound && m.Entity == id })
	}
}

// walkSavedMembers re-finds the dispatched identity in CURRENT membership.
// Removal ends this stage; an appended successor participates. No snapshot
// sort substitutes for SAV-GRPMUTATE-569's local successor rule.
func (w *World) walkSavedMembers(g *SavedGroup, dispatch func(int)) {
	if len(g.Members) == 0 {
		return
	}
	id := g.ID
	m := g.Members[0]
	for {
		if !m.Bound {
			return
		}
		i := indexOfEntity(w.entities, m.Entity)
		if i < 0 {
			return
		}
		if !w.motionActive(w.entities[i].ID) {
			dispatch(i)
		}
		g = w.savedGroupByID(id)
		if g == nil {
			return
		}
		at := slices.IndexFunc(g.Members, func(v SavedGroupMember) bool { return v.Bound && v.Entity == m.Entity })
		if at < 0 || at+1 == len(g.Members) {
			return
		}
		m = g.Members[at+1]
	}
}

func (w *World) savedLiving(g *SavedGroup) ([]int, bool) {
	var out []int
	for _, m := range g.Members {
		if !m.Bound {
			return nil, false
		}
		i := indexOfEntity(w.entities, m.Entity)
		if i < 0 {
			return nil, false
		}
		if w.entities[i].Alive() && !w.entities[i].OffMap && !w.stoneCursed(i) {
			out = append(out, i)
		}
	}
	return out, true
}

func (w *World) savedMove(i int, o *SavedActorOrder) {
	x, y := int32(o.Raw[10]), int32(o.Raw[11])
	if x >= w.bounds.Width || y >= w.bounds.Height {
		return
	}
	e := &w.entities[i]
	if e.X == x && e.Y == y {
		w.clearOrder(i)
		o.Raw[8] = 0xb
		w.acquireStanding(i)
		return
	}
	w.cancelTurnForTargetChange(i, x, y)
	if !e.HasTarget || e.TargetX != x || e.TargetY != y {
		w.clearOrder(i)
	}
	e.TargetX, e.TargetY, e.HasTarget = x, y, true
	o.Raw[8] = 1
}

func (w *World) savedPatrol(i int, o *SavedActorOrder) {
	if !slices.Contains(o.Patrol, binary.LittleEndian.Uint16(o.Raw[2:])) {
		return
	}
	e := &w.entities[i]
	if binary.LittleEndian.Uint32(o.Raw[4:]) != 0 {
		binary.LittleEndian.PutUint16(o.Raw[:], uint16(e.X)|uint16(e.Y)<<8)
		binary.LittleEndian.PutUint32(o.Raw[4:], 0)
	}
	e.PostX, e.PostY = int32(o.Raw[0]), int32(o.Raw[1])
	w.savedGuard(i, o)
	if o.Raw[8] != 0 && o.Raw[8] != 0xb {
		return
	}
	binary.LittleEndian.PutUint32(o.Raw[4:], 1)
	cursor := binary.LittleEndian.Uint16(o.Raw[2:])
	if uint16(e.X)|uint16(e.Y)<<8 == cursor {
		at := slices.Index(o.Patrol, cursor)
		cursor = o.Patrol[(at+1)%len(o.Patrol)]
		binary.LittleEndian.PutUint16(o.Raw[2:], cursor)
	}
	binary.LittleEndian.PutUint16(o.Raw[10:], cursor)
	w.savedMove(i, o)
}

func (w *World) savedActorDispatch(i int) {
	if _, using := w.structureUseIndex(w.entities[i].ID); using {
		return
	}
	o := w.savedOrder(w.entities[i].ID)
	if o == nil || !w.entities[i].Alive() || w.entities[i].OffMap {
		return
	}
	switch o.State {
	case 1:
		w.savedMove(i, o)
	case 3:
		w.savedEngage(i, o)
	case 0xa:
		w.savedPatrol(i, o)
	case 0xb:
		w.savedGuard(i, o)
	case 0xc:
		w.savedAcquire(i, o)
	case 0x16:
		// Only this build's player setter supplies Retreat's native order.
		// The same raw source number remains an unsupported continuation.
		if o.Authored {
			w.armRetreat(i)
			if w.entities[i].HasTarget {
				w.syncSavedDestination(i)
			}
		}
	case 8, 0x11:
		if !o.Authored {
			if o.RepairStage != 0 || !o.EscortBound || !savedEscortBindingValid(*o, w.entities, w.originalDead) {
				return
			}
			e := &w.entities[i]
			e.EscortTarget, e.HasEscortTarget, e.EscortRange = o.EscortTarget, true, o.Raw[0x70]
			e.ActorState = uint8(o.State)
		}
		ti := indexOfEntity(w.entities, w.entities[i].EscortTarget)
		if ti < 0 || ti == i {
			return
		}
		// The branch compares the stored byte, including zero. Its stop
		// operand alone falls back to ScanRange (AI-DEFEND-111/FOLLOW-112).
		closing := cellOf(w.entities[i]).chebyshevTo(cellOf(w.entities[ti])) > int64(o.Raw[0x70])
		if closing {
			w.escortClose(i, ti)
		} else if o.State == 8 {
			w.armDefend(i)
		} else {
			w.armFollow(i)
		}
		if w.entities[i].HasTarget {
			w.syncSavedDestination(i)
		}
		if closing {
			o.Raw[8], o.Raw[0x14] = 4, o.Raw[0x70]
			if g := w.savedGroupFor(o.Entity); o.Authored && g != nil && !g.Authored {
				target := w.currentActionKey(w.entities[i].EscortTarget, AttackTargetUnit)
				binary.LittleEndian.PutUint32(o.Raw[0x10:], target)
				binary.LittleEndian.PutUint32(o.Raw[0x18:], target)
			}
			if o.Raw[0x14] == 0 {
				o.Raw[0x14] = w.entities[i].ScanRange
			}
		}
	}
}

func (w *World) savedGroupPass(obs *castObs) {
	if w.savedGroups == nil {
		return
	}
	// This build visits the registry in retained LOAD/append order. Original
	// first-loaded global chronology is not established by the local claim.
	count := len(w.savedGroups.Groups)
	for gi := 0; gi < count; gi++ {
		g := &w.savedGroups.Groups[gi]
		id := g.ID
		if g.AI[0x45] == 0 {
			continue
		}
		if len(g.Members) == 0 {
			g.AI[0x20] = 0xff
		}
		order := g.AI[0x20]
		members, complete := w.savedLiving(g)
		if !complete {
			continue
		}
		if w.savedPrimaryIssue(g) != "" {
			// Refuse the whole primary operation before changing any member.
			// The independently supported common tail still runs below.
		} else if order == 0 || order == 0xff {
			w.walkSavedMembers(g, func(i int) {
				if order == 0 {
					if !w.nativePatrol(i) {
						w.savedActorDispatch(i)
					}
				} else if w.entities[i].Alive() && !w.entities[i].OffMap {
					w.savedAcquire(i, w.savedOrder(w.entities[i].ID))
				}
			})
		} else if len(members) > 0 && (validGroupOrder(order) || order == 0x11) {
			if order == 0x11 {
				w.savedRoam(g, members)
				order = orderSwarm2
			}
			w.savedDecision(g, order, obs)
		}
		// Every byte reaches a fresh current-head withdrawal stage.
		g = w.savedGroupByID(id)
		w.walkSavedMembers(g, func(i int) {
			if w.nativePatrol(i) {
				return
			}
			before := w.entities[i]
			w.withdrawalActor(i, nil)
			if after := w.entities[i]; after.HasTarget && (after.TargetX != before.TargetX || after.TargetY != before.TargetY || !before.HasTarget) {
				w.syncSavedDestination(i)
			}
		})
	}
}
