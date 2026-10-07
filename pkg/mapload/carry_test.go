package mapload_test

// The continuity hotfix, the loader's half: what a party member EARNED in one
// mission reaches the next one.
//
// Before it, every mission opened on a freshly minted party — so winning
// mission 10 and starting mission 20 threw away the experience, the pack and
// the worn set the player had just spent a mission accumulating.
//
// EVERY ASSERTION HERE IS AN EXACT VALUE. "The second mission's hero has some
// experience" is the trap this file is written against: it passes for a carry
// that hands over a fresh hero who earns it again, it passes for a carry that
// rounds the number down to the level it implies, and it passes for a carry
// that keeps the level and drops the residue. The numbers below are the
// numbers the first mission ended on and nothing else is accepted.

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The experience the first mission is made to end on. BOTH SLOTS CARRY A
// RESIDUE the level-to-experience direction cannot reproduce: 700 is level 5,
// which accounts for 610, and 1700 is level 10, which accounts for 1593. So a
// carry that went round through the level alone would re-mint [610 0 0 1593 0
// 0] and this file's own numbers would say so.
var (
	carryXP    = [data.SkillSlots]int32{700, 0, 0, 1700, 0, 0}
	carryLevel = [data.SkillSlots]int32{5, 0, 0, 10, 0, 0}
	carryReset = [data.SkillSlots]int32{610, 0, 0, 1593, 0, 0}
)

// Codes the fixture carries. Field B of an item code is its equipment slot, so
// 0x01xx is a slot-1 item and 0x03xx a slot-3 one; the pack holds two codes
// that name no slot at all, which is what a pack is for.
const (
	carryWornSlot1 uint16 = 0x0123
	carryWornSlot3 uint16 = 0x0345
	carryPackA     uint16 = 0x0e07
	carryPackB     uint16 = 0x0e11
)

// carryMap is a map with one authorised drop cell, large enough for a party to
// stand around it — startMap's own fixture, named here so this file's two
// starts are demonstrably the same map.
func carryMap(t *testing.T) *alm.Map {
	t.Helper()
	return startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
}

// endedWith is the world a mission is FINISHED in: the world a start built,
// with one entity holding the experience, the pack and the worn set it would
// have accumulated.
//
// It rebuilds rather than mutates because a sim.World's state is its own and
// this package cannot reach in — which is the right shape for the test anyway:
// what CarryParty is handed is a world some other code produced, and building
// one here proves it reads that world rather than remembering the party.
func endedWith(t *testing.T, m *alm.Map, w *sim.World, id sim.EntityID,
	mutate func(*sim.Entity), items []uint16, eq [sim.EquipSlots]uint16) *sim.World {
	t.Helper()

	ents := w.Entities()
	found := false
	for i := range ents {
		if ents[i].ID == id {
			mutate(&ents[i])
			found = true
		}
	}
	if !found {
		t.Fatalf("world holds no entity %d", id)
	}

	stock := []sim.Stock{}
	for _, s := range w.Stock() {
		if s.ID != id {
			stock = append(stock, s)
		}
	}
	stock = append(stock, sim.Stock{ID: id, Items: items, Equipped: eq})

	out, err := sim.NewStockedWorld(mapload.Seed, w.Bounds(), sim.ModeCanonical,
		mapload.Planes(m, nil), ents, nil, w.Relations(), w.Sacks(), stock)
	if err != nil {
		t.Fatalf("rebuild the finished world: %v", err)
	}
	return out
}

func earned(xp, level [data.SkillSlots]int32) func(*sim.Entity) {
	return func(e *sim.Entity) {
		e.SkillXP, e.Skill = xp, level
		e.NativeTraining = sim.NativeTraining{Present: true, Levels: level}
	}
}

