package sim

import (
	"cmp"
	"encoding/binary"
	"reflect"
	"slices"
)

// constructCarriers gives a World every carrier a World loaded from an
// original SAV holds, so one rule runs after a fresh start and after a LOAD.
// A carrier the World already holds is kept. It runs in newWorld, the
// controlled script copy and the byte decoder (after the whole form). Every
// World takes the Group registry; a second-game World keeps its other native
// paths (gameProfile). native
// names a World decoded from an older engine's state, whose Group registry
// carries its native Group records (constructedGroups).
func (w *World) constructCarriers(native bool) {
	w.constructSessionClock()
	w.constructEffectDriver()
	w.constructGroups(native)
	w.constructMovers()
}

// constructEffectDriver gives a first-game World the world-effect driver table
// the original-SAV importer arms (ImportOriginalWorldEffectDrivers): at
// construction it holds no area and no projectile row. A second-game World
// keeps the table its first projectile creates.
func (w *World) constructEffectDriver() {
	if w.game.nativeEffectDriver || w.savedWorldEffects != nil {
		return
	}
	w.savedWorldEffects = &SavedWorldEffects{}
}

// constructGroups gives a World the Group registry the original's
// mission start builds and the original-SAV importer fills (ImportSavedGroups).
// A World that holds a registry keeps it.
func (w *World) constructGroups(native bool) {
	if w.savedGroups != nil { // Keep a registry already built or imported.
		return
	}
	w.savedGroups = w.constructedGroups(native)
}

// groupsAsConstructed reports whether the World's Group registry, with its
// Player containers and formations, is exactly the one constructGroups builds
// from the current state. The byte form omits such a registry, and the decoder
// constructs it again.
func (w *World) groupsAsConstructed() bool {
	s := w.savedGroups
	if s == nil || !s.fresh || s.native { // Decode can omit the registry until the complete form is read.
		return false
	}
	return reflect.DeepEqual(cloneSavedGroups(s), cloneSavedGroups(w.constructedGroups(false)))
}

// GroupsConstructed reports whether the World's Group registry is the one
// constructGroups built, as play has changed it since, and not one an import
// installed.
func (w *World) GroupsConstructed() bool {
	return w.savedGroups.fresh
}

// GroupsNative reports whether the constructed registry carries an older
// engine's native Group records (constructedGroups).
func (w *World) GroupsNative() bool {
	return w.GroupsConstructed() && w.savedGroups.native
}

// AdoptConstructedGroups marks a restored registry as the constructed one, so
// a World that a LOAD restores from the SAV of a World holding its constructed
// registry holds it the same way. It reports whether a registry was marked.
func (w *World) AdoptConstructedGroups(native bool) bool {
	held := cloneSavedGroups(w.savedGroups)
	held.fresh, held.native = true, native
	w.savedGroups = held
	return true
}

// WithGroupRegistry is the World a SAVE writes its Group records from: w when
// it holds a Group registry, else a copy holding the registry the constructor
// builds from the current state and its native Group records, so every save
// takes the one roster path. w is not changed: the copy takes the
// constructor's registry (constructGroups), and w is returned when the
// constructor kept the registry w holds.
func (w *World) WithGroupRegistry() *World {
	next := *w
	next.constructGroups(true)
	if next.savedGroups == w.savedGroups {
		return w
	}
	return &next
}

// groupsForm is the registry the byte form writes: none when the decoder
// constructs the same one again and the game's form omits it.
func (w *World) groupsForm() *savedGroupState {
	if !w.game.formHoldsConstructedGroups && w.groupsAsConstructed() {
		return nil
	}
	return w.savedGroups
}

