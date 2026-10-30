package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The end-of-mission cull stores the state word +0x50 = 0 and +0x136 = 1 on
// every actor it keeps (SAV-1353); the rest of a kept actor's town state is
// what the mission left. Its Player +0x3d = 0 (SAV-1358) needs no carrier:
// the town Player's constructor holds the same 0.
const (
	cullActorState = 0
	cullActorDone  = 1
)

// An actor built in the town (hire, tavern, AddHero) holds the Unit
// constructor's values at a town STORE: +0x50 = 0x0b, +0x136 = 1 and the
// action-run end +0x138 = 0 (SAV-1353).
const (
	townBuiltState    = siegeHireStateWord
	townBuiltDone     = siegeHireU136
	townBuiltDeadline = 0
)

// currentActionRunEnd is the actor's +0x138 now: its current run end, else the
// end an original LOAD read for an actor that has not started a run since.
func currentActionRunEnd(e sim.Entity) uint32 {
	if !e.ActionClock.Known && e.NativeBasis.ScalarIsKnown(sim.ScalarU138) {
		return e.NativeBasis.Scalars[sim.ScalarU138]
	}
	return e.ActionClock.End
}

// carryKeptActors gives each carried member bound to a living actor of the
// ended mission the state that actor holds after the cull. A member the
// mission did not hold keeps the state it had.
func carryKeptActors(world *sim.World, carried []mapload.PartyMember, byParty map[string]sim.EntityID) error {
	entities := make(map[sim.EntityID]sim.Entity)
	for _, e := range world.Entities() {
		entities[e.ID] = e
	}
	motions, _, _, present := world.SavedActorMotions()
	byEntity := make(map[sim.EntityID]sim.SavedActorMotion, len(motions))
	for _, m := range motions {
		byEntity[m.Entity] = m
	}
	for i := range carried {
		id, bound := byParty[carried[i].ID]
		e, live := entities[id]
		if !bound || !live || !e.Alive() {
			continue
		}
		kept, err := keptFromMission(world, e, byEntity, present)
		if err != nil {
			return err
		}
		carried[i].Kept = kept
	}
	return nil
}

// keptFromMission is the town state of one actor the cull kept: the cull's
// two stores, and the run end, timer byte, Mover and word lists the mission
// left. The engine holds no Order for a member after the cull.
func keptFromMission(world *sim.World, e sim.Entity, motions map[sim.EntityID]sim.SavedActorMotion, present bool) (*mapload.KeptActor, error) {
	k := &mapload.KeptActor{State: cullActorState, Done: cullActorDone, Deadline: currentActionRunEnd(e)}
	if b := sim.NativeBasisNow(e); b.ScalarIsKnown(sim.ScalarU6C) {
		k.Timer = uint8(b.Scalars[sim.ScalarU6C])
	}
	old, imported := motions[e.ID]
	if imported && old.Current {
		mover := old.SavedMover()
		mover[0] = e.Facing
		keepMover(k, mover)
		k.StaticRoute, k.DynamicRoute = cloneWords(old.StaticRoute), cloneWords(old.DynamicRoute)
		return k, nil
	}
	constructed := false
	if !imported {
		old = sim.SavedActorMotion{Entity: e.ID}
		constructed = world.ApplyMover(e.ID, &old.Mover)
		if !constructed && present {
			return k, nil
		}
	}
	m, err := sim.ProjectActorMotion(e, old, world.Route(e.ID), !present || constructed)
	if err != nil {
		return nil, fmt.Errorf("kept actor %d motion: %w", e.ID, err)
	}
	keepMover(k, m.SavedMover())
	k.StaticRoute, k.DynamicRoute = cloneWords(m.StaticRoute), cloneWords(m.DynamicRoute)
	return k, nil
}

// keptFromCharacter is the carrier a town LOAD fills from an actor record. A
// Mover or Order the town writer's constructor would write anyway is left
// absent, and a record holding only the town-built values gives no carrier,
// so a town state crosses a SAVE and LOAD unchanged.
func keptFromCharacter(c sav.Character) *mapload.KeptActor {
	t := c.Town
	k := &mapload.KeptActor{State: t.State, Done: t.Done, Deadline: t.Deadline, Timer: t.Timer,
		StaticRoute: cloneWords(t.StaticRoute), DynamicRoute: cloneWords(t.DynamicRoute)}
	if len(t.Mover) == len(k.Mover) {
		keepMover(k, [180]byte(t.Mover))
	}
	if len(t.Order) == len(k.Order) && (len(t.OrderRing) != 0 || slices.ContainsFunc(t.Order, func(b byte) bool { return b != 0 })) {
		copy(k.Order[:], t.Order)
		k.OrderRing = cloneWords(t.OrderRing)
		k.OrderPresent = true
	}
	if k.State == townBuiltState && k.Done == townBuiltDone && k.Deadline == townBuiltDeadline && k.Timer == 0 &&
		!k.MoverPresent && !k.OrderPresent && k.StaticRoute == nil && k.DynamicRoute == nil {
		return nil
	}
	return k
}