// heroWith is a party of one, standing on the fixture's own class key,
// holding the statistics a generated character starts near, and — 0134's
// own widening of the mint (plan D-5) — wearing w's own code in slot 1
// whenever w is handed one.
//
// THIS FIXTURE BUILDS A PartyMember BY HAND rather than through
// GeneratedWornSet, which needs a base row this file has none of: it is
// this file's own business to state in Worn what the pre-0134 mint used to
// derive from Weapon automatically, on wornFromWeapon's own ground
// (spawn.go) — a weapon's own class constant always composes into slot 1,
// so there is no second slot this fixture could name.
func heroWith(w *data.Weapon) []mapload.PartyMember {
	var worn [sim.EquipSlots]uint16
	if w != nil {
		worn[0] = uint16(w.Code)
	}
	return []mapload.PartyMember{{
		Class:  100,
		Hero:   data.Hero{Body: 25, Reaction: 25, Mind: 25, Spirit: 25},
		Weapon: w,
		Worn:   worn,
	}}
}

// The whole of the RPG half, end to end: mission one is started, finished
// holding something, and mission two is started from what it left.
func TestWhatAMemberEarnedInOneMissionIsWhatHeStartsTheNextWith(t *testing.T) {
	m := carryMap(t)
	w1, st1 := mustStart(t, m, heroWith(nil))
	id1 := st1.IDs[0]

	var eq [sim.EquipSlots]uint16
	eq[0] = carryWornSlot1
	eq[2] = carryWornSlot3
	pack := []uint16{carryPackA, carryPackB}
	done := endedWith(t, m, w1, id1, earned(carryXP, carryLevel), pack, eq)

	next := mapload.CarryParty(heroWith(nil), done, st1.IDs)
	if len(next) != 1 || next[0].Carry == nil {
		t.Fatalf("CarryParty produced %d member(s), carry %v", len(next), next[0].Carry)
	}

	if next[0].Hero.Skill != carryLevel {
		t.Errorf("carried levels %v, want %v — the entity's own stored Skill",
			next[0].Hero.Skill, carryLevel)
	}

	w2, st2 := mustStart(t, m, next)
	id2 := st2.IDs[0]
	e2 := entityOf(t, w2, id2)

	// THE EXACT INTEGER, NOT THE LEVEL'S OWN. carryReset is what a carry
	// through the level alone would have produced, and it is named in the
	// failure so a reader sees at once which of the two happened.
	if e2.SkillXP != carryXP {
		what := "some other number"
		if e2.SkillXP == carryReset {
			what = "the level's own seed — the residue was dropped"
		}
		t.Errorf("second mission's experience %v, want %v (%s)", e2.SkillXP, carryXP, what)
	}

	items, ok := w2.Carried(id2)
	if !ok || !reflect.DeepEqual(items, pack) {
		t.Errorf("second mission's pack %v (ok %v), want %v", items, ok, pack)
	}
	got, ok := w2.Equipped(id2)
	if !ok || got != eq {
		t.Errorf("second mission's worn set %v (ok %v), want %v", got, ok, eq)
	}
}

func TestWhatAMageLearnsCrossesTheMissionBoundary(t *testing.T) {
	m := carryMap(t)
	party := heroWith(nil)
	party[0].Mage = true
	party[0].KnownSpells = uint32(1) << 4
	w1, st1 := mustStart(t, m, party)
	const learned = uint32(1) << 26
	done := endedWith(t, m, w1, st1.IDs[0], func(e *sim.Entity) {
		e.KnownSpells |= learned
	}, nil, [sim.EquipSlots]uint16{})

	nextParty := mapload.CarryParty(party, done, st1.IDs)
	want := party[0].KnownSpells | learned
	if got := nextParty[0].KnownSpells; got != want {
		t.Fatalf("carried KnownSpells = %#x, want %#x", got, want)
	}
	w2, st2 := mustStart(t, m, nextParty)
	if got := entityOf(t, w2, st2.IDs[0]).KnownSpells; got != want {
		t.Fatalf("next mission KnownSpells = %#x, want %#x", got, want)
	}
}