// constructedGroups builds the mission-start registry: one Group per (owner,
// group) pair, Players in their current order (else slot order), a Player's
// Groups in placement order (AI-449, AI-GROUP-009). The constructor writes
// +0x45 = 1 and zeroes the rest (AI-449). Stand Ground on the controlled
// slot stores only +0x20 = 3 (AI-451, SAV-1219); every other Player's Groups
// take the guard setter (AI-450, SAV-1218); the trigger binder sets +0x48 = 1
// on a Group the script names (AI-452). A member's order holds its post cell,
// +0x14 its reach and +0x71 one (SAV-1220), in state 0x0b under Guard and
// 0x0c under Stand Ground. With native set the registry is an older engine's:
// a pair with a native Group record takes its order, notice base, commanded
// cell and roam counter, the AI bytes that engine held.
func (w *World) constructedGroups(native bool) *savedGroupState {
	s := &savedGroupState{fresh: true, native: native}
	type key struct{ owner, group uint32 }
	var keys []key
	for i := range w.entities {
		e := &w.entities[i]
		if k := (key{e.Owner, effectiveGroup(e)}); e.Decay < DecayBones && !slices.Contains(keys, k) {
			keys = append(keys, k)
		}
	}
	// The registry walks Groups Player by Player. With current Players for
	// every owner that is their order (constructGroupPlayers); otherwise slot
	// order.
	rank := func(owner uint32) uint32 { return owner }
	if w.currentPlayers != nil {
		ranks := map[uint32]uint32{}
		for i, p := range w.currentPlayers.Players {
			if _, seen := ranks[p.Slot]; !seen {
				ranks[p.Slot] = uint32(i)
			}
		}
		covered := true
		for _, k := range keys {
			_, found := ranks[k.owner]
			covered = covered && found
		}
		if covered {
			rank = func(owner uint32) uint32 { return ranks[owner] }
		}
	}
	slices.SortStableFunc(keys, func(a, b key) int { return cmp.Compare(rank(a.owner), rank(b.owner)) })
	named := w.scriptNamedGroups()
	for _, k := range keys {
		var members []int
		for i := range w.entities {
			if e := &w.entities[i]; e.Owner == k.owner && effectiveGroup(e) == k.group && e.Decay < DecayBones {
				members = append(members, i)
			}
		}
		g, orders := w.constructedGroup(k.owner, k.group, members, named, native)
		g.ID = uint32(len(s.Groups) + 1)
		s.Orders = append(s.Orders, orders...)
		s.Groups = append(s.Groups, g)
	}
	s.HighWater = uint32(len(s.Groups))
	w.constructGroupPlayers(s)
	return s
}

// constructedGroup is the Group constructedGroups builds for owner's Group
// selector holding the entities at members, and its members' orders.
func (w *World) constructedGroup(owner, selector uint32, members []int, named func(uint32, []int) bool, native bool) (SavedGroup, []SavedActorOrder) {
	g := SavedGroup{Selector: selector, Owner: SavedGroupReference{Class: 1, Owner: owner}}
	setGroupWords(w, &g, members, rowGate, groupValues{active: 1})
	state := uint32(0)
	switch owner {
	case 0:
		// No Player holds the Group: no stance setter runs and no member
		// gets an order block, so the Group dispatch decides nothing for
		// it, as a Group with no Player record.
	case SelfSlot:
		setGroupWords(w, &g, members, rowStanceStand, groupValues{order: orderStandGround})
		state = uint32(actorStateAcquire)
	default:
		// The guard's programme byte is 1, the second game's 18.
		setGroupWords(w, &g, members, rowStanceGuard, groupValues{order: orderGuard})
		state = uint32(actorStateGuard)
	}
	if record := w.nativeGroupRecord(owner, selector); native && record != nil {
		setGroupWords(w, &g, members, nativeRecordRow(record), groupValues{order: record.order, notice: record.base,
			roam: record.roamCounter, commanded: cell{x: record.commandedX, y: record.commandedY}})
	}
	if named(selector, members) {
		setGroupWords(w, &g, members, rowNamed, groupValues{})
	}
	var orders []SavedActorOrder
	for _, i := range members {
		e := w.entities[i]
		g.Members = append(g.Members, SavedGroupMember{Archive: e.SourceBinding.ArchiveIndex, Entity: e.ID, Bound: true})
		if owner != 0 {
			orders = append(orders, constructedOrder(e, state))
		}
	}
	return g, orders
}