// keepMover puts m in the carrier without its turn-rate byte 10, which the
// Human's current basis supplies. A Mover the constructor writes anyway stays
// absent, so the carrier has one shape whether a mission or a LOAD filled it.
func keepMover(k *mapload.KeptActor, m [180]byte) {
	m[10] = 0
	k.Mover, k.MoverPresent = m, !constructorMover(m)
	if !k.MoverPresent {
		k.Mover = [180]byte{}
	}
}

// constructorMover reports a Mover one of the town writer's constructors
// writes: zero apart from the domain mask (byte 5) and the turn rate (byte
// 10), and for a siege hire bytes 8 and 9 holding 5 and 255.
func constructorMover(m [180]byte) bool {
	if [2]byte{m[8], m[9]} != [2]byte{} && [2]byte{m[8], m[9]} != [2]byte{5, 255} {
		return false
	}
	for i, b := range m {
		if b != 0 && i != 5 && i != 8 && i != 9 && i != 10 {
			return false
		}
	}
	return true
}

// cloneWords copies a word list; an empty list is nil, as the carrier holds it.
func cloneWords(w []uint16) []uint16 {
	if len(w) == 0 {
		return nil
	}
	return slices.Clone(w)
}

// keepLoadedTownActors fills the carrier of each restored member from the
// actor record it was restored from; the two lists are parallel.
func keepLoadedTownActors(party []mapload.PartyMember, chars []sav.Character) {
	if len(party) != len(chars) {
		return
	}
	for i := range party {
		party[i].Kept = keptFromCharacter(chars[i])
	}
}

// writeTownActors writes each party actor's carried town state into the
// city's actor records, which follow the party's order. A member without a
// carrier was built in the town and holds the constructor's values.
func writeTownActors(city *sav.CityData, party []mapload.PartyMember) error {
	n := 0
	for _, o := range city.Objects {
		if o.Unit == nil || (o.Class != "Human" && o.Class != "Unit") {
			continue
		}
		if n >= len(party) {
			return fmt.Errorf("city actor records outnumber the party")
		}
		if err := writeTownActor(o.Unit, party[n].Kept); err != nil {
			return err
		}
		n++
	}
	if n != len(party) {
		return fmt.Errorf("city actor records number %d for a party of %d", n, len(party))
	}
	return nil
}

func writeTownActor(u *sav.CityUnitData, k *mapload.KeptActor) error {
	if len(u.Scalar1) != 19 || len(u.Scalar2) != 55 || len(u.Raw154) != 180 || len(u.Raw158) != 148 {
		return fmt.Errorf("city actor record has an unexpected scalar or object width")
	}
	if k == nil {
		binary.LittleEndian.PutUint32(u.Scalar1[4:8], townBuiltState)
		u.Scalar2[41] = townBuiltDone
		binary.LittleEndian.PutUint32(u.Scalar2[42:46], townBuiltDeadline)
		return nil
	}
	binary.LittleEndian.PutUint32(u.Scalar1[4:8], k.State)
	u.Scalar1[18] = k.Timer
	u.Scalar2[41] = k.Done
	binary.LittleEndian.PutUint32(u.Scalar2[42:46], k.Deadline)
	if k.MoverPresent {
		// Byte 10 is the turn rate the Human's current basis supplies.
		rate := u.Raw154[10]
		copy(u.Raw154, k.Mover[:])
		u.Raw154[10] = rate
		u.Words15c, u.Words178 = slices.Clone(k.StaticRoute), slices.Clone(k.DynamicRoute)
	}
	if k.OrderPresent {
		copy(u.Raw158, k.Order[:])
		u.Words158 = slices.Clone(k.OrderRing)
	}
	return nil
}

// missionPlayerColor is the Player +0x44 a mission's Player build stores for
// slot: the map's group colour plus one, or 1 when the map names no group for
// the slot (SAV-1359).
func missionPlayerColor(m *alm.Map, slot uint32) uint32 {
	if m != nil && slot > 0 && int(slot) <= len(m.Groups) {
		return m.Groups[slot-1].Color + 1
	}
	return 1
}

// writeTownPlayerColor writes the town's carried local Player +0x44 into the
// fresh Player record; byte 14 of its fixed run (SAV-1359).
func writeTownPlayerColor(city *sav.CityData, color uint8) error {
	for _, o := range city.Objects {
		if o.Class == "Player" && o.Player != nil {
			if len(o.Player.Fixed) != 51 {
				return fmt.Errorf("city Player record has an unexpected fixed width")
			}
			o.Player.Fixed[14] = color
			return nil
		}
	}
	return fmt.Errorf("city document has no Player record")
}