// A member who ended a mission wearing NOTHING arrives wearing nothing, and is
// not handed his class row's starting weapon a second time.
//
// This is the case that makes Carry a pointer. A player who drops his sword
// and wins would otherwise find it back in his hand at the next mission's
// first tick, every time, with nothing to say why.
func TestACarriedMemberIsNotReArmedFromHisClassRow(t *testing.T) {
	m := carryMap(t)
	weapon := &data.Weapon{Code: data.ItemCode(carryWornSlot1)}

	// The control: the SAME member with no carry at all is armed, so the
	// assertion below is about the carry and not about a weapon that never
	// reached the mint.
	w0, st0 := mustStart(t, m, heroWith(weapon))
	armed, _ := w0.Equipped(st0.IDs[0])
	if armed[0] != carryWornSlot1 {
		t.Fatalf("a member with no carry wears %v in slot 1, want 0x%04x — "+
			"the control for this test does not hold", armed, carryWornSlot1)
	}

	done := endedWith(t, m, w0, st0.IDs[0], earned([data.SkillSlots]int32{}, [data.SkillSlots]int32{}),
		nil, [sim.EquipSlots]uint16{})
	next := mapload.CarryParty(heroWith(weapon), done, st0.IDs)

	w2, st2 := mustStart(t, m, next)
	got, _ := w2.Equipped(st2.IDs[0])
	if got != ([sim.EquipSlots]uint16{}) {
		t.Errorf("a member who ended bare arrives wearing %v, want nothing", got)
	}
	items, _ := w2.Carried(st2.IDs[0])
	if len(items) != 0 {
		t.Errorf("a member who ended empty-handed arrives holding %v, want nothing", items)
	}
}

// A nil carry is every party this tree built before the hotfix, and it changes
// nothing: the mint takes the arm it always took, at the values it always
// produced.
func TestANilCarryMintsExactlyWhatItAlwaysDid(t *testing.T) {
	m := carryMap(t)
	p := heroWith(&data.Weapon{Code: data.ItemCode(carryWornSlot1)})
	p[0].Hero.Skill = carryLevel

	w, st := mustStart(t, m, p)
	e := entityOf(t, w, st.IDs[0])

	// The level's own seed, which is exactly what Hero.Reward() produces and
	// what this tree minted before Carry existed.
	if e.SkillXP != carryReset {
		t.Errorf("experience %v, want %v — the levels' own seed", e.SkillXP, carryReset)
	}
	eq, _ := w.Equipped(st.IDs[0])
	if eq[0] != carryWornSlot1 {
		t.Errorf("slot 1 = 0x%04x, want 0x%04x — the class row's own weapon", eq[0], carryWornSlot1)
	}
}

// A member whose entity did not survive is carried forward as he ENTERED that
// mission: nothing invented for him, nothing taken away.
func TestAMemberWhoseEntityIsGoneCarriesNothingForward(t *testing.T) {
	m := carryMap(t)
	w, st := mustStart(t, m, heroWith(nil))

	// A world that never held the id at all, which is the same answer a world
	// holding it dead gives.
	empty, err := sim.NewTerrainWorld(mapload.Seed, w.Bounds(), sim.ModeCanonical, mapload.Planes(m, nil), nil, nil)
	if err != nil {
		t.Fatalf("NewTerrainWorld: %v", err)
	}
	in := heroWith(nil)
	in[0].Hero.Skill = carryLevel

	next := mapload.CarryParty(in, empty, st.IDs)
	if len(next) != 1 {
		t.Fatalf("CarryParty produced %d member(s), want 1", len(next))
	}
	if next[0].Carry != nil {
		t.Errorf("carry %+v, want none for an entity the world does not hold", *next[0].Carry)
	}
	if next[0].Hero.Skill != carryLevel {
		t.Errorf("levels %v, want %v — the member as he entered", next[0].Hero.Skill, carryLevel)
	}
}