// constructedOrder is a member's order block after the stance setter: the post
// cell, reach and +0x71 one. A member already under a native actor order
// (patrol, escort, retreat) keeps that order, authored as a native command.
func constructedOrder(e Entity, state uint32) SavedActorOrder {
	o := SavedActorOrder{Entity: e.ID, State: state}
	binary.LittleEndian.PutUint16(o.Raw[:], uint16(e.PostX)|uint16(e.PostY)<<8)
	o.Raw[0x14], o.Raw[0x71] = uint8(e.Reach), 1
	// The spawn setup writes the class spell slots (AI-341, SAV-1220).
	win := e.slotWindow()
	copy(o.Raw[orderSlotWindowStart:orderSlotWindowEnd], win[:])
	switch e.ActorState {
	case actorStatePatrol:
		if p, ok := PatrolOrder(e, o.Raw); ok {
			p.Authored = true
			return p
		}
	case actorStateDefend, actorStateFollow, actorStateRetreat:
		o.State, o.Authored = uint32(e.ActorState), true
	}
	return o
}

func (w *World) nativeGroupRecord(owner, group uint32) *groupAI {
	for i := range w.groups {
		if w.groups[i].owner == owner && w.groups[i].group == group {
			return &w.groups[i]
		}
	}
	return nil
}

// scriptNamedGroups reports whether the compiled script names a Group by its
// selector or names one of its members as a unit: the references the trigger
// binder resolves to a Group (AI-452).
func (w *World) scriptNamedGroups() func(selector uint32, members []int) bool {
	groups, units := map[uint32]bool{}, map[EntityID]bool{}
	note := func(group uint32, hasGroup bool, u, u2 EntityID, has, has2 bool) {
		if hasGroup {
			groups[group] = true
		}
		if has {
			units[u] = true
		}
		if has2 {
			units[u2] = true
		}
	}
	if w.script != nil {
		for _, c := range w.script.checks {
			note(c.Group, c.HasGroup, c.Unit, c.Unit2, c.HasUnit, c.HasUnit2)
		}
		for _, in := range w.script.instants {
			note(in.Group, in.HasGroup, in.Unit, in.Unit2, in.HasUnit, in.HasUnit2)
		}
	}
	return func(selector uint32, members []int) bool {
		if groups[selector] {
			return true
		}
		for _, i := range members {
			if units[w.entities[i].ID] {
				return true
			}
		}
		return false
	}
}

// constructGroupPlayers attaches the Group Players sub-carrier: the World's
// current Players and, for each Group, the Player whose slot owns it. A World
// without current Players, or with a Group no Player owns, holds none.
func (w *World) constructGroupPlayers(s *savedGroupState) {
	if w.currentPlayers == nil {
		return
	}
	players := w.currentPlayers.Players
	containers := make([]uint32, len(s.Groups))
	for i := range s.Groups {
		at := slices.IndexFunc(players, func(p SavedGroupPlayer) bool { return p.Slot == s.Groups[i].Owner.Owner })
		if at < 0 {
			return
		}
		containers[i] = players[at].ID
	}
	for i := range s.Groups {
		s.Groups[i].ContainerID = containers[i]
	}
	s.Players, s.PlayersPresent = slices.Clone(players), true
}

// syncConstructedMemberArchives gives each member of a constructed registry
// the archive index of its actor's restored source binding, the value
// constructedGroups reads at mission start.
func (w *World) syncConstructedMemberArchives() {
	if !w.savedGroups.fresh {
		return
	}
	w.savedGroups = cloneSavedGroups(w.savedGroups)
	for gi := range w.savedGroups.Groups {
		g := &w.savedGroups.Groups[gi]
		for mi := range g.Members {
			if i := indexOfEntity(w.entities, g.Members[mi].Entity); g.Members[mi].Bound && i >= 0 {
				g.Members[mi].Archive = w.entities[i].SourceBinding.ArchiveIndex
			}
		}
	}
}

// reconstructFreshGroups builds a constructed registry again after a load-time
// edit to the mission-start state, so the registry stays the one the
// spawner builds from that state. The caller keeps a registry that differs
// from the constructed one.
func (w *World) reconstructFreshGroups() {
	if w.savedGroups.fresh {
		w.savedGroups = w.constructedGroups(w.savedGroups.native)
	}
}