// A member whose entity is STILL IN THE WORLD but is no longer alive carries
// nothing forward either.
//
// It is the same rule as the one above and a different arm of it: a world that
// no longer holds the id and one that holds it downed answer the same question
// the same way, which is settleNotices' own test one tier up. Written
// separately because only this one exercises the aliveness check — an absent
// entity is refused by the lookup before aliveness is ever asked.
func TestADownedMemberCarriesNothingForward(t *testing.T) {
	m := carryMap(t)
	w, st := mustStart(t, m, heroWith(nil))

	done := endedWith(t, m, w, st.IDs[0], func(e *sim.Entity) {
		e.SkillXP = carryXP
		e.HP = 0 // downed: zero health with a health system behind it
	}, nil, [sim.EquipSlots]uint16{})

	e := entityOf(t, done, st.IDs[0])
	if e.Alive() {
		t.Fatalf("setup: entity HP %d/%d still reads as alive", e.HP, e.MaxHP)
	}

	in := heroWith(nil)
	in[0].Hero.Skill = carryLevel
	next := mapload.CarryParty(in, done, st.IDs)

	if next[0].Carry != nil {
		t.Errorf("carry %+v, want none for an entity that did not survive", *next[0].Carry)
	}
	if next[0].Hero.Skill != carryLevel {
		t.Errorf("levels %v, want %v — the member as he entered", next[0].Hero.Skill, carryLevel)
	}
}

// CarryParty never writes through its argument: the mission that has just been
// won still holds the party it was STARTED with, so a reader asking what a
// member began that mission as still gets that answer.
func TestCarryPartyLeavesTheFinishedMissionsPartyAlone(t *testing.T) {
	m := carryMap(t)
	w1, st1 := mustStart(t, m, heroWith(nil))
	started := heroWith(nil)
	done := endedWith(t, m, w1, st1.IDs[0], earned(carryXP, carryLevel), nil, [sim.EquipSlots]uint16{})

	next := mapload.CarryParty(started, done, st1.IDs)
	if started[0].Carry != nil {
		t.Errorf("the finished mission's own party gained a carry: %+v", *started[0].Carry)
	}
	if started[0].Hero.Skill != ([data.SkillSlots]int32{}) {
		t.Errorf("the finished mission's own party gained levels: %v", started[0].Hero.Skill)
	}
	if next[0].Hero.Skill != carryLevel {
		t.Errorf("the carried party's levels %v, want %v", next[0].Hero.Skill, carryLevel)
	}
}

func TestASavedDoesNotCrossTheMissionBoundary(t *testing.T) {
	m := carryMap(t)
	savedCell := mapload.Cell{X: 44, Y: 51}
	resumed := heroWith(nil)
	resumed[0].Saved = &mapload.Saved{
		Cell: savedCell, HP: 12, MaxHP: 131, Mana: 3, MaxMana: 90, MapUnitID: 21,
	}
	w1, st1 := mustStart(t, m, resumed)
	if st1.Cells[0] != savedCell {
		t.Fatalf("the resumed member started at %v, want the save's own %v", st1.Cells[0], savedCell)
	}
	done := endedWith(t, m, w1, st1.IDs[0], earned(carryXP, carryLevel), nil,
		[sim.EquipSlots]uint16{})

	next := mapload.CarryParty(resumed, done, st1.IDs)
	if next[0].Saved != nil {
		t.Fatalf("the carried member still holds a Saved: %+v", *next[0].Saved)
	}
	// A member whose entity did not survive is cleared too: the whole party is
	// moving to a map the save says nothing about.
	dead := mapload.CarryParty(resumed, done, []sim.EntityID{9999})
	if dead[0].Saved != nil {
		t.Errorf("a member whose entity did not survive kept a Saved: %+v", *dead[0].Saved)
	}
	// The finished mission's own party is untouched, on CarryParty's own terms.
	if resumed[0].Saved == nil {
		t.Errorf("the finished mission's own party lost its Saved")
	}

	// Started into the next mission, he stands at THAT map's drop cell.
	_, st2 := mustStart(t, m, next)
	if st2.Cells[0] != st2.Drop {
		t.Errorf("the carried member was placed at %v, want the next map's drop cell %v",
			st2.Cells[0], st2.Drop)
	}
	if st2.Cells[0] == savedCell {
		t.Errorf("the carried member is still standing on the previous mission's cell %v", savedCell)
	}
}

// entityOf is w's entity id, or a fatal — the lookup this file does four
// times, written once.
func entityOf(t *testing.T, w *sim.World, id sim.EntityID) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("world holds no entity %d", id)
	return sim.Entity{}
}
